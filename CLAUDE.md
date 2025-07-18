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
```bash
# Generate a digest with Smart Headlines and different formats
briefly digest --format newsletter --output digests input/links.md
briefly digest --format email --output digests input/links.md  # v1.0: HTML email

# Generate digests with AI banner images (Sprint 3: AI-Generated Banner Images)
briefly digest --with-banner --format newsletter input/links.md
briefly digest --with-banner --format email input/links.md

# Smart Headlines are automatically generated for all digests based on content

# Cost estimation (dry run)
briefly digest --dry-run input/links.md

# Cache management
briefly cache stats
briefly cache clear --confirm

# Terminal UI for browsing articles
briefly tui

# Quick article summarization
briefly summarize https://example.com/article

# My Take feature - add personal commentary
briefly my-take add 1234abcd "Your personal take"
briefly my-take regenerate 1234abcd

# v1.0 Multi-Channel Features
# HTML Email generation
briefly digest --format email input/links.md

# Slack/Discord integration
briefly send-slack input/links.md --webhook https://hooks.slack.com/...
briefly send-discord input/links.md --webhook https://discord.com/api/webhooks/...

# TTS audio generation
briefly generate-tts input/links.md --provider openai --voice alloy
briefly generate-tts input/links.md --provider elevenlabs --voice Rachel
```

## Architecture Overview

### Project Structure
- **`cmd/briefly/main.go`**: Main application entry point
- **`cmd/cmd/root.go`**: CLI command definitions using Cobra framework
- **`internal/`**: Core application modules organized by domain
- **`llmclient/`**: Legacy Gemini client (being phased out in favor of `internal/llm`)
- **`research/`**: Empty directory for future research data storage

### Core Architecture Patterns

**Modular Internal Packages**: The application is organized into focused internal packages:
- `core/`: Central data structures (Article, Summary, Link, Feed, Digest)
- `fetch/`: Web scraping and content extraction with intelligent HTML parsing
- `llm/`: LLM client abstraction layer for Gemini API interactions
- `store/`: SQLite-based caching system for articles, summaries, and analytics
- `templates/`: Digest format templates (brief, standard, detailed, newsletter, email)
- `render/`: Output generation and formatting
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
The application uses a hierarchical configuration system with Viper:
1. Default values
2. Configuration file (`.briefly.yaml`)
3. Environment variables (especially `GEMINI_API_KEY`)
4. Command-line flags

### LLM Integration
- Primary: Gemini API via `internal/llm` package
- Legacy: Direct client in `llmclient/` (being phased out)
- Functions: Article summarization, title generation, research query generation, sentiment analysis

### Notable Design Decisions
- **Cache-First Architecture**: Aggressive caching to minimize API costs and improve performance
- **Format-Driven Templates**: Template system allows different digest styles while maintaining consistent structure
- **Insights-First Approach**: Every digest automatically includes comprehensive AI-powered analytics
- **Embedding-Based Clustering**: Uses LLM embeddings for intelligent topic grouping
- **Concurrent Processing**: Parallel processing of articles and insights for performance

## Development Context

### API Requirements
- Gemini API key required (set via `GEMINI_API_KEY` environment variable)
- Optional: OpenAI API key for banner image generation (set via `OPENAI_API_KEY` environment variable)
- Optional: Google Custom Search API for research features
- Optional: SerpAPI for enhanced search capabilities

### Testing Strategy
The application includes unit tests for core functionality. Current test coverage includes:
- Core data structures (`internal/core/core_test.go`)
- Search functionality (`internal/search/search_test.go`)
- Cost estimation (`internal/cost/estimation_test.go`)
- Template rendering (`internal/templates/templates_test.go`)

Use GitHub Actions CI/CD pipeline for automated testing with race detection and coverage reporting.

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
3. Add CLI commands in `cmd/cmd/root.go`
4. Add configuration options and validation

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