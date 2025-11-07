# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Version Information

**Current Version:** v3.1.0-hierarchical-summarization
**Architecture:** Database-driven pipeline with hierarchical summarization

## Development Commands

### Building and Running
```bash
# Build the main application
go build -o briefly ./cmd/briefly

# Build and install to $GOPATH/bin
go install ./cmd/briefly

# Run from source during development
go run ./cmd/briefly digest generate --since 7

# Run tests (standard)
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests with race detection and coverage (CI mode)
go test -race -coverprofile=coverage.out -covermode=atomic ./...

# Run linting (install golangci-lint if needed)
golangci-lint run --timeout=5m
# If not in PATH: $(go env GOPATH)/bin/golangci-lint run --timeout=5m

# Basic Go formatting and vetting
go fmt ./...
go vet ./...

# Clean dependencies
go mod tidy
```

### Core Commands

**Feed Management:**
```bash
# Add RSS/Atom feeds
briefly feed add https://hnrss.org/newest
briefly feed add https://blog.golang.org/feed.atom

# List all feeds
briefly feed list

# Remove a feed
briefly feed remove <feed-id>
```

**News Aggregation:**
```bash
# Aggregate articles from feeds (run daily)
briefly aggregate --since 24

# Aggregate with specific themes
briefly aggregate --since 24 --themes
```

**Weekly Digest Generation:**
```bash
# Generate LinkedIn-ready digest from classified articles (database-driven)
briefly digest generate --since 7

# Generate with specific output directory
briefly digest generate --since 7 --output digests

# List recent digests
briefly digest list --limit 20

# Show specific digest
briefly digest show <digest-id>
```

**Quick Article Summary:**
```bash
# Get quick summary of single article
briefly read https://example.com/article

# Force fresh fetch (bypass cache)
briefly read --no-cache https://example.com/article

# Raw output without formatting
briefly read --raw https://example.com/article
```

**Cache Management:**
```bash
# View cache statistics
briefly cache stats

# Clear all cached data
briefly cache clear --confirm
```

**Theme Management (Phase 0):**
```bash
# List all enabled themes
briefly theme list

# List all themes (including disabled)
briefly theme list --all

# Add a new theme
briefly theme add "Theme Name" --description "Description" --keywords "keyword1,keyword2"

# Update a theme
briefly theme update <id> --description "New description" --keywords "new,keywords"

# Enable/disable themes
briefly theme enable <id>
briefly theme disable <id>

# Remove a theme
briefly theme remove <id>
```

**Manual URL Submission (Phase 0):**
```bash
# Submit one or more URLs for processing
briefly url add https://example.com/article1
briefly url add https://example.com/article1 https://example.com/article2

# List submitted URLs
briefly url list
briefly url list --status pending
briefly url list --status processed

# Check status of a specific URL
briefly url status <id>

# Retry failed URLs
briefly url retry <id>
briefly url retry --all  # Retry all failed

# Clear processed/failed URLs
briefly url clear --processed --failed
```

## Architecture Overview

### Design Philosophy

The architecture is designed around a **database-driven news aggregation workflow** with **hierarchical summarization**:

1. **Aggregate** - Fetch articles from RSS feeds and manual submissions
2. **Classify** - Categorize articles by theme using LLM
3. **Store** - Persist articles in PostgreSQL with relationships
4. **Digest** - Generate weekly digests using hierarchical summarization
5. **Render** - Create LinkedIn-ready markdown output

### Key Innovation: Hierarchical Summarization

The digest generation uses a **two-stage hierarchical approach**:

**Stage 1: Cluster-Level Narratives**
- For each topic cluster, generate a comprehensive narrative from **ALL articles** in that cluster
- Each cluster narrative is 2-3 paragraphs synthesizing all related articles
- No articles are excluded (no "top 3" limitation)

**Stage 2: Executive Summary**
- Synthesize cluster narratives into a cohesive executive summary
- References articles by citation number `[1][2][3]`
- Short, concise, but grounded in ALL articles

**Benefits:**
- ✅ **No information loss** - Every article contributes to the digest
- ✅ **Well-grounded summaries** - Executive summary reflects all content
- ✅ **Maintains conciseness** - Summary stays short by synthesizing clusters, not all 20+ individual articles

