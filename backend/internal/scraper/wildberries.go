package scraper

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
	"github.com/go-rod/stealth"
	"github.com/sirupsen/logrus"
)

type WildberriesScraper struct {
	logger  *logrus.Logger
	proxies []string
	debug   bool
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
	logger.SetLevel(logrus.DebugLevel)

	return &WildberriesScraper{
		logger:  logger,
		proxies: proxies,
	}
}

func NewWildberriesScraperWithDebug() *WildberriesScraper {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger.SetLevel(logrus.DebugLevel)

	return &WildberriesScraper{
		logger:  logger,
		proxies: []string{},
		debug:   true,
	}
}

func (s *WildberriesScraper) WithProxies(proxies []string) *WildberriesScraper {
	s.proxies = proxies
	return s
}

func (s *WildberriesScraper) calculateRelevance(title, brand, model string) float64 {
	titleLower := strings.ToLower(title)
	brandLower := strings.ToLower(brand)
	modelLower := strings.ToLower(model)

	score := 0.0
	maxScore := 100.0

	if strings.Contains(titleLower, brandLower) {
		score += 50
	} else if strings.Contains(brandLower, titleLower) || strings.Contains(titleLower, strings.Split(brandLower, " ")[0]) {
		score += 25
	}

	modelWords := strings.Fields(modelLower)
	modelMatched := 0
	for _, word := range modelWords {
		if len(word) > 2 && strings.Contains(titleLower, word) {
			modelMatched++
		}
	}
	if modelMatched > 0 {
		score += float64(modelMatched) * 25 / float64(len(modelWords))
	}

	exactModel := strings.ReplaceAll(modelLower, " ", "")
	if strings.Contains(strings.ReplaceAll(titleLower, " ", ""), exactModel) {
		score += 25
	}

	badWords := []string{"чехол", "кейс", "сумка", "струны", "ремень", "тюнер", "педаль", "усилитель", "кабель", "медиатор", "стойка", "кронштейн", "home", "age", "decor", "свет", "лампа", "одежда", "игрушка", "книга", "dvd"}
	for _, bad := range badWords {
		if strings.Contains(titleLower, bad) && !strings.Contains(titleLower, brandLower) && !strings.Contains(titleLower, modelLower) {
			score -= 30
			break
		}
	}

	if score < 0 {
		score = 0
	}

	return score / maxScore
}

func (s *WildberriesScraper) isRelevantResult(title, brand, model string) bool {
	relevance := s.calculateRelevance(title, brand, model)
	s.logger.Debugf("[WB] Relevance for '%s': %.2f (brand: %s, model: %s)", title, relevance, brand, model)
	return relevance >= 0.3
}

