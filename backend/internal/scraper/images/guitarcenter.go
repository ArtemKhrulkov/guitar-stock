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

	imageURL := s.extractImage(instance.Page)

	if imageURL == "" {
		s.logger.Infof("[GuitarCenter] No image found for %s %s", brand, model)
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
			var img = document.querySelector('img.product-image') || 
			          document.querySelector('[data-testid="product-image"] img') ||
			          document.querySelector('.product-detail-image img') ||
			          document.querySelector('.product-media-gallery img');
			if (img) {
				return img.src || img.getAttribute('data-src') || '';
			}
			return '';
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
		for _, el := range els[:3] {
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
		`"image":"(https://[^"]+guitarcenter[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`"primaryImage":"(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`data-image="([^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`https://[^"]*guitarcenter[^"]*\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?`,
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

func (s *GuitarCenterScraper) isValidImageURL(url string) bool {
	if url == "" {
		return false
	}
	if !strings.HasPrefix(url, "http") {
		return false
	}
	badPatterns := []string{"placeholder", "logo", "icon", "avatar", "banner"}
	for _, bad := range badPatterns {
		if strings.Contains(strings.ToLower(url), bad) {
			return false
		}
	}
	return true
}
