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
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
	"github.com/sirupsen/logrus"
)

type ManufacturerScraper struct {
	logger  *logrus.Logger
	timeout time.Duration
	browser *rod.Browser
}

func NewManufacturerScraper() *ManufacturerScraper {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger.SetLevel(logrus.InfoLevel)

	return &ManufacturerScraper{
		logger:  logger,
		timeout: 60 * time.Second,
	}
}

func (s *ManufacturerScraper) Name() string {
	return "manufacturer"
}

func (s *ManufacturerScraper) Priority() int {
	return 2
}

func (s *ManufacturerScraper) Search(ctx context.Context, brand, model string) (*ImageResult, error) {
	s.logger.Infof("[Manufacturer] Searching for: %s %s", brand, model)

	searchURL := s.buildSearchURL(brand, model)
	if searchURL == "" {
		s.logger.Infof("[Manufacturer] No search URL for brand: %s", brand)
		return nil, nil
	}

	s.logger.Infof("[Manufacturer] Search URL: %s", searchURL)

	browser, page, err := s.launchBrowser(ctx)
	if err != nil {
		s.logger.Errorf("[Manufacturer] Failed to launch browser: %v", err)
		return nil, err
	}
	defer browser.Close()

	s.logger.Infof("[Manufacturer] Navigating to %s...", searchURL)
	if err := page.Timeout(30 * time.Second).Navigate(searchURL); err != nil {
		s.logger.Warnf("[Manufacturer] Navigation failed: %v", err)
		return nil, nil
	}
	s.logger.Infof("[Manufacturer] Page loaded, waiting...")

	time.Sleep(3 * time.Second)

	imageURL := s.extractImage(page)

	if imageURL == "" {
		s.logger.Infof("[Manufacturer] No image found for %s %s from %s", brand, model, searchURL)
		return nil, nil
	}

	result := &ImageResult{
		URL:    imageURL,
		Source: "manufacturer:" + strings.ToLower(brand),
		Width:  1200,
		Height: 800,
	}

	if !result.IsValid() || result.IsPlaceholder() {
		return nil, nil
	}

	s.logger.Infof("[Manufacturer] Found image: %s", imageURL)
	return result, nil
}

func (s *ManufacturerScraper) launchBrowser(ctx context.Context) (*rod.Browser, *rod.Page, error) {
	s.logger.Info("[Manufacturer] Launching browser...")

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
		Set("headless").
		Set("disable-web-security")

	s.logger.Info("[Manufacturer] Browser config ready, launching...")

	urlStr, err := l.Launch()
	if err != nil {
		s.logger.Errorf("[Manufacturer] Failed to launch: %v", err)
		return nil, nil, err
	}

	s.logger.Info("[Manufacturer] Browser launched successfully")

	browser := rod.New().ControlURL(urlStr)
	if err := browser.Connect(); err != nil {
		return nil, nil, err
	}

	browser.MustIgnoreCertErrors(true)
	page := stealth.MustPage(browser)

	return browser, page, nil
}

func (s *ManufacturerScraper) buildSearchURL(brand, model string) string {
	brandLower := strings.ToLower(brand)
	modelSlug := s.slugify(model)

	switch brandLower {
	case "gibson":
		return fmt.Sprintf("https://www.gibson.com/en-us/guitars/%s", modelSlug)
	case "fender":
		return fmt.Sprintf("https://www.fender.com/en-US/search?q=%s", url.QueryEscape(model))
	case "ibanez":
		return fmt.Sprintf("https://www.ibanez.com/usa/products/electric-guitars/%s", modelSlug)
	case "esp":
		return fmt.Sprintf("https://www.espguitars.com/products/search?q=%s", url.QueryEscape(model))
	case "schecter":
		return fmt.Sprintf("https://www.schecterguitars.com/products/search?q=%s", url.QueryEscape(model))
	case "yamaha":
		return fmt.Sprintf("https://usa.yamaha.com/products/musical_instruments/guitars_basses/electric_guitars/index.html?search=%s", url.QueryEscape(model))
	case "music man", "musicman":
		return fmt.Sprintf("https://www.music-man.com/family-list?search=%s", url.QueryEscape(model))
	case "greco":
		return s.buildGrecoURL(model)
	case "burny":
		return fmt.Sprintf("https://burny.jp/products/%s", modelSlug)
	case "squier":
		return fmt.Sprintf("https://www.fender.com/en-US/squier/?search=%s", url.QueryEscape(model))
	case "gretsch":
		return fmt.Sprintf("https://www.gretschguitars.com/search?q=%s", url.QueryEscape(model))
	case "sterling", "sterling by music man", "sterlingbymusicman":
		return fmt.Sprintf("https://www.sterlingbymusicman.com/search?q=%s", url.QueryEscape(model))
	default:
		return fmt.Sprintf("https://www.google.com/search?q=%s&tbm=isch", url.QueryEscape(fmt.Sprintf("%s %s guitar", brand, model)))
	}
}

