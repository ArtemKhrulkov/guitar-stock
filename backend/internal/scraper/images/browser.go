package images

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/stealth"
	"github.com/sirupsen/logrus"
)

type BrowserLauncher struct {
	logger      *logrus.Logger
	browserPath string
	proxyURL    string
}

type BrowserInstance struct {
	Browser *rod.Browser
	Page    *rod.Page
}

func NewBrowserLauncher() *BrowserLauncher {
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logger.SetLevel(logrus.InfoLevel)

	browserPath := detectChromePath()
	if browserPath == "" {
		logger.Warn("Chrome not found, browser scrapers will use HTTP fallback only")
	}

	proxyURL := os.Getenv("PROXY_URLS")

	return &BrowserLauncher{
		logger:      logger,
		browserPath: browserPath,
		proxyURL:    proxyURL,
	}
}

func detectChromePath() string {
	if envPath := os.Getenv("CHROME_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath
		}
	}

	switch runtime.GOOS {
	case "darwin":
		paths := []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
		}
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}

	case "linux":
		paths := []string{
			"/usr/bin/chromium",
			"/usr/bin/chromium-browser",
			"/usr/bin/google-chrome",
			"/usr/bin/google-chrome-stable",
			"/snap/bin/chromium",
		}
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}

	case "windows":
		paths := []string{
			`C:\Program Files\Google\Chrome\Application\chrome.exe`,
			`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		}
		for _, path := range paths {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	return ""
}

func (b *BrowserLauncher) HasBrowser() bool {
	return b.browserPath != ""
}

func (b *BrowserLauncher) Launch(ctx context.Context) (*BrowserInstance, error) {
	if b.browserPath == "" {
		return nil, &BrowserNotFoundError{Message: "Chrome not found on system"}
	}

	logger := b.logger
	logger.Infof("Launching browser: %s", b.browserPath)

	l := launcher.New().
		Bin(b.browserPath).
		NoSandbox(true).
		Set("disable-gpu").
		Set("disable-dev-shm-usage").
		Set("disable-setuid-sandbox").
		Set("no-first-run").
		Set("no-zygote").
		Set("disable-blink-features", "AutomationControlled").
		Set("exclude-switches", "enable-automation").
		Set("disable-infobars").
		Set("headless")

	l = l.Set("user-agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	if b.proxyURL != "" {
		proxyParts := strings.Split(b.proxyURL, ",")
		if len(proxyParts) > 0 {
			l = l.Proxy(proxyParts[0])
		}
	}

	urlStr, err := l.Launch()
	if err != nil {
		logger.Errorf("Failed to launch browser: %v", err)
		return nil, err
	}

	browser := rod.New().ControlURL(urlStr)
	if err := browser.Connect(); err != nil {
		logger.Errorf("Failed to connect to browser: %v", err)
		return nil, err
	}

	browser.MustIgnoreCertErrors(true)
	page := stealth.MustPage(browser)

	return &BrowserInstance{
		Browser: browser,
		Page:    page,
	}, nil
}

func (b *BrowserLauncher) LaunchWithTimeout(ctx context.Context, timeout time.Duration) (*BrowserInstance, error) {
	instance, err := b.Launch(ctx)
	if err != nil {
		return nil, err
	}

	instance.Browser = instance.Browser.Timeout(timeout)

	return instance, nil
}

func (i *BrowserInstance) Close() {
	if i.Browser != nil {
		i.Browser.Close()
	}
}

type BrowserNotFoundError struct {
	Message string
}

func (e *BrowserNotFoundError) Error() string {
	return e.Message
}

func IsBrowserNotFound(err error) bool {
	_, ok := err.(*BrowserNotFoundError)
	return ok
}

func EnsureChromeInstalled() error {
	path := detectChromePath()
	if path == "" {
		var installCmd string
		switch runtime.GOOS {
		case "darwin":
			installCmd = "brew install --cask google-chrome"
		case "linux":
			installCmd = "apt install chromium or apt install chromium-browser"
		case "windows":
			installCmd = "Download Chrome from https://www.google.com/chrome/"
		}
		return &ChromeNotInstalledError{InstallCommand: installCmd}
	}
	return nil
}

type ChromeNotInstalledError struct {
	InstallCommand string
}

func (e *ChromeNotInstalledError) Error() string {
	return "Chrome is not installed. Install with: " + e.InstallCommand
}

func WhichBrowser() string {
	return detectChromePath()
}

func CheckBrowser() error {
	path := detectChromePath()
	if path == "" {
		return EnsureChromeInstalled()
	}

	cmd := exec.Command(path, "--version")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	logrus.Infof("Found Chrome: %s", strings.TrimSpace(string(output)))
	return nil
}
