package images

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/sirupsen/logrus"
)

type BingScraper struct {
	logger   *logrus.Logger
	timeout  time.Duration
	launcher *BrowserLauncher
}

func NewBingScraper() *BingScraper {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger.SetLevel(logrus.InfoLevel)

	return &BingScraper{
		logger:   logger,
		timeout:  30 * time.Second,
		launcher: NewBrowserLauncher(),
	}
}

func (s *BingScraper) Name() string {
	return "bing"
}

func (s *BingScraper) Priority() int {
	return 3
}

func (s *BingScraper) Search(ctx context.Context, brand, model string) (*ImageResult, error) {
	s.logger.Infof("[Bing] Searching for: %s %s", brand, model)

	searchURL := s.buildSearchURL(brand, model)

	instance, err := s.launcher.Launch(ctx)
	if err != nil {
		s.logger.Errorf("[Bing] Failed to launch browser: %v", err)
		return nil, err
	}
	defer instance.Close()

	page := instance.Page

	if err := page.Timeout(s.timeout).Navigate(searchURL); err != nil {
		s.logger.Warnf("[Bing] Navigation failed: %v", err)
		return nil, nil
	}

	time.Sleep(2 * time.Second)

	func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Debugf("[Bing] Page closed during wait: %v", r)
			}
		}()
		page.MustWaitLoad()
	}()

	time.Sleep(3 * time.Second)

	var imageURL string
	func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Debugf("[Bing] Panic during extraction: %v", r)
			}
		}()
		imageURL = s.extractImage(page)
	}()

	if imageURL == "" {
		s.logger.Infof("[Bing] No image found for %s %s", brand, model)
		return nil, nil
	}

	imageURLToCheck := imageURL
	bangThumbnailWidth := GetBingThumbnailWidth(imageURL)
	if bangThumbnailWidth > 0 && bangThumbnailWidth < MinWidth {
		s.logger.Debugf("[Bing] Found small thumbnail (w=%d), trying to resolve redirect...", bangThumbnailWidth)
		resolvedURL, err := ResolveBingRedirect(ctx, imageURL)
		if err != nil {
			s.logger.Debugf("[Bing] Failed to resolve redirect: %v", err)
		} else if resolvedURL != imageURL {
			s.logger.Debugf("[Bing] Resolved to: %s", resolvedURL)
			imageURLToCheck = resolvedURL
		}
	}

	width, height, err := FetchImageDimensions(imageURLToCheck)
	if err != nil {
		s.logger.Debugf("[Bing] Failed to fetch dimensions for %s: %v", imageURLToCheck, err)
		if IsSmallBingThumbnail(imageURLToCheck) {
			s.logger.Debugf("[Bing] URL indicates small image (w=%d), skipping", GetBingThumbnailWidth(imageURLToCheck))
			return nil, nil
		}
		s.logger.Infof("[Bing] Using fallback dimensions")
		width, height = 800, 600
	}

	if width < MinWidth || height < MinHeight {
		s.logger.Debugf("[Bing] Image too small: %dx%d, skipping", width, height)
		return nil, nil
	}

	result := &ImageResult{
		URL:    imageURLToCheck,
		Source: "bing",
		Width:  width,
		Height: height,
	}

	if !result.IsValid() || result.IsPlaceholder() {
		s.logger.Debugf("[Bing] Image validation failed: %dx%d, skipping", width, height)
		return nil, nil
	}

	s.logger.Infof("[Bing] Found image: %s (%dx%d)", imageURLToCheck, width, height)
	return result, nil
}

func (s *BingScraper) buildSearchURL(brand, model string) string {
	query := url.QueryEscape(fmt.Sprintf("%s %s guitar", brand, model))
	return fmt.Sprintf("https://www.bing.com/images/search?q=%s", query)
}

func (s *BingScraper) extractImage(page *rod.Page) string {
	jsCode := `
		function() {
			var images = document.querySelectorAll('img.mimg');
			var results = [];
			for (var i = 0; i < Math.min(images.length, 5); i++) {
				var img = images[i];
				var src = img.src || img.getAttribute('data-src') || img.getAttribute('data-full-url');
				if (src && src.startsWith('http') && (src.includes('.jpg') || src.includes('.jpeg') || src.includes('.png') || src.includes('.webp'))) {
					results.push(src);
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

func (s *BingScraper) fallbackExtract(page *rod.Page) string {
	selectors := []string{
		"img.mimg",
		"img[class*='img']",
		"a.iusc img",
		".img_container img",
	}

	for _, selector := range selectors {
		els, err := page.Elements(selector)
		if err == nil && len(els) > 0 {
			for _, el := range els[:3] {
				src, _ := el.Attribute("src")
				if src != nil && *src != "" && s.isValidImageURL(*src) {
					return *src
				}

				dataSrc, _ := el.Attribute("data-src")
				if dataSrc != nil && *dataSrc != "" && s.isValidImageURL(*dataSrc) {
					return *dataSrc
				}

				dataFullURL, _ := el.Attribute("data-full-url")
				if dataFullURL != nil && *dataFullURL != "" && s.isValidImageURL(*dataFullURL) {
					return *dataFullURL
				}
			}
		}
	}

	html, _ := page.HTML()
	return s.extractFromHTML(html)
}

func (s *BingScraper) extractFromHTML(html string) string {
	patterns := []string{
		`mimg src="([^"]+)"`,
		`data-src="(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`"turl":"(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`"murl":"(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			for _, m := range match {
				if s.isValidImageURL(m) {
					return m
				}
			}
		}
	}

	return ""
}

func (s *BingScraper) isValidImageURL(url string) bool {
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
