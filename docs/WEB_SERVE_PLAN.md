# Web Serve Capability - Implementation Plan

**Version:** v3.2.0
**Status:** Planning Phase
**Target:** Add web serving capability for RSS summaries
**Date:** October 28, 2025

---

## 📋 Executive Summary

This document outlines the plan to add web serving capability to **briefly**, transforming it from a CLI-only tool into a web-accessible news aggregator. The implementation will build on the existing simplified architecture (v3.0) and PostgreSQL persistence layer to serve aggregated RSS content via HTTP.

**Key Goals:**
- ✅ Add `briefly serve` command for HTTP server
- ✅ REST API endpoints for articles, digests, and feeds
- ✅ Simple web frontend (HTMX + server-side rendering)
- ✅ Read from database populated by cron job (`briefly aggregate`)
- ✅ Deploy-ready architecture (Docker, cloud-friendly)

---

## 🏗️ Architecture Overview

### Current State (v3.1.0)
```
┌─────────────┐
│   CLI User  │
└──────┬──────┘
       │
       ↓
┌──────────────────────────────────┐
│   briefly CLI                    │
│   - aggregate (fetch RSS)        │
│   - feed (manage sources)        │
│   - digest (generate summaries)  │
└──────────────┬───────────────────┘
               │
               ↓
┌──────────────────────────────────┐
│   PostgreSQL Database            │
│   - articles                     │
│   - summaries                    │
│   - feeds                        │
│   - feed_items                   │
│   - digests                      │
└──────────────────────────────────┘
```

### Target State (v3.2.0)
```
┌─────────────┐     ┌─────────────┐
│   CLI User  │     │  Web Browser│
└──────┬──────┘     └──────┬──────┘
       │                   │
       │                   │ HTTP
       │                   ↓
       │          ┌──────────────────────────┐
       │          │  briefly serve           │
       │          │  - Web UI (HTMX)         │
       │          │  - REST API (JSON)       │
       │          └──────────┬───────────────┘
       │                     │
       ↓                     │
┌──────────────────────────────────────┐
│   briefly CLI                        │
│   - aggregate (cron)                 │
│   - feed (manage)                    │
│   - digest (generate)                │
└──────────────┬───────────────────────┘
               │
               ↓
┌──────────────────────────────────────┐
│   PostgreSQL Database                │
│   - articles (content)               │
│   - summaries (LLM-generated)        │
│   - feeds (RSS sources)              │
│   - feed_items (raw items)           │
│   - digests (rendered output)        │
└──────────────────────────────────────┘
```

---

## 📦 New Components

### 1. HTTP Server Package (`internal/server/`)

**Purpose:** Core HTTP server infrastructure

**Files:**
- `server.go` - HTTP server setup, lifecycle management
- `routes.go` - Route definitions and URL mapping
- `middleware.go` - Logging, CORS, recovery, rate limiting
- `handlers.go` - HTML handlers (server-side rendered)
- `api.go` - REST API handlers (JSON responses)
- `server_test.go` - Unit tests

**Key Features:**
- Graceful shutdown
- Request/response logging
- Error recovery
- CORS support
- Rate limiting
- Static file serving

**Example Structure:**
```go
package server

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "briefly/internal/persistence"
)

type Server struct {
    router     *chi.Mux
    db         persistence.Database
    config     Config
    httpServer *http.Server
}

func New(db persistence.Database, cfg Config) *Server {
    s := &Server{
        router: chi.NewRouter(),
        db:     db,
        config: cfg,
    }
    s.setupRoutes()
    s.setupMiddleware()
    return s
}

func (s *Server) Start() error {
    // Start HTTP server
}

func (s *Server) Shutdown(ctx context.Context) error {
    // Graceful shutdown
}
```

---

### 2. Serve Command (`cmd/handlers/serve.go`)

**Purpose:** CLI command to start HTTP server

**Command Signature:**
```bash
briefly serve [flags]

Flags:
  --port INT          HTTP port (default: 8080)
  --host STRING       Host address (default: "0.0.0.0")
  --static-dir PATH   Static files directory (default: "web/static")
  --template-dir PATH Template directory (default: "web/templates")
  --reload            Auto-reload templates in dev mode
```

**Implementation:**
```go
func NewServeCmd() *cobra.Command {
    var (
        port        int
        host        string
        staticDir   string
        templateDir string
        reload      bool
    )

    cmd := &cobra.Command{
        Use:   "serve",
        Short: "Start HTTP server for web interface",
        Long: `Start the briefly web server to browse aggregated articles.

