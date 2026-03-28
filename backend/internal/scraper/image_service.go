package scraper

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"guitar-stock/internal/repository"
	imgscrape "guitar-stock/internal/scraper/images"
)

type ImageService struct {
	db         *gorm.DB
	guitarRepo *repository.GuitarRepository
	logger     *logrus.Logger
	scrapers   []imgscrape.ImageScraper
	semaphore  *Semaphore
}

func NewImageService(db *gorm.DB) *ImageService {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	scrapers := []imgscrape.ImageScraper{
		imgscrape.NewSweetwaterScraper(),
		imgscrape.NewManufacturerScraper(),
		imgscrape.NewBingScraper(),
		imgscrape.NewGoogleScraper(),
	}

	return &ImageService{
		db:         db,
		guitarRepo: repository.NewGuitarRepository(db),
		logger:     logger,
		scrapers:   scrapers,
		semaphore:  NewSemaphore(5),
	}
}

func (s *ImageService) RegisterScraper(scraper imgscrape.ImageScraper) {
	s.scrapers = append(s.scrapers, scraper)
}

func (s *ImageService) ScrapeGuitar(ctx context.Context, guitarID uuid.UUID) (*imgscrape.ImageResult, error) {
	brand, err := s.guitarRepo.GetBrand(guitarID)
	if err != nil {
		return nil, err
	}

	guitar, err := s.guitarRepo.FindByID(guitarID)
	if err != nil {
		return nil, err
	}

	s.logger.Infof("[ImageService] Starting image scrape for: %s %s", brand.Name, guitar.Model)

	result := s.tryAllScrapers(ctx, brand.Name, guitar.Model)
	if result == nil {
		s.logger.Infof("[ImageService] No image found for %s %s", brand.Name, guitar.Model)
		return nil, nil
	}

	if err := s.updateGuitarImage(guitarID, result); err != nil {
		s.logger.Errorf("[ImageService] Failed to update guitar image: %v", err)
		return result, err
	}

	s.logger.Infof("[ImageService] Successfully scraped image for %s %s from %s",
		brand.Name, guitar.Model, result.Source)
	return result, nil
}

func (s *ImageService) tryAllScrapers(ctx context.Context, brand, model string) *imgscrape.ImageResult {
	for _, scraper := range s.scrapers {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		searchCtx, cancel := context.WithTimeout(ctx, 60*time.Second)

		s.logger.Infof("[ImageService] Trying scraper: %s", scraper.Name())
		result, err := scraper.Search(searchCtx, brand, model)
		cancel()

		if err != nil {
			s.logger.Warnf("[ImageService] Scraper %s failed: %v", scraper.Name(), err)
			continue
		}

		if result != nil && result.IsValid() && !result.IsPlaceholder() {
			s.logger.Infof("[ImageService] Found image from %s: %s", scraper.Name(), result.URL)
			return result
		}
	}

	return nil
}

func (s *ImageService) updateGuitarImage(guitarID uuid.UUID, result *imgscrape.ImageResult) error {
	now := time.Now()
	return s.guitarRepo.UpdateImage(guitarID, result.URL, result.Source, &now)
}

func (s *ImageService) ScrapeAll(ctx context.Context) (*ImageScrapeResult, error) {
	ids, err := s.guitarRepo.FindIDsWithoutImages()
	if err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		s.logger.Info("[ImageService] All guitars already have images")
		return &ImageScrapeResult{Total: 0, Success: 0, Failed: 0}, nil
	}

	s.logger.Infof("[ImageService] Starting image scrape for %d guitars (batch size: 5, concurrency: 5)", len(ids))

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	var wg sync.WaitGroup
	var completed int64
	var success int64
	var failed int64

	batchSize := 5
	for i := 0; i < len(ids); i += batchSize {
		select {
		case <-ctx.Done():
			s.logger.Info("[ImageService] Context cancelled, stopping")
			goto waitForCompletion
		case <-sigChan:
			s.logger.Info("[ImageService] Received shutdown signal, waiting for completion...")
			goto waitForCompletion
		default:
		}

		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[i:end]

		for _, id := range batch {
			s.semaphore.Acquire()
			wg.Add(1)

			go func(guitarID uuid.UUID) {
				defer wg.Done()
				defer s.semaphore.Release()

				select {
				case <-ctx.Done():
					s.logger.Infof("[ImageService] Skipping guitar due to shutdown: %s", guitarID)
					atomic.AddInt64(&failed, 1)
					return
				default:
				}

				_, err := s.ScrapeGuitar(ctx, guitarID)
				atomic.AddInt64(&completed, 1)

				if err != nil || ctx.Err() != nil {
					atomic.AddInt64(&failed, 1)
				} else {
					atomic.AddInt64(&success, 1)
				}
			}(id)
		}
	}

waitForCompletion:
	wg.Wait()

	result := &ImageScrapeResult{
		Total:     len(ids),
		Success:   int(success),
		Failed:    int(failed),
		Completed: int(completed),
	}

	s.logger.Infof("[ImageService] Completed: %d/%d (success: %d, failed: %d)",
		completed, len(ids), success, failed)

	return result, nil
}

func (s *ImageService) GetGuitarsWithoutImages() ([]uuid.UUID, error) {
	return s.guitarRepo.FindIDsWithoutImages()
}

type ImageScrapeResult struct {
	Total     int `json:"total"`
	Success   int `json:"success"`
	Failed    int `json:"failed"`
	Completed int `json:"completed"`
}

func (s *ImageService) IsPlaceholderImage(imageURL *string) bool {
	if imageURL == nil || *imageURL == "" {
		return true
	}

	placeholderPatterns := []string{
		"via.placeholder.com",
		"placeholder.com",
		"placehold.co",
		"placeholder.nl",
		"placehold.it",
	}

	lowerURL := strings.ToLower(*imageURL)
	for _, pattern := range placeholderPatterns {
		if strings.Contains(lowerURL, pattern) {
			return true
		}
	}

	return false
}
