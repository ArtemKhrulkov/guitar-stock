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

	"github.com/go-rod/rod"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type WildberriesImageScraper struct {
	logger   *logrus.Logger
	timeout  time.Duration
	launcher *BrowserLauncher
	db       *gorm.DB
}

func NewWildberriesImageScraper(db *gorm.DB) *WildberriesImageScraper {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger.SetLevel(logrus.InfoLevel)

	return &WildberriesImageScraper{
		logger:   logger,
		timeout:  60 * time.Second,
		launcher: NewBrowserLauncher(),
		db:       db,
	}
}

func (s *WildberriesImageScraper) Name() string {
	return "wildberries"
}

func (s *WildberriesImageScraper) Priority() int {
	return 5
}

func (s *WildberriesImageScraper) Search(ctx context.Context, brand, model string) (*ImageResult, error) {
	s.logger.Infof("[Wildberries] Searching for: %s %s", brand, model)

	productURLs := s.getExistingPurchaseLinks(ctx, brand, model)
	if len(productURLs) > 0 {
		s.logger.Infof("[Wildberries] Found %d existing purchase links", len(productURLs))
		for _, productURL := range productURLs {
			imageURL, err := s.extractImageFromURL(ctx, productURL)
			if err != nil {
				s.logger.Warnf("[Wildberries] Failed to extract from %s: %v", productURL, err)
				continue
			}
			if imageURL != "" && s.isValidProductImage(imageURL) {
				s.logger.Infof("[Wildberries] Found image from existing link: %s", imageURL)
				return &ImageResult{
					URL:    imageURL,
					Source: "wildberries",
					Width:  800,
					Height: 600,
				}, nil
			}
		}
	}

	s.logger.Infof("[Wildberries] No existing links, searching for: %s %s", brand, model)
	searchURLs := s.buildSearchURLs(brand, model)
	for _, searchURL := range searchURLs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		productURL, err := s.findProductURL(ctx, searchURL)
		if err != nil || productURL == "" {
			continue
		}

		imageURL, err := s.extractImageFromURL(ctx, productURL)
		if err != nil {
			s.logger.Warnf("[Wildberries] Failed to extract image from %s: %v", productURL, err)
			continue
		}

		if imageURL != "" && s.isValidProductImage(imageURL) {
			s.logger.Infof("[Wildberries] Found image: %s", imageURL)
			return &ImageResult{
				URL:    imageURL,
				Source: "wildberries",
				Width:  800,
				Height: 600,
			}, nil
		}
	}

	s.logger.Infof("[Wildberries] No image found for %s %s", brand, model)
	return nil, nil
}

func (s *WildberriesImageScraper) getExistingPurchaseLinks(ctx context.Context, brand, model string) []string {
	if s.db == nil {
		return nil
	}

	type PurchaseLink struct {
		URL string
	}
	var links []PurchaseLink

	query := `SELECT url FROM purchase_links WHERE platform = 'wildberries' AND guitar_id IN (
		SELECT id FROM guitars WHERE brand_id IN (
			SELECT id FROM brands WHERE LOWER(name) = LOWER(?)
		) AND LOWER(model) LIKE LOWER(?)
	)`

	modelPattern := "%" + strings.ToLower(model) + "%"
	if err := s.db.Raw(query, brand, modelPattern).Scan(&links).Error; err != nil {
		s.logger.Warnf("[Wildberries] Failed to get existing links: %v", err)
		return nil
	}

	urls := make([]string, 0, len(links))
	for _, link := range links {
		urls = append(urls, link.URL)
	}
	return urls
}

func (s *WildberriesImageScraper) buildSearchURLs(brand, model string) []string {
	encoded := url.QueryEscape(fmt.Sprintf("%s %s гитара", brand, model))
	return []string{
		fmt.Sprintf("https://www.wildberries.ru/catalog/0/search.aspx?search=%s", encoded),
		fmt.Sprintf("https://www.wildberries.ru/catalog/0/search.aspx?search=%s+%s", url.QueryEscape(brand), url.QueryEscape(model)),
	}
}

