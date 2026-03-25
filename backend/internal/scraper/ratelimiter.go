package scraper

import (
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type RateLimiter struct {
	mu          sync.Mutex
	lastRequest map[string]time.Time
	delay       time.Duration
	logger      *logrus.Logger
}

func NewRateLimiter(delay time.Duration) *RateLimiter {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	return &RateLimiter{
		lastRequest: make(map[string]time.Time),
		delay:       delay,
		logger:      logger,
	}
}

func (r *RateLimiter) Wait(domain string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	if last, exists := r.lastRequest[domain]; exists {
		elapsed := now.Sub(last)
		if elapsed < r.delay {
			waitTime := r.delay - elapsed
			r.logger.Infof("[RATELIMIT] %s: waiting %v (elapsed: %v)", domain, waitTime, elapsed)
			time.Sleep(waitTime)
			now = time.Now()
		}
	}

	r.lastRequest[domain] = now
}

func (r *RateLimiter) Reset(domain string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.lastRequest, domain)
}

func (r *RateLimiter) ResetAll() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastRequest = make(map[string]time.Time)
}