The server provides:
  • Web UI for browsing articles and digests
  • REST API for programmatic access
  • Real-time article updates

The server reads from the database populated by 'briefly aggregate'.
Run aggregation separately (e.g., via cron) to keep content fresh.

Examples:
  # Start server on default port 8080
  briefly serve

  # Start on custom port with dev mode
  briefly serve --port 3000 --reload`,
        RunE: func(cmd *cobra.Command, args []string) error {
            return runServe(cmd.Context(), port, host, staticDir, templateDir, reload)
        },
    }

    cmd.Flags().IntVar(&port, "port", 8080, "HTTP server port")
    cmd.Flags().StringVar(&host, "host", "0.0.0.0", "HTTP server host")
    cmd.Flags().StringVar(&staticDir, "static-dir", "web/static", "Static files directory")
    cmd.Flags().StringVar(&templateDir, "template-dir", "web/templates", "Template directory")
    cmd.Flags().BoolVar(&reload, "reload", false, "Auto-reload templates (dev mode)")

    return cmd
}
```

---

### 3. HTML Templates (`web/templates/`)

**Purpose:** Server-side rendered views using Go html/template

**Technology Choice: HTMX + Go Templates**
- **Pros:** Simple, fast, no build pipeline, server-side rendering, small footprint
- **Cons:** Less interactive than React/Vue
- **Alternative:** Templ (type-safe Go templates) or full SPA (React)

**Template Structure:**
```
web/
├── templates/
│   ├── layouts/
│   │   └── base.html           # Base layout with header/footer
│   ├── pages/
│   │   ├── home.html            # Homepage with latest articles
│   │   ├── articles.html        # Article list with filters
│   │   ├── article.html         # Single article view
│   │   ├── digests.html         # Digest list
│   │   ├── digest.html          # Single digest view
│   │   └── feeds.html           # Feed management
│   └── components/
│       ├── article-card.html    # Reusable article card
│       ├── pagination.html      # Pagination component
│       └── filter.html          # Filter controls
└── static/
    ├── css/
    │   └── styles.css           # Custom styles
    ├── js/
    │   └── app.js               # Client-side JS
    └── images/
        └── logo.png
```

**Example Base Template:**
```html
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - Briefly</title>

    <!-- Tailwind CSS for styling -->
    <link href="https://cdn.jsdelivr.net/npm/tailwindcss@2/dist/tailwind.min.css" rel="stylesheet">

    <!-- HTMX for dynamic interactions -->
    <script src="https://unpkg.com/htmx.org@1.9.10"></script>

    <link rel="stylesheet" href="/static/css/styles.css">
</head>
<body class="bg-gray-50">
    <nav class="bg-white shadow-sm">
        <div class="container mx-auto px-4 py-3">
            <div class="flex items-center justify-between">
                <h1 class="text-2xl font-bold text-blue-600">Briefly</h1>
                <div class="space-x-4">
                    <a href="/" class="text-gray-700 hover:text-blue-600">Home</a>
                    <a href="/articles" class="text-gray-700 hover:text-blue-600">Articles</a>
                    <a href="/digests" class="text-gray-700 hover:text-blue-600">Digests</a>
                    <a href="/feeds" class="text-gray-700 hover:text-blue-600">Feeds</a>
                </div>
            </div>
        </div>
    </nav>

    <main class="container mx-auto px-4 py-8">
        {{template "content" .}}
    </main>

    <footer class="bg-white border-t mt-12">
        <div class="container mx-auto px-4 py-6 text-center text-gray-600">
            <p>Briefly - AI-Powered News Aggregator</p>
        </div>
    </footer>
</body>
</html>
```

---

## 🔌 REST API Endpoints

### Article Endpoints

#### `GET /api/articles`
List articles with pagination and filtering

**Query Parameters:**
- `page` (int) - Page number (default: 1)
- `limit` (int) - Items per page (default: 20, max: 100)
- `category` (string) - Filter by category
- `since` (string) - ISO 8601 date (e.g., "2025-10-27T00:00:00Z")
- `unprocessed` (bool) - Only unprocessed articles
- `sort` (string) - Sort field (date, title)
- `order` (string) - Sort order (asc, desc)

**Response:**
```json
{
  "data": {
    "articles": [
      {
        "id": "abc123",
        "url": "https://example.com/article",
        "title": "Article Title",
        "summary": "Article summary...",
        "content_type": "html",
        "category": "AI",
        "published_at": "2025-10-27T10:00:00Z",
        "created_at": "2025-10-27T11:00:00Z",
        "processed": false
      }
    ],
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 150,
      "total_pages": 8,
      "has_next": true,
      "has_prev": false
    }
  },
  "meta": {
    "timestamp": "2025-10-28T12:00:00Z",
    "version": "v3.2.0"
  }
}
```