func (s *WildberriesImageScraper) findProductURL(ctx context.Context, searchURL string) (string, error) {
	instance, err := s.launcher.Launch(ctx)
	if err != nil {
		return "", err
	}
	defer instance.Close()

	page := instance.Page

	s.logger.Infof("[Wildberries] Navigating to search: %s", searchURL)
	if err := page.Timeout(30 * time.Second).Navigate(searchURL); err != nil {
		return "", fmt.Errorf("navigation failed: %w", err)
	}

	time.Sleep(3 * time.Second)

	productURL := s.extractProductURL(page)
	return productURL, nil
}

func (s *WildberriesImageScraper) extractProductURL(page *rod.Page) string {
	jsCode := `
		function() {
			var cards = document.querySelectorAll('article.product-card a.j-card-link, article.product-card a[href*="/detail.aspx"]');
			for (var i = 0; i < Math.min(cards.length, 5); i++) {
				var href = cards[i].href;
				if (href && href.includes('/detail.aspx')) {
					return href;
				}
			}
			
			var links = document.querySelectorAll('a[href*="/catalog/"][href*="/detail.aspx"]');
			for (var i = 0; i < Math.min(links.length, 5); i++) {
				var href = links[i].href;
				if (href) {
					return href;
				}
			}
			
			return '';
		}
	`

	result := page.MustEval(jsCode)
	return result.Str()
}

func (s *WildberriesImageScraper) extractImageFromURL(ctx context.Context, productURL string) (string, error) {
	instance, err := s.launcher.Launch(ctx)
	if err != nil {
		return "", err
	}
	defer instance.Close()

	page := instance.Page

	s.logger.Infof("[Wildberries] Establishing session...")
	if err := page.Timeout(30 * time.Second).Navigate("https://www.wildberries.ru"); err != nil {
		s.logger.Warnf("[Wildberries] Failed to visit main page: %v", err)
	} else {
		time.Sleep(3 * time.Second)
	}

	s.logger.Infof("[Wildberries] Extracting image from: %s", productURL)
	if err := page.Timeout(30 * time.Second).Navigate(productURL); err != nil {
		return "", fmt.Errorf("navigation failed: %w", err)
	}

	time.Sleep(3 * time.Second)

	if s.isBlockedPage(page) {
		s.logger.Warnf("[Wildberries] Page is blocked by Wildberries (suspicious activity). Consider using proxies.")
		return "", nil
	}

	pageInfo := s.getPageDebugInfo(page)
	s.logger.Debugf("[Wildberries] Page info: %s", pageInfo)

	imageURL := s.extractMainImage(page)
	if imageURL != "" && s.isValidProductImage(imageURL) {
		s.logger.Infof("[Wildberries] Found image via JS: %s", imageURL)
		return imageURL, nil
	}

	imageURL = s.extractAllImages(page)
	if imageURL != "" && s.isValidProductImage(imageURL) {
		s.logger.Infof("[Wildberries] Found image via all images: %s", imageURL)
		return imageURL, nil
	}

	imageURL = s.fallbackExtract(page)
	if imageURL != "" && s.isValidProductImage(imageURL) {
		s.logger.Infof("[Wildberries] Found image via HTML fallback: %s", imageURL)
		return imageURL, nil
	}

	imageURL = s.extractFromHTTP(productURL)
	if imageURL != "" {
		s.logger.Infof("[Wildberries] Found image via HTTP: %s", imageURL)
		return imageURL, nil
	}

	s.logger.Warnf("[Wildberries] No image found on page: %s", productURL)
	return "", nil
}

func (s *WildberriesImageScraper) isBlockedPage(page *rod.Page) bool {
	jsCode := `
		function() {
			var bodyText = document.body ? document.body.innerText || '' : '';
			return bodyText.includes('Подозрительная активность') || 
				   bodyText.includes('suspicious activity') ||
				   bodyText.includes('almost ready') ||
				   document.title.includes('almost ready');
		}
	`
	result := page.MustEval(jsCode)
	return result.Bool()
}

