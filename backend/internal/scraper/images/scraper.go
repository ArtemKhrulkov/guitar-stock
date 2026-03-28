package images

import (
	"context"
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
	if r.Width > 0 && r.Width < 400 {
		return false
	}
	if r.Height > 0 && r.Height < 300 {
		return false
	}
	return true
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
