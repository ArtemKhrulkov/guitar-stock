package images

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type SweetwaterScraper struct {
	client  *http.Client
	logger  *logrus.Logger
	timeout time.Duration
}

func NewSweetwaterScraper() *SweetwaterScraper {
	return &SweetwaterScraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:  logrus.New(),
		timeout: 30 * time.Second,
	}
}

func (s *SweetwaterScraper) Name() string {
	return "sweetwater"
}

func (s *SweetwaterScraper) Priority() int {
	return 1
}

func (s *SweetwaterScraper) Search(ctx context.Context, brand, model string) (*ImageResult, error) {
	s.logger.Infof("[Sweetwater] Searching for: %s %s", brand, model)

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	query := fmt.Sprintf("%s %s guitar", brand, model)
	searchURL := fmt.Sprintf("https://www.sweetwater.com/store/detail/%s", url.QueryEscape(query))

	searchURL = s.buildSearchURL(brand, model)
	s.logger.Debugf("[Sweetwater] Search URL: %s", searchURL)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}

	s.setHeaders(req)

	resp, err := s.client.Do(req)
	if err != nil {
		s.logger.Errorf("[Sweetwater] Request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	var imageURL string

	if resp.StatusCode != http.StatusOK {
		s.logger.Warnf("[Sweetwater] Non-200 response: %d, trying API fallback", resp.StatusCode)
		imageURL = s.tryAPI(ctx, brand, model)
		if imageURL == "" {
			return nil, nil
		}
		return &ImageResult{
			URL:    imageURL,
			Source: "sweetwater",
			Width:  1400,
			Height: 1000,
		}, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	imageURL = s.extractImageFromHTML(string(body))
	if imageURL == "" {
		s.logger.Debugf("[Sweetwater] No image found in HTML, trying API")
		imageURL = s.tryAPI(ctx, brand, model)
	}

	if imageURL == "" {
		s.logger.Infof("[Sweetwater] No image found for %s %s", brand, model)
		return nil, nil
	}

	result := &ImageResult{
		URL:    imageURL,
		Source: "sweetwater",
		Width:  1400,
		Height: 1000,
	}

	if !result.IsValid() || result.IsPlaceholder() {
		return nil, nil
	}

	s.logger.Infof("[Sweetwater] Found image: %s", imageURL)
	return result, nil
}

func (s *SweetwaterScraper) buildSearchURL(brand, model string) string {
	query := strings.ReplaceAll(fmt.Sprintf("%s %s", brand, model), " ", "-")
	query = strings.ToLower(query)
	query = regexp.MustCompile(`[^a-z0-9\-]+`).ReplaceAllString(query, "-")
	query = strings.Trim(query, "-")

	return fmt.Sprintf("https://www.sweetwater.com/store/detail/%s", query)
}

func (s *SweetwaterScraper) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}

func (s *SweetwaterScraper) extractImageFromHTML(html string) string {
	patterns := []string{
		`"image":"(https://[^"]+sweetwater[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`"primaryImage":"(https://[^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`data-image="([^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`<img[^>]+class="[^"]*product[^"]*"[^>]+src="([^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`<img[^>]+id="product-image"[^>]+src="([^"]+\.(?:jpg|jpeg|png|webp)(?:\?[^"]*)?)"`,
		`https://c1\.staticsweetwater\.com[^"']+\.(?:jpg|jpeg|png|webp)(?:\?[^"']*)?`,
		`https://media\.sweetwater\.com[^"']+\.(?:jpg|jpeg|png|webp)(?:\?[^"']*)?`,
		`https://i\.sweetwater\.com[^"']+\.(?:jpg|jpeg|png|webp)(?:\?[^"']*)?`,
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

func (s *SweetwaterScraper) isValidImageURL(url string) bool {
	if strings.Contains(url, "placeholder") || strings.Contains(url, "logo") {
		return false
	}
	if strings.Contains(url, ".svg") || strings.Contains(url, ".gif") {
		return false
	}
	return true
}

func (s *SweetwaterScraper) tryAPI(ctx context.Context, brand, model string) string {
	query := fmt.Sprintf("%s %s guitar", brand, model)
	apiURL := fmt.Sprintf("https://www.sweetwater.com/api/ss/v1/search?term=%s&limit=5", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return ""
	}

	s.setHeaders(req)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var result struct {
		Products []struct {
			Name         string `json:"name"`
			PrimaryImage struct {
				URL string `json:"url"`
			} `json:"primaryImage"`
		} `json:"products"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}

	for _, product := range result.Products {
		if product.PrimaryImage.URL != "" {
			return product.PrimaryImage.URL
		}
	}

	return ""
}
