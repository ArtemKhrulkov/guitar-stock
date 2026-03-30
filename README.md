# Guitar Stories

A web application for guitar enthusiasts to browse guitar catalogs, explore detailed descriptions, view famous players, and find purchase links from various e-commerce platforms.

![Guitar Stories](https://via.placeholder.com/1200x400/1a1a1a/FFB300?text=Guitar+Stories)

## Features

- **Guitar Catalog** - Browse 63+ guitars across 12 legendary brands
- **Advanced Filters** - Filter by brand, type (electric/acoustic/bass), and price range
- **Detailed Specifications** - Wood types, pickup configurations, hardware details
- **Famous Players** - See which legendary guitarists used each model
- **Purchase Links** - Direct links to Wildberries with current prices
- **Responsive Design** - Works on desktop and mobile
- **Dark Theme** - Easy on the eyes for late-night gear research

## Tech Stack

### Frontend
- **Framework**: Nuxt 3 (Vue 3 SSR)
- **Styling**: TailwindCSS + Vuetify
- **State**: Pinia
- **Image Optimization**: Nuxt Image

### Backend
- **Language**: Go 1.21+
- **Framework**: Gin
- **Database**: PostgreSQL 15
- **ORM**: GORM

### Infrastructure
- **Containerization**: Docker Compose
- **Scraping**: Custom scrapers with browser automation (go-rod)

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Go 1.21+ (for running scraper CLI)
- Chrome/Chromium (for image scraping)

### 1. Clone and Setup

```bash
git clone <repository-url>
cd guitar-stories
cp .env.example .env
```

### 2. Start Services

```bash
make dev
```

This starts:
- Frontend at http://localhost:3000
- Backend at http://localhost:8080
- PostgreSQL at localhost:5432

### 3. Scrape Images (Optional)

The image scraper runs outside Docker for better browser automation performance:

```bash
# Check if Chrome is installed
go run ./backend/cmd/scraper/main.go --check

# Scrape all guitar images
go run ./backend/cmd/scraper/main.go --all

# Or build and run
cd backend && go build -o bin/image-scraper ./cmd/scraper
./bin/image-scraper --all
```

## Project Structure

```
guitar-stories/
├── backend/                 # Go API server
│   ├── cmd/
│   │   └── scraper/        # Image scraper CLI
│   └── internal/
│       ├── handlers/        # HTTP handlers
│       ├── models/          # Database models
│       ├── repository/      # Data access layer
│       └── scraper/         # Purchase link scrapers
├── frontend/                # Nuxt 3 application
│   ├── pages/              # Route pages
│   ├── components/          # Vue components
│   └── composables/        # Reusable logic
├── docker-compose.yml      # Production setup
├── docker-compose.dev.yml  # Development setup
└── Makefile               # Development commands
```

## Database

### Schema
- **brands** - 12 guitar manufacturers (Gibson, Fender, Ibanez, etc.)
- **guitars** - 63 models with specifications in JSONB
- **players** - Famous guitarists and their gear
- **guitar_players** - Many-to-many relationships
- **purchase_links** - E-commerce product links with prices

### Connect to Database

```bash
make db-connect
```

## API Endpoints

### Public
- `GET /api/brands` - List all brands
- `GET /api/brands/:id` - Brand detail with guitars
- `GET /api/guitars` - List guitars (supports filters)
- `GET /api/guitars/:id` - Guitar detail with players & links
- `GET /api/players` - List all players
- `GET /api/players/:id` - Player detail with guitars
- `GET /api/search?q=...` - Full-text search

### Admin (Basic Auth)
- `POST /api/admin/scrape/all` - Scrape purchase links for all guitars
- `POST /api/admin/scrape/:id` - Scrape links for specific guitar
- `GET/POST/DELETE /api/admin/links` - Manage purchase links

## Makefile Commands

```bash
make dev           # Start development environment
make dev-down      # Stop development environment
make logs          # View all logs
make logs-backend  # View backend logs only
make db-connect    # Connect to database via psql
make db-reset      # Reset database (WARNING: deletes all data)
make clean         # Clean up containers and volumes
```

## Image Scraping

Images are scraped from multiple sources in priority order:
1. **Bing Images** - Fastest, most reliable
2. **Google Images** - Browser automation
3. **Sweetwater** - HTTP requests
4. **Manufacturer Sites** - Brand official images
5. **GuitarCenter** - Fallback option
6. **Wildberries** - Russian market (requires proxy)

All 63 guitars currently have images scraped from Bing.

## Environment Variables

### Backend (.env)
```
PORT=8080
DATABASE_URL=postgres://postgres:postgres@localhost:5432/guitar_stock
GIN_MODE=debug
ALLOWED_ORIGINS=http://localhost:3000
SCRAPER_RATE_LIMIT=60
```

### Frontend (.env)
```
NUXT_PUBLIC_API_URL=/api
NUXT_PUBLIC_API_BACKEND_URL=http://localhost:8080
```

## Brands in Database

| Brand | Country | Guitars |
|-------|---------|---------|
| Gibson | USA | 8 |
| Fender | USA | 6 |
| Ibanez | Japan | 6 |
| ESP | Japan | 5 |
| Schecter | USA | 5 |
| Yamaha | Japan | 5 |
| Greco | Japan | 5 |
| Squier | Japan | 5 |
| Gretsch | USA | 5 |
| Sterling | Japan | 5 |
| Burny | Japan | 4 |
| Music Man | USA | 4 |

## Contributing

1. Create a feature branch
2. Make your changes
3. Test locally with `make dev`
4. Commit with clear messages
5. Push and create a PR

## License

MIT