func (s *WildberriesScraper) Search(ctx context.Context, brand, model string) ([]SearchResult, error) {
	var results []SearchResult
	query := fmt.Sprintf("%s %s гитара", brand, model)
	searchURL := fmt.Sprintf("https://www.wildberries.ru/catalog/0/search.aspx?search=%s", strings.ReplaceAll(query, " ", "%20"))

	s.logger.Infof("[WB] Searching: %s", searchURL)

	browserPath := "/usr/bin/chromium"
	l := launcher.New().
		Bin(browserPath).
		NoSandbox(true).
		Set("disable-gpu").
		Set("disable-dev-shm-usage").
		Set("disable-setuid-sandbox").
		Set("no-first-run").
		Set("no-zygote").
		Set("single-process", "false").
		Set("disable-blink-features", "AutomationControlled").
		Set("exclude-switches", "enable-automation").
		Set("disable-infobars")

	var proxyHost, proxyUser, proxyPass string
	if len(s.proxies) > 0 {
		proxyURL := s.proxies[0]
		s.logger.Infof("[WB] Using proxy: %s", maskProxy(proxyURL))

		proxyHost, proxyUser, proxyPass = parseProxyURL(proxyURL)
		if proxyHost != "" {
			l = l.Set(flags.ProxyServer, proxyHost)
		}
	}

	url, err := l.Launch()
	if err != nil {
		s.logger.Errorf("[WB] Failed to launch browser: %v", err)
		return results, err
	}

	browser := rod.New().ControlURL(url)
	if err := browser.Connect(); err != nil {
		s.logger.Errorf("[WB] Failed to connect to browser: %v", err)
		return results, err
	}
	defer browser.Close()

	browser.MustIgnoreCertErrors(true)

	if proxyUser != "" && proxyPass != "" {
		go func() {
			browser.HandleAuth(proxyUser, proxyPass)()
		}()
		s.logger.Infof("[WB] Proxy auth handler started for user: %s", proxyUser)
	}

	page := stealth.MustPage(browser)

	page.MustNavigate("https://httpbin.org/ip")
	page.MustWaitLoad()
	s.logger.Infof("[WB] Proxy auth verified")

	s.logger.Infof("[WB] Going to main page first...")
	page.Timeout(60 * time.Second).MustNavigate("https://www.wildberries.ru")
	time.Sleep(5 * time.Second)

	s.logger.Infof("[WB] Navigating to search...")
	page.MustNavigate(searchURL)
	page = page.Timeout(300 * time.Second)

	s.logger.Infof("[WB] Waiting for page to load...")
	time.Sleep(10 * time.Second)

	page.MustWaitLoad()

	s.logger.Infof("[WB] Waiting for search results to render...")
	time.Sleep(15 * time.Second)

	elements, err := page.Elements("article.product-card")
	if err != nil || len(elements) == 0 {
		s.logger.Warnf("[WB] Trying alternative selectors...")
		time.Sleep(2 * time.Second)
		elements, err = page.Elements(".j-card-item")
		if err != nil || len(elements) == 0 {
			elements, err = page.Elements("a[href*='/detail.aspx']")
		}
	}

	s.logger.Debugf("[WB] All selectors found elements: %d", len(elements))

	if err != nil {
		s.logger.Errorf("[WB] Failed to find elements: %v", err)
		return results, nil
	}

	s.logger.Infof("[WB] Found %d elements", len(elements))

	seen := make(map[string]bool)
	for i, el := range elements {
		var hrefVal *string
		var titleVal string

		linkEl, err := el.Element("a[href*='/detail.aspx']")
		if err == nil && linkEl != nil {
			hrefVal, _ = linkEl.Attribute("href")
			titleVal, _ = linkEl.Text()
			if titleVal == "" {
				if ariaLabel, _ := linkEl.Attribute("aria-label"); ariaLabel != nil {
					titleVal = *ariaLabel
				}
			}
		} else {
			hrefVal, _ = el.Attribute("href")
			titleVal, _ = el.Text()
		}

		if hrefVal == nil {
			continue
		}

		url := *hrefVal
		if strings.HasPrefix(url, "//") {
			url = "https://" + url[2:]
		} else if strings.HasPrefix(url, "/") {
			url = "https://www.wildberries.ru" + url
		}

		if seen[url] {
			continue
		}

		seen[url] = true

		if titleVal == "" {
			ariaLabel, _ := el.Attribute("aria-label")
			if ariaLabel != nil {
				titleVal = *ariaLabel
			}
		}

		if titleVal == "" {
			continue
		}

		s.logger.Debugf("[WB] Processing title: '%s'", titleVal)
		relevanceScore := s.calculateRelevance(titleVal, brand, model)
		s.logger.Debugf("[WB] Relevance for '%s': %.2f (threshold: 0.3)", titleVal, relevanceScore)

		if !s.isRelevantResult(titleVal, brand, model) {
			s.logger.Debugf("[WB] Skipping irrelevant result: %s", titleVal)
			continue
		}

		price := extractWildberriesPrice(el)

		result := SearchResult{
			URL:     url,
			Title:   strings.TrimSpace(titleVal),
			InStock: price > 0,
		}

		if price > 0 {
			result.PriceRUB = &price
		}

		s.logger.Debugf("[WB] Found: %s - %s - Price: %v RUB", titleVal, url, price)
		results = append(results, result)

		if i >= 9 {
			break
		}
	}

	if len(results) > 10 {
		results = results[:10]
	}

	s.logger.Infof("[WB] Found %d results", len(results))
	return results, nil
}

