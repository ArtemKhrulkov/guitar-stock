package images

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Scraper struct {
	logger   *logrus.Logger
	scrapers []ImageScraper
}

func NewScraper(db *gorm.DB) *Scraper {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger.SetLevel(logrus.InfoLevel)

	scrapers := []ImageScraper{
		NewBingScraper(),
		NewGoogleScraper(),
		NewSweetwaterScraper(),
		NewManufacturerScraper(),
		NewGuitarCenterScraper(),
		NewWildberriesImageScraper(db),
	}

	return &Scraper{
		logger:   logger,
		scrapers: scrapers,
	}
}

func (s *Scraper) Scrape(ctx context.Context, brand, model string) (*ImageResult, error) {
	for _, scraper := range s.scrapers {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		searchCtx, cancel := context.WithTimeout(ctx, 60*time.Second)

		s.logger.Debugf("[Scraper] Trying %s for %s %s", scraper.Name(), brand, model)
		result, err := scraper.Search(searchCtx, brand, model)
		cancel()

		if err != nil {
			if IsBrowserNotFound(err) {
				s.logger.Warnf("[Scraper] Browser not found, skipping %s", scraper.Name())
				continue
			}
			s.logger.Debugf("[Scraper] %s failed: %v", scraper.Name(), err)
			continue
		}

		if result != nil && result.IsValid() && !result.IsPlaceholder() {
			s.logger.Debugf("[Scraper] Found image from %s: %s", scraper.Name(), result.URL)
			return result, nil
		}
	}

	return nil, nil
}