func (s *ManufacturerScraper) buildGrecoURL(model string) string {
	modelSlug := s.slugify(model)
	return fmt.Sprintf("https://www.greco.co.jp/products/%s", modelSlug)
}

func (s *ManufacturerScraper) slugify(text string) string {
	text = strings.ToLower(text)
	text = strings.ReplaceAll(text, " ", "-")
	text = strings.ReplaceAll(text, "'", "")
	text = strings.ReplaceAll(text, "\"", "")
	text = strings.ReplaceAll(text, ".", "")
	text = strings.ReplaceAll(text, "/", "-")

	re := regexp.MustCompile(`[^a-z0-9\-]+`)
	text = re.ReplaceAllString(text, "-")
	text = strings.Trim(text, "-")
	text = strings.Trim(text, "-")

	for strings.Contains(text, "--") {
		text = strings.ReplaceAll(text, "--", "-")
	}

	return text
}

func (s *ManufacturerScraper) extractImage(page *rod.Page) string {
	selectors := []string{
		"img.product-image",
		"img[class*='product']",
		"img[class*='gallery']",
		"img[class*='main']",
		"img[class*='hero']",
		"div.product-image img",
		"div.gallery img",
		"figure img",
		"[data-image] img",
		"meta[property='og:image']",
		"meta[itemprop='image']",
	}

	for _, selector := range selectors {
		if strings.HasPrefix(selector, "meta") {
			el, err := page.Element(selector)
			if err == nil {
				if content, _ := el.Attribute("content"); content != nil && *content != "" {
					if strings.Contains(*content, "http") && (strings.Contains(*content, ".jpg") || strings.Contains(*content, ".jpeg") || strings.Contains(*content, ".png") || strings.Contains(*content, ".webp")) {
						return *content
					}
				}
			}
		} else {
			el, err := page.Element(selector)
			if err == nil {
				src, _ := el.Attribute("src")
				if src != nil && *src != "" && s.isValidImageURL(*src) {
					return *src
				}

				dataSrc, _ := el.Attribute("data-src")
				if dataSrc != nil && *dataSrc != "" && s.isValidImageURL(*dataSrc) {
					return *dataSrc
				}

				dataSrcSet, _ := el.Attribute("data-srcset")
				if dataSrcSet != nil && *dataSrcSet != "" {
					urls := strings.Split(*dataSrcSet, ",")
					for _, u := range urls {
						parts := strings.Split(strings.Trim(u, " "), " ")
						if len(parts) >= 1 && s.isValidImageURL(parts[0]) {
							return parts[0]
						}
					}
				}
			}
		}
	}

	html, _ := page.HTML()
	return s.extractImageFromHTML(html)
}

func (s *ManufacturerScraper) extractImageFromHTML(html string) string {
	patterns := []string{
		`"image":"(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`og:image["\s]+content="([^"]+)"`,
		`property=["']og:image["']\s+content=["']([^"']+)["']`,
		`src="(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"[^>]*>`,
		`data-src="(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
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

func (s *ManufacturerScraper) isValidImageURL(url string) bool {
	if url == "" {
		return false
	}
	if !strings.HasPrefix(url, "http") {
		return false
	}
	if strings.Contains(url, "placeholder") || strings.Contains(url, "logo") || strings.Contains(url, "icon") {
		return false
	}
	if !strings.Contains(url, ".jpg") && !strings.Contains(url, ".jpeg") && !strings.Contains(url, ".png") && !strings.Contains(url, ".webp") && !strings.Contains(url, ".gif") {
		return false
	}
	return true
}

func (s *ManufacturerScraper) getHTTPImage(ctx context.Context, urlStr string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 response: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return s.extractImageFromHTML(string(body)), nil
}