func (s *WildberriesImageScraper) getPageDebugInfo(page *rod.Page) string {
	jsCode := `
		function() {
			var info = {
				title: document.title.substring(0, 50),
				imgCount: document.querySelectorAll('img').length,
				hasErrors: false,
				allSrcs: []
			};
			
			var imgs = document.querySelectorAll('img');
			for (var i = 0; i < Math.min(imgs.length, 10); i++) {
				var img = imgs[i];
				var src = img.src || '';
				var dataSrc = img.getAttribute('data-src') || img.getAttribute('data-image') || img.getAttribute('data-original') || '';
				if (src || dataSrc) {
					info.allSrcs.push((src || dataSrc).substring(0, 100));
				}
			}
			
			var scripts = document.querySelectorAll('script');
			for (var i = 0; i < scripts.length; i++) {
				var text = scripts[i].textContent || '';
				if (text.includes('"image"') && text.length < 3000) {
					info.sampleScript = text.substring(0, 300);
					break;
				}
			}
			
			var metaOg = document.querySelector('meta[property="og:image"]');
			if (metaOg) {
				info.ogImage = (metaOg.getAttribute('content') || '').substring(0, 100);
			}
			
			var bodyText = (document.body ? document.body.innerText || '' : '').substring(0, 100);
			info.bodySnippet = bodyText;
			
			return JSON.stringify(info);
		}
	`

	result := page.MustEval(jsCode)
	return result.Str()
}

func (s *WildberriesImageScraper) extractMainImage(page *rod.Page) string {
	jsCode := `
		function() {
			var result = {
				ogImage: '',
				jsonImage: '',
				productImage: '',
				bestImage: '',
				debug: []
			};
			
			var ogImage = document.querySelector('meta[property="og:image"]');
			if (ogImage && ogImage.content) {
				result.ogImage = ogImage.content;
				result.debug.push('og: ' + result.ogImage.substring(0, 80));
			}
			
			var scripts = document.querySelectorAll('script[type="application/ld+json"]');
			for (var i = 0; i < scripts.length; i++) {
				try {
					var data = JSON.parse(scripts[i].textContent);
					if (data && data.image) {
						var img = Array.isArray(data.image) ? data.image[0] : data.image;
						if (img && typeof img === 'string') {
							result.jsonImage = img;
							result.debug.push('ld+json: ' + img.substring(0, 80));
						}
					}
				} catch(e) {}
			}
			
			var dataAttrs = ['data-src', 'data-image', 'data-original', 'data-wbkey', 'data-img'];
			var selectors = [
				'.product-gallery img',
				'.swiper-slide img',
				'[itemprop="image"]',
				'.j-gallery img',
				'.gallery img',
				'.photo-zoom img',
				'.detail-gallery img'
			];
			
			for (var s = 0; s < selectors.length; s++) {
				var img = document.querySelector(selectors[s]);
				if (img) {
					for (var a = 0; a < dataAttrs.length; a++) {
						var src = img.getAttribute(dataAttrs[a]);
						if (src) {
							result.productImage = src;
							result.debug.push('selector: ' + src.substring(0, 80));
							break;
						}
					}
					if (!result.productImage && img.src) {
						result.productImage = img.src;
						result.debug.push('src: ' + img.src.substring(0, 80));
					}
					if (result.productImage) break;
				}
			}
			
			var allImages = document.querySelectorAll('img');
			var bestSize = 0;
			for (var i = 0; i < Math.min(allImages.length, 30); i++) {
				var img = allImages[i];
				for (var a = 0; a < dataAttrs.length; a++) {
					var src = img.getAttribute(dataAttrs[a]);
					if (!src) src = img.src;
					if (src && src.length > 10) {
						var sizeMatch = src.match(/@(\d+)x/);
						var size = sizeMatch ? parseInt(sizeMatch[1]) : 1;
						if (src.includes('@') && size > bestSize) {
							bestSize = size;
							result.bestImage = src;
						}
					}
				}
			}
			
			if (result.bestImage) return result.bestImage;
			if (result.jsonImage) return result.jsonImage;
			if (result.ogImage) return result.ogImage;
			if (result.productImage) return result.productImage;
			
			return JSON.stringify({error: 'no images found', debug: result.debug});
		}
	`

	result := page.MustEval(jsCode)
	returnStr := result.Str()

	if strings.HasPrefix(returnStr, "{") && strings.Contains(returnStr, "error") {
		s.logger.Debugf("[Wildberries] extractMainImage: %s", returnStr)
		return ""
	}

	return returnStr
}

