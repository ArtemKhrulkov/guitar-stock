package images

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/sirupsen/logrus"
)

type GuitarCenterScraper struct {
	logger   *logrus.Logger
	timeout  time.Duration
	launcher *BrowserLauncher
}

func NewGuitarCenterScraper() *GuitarCenterScraper {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger.SetLevel(logrus.InfoLevel)

	return &GuitarCenterScraper{
		logger:   logger,
		timeout:  30 * time.Second,
		launcher: NewBrowserLauncher(),
	}
}

func (s *GuitarCenterScraper) Name() string {
	return "guitarcenter"
}

func (s *GuitarCenterScraper) Priority() int {
	return 5
}

func (s *GuitarCenterScraper) Search(ctx context.Context, brand, model string) (*ImageResult, error) {
	s.logger.Infof("[GuitarCenter] Searching for: %s %s", brand, model)

	searchURL := s.buildSearchURL(brand, model)
	s.logger.Infof("[GuitarCenter] Search URL: %s", searchURL)

	if imageURL := s.tryHTTP(searchURL); imageURL != "" {
		width, height, err := FetchImageDimensions(imageURL)
		if err != nil {
			s.logger.Debugf("[GuitarCenter] Failed to fetch dimensions for %s: %v", imageURL, err)
			s.logger.Infof("[GuitarCenter] Using fallback dimensions")
			width, height = 800, 600
		}
		if width < MinWidth || height < MinHeight {
			s.logger.Debugf("[GuitarCenter] Image too small: %dx%d, skipping", width, height)
			return nil, nil
		}
		result := &ImageResult{
			URL:    imageURL,
			Source: "guitarcenter",
			Width:  width,
			Height: height,
		}
		if !result.IsPlaceholder() {
			s.logger.Infof("[GuitarCenter] Found image via HTTP: %s", imageURL)
			return result, nil
		}
	}

	instance, err := s.launcher.Launch(ctx)
	if err != nil {
		s.logger.Errorf("[GuitarCenter] Failed to launch browser: %v", err)
		return nil, nil
	}
	defer instance.Close()

	s.logger.Infof("[GuitarCenter] Navigating to %s...", searchURL)
	if err := instance.Page.Timeout(20 * time.Second).Navigate(searchURL); err != nil {
		s.logger.Warnf("[GuitarCenter] Navigation failed: %v", err)
		return nil, nil
	}

	time.Sleep(2 * time.Second)

	imageURL := s.safeExtractImage(instance.Page)

	if imageURL == "" {
		s.logger.Infof("[GuitarCenter] No image found for %s %s", brand, model)
		return nil, nil
	}

	if s.isHeroImage(imageURL) {
		s.logger.Debugf("[GuitarCenter] Skipping hero image: %s", imageURL)
		return nil, nil
	}

	if !s.isValidImageURL(imageURL) {
		s.logger.Debugf("[GuitarCenter] Invalid/placeholder image: %s", imageURL)
		return nil, nil
	}

	width, height, err := FetchImageDimensions(imageURL)
	if err != nil {
		s.logger.Debugf("[GuitarCenter] Failed to fetch dimensions for %s: %v", imageURL, err)
		s.logger.Infof("[GuitarCenter] Using fallback dimensions")
		width, height = 800, 600
	}

	if width < MinWidth || height < MinHeight {
		s.logger.Debugf("[GuitarCenter] Image too small: %dx%d, skipping", width, height)
		return nil, nil
	}

	result := &ImageResult{
		URL:    imageURL,
		Source: "guitarcenter",
		Width:  width,
		Height: height,
	}

	if result.IsPlaceholder() {
		return nil, nil
	}

	s.logger.Infof("[GuitarCenter] Found image: %s", imageURL)
	return result, nil
}

func (s *GuitarCenterScraper) buildSearchURL(brand, model string) string {
	query := url.QueryEscape(fmt.Sprintf("%s %s", brand, model))
	return fmt.Sprintf("https://www.guitarcenter.com/search?searchTerm=%s", query)
}

func (s *GuitarCenterScraper) tryHTTP(searchURL string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	return s.extractFromHTML(string(body))
}

func (s *GuitarCenterScraper) extractImage(page *rod.Page) string {
	jsCode := `
		function() {
			var results = [];
			
			var products = document.querySelectorAll('.product-card, .product-result, [data-product-id], .product-item');
			for (var i = 0; i < Math.min(products.length, 3); i++) {
				var product = products[i];
				var img = product.querySelector('img');
				if (img) {
					var src = img.src || img.getAttribute('data-src');
					if (src && !src.includes('hero') && !src.includes('Hero') && !src.includes('banner') && !src.includes('Banner')) {
						results.push(src);
					}
				}
			}
			
			if (results.length === 0) {
				var img = document.querySelector('.product-detail-image img') ||
				          document.querySelector('.product-media-gallery img') ||
				          document.querySelector('[data-testid="product-image"] img');
				if (img) {
					var src = img.src || img.getAttribute('data-src');
					if (src && !src.includes('hero') && !src.includes('Hero')) {
						results.push(src);
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

func (s *GuitarCenterScraper) fallbackExtract(page *rod.Page) string {
	els, err := page.Elements("img[src*='guitarcenter']")
	if err == nil && len(els) > 0 {
		for _, el := range els[:5] {
			src, _ := el.Attribute("src")
			if src != nil && *src != "" && s.isValidImageURL(*src) {
				return *src
			}
		}
	}

	html, _ := page.HTML()
	return s.extractFromHTML(html)
}

func (s *GuitarCenterScraper) extractFromHTML(html string) string {
	patterns := []string{
		`"primaryImage":"(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`"image":"(https://[^"]+MMGS7[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`"image":"(https://[^"]+guitarcenter[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`data-image="([^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				url := match[1]
				if s.isValidImageURL(url) && !s.isHeroImage(url) {
					return url
				}
			} else if len(match) == 1 {
				url := match[0]
				if s.isValidImageURL(url) && !s.isHeroImage(url) {
					return url
				}
			}
		}
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			if len(match) >= 2 {
				url := match[1]
				if s.isValidImageURL(url) {
					return url
				}
			} else if len(match) == 1 {
				url := match[0]
				if s.isValidImageURL(url) {
					return url
				}
			}
		}
	}

	return ""
}

func (s *GuitarCenterScraper) isHeroImage(url string) bool {
	badPatterns := []string{
		"subhero", "hero", "Subhero", "Hero",
		"banner", "Banner", "carousel", "Carousel",
		"promo", "Promo", "landing", "Landing",
	}
	for _, bad := range badPatterns {
		if strings.Contains(strings.ToLower(url), bad) {
			return true
		}
	}
	return false
}

func (s *GuitarCenterScraper) isValidImageURL(url string) bool {
	if url == "" {
		return false
	}
	if !strings.HasPrefix(url, "http") {
		return false
	}
	urlLower := strings.ToLower(url)
	badPatterns := []string{
		"placeholder", "logo", "icon", "avatar", "banner",
		"transparent", "spacer", "pixel", "1x1", "data:image",
		"m26895000001000", "no-image", "no_image",
	}
	for _, bad := range badPatterns {
		if strings.Contains(urlLower, bad) {
			return false
		}
	}
	return true
}

func (s *GuitarCenterScraper) safeExtractImage(page *rod.Page) string {
	defer func() {
		if r := recover(); r != nil {
			s.logger.Debugf("[GuitarCenter] Panic during image extraction: %v", r)
		}
	}()
	return s.extractImage(page)
}
