# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Version Information

**Current Version:** v3.0-simplified (simplify-architecture branch)
**Architecture:** Clean pipeline-based design focused on weekly digest generation

## Development Commands

### Building and Running
```bash
# Build the main application
go build -o briefly ./cmd/briefly

# Build and install to $GOPATH/bin
go install ./cmd/briefly

# Run from source during development
go run ./cmd/briefly digest input/links.md

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

### Core Commands (v3.0 Simplified)

**Weekly Digest Generation:**
```bash
# Generate LinkedIn-ready digest from markdown file with URLs
briefly digest input/weekly-links.md

# Specify custom output directory
briefly digest --output digests input/weekly-links.md

# Generate with banner image (not yet implemented)
briefly digest --with-banner input/weekly-links.md

# Dry run (cost estimation - not yet implemented)
briefly digest --dry-run input/weekly-links.md
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

## Architecture Overview (v3.0 Simplified)

### Design Philosophy

The v3.0 architecture is a **breaking refactor** designed around the actual weekly digest workflow:

1. **Collect URLs manually** (outside app)
2. **Run digest command** to process URLs
3. **Cluster articles** by topic similarity
4. **Generate executive summary** from top articles
5. **Render LinkedIn-ready markdown**
6. **Copy to LinkedIn** (manual)

### Project Structure

```
briefly/
├── cmd/
│   ├── briefly/main.go          # Entry point (uses ExecuteSimplified)
│   └── handlers/                 # Cobra command handlers
│       ├── root_simplified.go    # 3-command root (digest, read, cache)
│       ├── digest_simplified.go  # Weekly digest generation
│       ├── read_simplified.go    # Quick article summary
│       └── cache.go              # Cache management
├── internal/
│   ├── parser/                   # NEW: URL parsing from markdown
│   ├── summarize/                # NEW: Centralized summarization with prompts
│   ├── narrative/                # NEW: Executive summary generation
│   ├── pipeline/                 # NEW: Orchestration layer
│   │   ├── pipeline.go           # Core orchestrator
│   │   ├── interfaces.go         # Component contracts
│   │   ├── adapters.go           # Wrapper adapters for existing packages
│   │   └── builder.go            # Fluent API for construction
│   ├── clustering/               # K-means topic clustering
│   ├── core/                     # Core data structures
│   ├── fetch/                    # Content fetching (HTML, PDF, YouTube)
│   ├── llm/                      # LLM client for Gemini API
│   ├── store/                    # SQLite caching
│   ├── templates/                # Digest format templates
│   ├── render/                   # Output formatting
│   ├── email/                    # HTML email templates
│   ├── config/                   # Configuration management
│   └── logger/                   # Structured logging
├── docs/
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

### Pipeline Architecture (v3.0)

**Core Concept:** Clean separation of concerns with adapter pattern

**9-Step Processing Pipeline:**

1. **Parse URLs** - Extract URLs from markdown file (parser)
2. **Fetch & Summarize** - Retrieve content and generate summaries (fetch, summarize)
3. **Generate Embeddings** - Create 768-dim vectors for clustering (llm)
4. **Cluster Articles** - Group by topic similarity using K-means (clustering)
5. **Order Articles** - Sort articles within clusters (stubbed)
6. **Executive Summary** - Generate narrative from top 3 per cluster (narrative)
7. **Build Digest** - Construct final digest structure (pipeline)
8. **Render Markdown** - Create LinkedIn-ready output (templates, render)
9. **Generate Banner** - Optional AI image generation (future)

**Key Files:**

- `internal/pipeline/pipeline.go` - Central orchestrator with comprehensive logging
- `internal/pipeline/interfaces.go` - Component contracts (URLParser, ContentFetcher, etc.)
- `internal/pipeline/adapters.go` - Wrappers for existing packages
- `internal/pipeline/builder.go` - Fluent API for pipeline construction

### Data Flow

```
Markdown File → Parser → URLs
    ↓
URLs → Fetcher → Articles (with cache)
    ↓
Articles → Summarizer → Summaries (with cache)
    ↓
Summaries → LLM → Embeddings (768-dim vectors)
    ↓
Articles + Embeddings → Clusterer → TopicClusters (K-means)
    ↓
Clusters → Orderer → OrderedClusters (priority-based)
    ↓
Clusters + Articles + Summaries → Narrative → Executive Summary
    ↓
All Data → Builder → Digest Structure
    ↓
Digest → Renderer → Markdown File
    ↓
Output: digests/digest_signal_2025-10-01.md
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

**TopicCluster** (`internal/core/core.go`):
- `Label` - Auto-generated cluster name
- `ArticleIDs` []string - Articles in this cluster
- `Centroid` []float64 - K-means centroid

**Digest** (`internal/core/core.go`):
- `ArticleGroups` []ArticleGroup - Clustered articles
- `DigestSummary` string - Executive summary
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

**Optional:**
- `OPENAI_API_KEY` - For future banner generation

**Configuration:**
Set in `.env` file or environment:
```bash
export GEMINI_API_KEY="your-key-here"
```

## Performance Considerations

- **SQLite caching** reduces redundant API calls
- **Concurrent processing** planned but not yet implemented
- **Typical processing time**: ~2-3 minutes for 13 articles
- **Cache hit rate**: 0-60% depending on previous runs

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

# Generate digest
./briefly digest input/weekly-links.md

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
./briefly read --help
```

## Documentation

- `docs/simplified-architecture/` - Architecture design documents
- `docs/simplified-architecture/UNUSED_PACKAGES.md` - List of removed packages
- README.md - User-facing documentation
