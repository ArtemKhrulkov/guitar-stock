package scraper

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
)

type ProxyMetrics struct {
	TotalRequests int64
	SuccessCount  int64
	FailureCount  int64
	LastUsed      time.Time
	LastError     string
	IsAvailable   bool
}

type ProxyPool struct {
	proxies   []string
	available map[string]bool
	metrics   map[string]*ProxyMetrics
	index     int
	mu        sync.RWMutex
	logger    *logrus.Logger
}

func NewProxyPool(proxyURLs []string) *ProxyPool {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	available := make(map[string]bool)
	metrics := make(map[string]*ProxyMetrics)

	for _, proxy := range proxyURLs {
		available[proxy] = true
		metrics[proxy] = &ProxyMetrics{
			TotalRequests: 0,
			SuccessCount:  0,
			FailureCount:  0,
			IsAvailable:   true,
		}
	}

	logger.Infof("[PROXY] Initialized pool with %d proxies", len(proxyURLs))

	return &ProxyPool{
		proxies:   proxyURLs,
		available: available,
		metrics:   metrics,
		index:     0,
		logger:    logger,
	}
}

func (p *ProxyPool) Get() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.proxies) == 0 {
		return ""
	}

	for i := 0; i < len(p.proxies); i++ {
		idx := p.index % len(p.proxies)
		p.index++

		proxy := p.proxies[idx]
		if p.available[proxy] {
			p.metrics[proxy].LastUsed = time.Now()
			p.metrics[proxy].TotalRequests++
			p.logger.Debugf("[PROXY] Selected: %s", maskProxy(proxy))
			return proxy
		}
	}

	p.logger.Warn("[PROXY] No available proxies")
	return ""
}

func (p *ProxyPool) RecordSuccess(proxy string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if m, exists := p.metrics[proxy]; exists {
		atomic.AddInt64(&m.SuccessCount, 1)
		p.logger.Debugf("[PROXY] Success: %s (total: %d)", maskProxy(proxy), m.SuccessCount)
	}
}

func (p *ProxyPool) RecordFailure(proxy, errorMsg string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if m, exists := p.metrics[proxy]; exists {
		atomic.AddInt64(&m.FailureCount, 1)
		m.LastError = errorMsg

		failureRate := m.FailureCount
		if m.TotalRequests > 0 {
			failureRate = m.FailureCount * 100 / m.TotalRequests
		}

		if failureRate > 50 && m.TotalRequests > 3 {
			p.available[proxy] = false
			p.logger.Warnf("[PROXY] Marked unavailable (failure rate: %d%%): %s", failureRate, maskProxy(proxy))
		} else {
			p.logger.Warnf("[PROXY] Failure recorded: %s - %s", maskProxy(proxy), errorMsg)
		}
	}
}

func (p *ProxyPool) MarkUnavailable(proxy string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.available[proxy] {
		p.available[proxy] = false
		p.logger.Warnf("[PROXY] Marked unavailable: %s", maskProxy(proxy))
	}
}

func (p *ProxyPool) Has() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, available := range p.available {
		if available {
			return true
		}
	}
	return false
}

func (p *ProxyPool) AvailableCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	count := 0
	for _, available := range p.available {
		if available {
			count++
		}
	}
	return count
}

func (p *ProxyPool) GetStats() map[string]ProxyMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := make(map[string]ProxyMetrics)
	for proxy, m := range p.metrics {
		stats[maskProxy(proxy)] = ProxyMetrics{
			TotalRequests: m.TotalRequests,
			SuccessCount:  m.SuccessCount,
			FailureCount:  m.FailureCount,
			LastUsed:      m.LastUsed,
			LastError:     m.LastError,
			IsAvailable:   p.available[proxy],
		}
	}
	return stats
}

func (p *ProxyPool) LogStats() {
	stats := p.GetStats()
	p.logger.Info("[PROXY] === Proxy Statistics ===")
	for proxy, m := range stats {
		status := "available"
		if !m.IsAvailable {
			status = "unavailable"
		}
		p.logger.Infof("[PROXY] %s: %s | total: %d | success: %d | failure: %d | last_used: %s",
			proxy, status, m.TotalRequests, m.SuccessCount, m.FailureCount, m.LastUsed.Format("15:04:05"))
	}
	p.logger.Info("[PROXY] =============================")
}

func maskProxy(proxy string) string {
	if proxy == "" {
		return "(direct)"
	}

	if strings.Contains(proxy, "@") {
		parts := strings.Split(proxy, "@")
		if len(parts) == 2 {
			authParts := strings.Split(parts[0], ":")
			if len(authParts) == 2 {
				return fmt.Sprintf("%s:***@%s", authParts[0], parts[1])
			}
			return "***:@" + parts[1]
		}
	}
	return proxy
}