### Project Structure

```
briefly/
├── cmd/
│   ├── briefly/main.go          # Entry point (uses ExecuteSimplified)
│   └── handlers/                 # Cobra command handlers
│       ├── root_simplified.go    # Root command
│       ├── digest_generate.go    # Database-driven digest generation
│       ├── digest.go             # Digest command group
│       ├── aggregate.go          # News aggregation
│       ├── feed.go               # Feed management
│       ├── read_simplified.go    # Quick article summary
│       ├── cache.go              # Cache management
│       ├── theme.go              # Theme management
│       └── manual_url.go         # Manual URL submission
├── internal/
│   ├── parser/                   # URL parsing from markdown
│   ├── summarize/                # Centralized summarization with prompts
│   ├── narrative/                # Executive summary generation
│   ├── pipeline/                 # Orchestration layer
│   │   ├── pipeline.go           # Core orchestrator
│   │   ├── interfaces.go         # Component contracts
│   │   ├── adapters.go           # Wrapper adapters for existing packages
│   │   ├── builder.go            # Fluent API for construction
│   │   └── theme_categorizer.go  # NEW Phase 0: Theme-based categorization
│   ├── clustering/               # K-means topic clustering
│   ├── core/                     # Core data structures (Article, Summary, Digest, Theme, ManualURL)
│   ├── fetch/                    # Content fetching (HTML, PDF, YouTube)
│   ├── llm/                      # LLM client for Gemini API
│   │   └── traced_client.go      # NEW Phase 0: LangFuse-traced LLM client
│   ├── observability/            # NEW Phase 0: Observability infrastructure
│   │   ├── langfuse.go           # LangFuse tracing (local logging mode)
│   │   └── posthog.go            # PostHog analytics tracking
│   ├── themes/                   # NEW Phase 0: Theme classification system
│   │   └── classifier.go         # LLM-based theme classifier
│   ├── persistence/              # NEW Phase 0: Database abstraction layer
│   │   ├── interfaces.go         # Repository interfaces
│   │   ├── postgres_repos.go     # PostgreSQL implementations
│   │   └── migrations/           # Database migrations (001-007+)
│   ├── sources/                  # NEW Phase 0: Feed source management
│   │   └── manager.go            # RSS feeds + manual URL aggregation
│   ├── server/                   # NEW Phase 0: Web server
│   │   ├── server.go             # HTTP server setup
│   │   ├── theme_handlers.go     # Theme management API
│   │   ├── manual_url_handlers.go # Manual URL API
│   │   └── web_pages.go          # Web UI pages (/themes, /submit)
│   ├── store/                    # SQLite caching (being phased out for PostgreSQL)
│   ├── templates/                # Digest format templates
│   ├── render/                   # Output formatting
│   ├── email/                    # HTML email templates
│   ├── config/                   # Configuration management
│   └── logger/                   # Structured logging
├── docs/
│   ├── executions/               # NEW Phase 0: Execution tracking
│   │   └── 2025-10-31.md         # Phase 0-1 implementation plan
│   └── simplified-architecture/  # Architecture design documents
│       ├── data-flow.md
│       ├── components.md
│       ├── data-model.yaml
│       ├── api-contracts.yaml
│       └── UNUSED_PACKAGES.md
└── test/
    └── (integration tests removed - pending rewrite)
```

### Removed Packages (v3.0 Cleanup)

**18 packages removed** (~18,797 lines) that were not part of the core weekly digest workflow:

- `alerts/` - Alert monitoring system
- `categorization/` - Replaced by clustering
- `cost/` - API cost estimation
- `deepresearch/` - Multi-stage research pipeline
- `feeds/` - RSS feed processing
- `interactive/` - Interactive selection mode
- `messaging/` - Slack/Discord integration
- `ordering/` - Article ordering (stubbed in pipeline)
- `relevance/` - Relevance scoring system
- `research/` - Research query generation
- `search/` - Web search integration
- `sentiment/` - Sentiment analysis
- `services/` - Service layer (replaced by pipeline interfaces)
- `summaries/` - Legacy summary handling
- `trends/` - Trend analysis
- `tts/` - Text-to-speech generation
- `tui/` - Terminal UI browser
- `visual/` - Banner generation (future)

