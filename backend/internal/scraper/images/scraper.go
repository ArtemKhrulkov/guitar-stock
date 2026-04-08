package images

import (
	"context"
	"crypto/tls"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	MinWidth  = 300
	MinHeight = 200
)

type ImageResult struct {
	URL       string
	Source    string
	Width     int
	Height    int
	Thumbnail string
}

func (r *ImageResult) IsValid() bool {
	if r.URL == "" {
		return false
	}
	if r.Width > 0 && r.Width < MinWidth {
		return false
	}
	if r.Height > 0 && r.Height < MinHeight {
		return false
	}
	return true
}

func (r *ImageResult) IsMediumQuality() bool {
	if r.Width == 0 || r.Height == 0 {
		return false
	}
	return r.Width >= MinWidth && r.Height >= MinHeight
}

func (r *ImageResult) QualityScore() int {
	if r.Width == 0 || r.Height == 0 {
		return 0
	}
	return r.Width * r.Height
}

func (r *ImageResult) IsPlaceholder() bool {
	placeholderDomains := []string{
		"via.placeholder.com",
		"placeholder.com",
		"placehold.co",
		"placeholder.nl",
	}
	for _, domain := range placeholderDomains {
		if len(r.URL) > len(domain) && (r.URL[:len(domain)] == domain ||
			contains(r.URL, domain)) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

type ImageScraper interface {
	Name() string
	Search(ctx context.Context, brand, model string) (*ImageResult, error)
	Priority() int
}

type ImageScrapers []ImageScraper

func (s ImageScrapers) Len() int           { return len(s) }
func (s ImageScrapers) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
func (s ImageScrapers) Less(i, j int) bool { return s[i].Priority() < s[j].Priority() }

func FetchImageDimensions(url string) (width, height int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/*")
	req.Header.Set("Accept-Encoding", "identity")

	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("non-200 status: %d", resp.StatusCode)
	}

	limitReader := io.LimitReader(resp.Body, 512*1024)
	config, format, err := image.DecodeConfig(limitReader)
	if err != nil {
		return 0, 0, fmt.Errorf("decode failed: %w (format: %s)", err, format)
	}

	return config.Width, config.Height, nil
}

func GetBingThumbnailWidth(urlStr string) int {
	u, err := url.Parse(urlStr)
	if err != nil {
		return 0
	}
	query := u.Query()
	if w := query.Get("w"); w != "" {
		if width, err := strconv.Atoi(w); err == nil {
			return width
		}
	}
	return 0
}

func IsSmallBingThumbnail(urlStr string) bool {
	if !strings.Contains(urlStr, "bing.com") {
		return false
	}
	width := GetBingThumbnailWidth(urlStr)
	return width > 0 && width < MinWidth
}

func ResolveBingRedirect(ctx context.Context, thumbnailURL string) (string, error) {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport, Timeout: 15 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", thumbnailURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/*")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.Request.URL != nil && resp.Request.URL.String() != thumbnailURL {
		return resp.Request.URL.String(), nil
	}

	return thumbnailURL, nil
}