#### `GET /api/articles/:id`
Get single article by ID

**Response:**
```json
{
  "data": {
    "article": {
      "id": "abc123",
      "url": "https://example.com/article",
      "title": "Article Title",
      "summary": "Detailed summary...",
      "cleaned_text": "Full article text...",
      "content_type": "html",
      "category": "AI",
      "published_at": "2025-10-27T10:00:00Z",
      "created_at": "2025-10-27T11:00:00Z",
      "processed": true,
      "embedding": [...],
      "cluster_id": 5
    }
  },
  "meta": {...}
}
```

#### `GET /api/articles/recent`
Get recent articles (last 24 hours)

**Query Parameters:**
- `hours` (int) - Lookback period (default: 24)
- `limit` (int) - Max articles to return

---

### Digest Endpoints

#### `GET /api/digests`
List all digests

**Response:**
```json
{
  "data": {
    "digests": [
      {
        "id": "digest-2025-10-27",
        "date": "2025-10-27",
        "article_count": 12,
        "format": "newsletter",
        "created_at": "2025-10-27T18:00:00Z"
      }
    ],
    "pagination": {...}
  },
  "meta": {...}
}
```

#### `GET /api/digests/:id`
Get single digest with full content

**Response:**
```json
{
  "data": {
    "digest": {
      "id": "digest-2025-10-27",
      "date": "2025-10-27",
      "content": {
        "executive_summary": "...",
        "articles": [...],
        "categories": [...]
      },
      "article_count": 12,
      "format": "newsletter",
      "created_at": "2025-10-27T18:00:00Z"
    }
  },
  "meta": {...}
}
```

#### `GET /api/digests/latest`
Get the most recent digest

---

### Feed Endpoints

#### `GET /api/feeds`
List all feeds

**Query Parameters:**
- `active` (bool) - Filter by active status

**Response:**
```json
{
  "data": {
    "feeds": [
      {
        "id": "feed-123",
        "url": "https://hnrss.org/newest",
        "title": "Hacker News",
        "active": true,
        "last_fetched": "2025-10-28T10:00:00Z",
        "item_count": 150,
        "error_count": 0,
        "created_at": "2025-10-01T00:00:00Z"
      }
    ]
  },
  "meta": {...}
}
```

#### `GET /api/feeds/:id/stats`
Get statistics for a specific feed

**Response:**
```json
{
  "data": {
    "stats": {
      "feed_id": "feed-123",
      "total_items": 150,
      "processed_items": 120,
      "unprocessed_items": 30,
      "average_items_per_day": 5.2,
      "last_fetch_success": true,
      "uptime_percentage": 98.5
    }
  },
  "meta": {...}
}
```

---

## 🌐 Web Pages (HTML Views)

### 1. Homepage (`/`)

**Purpose:** Landing page with latest articles and quick stats

**Features:**
- Hero section with tagline
- Latest articles (last 24 hours)
- Quick statistics (total articles, active feeds, latest digest)
- Call-to-action links

**HTMX Example:**
```html
<div class="grid grid-cols-1 md:grid-cols-3 gap-6">
    <!-- Stats Cards -->
    <div class="bg-white p-6 rounded-lg shadow">
        <h3 class="text-gray-500 text-sm">Total Articles</h3>
        <p class="text-3xl font-bold text-blue-600"
           hx-get="/api/articles?count=true"
           hx-trigger="load">
            Loading...
        </p>
    </div>

    <!-- Latest Articles -->
    <div class="col-span-2">
        <h2 class="text-2xl font-bold mb-4">Latest Articles</h2>
        <div hx-get="/api/articles/recent"
             hx-trigger="load"
             hx-swap="innerHTML">
            Loading articles...
        </div>
    </div>
</div>
```

---

### 2. Article List (`/articles`)

**Purpose:** Browse and filter articles

**Features:**
- Filters: category, date range, processed status
- Sorting: date, title, category
- Pagination
- Search box
- HTMX infinite scroll

**Layout:**
```
┌────────────────────────────────────────┐
│  [Search Box]         [Filters]        │
├────────────────────────────────────────┤
│  ┌──────────────────────────────────┐  │
│  │ Article Card 1                   │  │
│  │ Title, Summary, Category, Date   │  │
│  └──────────────────────────────────┘  │
│  ┌──────────────────────────────────┐  │
│  │ Article Card 2                   │  │
│  └──────────────────────────────────┘  │
│  ...                                   │
├────────────────────────────────────────┤
│  [Pagination: < 1 2 3 4 5 >]          │
└────────────────────────────────────────┘
```

