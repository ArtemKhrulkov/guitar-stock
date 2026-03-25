package scraper

import (
	"context"
)

type Platform string

const (
	Ozon        Platform = "ozon"
	Wildberries Platform = "wildberries"
)

type SearchResult struct {
	URL      string
	Title    string
	PriceRUB *float64
	PriceUSD *float64
	InStock  bool
}

type Scraper interface {
	Search(ctx context.Context, brand, model string) ([]SearchResult, error)
}

func NewScraper(platform Platform) Scraper {
	return NewScraperWithProxies(platform, nil)
}

func NewScraperWithProxies(platform Platform, proxies []string) Scraper {
	switch platform {
	case Ozon:
		if len(proxies) > 0 {
			return NewOzonScraperWithProxies(proxies)
		}
		return NewOzonScraper()
	case Wildberries:
		if len(proxies) > 0 {
			return NewWildberriesScraperWithProxies(proxies)
		}
		return NewWildberriesScraper()
	default:
		return NewOzonScraper()
	}
}