### Pipeline Architecture

**Core Concept:** Database-driven workflow with hierarchical summarization

**Digest Generation Pipeline (9 Steps):**

1. **Parse URLs** - Extract URLs from database (feeds + manual submissions)
2. **Fetch & Summarize** - Retrieve content and generate summaries (fetch, summarize)
3. **Generate Embeddings** - Create 768-dim vectors for clustering (llm)
4. **Cluster Articles** - Group by topic similarity using K-means (clustering)
5. **🆕 Generate Cluster Narratives** - Synthesize ALL articles in each cluster into 2-3 paragraph narrative (hierarchical stage 1)
6. **Generate Digest Content** - Create executive summary from cluster narratives (hierarchical stage 2)
7. **Build Digest** - Construct final digest structure
8. **Render Markdown** - Create LinkedIn-ready output
9. **Store in Database** - Persist digest with relationships

**Key Files:**

- `internal/pipeline/pipeline.go` - Central orchestrator (GenerateDigests)
- `internal/narrative/generator.go` - Hierarchical summarization logic
- `internal/core/core.go` - ClusterNarrative and TopicCluster structs
- `internal/pipeline/interfaces.go` - Component contracts

### Data Flow (Hierarchical Summarization)

```
Database (Articles) → Fetcher → Articles + Summaries (with cache)
    ↓
Summaries → LLM → Embeddings (768-dim vectors)
    ↓
Articles + Embeddings → Clusterer → TopicClusters (K-means)
    ↓
TopicClusters + ALL Articles → ClusterNarrative Generator → Cluster Narratives
    ↓                                                           (2-3 paragraphs each)
Cluster Narratives → Executive Summary Generator → Digest Summary
    ↓
All Data → Builder → Digest Structure
    ↓
Digest → Renderer → Markdown File
    ↓
Output: digests/digest_2025-11-06.md
```

**Hierarchical Flow:**
```
Stage 1: Articles (per cluster) → Cluster Narrative (synthesizes ALL)
Stage 2: Cluster Narratives → Executive Summary (concise synthesis)
```

### Core Data Structures

**Article** (`internal/core/core.go`):
- `ID`, `URL`, `Title`, `ContentType` (html, pdf, youtube)
- `CleanedText`, `RawContent`
- `TopicCluster`, `ClusterConfidence`
- `Embedding` []float64 (populated during pipeline)

**Summary** (`internal/core/core.go`):
- `ID`, `ArticleIDs` []string
- `SummaryText`, `ModelUsed`
- Used for both article summaries and executive summaries

**ClusterNarrative** (`internal/core/core.go`) - NEW for hierarchical summarization:
- `Title` string - Short, punchy cluster title (5-8 words)
- `Summary` string - 2-3 paragraph narrative synthesizing ALL articles
- `KeyThemes` []string - 3-5 main themes from the cluster
- `ArticleRefs` []int - Citation numbers of articles included
- `Confidence` float64 - Cluster coherence confidence (0-1)

**TopicCluster** (`internal/core/core.go`):
- `Label` - Auto-generated cluster name
- `ArticleIDs` []string - Articles in this cluster
- `Centroid` []float64 - K-means centroid
- `Narrative` *ClusterNarrative - Generated cluster summary (hierarchical summarization)

**Digest** (`internal/core/core.go`):
- `ArticleGroups` []ArticleGroup - Clustered articles
- `DigestSummary` string - Executive summary (generated from cluster narratives)
- `KeyMoments` []KeyMoment - Important quotes with citations
- `Metadata` - Title, date, article count

### Component Interfaces (v3.0)

All major components implement clean interfaces defined in `internal/pipeline/interfaces.go`:

```go
type URLParser interface {
    ParseMarkdownFile(filePath string) ([]core.Link, error)
}

type ContentFetcher interface {
    FetchArticle(ctx context.Context, url string) (*core.Article, error)
}

type ArticleSummarizer interface {
    SummarizeArticle(ctx context.Context, article *core.Article) (*core.Summary, error)
}

type EmbeddingGenerator interface {
    GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
}

type TopicClusterer interface {
    ClusterArticles(ctx context.Context, articles []core.Article,
        summaries []core.Summary, embeddings map[string][]float64) ([]core.TopicCluster, error)
}

type NarrativeGenerator interface {
    GenerateExecutiveSummary(ctx context.Context, clusters []core.TopicCluster,
        articles map[string]core.Article, summaries map[string]core.Summary) (string, error)
}

type MarkdownRenderer interface {
    RenderDigest(ctx context.Context, digest *core.Digest, outputPath string) (string, error)
    RenderQuickRead(ctx context.Context, article *core.Article, summary *core.Summary) (string, error)
}
```

### Configuration Management

**Hierarchical Configuration (Viper):**

1. Command-line flags (highest priority)
2. Environment variables (loaded from `.env` via `godotenv`)
3. Configuration file (`.briefly.yaml` or `--config`)
4. Default values (lowest priority)

**Key Settings:**
```yaml
ai:
  gemini:
    api_key: "your-gemini-api-key"
    model: "gemini-2.5-flash-preview-05-20"

cache:
  enabled: true
  directory: ".briefly-cache"
  ttl: 24h

clustering:
  min_clusters: 2
  max_clusters: 5
  algorithm: "kmeans"
```

### Caching Strategy

**Multi-layer SQLite caching** (`.briefly-cache/`):

- **Articles**: 24-hour TTL, content hash validation
- **Summaries**: 7-day TTL, linked to article content hash
- **Digest metadata**: Persistent for trend analysis

**Cache Commands:**
```bash
briefly cache stats   # View statistics
briefly cache clear --confirm  # Clear all data
```

### Testing

**Test Coverage (v3.0):**
- `internal/parser/parser_test.go` - 7 test suites (✓ passing)
- `internal/summarize/summarizer_test.go` - 14 test suites (✓ passing)
- `internal/core/core_test.go` - Core data structures
- `internal/llm/llm_test.go` - LLM operations
- `internal/templates/templates_test.go` - Template rendering
- `internal/fetch/fetch_test.go` - Content fetching
- `internal/store/store_test.go` - Store operations
- `internal/email/email_test.go` - Email generation
- `internal/render/render_test.go` - Render functionality

**Note:** Integration tests were removed during simplification and need rewrite.

**Run Tests:**
```bash
# Run all unit tests
go test ./...

# Run specific package tests
go test ./internal/parser
go test ./internal/summarize

# Run with race detection (CI mode)
go test -race ./...
```

### Development Patterns

**Pipeline Construction:**
```go
// Build pipeline with dependencies
builder := pipeline.NewBuilder().
    WithLLMClient(llmClient).
    WithCacheDir(".briefly-cache").
    Build()

pipe, err := builder.Build()

// Execute digest generation
result, err := pipe.GenerateDigest(ctx, pipeline.DigestOptions{
    InputFile:      "input/links.md",
    OutputPath:     "digests",
    GenerateBanner: false,
})
```

**Error Handling:**
- Graceful degradation: article failures don't stop pipeline
- Non-fatal errors: executive summary failure continues execution
- Comprehensive logging: every step shows progress

**Context Propagation:**
```go
// All service methods accept context for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
result, err := pipe.GenerateDigest(ctx, opts)
```

## Common Workflows

### Adding New Content Fetchers

1. Implement content detection in `internal/fetch/processor.go`:
   ```go
   func (cp *ContentProcessor) detectContentType(url string) (core.ContentType, error)
   ```

2. Add processing function (e.g., `ProcessNewType`):
   ```go
   func ProcessNewType(link core.Link) (core.Article, error) {
       // Fetch and parse content
       // Populate article with URL, Title, CleanedText, ContentType
       article := core.Article{
           ID:          uuid.NewString(),
           URL:         link.URL,  // IMPORTANT: Set URL field
           Title:       extractedTitle,
           ContentType: core.ContentTypeNew,
           CleanedText: cleanedContent,
           DateFetched: time.Now().UTC(),
       }
       return article, nil
   }
   ```