**HTMX Infinite Scroll:**
```html
<div id="article-list">
    <!-- Initial articles -->
    {{range .Articles}}
        {{template "article-card" .}}
    {{end}}

    <!-- Load more trigger -->
    <div hx-get="/articles?page={{.NextPage}}"
         hx-trigger="revealed"
         hx-swap="afterend">
        <p class="text-center">Loading more...</p>
    </div>
</div>
```

---

### 3. Article Detail (`/articles/:id`)

**Purpose:** Full article view with summary and metadata

**Features:**
- Article title and source URL
- Full cleaned text
- LLM-generated summary
- Category and tags
- Published date
- Link to original article
- Related articles (same cluster)

**Layout:**
```
┌────────────────────────────────────────┐
│  ← Back to Articles                    │
├────────────────────────────────────────┤
│  Article Title                         │
│  Category: AI | Published: 2025-10-27  │
│  Source: [example.com]                 │
├────────────────────────────────────────┤
│  📝 Summary                            │
│  [LLM-generated summary]               │
├────────────────────────────────────────┤
│  📄 Full Content                       │
│  [Full article text]                   │
├────────────────────────────────────────┤
│  🔗 Related Articles                   │
│  - Related Article 1                   │
│  - Related Article 2                   │
└────────────────────────────────────────┘
```

---

### 4. Digest List (`/digests`)

**Purpose:** Browse past digests

**Features:**
- List of all generated digests
- Date, article count, format
- Preview of executive summary
- Link to full digest

---

### 5. Digest View (`/digests/:id`)

**Purpose:** View rendered digest in HTML

**Features:**
- Full digest content (executive summary, articles, categories)
- Links to source articles
- Share functionality (copy link, email)
- Download as PDF (future)

---

### 6. Feed Management (`/feeds`)

**Purpose:** Manage RSS feed sources

**Features:**
- List of active and inactive feeds
- Add new feed (URL input)
- Enable/disable feeds
- View feed statistics
- Remove feeds

**Admin Features (Optional):**
- Authentication required
- Only accessible to admin users

---

## ⚙️ Configuration

### Server Configuration (`internal/config/config.go`)

Add to existing `Config` struct:
```go
type Config struct {
    // ... existing fields ...

    Server ServerConfig `yaml:"server"`
}

type ServerConfig struct {
    Host            string        `yaml:"host"`
    Port            int           `yaml:"port"`
    ReadTimeout     time.Duration `yaml:"read_timeout"`
    WriteTimeout    time.Duration `yaml:"write_timeout"`
    ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`

    StaticDir       string        `yaml:"static_dir"`
    TemplateDir     string        `yaml:"template_dir"`

    CORS            CORSConfig    `yaml:"cors"`
    RateLimit       RateLimitConfig `yaml:"rate_limit"`
}

type CORSConfig struct {
    Enabled        bool     `yaml:"enabled"`
    AllowedOrigins []string `yaml:"allowed_origins"`
}

type RateLimitConfig struct {
    Enabled            bool `yaml:"enabled"`
    RequestsPerMinute  int  `yaml:"requests_per_minute"`
}
```

### Example `.briefly.yaml`

```yaml
# AI Configuration
ai:
  gemini:
    api_key: "${GEMINI_API_KEY}"
    model: "gemini-2.5-flash-preview-05-20"
    embedding_model: "text-embedding-004"

# Database Configuration
database:
  connection_string: "${DATABASE_URL}"
  max_connections: 25
  idle_connections: 5

# Server Configuration (NEW)
server:
  host: "0.0.0.0"
  port: 8080
  read_timeout: "15s"
  write_timeout: "15s"
  shutdown_timeout: "10s"

  static_dir: "web/static"
  template_dir: "web/templates"

  cors:
    enabled: true
    allowed_origins:
      - "http://localhost:3000"
      - "http://localhost:8080"

  rate_limit:
    enabled: true
    requests_per_minute: 60

# Cache Configuration
cache:
  enabled: true
  directory: ".briefly-cache"
  ttl:
    articles: "24h"
    summaries: "168h"
