package scraper

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/sirupsen/logrus"
)

type BrowserContext struct {
	ctx    context.Context
	cancel context.CancelFunc
	inUse  bool
	pool   *BrowserPool
}

func (bc *BrowserContext) Close() {
	if bc.cancel != nil {
		bc.cancel()
	}
}

type BrowserPool struct {
	pool       chan *BrowserContext
	size       int
	inUse      int64
	userAgents []string
	proxies    *ProxyPool
	logger     *logrus.Logger
	mu         sync.Mutex
}

func findChromePath() string {
	paths := []string{
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/usr/local/bin/chromium",
		"/usr/local/bin/chrome",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	cmd := exec.Command("which", "chromium", "chrome", "google-chrome", "chromium-browser")
	if output, err := cmd.Output(); err == nil {
		path := strings.TrimSpace(string(output))
		if path != "" {
			return path
		}
	}

	return ""
}

func NewBrowserPool(size int, userAgents []string, proxies *ProxyPool) *BrowserPool {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})

	chromePath := findChromePath()
	if chromePath != "" {
		logger.Infof("[BROWSER] Found Chrome at: %s", chromePath)
	}

	pool := make(chan *BrowserContext, size)
	for i := 0; i < size; i++ {
		pool <- nil
	}

	logger.Infof("[BROWSER] Initialized pool with size %d", size)

	return &BrowserPool{
		pool:       pool,
		size:       size,
		userAgents: userAgents,
		proxies:    proxies,
		logger:     logger,
	}
}

func (bp *BrowserPool) Acquire() (*BrowserContext, error) {
	bp.mu.Lock()
	bc := <-bp.pool
	bp.mu.Unlock()

	if bc == nil {
		bc = &BrowserContext{pool: bp}
	}

	if bc.ctx != nil && bc.cancel != nil {
		bc.cancel()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)

	uaIndex := time.Now().UnixNano() % int64(len(bp.userAgents))
	userAgent := bp.userAgents[uaIndex]

	allocOpts := []chromedp.ExecAllocatorOption{
		chromedp.UserAgent(userAgent),
		chromedp.NoSandbox,
		chromedp.Headless,
		chromedp.DisableGPU,
	}

	if chromePath := findChromePath(); chromePath != "" {
		allocOpts = append(allocOpts, chromedp.ExecPath(chromePath))
	}

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, allocOpts...)

	browserCtx, browserCancel := chromedp.NewContext(allocCtx)

	bc.ctx = browserCtx
	bc.cancel = func() {
		browserCancel()
		allocCancel()
		cancel()
	}
	bc.inUse = true

	currentUse := atomic.AddInt64(&bp.inUse, 1)
	bp.logger.Debugf("[BROWSER] Acquired (in use: %d/%d, ua: %s...)", currentUse, bp.size, userAgent[:50])

	return bc, nil
}

func (bp *BrowserPool) Release(bc *BrowserContext) {
	if bc == nil {
		return
	}

	if bc.cancel != nil {
		bc.cancel()
	}

	bc.ctx = nil
	bc.cancel = nil
	bc.inUse = false

	currentUse := atomic.AddInt64(&bp.inUse, -1)
	bp.logger.Debugf("[BROWSER] Released (in use: %d/%d)", currentUse, bp.size)

	bp.mu.Lock()
	bp.pool <- bc
	bp.mu.Unlock()
}

func (bp *BrowserPool) Close() {
	bp.logger.Info("[BROWSER] Closing pool...")

	for {
		select {
		case bc := <-bp.pool:
			if bc != nil {
				bc.Close()
			}
			if atomic.LoadInt64(&bp.inUse) == 0 {
				goto done
			}
		default:
			goto done
		}
	}

done:
	close(bp.pool)
	bp.logger.Info("[BROWSER] Pool closed")
}

func (bp *BrowserPool) Stats() (inUse, total int) {
	return int(atomic.LoadInt64(&bp.inUse)), bp.size
}
