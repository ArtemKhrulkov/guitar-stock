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

type ThomannScraper struct {
	logger   *logrus.Logger
	timeout  time.Duration
	launcher *BrowserLauncher
}

func NewThomannScraper() *ThomannScraper {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger.SetLevel(logrus.InfoLevel)

	return &ThomannScraper{
		logger:   logger,
		timeout:  25 * time.Second,
		launcher: NewBrowserLauncher(),
	}
}

func (s *ThomannScraper) Name() string {
	return "thomann"
}

func (s *ThomannScraper) Priority() int {
	return 13
}

func (s *ThomannScraper) Search(ctx context.Context, brand, model string) (*ImageResult, error) {
	s.logger.Infof("[Thomann] Searching for: %s %s", brand, model)

	searchURL := s.buildSearchURL(brand, model)
	s.logger.Debugf("[Thomann] Search URL: %s", searchURL)

	instance, err := s.launcher.Launch(ctx)
	if err != nil {
		s.logger.Errorf("[Thomann] Failed to launch browser: %v", err)
		return nil, err
	}
	defer instance.Close()

	page := instance.Page

	if err := page.Timeout(s.timeout).Navigate(searchURL); err != nil {
		s.logger.Warnf("[Thomann] Navigation failed: %v", err)
		return nil, nil
	}

	time.Sleep(2 * time.Second)

	imageURL := s.safeExtractImage(page)

	if imageURL == "" {
		s.logger.Infof("[Thomann] No image found for %s %s", brand, model)
		return nil, nil
	}

	if !s.isValidImageURL(imageURL) {
		s.logger.Debugf("[Thomann] Invalid image URL: %s", imageURL)
		return nil, nil
	}

	width, height, err := FetchImageDimensions(imageURL)
	if err != nil {
		s.logger.Debugf("[Thomann] Failed to fetch dimensions for %s: %v", imageURL, err)
		width, height = 0, 0
	}

	if width < MinWidth || height < MinHeight {
		s.logger.Debugf("[Thomann] Image too small: %dx%d, skipping", width, height)
		return nil, nil
	}

	result := &ImageResult{
		URL:    imageURL,
		Source: "thomann",
		Width:  width,
		Height: height,
	}

	if !result.IsValid() || result.IsPlaceholder() {
		s.logger.Debugf("[Thomann] Image validation failed: %dx%d, skipping", width, height)
		return nil, nil
	}

	s.logger.Infof("[Thomann] Found image: %s (%dx%d)", imageURL, width, height)
	return result, nil
}

func (s *ThomannScraper) buildSearchURL(brand, model string) string {
	query := url.QueryEscape(fmt.Sprintf("%s %s", brand, model))
	return fmt.Sprintf("https://www.thomann.de/intl/search_music.html?search=%s", query)
}

func (s *ThomannScraper) extractImage(page *rod.Page) string {
	jsCode := `
		function() {
			var results = [];
			
			var products = document.querySelectorAll('.product-list__item, .product-list-item, article[data-product-id], .js-product-item');
			for (var i = 0; i < Math.min(products.length, 5); i++) {
				var product = products[i];
				var img = product.querySelector('img');
				if (img) {
					var src = img.src || img.getAttribute('data-src') || img.getAttribute('data-lazy-src');
					if (src && src.startsWith('http') && !src.includes('logo') && !src.includes('placeholder')) {
						results.push(src.split('?')[0]);
					}
				}
			}
			
			if (results.length === 0) {
				var imgs = document.querySelectorAll('img[src*="thomann.de"], img[src*="thomann"]');
				for (var i = 0; i < Math.min(imgs.length, 5); i++) {
					var img = imgs[i];
					var src = img.src;
					if (src && !src.includes('logo') && !src.includes('banner') && !src.includes('header')) {
						results.push(src.split('?')[0]);
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

func (s *ThomannScraper) fallbackExtract(page *rod.Page) string {
	jsCode := `
		function() {
			var imgs = document.querySelectorAll('img');
			var results = [];
			for (var i = 0; i < Math.min(imgs.length, 10); i++) {
				var img = imgs[i];
				var src = img.src || img.getAttribute('data-src');
				if (src && src.includes('thomann')) {
					if (!src.includes('logo') && !src.includes('banner') && !src.includes('icon')) {
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

func (s *ThomannScraper) isValidImageURL(url string) bool {
	if url == "" {
		return false
	}
	if !strings.HasPrefix(url, "http") {
		return false
	}
	badPatterns := []string{
		"placeholder", "logo", "icon", "avatar", "banner",
		"transparent", "spacer", "pixel", "1x1", "data:image",
		"cookie", "consent", "popup", "overlay", "close",
	}
	for _, bad := range badPatterns {
		if strings.Contains(strings.ToLower(url), bad) {
			return false
		}
	}
	return true
}

func (s *ThomannScraper) safeExtractImage(page *rod.Page) string {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Debugf("[Thomann] Panic during image extraction: %v", r)
		}
	}()
	return s.extractImage(page)
}