```

---

## 🚀 Implementation Phases

### Phase 1: HTTP Server Foundation (2-3 hours)

**Goal:** Basic HTTP server with health checks

**Tasks:**
1. Create `internal/server/` package
2. Implement HTTP server with chi router
3. Add middleware (logging, recovery, CORS)
4. Create `cmd/handlers/serve.go` command
5. Add server configuration to `internal/config/`
6. Implement health check endpoints

**Deliverables:**
- `briefly serve` command works
- `GET /health` returns 200 OK
- `GET /api/status` returns server status JSON
- Server logs requests
- Graceful shutdown works

**Testing:**
```bash
# Start server
briefly serve --port 8080

# Test health check
curl http://localhost:8080/health
# Expected: {"status": "ok"}

# Test status endpoint
curl http://localhost:8080/api/status
# Expected: {"version": "v3.2.0", "uptime": "5m", ...}

# Test graceful shutdown (Ctrl+C)
# Expected: "Server shutting down gracefully..."
```

---

### Phase 2: REST API Endpoints (3-4 hours)

**Goal:** Complete JSON API for articles, digests, feeds

**Tasks:**
1. Implement article API endpoints
   - `GET /api/articles` (with pagination, filtering)
   - `GET /api/articles/:id`
   - `GET /api/articles/recent`
2. Implement digest API endpoints
   - `GET /api/digests`
   - `GET /api/digests/:id`
   - `GET /api/digests/latest`
3. Implement feed API endpoints
   - `GET /api/feeds`
   - `GET /api/feeds/:id/stats`
4. Add pagination helper functions
5. Add error handling middleware
6. Write API documentation

**Deliverables:**
- All API endpoints functional
- Proper error responses (4xx, 5xx)
- Pagination works correctly
- API documented (OpenAPI/Swagger)

**Testing:**
```bash
# Test article listing
curl "http://localhost:8080/api/articles?limit=5&page=1" | jq

# Test single article
curl "http://localhost:8080/api/articles/abc123" | jq

# Test digest listing
curl "http://localhost:8080/api/digests" | jq

# Test feed stats
curl "http://localhost:8080/api/feeds/feed-123/stats" | jq
```

---

### Phase 3: Web Frontend (4-5 hours)

**Goal:** Complete HTMX-powered web UI

**Tasks:**
1. Set up template system
   - Create base layout template
   - Create component templates
2. Implement homepage
   - Latest articles
   - Quick stats
3. Implement article list page
   - Filters and sorting
   - Pagination
   - HTMX infinite scroll
4. Implement article detail page
   - Full content view
   - Related articles
5. Implement digest pages
   - Digest list
   - Digest view
6. Implement feed management page
7. Add CSS styling (Tailwind)
8. Add client-side JavaScript (minimal)

**Deliverables:**
- All web pages functional
- Responsive design (mobile-friendly)
- HTMX interactions work
- Fast page loads (<1s)

**Testing:**
```bash
# Visit pages in browser
open http://localhost:8080/
open http://localhost:8080/articles
open http://localhost:8080/articles/abc123
open http://localhost:8080/digests
open http://localhost:8080/feeds
```

---

### Phase 4: Enhanced Features (3-4 hours, Optional)

**Goal:** Add advanced functionality

**Tasks:**
1. **Search Functionality**
   - Full-text search using PostgreSQL
   - Search across title, summary, content
   - Highlight search results

2. **Real-time Updates**
   - Server-Sent Events (SSE) for article updates
   - Live notification when new articles arrive

3. **Digest Generation from Web**
   - Web form to trigger digest generation
   - Select articles manually from UI
   - Preview digest before saving

4. **RSS Feed Output**
   - Generate RSS 2.0 feed from articles
   - `GET /feed.xml` endpoint
   - `GET /feed.atom` for Atom format

5. **Authentication (Optional)**
   - Simple API key authentication
   - Admin-only feed management
   - User preferences

**Deliverables:**
- At least 2 enhanced features implemented
- Search works across all content
- RSS feed validates

---

## 🐳 Deployment

### Development Setup

**Terminal 1: Run Aggregation (Cron Simulation)**
```bash
# Fetch new articles every hour
watch -n 3600 "briefly aggregate --since 1"
```

**Terminal 2: Start Web Server**
```bash
briefly serve --port 8080 --reload
```

**Terminal 3: Monitor Database**
```bash
psql briefly -c "SELECT COUNT(*) FROM articles;"
psql briefly -c "SELECT COUNT(*) FROM feed_items WHERE processed = false;"
```

---

### Docker Deployment

**Dockerfile:**
```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /briefly ./cmd/briefly

FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /briefly .
COPY web/ ./web/

EXPOSE 8080

