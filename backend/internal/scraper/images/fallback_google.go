package images

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/sirupsen/logrus"
)

type FallbackGoogleScraper struct {
	logger   *logrus.Logger
	timeout  time.Duration
	launcher *BrowserLauncher
}

func NewFallbackGoogleScraper() *FallbackGoogleScraper {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger.SetLevel(logrus.InfoLevel)

	return &FallbackGoogleScraper{
		logger:   logger,
		timeout:  45 * time.Second,
		launcher: NewBrowserLauncher(),
	}
}

func (s *FallbackGoogleScraper) Name() string {
	return "fallback-google"
}

func (s *FallbackGoogleScraper) Priority() int {
	return 10
}

func (s *FallbackGoogleScraper) Search(ctx context.Context, brand, model string) (*ImageResult, error) {
	s.logger.Infof("[FallbackGoogle] Searching for: %s %s", brand, model)

	queries := s.getSearchQueries(brand, model)

	for qi, query := range queries {
		s.logger.Debugf("[FallbackGoogle] Trying query %d/%d: %s", qi+1, len(queries), query)

		imageURL := s.tryQuery(ctx, query)
		if imageURL == "" {
			continue
		}

		thumbnailWidth := 0
		imageURLToCheck := imageURL
		if strings.Contains(imageURL, "google.com") || strings.Contains(imageURL, "gstatic.com") {
			if u, err := url.Parse(imageURL); err == nil {
				if w := u.Query().Get("w"); w != "" {
					if parsedW, err := strconv.Atoi(w); err == nil {
						thumbnailWidth = parsedW
					}
				}
			}
			if thumbnailWidth > 0 && thumbnailWidth < MinWidth {
				s.logger.Debugf("[FallbackGoogle] Found small thumbnail (%dx%d), resolving redirect...", thumbnailWidth)
				resolvedURL, err := ResolveBingRedirect(ctx, imageURL)
				if err != nil {
					s.logger.Debugf("[FallbackGoogle] Failed to resolve redirect: %v", err)
				} else if resolvedURL != imageURL {
					s.logger.Debugf("[FallbackGoogle] Resolved to: %s", resolvedURL)
					imageURLToCheck = resolvedURL
				}
			}
		}

		width, height, err := FetchImageDimensions(imageURLToCheck)
		if err != nil {
			s.logger.Debugf("[FallbackGoogle] Failed to fetch dimensions for %s: %v", imageURLToCheck, err)
			width, height = 0, 0
		}

		if width < MinWidth || height < MinHeight {
			s.logger.Debugf("[FallbackGoogle] Image too small (%dx%d), trying next query", width, height)
			continue
		}

		result := &ImageResult{
			URL:    imageURLToCheck,
			Source: "fallback-google",
			Width:  width,
			Height: height,
		}

		if !result.IsValid() || result.IsPlaceholder() {
			s.logger.Debugf("[FallbackGoogle] Image validation failed: %dx%d, trying next query", width, height)
			continue
		}

		s.logger.Infof("[FallbackGoogle] Found image via query %d: %s (%dx%d)", qi+1, imageURLToCheck, width, height)
		return result, nil
	}

	s.logger.Infof("[FallbackGoogle] No suitable image found for %s %s after %d queries", brand, model, len(queries))
	return nil, nil
}

func (s *FallbackGoogleScraper) getSearchQueries(brand, model string) []string {
	queries := []string{
		fmt.Sprintf("%s %s guitar official site", brand, model),
		fmt.Sprintf("%s %s electric guitar", brand, model),
		fmt.Sprintf("%s %s acoustic guitar", brand, model),
		fmt.Sprintf("%s %s guitar product", brand, model),
		fmt.Sprintf("%s %s guitar", brand, model),
	}
	return queries
}

