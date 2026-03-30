package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"guitar-stock/internal/config"
	"guitar-stock/internal/database"
	"guitar-stock/internal/repository"
	imgscrape "guitar-stock/internal/scraper/images"
)

var (
	flagAll         = flag.Bool("all", false, "Scrape all guitars without images")
	flagGuitarID    = flag.String("guitar-id", "", "Scrape specific guitar by ID")
	flagBatchSize   = flag.Int("batch-size", 3, "Number of guitars to process in each batch")
	flagConcurrency = flag.Int("concurrency", 2, "Number of concurrent scrapers")
	flagCheck       = flag.Bool("check", false, "Check if Chrome is installed and exit")
	flagHelp        = flag.Bool("help", false, "Show help")
	flagVerbose     = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()

	if *flagHelp {
		printHelp()
		os.Exit(0)
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})
	logrus.SetOutput(os.Stdout)

	if *flagVerbose {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if *flagCheck {
		if err := imgscrape.CheckBrowser(); err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		logrus.Info("Chrome is installed and ready")
		os.Exit(0)
	}

	if !*flagAll && *flagGuitarID == "" {
		logrus.Error("Please specify --all or --guitar-id")
		printHelp()
		os.Exit(1)
	}

	logrus.Info("=== Guitar Image Scraper CLI ===")

	if err := imgscrape.CheckBrowser(); err != nil {
		logrus.Warn(err)
		logrus.Warn("HTTP-only scrapers will still work, but browser scrapers will be skipped")
	}

	cfg, err := config.Load()
	if err != nil {
		logrus.Fatalf("Failed to load config: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		logrus.Fatalf("Failed to connect to database: %v", err)
	}

	guitarRepo := repository.NewGuitarRepository(db)
	scraper := imgscrape.NewScraper(db)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logrus.Info("Received shutdown signal, stopping...")
		cancel()
	}()

	var guitarIDs []uuid.UUID

	if *flagAll {
		ids, err := guitarRepo.FindIDsWithoutImages()
		if err != nil {
			logrus.Fatalf("Failed to find guitars without images: %v", err)
		}
		if len(ids) == 0 {
			logrus.Info("All guitars already have images!")
			os.Exit(0)
		}
		guitarIDs = ids
		logrus.Infof("Found %d guitars without images", len(ids))
	} else if *flagGuitarID != "" {
		id, err := uuid.Parse(*flagGuitarID)
		if err != nil {
			logrus.Fatalf("Invalid guitar ID: %v", err)
		}
		guitarIDs = []uuid.UUID{id}
	}

	startTime := time.Now()
	result := scrapeGuitars(ctx, scraper, guitarRepo, guitarIDs, *flagBatchSize, *flagConcurrency)

	elapsed := time.Since(startTime)

	logrus.Info("=== Results ===")
	logrus.Infof("Total:     %d", result.Total)
	logrus.Infof("Success:   %d", result.Success)
	logrus.Infof("Failed:    %d", result.Failed)
	logrus.Infof("Skipped:   %d (browser not available)", result.Skipped)
	logrus.Infof("Time:      %v", elapsed.Round(time.Second))

	if result.Failed > 0 {
		os.Exit(1)
	}
}

func scrapeGuitars(ctx context.Context, scraper *imgscrape.Scraper, guitarRepo *repository.GuitarRepository, guitarIDs []uuid.UUID, batchSize, concurrency int) *ScrapeResult {
	result := &ScrapeResult{}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, concurrency)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for i := 0; i < len(guitarIDs); i += batchSize {
		select {
		case <-ctx.Done():
			logrus.Info("Context cancelled, stopping...")
			goto waitForCompletion
		default:
		}

		end := i + batchSize
		if end > len(guitarIDs) {
			end = len(guitarIDs)
		}
		batch := guitarIDs[i:end]

		logrus.Infof("Processing batch %d-%d of %d...", i+1, end, len(guitarIDs))

		for _, guitarID := range batch {
			semaphore <- struct{}{}
			wg.Add(1)

			go func(id uuid.UUID) {
				defer wg.Done()
				defer func() { <-semaphore }()

				select {
				case <-ctx.Done():
					atomic.AddInt64(&result.Skipped, 1)
					return
				default:
				}

				r := scrapeSingleGuitar(ctx, scraper, guitarRepo, id)
				atomic.AddInt64(&result.Total, 1)

				if r.Success {
					atomic.AddInt64(&result.Success, 1)
				} else if r.Skipped {
					atomic.AddInt64(&result.Skipped, 1)
				} else {
					atomic.AddInt64(&result.Failed, 1)
				}
			}(guitarID)
		}
	}

