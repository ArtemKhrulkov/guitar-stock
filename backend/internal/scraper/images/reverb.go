package images

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/sirupsen/logrus"
)

type ReverbScraper struct {
	logger   *logrus.Logger
	timeout  time.Duration
	launcher *BrowserLauncher
}

func NewReverbScraper() *ReverbScraper {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger.SetLevel(logrus.InfoLevel)

	return &ReverbScraper{
		logger:   logger,
		timeout:  30 * time.Second,
		launcher: NewBrowserLauncher(),
	}
}

func (s *ReverbScraper) Name() string {
	return "reverb"
}

func (s *ReverbScraper) Priority() int {
	return 12
}

func (s *ReverbScraper) Search(ctx context.Context, brand, model string) (*ImageResult, error) {
	s.logger.Infof("[Reverb] Searching for: %s %s", brand, model)

	searchURL := s.buildSearchURL(brand, model)
	s.logger.Debugf("[Reverb] Search URL: %s", searchURL)

	instance, err := s.launcher.Launch(ctx)
	if err != nil {
		s.logger.Errorf("[Reverb] Failed to launch browser: %v", err)
		return nil, err
	}
	defer instance.Close()

	page := instance.Page

	if err := page.Timeout(s.timeout).Navigate(searchURL); err != nil {
		s.logger.Warnf("[Reverb] Navigation failed: %v", err)
		return nil, nil
	}

	time.Sleep(3 * time.Second)

	imageURL := s.safeExtractImage(page)
	if imageURL == "" {
		s.logger.Infof("[Reverb] No image found for %s %s", brand, model)
		return nil, nil
	}

	width, height, err := FetchImageDimensions(imageURL)
	if err != nil {
		s.logger.Debugf("[Reverb] Failed to fetch dimensions for %s: %v", imageURL, err)
		width, height = 0, 0
	}

	if width < MinWidth || height < MinHeight {
		s.logger.Debugf("[Reverb] Image too small: %dx%d, skipping", width, height)
		return nil, nil
	}

	result := &ImageResult{
		URL:    imageURL,
		Source: "reverb",
		Width:  width,
		Height: height,
	}

	if !result.IsValid() || result.IsPlaceholder() {
		s.logger.Debugf("[Reverb] Image validation failed: %dx%d, skipping", width, height)
		return nil, nil
	}

	s.logger.Infof("[Reverb] Found image: %s (%dx%d)", imageURL, width, height)
	return result, nil
}

func (s *ReverbScraper) buildSearchURL(brand, model string) string {
	query := url.QueryEscape(fmt.Sprintf("%s %s electric guitar", brand, model))
	return fmt.Sprintf("https://reverb.com/marketplace?q=%s", query)
}

func (s *ReverbScraper) extractImage(page *rod.Page) string {
	jsCode := `
		function() {
			var results = [];
			
			var cards = document.querySelectorAll('[data-testid="listing-card"], .listing-card, a[href*="/listings/"]');
			for (var i = 0; i < Math.min(cards.length, 5); i++) {
				var card = cards[i];
				var img = card.querySelector('img');
				if (img) {
					var src = img.src || img.getAttribute('data-src') || img.getAttribute('srcset');
					if (src) {
						if (src.startsWith('http')) {
							results.push(src.split('?')[0]);
						} else if (src.includes('reverbcdn.com')) {
							results.push('https://' + src.split('?')[0]);
						}
					}
				}
			}
			
			if (results.length === 0) {
				var imgs = document.querySelectorAll('img[src*="reverbcdn.com"]');
				for (var i = 0; i < Math.min(imgs.length, 5); i++) {
					var img = imgs[i];
					var src = img.src;
					if (src && src.includes('reverbcdn.com')) {
						var cleanSrc = src.split('?')[0];
						if (!cleanSrc.includes('avatar') && !cleanSrc.includes('logo')) {
							results.push(cleanSrc);
						}
					}
				}
			}
			
			return results.length > 0 ? results[0] : '';
		}
	`

	result := page.MustEval(jsCode)
	imageURL := result.Str()

	if imageURL == "" {
		imageURL = s.fallbackExtract(page)
	}

	return imageURL
}

func (s *ReverbScraper) fallbackExtract(page *rod.Page) string {
	jsCode := `
		function() {
			var imgs = document.querySelectorAll('img');
			var results = [];
			for (var i = 0; i < Math.min(imgs.length, 10); i++) {
				var img = imgs[i];
				var src = img.src || img.getAttribute('data-src');
				if (src && src.includes('reverb')) {
					if (!src.includes('avatar') && !src.includes('logo') && !src.includes('badge')) {
						results.push(src.split('?')[0]);
					}
				}
			}
			return results.length > 0 ? results[0] : '';
		}
	`

	result := page.MustEval(jsCode)
	return result.Str()
}

func (s *ReverbScraper) isValidImageURL(url string) bool {
	if url == "" {
		return false
	}
	if !strings.HasPrefix(url, "http") {
		return false
	}
	badPatterns := []string{
		"placeholder", "logo", "icon", "avatar", "banner",
		"transparent", "spacer", "pixel", "1x1", "data:image",
	}
	for _, bad := range badPatterns {
		if strings.Contains(strings.ToLower(url), bad) {
			return false
		}
	}
	return true
}

func (s *ReverbScraper) safeExtractImage(page *rod.Page) string {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Debugf("[Reverb] Panic during image extraction: %v", r)
		}
	}()
	return s.extractImage(page)
}
