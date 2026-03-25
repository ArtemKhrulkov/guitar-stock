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
	switch platform {
	case Ozon:
		return NewOzonScraper()
	case Wildberries:
		return NewWildberriesScraper()
	default:
		return NewOzonScraper()
	}
}
