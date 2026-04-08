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

type ManufacturerScraper struct {
	logger   *logrus.Logger
	timeout  time.Duration
	launcher *BrowserLauncher
}

func NewManufacturerScraper() *ManufacturerScraper {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger.SetLevel(logrus.InfoLevel)

	return &ManufacturerScraper{
		logger:   logger,
		timeout:  15 * time.Second,
		launcher: NewBrowserLauncher(),
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

	imageURL := s.tryHTTP(searchURL)
	if imageURL != "" {
		width, height, err := FetchImageDimensions(imageURL)
		if err != nil {
			s.logger.Debugf("[Manufacturer] Failed to fetch dimensions for %s: %v", imageURL, err)
			s.logger.Infof("[Manufacturer] Using fallback dimensions")
			width, height = 1200, 800
		}
		if width < MinWidth || height < MinHeight {
			s.logger.Debugf("[Manufacturer] Image too small: %dx%d, skipping", width, height)
			return nil, nil
		}
		result := &ImageResult{
			URL:    imageURL,
			Source: "manufacturer:" + strings.ToLower(brand),
			Width:  width,
			Height: height,
		}
		if !result.IsPlaceholder() {
			s.logger.Infof("[Manufacturer] Found image via HTTP: %s", imageURL)
			return result, nil
		}
	}

	if !s.launcher.HasBrowser() {
		s.logger.Debugf("[Manufacturer] No browser available, skipping browser fallback")
		return nil, nil
	}

	launchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	instance, err := s.launcher.Launch(launchCtx)
	if err != nil {
		s.logger.Debugf("[Manufacturer] Failed to launch browser: %v", err)
		return nil, nil
	}
	defer instance.Close()

	s.logger.Infof("[Manufacturer] Trying browser fallback...")

	if err := instance.Page.Timeout(30 * time.Second).Navigate(searchURL); err != nil {
		s.logger.Debugf("[Manufacturer] Browser navigation failed: %v", err)
		return nil, nil
	}

	time.Sleep(3 * time.Second)

	imageURL = s.extractImage(instance.Page)
	if imageURL == "" {
		s.logger.Debugf("[Manufacturer] No image found from browser for %s %s", brand, model)
		return nil, nil
	}

	width, height, err := FetchImageDimensions(imageURL)
	if err != nil {
		s.logger.Debugf("[Manufacturer] Failed to fetch dimensions for %s: %v", imageURL, err)
		s.logger.Infof("[Manufacturer] Using fallback dimensions")
		width, height = 1200, 800
	}

	if width < MinWidth || height < MinHeight {
		s.logger.Debugf("[Manufacturer] Image too small: %dx%d, skipping", width, height)
		return nil, nil
	}

	result := &ImageResult{
		URL:    imageURL,
		Source: "manufacturer:" + strings.ToLower(brand),
		Width:  width,
		Height: height,
	}

	if result.IsPlaceholder() {
		return nil, nil
	}

	s.logger.Infof("[Manufacturer] Found image: %s", imageURL)
	return result, nil
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

func (s *ManufacturerScraper) tryHTTP(urlStr string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
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

	return s.extractImageFromHTML(string(body))
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