func (s *FallbackGoogleScraper) tryQuery(ctx context.Context, query string) string {
	searchURL := s.buildSearchURL(query)
	s.logger.Debugf("[FallbackGoogle] Search URL: %s", searchURL)

	instance, err := s.launcher.Launch(ctx)
	if err != nil {
		s.logger.Errorf("[FallbackGoogle] Failed to launch browser: %v", err)
		return ""
	}
	defer instance.Close()

	page := instance.Page

	if err := page.Timeout(s.timeout).Navigate(searchURL); err != nil {
		s.logger.Warnf("[FallbackGoogle] Navigation failed: %v", err)
		return ""
	}

	time.Sleep(2 * time.Second)

	func() {
		defer func() {
			if r := recover(); r != nil {
				s.logger.Debugf("[FallbackGoogle] Page closed during wait: %v", r)
			}
		}()
		page.MustWaitLoad()
	}()

	time.Sleep(3 * time.Second)

	imageURL := s.extractImage(page)
	if imageURL == "" {
		s.logger.Debugf("[FallbackGoogle] No images found for query: %s", query)
		return ""
	}

	s.logger.Debugf("[FallbackGoogle] Found image URL: %s", imageURL)
	return imageURL
}

func (s *FallbackGoogleScraper) buildSearchURL(query string) string {
	return fmt.Sprintf("https://www.google.com/search?q=%s&tbm=isch", url.QueryEscape(query))
}

func (s *FallbackGoogleScraper) extractImage(page *rod.Page) string {
	jsCode := `
		function() {
			var results = [];
			
			var imgs = document.querySelectorAll('img.Q4LuWd');
			for (var i = 0; i < Math.min(imgs.length, 10); i++) {
				var img = imgs[i];
				var src = img.getAttribute('data-src') || img.src;
				if (src && src.startsWith('http') && (src.includes('.jpg') || src.includes('.jpeg') || src.includes('.png') || src.includes('.webp'))) {
					if (!src.includes('gstatic.com') && !src.includes('google.com/images')) {
						results.push(src);
					}
				}
			}
			
			if (results.length === 0) {
				var allImages = document.querySelectorAll('img');
				for (var i = 0; i < Math.min(allImages.length, 10); i++) {
					var img = allImages[i];
					var src = img.src || img.getAttribute('data-src');
					if (src && src.startsWith('http') && (src.includes('.jpg') || src.includes('.jpeg') || src.includes('.png') || src.includes('.webp'))) {
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

func (s *FallbackGoogleScraper) fallbackExtract(page *rod.Page) string {
	selectors := []string{
		"img.Q4vWd",
		"img[class*='img']",
		"a.iusc img",
	}

	for _, selector := range selectors {
		els, err := page.Elements(selector)
		if err == nil && len(els) > 0 {
			for _, el := range els[:5] {
				src, _ := el.Attribute("src")
				if src != nil && *src != "" && s.isValidImageURL(*src) {
					return *src
				}

				dataSrc, _ := el.Attribute("data-src")
				if dataSrc != nil && *dataSrc != "" && s.isValidImageURL(*dataSrc) {
					return *dataSrc
				}
			}
		}
	}

	html, _ := page.HTML()
	return s.extractFromHTML(html)
}

func (s *FallbackGoogleScraper) extractFromHTML(html string) string {
	patterns := []string{
		`"tu"(?:\s*:\s*)?"(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`"ou":"(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`src="(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
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
			}
		}
	}

	return ""
}

func (s *FallbackGoogleScraper) isValidImageURL(url string) bool {
	if url == "" {
		return false
	}
	if !strings.HasPrefix(url, "http") {
		return false
	}
	badPatterns := []string{
		"placeholder", "logo", "icon", "avatar", "banner",
		"transparent", "spacer", "pixel", "1x1", "data:image",
		"gstatic.com", "google.com/images", "googleusercontent.com",
	}
	for _, bad := range badPatterns {
		if strings.Contains(strings.ToLower(url), bad) {
			return false
		}
	}
	return true
}
