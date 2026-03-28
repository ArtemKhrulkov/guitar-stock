package images

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
	"github.com/sirupsen/logrus"
)

type GoogleScraper struct {
	logger  *logrus.Logger
	timeout time.Duration
}

func NewGoogleScraper() *GoogleScraper {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger.SetLevel(logrus.InfoLevel)

	return &GoogleScraper{
		logger:  logger,
		timeout: 60 * time.Second,
	}
}

func (s *GoogleScraper) Name() string {
	return "google"
}

func (s *GoogleScraper) Priority() int {
	return 4
}

func (s *GoogleScraper) Search(ctx context.Context, brand, model string) (*ImageResult, error) {
	s.logger.Infof("[Google] Searching for: %s %s", brand, model)

	searchURL := s.buildSearchURL(brand, model)
	s.logger.Debugf("[Google] Search URL: %s", searchURL)

	browser, page, err := s.launchBrowser(ctx)
	if err != nil {
		s.logger.Errorf("[Google] Failed to launch browser: %v", err)
		return nil, err
	}
	defer browser.Close()

	if err := page.Timeout(s.timeout).Navigate(searchURL); err != nil {
		s.logger.Warnf("[Google] Navigation failed: %v", err)
		return nil, nil
	}

	time.Sleep(5 * time.Second)
	page.MustWaitLoad()

	time.Sleep(5 * time.Second)

	imageURL := s.extractImage(page)

	if imageURL == "" {
		s.logger.Infof("[Google] No image found for %s %s", brand, model)
		return nil, nil
	}

	result := &ImageResult{
		URL:    imageURL,
		Source: "google",
		Width:  800,
		Height: 600,
	}

	if !result.IsValid() || result.IsPlaceholder() {
		return nil, nil
	}

	s.logger.Infof("[Google] Found image: %s", imageURL)
	return result, nil
}

func (s *GoogleScraper) buildSearchURL(brand, model string) string {
	query := url.QueryEscape(fmt.Sprintf("%s %s guitar official", brand, model))
	return fmt.Sprintf("https://www.google.com/search?q=%s&tbm=isch", query)
}

func (s *GoogleScraper) launchBrowser(ctx context.Context) (*rod.Browser, *rod.Page, error) {
	browserPath := "/usr/bin/chromium"
	l := launcher.New().
		Bin(browserPath).
		NoSandbox(true).
		Set("disable-gpu").
		Set("disable-dev-shm-usage").
		Set("disable-setuid-sandbox").
		Set("no-first-run").
		Set("no-zygote").
		Set("disable-blink-features", "AutomationControlled").
		Set("exclude-switches", "enable-automation").
		Set("disable-infobars").
		Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	urlStr, err := l.Launch()
	if err != nil {
		return nil, nil, err
	}

	browser := rod.New().ControlURL(urlStr)
	if err := browser.Connect(); err != nil {
		return nil, nil, err
	}

	browser.MustIgnoreCertErrors(true)
	page := stealth.MustPage(browser)

	return browser, page, nil
}

func (s *GoogleScraper) extractImage(page *rod.Page) string {
	jsCode := `
		function() {
			var results = [];
			
			var imgs = document.querySelectorAll('img.Q4LuWd');
			for (var i = 0; i < Math.min(imgs.length, 5); i++) {
				var img = imgs[i];
				var src = img.getAttribute('data-src') || img.src;
				if (src && src.startsWith('http') && (src.includes('.jpg') || src.includes('.jpeg') || src.includes('.png') || src.includes('.webp'))) {
					if (!src.includes('gstatic.com') && !src.includes('google.com/images')) {
						results.push(src);
					}
				}
			}
			
			if (results.length === 0) {
				var allImgs = document.querySelectorAll('img');
				for (var i = 0; i < Math.min(allImgs.length, 10); i++) {
					var img = allImgs[i];
					var src = img.getAttribute('data-src') || img.src;
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

func (s *GoogleScraper) fallbackExtract(page *rod.Page) string {
	jsCode := `
		function() {
			var results = [];
			var links = document.querySelectorAll('a.wXeWr');
			for (var i = 0; i < Math.min(links.length, 5); i++) {
				var link = links[i];
				var img = link.querySelector('img');
				if (img) {
					var src = img.src || img.getAttribute('data-src');
					if (src && src.startsWith('http')) {
						results.push(src);
					}
				}
			}
			return results.length > 0 ? results[0] : '';
		}
	`

	result := page.MustEval(jsCode)
	return result.Str()
}

func (s *GoogleScraper) extractFromHTML(html string) string {
	patterns := []string{
		`"ou":"(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`"ru":"(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`data-src="(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`src="(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
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

func (s *GoogleScraper) isValidImageURL(url string) bool {
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
