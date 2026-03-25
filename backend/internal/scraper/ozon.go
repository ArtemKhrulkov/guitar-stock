package scraper

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/sirupsen/logrus"
)

type OzonScraper struct {
	logger *logrus.Logger
}

func NewOzonScraper() *OzonScraper {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	return &OzonScraper{logger: logger}
}

func (s *OzonScraper) Search(ctx context.Context, brand, model string) ([]SearchResult, error) {
	var results []SearchResult
	query := fmt.Sprintf("%s %s гитара", brand, model)
	searchURL := fmt.Sprintf("https://www.ozon.ru/search/?text=%s", strings.ReplaceAll(query, " ", "+"))

	s.logger.Infof("[OZON] Searching: %s", searchURL)

	searchCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	allocCtx, allocCancel := chromedp.NewExecAllocator(searchCtx,
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.DisableGPU,
	)
	defer allocCancel()

	browserCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()

	var productLinks []string
	var productTitles []string

	err := chromedp.Run(browserCtx,
		chromedp.Navigate(searchURL),
		chromedp.Sleep(5*time.Second),
		chromedp.WaitVisible(`[data-widget="searchResultsV2"]`, chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			var jsResults []map[string]string
			err := chromedp.Evaluate(`
				(function() { 
					const items = document.querySelectorAll('[data-widget="searchResultsV2"] a[href*="/product/"]'); 
					return Array.from(items).map(a => ({ href: a.href, title: a.textContent.trim() || a.getAttribute('aria-label') || '' })); 
				})()
			`, &jsResults).Do(ctx)

			if err != nil {
				s.logger.Printf("[OZON] Error evaluating: %v", err)
				return err
			}

			for _, item := range jsResults {
				if item["href"] != "" {
					productLinks = append(productLinks, item["href"])
					productTitles = append(productTitles, item["title"])
				}
			}
			return nil
		}),
	)

	if err != nil {
		s.logger.Printf("[OZON] Chromedp error: %v", err)
		return results, nil
	}

	for i, link := range productLinks {
		title := ""
		if i < len(productTitles) {
			title = strings.TrimSpace(productTitles[i])
		}

		result := SearchResult{
			URL:     link,
			Title:   title,
			InStock: true,
		}

		if link != "" {
			s.logger.Printf("[OZON] Found: %s - %s", title, link)
			results = append(results, result)
		}
	}

	if len(results) > 10 {
		results = results[:10]
	}

	s.logger.Printf("[OZON] Found %d results", len(results))
	return results, nil
}

func parsePrice(text string) *float64 {
	re := regexp.MustCompile(`[\d\s]+`)
	match := re.FindString(text)
	if match == "" {
		return nil
	}

	cleaned := strings.ReplaceAll(match, " ", "")
	price, err := strconv.ParseFloat(cleaned, 64)
	if err != nil {
		return nil
	}

	return &price
}