waitForCompletion:
	wg.Wait()

	return result
}

func scrapeSingleGuitar(ctx context.Context, scraper *imgscrape.Scraper, guitarRepo *repository.GuitarRepository, guitarID uuid.UUID) *SingleResult {
	brand, err := guitarRepo.GetBrand(guitarID)
	if err != nil {
		logrus.Errorf("Failed to get brand for %s: %v", guitarID, err)
		return &SingleResult{Success: false}
	}

	guitar, err := guitarRepo.FindByID(guitarID)
	if err != nil {
		logrus.Errorf("Failed to get guitar %s: %v", guitarID, err)
		return &SingleResult{Success: false}
	}

	logrus.Infof("Scraping: %s %s", brand.Name, guitar.Model)

	scrapeCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	imageResult, err := scraper.Scrape(scrapeCtx, brand.Name, guitar.Model)
	if err != nil {
		if imgscrape.IsBrowserNotFound(err) {
			logrus.Warnf("Browser not available, skipping %s %s", brand.Name, guitar.Model)
			return &SingleResult{Skipped: true}
		}
		logrus.Errorf("Failed to scrape %s %s: %v", brand.Name, guitar.Model, err)
		return &SingleResult{Success: false}
	}

	if imageResult == nil {
		logrus.Infof("No image found for %s %s", brand.Name, guitar.Model)
		return &SingleResult{Success: false}
	}

	now := time.Now()
	if err := guitarRepo.UpdateImage(guitarID, imageResult.URL, imageResult.Source, &now); err != nil {
		logrus.Errorf("Failed to update image for %s %s: %v", brand.Name, guitar.Model, err)
		return &SingleResult{Success: false}
	}

	logrus.Infof("✓ Found image for %s %s: %s (%s)", brand.Name, guitar.Model, imageResult.Source, truncateURL(imageResult.URL, 60))
	return &SingleResult{Success: true}
}

type ScrapeResult struct {
	Total   int64
	Success int64
	Failed  int64
	Skipped int64
}

type SingleResult struct {
	Success bool
	Skipped bool
}

func printHelp() {
	fmt.Println(`
Guitar Image Scraper CLI

Usage:
  go run ./backend/cmd/scraper/main.go [flags]

Flags:
  --all              Scrape all guitars without images
  --guitar-id <uuid> Scrape a specific guitar by ID
  --batch-size <n>   Batch size (default: 3)
  --concurrency <n>  Concurrent scrapers (default: 2)
  --check            Check if Chrome is installed
  --v                Verbose output
  --help             Show this help

Environment Variables:
  DATABASE_URL  PostgreSQL connection string
                 Default: postgres://postgres:postgres@localhost:5432/guitar_stock
  CHROME_PATH   Path to Chrome binary (auto-detected if not set)
  PROXY_URLS    Proxy URLs (optional)

Examples:
  # Check if Chrome is installed
  go run ./backend/cmd/scraper/main.go --check

  # Scrape all guitars without images
  go run ./backend/cmd/scraper/main.go --all

  # Scrape specific guitar
  go run ./backend/cmd/scraper/main.go --guitar-id 550e8400-e29b-41d4-a716-446655440000

  # Use custom database
  DATABASE_URL="postgres://user:pass@host:5432/db" go run ./backend/cmd/scraper/main.go --all

  # Higher concurrency for faster scraping
  go run ./backend/cmd/scraper/main.go --all --concurrency 4 --batch-size 5
`)
}

func truncateURL(url string, maxLen int) string {
	if len(url) <= maxLen {
		return url
	}
	return url[:maxLen-3] + "..."
}
