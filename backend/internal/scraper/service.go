package scraper

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
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

	return &Service{
		config:           config,
		limiter:          limiter,
		proxies:          proxies,
		semaphore:        semaphore,
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

	return &Service{
		config:           config,
		limiter:          limiter,
		proxies:          proxies,
		semaphore:        semaphore,
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

	s.updateGuitarPriceRange(guitarID, guitar.Model, allLinks)

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
	var completed int64

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
			atomic.AddInt64(&completed, 1)
		}(id)
	}

waitForCompletion:
	wg.Wait()

	s.logger.Infof("[SCRAPER] Completed %d/%d guitars", completed, len(ids))
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
	s.proxies.LogStats()
	s.logger.Info("[SCRAPER] Scraper service closed")
}

func (s *Service) updateGuitarPriceRange(guitarID uuid.UUID, model string, links []models.PurchaseLink) {
	var pricesRUB, pricesUSD []float64

	for _, link := range links {
		if link.PriceRUB != nil && *link.PriceRUB > 0 {
			pricesRUB = append(pricesRUB, *link.PriceRUB)
		}
		if link.PriceUSD != nil && *link.PriceUSD > 0 {
			pricesUSD = append(pricesUSD, *link.PriceUSD)
		}
	}

	priceRange := FormatPriceRange(pricesRUB, pricesUSD, s.config.ExchangeRate)
	if priceRange == "" {
		s.logger.Infof("[SCRAPER] No valid prices found for %s, skipping price_range update", model)
		return
	}

	if err := s.guitarRepo.UpdatePriceRange(guitarID, priceRange); err != nil {
		s.logger.Errorf("[SCRAPER] Failed to update price_range for %s: %v", model, err)
	} else {
		s.logger.Infof("[SCRAPER] Updated price_range for %s: %s", model, priceRange)
	}
}

type PriceRangeSyncResult struct {
	Total   int                    `json:"total"`
	Updated int                    `json:"updated"`
	Skipped int                    `json:"skipped"`
	Errors  int                    `json:"errors"`
	DryRun  bool                   `json:"dry_run"`
	Details []PriceRangeSyncDetail `json:"details"`
}

type PriceRangeSyncDetail struct {
	GuitarID     string `json:"guitar_id"`
	Model        string `json:"model"`
	CurrentRange string `json:"current_range,omitempty"`
	NewRange     string `json:"new_range,omitempty"`
	Skipped      bool   `json:"skipped,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

func (s *Service) SyncPriceRanges(dryRun bool) (*PriceRangeSyncResult, error) {
	guitars, err := s.guitarRepo.FindAllWithPurchaseLinks()
	if err != nil {
		return nil, err
	}

	result := &PriceRangeSyncResult{
		Total:   len(guitars),
		DryRun:  dryRun,
		Details: []PriceRangeSyncDetail{},
	}

	for _, guitar := range guitars {
		var pricesRUB, pricesUSD []float64
		for _, link := range guitar.PurchaseLinks {
			if link.PriceRUB != nil && *link.PriceRUB > 0 {
				pricesRUB = append(pricesRUB, *link.PriceRUB)
			}
			if link.PriceUSD != nil && *link.PriceUSD > 0 {
				pricesUSD = append(pricesUSD, *link.PriceUSD)
			}
		}

		newRange := FormatPriceRange(pricesRUB, pricesUSD, s.config.ExchangeRate)

		if newRange == "" {
			result.Skipped++
			result.Details = append(result.Details, PriceRangeSyncDetail{
				GuitarID: guitar.ID.String(),
				Model:    guitar.Model,
				Skipped:  true,
				Reason:   "no valid prices in purchase links",
			})
			continue
		}

		if dryRun {
			result.Updated++
			result.Details = append(result.Details, PriceRangeSyncDetail{
				GuitarID:     guitar.ID.String(),
				Model:        guitar.Model,
				CurrentRange: guitar.PriceRange,
				NewRange:     newRange,
			})
		} else {
			if err := s.guitarRepo.UpdatePriceRange(guitar.ID, newRange); err != nil {
				result.Errors++
				s.logger.Errorf("[SYNC] Failed to update price_range for %s: %v", guitar.Model, err)
				result.Details = append(result.Details, PriceRangeSyncDetail{
					GuitarID: guitar.ID.String(),
					Model:    guitar.Model,
					Skipped:  true,
					Reason:   "update failed: " + err.Error(),
				})
			} else {
				result.Updated++
				result.Details = append(result.Details, PriceRangeSyncDetail{
					GuitarID:     guitar.ID.String(),
					Model:        guitar.Model,
					CurrentRange: guitar.PriceRange,
					NewRange:     newRange,
				})
				s.logger.Infof("[SYNC] Updated price_range for %s: %s", guitar.Model, newRange)
			}
		}
	}

	return result, nil
}