CMD ["./briefly", "serve", "--port", "8080"]
```

**docker-compose.yml (Extended):**
```yaml
version: "3.9"

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: briefly
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: briefly
    volumes:
      - postgres-data:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U briefly"]
      interval: 10s
      timeout: 5s
      retries: 5

  aggregator:
    build: .
    command: sh -c "while true; do ./briefly aggregate --since 1; sleep 3600; done"
    environment:
      DATABASE_URL: "postgres://briefly:${POSTGRES_PASSWORD}@postgres:5432/briefly?sslmode=disable"
      GEMINI_API_KEY: ${GEMINI_API_KEY}
    depends_on:
      postgres:
        condition: service_healthy

  web:
    build: .
    command: ["./briefly", "serve", "--port", "8080", "--host", "0.0.0.0"]
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: "postgres://briefly:${POSTGRES_PASSWORD}@postgres:5432/briefly?sslmode=disable"
      GEMINI_API_KEY: ${GEMINI_API_KEY}
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  postgres-data:
```

**.env:**
```bash
POSTGRES_PASSWORD=your-secure-password
GEMINI_API_KEY=your-gemini-api-key
```

**Run:**
```bash
docker-compose up -d
```

---

### Cloud Deployment Options

#### **Option 1: Render.com**
**Pros:** Simple, managed PostgreSQL, free tier
**Cons:** Cold starts on free tier

**Setup:**
1. Create PostgreSQL database
2. Create web service (Dockerfile)
3. Add cron job for aggregation
4. Set environment variables

#### **Option 2: Railway.app**
**Pros:** Very simple, good free tier, automatic deploys
**Cons:** Limited free hours

**Setup:**
1. Connect GitHub repo
2. Add PostgreSQL plugin
3. Add web service
4. Add cron service (separate instance)

#### **Option 3: Fly.io**
**Pros:** Global edge deployment, fast
**Cons:** More complex setup

**fly.toml:**
```toml
app = "briefly"

[build]
  builder = "paketobuildpacks/builder:base"

[[services]]
  http_checks = []
  internal_port = 8080
  protocol = "tcp"

  [[services.ports]]
    handlers = ["http"]
    port = 80

  [[services.ports]]
    handlers = ["tls", "http"]
    port = 443

[env]
  PORT = "8080"
```

#### **Option 4: DigitalOcean App Platform**
**Pros:** Managed, good performance
**Cons:** Not free

**Setup:**
1. Create app from GitHub
2. Add managed PostgreSQL
3. Configure web + worker components
4. Set environment variables

---

## 📊 Database Queries

### Queries to Add to Repositories

**`ArticleRepository` Extensions:**
```go
// GetRecent returns articles from the last N hours
func (r *ArticleRepo) GetRecent(ctx context.Context, hours int, limit int) ([]core.Article, error)

// GetByCategory returns articles filtered by category
func (r *ArticleRepo) GetByCategory(ctx context.Context, category string, page, limit int) ([]core.Article, int, error)

// GetUnprocessed returns articles that haven't been used in digests
func (r *ArticleRepo) GetUnprocessed(ctx context.Context, limit int) ([]core.Article, error)

// Search performs full-text search across articles
func (r *ArticleRepo) Search(ctx context.Context, query string, page, limit int) ([]core.Article, int, error)
```

**`DigestRepository` Extensions:**
```go
// GetLatest returns the most recent digest
func (r *DigestRepo) GetLatest(ctx context.Context) (*core.Digest, error)

// GetByDateRange returns digests within a date range
func (r *DigestRepo) GetByDateRange(ctx context.Context, start, end time.Time) ([]core.Digest, error)
```

**`FeedRepository` Extensions:**
```go
// GetStats returns statistics for a feed
func (r *FeedRepo) GetStats(ctx context.Context, feedID string) (*FeedStats, error)

// GetAllStats returns statistics for all feeds
func (r *FeedRepo) GetAllStats(ctx context.Context) ([]FeedStats, error)
```

**SQL Examples:**

```sql
-- Recent articles
SELECT * FROM articles
WHERE created_at > NOW() - INTERVAL '24 hours'
ORDER BY created_at DESC
LIMIT 20;

-- Articles by category with pagination
SELECT * FROM articles
WHERE category = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- Full-text search
SELECT *, ts_rank(
  to_tsvector('english', title || ' ' || COALESCE(cleaned_text, '')),
  plainto_tsquery('english', $1)
) AS rank
FROM articles
WHERE to_tsvector('english', title || ' ' || COALESCE(cleaned_text, ''))
  @@ plainto_tsquery('english', $1)