func (s *WildberriesImageScraper) extractAllImages(page *rod.Page) string {
	jsCode := `
		function() {
			var result = {
				jsonImages: [],
				imageUrls: [],
				allSrcs: [],
				totalFound: 0
			};
			
			var scripts = document.querySelectorAll('script');
			for (var i = 0; i < scripts.length; i++) {
				var text = scripts[i].textContent || '';
				var imgMatches = text.match(/"image"\s*:\s*\[?"(https?:\/\/[^"]+(?:jpg|jpeg|png|webp)[^"]*)"/gi);
				if (imgMatches) {
					for (var m = 0; m < imgMatches.length; m++) {
						var match = imgMatches[m].match(/"(https?:\/\/[^"]+)/);
						if (match && match[1]) {
							result.jsonImages.push(match[1]);
						}
					}
				}
				
				var wbImgMatches = text.match(/https?:\/\/[a-z0-9-]+\.(?:wbimages|wb\.ru|wbshop|wbjson)[^\s"'<>]+\.(?:jpg|jpeg|png|webp)(?:\?[^"'<>\s]*)?/gi);
				if (wbImgMatches) {
					for (var j = 0; j < wbImgMatches.length; j++) {
						if (result.jsonImages.indexOf(wbImgMatches[j]) === -1) {
							result.jsonImages.push(wbImgMatches[j]);
						}
					}
				}
			}
			
			var allImages = document.querySelectorAll('img');
			for (var i = 0; i < allImages.length; i++) {
				var img = allImages[i];
				var attrs = ['src', 'data-src', 'data-image', 'data-original', 'data-wbkey'];
				for (var a = 0; a < attrs.length; a++) {
					var src = img.getAttribute(attrs[a]);
					if (src && src.length > 10) {
						result.allSrcs.push(src.substring(0, 100));
						if (src.includes('wbimages') || src.includes('wb.ru') || src.includes('wbshop') || src.includes('wbjson')) {
							result.imageUrls.push(src);
						}
					}
				}
			}
			
			result.totalFound = result.jsonImages.length + result.imageUrls.length;
			
			if (result.jsonImages.length > 0) {
				result.best = result.jsonImages[0];
			} else if (result.imageUrls.length > 0) {
				var bestSize = 0;
				for (var i = 0; i < result.imageUrls.length; i++) {
					var sizeMatch = result.imageUrls[i].match(/@(\d+)x/);
					var size = sizeMatch ? parseInt(sizeMatch[1]) : 0;
					if (size > bestSize) {
						bestSize = size;
						result.best = result.imageUrls[i];
					}
				}
			}
			
			return JSON.stringify(result);
		}
	`

	result := page.MustEval(jsCode)
	resultStr := result.Str()

	var info struct {
		JSONImages []string `json:"jsonImages"`
		ImageURLs  []string `json:"imageUrls"`
		AllSrcs    []string `json:"allSrcs"`
		TotalFound int      `json:"totalFound"`
		Best       string   `json:"best"`
	}

	if err := json.Unmarshal([]byte(resultStr), &info); err != nil {
		s.logger.Debugf("[Wildberries] extractAllImages parse error: %v, raw: %s", err, resultStr[:min(len(resultStr), 200)])
		return ""
	}

	s.logger.Debugf("[Wildberries] Total found: %d (JSON: %d, URLs: %d)", info.TotalFound, len(info.JSONImages), len(info.ImageURLs))
	if len(info.AllSrcs) > 0 {
		s.logger.Debugf("[Wildberries] Sample srcs: %v", info.AllSrcs[:min(3, len(info.AllSrcs))])
	}
	if len(info.JSONImages) > 0 {
		s.logger.Debugf("[Wildberries] JSON images: %v", info.JSONImages[:min(2, len(info.JSONImages))])
	}

	return info.Best
}