func parseProxyURL(proxyURL string) (host, username, password string) {
	proxyURL = strings.TrimPrefix(proxyURL, "http://")
	proxyURL = strings.TrimPrefix(proxyURL, "https://")

	if strings.Contains(proxyURL, "@") {
		parts := strings.Split(proxyURL, "@")
		if len(parts) == 2 {
			authParts := strings.Split(parts[0], ":")
			if len(authParts) == 2 {
				username = authParts[0]
				password = authParts[1]
			}
			host = parts[1]
		}
	} else {
		host = proxyURL
	}

	return host, username, password
}

func extractPrice(text string) float64 {
	priceRegex := regexp.MustCompile(`(\d[\d\s]*)\s*₽`)
	matches := priceRegex.FindStringSubmatch(text)
	if len(matches) >= 2 {
		priceStr := strings.ReplaceAll(matches[1], " ", "")
		price, err := strconv.ParseFloat(priceStr, 64)
		if err == nil && price >= 100 && price <= 10000000 {
			return price
		}
	}

	priceRegex2 := regexp.MustCompile(`(\d[\d\s]*)\s*руб`)
	matches2 := priceRegex2.FindStringSubmatch(text)
	if len(matches2) >= 2 {
		priceStr := strings.ReplaceAll(matches2[1], " ", "")
		price, err := strconv.ParseFloat(priceStr, 64)
		if err == nil && price >= 100 && price <= 10000000 {
			return price
		}
	}

	altRegex := regexp.MustCompile(`(\d{3,5}[\d\s]*)`)
	matches3 := altRegex.FindStringSubmatch(text)
	if len(matches3) >= 2 {
		priceStr := strings.ReplaceAll(matches3[1], " ", "")
		price, err := strconv.ParseFloat(priceStr, 64)
		if err == nil && price >= 1000 && price <= 1000000 {
			return price
		}
	}

	return 0
}

func extractPriceFromCard(text string) float64 {
	if text == "" {
		return 0
	}

	patterns := []string{
		`(\d[\d\s]*)\s*₽`,
		`(\d{3,}[\d\s]*)`,
		`(\d[\d\s]*)\s*руб`,
	}

	var maxPrice float64

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(text, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}
			priceStr := strings.ReplaceAll(match[1], " ", "")
			price, err := strconv.ParseFloat(priceStr, 64)
			if err == nil && price > maxPrice {
				maxPrice = price
			}
		}
	}

	if maxPrice < 100 {
		return 0
	}

	return maxPrice
}

func extractWildberriesPrice(el *rod.Element) float64 {
	jsCode := `
		function() {
			var text = this.innerText || this.textContent || '';
			var priceMatch = text.match(/(\d[\d\s]*)\s*₽/);
			if (priceMatch) {
				var price = priceMatch[1].replace(/\s/g, '');
				if (parseInt(price) > 1000) return price;
			}
			
			var els = this.querySelectorAll('*');
			var maxPrice = 0;
			for (var i = 0; i < els.length; i++) {
				var el = els[i];
				var elText = el.innerText || el.textContent || '';
				var m = elText.match(/^(\d[\d\s]*)\s*₽$/);
				if (m) {
					var p = parseInt(m[1].replace(/\s/g, ''));
					if (p > maxPrice && p > 1000) {
						maxPrice = p;
					}
				}
			}
			return maxPrice > 0 ? maxPrice : '';
		}
	`

	result := el.MustEval(jsCode)
	priceStr := result.Str()
	if priceStr != "" {
		price, err := strconv.ParseFloat(priceStr, 64)
		if err == nil && price > 0 {
			return price
		}
	}

	return 0
}