ORDER BY rank DESC
LIMIT 20;

-- Feed statistics
SELECT
  f.id,
  f.title,
  COUNT(fi.id) AS total_items,
  COUNT(fi.id) FILTER (WHERE fi.processed = true) AS processed_items,
  COUNT(fi.id) FILTER (WHERE fi.processed = false) AS unprocessed_items,
  MAX(fi.published) AS last_item_date,
  f.last_fetched,
  f.error_count
FROM feeds f
LEFT JOIN feed_items fi ON f.id = fi.feed_id
WHERE f.id = $1
GROUP BY f.id;
```

---

## 🧪 Testing Strategy

### Unit Tests

**`internal/server/server_test.go`:**
```go
func TestServer_HealthCheck(t *testing.T) {
    // Test health check endpoint
}

func TestServer_GracefulShutdown(t *testing.T) {
    // Test graceful shutdown
}
```

**`internal/server/api_test.go`:**
```go
func TestAPI_ListArticles(t *testing.T) {
    // Test article listing with pagination
}

func TestAPI_GetArticle(t *testing.T) {
    // Test single article retrieval
}

func TestAPI_Pagination(t *testing.T) {
    // Test pagination logic
}
```

### Integration Tests

**`test/integration/server_test.go`:**
```go
func TestServerIntegration(t *testing.T) {
    // Start server, make requests, verify responses
}

func TestArticleAPIIntegration(t *testing.T) {
    // Test full article API flow
}
```

### Load Testing

**Using `wrk` or `hey`:**
```bash
# Install hey
go install github.com/rakyll/hey@latest

# Test article listing
hey -n 1000 -c 10 http://localhost:8080/api/articles