func (s *WildberriesImageScraper) fallbackExtract(page *rod.Page) string {
	html, _ := page.HTML()

	patterns := []string{
		`property=["']og:image["']\s+content=["']([^"']+)["']`,
		`content=["']([^"']*wbimages[^"']+)["']`,
		`https://[a-z0-9-]+\.wbimages\.com[^"'\s]+@[^"'\s]+\.(?:jpg|jpeg|png|webp)`,
		`https://[a-z0-9-]+\.wb\.ru[^"'\s]+@[^"'\s]+\.(?:jpg|jpeg|png|webp)`,
		`https://[a-z0-9-]+\.wbshop[^"'\s]+@[^"'\s]+\.(?:jpg|jpeg|png|webp)`,
		`https://[a-z0-9-]+\.wbjson[^"'\s]+@[^"'\s]+\.(?:jpg|jpeg|png|webp)`,
		`src=["'](https://[^"']+(?:wbimages|wb\.ru|wbshop|wbjson)[^"']*\.(?:jpg|jpeg|png|webp)[^"']*)["']`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			for _, m := range match {
				if s.isValidProductImage(m) {
					return m
				}
			}
		}
	}

	return ""
}

func (s *WildberriesImageScraper) extractFromHTTP(productURL string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", productURL, nil)
	if err != nil {
		return ""
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en;q=0.8")

	client := &http.Client{Timeout: 15 * time.Second}
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

	html := string(body)

	patterns := []string{
		`property=["']og:image["']\s+content=["']([^"']+)["']`,
		`content=["']([^"']*wbimages[^"']+)["']`,
		`https://[a-z0-9-]+\.wbimages\.com[^"'\s]+@[^"'\s]+\.(?:jpg|jpeg|png|webp)`,
		`https://[a-z0-9-]+\.wb\.ru[^"'\s]+@[^"'\s]+\.(?:jpg|jpeg|png|webp)`,
		`https://[a-z0-9-]+\.wbshop[^"'\s]+@[^"'\s]+\.(?:jpg|jpeg|png|webp)`,
		`https://[a-z0-9-]+\.wbjson[^"'\s]+@[^"'\s]+\.(?:jpg|jpeg|png|webp)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindAllStringSubmatch(html, -1)
		for _, match := range matches {
			for _, m := range match {
				if s.isValidProductImage(m) {
					s.logger.Debugf("[Wildberries] HTTP fallback found: %s", m[:min(len(m), 80)])
					return m
				}
			}
		}
	}

	return ""
}

func (s *WildberriesImageScraper) isValidProductImage(imageURL string) bool {
	if imageURL == "" {
		return false
	}

	if !strings.HasPrefix(imageURL, "http") {
		return false
	}

	lowerURL := strings.ToLower(imageURL)

	badPatterns := []string{
		"data:image", "1x1", "pixel", "placeholder", "logo", "banner", "icon",
	}

	for _, bad := range badPatterns {
		if strings.Contains(lowerURL, bad) {
			return false
		}
	}

	wildberriesDomains := []string{
		"wbimages", "wb.ru", "wbshop", "wbjson", "wildberries", "cdn1", "cdn2", "cdn3",
		"static", "image", "photo", "img", "goods",
	}

	for _, domain := range wildberriesDomains {
		if strings.Contains(lowerURL, domain) {
			if strings.Contains(lowerURL, ".jpg") || strings.Contains(lowerURL, ".jpeg") ||
				strings.Contains(lowerURL, ".png") || strings.Contains(lowerURL, ".webp") ||
				strings.Contains(lowerURL, ".gif") {
				return true
			}
		}
	}

	return false
}

func (s *WildberriesImageScraper) GetProductURLsByGuitarID(guitarID uuid.UUID) ([]string, error) {
	if s.db == nil {
		return nil, fmt.Errorf("database not connected")
	}

	type PurchaseLink struct {
		URL string
	}
	var links []PurchaseLink

	err := s.db.Table("purchase_links").
		Select("url").
		Where("guitar_id = ? AND platform = ?", guitarID, "wildberries").
		Scan(&links).Error

	if err != nil {
		return nil, err
	}

	urls := make([]string, len(links))
	for i, link := range links {
		urls[i] = link.URL
	}
	return urls, nil
}
