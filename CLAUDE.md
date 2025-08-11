# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
```bash
# Build the main application
go build -o briefly ./cmd/briefly

# Build and install to $GOPATH/bin
go install ./cmd/briefly

# Run from source during development
go run ./cmd/briefly digest input/test-links.md

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

### Key Development Commands

**Core Digest Generation:**
```bash
# v2.0 Smart Concise Digests with relevance filtering
briefly digest input/links.md                                   # Standard 400-word digest
briefly digest --format scannable input/links.md               # New scannable newsletter format
briefly digest --format brief --max-words 200 input/links.md   # Ultra-concise 200-word digest

# Multi-channel outputs
briefly digest --format email input/links.md                   # HTML email format
briefly digest --format slack --slack-webhook "URL" input/links.md
briefly digest --format discord --discord-webhook "URL" input/links.md
briefly digest --format audio --tts-provider openai input/links.md

# AI-powered enhancements
briefly digest --with-banner --format newsletter input/links.md  # AI banner images
briefly digest --interactive input/links.md                      # Interactive my-take workflow
briefly digest --single https://example.com/article             # Single article processing

# Cost estimation and debugging
briefly digest --dry-run input/links.md                         # Estimate API costs
briefly digest --list-formats                                   # Show available formats
```

**Advanced Research Commands:**
```bash
# Deep research with multi-stage pipeline
briefly research --topic "AI development trends" --depth 3      # 3-iteration research
briefly research --topic "cybersecurity" --max-sources 20      # Extended research

# Research integration with existing content
briefly research --integrate-with-digest digest-id             # Enhance existing digest
briefly research --list                                        # Show research sessions
```

**Content Management:**
```bash
# Cache operations with detailed stats
briefly cache stats                                            # Show cache statistics
briefly cache clear --confirm                                  # Clear all cached data

# Interactive terminal UI with relevance scoring
briefly tui                                                    # Browse articles with scoring

