package scraper

import (
	"context"
	"fmt"
	"strings"

	"github.com/gocolly/colly"
	"github.com/gocolly/colly/proxy"
	"github.com/sirupsen/logrus"
)

type WildberriesScraper struct {
	logger  *logrus.Logger
	proxies []string
}

func NewWildberriesScraper() *WildberriesScraper {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	return &WildberriesScraper{
		logger:  logger,
		proxies: []string{},
	}
}

func NewWildberriesScraperWithProxies(proxies []string) *WildberriesScraper {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	return &WildberriesScraper{
		logger:  logger,
		proxies: proxies,
	}
}

func (s *WildberriesScraper) Search(ctx context.Context, brand, model string) ([]SearchResult, error) {
	var results []SearchResult
	query := fmt.Sprintf("%s %s гитара", brand, model)
	searchURL := fmt.Sprintf("https://www.wildberries.ru/catalog/0/search.aspx?search=%s", strings.ReplaceAll(query, " ", "%20"))

	s.logger.Infof("[WB] Searching: %s", searchURL)

	c := colly.NewCollector(
		colly.AllowedDomains("wildberries.ru", "www.wildberries.ru"),
		colly.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36"),
		colly.MaxDepth(2),
	)

	if len(s.proxies) > 0 {
		rp, err := proxy.RoundRobinProxySwitcher(s.proxies...)
		if err == nil {
			c.SetProxyFunc(rp)
			s.logger.Infof("[WB] Using %d proxies", len(s.proxies))
		}
	}

	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
		r.Headers.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")
		r.Headers.Set("Accept-Encoding", "gzip, deflate, br")
		r.Headers.Set("Connection", "keep-alive")
		r.Headers.Set("Upgrade-Insecure-Requests", "1")
		r.Headers.Set("Sec-Fetch-Dest", "document")
		r.Headers.Set("Sec-Fetch-Mode", "navigate")
		r.Headers.Set("Sec-Fetch-Site", "none")
		r.Headers.Set("Sec-Fetch-User", "?1")
		r.Headers.Set("Cache-Control", "max-age=0")
	})

	c.OnHTML("a[href*='/catalog/']", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		if href == "" {
			return
		}

		if !strings.HasPrefix(href, "http") {
			href = "https://www.wildberries.ru" + href
		}

		title := strings.TrimSpace(e.Text)
		if title == "" {
			title = e.Attr("data-link-name")
		}

		result := SearchResult{
			URL:     href,
			Title:   title,
			InStock: true,
		}

		s.logger.Printf("[WB] Found: %s - %s", title, href)
		results = append(results, result)
	})

	c.OnError(func(r *colly.Response, err error) {
		s.logger.Printf("[WB] Colly error (status %d): %v", r.StatusCode, err)
	})

	err := c.Visit(searchURL)
	if err != nil {
		s.logger.Printf("[WB] Visit error: %v", err)
		return results, nil
	}

	c.Wait()

	if len(results) > 10 {
		results = results[:10]
	}

	s.logger.Printf("[WB] Found %d results", len(results))
	return results, nil
}
