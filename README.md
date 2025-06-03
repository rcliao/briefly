# Briefly: AI-Powered Digest Generator

Briefly is a modern command-line application written in Go that takes a Markdown file containing a list of URLs, fetches the content from each URL, summarizes the text using a Large Language Model (LLM) via the Gemini API, and then generates a cohesive Markdown-formatted digest of all the summarized content.

## Features

- **Smart Content Processing**: Reads URLs from Markdown files and intelligently extracts main article content
- **AI-Powered Summarization**: Uses Gemini API to generate concise, meaningful summaries
- **Multiple Digest Formats**: Choose from brief, standard, detailed, or newsletter formats
- **Prompt Corner**: Newsletter format includes AI-generated prompts based on digest content that readers can copy and use with any LLM (ChatGPT, Gemini, Claude, etc.)
- **Personal Commentary**: Add your own "My Take" to any digest with AI-powered regeneration that integrates your voice throughout the entire content
- **Intelligent Caching**: SQLite-based caching system to avoid re-processing articles and summaries
- **Cost Estimation**: Dry-run mode to estimate API costs before processing
- **Template System**: Customizable output formats with built-in templates
- **Terminal UI**: Interactive TUI for browsing articles and summaries
- **Modern CLI**: Built with Cobra for intuitive command-line experience
- **Structured Logging**: Comprehensive logging with multiple output formats
- **Configuration Management**: Flexible configuration via files, environment variables, or flags

## Prerequisites

- Go (version 1.23 or higher recommended)
- A Gemini API Key

## Installation

### From Source

1. **Clone the Repository:**

   ```bash
   git clone https://github.com/rcliao/briefly.git
   cd briefly
   ```

2. **Install Dependencies:**

   ```bash
   go mod tidy
   ```

3. **Build the Application:**

   ```bash
   # Build for current platform
   go build -o briefly ./cmd/briefly
   
   # Or build and install to $GOPATH/bin
   go install ./cmd/briefly
   ```

### Pre-built Binaries

