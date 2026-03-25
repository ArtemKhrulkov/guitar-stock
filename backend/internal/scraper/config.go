package scraper

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type ScraperConfig struct {
	MaxConcurrent int
	RateLimit     time.Duration
	Retries       int
	RetryDelay    time.Duration
	ProxyURLs     []string
	UserAgents    []string
}

func LoadConfig() *ScraperConfig {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	maxConcurrent := getEnvInt("SCRAPER_MAX_CONCURRENT", 3)
	rateLimit := getEnvDuration("SCRAPER_RATE_LIMIT", 120*time.Second)
	retries := getEnvInt("SCRAPER_RETRIES", 3)
	retryDelay := getEnvDuration("SCRAPER_RETRY_DELAY", 10*time.Second)
	proxyURLs := getEnvProxyURLs("PROXY_URLS")

	userAgents := []string{
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.15",
	}

	logger.Infof("[CONFIG] Loaded scraper config: max_concurrent=%d, rate_limit=%v, retries=%d, retry_delay=%v, proxies=%d",
		maxConcurrent, rateLimit, retries, retryDelay, len(proxyURLs))

	return &ScraperConfig{
		MaxConcurrent: maxConcurrent,
		RateLimit:     rateLimit,
		Retries:       retries,
		RetryDelay:    retryDelay,
		ProxyURLs:     proxyURLs,
		UserAgents:    userAgents,
	}
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if intVal, err := strconv.Atoi(val); err == nil {
			return intVal
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			return duration
		}
	}
	return defaultVal
}

func getEnvProxyURLs(key string) []string {
	if val := os.Getenv(key); val != "" {
		urls := strings.Split(val, ",")
		result := make([]string, 0, len(urls))
		for _, url := range urls {
			trimmed := strings.TrimSpace(url)
			if trimmed != "" {
				result = append(result, trimmed)
			}
		}
		return result
	}
	return nil
}