# Article processing and analysis
briefly summarize https://example.com/article                  # Single article summary
briefly categorize input/links.md                             # Topic categorization analysis
```

**Personal Commentary System:**
```bash
# My Take feature with AI regeneration
briefly my-take list                                           # Show available digests
briefly my-take add 1234abcd "Your perspective"               # Add personal commentary
briefly my-take regenerate 1234abcd                           # AI-powered full regeneration
```

## Architecture Overview

### Project Structure
- **`cmd/briefly/main.go`**: Main application entry point
- **`cmd/handlers/`**: CLI command definitions using Cobra framework (root.go, digest.go, etc.)
- **`internal/`**: Core application modules organized by domain with service-oriented architecture
- **`llmclient/`**: Legacy Gemini client (being phased out in favor of `internal/llm`)
- **`research/`**: Generated research reports and session data
- **`test/`**: Integration tests and mock implementations
- **`docs/`**: Requirements, configuration guides, and architectural documentation

### Core Architecture Patterns

**Modular Internal Packages**: The application is organized into focused internal packages:
- `core/`: Central data structures (Article, Summary, Link, Feed, Digest)
- `fetch/`: Web scraping and content extraction with intelligent HTML parsing (HTML, PDF, YouTube)
- `llm/`: LLM client abstraction layer for Gemini API interactions
- `store/`: SQLite-based caching system for articles, summaries, and analytics
- `templates/`: Digest format templates (brief, standard, detailed, newsletter, scannable, email)
- `render/`: Output generation and formatting
- `config/`: Hierarchical configuration management with Viper
- `services/`: Service layer interfaces and implementations for dependency injection
- `relevance/`: Unified relevance scoring architecture with multiple scorer implementations
- `categorization/`: Article categorization and topic detection
- `interactive/`: Interactive CLI workflows and user input handling
- `deepresearch/`: Multi-stage research pipeline (planner, fetcher, ranker, synthesizer)
- `email/`: HTML email template system with responsive design (v1.0)
- `messaging/`: Slack/Discord integration with webhook support (v1.0)
- `tts/`: Text-to-Speech audio generation with multiple providers (v1.0)
- `visual/`: AI banner image generation using DALL-E with content theme analysis (Sprint 3)

**AI-Powered Insights Pipeline**: Comprehensive analytics automatically integrated:
- `sentiment/`: Sentiment analysis with emoji indicators
- `alerts/`: Configurable alert monitoring and evaluation
- `trends/`: Historical trend analysis and comparison
- `research/`: Deep research with AI-generated queries
- `deepresearch/`: Advanced deep research pipeline with iterative query generation
- `clustering/`: Topic clustering and content organization
- **Smart Headlines**: AI-generated compelling titles based on digest content and format

**Caching Strategy**: Multi-layer SQLite caching system:
- Article content cached for 24 hours to avoid re-fetching
- Summaries cached for 7 days with content hash validation
- Digest metadata and analytics stored for trend analysis
- RSS feed items tracked for automatic content discovery

### Data Flow
1. **Input Processing**: URLs extracted from Markdown files
2. **Content Fetching**: Intelligent HTML parsing with cache-first strategy
3. **AI Processing**: LLM-powered summarization with format-specific prompts
4. **Insights Generation**: Parallel processing of sentiment, alerts, trends, and research
5. **Topic Clustering**: Automatic categorization using embedding-based clustering
6. **Smart Headline Generation**: AI-powered title creation based on digest content and format
7. **Template Rendering**: Format-specific output generation with insights integration

### Configuration Management
The application uses a sophisticated hierarchical configuration system with Viper:

**Configuration Precedence (highest to lowest):**
1. Command-line flags
2. Environment variables (loaded from `.env` file via `godotenv`)
3. Configuration file (`.briefly.yaml` or specified via `--config`)
4. Default values

**Key Configuration Patterns:**
- **Centralized Config Module**: `internal/config/config.go` provides unified access via `config.Get()`
- **Environment Variable Loading**: Automatic `.env` file loading for development
- **Nested Configuration**: Complex nested structures for AI, search providers, output formats
- **Provider-Specific Settings**: Separate config blocks for Gemini, OpenAI, Google Search, SerpAPI
- **Template-Driven Outputs**: Configuration drives template selection and rendering options

### LLM Integration
- Primary: Gemini API via `internal/llm` package
- Legacy: Direct client in `llmclient/` (being phased out)
- Functions: Article summarization, title generation, research query generation, sentiment analysis

### Service Architecture
The application uses a service-oriented architecture with dependency injection:

**Service Layer (`internal/services/`):**
- **Interface-Driven Design**: Clean interfaces for all major components (DigestService, ResearchService, etc.)
- **Dependency Injection**: Services are injected via constructors for testability
- **Mock Support**: Comprehensive mock implementations for testing (`test/mocks/services_mock.go`)

**Key Service Interfaces:**
- `DigestService`: Digest generation and processing
- `ArticleProcessor`: Content fetching and processing
- `ResearchService`: Deep research and content discovery
- `LLMService`: AI operations (summarization, embedding, sentiment analysis)
- `CacheService`: Caching operations with statistics
- `MessagingService`: Multi-channel output (Slack, Discord)
- `TTSService`: Text-to-speech generation

### Relevance Scoring Architecture (v2.0)
Unified scoring system shared across all commands:

**Core Interfaces (`internal/relevance/interfaces.go`):**
- `Scorer`: Main scoring interface with batch support
- `Scorable`: Interface for content that can be scored
- `Criteria`: Scoring parameters with weights and context

**Implementation Strategy:**
- `KeywordScorer`: Fast keyword-based scoring for digest filtering
- `ScoringWeights`: Configurable weights for different contexts (digest, research, TUI)
- **Context-Aware Profiles**: Different weight profiles optimize scoring for specific use cases

### Notable Design Decisions
- **Cache-First Architecture**: Aggressive caching to minimize API costs and improve performance
- **Format-Driven Templates**: Template system allows different digest styles while maintaining consistent structure
- **Insights-First Approach**: Every digest automatically includes comprehensive AI-powered analytics
- **Embedding-Based Clustering**: Uses LLM embeddings for intelligent topic grouping
- **Concurrent Processing**: Parallel processing of articles and insights for performance
- **Service-Oriented Design**: Clean separation of concerns with dependency injection
- **Interface-First Development**: All major components use interfaces for testability and extensibility

## Development Context

### API Requirements
- Gemini API key required (set via `GEMINI_API_KEY` environment variable)
- Optional: OpenAI API key for banner image generation (set via `OPENAI_API_KEY` environment variable)
- Optional: Google Custom Search API for research features
- Optional: SerpAPI for enhanced search capabilities

### Testing Strategy
The application includes comprehensive testing with 15 test files and 185+ test functions:

**Unit Tests:**
- Core data structures (`internal/core/core_test.go`)
- LLM operations (`internal/llm/llm_test.go`) 
- Search functionality (`internal/search/search_test.go`)
- Cost estimation (`internal/cost/estimation_test.go`)
- Template rendering (`internal/templates/templates_test.go`)
- Relevance scoring (`internal/relevance/keyword_scorer_test.go`)
- Content fetching (`internal/fetch/fetch_test.go`)
- Store operations (`internal/store/store_test.go`)
- Email generation (`internal/email/email_test.go`)
- Sentiment analysis (`internal/sentiment/sentiment_test.go`)
- Visual components (`internal/visual/banner_test.go`, `internal/visual/dalle_test.go`)
- Render functionality (`internal/render/render_test.go`)

**Integration Tests:**
- End-to-end digest generation (`test/integration/digest_test.go`)
- Multi-format output validation (`test/integration/multiformat_test.go`)
- Mock service implementations (`test/mocks/services_mock.go`)

**GitHub Actions CI/CD:**
- Go version 1.21 with module caching
- Parallel test execution with race detection and coverage reporting
- Automated linting with golangci-lint
- Binary build verification

### Linting and Code Quality
The project uses **golangci-lint** as the primary linting tool:
- Runs automatically in CI/CD pipeline on every push and PR
- Uses default golangci-lint configuration (no custom `.golangci.yml`)
- Includes comprehensive Go linting rules for code quality and consistency
- Recent commit history shows active linting compliance maintenance

Install golangci-lint locally for development:
```bash
# macOS
brew install golangci-lint

# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Or using go install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Error Handling
- Graceful degradation when articles fail to fetch or process
- Comprehensive logging via `internal/logger` with structured output
- Cache isolation prevents individual failures from affecting overall operation

### Performance Considerations
- SQLite caching reduces redundant API calls and improves response times
- Concurrent processing of articles and insights
- Intelligent content extraction focuses on main article text
- Cost estimation features help manage API usage

## Common Workflows

### Adding New Digest Formats
1. Define template in `internal/templates/templates.go`
2. Add format handling in `cmd/cmd/root.go` digest command
3. Update format validation in CLI
4. For email formats, add templates to `internal/email/email.go`
5. Add corresponding unit tests in `internal/templates/templates_test.go`

### Extending Insights Features
1. Create new analyzer in appropriate `internal/` package
2. Integrate into digest generation pipeline in `runDigest()`
3. Add CLI commands for standalone usage
4. Write unit tests for new functionality
5. Update core data structures in `internal/core/core.go` if needed

### Adding New Multi-Channel Output
1. Create new package in `internal/` (e.g., `internal/newchannel/`)
2. Implement converter functions for digest data
3. Add CLI commands in `cmd/handlers/digest.go`
4. Add configuration options to `internal/config/config.go`
5. Update service interfaces in `internal/services/interfaces.go`
6. Add corresponding unit tests

### Extending the Deep Research Pipeline
The deep research system (`internal/deepresearch/`) uses a multi-stage architecture:

**Pipeline Stages:**
1. **Planner** (`planner.go`): Decomposes topics into research sub-queries using LLM
2. **Search** (`search.go`): Executes queries using configured search providers
3. **Fetcher** (`fetcher.go`): Retrieves full content from discovered URLs
4. **Ranker** (`ranker.go`): Ranks sources by relevance using unified scoring system
5. **Synthesizer** (`synthesizer.go`): Generates comprehensive research briefs

**Extension Points:**
- Implement `Planner` interface for custom query generation strategies
- Add new search providers by implementing `SearchProvider` interface
- Extend `ContentFetcher` for new content types (beyond HTML, PDF, YouTube)
- Create custom ranking algorithms via `Ranker` interface

### Integrating with Relevance Scoring
All content processing should use the unified relevance system:

```go
// Example: Adding relevance scoring to a new feature
import "briefly/internal/relevance"

scorer := relevance.NewKeywordScorer()
criteria := relevance.Criteria{
    Query: "your topic",
    Context: "digest", // or "research", "tui"
    Threshold: 0.6,
}
score, err := scorer.Score(ctx, article, criteria)
```

**Key Integration Patterns:**
- Use `relevance.Criteria` with appropriate context for your use case
- Leverage existing scoring profiles from `profiles.go`
- Implement `relevance.Scorable` interface for new content types

### Configuring Banner Image Generation (Sprint 3)
1. Set OpenAI API key: `OPENAI_API_KEY` environment variable or `visual.openai.api_key` in config
2. Configure banner settings in `.briefly.yaml`:
   ```yaml
   visual:
     openai:
       api_key: "your-openai-api-key-here"
       model: "gpt-image-1"  # Latest image generation model
     banners:
       default_style: "tech"  # minimalist, tech, professional
       width: 1536           # Landscape format (3:2 ratio)
       height: 1024          # Supported sizes: 1024x1024, 1024x1536, 1536x1024
   ```
3. Use `--with-banner` flag: `briefly digest --with-banner --format newsletter input/links.md`
4. Banner images are automatically included in newsletter and email formats when enabled
5. Images are generated using the latest OpenAI image generation API with base64 encoding
6. Supported image sizes: 1024x1024 (square), 1536x1024 (landscape), 1024x1536 (portrait)

### Cache Management
- Cache is stored in `.briefly-cache/` directory (SQLite database)
- Use `briefly cache stats` to monitor usage
- Clear cache when debugging content extraction issues

### Running Single Tests
```bash
# Run tests for a specific package
go test ./internal/core

# Run a specific test function
go test ./internal/core -run TestLinkCreation

# Run tests with verbose output for debugging
go test -v ./internal/templates -run TestTemplateRendering

# Run integration tests
go test ./test/integration/...

# Run tests with race detection (same as CI)
go test -race ./internal/relevance
```

### Development Patterns Specific to Briefly

**Service Construction Pattern:**
```go
// Services are constructed with dependency injection
digestService := services.NewDigestService(
    llmService,
    templateService,
    cacheService,
)
```

**Configuration Access Pattern:**
```go
// Access configuration through centralized module
cfg := config.Get()
apiKey := cfg.AI.Gemini.APIKey
searchProvider := cfg.Search.DefaultProvider
```

**Error Handling Pattern:**
```go
// Graceful degradation with logging
if err := processArticle(article); err != nil {
    logger.Warnf("Failed to process article %s: %v", article.URL, err)
    // Continue processing other articles
}
```

**Context Propagation Pattern:**
```go
// All service methods accept context for cancellation
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
digest, err := digestService.GenerateDigest(ctx, urls, "newsletter")
```

### Linting Commands
```bash
# Install golangci-lint if not already installed
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin

# Run full linting suite (same as CI)
golangci-lint run --timeout=5m

# If golangci-lint is not in PATH, use full path
$(go env GOPATH)/bin/golangci-lint run --timeout=5m

# Run linting on specific directory
golangci-lint run ./internal/core

# Run basic Go tools
go fmt ./...
go vet ./...

# Auto-fix formatting issues
gofmt -w .
```

## v1.0 Multi-Channel Architecture

### HTML Email System
- **Templates**: Responsive HTML templates in `internal/email/`
- **Styles**: Inline CSS for email client compatibility
- **Formats**: Default, newsletter, and minimal styles
- **Usage**: `briefly digest --format email` or `templates.RenderHTMLEmail()`

### Messaging Integration
- **Platforms**: Slack and Discord webhook support
- **Formats**: Bullets, summary, and highlights formats
- **Features**: Sentiment emojis, article limits, rich formatting
- **Usage**: `briefly send-slack` and `briefly send-discord` commands

### TTS Audio Generation
- **Providers**: OpenAI TTS, ElevenLabs, Google Cloud TTS, and mock
- **Features**: Voice selection, speed control, article limits
- **Processing**: Markdown cleanup, speech-friendly formatting
- **Usage**: `briefly generate-tts` with provider-specific configuration