Check the [Releases](https://github.com/rcliao/briefly/releases) page for pre-built binaries for your platform.

## Configuration

### API Key Setup

You can provide your Gemini API key in several ways:

1. **Environment Variable (Recommended):**
   ```bash
   export GEMINI_API_KEY="your_api_key_here"
   ```

2. **`.env` File:**
   Create a `.env` file in the project root:
   ```
   GEMINI_API_KEY=your_api_key_here
   ```

3. **Configuration File:**
   Create a `.briefly.yaml` file in your home directory or current directory:
   ```yaml
   gemini:
     api_key: "your_api_key_here"
     model: "gemini-1.5-flash-latest"
   output:
     directory: "digests"
   ```

4. **Command-line Flag:**
   Use the `--config` flag to specify an API key in a config file.

### Configuration Precedence

Configuration is loaded in the following order (later sources override earlier ones):
1. Default values
2. Configuration file (`.briefly.yaml`)
3. Environment variables
4. Command-line flags

## Usage

Briefly uses a modern CLI interface with subcommands. Here are the main commands:

### Generate a Digest

```bash
# Basic usage
briefly digest input/my-links.md

# Specify output directory and format
briefly digest --output ./my-digests --format newsletter input/my-links.md

# Estimate costs before processing (dry run)
briefly digest --dry-run input/my-links.md

# Use custom configuration file
briefly --config ~/.my-config.yaml digest input/my-links.md
```

### Available Digest Formats

Use the `--format` flag to specify the output style:

- `brief`: Concise digest with key highlights only
- `standard`: Balanced digest with summaries and key points (default)
- `detailed`: Comprehensive digest with full summaries and analysis
- `newsletter`: Newsletter-style digest optimized for sharing, includes "Prompt Corner" with AI-generated prompts readers can copy and use

```bash
# List all available formats
briefly formats
```

### Cache Management

Briefly includes intelligent caching to avoid re-processing articles:

```bash
# View cache statistics
briefly cache stats

# Clear all cached data
briefly cache clear --confirm
```

### My Take Feature

Transform any generated digest into a personalized version that reflects your voice and perspective throughout the entire content using AI-powered regeneration:

```bash
# List all digests and their my-take status
briefly my-take list

# Add your take to a digest (interactive mode)
briefly my-take add 1234abcd

# Add your take directly from command line
briefly my-take add 1234abcd "This digest highlights important trends in AI development that I think will impact our industry significantly."

# Update an existing take
briefly my-take add 1234abcd "Updated thoughts: The AI developments are even more significant than I initially thought."

# Regenerate digest with your perspective woven throughout
briefly my-take regenerate 1234abcd
```

**My Take Features:**
- **AI-Powered Regeneration**: Uses Gemini LLM to completely rewrite digests with your personal voice integrated naturally throughout
- **Seamless Integration**: Your perspective becomes part of the narrative flow, not just an appended section
- **Partial ID Matching**: Use just the first few characters of a digest ID (e.g., `1234` instead of the full UUID)
- **Multiple Input Methods**: Add takes interactively or via command-line arguments
- **Update Support**: Easily modify existing takes and regenerate with new perspectives
- **Timestamped Output**: Creates new files with `_with_my_take_` naming convention to preserve originals
- **Format Preservation**: Maintains the original digest format while incorporating your voice

**Example Transformation:**

*Original digest excerpt:*
```markdown
# Daily Digest - 2025-05-30

Here's what's worth knowing from today's articles:

## Executive Summary
The example domain (https://example.com) is freely available for illustrative use...
```

*Your take: "This brief format is really convenient for quick updates"*

*Regenerated digest:*
```markdown
# Brief Digest - 2025-05-30

Quick highlights from today's reading – I find this brief format really convenient for staying up-to-date without getting bogged down!

## Executive Summary
This week's highlight is a bit meta, but honestly, a real time-saver: I discovered that the domain example.com is available for illustrative purposes...
```

### Terminal User Interface

Launch an interactive TUI to browse articles and summaries:

```bash
briefly tui
```

### Prompt Corner Feature

The newsletter format includes a special "Prompt Corner" section that automatically generates interesting prompts based on the digest content. These prompts are designed to be copied and pasted into any LLM (ChatGPT, Gemini, Claude, etc.) for further exploration of the topics covered.

**Example Prompt Corner Output:**
```markdown
## 🎯 Prompt Corner

Here are some prompts inspired by today's digest:

```
"Act as a senior software engineer. I'm trying to refactor a legacy section of Python code. Using the capabilities of a hypothetical 'Claude Opus 4' coding model with access to the filesystem and web search, propose a refactoring plan, including justifications and potential risks."
```
This prompt simulates using advanced AI coding features for real-world refactoring problems.

```
"I have a list of small bug fixes for a Node.js application. As GitHub Copilot Coding Agent, suggest a prioritized order for these tasks, outlining the approach and estimated time for each."
```
This prompt leverages AI task delegation capabilities for project management.
```

The prompts are:
- **Contextual**: Directly inspired by the articles in your digest
- **Practical**: Ready to use for real development scenarios  
- **Portable**: Work with any LLM platform
- **Educational**: Include explanations of what each prompt accomplishes

### Command-line Options

**Global Flags:**
- `--config`: Specify a configuration file

**Digest Command Flags:**
- `--output, -o`: Output directory for digest files (default: "digests")
- `--format, -f`: Digest format: brief, standard, detailed, newsletter (default: "standard")
- `--dry-run`: Estimate costs without making API calls

### Examples

```bash
# Basic digest generation
briefly digest input/weekly-links.md

```bash
# Newsletter format with custom output directory
briefly digest --format newsletter --output ./newsletters input/links.md

# Cost estimation before processing
briefly digest --dry-run input/expensive-links.md

# Using environment variable for API key
export GEMINI_API_KEY="your_key_here"
briefly digest input/links.md

# Complete workflow with AI-powered personal commentary
briefly digest input/weekly-links.md                    # Generate digest
briefly my-take list                                     # See available digests  
briefly my-take add 1234abcd "Great insights this week!" # Add your perspective
briefly my-take regenerate 1234abcd                     # AI regenerates entire digest with your voice integrated throughout
```

## Input File Format

Input files should be Markdown files containing URLs. Briefly will extract all HTTP/HTTPS URLs found anywhere in the file.

### Example Input File

```markdown
---
date: 2025-05-30
title: "Weekly Tech Links"
---

# Interesting Articles This Week

Here are some articles I found interesting:

- https://example.com/article-1
- https://news.site.com/important-update
- Check this out: https://blog.example.org/research-paper

## AI and Development

- [Claude 4 Release](https://anthropic.com/news/claude-4)
- https://zed.dev/blog/fastest-ai-code-editor

Some inline links like https://github.com/project/repo are also extracted.
```

The application will automatically extract all URLs regardless of their formatting (plain text, markdown links, inline, etc.).

## How It Works

### Digest Generation

1. **URL Extraction**: Parses the input Markdown file to find all HTTP/HTTPS URLs
2. **Content Fetching**: Downloads and extracts main content from each URL using intelligent HTML parsing
3. **Smart Caching**: Checks cache for previously processed articles to avoid redundant API calls
4. **Content Cleaning**: Removes boilerplate content (navigation, ads, etc.) to focus on main article text
5. **AI Summarization**: Uses Gemini API to generate concise summaries of each article
6. **Template Processing**: Applies the selected format template to structure the output
7. **Final Digest Generation**: Creates a cohesive digest with proper citations and formatting
8. **Output**: Saves the final digest as a Markdown file and displays cache statistics

### My Take Regeneration

1. **Personal Perspective Storage**: Your "my take" is stored in the local database linked to the specific digest
2. **Content Retrieval**: System retrieves the original digest content and your personal take
3. **AI-Powered Rewriting**: Gemini LLM receives sophisticated prompts to completely rewrite the digest incorporating your voice naturally throughout
4. **Cohesive Integration**: Your perspective becomes part of the narrative flow rather than a separate section
5. **Timestamped Output**: Creates a new file with `_with_my_take_` suffix while preserving the original

### Intelligent Features

- **Caching**: Articles and summaries are cached to avoid re-processing
- **Content Extraction**: Advanced HTML parsing focuses on main article content
- **Cost Estimation**: Dry-run mode provides cost estimates before processing
- **Error Handling**: Graceful handling of failed URLs with detailed logging
- **Multiple Formats**: Choose from different digest styles for various use cases

## Advanced Usage

### Configuration Management

Create a `.briefly.yaml` configuration file for persistent settings:

```yaml
# Gemini AI Configuration
gemini:
  api_key: ""  # Or use GEMINI_API_KEY environment variable
  model: "gemini-1.5-flash-latest"

# Output Configuration
output:
  directory: "digests"

# Future configuration options can be added here
# cache:
#   enabled: true
#   ttl: "24h"
```

### Development and Testing

```bash
# Run from source during development
go run ./cmd/briefly digest input/test-links.md

# Run tests
go test ./...

# Build for multiple platforms
GOOS=linux GOARCH=amd64 go build -o briefly-linux-amd64 ./cmd/briefly
GOOS=windows GOARCH=amd64 go build -o briefly-windows-amd64.exe ./cmd/briefly
GOOS=darwin GOARCH=amd64 go build -o briefly-darwin-amd64 ./cmd/briefly
```

### API Cost Management

Briefly includes built-in cost estimation to help manage Gemini API usage:

```bash
# Estimate costs before processing
briefly digest --dry-run input/large-link-list.md

# Example output:
# Cost Estimation for Digest Generation
# =====================================
# Articles to process: 25
# Estimated tokens per article: ~2000
# Total estimated input tokens: ~50,000
# Estimated output tokens: ~5,000
# 
# Estimated costs (USD):
# - Input tokens: $0.025
# - Output tokens: $0.015
# - Total estimated cost: $0.040
```

### Troubleshooting

**Common Issues:**

1. **API Key not found**: Ensure `GEMINI_API_KEY` is set or configured in `.briefly.yaml`
2. **Permission denied**: Make sure the output directory is writable
3. **Network timeouts**: Some websites may be slow or block requests
4. **Cache issues**: Clear cache with `briefly cache clear --confirm`

**Debug Logging:**

The application provides detailed logging. Check logs for specific error messages when articles fail to process.

## Project Structure

```
briefly/
├── cmd/
│   ├── briefly/              # Main application entry point
│   │   └── main.go
│   ├── cmd/                  # CLI commands and configuration
│   │   └── root.go          # Cobra CLI setup and command definitions
│   └── main.go              # Alternative entry point
├── internal/                # Internal packages
│   ├── core/                # Core data structures (Article, Summary, etc.)
│   ├── cost/                # Cost estimation functionality
│   ├── fetch/               # URL fetching and content extraction
│   ├── llm/                 # LLM client abstraction
│   ├── logger/              # Structured logging setup
│   ├── render/              # Digest rendering and output
│   ├── store/               # SQLite caching system
│   ├── templates/           # Digest format templates
│   └── tui/                 # Terminal user interface
├── llmclient/               # Legacy Gemini client (being phased out)
│   └── gemini_client.go
├── input/                   # Example input files
├── digests/                 # Generated digest outputs
├── temp_content/            # Cached article content
├── docs/                    # Documentation
├── .env                     # Environment variables (local)
├── .briefly.yaml           # Configuration file
├── go.mod                   # Go module definition
├── go.sum                   # Dependency checksums
└── README.md               # This file
```

### Key Components

- **`cmd/briefly/main.go`**: Application entry point
- **`cmd/cmd/root.go`**: CLI command definitions and routing
- **`internal/core/`**: Core data structures and business logic
- **`internal/fetch/`**: Web scraping and content extraction
- **`internal/llm/`**: AI/LLM integration layer
- **`internal/store/`**: SQLite-based caching system
- **`internal/templates/`**: Output format templates
- **`internal/tui/`**: Interactive terminal interface

## Further Development

See [`docs/plan/execution_plan_v0.md`](docs/plan/execution_plan_v0.md) for the complete development roadmap and current implementation status.

**Current Status**: Most core features are implemented and production-ready. The primary remaining work involves completing the Prompt Corner database and enhanced TUI features.

**Next Priority**: Complete the v0.2 "Human Voice" milestone by implementing:
- Prompt Corner database with ratings and usage tracking
- Enhanced TUI for prompt management and rating
- Clipboard integration for easy prompt copying

- **Phase 5: Testing and Documentation**
  - Write unit tests for key functions.
  - Write integration tests for the end-to-end workflow.
  - Continuously update documentation.