3. Add to processor switch in `ProcessArticle()`

### Extending Summarization

**Add New Prompt Type:**

1. Define prompt in `internal/summarize/prompts.go`:
   ```go
   func BuildNewStylePrompt(content string, opts PromptOptions) string
   ```

2. Use in `Summarizer.SummarizeArticle()`:
   ```go
   prompt := prompts.BuildNewStylePrompt(article.CleanedText, opts)
   ```

### Adding Pipeline Steps

1. Define interface in `internal/pipeline/interfaces.go`
2. Implement adapter in `internal/pipeline/adapters.go`
3. Add to pipeline in `internal/pipeline/pipeline.go`
4. Wire up in builder: `internal/pipeline/builder.go`

### Debugging

**Comprehensive Logging:**

Every pipeline step logs progress:
```
📄 Step 1/9: Parsing URLs from input/links.md...
   ✓ Found 16 URLs

🔍 Step 2/9: Fetching and summarizing articles...
   [1/16] Processing: https://example.com
           ✓ Cache hit
   [2/16] Processing: https://example.com/2
           ✓ Fetched and summarized
   ...

🧠 Step 3/9: Generating embeddings for clustering...
   [1/13] Generating embedding for summary abc123
           ✓ Embedding generated (768 dimensions)
```

**Cache Debugging:**
```bash
# Clear cache if seeing stale data
briefly cache clear --confirm

# Check cache statistics
briefly cache stats
```

## API Requirements

**Required:**
- `GEMINI_API_KEY` - Gemini API key for summarization and embeddings
- `DATABASE_URL` - PostgreSQL connection string (Phase 0+)