# Test homepage
hey -n 500 -c 5 http://localhost:8080/
```

### End-to-End Testing

**Manual Test Checklist:**
- [ ] Homepage loads and displays latest articles
- [ ] Article list shows correct pagination
- [ ] Filters work (category, date)
- [ ] Single article view displays full content
- [ ] Digest list shows all digests
- [ ] Digest view renders correctly
- [ ] Feed management works (add, list, remove)
- [ ] API endpoints return correct JSON
- [ ] Error pages display properly (404, 500)
- [ ] Mobile responsive design works
- [ ] HTMX interactions work smoothly

---

## 📈 Monitoring & Observability

### Metrics to Track

**Server Metrics:**
- Request count (total, by endpoint)
- Response times (p50, p95, p99)
- Error rates (4xx, 5xx)
- Active connections

**Application Metrics:**
- Database query times
- Cache hit rate
- Template render times
- API response sizes

**Business Metrics:**
- Total articles in database
- Articles per feed
- Digest generation count
- Active feeds

### Logging

**Structured Logging:**
```go
log.Info("HTTP request",
    "method", r.Method,
    "path", r.URL.Path,
    "status", status,
    "duration", duration,
    "ip", r.RemoteAddr,
)
```

### Health Checks

**`/health` Endpoint:**
```json
{
  "status": "ok",
  "checks": {
    "database": "ok",
    "memory": "ok"
  }
}
```

**`/api/status` Endpoint:**
```json
{
  "version": "v3.2.0",
  "uptime": "5h 23m",
  "database": {
    "connected": true,
    "articles": 1250,
    "digests": 15,
    "feeds": 12
  }
}
```

---

## 🔒 Security Considerations

### Current State
- No authentication (public read access)
- CORS enabled for specific origins
- Rate limiting enabled
- Input validation on API parameters

### Future Enhancements
1. **API Key Authentication** (for feed management)
2. **Admin Panel** (protected by auth)
3. **HTTPS/TLS** (in production)
4. **Content Security Policy (CSP)**
5. **XSS Protection** (template escaping)
6. **SQL Injection Protection** (parameterized queries)

---

## 📚 Dependencies

### New Go Packages

**Router:**
- `github.com/go-chi/chi/v5` - Lightweight HTTP router

**OR Alternative:**
- `github.com/gorilla/mux` - Popular router
- Standard library `net/http` with custom multiplexer

**Middleware:**
- `github.com/go-chi/cors` - CORS middleware
- `github.com/go-chi/httprate` - Rate limiting

**Templates:**
- Standard library `html/template` - Server-side templates

**OR Alternative:**
- `github.com/a-h/templ` - Type-safe Go templates

### Frontend Dependencies (CDN)

**HTMX:**
```html
<script src="https://unpkg.com/htmx.org@1.9.10"></script>
```

**Tailwind CSS:**
```html
<link href="https://cdn.jsdelivr.net/npm/tailwindcss@2/dist/tailwind.min.css" rel="stylesheet">
```

**OR Alpine.js (for client-side interactions):**
```html
<script defer src="https://cdn.jsdelivr.net/npm/alpinejs@3.x.x/dist/cdn.min.js"></script>
```

---

## 🎯 Success Criteria

### Functional Requirements
- ✅ `briefly serve` command starts HTTP server
- ✅ All REST API endpoints return correct JSON
- ✅ Web pages render correctly with HTMX
- ✅ Pagination works on article list
- ✅ Article detail page shows full content
- ✅ Digest view renders markdown correctly
- ✅ Feed management UI works

### Performance Requirements
- ✅ Homepage loads in <1 second
- ✅ API responses return in <200ms (p95)
- ✅ Database queries optimized with indexes
- ✅ Server handles 100 concurrent requests

### Quality Requirements
- ✅ No server crashes under load
- ✅ Graceful error handling (404, 500)
- ✅ Responsive design (mobile-friendly)
- ✅ Browser compatibility (Chrome, Firefox, Safari)
- ✅ Unit test coverage >70%

---

## 📅 Timeline Estimate

**Total Time: 12-16 hours (2-3 days part-time)**

| Phase | Time | Description |
|-------|------|-------------|
| Phase 1: HTTP Server Foundation | 2-3 hours | Basic server, health checks, configuration |
| Phase 2: REST API Endpoints | 3-4 hours | Complete JSON API for articles, digests, feeds |
| Phase 3: Web Frontend | 4-5 hours | HTMX templates, pages, styling |
| Phase 4: Enhanced Features | 3-4 hours | Search, real-time updates, RSS output (optional) |

**Additional Time:**
- Testing: +2-3 hours
- Documentation: +1-2 hours
- Deployment: +1-2 hours

---

## 🚦 Next Steps

### Immediate Actions

1. **Review this plan** with stakeholders
2. **Set up development environment**
   - Ensure PostgreSQL is running
   - Test `briefly aggregate` command
   - Verify database has data

3. **Start Phase 1: HTTP Server Foundation**
   - Create `internal/server/` package
   - Implement basic HTTP server
   - Add `briefly serve` command

### Questions to Resolve

1. **Template System Choice:**
   - Go `html/template` (simple, standard)
   - Templ (type-safe, compile-time)
   - Full SPA with React (complex, requires build)
   - **Recommendation:** Start with `html/template` + HTMX

2. **Router Choice:**
   - `chi` (lightweight, fast)
   - `gorilla/mux` (popular, well-tested)
   - Standard library (minimal dependencies)
   - **Recommendation:** `chi` for simplicity

3. **Authentication:**
   - None (public read-only)
   - API key (simple, for admin)
   - OAuth (complex, for multi-user)
   - **Recommendation:** Start without auth, add API key later

4. **Deployment Target:**
   - Docker Compose (local/VPS)
   - Render.com (managed, simple)
   - Railway.app (managed, auto-deploy)
   - Fly.io (edge, global)
   - **Recommendation:** Start with Docker Compose, then Render.com

---

## 📖 References

### Documentation to Create

1. **API Documentation** (OpenAPI/Swagger)
2. **Deployment Guide** (Docker, cloud platforms)
3. **Development Setup** (local environment)
4. **Architecture Diagram** (system overview)

### Related Documents

- **NEWS_AGGREGATOR.md** - Phase 1 implementation (RSS aggregation)
- **DOCKER.md** - Docker setup guide
- **MIGRATIONS.md** - Database migration guide
- **api-contracts.yaml** - API contract specification

---

## ✅ Summary

This plan outlines a complete implementation of web serving capability for **briefly**, transforming it from a CLI-only tool into a web-accessible news aggregator. The implementation follows the existing simplified architecture (v3.0), leveraging PostgreSQL persistence and the pipeline system.

**Key Highlights:**
- **Simple stack:** Go server + HTMX + Tailwind CSS (no complex frontend build)
- **Modular design:** Clean separation between API and web UI
- **Scalable:** Ready for Docker and cloud deployment
- **Maintainable:** Follows existing code patterns and conventions
- **Fast:** Server-side rendering, aggressive caching, optimized queries

**Recommended Next Step:** Begin with Phase 1 (HTTP Server Foundation) to establish the foundation, then iterate through phases 2-4.

---

**Document Version:** 1.0
**Last Updated:** October 28, 2025
**Author:** Claude (AI Assistant)
**Status:** Ready for implementation
