# Briefly - System Architecture

**Version:** 2.1
**Date:** 2025-10-31
**Last Updated:** 2025-10-31

This document describes the high-level architecture and technical design of Briefly. This should be the living document that is always up-to-date with the current implementation.

---

## Table of Contents

1. [Current State Analysis](#current-state-analysis)
2. [Architecture Overview](#architecture-overview)
3. [Component Design](#component-design)
4. [Data Model](#data-model)
5. [API Design](#api-design)
6. [Multi-Agent Architecture](#multi-agent-architecture)
7. [LangFuse Observability](#langfuse-observability)
8. [PostHog Analytics](#posthog-analytics)
9. [RAG Implementation](#rag-implementation)
10. [Evaluation Framework](#evaluation-framework)
11. [Technical Decisions](#technical-decisions)
12. [Deployment Architecture](#deployment-architecture)
13. [Appendix: Embedding Explanation](#appendix-embedding-explanation)

---

## Current State Analysis

*(Same as v1.0 - strong foundations exist)*

### What Exists (v3.0 Simplified Architecture)

#### ✅ Strong Foundations

**Pipeline Architecture:**
- 9-step processing pipeline (`internal/pipeline/`)
- Clean interface-driven design
- Adapter pattern for legacy compatibility
- Builder pattern for dependency injection

**Content Processing:**
- Multi-format fetching (HTML, PDF, YouTube) - `internal/fetch/`
- LLM-powered summarization - `internal/summarize/`
- K-means clustering - `internal/clustering/`
- Embedding generation (768-dim Gemini) - `internal/llm/`
- Category classification - NEW in v3.0

**Data Persistence:**
- **SQLite:** CLI caching layer (articles, summaries, digests)
- **PostgreSQL:** Production database with repository pattern
- Clean interfaces in `internal/persistence/`
- Migration system (`migrate.go`)

**Web Server:**
- Chi router with middleware - `internal/server/`
- API endpoint skeletons defined
- Graceful shutdown handling
- Server command handler (`cmd/handlers/serve.go`)

**RSS Infrastructure:**
- RSS/Atom parsing - `internal/feeds/`
- Conditional GET support (ETag, Last-Modified)
- Feed storage in both SQLite and PostgreSQL
- Aggregation command - `cmd/handlers/aggregate.go`

#### ⚠️ Partially Complete

**Narrative Generation:**
- Executive summary generation exists but unstable
- Non-fatal failures in production

**Article Ordering:**
- Completely stubbed (returns clusters unchanged)
- Interface defined but not implemented

**Web Handlers:**
- Endpoint routes defined in `internal/server/`
- Handler implementations incomplete

#### ❌ Missing Components

**Search Integration:**
- No web search capabilities
- No search API clients
- No search query management

**Theme System:**
- No theme/category filtering
- No admin theme configuration
- No relevance scoring by theme

**Manual URL Submission:**
- No CLI command to add URLs
- No web form for URL submission
- No manual URL tracking table

**Multi-Agent System:**
- Single-threaded processing only
- No agent orchestration framework
- No task delegation system

**LLM Observability:**
- Basic logging exists
- No LangFuse integration
- No cost/token tracking dashboard
- No prompt performance metrics

**Product Analytics:**
- No usage tracking
- No user behavior analysis
- No PostHog integration

**Evaluation Framework:**
- No test datasets
- No evaluation criteria
- No LLM-as-judge system
- No prompt comparison tools

**RAG System:**
- Embeddings generated but not indexed
- No vector search (pgvector not set up)
- No similarity retrieval
- No context augmentation

**Structured Article Sections:**
- Basic summaries exist
- No Key Moments, Perspectives, Why It Matters
- No quotes extraction
- No timeline/context generation

**Automation:**
- No background scheduler
- No fixed-schedule publishing
- Manual digest generation only

**Web UI:**
- No HTML templates
- No frontend assets
- No static file serving
- No user-facing interface

**Admin System:**
- No authentication
- No admin dashboard
- No feed management UI
- No theme configuration UI

---

## Architecture Overview

### High-Level System Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                      ADMIN INTERFACE                             │
│  • Configure RSS feeds, search terms, themes                    │
│  • Manually submit URLs (CLI or web form)                       │
│  • View LangFuse observability dashboard                        │
│  • View PostHog analytics dashboard                             │
│  • Run eval CLI commands for prompt testing                     │
└────────────────────────┬────────────────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────────────────┐
│               WEEKLY SCHEDULED AGGREGATION                       │
│  (Every Monday 6:00 AM - can run manually via CLI)             │
│                                                                  │
│  Step 1: CONTENT AGGREGATION                                    │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Multi-Agent Content Aggregation                          │  │
│  │  ┌─────────────────────────────────────┐                 │  │
│  │  │  Research Lead (Manager Agent)      │                 │  │
│  │  │  - Plans aggregation strategy       │                 │  │
│  │  │  - Delegates tasks to workers       │                 │  │
│  │  │  - Aggregates results               │                 │  │
│  │  └───────────┬─────────────────────────┘                 │  │
│  │              │                                            │  │
│  │     ┌────────┴────────┬─────────────┬─────────────┐     │  │
│  │     ▼                 ▼             ▼             ▼     │  │
│  │  [RSS Worker]   [Search Worker] [Manual URLs] [Fetch]  │  │
│  │                                                           │  │
│  │  Sources:                                                │  │
│  │  • RSS Feeds (community-curated)                        │  │
│  │  • Web Search (Google/Bing)                             │  │
│  │  • Manual URLs (admin-submitted)                        │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Step 2: THEME CLASSIFICATION & FILTERING                       │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  LLM-Based Theme Classification                           │  │
│  │                                                            │  │
│  │  For each article:                                        │  │
│  │  1. Extract themes: GenAI, Gaming, Technology, etc.      │  │
│  │  2. Calculate relevance score per theme (0-1)            │  │
│  │  3. Filter low-relevance articles (configurable)         │  │
│  │                                                            │  │
│  │  LangFuse Tracking: Token usage, latency, classification │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Step 3: STRUCTURED ARTICLE PROCESSING                          │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  RAG-Enhanced Article Processing                          │  │
│  │                                                            │  │
│  │  For each article:                                        │  │
│  │  1. Fetch content (HTML/PDF/YouTube)                     │  │
│  │  2. Generate embedding (768-dim vector)                  │  │
│  │  3. RAG: Retrieve similar articles for context           │  │
│  │  4. Generate structured summary:                         │  │
│  │     • Summary (overview)                                 │  │
│  │     • Key Moments (highlights)                           │  │
│  │     • Perspectives (multiple viewpoints)                 │  │
│  │     • Why It Matters (significance)                      │  │
│  │     • Context (background)                               │  │
│  │  5. Store in pgvector                                    │  │
│  │                                                            │  │
│  │  LangFuse Tracking: Summarization cost, quality          │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Step 4: CLUSTERING & ORGANIZATION                              │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Topic Clustering (K-means on embeddings)                │  │
│  │                                                            │  │
│  │  1. Group articles by theme                              │  │
│  │  2. K-means clustering within each theme                 │  │
│  │  3. RAG-enhanced cluster labeling                        │  │
│  │  4. Article ordering within clusters (by date/priority)  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                  │
│  Step 5: DIGEST GENERATION                                      │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Executive Summary & Rendering                            │  │
│  │                                                            │  │
│  │  1. Select top articles per cluster                      │  │
│  │  2. Generate executive summary per theme                 │  │
│  │  3. Render markdown + HTML digest                        │  │
│  │  4. Publish to database (make "latest")                  │  │
│  │  5. Optional: Send email notifications                   │  │
│  │                                                            │  │
│  │  LangFuse Tracking: Summary generation cost              │  │
│  └──────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                   PUBLIC WEB INTERFACE                           │
│                                                                   │
│  • Homepage: Latest digest (theme-filtered)                     │
│  • Digest viewer: Structured sections with citations            │
│  • Article browser: Search, filter by theme/date                │
│  • Past digests: Historical archive                             │
│  • PostHog Analytics: Track user engagement                     │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                 CROSS-CUTTING CONCERNS                           │
│                                                                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │  LangFuse    │  │  PostHog     │  │  RAG Vector Store    │  │
│  │              │  │              │  │                      │  │
│  │  • LLM traces│  │  • Page views│  │  • pgvector index    │  │
│  │  • Token/cost│  │  • User flow │  │  • Similarity search │  │
│  │  • Latency   │  │  • Engagement│  │  • Context retrieval │  │
│  │  • Prompts   │  │  • Themes    │  │  • 768-dim vectors   │  │
│  └──────────────┘  └──────────────┘  └──────────────────────┘  │
│                                                                   │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  CLI Evaluation Framework                                 │  │
│  │  • Golden datasets (100+ examples per task)              │  │
│  │  • LLM-as-judge evaluation                               │  │
│  │  • Prompt comparison                                     │  │
│  │  • Not exposed via web API (CLI only)                    │  │
│  └──────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

### Technology Stack

**Backend:**
- **Language:** Go 1.21+
- **Web Framework:** Chi router
- **Database:** PostgreSQL 15+ with extensions:
  - `pgvector` - Vector similarity search (IVFFlat or HNSW indexes)
  - `pg_trgm` - Full-text search
- **LLM API:** Google Gemini (existing)
- **Scheduler:** Manual CLI commands + optional `robfig/cron` v3 OR infrastructure CRON

**Frontend:**
- **Templating:** Go `html/template`
- **Interactivity:** HTMX (minimal JavaScript)
- **Styling:** TailwindCSS (CDN initially)
- **No custom dashboards:** External tools (LangFuse, PostHog)

**Observability & Analytics:**
- **LangFuse:** LLM observability (replaces custom observability)
- **PostHog:** Product analytics (frontend + backend events)

**Infrastructure:**
- **Containerization:** Docker
- **Deployment:** Railway / Fly.io / VPS
- **Database Hosting:** Supabase / Neon (managed PostgreSQL with pgvector)

**Development:**
- **Linting:** golangci-lint
- **Testing:** Go standard testing + table-driven tests
- **Migrations:** Custom migration system (existing)

---

## Component Design

### 1. Theme System (`internal/themes/`)

#### Purpose

Admin-configured categories (GenAI, Gaming, Technology) for filtering and organizing content.

#### Data Model

```sql
CREATE TABLE themes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,  -- 'GenAI', 'Gaming', 'Technology'
    description TEXT,
    keywords TEXT[],  -- Hints for LLM classification
    active BOOLEAN DEFAULT true,
    priority INT DEFAULT 0,  -- Display order
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Default themes
INSERT INTO themes (name, description, keywords) VALUES
('GenAI', 'Generative AI and LLM developments', ARRAY['AI', 'LLM', 'GPT', 'Claude', 'Gemini', 'diffusion', 'transformer']),
('Gaming', 'Video games and gaming industry', ARRAY['game', 'gaming', 'console', 'Steam', 'PlayStation', 'Xbox']),
('Technology', 'General tech news', ARRAY['tech', 'software', 'hardware', 'startup', 'developer']);
```

#### Interface

```go
// ThemeClassifier determines article themes
type ThemeClassifier interface {
    // ClassifyArticle assigns themes and relevance scores
    ClassifyArticle(ctx context.Context, article core.Article) ([]ThemeScore, error)

    // FilterByRelevance removes low-relevance articles
    FilterByRelevance(articles []core.Article, threshold float64) ([]core.Article, error)
}

type ThemeScore struct {
    ThemeID    string
    ThemeName  string
    Relevance  float64  // 0.0 - 1.0
    Reasoning  string   // LLM explanation
}
```

#### Implementation

```go
type LLMThemeClassifier struct {
    llmClient   *llm.Client
    themeRepo   persistence.ThemeRepository
    langfuse    *langfuse.Client
}

// ClassifyArticle uses LLM with theme keywords as context
func (c *LLMThemeClassifier) ClassifyArticle(ctx context.Context, article core.Article) ([]ThemeScore, error) {
    themes, _ := c.themeRepo.List(ctx, persistence.ListOptions{Active: true})

    prompt := buildClassificationPrompt(article, themes)

    // LangFuse trace
    trace := c.langfuse.StartTrace("theme_classification", map[string]interface{}{
        "article_id": article.ID,
        "article_url": article.URL,
    })
    defer trace.End()

    response, err := c.llmClient.GenerateText(ctx, prompt, llm.Options{
        Temperature: 0.0,  // Deterministic classification
    })

    // Parse JSON response: [{"theme": "GenAI", "relevance": 0.95, "reasoning": "..."}]
    scores := parseThemeScores(response)

    trace.SetMetadata("themes_found", len(scores))
    trace.SetMetadata("top_theme", scores[0].ThemeName)

    return scores, nil
}
```

#### CLI Commands

```bash
# Theme management
briefly theme list                           # List all themes
briefly theme add "Web3" --keywords "blockchain,crypto,NFT"
briefly theme edit "GenAI" --description "..."
briefly theme disable "Gaming"

# Test classification
briefly classify-article https://example.com/article
# Output:
# Theme: GenAI (relevance: 0.95)
# Theme: Technology (relevance: 0.65)
```

---

### 2. Manual URL Submission (`internal/sources/manual.go`)

#### Purpose

Allow admin to manually submit URLs for processing outside of RSS/search.

#### Data Model

```sql
CREATE TABLE manual_urls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url TEXT NOT NULL UNIQUE,
    title TEXT,  -- Optional user-provided title
    notes TEXT,  -- Admin notes about why this was added
    theme_hint TEXT,  -- Suggested theme
    submitted_by TEXT DEFAULT 'admin',
    processed BOOLEAN DEFAULT false,
    article_id UUID REFERENCES articles(id),  -- Linked after processing
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_manual_urls_processed ON manual_urls(processed);
```

#### CLI Commands

```bash
# Add single URL
briefly add-url https://example.com/article \
  --title "Important Article" \
  --theme "GenAI" \
  --notes "Follow-up to last week's coverage"

# Add multiple URLs from file
briefly add-urls urls.txt

# urls.txt format:
# https://example.com/1
# https://example.com/2 | Custom Title | GenAI | These are related

# List unprocessed manual URLs
briefly list-manual-urls --unprocessed

# Process manual URLs (fetch + summarize)
briefly process-manual-urls
```

#### Web Admin Form

```html
<!-- Admin UI: /admin/manual-urls -->
<form method="POST" action="/api/admin/manual-urls">
  <input type="url" name="url" required placeholder="https://..." />
  <input type="text" name="title" placeholder="Optional title" />
  <select name="theme_hint">
    <option value="">Auto-detect</option>
    <option value="GenAI">GenAI</option>
    <option value="Gaming">Gaming</option>
    <option value="Technology">Technology</option>
  </select>
  <textarea name="notes" placeholder="Why this article?"></textarea>
  <button type="submit">Add URL</button>
</form>

<!-- Batch upload -->
<form method="POST" action="/api/admin/manual-urls/batch" enctype="multipart/form-data">
  <input type="file" name="file" accept=".txt,.csv" />
  <button type="submit">Upload URLs</button>
</form>
```

---

### 3. Structured Article Sections (`internal/summarize/structured.go`)

#### Purpose

Generate Kagi-inspired structured summaries with multiple sections.

#### Data Model

```sql
-- Enhanced summaries table
ALTER TABLE summaries ADD COLUMN key_moments TEXT;       -- Bullet points (includes quotes)
ALTER TABLE summaries ADD COLUMN perspectives TEXT[];    -- Multiple viewpoints
ALTER TABLE summaries ADD COLUMN why_important TEXT;     -- Significance explanation
ALTER TABLE summaries ADD COLUMN context TEXT;           -- Background info
ALTER TABLE summaries ADD COLUMN timeline JSONB;         -- Chronological events

-- Example timeline JSONB:
-- [
--   {"date": "2025-10-15", "event": "Initial announcement"},
--   {"date": "2025-10-20", "event": "Beta release"}
-- ]
```

#### Implementation (Using Structured Output API)

**Use Gemini's Native Structured Output (`response_schema`) instead of prompt-based JSON:**

```go
import "google.golang.org/genai"

type StructuredSummarizer struct {
    llmClient *llm.Client
    langfuse  *langfuse.Client
}

// StructuredSummary schema for type-safe responses
type StructuredSummary struct {
    Summary       string          `json:"summary" jsonschema:"required,description=2-3 sentence overview"`
    KeyMoments    []string        `json:"key_moments" jsonschema:"required,minItems=3,maxItems=5,description=Bullet points including notable quotes"`
    Perspectives  []string        `json:"perspectives,omitempty" jsonschema:"description=Different viewpoints if applicable"`
    WhyImportant  string          `json:"why_important" jsonschema:"required,description=Significance and implications"`
    Context       string          `json:"context,omitempty" jsonschema:"description=Background information"`
    Timeline      []TimelineEvent `json:"timeline,omitempty" jsonschema:"description=Chronological events if applicable"`
}

type TimelineEvent struct {
    Date  string `json:"date" jsonschema:"required,format=date"`
    Event string `json:"event" jsonschema:"required"`
}

// GenerateStructuredSummary using Gemini structured output API
func (s *StructuredSummarizer) GenerateStructuredSummary(ctx context.Context, article core.Article, ragContext string) (*StructuredSummary, error) {
    trace := s.langfuse.StartTrace("structured_summarization", map[string]interface{}{
        "article_id": article.ID,
    })
    defer trace.End()

    prompt := buildStructuredPrompt(article, ragContext)

    // Use Gemini structured output (no JSON in prompt needed!)
    response, err := s.llmClient.GenerateStructuredText(ctx, prompt, llm.Options{
        Temperature: 0.3,
        ResponseSchema: &genai.Schema{
            Type: genai.TypeObject,
            Properties: map[string]*genai.Schema{
                "summary": {
                    Type: genai.TypeString,
                    Description: "2-3 sentence overview of the article",
                },
                "key_moments": {
                    Type: genai.TypeArray,
                    Items: &genai.Schema{Type: genai.TypeString},
                    Description: "3-5 bullet points of important developments (include notable quotes)",
                },
                "perspectives": {
                    Type: genai.TypeArray,
                    Items: &genai.Schema{Type: genai.TypeString},
                    Description: "Different viewpoints on the topic if applicable",
                },
                "why_important": {
                    Type: genai.TypeString,
                    Description: "Significance and implications (1-2 sentences)",
                },
                "context": {
                    Type: genai.TypeString,
                    Description: "Background information for understanding",
                },
                "timeline": {
                    Type: genai.TypeArray,
                    Items: &genai.Schema{
                        Type: genai.TypeObject,
                        Properties: map[string]*genai.Schema{
                            "date":  {Type: genai.TypeString, Description: "Date (YYYY-MM-DD)"},
                            "event": {Type: genai.TypeString, Description: "Event description"},
                        },
                        Required: []string{"date", "event"},
                    },
                    Description: "Chronological events if applicable",
                },
            },
            Required: []string{"summary", "key_moments", "why_important"},
        },
    })

    // Response is already typed StructuredSummary - no parsing needed!
    summary := response.(*StructuredSummary)

    trace.SetMetadata("sections_generated", 5)
    trace.SetMetadata("key_moments_count", len(summary.KeyMoments))

    return summary, nil
}
```

**Benefits of Structured Output API:**
- ✅ No JSON parsing errors
- ✅ No truncation mid-JSON
- ✅ Type-safe responses (guaranteed schema compliance)
- ✅ Automatic validation by Gemini
- ✅ Better reliability (Gemini enforces structure)
- ✅ Cleaner code (no manual JSON parsing)

#### Prompt Template

```
You are summarizing a tech article for developers who want to quickly understand key developments.

ARTICLE:
Title: {title}
URL: {url}
Content: {content}

CONTEXT (similar past coverage):
{rag_context}

Summarize with the following information:

1. **SUMMARY**: 2-3 sentence overview of the article
2. **KEY MOMENTS**: 3-5 bullet points of most important developments (include notable quotes)
3. **PERSPECTIVES**: Different viewpoints on the topic (if applicable)
4. **WHY IT MATTERS**: Significance and implications (1-2 sentences)
5. **CONTEXT**: Background information for understanding
6. **TIMELINE**: Chronological events (if applicable, with dates in YYYY-MM-DD format)

Note: The response will automatically be structured according to the schema.
No need to specify JSON format - it's handled by the API.
```

**Note:** With structured output API, we don't include JSON format instructions in the prompt. The schema enforces the structure automatically.

---

### 4. Citation Tracking (`internal/citations/`)

#### Purpose

Track source attribution for transparent citations (Kagi-inspired).

#### Data Model

```sql
CREATE TABLE citations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    summary_id UUID NOT NULL REFERENCES summaries(id) ON DELETE CASCADE,
    source_url TEXT NOT NULL,
    source_title TEXT,
    publisher TEXT,
    author TEXT,
    published_date TIMESTAMP,
    citation_type TEXT NOT NULL,  -- 'primary', 'supporting', 'quote'
    quote_text TEXT,  -- If citation_type = 'quote'
    accessed_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(summary_id, source_url)
);

CREATE INDEX idx_citations_summary ON citations(summary_id);
CREATE INDEX idx_citations_type ON citations(citation_type);
```

#### Interface

```go
type CitationTracker interface {
    // AddCitation records a source used in summary
    AddCitation(ctx context.Context, citation Citation) error

    // GetCitations retrieves all citations for a summary
    GetCitations(ctx context.Context, summaryID string) ([]Citation, error)

    // ExtractQuotes finds notable quotes in article
    ExtractQuotes(ctx context.Context, article core.Article) ([]Quote, error)
}

type Citation struct {
    SummaryID     string
    SourceURL     string
    SourceTitle   string
    Publisher     string
    Author        string
    PublishedDate *time.Time
    CitationType  string
    QuoteText     *string
}
```

#### Web UI Display

```html
<!-- Digest article with hover citations -->
<article>
  <h3>{title}</h3>
  <section class="summary">
    <p>
      Anthropic announced Claude 3.5 Sonnet improvements
      <sup class="citation" data-tooltip="TechCrunch, Oct 25, 2025">[1]</sup>
      with enhanced coding capabilities
      <sup class="citation" data-tooltip="The Verge, Oct 26, 2025">[2]</sup>.
    </p>
  </section>

  <section class="sources">
    <h4>Sources</h4>
    <ol>
      <li><a href="https://techcrunch.com/...">[1] TechCrunch</a> - "Anthropic releases Claude 3.5 Sonnet" (Oct 25, 2025)</li>
      <li><a href="https://theverge.com/...">[2] The Verge</a> - "Claude coding update" (Oct 26, 2025)</li>
    </ol>
  </section>
</article>
```

---

### 5. Content Sources Interface (`internal/sources/`)

#### Purpose

Unified interface for RSS, Search, and Manual URLs (implement separately).

#### Interface

```go
// ContentSource represents any source of article URLs
type ContentSource interface {
    // GetName returns source identifier
    GetName() string

    // GetType returns source type
    GetType() SourceType

    // FetchArticles retrieves new articles from this source
    FetchArticles(ctx context.Context, options FetchOptions) ([]core.Link, error)

    // GetStatus returns source health status
    GetStatus(ctx context.Context) (SourceStatus, error)
}

type SourceType string

const (
    SourceTypeRSS    SourceType = "rss"
    SourceTypeSearch SourceType = "search"
    SourceTypeManual SourceType = "manual"
)

type FetchOptions struct {
    MaxArticles  int
    SinceDate    *time.Time
    ThemeFilter  []string
}
```

#### Implementation

```go
// RSS Source
type RSSSource struct {
    feedID    string
    feedURL   string
    feedRepo  persistence.FeedRepository
    fetcher   *feeds.FeedFetcher
}

func (s *RSSSource) FetchArticles(ctx context.Context, opts FetchOptions) ([]core.Link, error) {
    feed, err := s.fetcher.FetchFeed(ctx, s.feedURL)
    // Convert feed items to core.Link
    return links, nil
}

// Search Source
type SearchSource struct {
    queryID      string
    searchClient search.SearchClient
}

func (s *SearchSource) FetchArticles(ctx context.Context, opts FetchOptions) ([]core.Link, error) {
    results, err := s.searchClient.Search(ctx, s.query, search.Options{
        MaxResults: opts.MaxArticles,
    })
    // Convert search results to core.Link
    return links, nil
}

// Manual Source
type ManualSource struct {
    urlRepo persistence.ManualURLRepository
}

func (s *ManualSource) FetchArticles(ctx context.Context, opts FetchOptions) ([]core.Link, error) {
    urls, err := s.urlRepo.ListUnprocessed(ctx)
    // Convert manual URLs to core.Link
    return links, nil
}
```

---

## Data Model

### Core Entities (PostgreSQL Schema)

#### Themes

```sql
CREATE TABLE themes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    keywords TEXT[],
    active BOOLEAN DEFAULT true,
    priority INT DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

#### Articles (Enhanced)

```sql
CREATE TABLE articles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL,
    content_type TEXT NOT NULL,  -- 'html', 'pdf', 'youtube'
    cleaned_text TEXT NOT NULL,
    raw_content TEXT,

    -- Source tracking
    source_type TEXT NOT NULL,  -- 'rss', 'search', 'manual'
    source_id UUID,  -- FK to feeds, search_queries, or manual_urls
    published_at TIMESTAMP,
    fetched_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Processing status
    processed BOOLEAN DEFAULT false,
    processing_error TEXT,

    -- Theme classification (NEW)
    theme_id UUID REFERENCES themes(id),
    theme_relevance FLOAT,  -- 0.0 - 1.0
    classification_reasoning TEXT,

    -- Clustering
    topic_cluster UUID,  -- FK to topic_clusters
    cluster_confidence FLOAT,
    category TEXT,
    tags TEXT[],
    quality_score FLOAT,

    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- Indexes
    INDEX idx_articles_url (url),
    INDEX idx_articles_source (source_type, source_id),
    INDEX idx_articles_theme (theme_id),
    INDEX idx_articles_processed (processed),
    INDEX idx_articles_published (published_at DESC)
);
```

#### Summaries (Enhanced with Structured Sections)

```sql
CREATE TABLE summaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,

    -- Basic summary
    summary_text TEXT NOT NULL,
    summary_type TEXT NOT NULL,  -- 'article', 'executive', 'cluster'
    model_used TEXT NOT NULL,
    token_count INT,
    cost_usd DECIMAL(10,6),

    -- Structured sections (Kagi-inspired)
    key_moments TEXT,         -- Bullet points (Markdown list)
    perspectives TEXT[],      -- Multiple viewpoints
    why_important TEXT,       -- Significance explanation
    key_quotes TEXT[],        -- Extracted quotes
    context TEXT,             -- Background information
    timeline JSONB,           -- [{"date": "...", "event": "..."}]

    -- RAG context
    rag_context_used BOOLEAN DEFAULT false,
    context_article_ids UUID[],

    -- Quality
    eval_score FLOAT,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(article_id, summary_type)
);
```

**Summary Types and Relationships:**

**Relationship:** Article (1) ──< Summaries (1 per type)

Most articles have ONE summary of type 'article'. The `summary_type` field enables different summary contexts:

1. **'article'** - Main article summary (one per article)
   - Generated during article processing
   - Stored with structured sections (key_moments, perspectives, why_important, context, timeline)
   - Cached with 7-day TTL
   - Example: Summary of single news article

2. **'cluster'** - Topic cluster summary (one per cluster, not per article)
   - Synthesizes multiple articles into cohesive narrative
   - Created during digest generation
   - References multiple article_ids via `context_article_ids`
   - Includes inline citations like [1], [2] referencing cluster articles
   - Example: "Model Releases" cluster summary covering 3 articles

3. **'executive'** - Digest-level summary (one per digest, not per article)
   - High-level overview of entire digest
   - Created after clustering completes
   - References top 3 articles from each cluster
   - Provides birds-eye view of the week's developments
   - Example: Weekly digest introduction summarizing all topics

**Database Constraint:**
```sql
-- One summary per article per type
UNIQUE(article_id, summary_type)
```

**Important Notes:**
- Each article typically has ONE 'article' summary (1:1 relationship with type discrimination)
- Cluster and executive summaries are NOT tied to a single article_id (would use NULL or junction table)
- The schema allows multiple summaries per article with different types, but in practice each type appears once per article

**Future Consideration:** Separate tables for different summary types:
- `article_summaries` (1:1 with articles)
- `cluster_summaries` (1:1 with topic_clusters)
- `digest_summaries` (1:1 with digests)

#### Citations (NEW)

```sql
CREATE TABLE citations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    summary_id UUID NOT NULL REFERENCES summaries(id) ON DELETE CASCADE,
    source_url TEXT NOT NULL,
    source_title TEXT,
    publisher TEXT,
    author TEXT,
    published_date TIMESTAMP,
    citation_type TEXT NOT NULL,  -- 'primary', 'supporting', 'quote'
    quote_text TEXT,
    accessed_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(summary_id, source_url)
);
```

#### Manual URLs (NEW)

```sql
CREATE TABLE manual_urls (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url TEXT NOT NULL UNIQUE,
    title TEXT,
    notes TEXT,
    theme_hint TEXT,
    submitted_by TEXT DEFAULT 'admin',
    processed BOOLEAN DEFAULT false,
    article_id UUID REFERENCES articles(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    INDEX idx_manual_urls_processed (processed)
);
```

#### Digests (Enhanced with Themes)

```sql
CREATE TABLE digests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    digest_date DATE NOT NULL,
    theme_id UUID REFERENCES themes(id),  -- NEW: Optional theme filter

    executive_summary TEXT,
    article_count INT NOT NULL,
    cluster_count INT NOT NULL,

    -- Rendering
    markdown_content TEXT,
    html_content TEXT,

    -- Publishing
    published BOOLEAN DEFAULT false,
    published_at TIMESTAMP,

    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),

    INDEX idx_digests_date (digest_date DESC),
    INDEX idx_digests_theme (theme_id),
    UNIQUE(digest_date, theme_id)  -- One digest per theme per date
);
```

#### Topic Clusters (Enhanced with Cluster Summaries)

```sql
CREATE TABLE topic_clusters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    label TEXT NOT NULL,
    description TEXT,

    -- NEW: Cluster-level summary
    summary_text TEXT,              -- Cohesive narrative of the cluster
    summary_citation_ids UUID[],    -- References to all articles in cluster

    centroid vector(768),
    article_count INT DEFAULT 0,
    keywords TEXT[],
    digest_id UUID,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

**Cluster Summary Purpose:**
- Creates a "brief of the brief" - one cohesive narrative per topic
- Synthesizes multiple articles into a single overview
- Includes citations to all articles in the cluster
- Allows users to quickly scan topics, then drill into individual articles

**Example:**
```
Cluster: "Model Releases"
Summary: "This week saw major LLM releases from multiple providers [1][2].
          Anthropic's Claude 3.5 Sonnet improvements [1] focus on coding capabilities,
          while Google's Gemini 2.0 [2] emphasizes multimodal features..."
Citations: [1] article-uuid-1, [2] article-uuid-2
```

#### RSS Feeds (Unchanged)

```sql
CREATE TABLE feeds (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    url TEXT NOT NULL UNIQUE,
    feed_type TEXT NOT NULL,
    last_modified TEXT,
    etag TEXT,
    active BOOLEAN DEFAULT true,
    last_fetched TIMESTAMP,
    last_error TEXT,
    fetch_count INT DEFAULT 0,
    error_count INT DEFAULT 0,
    fetch_interval_minutes INT DEFAULT 60,
    max_articles_per_fetch INT DEFAULT 20,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

#### Search Queries (NEW - for Phase 2)

```sql
CREATE TABLE search_queries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    topic TEXT NOT NULL,
    query_text TEXT NOT NULL,
    search_engine TEXT NOT NULL,  -- 'google', 'bing'
    active BOOLEAN DEFAULT true,
    max_results INT DEFAULT 10,
    search_interval_hours INT DEFAULT 168,  -- Weekly
    last_executed TIMESTAMP,
    last_result_count INT,
    last_error TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
```

#### Article Embeddings (pgvector)

```sql
-- Enable pgvector
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE article_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    embedding_type TEXT NOT NULL,  -- 'summary', 'full_text'
    embedding vector(768) NOT NULL,
    model_version TEXT NOT NULL DEFAULT 'text-embedding-004',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),

    UNIQUE(article_id, embedding_type)
);

-- Create index for similarity search (IVFFlat)
CREATE INDEX ON article_embeddings USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

-- Or HNSW for better performance (PostgreSQL 16+)
-- CREATE INDEX ON article_embeddings USING hnsw (embedding vector_cosine_ops);
```

### Data Relationships

```
Theme (1) ──< Articles (N)
    │
└──< Digests (N) (optional theme filtering)

Article (1) ──< Summaries (N)
    │
    ├──< ArticleEmbeddings (N)
    └──< TopicCluster (1)

Summary (1) ──< Citations (N)

Digest (1) ──< TopicClusters (N) ──< Articles (N)

ManualURL (1) ──< Article (1) (linked after processing)
```

---

## API Design

### REST API Endpoints

#### Public Endpoints

**Digests:**
```
GET  /api/digests                     List digests (paginated)
GET  /api/digests/latest              Get latest digest (default: all themes)
GET  /api/digests/latest?theme=GenAI  Get latest digest for theme
GET  /api/digests/{id}                Get digest by ID
GET  /api/digests/{date}              Get digest by date
GET  /api/digests/{date}?theme=GenAI  Get digest by date + theme
```

**Articles:**
```
GET  /api/articles                    List articles (paginated, filtered)
GET  /api/articles/{id}               Get article with structured summary
GET  /api/articles/{id}/citations     Get article citations
GET  /api/articles/recent             Recent articles (last 7 days)
GET  /api/articles/search             Full-text search
```

**Themes:**
```
GET  /api/themes                      List active themes
GET  /api/themes/{id}                 Get theme with article count
GET  /api/themes/{id}/articles        Articles by theme
```

**Topics:**
```
GET  /api/topics                      List all topic clusters
GET  /api/topics/{id}                 Get topic with articles
```

#### Admin Endpoints (Authenticated)

**Theme Management:**
```
GET    /api/admin/themes              List all themes (including inactive)
POST   /api/admin/themes              Create theme
PUT    /api/admin/themes/{id}         Update theme
DELETE /api/admin/themes/{id}         Delete theme
POST   /api/admin/themes/{id}/toggle  Enable/disable theme
```

**Feed Management:**
```
GET    /api/admin/feeds               List all feeds
POST   /api/admin/feeds               Create feed
PUT    /api/admin/feeds/{id}          Update feed
DELETE /api/admin/feeds/{id}          Delete feed
GET    /api/admin/feeds/{id}/stats    Feed statistics
POST   /api/admin/feeds/{id}/test     Test feed URL
```

**Manual URL Management:**
```
GET    /api/admin/manual-urls          List manual URLs
POST   /api/admin/manual-urls          Submit single URL
POST   /api/admin/manual-urls/batch    Upload file with URLs
DELETE /api/admin/manual-urls/{id}     Delete manual URL
GET    /api/admin/manual-urls/unprocessed  List pending URLs
POST   /api/admin/manual-urls/process   Trigger processing
```

**Search Management (Phase 2):**
```
GET    /api/admin/search              List search queries
POST   /api/admin/search              Create search query
PUT    /api/admin/search/{id}         Update query
DELETE /api/admin/search/{id}         Delete query
```

**Configuration:**
```
GET    /api/admin/config              Get all configuration
PUT    /api/admin/config              Update configuration
```

**~~Observability~~ - NOT IMPLEMENTED**
- Use LangFuse web dashboard instead

**~~Evaluation~~ - NOT IMPLEMENTED VIA API**
- CLI-only commands (see Evaluation Framework section)

**~~Jobs~~ - NOT IMPLEMENTED VIA API**
- Manual CLI commands + optional infrastructure CRON

### CLI Commands

**Theme Management:**
```bash
briefly theme list
briefly theme add "Web3" --keywords "blockchain,crypto"
briefly theme edit "GenAI" --description "..."
briefly classify-article <url>  # Test classification
```

**Content Submission:**
```bash
# Manual URLs
briefly add-url <url> [--title] [--theme] [--notes]
briefly add-urls <file>
briefly list-manual-urls [--unprocessed]
briefly process-manual-urls

# RSS feeds
briefly feed add <url> --name "Feed Name"
briefly feed list
briefly feed test <url>
```

**Quick Article Summary (Single Article):**
```bash
# Summarize any article you encounter during the day
briefly summarize https://example.com/article

# Output: Structured summary with all sections
# - Summary (2-3 sentence overview)
# - Key Moments (3-5 highlights including notable quotes)
# - Perspectives (different viewpoints if applicable)
# - Why It Matters (significance and implications)
# - Context (background information)

# Save summary to file
briefly summarize https://example.com/article --output summary.md

# Output format options
briefly summarize https://example.com/article --format json  # JSON output
briefly summarize https://example.com/article --format md    # Markdown (default)
briefly summarize https://example.com/article --format text  # Plain text

# Bypass cache (force fresh fetch)
briefly summarize https://example.com/article --no-cache

# Use Cases:
# - Quick understanding of an article you found
# - Test summarization quality before adding to digest
# - Generate summary for sharing with colleagues
# - Personal knowledge base building
```

**Pipeline Execution (Manual Testing):**
```bash
# Run each step independently
briefly aggregate [--source rss|search|manual]  # Step 1: Fetch URLs
briefly classify [--article-id <id>]            # Step 2: Theme classification
briefly summarize [--article-id <id>]           # Step 3: Generate summary
briefly cluster [--theme <name>]                # Step 4: K-means clustering
briefly generate-digest [--theme <name>]        # Step 5: Executive summary + render

# Run full pipeline
briefly digest generate [--theme <name>]        # All steps
```

**Evaluation (CLI-Only):**
```bash
# Dataset management
briefly eval dataset create "summarization-v1"
briefly eval dataset add "summarization-v1" --input <file> --expected <file>
briefly eval dataset list

# Run evaluations
briefly eval run --dataset "summarization-v1" --prompt "prompts/summary_v2.txt"
briefly eval compare --dataset "summarization-v1" \
  --prompts "prompts/v1.txt,prompts/v2.txt"

# View results
briefly eval report <run-id>
```

**Server:**
```bash
briefly serve --port 8080
```

### Request/Response Examples

**GET /api/digests/latest?theme=GenAI**

Response:
```json
{
  "id": "abc-123",
  "title": "GenAI Digest - October 31, 2025",
  "digest_date": "2025-10-31",
  "theme": {
    "id": "theme-1",
    "name": "GenAI"
  },
  "executive_summary": "This week saw major developments in...",
  "article_count": 15,
  "cluster_count": 3,
  "clusters": [
    {
      "id": "cluster-1",
      "label": "Model Releases",
      "article_count": 5,
      "articles": [
        {
          "id": "article-1",
          "title": "Anthropic announces Claude 3.5 Sonnet improvements",
          "url": "https://...",
          "summary": {
            "summary_text": "Anthropic released improvements...",
            "key_moments": ["Enhanced coding capabilities", "Better reasoning"],
            "perspectives": ["Technical perspective", "Business impact"],
            "why_important": "Represents significant advancement in...",
            "key_quotes": ["Quote from CEO..."],
            "context": "This follows previous release in..."
          },
          "citations": [
            {
              "source_url": "https://techcrunch.com/...",
              "source_title": "Anthropic releases Claude 3.5",
              "publisher": "TechCrunch",
              "published_date": "2025-10-29T10:00:00Z",
              "citation_type": "primary"
            }
          ],
          "published_at": "2025-10-29T10:00:00Z",
          "theme": "GenAI",
          "theme_relevance": 0.98
        }
      ]
    }
  ],
  "published_at": "2025-10-31T09:00:00Z"
}
```

---

## Multi-Agent Architecture

*(Simplified from v1.0 - keep Manager-Worker pattern but defer complex implementation)*

### Design Philosophy

**Manager-Worker Pattern:** Central Research Lead delegates tasks to worker pool, collects results.

**LangFuse Integration:** All agent executions traced in LangFuse, not custom observability.

### Agent Types

1. **Research Lead (Manager)**: Plans aggregation, delegates to workers
2. **RSS Worker**: Fetches articles from RSS feeds
3. **Search Worker**: Executes web searches (Phase 2)
4. **Manual Worker**: Processes manually submitted URLs
5. **Analysis Worker**: Summarizes and classifies articles

### Implementation (Simplified)

For MVP, implement as **sequential processing with clear separation**:

```go
// Simple agent interface
type Agent interface {
    Execute(ctx context.Context, task Task) (Result, error)
}

// Manager delegates sequentially (parallelization in later phase)
func (m *ResearchLeadAgent) Aggregate(ctx context.Context) ([]core.Article, error) {
    // 1. Delegate to RSS Worker
    rssLinks, _ := m.rssWorker.Execute(ctx, RSSTask{})

    // 2. Delegate to Manual Worker
    manualLinks, _ := m.manualWorker.Execute(ctx, ManualTask{})

    // 3. Combine results
    allLinks := append(rssLinks, manualLinks...)

    // 4. Delegate to Analysis Worker for each article
    articles := make([]core.Article, 0)
    for _, link := range allLinks {
        article, _ := m.analysisWorker.Execute(ctx, AnalysisTask{Link: link})
        articles = append(articles, article)
    }

    return articles, nil
}
```

### Go Concurrency Pattern

**Worker Pool with Goroutines and Channels:**

```go
import (
    "context"
    "sync"
)

type WorkerPool struct {
    workers    int
    taskChan   chan Task
    resultChan chan Result
    wg         *sync.WaitGroup
    ctx        context.Context
    cancel     context.CancelFunc
}

// NewWorkerPool creates a pool of worker goroutines
func NewWorkerPool(workers int) *WorkerPool {
    ctx, cancel := context.WithCancel(context.Background())
    return &WorkerPool{
        workers:    workers,
        taskChan:   make(chan Task, workers*2),  // Buffered channel
        resultChan: make(chan Result, workers*2),
        wg:         &sync.WaitGroup{},
        ctx:        ctx,
        cancel:     cancel,
    }
}

// Start launches worker goroutines
func (p *WorkerPool) Start() {
    for i := 0; i < p.workers; i++ {
        p.wg.Add(1)
        go p.worker(i)
    }
}

// worker processes tasks from channel
func (p *WorkerPool) worker(id int) {
    defer p.wg.Done()

    for {
        select {
        case task, ok := <-p.taskChan:
            if !ok {
                return  // Channel closed, exit
            }

            // Process task with LangFuse tracing
            result := p.processTask(task)
            p.resultChan <- result

        case <-p.ctx.Done():
            return  // Context cancelled
        }
    }
}

// SubmitTask sends task to worker pool (non-blocking)
func (p *WorkerPool) SubmitTask(task Task) error {
    select {
    case p.taskChan <- task:
        return nil
    case <-p.ctx.Done():
        return p.ctx.Err()
    }
}

// CollectResults gathers results from workers
func (p *WorkerPool) CollectResults(count int) []Result {
    results := make([]Result, 0, count)
    for i := 0; i < count; i++ {
        select {
        case result := <-p.resultChan:
            results = append(results, result)
        case <-p.ctx.Done():
            break
        }
    }
    return results
}

// Shutdown gracefully stops all workers
func (p *WorkerPool) Shutdown() {
    close(p.taskChan)       // Signal workers to stop
    p.wg.Wait()             // Wait for all workers to finish
    close(p.resultChan)     // Close result channel
    p.cancel()              // Cancel context
}
```

**Usage Example:**

```go
func (m *ResearchLeadAgent) AggregateParallel(ctx context.Context, links []core.Link) ([]core.Article, error) {
    // Create worker pool (5 workers for parallel processing)
    pool := NewWorkerPool(5)
    pool.Start()
    defer pool.Shutdown()

    // Submit tasks to workers
    for _, link := range links {
        task := AnalysisTask{
            Link:    link,
            TraceID: generateTraceID(),
        }
        pool.SubmitTask(task)
    }

    // Collect results from workers
    results := pool.CollectResults(len(links))

    // Convert results to articles
    articles := make([]core.Article, 0, len(results))
    for _, result := range results {
        if result.Error == nil {
            articles = append(articles, result.Article)
        }
    }

    return articles, nil
}
```

**Demonstrates:**
- **Goroutine worker pools** for parallel processing
- **Channel-based** task distribution (buffered channels for efficiency)
- **Context-aware** cancellation (`context.Context`)
- **sync.WaitGroup** for graceful shutdown
- **Go runtime scaling** (GOMAXPROCS automatically utilizes CPU cores)

**Performance Benefits:**
- Process 50 articles in parallel (5 workers)
- Reduces total time from ~15 minutes to ~3-5 minutes
- Efficient resource utilization (CPU + network I/O)
- Non-blocking task submission

---

## LangFuse Observability

### Why LangFuse?

**Replaces custom observability** to focus on proven tooling:

- ✅ **LLM-Specific Metrics**: Tokens, cost, latency, prompt versions
- ✅ **Trace Hierarchy**: Parent-child relationships for multi-agent calls
- ✅ **Prompt Management**: Version control for prompts
- ✅ **Web Dashboard**: Pre-built UI (no need to build custom)
- ✅ **Evaluation Integration**: Can link evals to production traces
- ✅ **Industry Standard**: Shows familiarity with LLMOps tools

### Integration Strategy

**Full SDK Integration:**

```bash
go get github.com/langfuse/langfuse-go
```

#### Trace Structure

```go
import "github.com/langfuse/langfuse-go"

// Initialize LangFuse
client := langfuse.New(langfuse.Config{
    PublicKey:  os.Getenv("LANGFUSE_PUBLIC_KEY"),
    SecretKey:  os.Getenv("LANGFUSE_SECRET_KEY"),
    Host:       os.Getenv("LANGFUSE_HOST"), // Self-hosted or cloud
})

// Trace a pipeline execution
trace := client.Trace(langfuse.TraceParams{
    Name:   "weekly_digest_generation",
    UserId: "admin",
    Metadata: map[string]interface{}{
        "digest_date": "2025-10-31",
        "theme":       "GenAI",
    },
})

// Trace LLM call
generation := trace.Generation(langfuse.GenerationParams{
    Name:   "theme_classification",
    Model:  "gemini-2.5-flash-preview",
    Input:  prompt,
    Metadata: map[string]interface{}{
        "article_id":  articleID,
        "article_url": articleURL,
    },
})

// After LLM call
generation.End(langfuse.GenerationEndParams{
    Output: response,
    Usage: langfuse.Usage{
        Input:  inputTokens,
        Output: outputTokens,
        Total:  inputTokens + outputTokens,
    },
    Metadata: map[string]interface{}{
        "theme_found": "GenAI",
        "relevance":   0.95,
    },
})

trace.End()
```

#### Instrumentation Points

**Trace all major operations:**

1. **Aggregation**: RSS/search/manual URL fetching
2. **Theme Classification**: LLM-based theme detection
3. **Summarization**: Structured summary generation
4. **Embedding**: Vector generation
5. **Clustering**: K-means clustering (not LLM, but metadata tracking)
6. **Executive Summary**: Digest narrative generation

#### Cost Tracking

LangFuse automatically calculates costs based on:
- Model type (Gemini Flash, etc.)
- Token counts (input + output)
- Custom cost configurations

**View in LangFuse Dashboard:**
- Total cost per digest generation
- Cost breakdown by operation type
- Cost trends over time

#### Prompt Management

**Store prompt versions in LangFuse:**

```go
// Retrieve prompt from LangFuse
prompt := client.GetPrompt("theme_classification_v3")

// Use prompt with variables
finalPrompt := prompt.Compile(map[string]interface{}{
    "article_title":   article.Title,
    "article_content": article.Content,
    "themes":          themeList,
})

// Trace will link to prompt version
generation := trace.Generation(langfuse.GenerationParams{
    Name:       "theme_classification",
    PromptName: "theme_classification",
    PromptVersion: 3,
    Input:      finalPrompt,
})
```

### LangFuse Dashboard

**Access via:** https://cloud.langfuse.com (or self-hosted)

**Key Views:**
- **Traces**: Hierarchical view of all operations
- **Generations**: All LLM calls with token/cost details
- **Prompts**: Version-controlled prompt library
- **Datasets**: Link to evaluation datasets
- **Sessions**: Group related traces (e.g., weekly digest)
- **Users**: Track admin actions

---

## PostHog Analytics

### Why PostHog?

**Product analytics to showcase user engagement:**

- ✅ **User Tracking**: Anonymous user sessions, page views
- ✅ **Feature Flags**: A/B test digest formats
- ✅ **Funnels**: Track user journey (homepage → digest → article)
- ✅ **Retention**: Measure repeat visitors
- ✅ **Self-Hosted or Cloud**: Flexible deployment
- ✅ **Open Source**: Can inspect code, self-host for free

**Portfolio Value:** Demonstrate "X users/day, Y% engagement" metrics.

### Integration Strategy

#### Frontend (JavaScript)

```html
<!-- Add to all pages -->
<script>
!function(t,e){var o,n,p,r;e.__SV||(window.posthog=e,e._i=[],e.init=function(i,s,a){function g(t,e){var o=e.split(".");2==o.length&&(t=t[o[0]],e=o[1]),t[e]=function(){t.push([e].concat(Array.prototype.slice.call(arguments,0)))}}(p=t.createElement("script")).type="text/javascript",p.async=!0,p.src=s.api_host+"/static/array.js",(r=t.getElementsByTagName("script")[0]).parentNode.insertBefore(p,r);var u=e;for(void 0!==a?u=e[a]=[]:a="posthog",u.people=u.people||[],u.toString=function(t){var e="posthog";return"posthog"!==a&&(e+="."+a),t||(e+=" (stub)"),e},u.people.toString=function(){return u.toString(1)+".people (stub)"},o="capture identify alias people.set people.set_once set_config register register_once unregister opt_out_capturing has_opted_out_capturing opt_in_capturing reset isFeatureEnabled onFeatureFlags getFeatureFlag getFeatureFlagPayload reloadFeatureFlags group updateEarlyAccessFeatureEnrollment getEarlyAccessFeatures getActiveMatchingSurveys getSurveys onSessionId".split(" "),n=0;n<o.length;n++)g(u,o[n]);e._i.push([i,s,a])},e.__SV=1)}(document,window.posthog||[]);

posthog.init('{PROJECT_API_KEY}', {
    api_host: 'https://app.posthog.com',
    // or self-hosted: 'https://posthog.yourdomain.com'
});
</script>
```

#### Backend (Go SDK)

```go
import "github.com/posthog/posthog-go"

// Initialize PostHog
client, _ := posthog.NewWithConfig(
    os.Getenv("POSTHOG_API_KEY"),
    posthog.Config{
        Endpoint: os.Getenv("POSTHOG_API_HOST"),
    },
)
defer client.Close()

// Track backend events
client.Enqueue(posthog.Capture{
    DistinctId: userID,  // Or anonymous ID
    Event:      "digest_generated",
    Properties: map[string]interface{}{
        "digest_date":    "2025-10-31",
        "theme":          "GenAI",
        "article_count":  15,
        "cluster_count":  3,
        "generation_time_ms": 45000,
    },
})
```

### Events to Track

#### Frontend Events

```javascript
// Page views (auto-tracked by PostHog)

// User interactions
posthog.capture('digest_viewed', {
    digest_id: 'abc-123',
    digest_date: '2025-10-31',
    theme: 'GenAI'
});

posthog.capture('article_clicked', {
    article_id: 'article-1',
    article_url: 'https://...',
    position_in_digest: 1,
    cluster_label: 'Model Releases'
});

posthog.capture('citation_clicked', {
    citation_url: 'https://...',
    publisher: 'TechCrunch'
});

posthog.capture('theme_filter_changed', {
    from_theme: 'All',
    to_theme: 'GenAI'
});

posthog.capture('cluster_expanded', {
    cluster_id: 'cluster-1',
    cluster_label: 'Model Releases'
});
```

#### Backend Events

```go
// Digest generation
client.Enqueue(posthog.Capture{
    Event: "digest_generated",
    Properties: map[string]interface{}{
        "digest_date": date,
        "theme": theme,
        "article_count": count,
        "generation_time_ms": duration,
        "cost_usd": cost,
    },
})

// Article processing
client.Enqueue(posthog.Capture{
    Event: "article_processed",
    Properties: map[string]interface{}{
        "article_id": id,
        "source_type": sourceType,
        "theme": theme,
        "processing_time_ms": duration,
    },
})
```

### PostHog Dashboard

**Key Metrics:**

1. **Daily Active Users (DAU)**
2. **Weekly Active Users (WAU)**
3. **Page Views per Session**
4. **Top Themes Viewed**
5. **Most Clicked Articles**
6. **Average Session Duration**
7. **Retention (7-day, 30-day)**
8. **Funnel: Homepage → Digest → Article → External Link**

---

## RAG Implementation

*(Same as v1.0, no major changes)*

### Vector Store (pgvector)

```sql
CREATE EXTENSION vector;

CREATE TABLE article_embeddings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    article_id UUID NOT NULL REFERENCES articles(id) ON DELETE CASCADE,
    embedding_type TEXT NOT NULL,
    embedding vector(768) NOT NULL,
    model_version TEXT NOT NULL DEFAULT 'text-embedding-004',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(article_id, embedding_type)
);

CREATE INDEX ON article_embeddings USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);
```

### Retriever Interface

```go
type Retriever interface {
    RetrieveContext(ctx context.Context, query string, options RetrievalOptions) ([]core.Article, error)
    RetrieveSimilarArticles(ctx context.Context, articleID string, limit int) ([]core.Article, error)
}

type RetrievalOptions struct {
    Limit         int
    MinSimilarity float64
    ThemeFilter   []string  // NEW: Filter by theme
    DateRange     *DateRange
}
```

### RAG Use Cases

1. **Theme Classification**: Retrieve similar articles to infer theme consistency
2. **Summarization**: Include context from past similar articles
3. **Cluster Labeling**: Use historical cluster names for consistency

---

## Evaluation Framework

### CLI-Only Approach

**No web API endpoints** - evaluation is admin-driven via CLI commands.

### Dataset Management

```bash
# Create dataset
briefly eval dataset create "summarization-v1" \
  --description "100 hand-labeled article summaries"

# Add examples from CSV
briefly eval dataset add "summarization-v1" \
  --input examples/inputs.jsonl \
  --expected examples/expected.jsonl

# examples/inputs.jsonl format:
# {"article_id": "abc", "title": "...", "content": "..."}

# examples/expected.jsonl format:
# {"summary": "...", "key_moments": [...], "why_important": "..."}
```

### Running Evaluations

```bash
# Evaluate current prompt
briefly eval run \
  --dataset "summarization-v1" \
  --prompt-file prompts/summary_v3.txt \
  --sample-size 50

# Compare multiple prompts
briefly eval compare \
  --dataset "summarization-v1" \
  --prompts "prompts/v1.txt,prompts/v2.txt,prompts/v3.txt"

# Output:
# Prompt v1: Avg score 3.8/5.0, Cost $0.45
# Prompt v2: Avg score 4.1/5.0, Cost $0.52
# Prompt v3: Avg score 4.3/5.0, Cost $0.48 ← Winner
```

### LLM-as-Judge

**Evaluation prompt:**

```
You are evaluating an AI-generated article summary.

ARTICLE:
{article_content}

AI SUMMARY:
{ai_summary}

EXPECTED SUMMARY:
{expected_summary}

CRITERIA:
1. Accuracy (1-5): Factual correctness
2. Conciseness (1-5): Appropriate length
3. Relevance (1-5): Focuses on important points
4. Completeness (1-5): Covers key moments

Provide scores and reasoning in JSON:
{
  "accuracy": {"score": 5, "reasoning": "..."},
  "conciseness": {"score": 4, "reasoning": "..."},
  "relevance": {"score": 5, "reasoning": "..."},
  "completeness": {"score": 4, "reasoning": "..."},
  "overall_score": 4.5,
  "confidence": 0.9
}
```

### Integration with LangFuse

**Link eval runs to LangFuse:**

```go
// During eval run, create LangFuse trace
trace := langfuse.Trace{
    Name: "eval_run_summarization_v1",
    Tags: []string{"evaluation", "summarization"},
    Metadata: map[string]interface{}{
        "dataset": "summarization-v1",
        "prompt_version": "v3",
    },
}

// Each example evaluation is a generation
for _, example := range dataset.Examples {
    generation := trace.Generation{
        Name: "eval_example",
        Input: example.Input,
        Output: aiOutput,
        Metadata: map[string]interface{}{
            "expected_output": example.Expected,
            "judge_score": score,
        },
    }
}
```

---

## Technical Decisions

### Decision Log

#### 1. Database: PostgreSQL + pgvector

**Decision:** Use PostgreSQL with pgvector for all data (articles, embeddings, analytics)

**Rationale:**
- pgvector enables similarity search without separate vector DB
- Single database reduces operational complexity
- Cost-effective (free tier available)
- Performance sufficient for <100K articles

**Alternatives Considered:**
- Pinecone/Weaviate: Adds cost and complexity
- ChromaDB: Not suitable for production multi-user

#### 2. Observability: LangFuse (Full Integration)

**Decision:** Replace custom observability with LangFuse SDK

**Rationale:**
- Proven tool shows industry familiarity
- Pre-built dashboard saves development time
- LLM-specific metrics (tokens, cost, prompts)
- Easier to demonstrate in portfolio ("integrated LangFuse")
- More time to focus on unique features (themes, structured summaries)

**Alternatives Considered:**
- Custom observability: Too much effort, reinventing wheel
- LangSmith: LangChain-focused, prefer independent tool

#### 3. Analytics: PostHog (Full Integration)

**Decision:** Use PostHog for product analytics

**Rationale:**
- Shows user engagement metrics for portfolio
- Open-source with self-hosted option
- Frontend + backend tracking
- Feature flags for A/B testing digest formats
- Free tier sufficient for early stages

**Alternatives Considered:**
- Google Analytics: Privacy concerns, ad blockers
- Custom tracking: Too simple, lacks insights

#### 4. Theme System Over Free-Form Tags

**Decision:** Admin-configured themes with LLM classification

**Rationale:**
- Controlled vocabulary prevents tag sprawl
- LLM classification more reliable with predefined themes
- Enables theme-specific digests
- Better UX for filtering

**Alternatives Considered:**
- Free-form tags: Too messy, inconsistent
- No themes: Can't filter to specific interests

#### 5. CLI-Only Evaluation (No Web API)

**Decision:** Evaluation framework accessed only via CLI

**Rationale:**
- Evaluation is admin/developer activity, not end-user
- Avoids building complex eval UI
- Faster iteration on prompts via CLI
- Results can be viewed in LangFuse dashboard

**Alternatives Considered:**
- Full web UI: Overkill for admin-only feature

#### 6. RSS and Search as Separate Phases

**Decision:** Implement RSS (Phase 1) and Search (Phase 2) separately

**Rationale:**
- RSS is simpler, proven infrastructure already exists
- Search adds API dependencies (Google/Bing)
- Allows testing theme classification with RSS first
- Can ship RSS-only MVP

**Alternatives Considered:**
- Implement both together: Increases complexity, delays MVP

#### 7. Manual URL Submission (Both CLI + Web)

**Decision:** Support both CLI commands and web admin form

**Rationale:**
- CLI for power users (scriptable, fast)
- Web for ease of use (batch upload, notes)
- Demonstrates full-stack capability

**Alternatives Considered:**
- CLI only: Less user-friendly
- Web only: Misses automation opportunities

#### 8. Weekly Publishing (Not Daily)

**Decision:** Fixed weekly schedule (e.g., every Monday 9am)

**Rationale:**
- Matches personal use case (weekly catch-up)
- Less pressure than daily
- More time for quality curation
- Inspired by Kagi's time-bounded model (theirs is daily)

**Alternatives Considered:**
- Daily: Too frequent for manual review
- On-demand: Loses scheduled cadence benefit

#### 9. Structured Summaries (Kagi-Inspired)

**Decision:** Adopt Kagi's multi-section summary format

**Rationale:**
- Proven UX from successful product
- Provides more value than single paragraph
- Differentiates from basic news aggregators
- Shows attention to detail and user experience

**Alternatives Considered:**
- Single paragraph summary: Too basic

#### 10. Infrastructure vs In-App CRON

**Decision:** Support both options (user's choice)

**Rationale:**
- Infrastructure CRON (crontab, systemd timer) is simpler, more reliable
- In-app scheduler useful for platforms without cron access (some PaaS)
- Flexibility demonstrates architectural awareness

**Alternatives Considered:**
- In-app only: Adds complexity
- Infrastructure only: Limits deployment options

---

## Deployment Architecture

### Infrastructure Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    CLIENT (Browser)                          │
└────────────────────────┬────────────────────────────────────┘
                         │ HTTPS
                         ▼
┌─────────────────────────────────────────────────────────────┐
│                   Railway / Fly.io                           │
│  ┌──────────────────────────────────────────────────────┐  │
│  │           Briefly Web Server (Go)                     │  │
│  │  - Chi HTTP router                                    │  │
│  │  - Public API endpoints                               │  │
│  │  - Admin API endpoints                                │  │
│  │  - PostHog event tracking (backend)                   │  │
│  └──────────────────────────────────────────────────────┘  │
└───────────────────────┬──────────────────────────────────────┘
                        │
            ┌───────────┼──────────┬──────────────┐
            │           │          │              │
            ▼           ▼          ▼              ▼
┌──────────────┐  ┌─────────┐  ┌──────────┐  ┌─────────────┐
│ PostgreSQL   │  │LangFuse │  │ PostHog  │  │External APIs│
│(Supabase/    │  │ Cloud   │  │  Cloud   │  │             │
│ Neon)        │  │         │  │          │  │ - Gemini    │
│              │  │         │  │          │  │ - Google    │
│ • Articles   │  │         │  │          │  │   Search    │
│ • Summaries  │  │         │  │          │  │ - Bing      │
│ • Themes     │  │         │  │          │  │   Search    │
│ • Digests    │  │         │  │          │  │             │
│ • Embeddings │  │         │  │          │  └─────────────┘
│  (pgvector)  │  │         │  │          │
└──────────────┘  └─────────┘  └──────────┘

CRON Job (Infrastructure-level or In-App)
    ↓
briefly digest generate --theme GenAI
    (Runs weekly, e.g., Monday 9am)
```

### Deployment Options

#### Option 1: Railway (Recommended for Simplicity)

**Pros:**
- Easy deployment (GitHub integration)
- Built-in PostgreSQL addon
- Generous free tier
- Automatic HTTPS

**Setup:**
```bash
railway login
railway init
railway add postgres
railway up
```

**CRON:** Use Railway's "Cron Jobs" feature or external service (cron-job.org)

**Cost:** ~$5-20/month (hobby plan)

#### Option 2: Fly.io (Recommended for Performance)

**Pros:**
- Global edge network
- Free tier includes PostgreSQL (with pgvector support)
- Dockerfile-based deployment
- Better for learning container orchestration

**Setup:**
```bash
fly auth login
fly launch
fly postgres create --name briefly-db --vm-size shared-cpu-1x --volume-size 10
fly deploy
```

**CRON:** Use Fly Machines API + external scheduler OR deploy separate cron machine

**Cost:** Free tier available, ~$10-30/month for production

#### Option 3: VPS (DigitalOcean, Hetzner)

**Pros:**
- Full control
- Cheapest at scale ($5-10/month)
- Can run Docker Compose
- Native crontab support

**Cons:**
- Manual setup and maintenance
- Manual HTTPS (Let's Encrypt)
- Manual scaling

**Setup:**
```bash
# On VPS
docker-compose up -d
crontab -e
# Add: 0 9 * * 1 cd /app && briefly digest generate --theme GenAI
```

### Environment Configuration

**.env.production:**
```bash
# Database
DATABASE_URL=postgresql://user:pass@host:5432/briefly?sslmode=require

# LLM APIs
GEMINI_API_KEY=your-gemini-api-key

# Search APIs (Phase 2)
GOOGLE_SEARCH_API_KEY=your-google-api-key
GOOGLE_SEARCH_ENGINE_ID=your-search-engine-id
BING_SEARCH_API_KEY=your-bing-api-key

# Observability
LANGFUSE_PUBLIC_KEY=your-langfuse-public-key
LANGFUSE_SECRET_KEY=your-langfuse-secret-key
LANGFUSE_HOST=https://cloud.langfuse.com  # or self-hosted

# Analytics
POSTHOG_API_KEY=your-posthog-api-key
POSTHOG_API_HOST=https://app.posthog.com  # or self-hosted

# Server
PORT=8080
ENVIRONMENT=production
LOG_LEVEL=info

# Scheduler (if using in-app)
SCHEDULER_ENABLED=true
SCHEDULER_TIMEZONE=America/Los_Angeles

# CRON expression for digest generation
# Daily: 0 9 * * * (every day at 9am)
# Weekly: 0 9 * * 1 (every Monday at 9am)
# Custom: Any valid cron expression
DIGEST_CRON=0 9 * * 1  # Default: Weekly on Monday
```

### Docker Setup

**Dockerfile:**
```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o briefly ./cmd/briefly

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/briefly .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/static ./static

EXPOSE 8080
CMD ["./briefly", "serve", "--port", "8080"]
```

**docker-compose.yml (local development):**
```yaml
version: '3.8'

services:
  postgres:
    image: pgvector/pgvector:pg16
    environment:
      POSTGRES_DB: briefly
      POSTGRES_USER: briefly
      POSTGRES_PASSWORD: briefly
    ports:
      - "5432:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data

  briefly:
    build: .
    ports:
      - "8080:8080"
    environment:
      DATABASE_URL: postgresql://briefly:briefly@postgres:5432/briefly?sslmode=disable
      GEMINI_API_KEY: ${GEMINI_API_KEY}
      LANGFUSE_PUBLIC_KEY: ${LANGFUSE_PUBLIC_KEY}
      LANGFUSE_SECRET_KEY: ${LANGFUSE_SECRET_KEY}
      POSTHOG_API_KEY: ${POSTHOG_API_KEY}
    depends_on:
      - postgres
    volumes:
      - ./templates:/root/templates
      - ./static:/root/static

volumes:
  pgdata:
```

---

## Appendix: Embedding Explanation

### What Are Embeddings? (Metaphor)

**The Library GPS Analogy:**

Imagine you have a massive library with millions of books but no Dewey Decimal System. Finding similar books would require reading every single one.

**Embeddings are like giving each book a precise GPS coordinate in a 768-dimensional space:**

- **Similar books cluster together**: All Python programming books end up in the same "neighborhood"
- **Distance = Similarity**: Books 10 feet apart are more similar than books 100 feet apart
- **Math-friendly**: Instead of "does this text contain 'Python'?" we can ask "what are the 5 nearest books to this coordinate?"

**Real Example for Briefly:**

```
Text: "Gemini 2.0 released with improved reasoning"
Embedding: [0.23, -0.45, 0.67, ..., 0.12] (768 numbers)

Text: "Claude 3.5 Sonnet gets better at coding"
Embedding: [0.21, -0.43, 0.69, ..., 0.15] (768 numbers)

Distance: SMALL → These are SIMILAR topics (both AI model releases)

Text: "New recipe for chocolate cake"
Embedding: [-0.88, 0.12, -0.34, ..., 0.56] (768 numbers)

Distance: LARGE → This is DIFFERENT (cooking, not AI)
```

**Why It's Powerful for Briefly:**

1. **Find similar articles WITHOUT keyword matching**
   - "AI safety regulation" and "Governance framework for artificial intelligence" have no common words
   - But their embeddings are close in vector space
   - RAG retrieves this context automatically

2. **Clustering naturally groups related content**
   - K-means clustering on embeddings finds topic groups
   - No manual categorization needed

3. **Semantic search**
   - Search "model releases" finds articles about Gemini, Claude, GPT, Llama
   - Even if they never use the word "release"

4. **Context-aware classification**
   - Retrieve similar past articles to classify new ones consistently
   - "This article is similar to previous GenAI articles"

**How We Use It:**

1. **Generate embedding**: Send article summary to Gemini embedding API
   - Input: "Anthropic announces Claude 3.5 improvements..."
   - Output: 768-dimensional vector `[0.23, -0.45, ...]`

2. **Store in pgvector**: Save to PostgreSQL with vector index
   ```sql
   INSERT INTO article_embeddings (article_id, embedding)
   VALUES ('abc-123', '[0.23, -0.45, ...]');
   ```

3. **Similarity search**: Find nearest neighbors
   ```sql
   SELECT article_id, embedding <=> '[0.23, -0.45, ...]' AS distance
   FROM article_embeddings
   ORDER BY distance
   LIMIT 5;
   ```

4. **Use context**: Include similar articles when summarizing/classifying
   - "Here are 3 similar articles we covered before..."
   - LLM generates more consistent, informed outputs

**Why 768 dimensions?**
- Gemini's `text-embedding-004` model outputs 768-dim vectors
- More dimensions = more nuanced understanding
- Trade-off: storage space vs accuracy

**Vector Operations:**
- **Cosine similarity**: Measures angle between vectors (0 = identical, 1 = opposite)
- **Euclidean distance**: Straight-line distance in 768-D space
- **IVFFlat/HNSW indexes**: Fast approximate nearest neighbor search (millions of vectors)

---

## Version History

**v2.1 (2025-10-31):**
- Extracted from main design document
- Added Go concurrency patterns with worker pools
- Added structured output API implementation details
- Enhanced clustering with cluster-level summaries