**Phase 0 Observability (Optional but Recommended):**
- `LANGFUSE_PUBLIC_KEY` - LangFuse public key for LLM tracing
- `LANGFUSE_SECRET_KEY` - LangFuse secret key
- `LANGFUSE_HOST` - LangFuse server URL (default: https://cloud.langfuse.com)
- `POSTHOG_API_KEY` - PostHog API key for analytics
- `POSTHOG_HOST` - PostHog server URL (default: https://app.posthog.com)

**Other Optional:**
- `OPENAI_API_KEY` - For future banner generation

**Configuration:**
Set in `.env` file or environment:
```bash
# Required
export GEMINI_API_KEY="your-key-here"
export DATABASE_URL="postgresql://user:pass@localhost:5432/briefly"

# Observability (Phase 0)
export LANGFUSE_PUBLIC_KEY="pk-lf-..."
export LANGFUSE_SECRET_KEY="sk-lf-..."
export LANGFUSE_HOST="https://cloud.langfuse.com"
export POSTHOG_API_KEY="phc_..."
export POSTHOG_HOST="https://app.posthog.com"
```

**Note:** LangFuse is currently in local logging mode. HTTP API integration pending.

## Performance Considerations

- **SQLite caching** reduces redundant API calls (being migrated to PostgreSQL)
- **Concurrent processing** planned but not yet implemented
- **Typical processing time**: ~2-3 minutes for 13 articles
- **Cache hit rate**: 0-60% depending on previous runs

## Phase 0 Features (Implemented)

### Theme System
**Database-driven theme classification with LLM-based relevance scoring**

- **10 Default Themes** seeded on first run (AI/ML, Cloud/DevOps, Software Engineering, etc.)
- **CLI Management**: Full CRUD operations via `briefly theme` commands
- **Web UI**: Theme management interface at `/themes`
- **LLM Classification**: Articles automatically classified using Gemini with JSON prompts
- **Relevance Threshold**: 0.4 (40%) minimum score required for theme assignment
- **Theme Structure**:
  - Name, description, keywords
  - Enable/disable toggle
  - Used in pipeline categorization

**Files**: `internal/themes/classifier.go`, `internal/pipeline/theme_categorizer.go`, `cmd/handlers/theme.go`

### Manual URL Submission
**User-submitted URLs with status tracking and automatic processing**

- **CLI Commands**: `briefly url add/list/status/retry/clear`
- **Web UI**: Submission form at `/submit`
- **Status Flow**: `pending` → `processing` → `processed`/`failed`
- **Auto-Processing**: Integrated with `briefly aggregate` command
- **Feed Integration**: Manual URLs converted to feed items for unified processing
- **Error Handling**: Failed URLs tracked with error messages, retry capability

**Files**: `internal/sources/manager.go` (AggregateManualURLs), `cmd/handlers/manual_url.go`

### Observability Infrastructure
**LangFuse + PostHog tracking for LLM operations and user analytics**

**LangFuse (LLM Tracing):**
- Wraps all Gemini API calls via `TracedClient`
- Tracks: prompts, completions, tokens, latency, costs
- Currently: Local logging mode (stdout)
- Future: HTTP API integration when SDK stabilizes
- Files: `internal/observability/langfuse.go`, `internal/llm/traced_client.go`

**PostHog (Analytics):**
- Fully integrated with official Go SDK
- Tracks key events:
  - Digest generation, article processing, theme classification
  - Manual URL submissions, article clicks, theme filters
  - LLM calls (model, operation, tokens, latency)
- Frontend tracking in web pages (`/themes`, `/submit`)
- Files: `internal/observability/posthog.go`

### Database Migration (PostgreSQL)
**Replaced SQLite with PostgreSQL for production scalability**

- **Repository Pattern**: Clean abstractions in `internal/persistence/interfaces.go`
- **7 Migrations** (as of Phase 0):
  1. Initial schema (articles, summaries, feeds)
  2. Feed items
  3. Themes table
  4. Manual URLs table
  5. Article-theme relationships
  6. Default theme seeds
  7. Manual submissions feed
- **Graceful Fallback**: Observability clients optional (no crashes if disabled)

## Known Issues / Future Work

1. **Executive Summary Generation**: Currently failing (non-fatal)
   - Located in `internal/narrative/generator.go`
   - Pipeline continues without it

2. **Banner Generation**: Stubbed, not implemented
   - Interface defined in `internal/pipeline/interfaces.go`
   - Adapter returns "not yet implemented"

3. **Integration Tests**: Removed during cleanup
   - Need rewrite for new pipeline architecture

4. **Article Ordering**: Stubbed implementation
   - `OrdererAdapter` in `internal/pipeline/adapters.go`
   - Currently returns clusters unchanged

5. **Concurrent Processing**: Not implemented
   - Articles processed sequentially
   - TODO in `pipeline.go:290`

## Migration Notes (v2.0 → v3.0)

**Breaking Changes:**
- 8 commands → 3 commands (digest, read, cache)
- Many advanced features removed (research, tui, messaging, etc.)
- Service layer replaced by pipeline architecture
- Integration tests need rewrite

**Benefits:**
- 56% fewer packages (32 → 14)
- ~20,000 lines of code removed
- Focused on core workflow
- Clean architecture with interfaces
- Comprehensive logging

**Upgrade Path:**
If you need removed features (research, sentiment, alerts, etc.), use v2.0 on the `main` branch.

## Git Workflow

**Branches:**
- `main` - v2.0 with all features
- `simplify-architecture` - v3.0 simplified (current development)

**Commit Tags:**
- `v2.0-before-simplification` - Last commit before refactor
- Future: `v3.0.0` - Release tag when complete

## Useful Commands Reference

```bash
# Build and test
go build -o briefly ./cmd/briefly
go test ./...

# Add feeds
./briefly feed add https://hnrss.org/newest

# Aggregate news (run daily)
./briefly aggregate --since 24

# Generate digest (database-driven with hierarchical summarization)
./briefly digest generate --since 7

# List recent digests
./briefly digest list --limit 20

# Quick read
./briefly read https://example.com/article

# Cache management
./briefly cache stats
./briefly cache clear --confirm

# Linting
golangci-lint run --timeout=5m

# View help
./briefly --help
./briefly digest --help
./briefly aggregate --help
```

## Documentation

- `docs/simplified-architecture/` - Architecture design documents
- `docs/simplified-architecture/UNUSED_PACKAGES.md` - List of removed packages
- README.md - User-facing documentation
