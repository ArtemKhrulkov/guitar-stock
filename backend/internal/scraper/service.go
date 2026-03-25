package scraper

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"guitar-stock/internal/models"
	"guitar-stock/internal/repository"
)

type Service struct {
	config           *ScraperConfig
	limiter          *RateLimiter
	proxies          *ProxyPool
	semaphore        *Semaphore
	browserPool      *BrowserPool
	db               *gorm.DB
	guitarRepo       *repository.GuitarRepository
	brandRepo        *repository.BrandRepository
	purchaseLinkRepo *repository.PurchaseLinkRepository
	priceHistoryRepo *repository.PriceHistoryRepository
	logger           *logrus.Logger
}

func NewService(db *gorm.DB) *Service {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	config := LoadConfig()

	limiter := NewRateLimiter(config.RateLimit)
	proxies := NewProxyPool(config.ProxyURLs)
	semaphore := NewSemaphore(config.MaxConcurrent)
	browserPool := NewBrowserPool(config.MaxConcurrent, config.UserAgents, proxies)

	return &Service{
		config:           config,
		limiter:          limiter,
		proxies:          proxies,
		semaphore:        semaphore,
		browserPool:      browserPool,
		db:               db,
		guitarRepo:       repository.NewGuitarRepository(db),
		brandRepo:        repository.NewBrandRepository(db),
		purchaseLinkRepo: repository.NewPurchaseLinkRepository(db),
		priceHistoryRepo: repository.NewPriceHistoryRepository(db),
		logger:           logger,
	}
}

func NewServiceWithRepos(db *gorm.DB, guitarRepo *repository.GuitarRepository, brandRepo *repository.BrandRepository, purchaseLinkRepo *repository.PurchaseLinkRepository, priceHistoryRepo *repository.PriceHistoryRepository) *Service {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	config := LoadConfig()

	limiter := NewRateLimiter(config.RateLimit)
	proxies := NewProxyPool(config.ProxyURLs)
	semaphore := NewSemaphore(config.MaxConcurrent)
	browserPool := NewBrowserPool(config.MaxConcurrent, config.UserAgents, proxies)

	return &Service{
		config:           config,
		limiter:          limiter,
		proxies:          proxies,
		semaphore:        semaphore,
		browserPool:      browserPool,
		db:               db,
		guitarRepo:       guitarRepo,
		brandRepo:        brandRepo,
		purchaseLinkRepo: purchaseLinkRepo,
		priceHistoryRepo: priceHistoryRepo,
		logger:           logger,
	}
}

func (s *Service) ScrapeGuitar(ctx context.Context, guitarID uuid.UUID) ([]models.PurchaseLink, error) {
	brand, err := s.guitarRepo.GetBrand(guitarID)
	if err != nil {
		return nil, err
	}

	guitar, err := s.guitarRepo.FindByID(guitarID)
	if err != nil {
		return nil, err
	}

	s.logger.Infof("[SCRAPER] Starting scrape for guitar: %s (%s %s)", guitar.Model, brand.Name, guitarID)

	platforms := []Platform{Ozon, Wildberries}
	var allLinks []models.PurchaseLink

	for _, platform := range platforms {
		domain := s.getDomain(platform)
		s.limiter.Wait(domain)

		s.logger.Infof("[SCRAPER] Scraping platform: %s", platform)

		var lastErr error

		for attempt := 0; attempt <= s.config.Retries; attempt++ {
			if attempt > 0 {
				s.logger.Infof("[SCRAPER] Retry %d/%d for %s", attempt, s.config.Retries, platform)
				time.Sleep(s.config.RetryDelay)
			}

			proxy := s.proxies.Get()
			var proxyList []string
			if proxy != "" {
				proxyList = []string{proxy}
			}

			scraper := NewScraperWithProxies(platform, proxyList)
			results, err := scraper.Search(ctx, brand.Name, guitar.Model)

			if err != nil {
				s.logger.Errorf("[SCRAPER] Error from %s: %v", platform, err)
				if proxy != "" {
					s.proxies.RecordFailure(proxy, err.Error())
				}
				lastErr = err

				if !s.proxies.Has() {
					s.logger.Warn("[SCRAPER] All proxies exhausted, giving up")
					break
				}
				continue
			}

			if proxy != "" {
				s.proxies.RecordSuccess(proxy)
			}

			s.logger.Infof("[SCRAPER] Found %d results from %s", len(results), platform)

			for _, result := range results {
				link, err := s.purchaseLinkRepo.CreateIfNotExists(
					guitarID,
					models.Platform(platform),
					result.URL,
					result.PriceRUB,
					result.PriceUSD,
					result.InStock,
				)
				if err == nil && link != nil {
					s.logger.Printf("[SCRAPER] Created link: %s - %s", platform, result.URL)
					history := &models.PriceHistory{
						PurchaseLinkID: link.ID,
						PriceRUB:       link.PriceRUB,
						PriceUSD:       link.PriceUSD,
					}
					s.priceHistoryRepo.Create(history)

					allLinks = append(allLinks, *link)
				}
			}

			lastErr = nil
			break
		}

		if lastErr != nil && !s.proxies.Has() {
			s.logger.Warn("[SCRAPER] All proxies exhausted, stopping")
			break
		}
	}

	if len(allLinks) == 0 {
		s.logger.Warnf("[SCRAPER] No results for %s. Consider adding manual purchase links via POST /api/admin/links", guitar.Model)
	}

	s.logger.Infof("[SCRAPER] Completed scrape for %s, found %d total links", guitar.Model, len(allLinks))
	return allLinks, nil
}

func (s *Service) ScrapeAll(ctx context.Context) error {
	ids, err := s.guitarRepo.FindAllIDs()
	if err != nil {
		return err
	}

	s.logger.Infof("[SCRAPER] Starting scrape for %d guitars", len(ids))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	completed := make(chan uuid.UUID, len(ids))

	for _, id := range ids {
		select {
		case <-ctx.Done():
			s.logger.Info("[SCRAPER] Context cancelled, stopping")
			break
		case <-sigChan:
			s.logger.Info("[SCRAPER] Received shutdown signal, waiting for completion...")
			goto waitForCompletion
		default:
		}

		s.semaphore.Acquire()

		wg.Add(1)
		go func(guitarID uuid.UUID) {
			defer wg.Done()
			defer s.semaphore.Release()

			select {
			case <-ctx.Done():
				s.logger.Infof("[SCRAPER] Skipping guitar due to shutdown: %s", guitarID)
				return
			default:
			}

			s.ScrapeGuitar(ctx, guitarID)
			completed <- guitarID
		}(id)
	}

waitForCompletion:
	wg.Wait()

	s.logger.Infof("[SCRAPER] Completed %d/%d guitars", len(completed), len(ids))
	s.proxies.LogStats()

	return nil
}

func (s *Service) StartScheduler() {
	// Scheduler implementation will be added here
	// Using robfig/cron for periodic scraping
}

func (s *Service) getDomain(platform Platform) string {
	switch platform {
	case Ozon:
		return "ozon.ru"
	case Wildberries:
		return "wildberries.ru"
	default:
		return "unknown.ru"
	}
}

func (s *Service) Close() {
	s.logger.Info("[SCRAPER] Closing scraper service...")
	s.browserPool.Close()
	s.proxies.LogStats()
	s.logger.Info("[SCRAPER] Scraper service closed")
}
