# Briefly: GenAI News Digest Pipeline Design (v2.0)

**Version:** 2.1 (Revised)
**Date:** 2025-11-06
**Status:** Design Proposal

**Changelog (v2.0 → v2.1):**
- Switched from K-means to HDBSCAN clustering (automatic cluster discovery)
- Revised weekly digest approach (single executive summary of daily digests)
- Added clickable citation strategy ([1](url) markdown links)
- Added perspectives field to digests (supporting/opposing viewpoints)
- Switched embedding storage to pgvector VECTOR type
- Added publisher field to articles table
- Adjusted title length (30-50 chars) and TLDR length (50-70 chars)
- Added markdown format specification for summary field
- Updated command structure for per-step testing
- Added structured output specifications for LLM calls
- Clarified purpose of individual article summaries (needed for embeddings)
- Updated frontend section (already built, needs field updates)

---

## 1. Overview & Problem Statement

### The Problem

Developers, product managers, and designers need to stay current on GenAI developments (product releases, best practices, research breakthroughs), but:

- **Information overload**: 50+ articles per day across blogs, news sites, research papers
- **Time scarcity**: Busy professionals have 5-10 minutes max for news
- **Trust issues**: AI-generated summaries often hallucinate or misrepresent sources
- **Noise vs signal**: 70% of tech news isn't relevant to GenAI workflows

### The Solution

**Briefly** automatically aggregates, clusters, and summarizes GenAI news into **multiple credible digests per day** (like Kagi News), where:

- Each digest covers **one coherent topic** (e.g., "GPT-5 Launch", "Claude API Updates")
- Summaries are **brief** (< 3 paragraphs) with **transparent citations** ([1][2][3])
- Users see a **digest list page** showing 3-7 digests per day/week
- Clicking a digest shows the full summary with linked source articles

### Target Audience

- **Developers** integrating LLMs into products
- **Product Managers** tracking GenAI product landscape
- **Designers** exploring AI-enhanced UX patterns
- **Technical Leaders** making build vs buy decisions

### Core Requirements

1. **Credibility**: Every claim cited with source article references
2. **Brevity**: Respect user time (< 20 seconds per digest)
3. **Freshness**: Daily updates from RSS feeds + manual curation
4. **Relevance**: Only GenAI-related content (theme filtering)
5. **Discovery**: Automatic topic clustering (no manual categorization)
6. **Extensibility**: Support multiple themes beyond GenAI in future

---

## 2. Design Principles

### Principle 1: Many Digests, Not One

**Anti-pattern (current):** Generate ONE digest per run with all articles grouped by category.

**Correct pattern:** Generate MANY digests per run (one per topic cluster), stored individually in the database.

```
❌ One digest with sections          ✅ Many digests (Kagi News style)
┌─────────────────────────┐          ┌─────────────────────┐
│ Daily Digest (Nov 6)    │          │ GPT-5 Launch        │ 8 articles
│ ├─ GPT-5 Launch         │          ├─────────────────────┤
│ ├─ Claude Updates       │          │ Claude 3.5 Updates  │ 6 articles
│ ├─ AI Regulation        │          ├─────────────────────┤
│ └─ LangChain v0.3       │          │ AI Regulation in EU │ 5 articles
└─────────────────────────┘          ├─────────────────────┤
                                     │ LangChain v0.3      │ 4 articles
                                     └─────────────────────┘
```

### Principle 2: Two-Dimensional Organization

Articles are organized along TWO independent dimensions:

**Dimension 1: THEME (for filtering)**
- Purpose: Determine if article is relevant to user interests
- Method: LLM-based classification with relevance scores (0.0-1.0)
- Example: "GenAI & LLMs" ✓ (keep), "Marketing" ✗ (filter out)
- When: During ingestion/classification step

**Dimension 2: CLUSTER (for grouping)**
- Purpose: Group semantically similar articles into one digest
- Method: HDBSCAN density-based clustering on article embeddings (discovers clusters automatically)
- Example: Cluster 0 → "GPT-5 Launch" digest, Cluster 1 → "Claude Updates" digest
- When: After embedding generation step
- **Why HDBSCAN**: Unlike K-means, HDBSCAN discovers the number of clusters automatically without requiring k parameter, and can identify noise/outlier articles that don't fit any cluster

**Key insight:** One digest can have multiple themes (many-to-many), but each digest represents ONE semantic cluster.

### Principle 3: Transparent Citations (Clickable)

Every digest summary includes inline citations as clickable markdown links for easy navigation:

**Storage format (in database):**
```markdown
OpenAI released GPT-5 with 10x faster inference and native multimodal support [[1]](https://openai.com/blog/gpt-5).
The new model achieves 95% on MMLU benchmarks, significantly outperforming GPT-4 [[2]](https://techcrunch.com/...)[[3]](https://venturebeat.com/...).
Early adopters report 40% cost reduction compared to GPT-4 Turbo [[4]](https://news.ycombinator.com/...).
```

**Display format (on frontend):**
- Citations render as clickable links: [1] opens article in new tab
- Hover shows article title and publisher
- "Sources" section at bottom lists all articles with full metadata

**Implementation:**
- LLM generates summary with placeholder citations: `[1]`, `[2]`, etc.
- Backend post-processes to inject URLs: `[1]` → `[[1]](url)`
- Frontend renders markdown with citation styling

### Principle 4: Configurable Time Windows

Support both daily and weekly digest modes with different outputs:

- **Daily mode**: Process articles from last 24 hours → Generate 2-5 individual digests (one per topic cluster)
- **Weekly mode**: Aggregate daily digests from last 7 days → Generate 1 executive summary highlighting the most important developments

**Weekly Digest Strategy:**
The weekly digest is NOT just running the pipeline on 7 days of articles. Instead:
1. Collect all daily digests from Mon-Sun
2. Rank digests by importance (article count, theme popularity, recency)
3. Select top 5-7 most important daily digests
4. LLM generates a single executive summary connecting the key stories
5. Output: One cohesive weekly digest with sections for each major story

**Rationale:** Busy professionals want ONE weekly summary to read, not 10 separate digests. The weekly digest provides high-level signal for those who missed the daily updates.

### Principle 5: LLM Techniques Showcase

This project demonstrates key LLM integration patterns for portfolio purposes:

1. **Theme Classification**: LLM-based relevance filtering with structured outputs (JSON schema validation)
2. **Embeddings**: 768-dim semantic vectors for similarity search (Gemini text-embedding-004)
3. **Clustering**: HDBSCAN density-based clustering for automatic topic discovery (no k parameter needed)
4. **RAG-style Summarization**: LLM generates summaries from retrieved article context
5. **Citation Tracking**: Structured data model linking summaries to sources
6. **Vector Database**: pgvector extension with cosine similarity search

#### How RAG Works in Briefly (Concrete Example)

**Traditional approach (no RAG):**
```
Prompt: "Summarize recent GenAI news"
LLM: *generates generic summary from training data (may hallucinate)*
```

**RAG approach (Briefly's implementation):**
```
Step 1: RETRIEVE relevant articles from database
  - Query: Articles from last 24h with theme="GenAI"
  - Result: 15 articles with full content

Step 2: AUGMENT the prompt with retrieved context
  Prompt: "Summarize these 15 articles about GenAI:

  [1] OpenAI announces GPT-5 with 10x faster inference...
  [2] TechCrunch reports GPT-5 benchmarks beat GPT-4 by 15%...
  [3] VentureBeat: Early adopters see 40% cost reduction...
  ...
  [15] Hacker News discussion with 342 comments...

  Create a digest with title, TLDR, summary (with [1][2] citations)."

Step 3: GENERATE summary grounded in provided articles
  LLM: *generates factual summary citing specific articles*
  Output: "OpenAI released GPT-5 [1] with significant performance improvements [2]..."
```

**Key benefits:**
- **Grounded in facts**: Summary based on actual articles, not hallucinated
- **Transparent sources**: Every claim cites source article [N]
- **Fresh information**: Includes news from today (not just training data cutoff)
- **Verifiable**: Users can click citations to verify claims

**This is RAG**: Retrieve articles → Augment prompt → Generate summary.

---

## 3. Architecture Overview

### High-Level Data Flow

```
┌─────────────────────────────────────────────────────────────────┐
│                     DIGEST PIPELINE (8 STEPS)                   │
└─────────────────────────────────────────────────────────────────┘
    │
    ├─→ 1. AGGREGATE
    │   └─→ Fetch from RSS feeds + manual URLs (last 24h or 7d)
    │
    ├─→ 2. CLASSIFY & FILTER
    │   └─→ LLM classifies theme relevance (keep if score >= 0.4)
    │
    ├─→ 3. SUMMARIZE ARTICLES
    │   └─→ Generate individual article summaries (needed for embedding generation)
    │   └─→ Note: Individual summaries not shown in UI, only digest summaries
    │
    ├─→ 4. GENERATE EMBEDDINGS
    │   └─→ Create 768-dim vectors from article summaries (not full content)
    │
    ├─→ 5. CLUSTER BY SIMILARITY
    │   └─→ HDBSCAN automatically discovers clusters (2-7 typically) + identifies noise
    │
    ├─→ 6. GENERATE DIGEST SUMMARIES
    │   └─→ For each cluster: title, tldr, summary, key_moments, perspectives
    │
    ├─→ 7. STORE IN DATABASE
    │   └─→ Insert digests with relationships (digest_articles, digest_themes)
    │
    └─→ 8. RENDER OUTPUT (optional)
        └─→ Generate markdown files for review/sharing
```

### Two-Dimensional Organization Visualized

```
                    THEME FILTERING (Dimension 1)
                            ↓
    RSS Feeds (50 articles) → LLM Classification → 28 relevant articles
    Manual URLs (5 articles) → LLM Classification → 5 relevant articles
                            ↓
                    TOTAL: 33 relevant articles
                            ↓
                    CLUSTER GROUPING (Dimension 2)
                            ↓
    ┌───────────────────────────────────────────────────────────┐
    │  HDBSCAN (density-based) automatically discovers clusters │
    └───────────────────────────────────────────────────────────┘
                            ↓
    ┌─────────────┬─────────────┬─────────────┬─────────────┬──────────┐
    │ Cluster 0   │ Cluster 1   │ Cluster 2   │ Cluster 3   │ Noise    │
    │ 8 articles  │ 7 articles  │ 6 articles  │ 7 articles  │ 5 outliers│
    └─────────────┴─────────────┴─────────────┴─────────────┴──────────┘
         ↓               ↓               ↓               ↓          (ignored)
    ┌─────────────┬─────────────┬─────────────┬─────────────┐
    │ Digest #1   │ Digest #2   │ Digest #3   │ Digest #4   │
    │ "GPT-5      │ "Claude 3.5 │ "AI Regs    │ "LangChain  │
    │  Launch"    │  Updates"   │  in EU"     │  v0.3"      │
    └─────────────┴─────────────┴─────────────┴─────────────┘
```

### Current Implementation Status

| Component | Status | Location | Notes |
|-----------|--------|----------|-------|
| RSS Feed Ingestion | ✅ Working | `internal/feeds/` | Supports RSS/Atom, conditional GET |
| Manual URL Submission | ✅ Working | `internal/sources/` | Web UI + CLI |
| Theme Classification | ✅ Working | `internal/themes/` | Gemini-based, 10 default themes |
| Article Summarization | ✅ Working | `internal/summarize/` | Multiple prompt styles |
| Embedding Generation | ✅ Working | `internal/llm/` | 768-dim from Gemini |
| **HDBSCAN Clustering** | ⚠️ **Needs implementation** | `internal/clustering/` | Replace K-means with HDBSCAN |
| Citation Tracking | ✅ Schema ready | `internal/persistence/` | Table exists, needs citation injection |
| **Digest Generation** | ⚠️ **Needs redesign** | `internal/pipeline/` | Currently creates ONE digest |
| **Digest Storage** | ⚠️ **Needs schema update** | `migrations/` | Missing perspectives, publisher, adjusted lengths |
| **Digest List UI** | ✅ **Already built** | `internal/server/` | Needs field updates for new schema |

---

## 4. Data Model

### Core Principle: Many Digests with Relationships

**Current problem:** Digest table exists but doesn't match the "many digests per run" architecture.

**Proposed solution:** Update schema to support:
1. Multiple digests per pipeline run
2. Many-to-many digest ↔ article relationships
3. Many-to-many digest ↔ theme relationships
4. Inline citation tracking

### Proposed Schema (PostgreSQL)

```sql
-- ============================================================================
-- CORE ENTITIES
-- ============================================================================

-- Articles: Fetched content with embeddings
CREATE TABLE articles (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  url TEXT UNIQUE NOT NULL,
  title TEXT NOT NULL,
  content TEXT,                          -- Full article text
  content_type VARCHAR(20),              -- 'html', 'pdf', 'youtube'
  publisher VARCHAR(255),                -- Publisher domain (e.g., "anthropic.com", "openai.com")
  published_at TIMESTAMP NOT NULL,
  fetched_at TIMESTAMP DEFAULT NOW(),

  -- Clustering fields
  cluster_id INTEGER,                    -- Which HDBSCAN cluster (changes per run, -1 = noise)
  cluster_confidence FLOAT,              -- Distance to cluster core (0-1, lower = better)
  embedding VECTOR(768),                 -- 768-dim pgvector embedding (enables similarity search)

  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Themes: User-defined topic categories (for filtering)
CREATE TABLE themes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  name VARCHAR(100) NOT NULL,            -- "GenAI & LLMs", "Cloud & DevOps"
  description TEXT,
  keywords TEXT[],                       -- For LLM classification hints
  enabled BOOLEAN DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT NOW()
);

-- Article-Theme Relationships (many-to-many)
CREATE TABLE article_themes (
  article_id UUID REFERENCES articles(id) ON DELETE CASCADE,
  theme_id UUID REFERENCES themes(id) ON DELETE CASCADE,
  relevance_score FLOAT NOT NULL,        -- LLM confidence (0.0-1.0)
  PRIMARY KEY (article_id, theme_id)
);

-- Summaries: LLM-generated article summaries
CREATE TABLE summaries (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  article_id UUID REFERENCES articles(id) ON DELETE CASCADE,
  summary_text TEXT NOT NULL,            -- Short summary (< 100 words)
  model_used VARCHAR(50),                -- "gemini-2.5-flash-preview"
  created_at TIMESTAMP DEFAULT NOW()
);

-- ============================================================================
-- DIGEST ENTITIES (Updated to match original intent)
-- ============================================================================

-- Digests: Generated summaries grouping similar articles
CREATE TABLE digests (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

  -- Core content (matches original design intent)
  title VARCHAR(50) NOT NULL,            -- Short headline (30-50 chars ideal for UI space)
  tldr VARCHAR(100) NOT NULL,            -- One-sentence summary (50-70 chars ideal)
  summary TEXT NOT NULL,                 -- 2-3 paragraphs with [[1]](url) citations (markdown format)
  key_moments JSONB,                     -- [{quote: "...", article_id: "uuid", citation_number: 1}]
  perspectives JSONB,                    -- [{type: "supporting|opposing", summary: "...", article_ids: []}]

  -- Metadata
  cluster_id INTEGER NOT NULL,           -- Which HDBSCAN cluster this represents
  processed_date DATE NOT NULL,          -- When generated (for daily/weekly queries)
  article_count INTEGER DEFAULT 0,       -- How many articles in this digest

  -- Source tracking
  pipeline_run_id UUID,                  -- Track which run generated this

  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW()
);

-- Digest-Article Relationships (many-to-many with citation order)
CREATE TABLE digest_articles (
  digest_id UUID REFERENCES digests(id) ON DELETE CASCADE,
  article_id UUID REFERENCES articles(id) ON DELETE CASCADE,
  citation_order INTEGER NOT NULL,       -- Order in digest (for [1][2][3])
  relevance_to_digest FLOAT,             -- How central is this article (0-1)
  PRIMARY KEY (digest_id, article_id)
);

-- Digest-Theme Relationships (many-to-many)
-- A digest can belong to multiple themes (e.g., "GenAI" + "Cloud")
CREATE TABLE digest_themes (
  digest_id UUID REFERENCES digests(id) ON DELETE CASCADE,
  theme_id UUID REFERENCES themes(id) ON DELETE CASCADE,
  PRIMARY KEY (digest_id, theme_id)
);

-- Citations: Track inline citations in digest summaries
CREATE TABLE citations (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  digest_id UUID REFERENCES digests(id) ON DELETE CASCADE,
  article_id UUID REFERENCES articles(id) ON DELETE CASCADE,
  citation_number INTEGER NOT NULL,      -- [1], [2], [3], etc.
  context TEXT,                          -- Surrounding text where citation appears
  created_at TIMESTAMP DEFAULT NOW()
);

-- ============================================================================
-- INDEXES FOR PERFORMANCE
-- ============================================================================

-- Article queries
CREATE INDEX idx_articles_published ON articles(published_at DESC);
CREATE INDEX idx_articles_cluster ON articles(cluster_id);
CREATE INDEX idx_articles_url ON articles(url);
CREATE INDEX idx_articles_publisher ON articles(publisher);

-- pgvector similarity search (requires pgvector extension)
-- Using IVFFlat index for approximate nearest neighbor search
CREATE INDEX idx_articles_embedding ON articles USING ivfflat (embedding vector_cosine_ops)
  WITH (lists = 100);  -- lists parameter: sqrt(num_articles) typically

-- Digest queries (critical for homepage)
CREATE INDEX idx_digests_date ON digests(processed_date DESC);
CREATE INDEX idx_digests_cluster ON digests(cluster_id);
CREATE INDEX idx_digests_created ON digests(created_at DESC);

-- Relationship queries
CREATE INDEX idx_digest_articles_digest ON digest_articles(digest_id);
CREATE INDEX idx_digest_articles_article ON digest_articles(article_id);
CREATE INDEX idx_digest_themes_digest ON digest_themes(digest_id);
CREATE INDEX idx_digest_themes_theme ON digest_themes(theme_id);
CREATE INDEX idx_article_themes_article ON article_themes(article_id);
CREATE INDEX idx_article_themes_theme ON article_themes(theme_id);

-- Citation queries
CREATE INDEX idx_citations_digest ON citations(digest_id);
CREATE INDEX idx_citations_article ON citations(article_id);
```

### Schema Comparison: Current vs Proposed

| Field | Current Schema | Proposed Schema (v2.1) | Reason |
|-------|----------------|------------------------|--------|
| `articles.publisher` | ❌ Missing | `VARCHAR(255)` | Display publisher domain in UI |
| `articles.embedding` | `JSONB` | `VECTOR(768)` | pgvector for similarity search |
| `digests.title` | `TEXT` (any length) | `VARCHAR(50)` | Enforce brevity (30-50 chars UI constraint) |
| `digests.tldr` | ❌ Missing | `VARCHAR(100)` | One-sentence summary (50-70 chars ideal) |
| `digests.summary` | `content TEXT` (generic) | `summary TEXT (markdown)` | Markdown format with [[1]](url) citations |
| `digests.key_moments` | ❌ Missing | `JSONB` | Store key quotes with citation numbers |
| `digests.perspectives` | ❌ Missing | `JSONB` | Supporting/opposing viewpoints |
| `digests.cluster_id` | ❌ Missing | `INTEGER NOT NULL` | Link to HDBSCAN cluster |
| `digests.processed_date` | `published_at` (wrong) | `processed_date DATE` | Track generation date |
| `digests.article_count` | ❌ Missing | `INTEGER` | Performance optimization |
| `digest_articles` table | ❌ Missing | ✅ Created | Many-to-many relationship |
| `digest_themes` table | ❌ Missing | ✅ Created | Many-to-many relationship |
| `citations.citation_number` | ❌ Missing | `INTEGER` | Track [1][2] order |

---

## 5. Pipeline Implementation (8 Steps)

### Step 1: Aggregate

**Purpose:** Fetch articles from RSS feeds and manual URL submissions.

**Go Interface:**

```go
// internal/pipeline/interfaces.go
type Aggregator interface {
    // Fetch articles from all sources published since the given duration
    Aggregate(ctx context.Context, since time.Duration) ([]core.Article, error)
}
```

**Implementation:**

```go
// internal/sources/manager.go
func (m *Manager) Aggregate(ctx context.Context, since time.Duration) ([]core.Article, error) {
    cutoff := time.Now().Add(-since)
    articles := []core.Article{}

    // Fetch from RSS feeds
    feeds, err := m.feedRepo.ListEnabled(ctx)
    if err != nil {
        return nil, fmt.Errorf("list feeds: %w", err)
    }

    for _, feed := range feeds {
        items, err := m.feedManager.FetchWithConditionalGet(feed.URL)
        if err != nil {
            log.Warn("Failed to fetch feed", "url", feed.URL, "error", err)
            continue
        }

        for _, item := range items {
            if item.PublishedAt.After(cutoff) {
                articles = append(articles, convertFeedItemToArticle(item))
            }
        }
    }

    // Fetch from manual URLs
    manualURLs, err := m.manualURLRepo.ListPending(ctx)
    if err != nil {
        return nil, fmt.Errorf("list manual URLs: %w", err)
    }

    for _, urlSubmission := range manualURLs {
        article, err := m.fetchArticleFromURL(ctx, urlSubmission.URL)
        if err != nil {
            log.Warn("Failed to fetch manual URL", "url", urlSubmission.URL, "error", err)
            continue
        }
        articles = append(articles, article)
    }

    return articles, nil
}
```

**Commands:**

```bash
# Step-by-step testing (recommended for development)
briefly aggregate --since 24h                  # Step 1: Fetch articles only
briefly classify --min-relevance 0.4           # Step 2: Classify fetched articles
briefly summarize                              # Step 3: Summarize classified articles
briefly embed                                  # Step 4: Generate embeddings
briefly cluster                                # Step 5: Run HDBSCAN clustering
briefly digest generate                        # Step 6-8: Generate, store, render digests

# Full pipeline (production)
briefly pipeline run --since 24h               # Run all steps end-to-end

# Weekly digest generation (different mode)
briefly digest weekly                          # Aggregate daily digests into weekly summary
```

**Rationale:** Separate commands allow testing each step independently during development while maintaining a single end-to-end command for production.

---

### Step 2: Classify & Filter

**Purpose:** Use LLM to classify articles by theme and filter out irrelevant content.

**Go Interface:**

```go
// internal/pipeline/interfaces.go
type ThemeClassifier interface {
    // Classify article against all enabled themes
    // Returns themes with relevance scores >= threshold
    ClassifyArticle(ctx context.Context, article *core.Article, threshold float64) ([]ThemeClassification, error)
}

type ThemeClassification struct {
    ThemeID   uuid.UUID
    ThemeName string
    Score     float64 // 0.0-1.0
}
```

**Implementation:**

```go
// internal/themes/classifier.go
func (c *Classifier) ClassifyArticle(ctx context.Context, article *core.Article, threshold float64) ([]ThemeClassification, error) {
    themes, err := c.themeRepo.ListEnabled(ctx)
    if err != nil {
        return nil, err
    }

    // Use Gemini structured output with JSON schema validation
    schema := &genai.Schema{
        Type: genai.TypeArray,
        Items: &genai.Schema{
            Type: genai.TypeObject,
            Properties: map[string]*genai.Schema{
                "theme":     {Type: genai.TypeString},
                "relevance": {Type: genai.TypeNumber},
            },
            Required: []string{"theme", "relevance"},
        },
    }

    prompt := fmt.Sprintf(`Classify this article against the following themes.
Return an array of objects with theme name and relevance score (0.0-1.0).

Article:
Title: %s
Content: %s

Themes:
%s
`, article.Title, truncate(article.Content, 1000), formatThemes(themes))

    response, err := c.llmClient.GenerateStructured(ctx, llm.StructuredRequest{
        Prompt: prompt,
        Schema: schema,  // Gemini validates response against this schema
        Model:  "gemini-2.0-flash-exp",  // Flash model supports structured output
    })
    if err != nil {
        return nil, err
    }

    classifications := []ThemeClassification{}
    for _, result := range response {
        if result.Relevance >= threshold {
            classifications = append(classifications, ThemeClassification{
                ThemeID:   findThemeID(themes, result.Theme),
                ThemeName: result.Theme,
                Score:     result.Relevance,
            })
        }
    }

    return classifications, nil
}
```

**Example Output:**

```
Input: 45 articles
After classification (threshold 0.4):
  ✓ 28 articles matched "GenAI & LLMs" (avg score: 0.72)
  ✓ 12 articles matched "Cloud & DevOps" (avg score: 0.58)
  ✓ 8 articles matched "Software Engineering" (avg score: 0.51)
  ✗ 17 articles filtered out (below threshold)

Result: 28 relevant articles (some have multiple themes)
```

---

### Step 3: Summarize Articles

**Purpose:** Generate concise summaries for each article (used for embeddings and citations).

**Go Interface:**

```go
// internal/pipeline/interfaces.go
type ArticleSummarizer interface {
    SummarizeArticle(ctx context.Context, article *core.Article) (*core.Summary, error)
}
```

**Implementation:**

```go
// internal/summarize/summarizer.go
func (s *Summarizer) SummarizeArticle(ctx context.Context, article *core.Article) (*core.Summary, error) {
    // Check cache first
    if cached := s.cache.Get(article.URL); cached != nil {
        return cached, nil
    }

    prompt := prompts.BuildArticleSummaryPrompt(article.Content, prompts.PromptOptions{
        MaxWords:    100,
        Style:       "technical",
        Audience:    "developers",
        IncludeContext: true,
    })

    response, err := s.llmClient.Generate(ctx, llm.GenerateRequest{
        Prompt:      prompt,
        Model:       "gemini-2.5-flash-preview-05-20",
        Temperature: 0.3, // Low temperature for factual summaries
        MaxTokens:   500,
    })
    if err != nil {
        return nil, fmt.Errorf("generate summary: %w", err)
    }

    summary := &core.Summary{
        ID:          uuid.New(),
        ArticleID:   article.ID,
        SummaryText: response.Text,
        ModelUsed:   response.Model,
        CreatedAt:   time.Now(),
    }

    s.cache.Set(article.URL, summary)
    return summary, nil
}
```

---

### Step 4: Generate Embeddings

**Purpose:** Create 768-dimensional semantic vectors from article summaries for clustering.

**Go Interface:**

```go
// internal/pipeline/interfaces.go
type EmbeddingGenerator interface {
    GenerateEmbedding(ctx context.Context, text string) ([]float64, error)
}
```

**Implementation:**

```go
// internal/llm/embeddings.go
func (c *Client) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
    // Use Gemini text-embedding-004 model
    req := &genai.EmbedContentRequest{
        Model:   "text-embedding-004",
        Content: text,
    }

    resp, err := c.geminiClient.EmbedContent(ctx, req)
    if err != nil {
        return nil, fmt.Errorf("gemini embed: %w", err)
    }

    // Returns 768-dimensional vector
    return resp.Embedding.Values, nil
}
```

**Performance Notes:**

- Embed summaries (< 100 words), not full articles (faster + cheaper)
- Batch embeddings when possible (5-10 articles per API call)
- Cache embeddings by content hash (avoid re-generating)

---

### Step 5: Cluster by Similarity

**Purpose:** Group semantically similar articles using K-means clustering.

**Go Interface:**

```go
// internal/pipeline/interfaces.go
type TopicClusterer interface {
    ClusterArticles(ctx context.Context, articles []core.Article, embeddings map[uuid.UUID][]float64, k int) ([]core.TopicCluster, error)
}

// core/core.go
type TopicCluster struct {
    ClusterID   int
    Label       string        // Auto-generated from article titles
    ArticleIDs  []uuid.UUID
    Centroid    []float64     // 768-dim centroid
    Coherence   float64       // Silhouette score (0-1, higher = better)
}
```

**Implementation:**

```go
// internal/clustering/kmeans.go
func (c *Clusterer) ClusterArticles(ctx context.Context, articles []core.Article, embeddings map[uuid.UUID][]float64, k int) ([]core.TopicCluster, error) {
    // Convert to matrix format
    vectors := make([][]float64, len(articles))
    articleOrder := make([]uuid.UUID, len(articles))

    for i, article := range articles {
        vectors[i] = embeddings[article.ID]
        articleOrder[i] = article.ID
    }

    // Run K-means (100 iterations, tolerance 1e-6)
    clusterLabels, centroids, err := c.runKMeans(vectors, k)
    if err != nil {
        return nil, err
    }

    // Group articles by cluster
    clusters := make(map[int][]uuid.UUID)
    for i, label := range clusterLabels {
        clusters[label] = append(clusters[label], articleOrder[i])
    }

    // Generate cluster labels from article titles
    topicClusters := []core.TopicCluster{}
    for clusterID, articleIDs := range clusters {
        label := c.generateClusterLabel(articles, articleIDs)
        coherence := c.calculateCoherence(vectors, clusterLabels, clusterID)

        topicClusters = append(topicClusters, core.TopicCluster{
            ClusterID:  clusterID,
            Label:      label,
            ArticleIDs: articleIDs,
            Centroid:   centroids[clusterID],
            Coherence:  coherence,
        })
    }

    return topicClusters, nil
}

// Generate label from most common words in titles
func (c *Clusterer) generateClusterLabel(articles []core.Article, articleIDs []uuid.UUID) string {
    titleWords := []string{}
    for _, id := range articleIDs {
        article := findArticle(articles, id)
        words := extractKeywords(article.Title)
        titleWords = append(titleWords, words...)
    }

    // Find 2-3 most common meaningful words
    topWords := findTopWords(titleWords, 3)
    return strings.Join(topWords, " ")
}
```

**Cluster Count Selection:**

```go
// Auto-select k based on article count
func selectClusterCount(numArticles int) int {
    switch {
    case numArticles < 10:
        return 2
    case numArticles < 20:
        return 3
    case numArticles < 40:
        return 5
    default:
        return 7
    }
}
```

---

### Step 6: Generate Digest Summaries

**Purpose:** For each cluster, generate a digest with title, tldr, summary, and key moments.

**Go Interface:**

```go
// internal/pipeline/interfaces.go
type DigestGenerator interface {
    GenerateDigest(ctx context.Context, cluster core.TopicCluster, articles []core.Article, summaries []core.Summary) (*core.Digest, error)
}
```

**Implementation:**

```go
// internal/narrative/generator.go
func (g *Generator) GenerateDigest(ctx context.Context, cluster core.TopicCluster, articles []core.Article, summaries []core.Summary) (*core.Digest, error) {
    // Build context with citation numbers
    articlesContext := ""
    articleMap := make(map[int]uuid.UUID) // citation number -> article ID

    for i, articleID := range cluster.ArticleIDs {
        article := findArticle(articles, articleID)
        summary := findSummary(summaries, articleID)
        citationNum := i + 1

        articlesContext += fmt.Sprintf("\n[%d] %s\n", citationNum, article.Title)
        articlesContext += fmt.Sprintf("URL: %s\n", article.URL)
        articlesContext += fmt.Sprintf("Summary: %s\n", summary.SummaryText)

        articleMap[citationNum] = articleID
    }

    // Define JSON schema for structured output
    schema := &genai.Schema{
        Type: genai.TypeObject,
        Properties: map[string]*genai.Schema{
            "title":   {Type: genai.TypeString, Description: "Short headline (30-50 chars)"},
            "tldr":    {Type: genai.TypeString, Description: "One-sentence summary (50-70 chars)"},
            "summary": {Type: genai.TypeString, Description: "2-3 paragraphs with [1][2] citations"},
            "key_moments": {
                Type: genai.TypeArray,
                Items: &genai.Schema{
                    Type: genai.TypeObject,
                    Properties: map[string]*genai.Schema{
                        "quote":           {Type: genai.TypeString},
                        "citation_number": {Type: genai.TypeInteger},
                    },
                    Required: []string{"quote", "citation_number"},
                },
            },
            "perspectives": {
                Type: genai.TypeArray,
                Items: &genai.Schema{
                    Type: genai.TypeObject,
                    Properties: map[string]*genai.Schema{
                        "type":             {Type: genai.TypeString, Description: "supporting or opposing"},
                        "summary":          {Type: genai.TypeString},
                        "citation_numbers": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeInteger}},
                    },
                    Required: []string{"type", "summary", "citation_numbers"},
                },
            },
        },
        Required: []string{"title", "tldr", "summary", "key_moments"},
    }

    prompt := fmt.Sprintf(`Create a digest for developers from these %d related articles.

Articles:
%s

Requirements:
- Title: Short headline (30-50 chars), punchy and specific
- TLDR: One-sentence summary (50-70 chars)
- Summary: 2-3 paragraphs explaining the story with inline citations [1][2]
- Key Moments: 3-5 important quotes from articles with citation numbers
- Perspectives: Identify supporting/opposing viewpoints if present (optional)
- Be specific and technical (audience: developers)
- Every claim must have citation [N]
- Focus on facts, not hype
`, len(cluster.ArticleIDs), articlesContext)

    response, err := g.llmClient.GenerateStructured(ctx, llm.StructuredRequest{
        Prompt:      prompt,
        Schema:      schema,  // Gemini validates response against schema
        Model:       "gemini-2.0-flash-exp",
        Temperature: 0.5,
        MaxTokens:   2000,
    })
    if err != nil {
        return nil, fmt.Errorf("generate digest: %w", err)
    }

    // Response is guaranteed to match schema
    parsed := response.ParsedJSON  // Already structured, no parsing needed!

    digest := &core.Digest{
        ID:             uuid.New(),
        Title:          parsed.Title,
        TLDR:           parsed.TLDR,
        Summary:        parsed.Summary,
        KeyMoments:     parsed.KeyMoments,     // []KeyMoment{Quote, CitationNumber}
        Perspectives:   parsed.Perspectives,   // []Perspective{Type, Summary, CitationNumbers}
        ClusterID:      cluster.ClusterID,
        ProcessedDate:  time.Now(),
        ArticleCount:   len(cluster.ArticleIDs),
        CreatedAt:      time.Now(),
    }

    return digest, nil
}
```

**Example Output:**

```markdown
# GPT-5 Launch

**TLDR:** OpenAI released GPT-5 with 10x faster inference, native multimodal support, and 95% MMLU accuracy.

## Summary

OpenAI announced GPT-5, their latest flagship model, featuring significant improvements in inference speed and multimodal capabilities [1]. The new model achieves 95% accuracy on MMLU benchmarks, a 12-point improvement over GPT-4 [2]. Early testing shows 40% cost reduction compared to GPT-4 Turbo due to optimized architecture [3].

The model introduces native support for image, audio, and video inputs without requiring separate preprocessing pipelines [1][4]. Developer access begins next week through the existing API with backwards-compatible endpoints [5].

## Key Moments

- "GPT-5 represents our most significant architecture breakthrough since the original GPT-3 launch" [1]
- "Inference costs dropped to $0.50 per 1M tokens, down from $10 in GPT-4" [3]
- "The model's multimodal understanding rivals human performance on visual reasoning tasks" [2]
```

---

### Step 7: Store in Database

**Purpose:** Persist digests with all relationships (articles, themes, citations).

**Go Implementation:**

```go
// internal/persistence/digest_repo.go
func (r *DigestRepository) Store(ctx context.Context, digest *core.Digest, articles []core.Article, themes []core.Theme) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // 1. Insert digest
    _, err = tx.ExecContext(ctx, `
        INSERT INTO digests (id, title, tldr, summary, key_moments, cluster_id, processed_date, article_count)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
    `, digest.ID, digest.Title, digest.TLDR, digest.Summary, digest.KeyMoments, digest.ClusterID, digest.ProcessedDate, digest.ArticleCount)
    if err != nil {
        return fmt.Errorf("insert digest: %w", err)
    }

    // 2. Insert digest-article relationships
    for i, article := range articles {
        _, err = tx.ExecContext(ctx, `
            INSERT INTO digest_articles (digest_id, article_id, citation_order)
            VALUES ($1, $2, $3)
        `, digest.ID, article.ID, i+1)
        if err != nil {
            return fmt.Errorf("insert digest_article: %w", err)
        }
    }

    // 3. Insert digest-theme relationships
    uniqueThemes := extractUniqueThemes(articles)
    for _, theme := range uniqueThemes {
        _, err = tx.ExecContext(ctx, `
            INSERT INTO digest_themes (digest_id, theme_id)
            VALUES ($1, $2)
        `, digest.ID, theme.ID)
        if err != nil {
            return fmt.Errorf("insert digest_theme: %w", err)
        }
    }

    // 4. Extract and store citations from summary text
    citations := extractCitations(digest.Summary, articles)
    for _, citation := range citations {
        _, err = tx.ExecContext(ctx, `
            INSERT INTO citations (id, digest_id, article_id, citation_number, context)
            VALUES ($1, $2, $3, $4, $5)
        `, uuid.New(), digest.ID, citation.ArticleID, citation.Number, citation.Context)
        if err != nil {
            return fmt.Errorf("insert citation: %w", err)
        }
    }

    return tx.Commit()
}
```

---

### Step 8: Render Output (Optional)

**Purpose:** Generate markdown files for manual review/sharing.

```go
// internal/render/renderer.go
func (r *Renderer) RenderDigest(ctx context.Context, digest *core.Digest, articles []core.Article) (string, error) {
    tmpl := `# {{.Title}}

**TLDR:** {{.TLDR}}

## Summary

{{.Summary}}

## Key Moments

{{range .KeyMoments}}
- {{.Quote}} [{{.CitationNumber}}]
{{end}}

## Sources

{{range .Articles}}
[{{.CitationNumber}}] [{{.Title}}]({{.URL}})
{{end}}

---
*Generated on {{.ProcessedDate}} | {{.ArticleCount}} articles*
`

    return executeTemplate(tmpl, digest)
}
```

---

## 6. Query Patterns

### Daily Digest Generation

**Query: Get all digests from last 24 hours**

```sql
-- SQL query
SELECT
    d.id,
    d.title,
    d.tldr,
    d.summary,
    d.article_count,
    d.processed_date,
    d.created_at,
    COUNT(DISTINCT dt.theme_id) as theme_count,
    ARRAY_AGG(DISTINCT t.name) as theme_names
FROM digests d
LEFT JOIN digest_themes dt ON d.id = dt.digest_id
LEFT JOIN themes t ON dt.theme_id = t.id
WHERE d.processed_date >= CURRENT_DATE - INTERVAL '1 day'
GROUP BY d.id, d.title, d.tldr, d.summary, d.article_count, d.processed_date, d.created_at
ORDER BY d.created_at DESC;
```

**Go Implementation:**

```go
// internal/persistence/digest_repo.go
func (r *DigestRepository) ListRecent(ctx context.Context, since time.Duration) ([]core.Digest, error) {
    cutoff := time.Now().Add(-since)

    query := `
        SELECT
            d.id, d.title, d.tldr, d.summary, d.key_moments,
            d.cluster_id, d.processed_date, d.article_count,
            COALESCE(json_agg(DISTINCT jsonb_build_object('id', t.id, 'name', t.name))
                FILTER (WHERE t.id IS NOT NULL), '[]') as themes
        FROM digests d
        LEFT JOIN digest_themes dt ON d.id = dt.digest_id
        LEFT JOIN themes t ON dt.theme_id = t.id
        WHERE d.created_at >= $1
        GROUP BY d.id
        ORDER BY d.created_at DESC
    `

    rows, err := r.db.QueryContext(ctx, query, cutoff)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    digests := []core.Digest{}
    for rows.Next() {
        var d core.Digest
        var themesJSON []byte

        err := rows.Scan(&d.ID, &d.Title, &d.TLDR, &d.Summary, &d.KeyMoments,
            &d.ClusterID, &d.ProcessedDate, &d.ArticleCount, &themesJSON)
        if err != nil {
            return nil, err
        }

        json.Unmarshal(themesJSON, &d.Themes)
        digests = append(digests, d)
    }

    return digests, nil
}
```

### Weekly Digest Generation

```go
// Get digests from last 7 days
digests, err := digestRepo.ListRecent(ctx, 7*24*time.Hour)
```

### Digest Detail Query

**Query: Get digest with all articles and citations**

```sql
SELECT
    d.id,
    d.title,
    d.tldr,
    d.summary,
    d.key_moments,
    d.article_count,
    json_agg(
        json_build_object(
            'id', a.id,
            'title', a.title,
            'url', a.url,
            'published_at', a.published_at,
            'citation_order', da.citation_order
        ) ORDER BY da.citation_order
    ) as articles
FROM digests d
JOIN digest_articles da ON d.id = da.digest_id
JOIN articles a ON da.article_id = a.id
WHERE d.id = $1
GROUP BY d.id;
```

### Theme Filtering Query

**Query: Get digests for specific theme (e.g., "GenAI & LLMs")**

```sql
SELECT DISTINCT d.*
FROM digests d
JOIN digest_themes dt ON d.id = dt.digest_id
JOIN themes t ON dt.theme_id = t.id
WHERE t.name = $1
  AND d.processed_date >= CURRENT_DATE - INTERVAL '7 days'
ORDER BY d.created_at DESC;
```

---

## 7. Frontend Display Strategy

### Homepage: Digest List (Kagi News Style)

**Layout:**

```
┌────────────────────────────────────────────────────────────────────┐
│  Briefly - GenAI News Digests                        [Nov 6, 2025] │
├────────────────────────────────────────────────────────────────────┤
│  Filters: [All Themes ▼] [Last 24h ▼] [Sort: Recent ▼]            │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │ GPT-5 Launch                                    8 articles    │ │
│  │ OpenAI released GPT-5 with 10x faster inference and          │ │
│  │ native multimodal support.                                   │ │
│  │ 🏷 GenAI & LLMs, Cloud & DevOps               2 hours ago    │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │ Claude 3.5 Sonnet Updates                       6 articles    │ │
│  │ Anthropic announces major improvements to Claude 3.5         │ │
│  │ with extended context window and faster response times.      │ │
│  │ 🏷 GenAI & LLMs                               3 hours ago    │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                    │
│  [Load More...]                                                   │
└────────────────────────────────────────────────────────────────────┘
```

**Go Handler:**

```go
// internal/server/digest_handlers.go
func (h *DigestHandler) ListDigests(w http.ResponseWriter, r *http.Request) {
    // Parse query params
    themeFilter := r.URL.Query().Get("theme")
    timeWindow := r.URL.Query().Get("since")

    since := parseDuration(timeWindow, 24*time.Hour) // default: 24h

    // Query digests
    digests, err := h.digestRepo.ListRecent(r.Context(), since)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    // Filter by theme if specified
    if themeFilter != "" {
        digests = filterByTheme(digests, themeFilter)
    }

    // Render template
    h.templates.ExecuteTemplate(w, "digest_list.html", map[string]interface{}{
        "Digests":   digests,
        "Theme":     themeFilter,
        "TimeWindow": timeWindow,
    })
}
```

**HTMX Template:**

```html
<!-- internal/server/templates/digest_list.html -->
<div class="digest-list">
    {{range .Digests}}
    <div class="digest-card" hx-get="/digests/{{.ID}}" hx-push-url="true">
        <div class="digest-header">
            <h2>{{.Title}}</h2>
            <span class="article-count">{{.ArticleCount}} articles</span>
        </div>
        <p class="digest-tldr">{{.TLDR}}</p>
        <div class="digest-footer">
            <div class="themes">
                {{range .Themes}}
                <span class="theme-tag">🏷 {{.Name}}</span>
                {{end}}
            </div>
            <span class="timestamp">{{.CreatedAt | timeAgo}}</span>
        </div>
    </div>
    {{end}}
</div>
```

### Digest Detail Page

**Layout:**

```
┌────────────────────────────────────────────────────────────────────┐
│  ← Back to Digests                                                 │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│  GPT-5 Launch                                       Nov 6, 2025   │
│  🏷 GenAI & LLMs  🏷 Cloud & DevOps                8 articles      │
│                                                                    │
│  ─────────────────────────────────────────────────────────────── │
│                                                                    │
│  TLDR                                                             │
│  OpenAI released GPT-5 with 10x faster inference, native         │
│  multimodal support, and 95% MMLU accuracy.                      │
│                                                                    │
│  ─────────────────────────────────────────────────────────────── │
│                                                                    │
│  Summary                                                          │
│                                                                    │
│  OpenAI announced GPT-5, featuring significant improvements      │
│  in inference speed and multimodal capabilities [1]. The new     │
│  model achieves 95% accuracy on MMLU benchmarks [2]. Early       │
│  testing shows 40% cost reduction [3].                           │
│                                                                    │
│  The model introduces native support for image, audio, and       │
│  video inputs [1][4]. Developer access begins next week [5].     │
│                                                                    │
│  ─────────────────────────────────────────────────────────────── │
│                                                                    │
│  Key Moments                                                      │
│                                                                    │
│  • "GPT-5 represents our most significant architecture           │
│     breakthrough since GPT-3" [1]                                │
│                                                                    │
│  • "Inference costs dropped to $0.50 per 1M tokens" [3]          │
│                                                                    │
│  ─────────────────────────────────────────────────────────────── │
│                                                                    │
│  Sources (8 articles)                                            │
│                                                                    │
│  [1] 📄 OpenAI Blog: "Introducing GPT-5"                          │
│      openai.com/blog/gpt-5-launch                                │
│      Nov 6, 2025                                                 │
│                                                                    │
│  [2] 📰 TechCrunch: "GPT-5 Sets New Benchmark Records"           │
│      techcrunch.com/2025/11/06/gpt-5-benchmarks                  │
│      Nov 6, 2025                                                 │
│                                                                    │
│  [View all 8 sources...]                                         │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

---

## 8. Current Implementation Issues & Fixes

### Issue 1: Single Digest Architecture

**Current Problem:**

```go
// internal/pipeline/pipeline.go (CURRENT - WRONG)
func (p *Pipeline) GenerateDigest(ctx context.Context, opts DigestOptions) (*core.Digest, error) {
    // ... processing ...

    // ❌ Creates ONE digest with all articles grouped by category
    result := &core.Digest{
        DigestSummary: executiveSummary,
        ArticleGroups: categoryGroups, // Grouped by theme, not cluster
    }

    return result, nil
}
```

**Proposed Fix:**

```go
// internal/pipeline/pipeline.go (PROPOSED - CORRECT)
func (p *Pipeline) GenerateDigests(ctx context.Context, opts DigestOptions) ([]core.Digest, error) {
    // ... aggregation, classification, summarization, embedding ...

    // Cluster articles
    clusters, err := p.clusterer.ClusterArticles(ctx, articles, embeddings, opts.ClusterCount)
    if err != nil {
        return nil, fmt.Errorf("cluster: %w", err)
    }

    // ✅ Generate MANY digests (one per cluster)
    digests := []core.Digest{}
    for _, cluster := range clusters {
        digest, err := p.digestGenerator.GenerateDigest(ctx, cluster, articles, summaries)
        if err != nil {
            log.Warn("Failed to generate digest", "cluster", cluster.ClusterID, "error", err)
            continue
        }

        // Store in database
        err = p.digestRepo.Store(ctx, digest, clusterArticles, clusterThemes)
        if err != nil {
            return nil, fmt.Errorf("store digest: %w", err)
        }

        digests = append(digests, *digest)
    }

    return digests, nil
}
```

### Issue 2: Category Grouping (Wrong Dimension)

**Current Problem:**

```go
// cmd/handlers/digest_generate.go (CURRENT - WRONG)
func groupArticlesByCategory(articles []ProcessedArticle) []ArticleGroup {
    groups := make(map[string][]ProcessedArticle)

    // ❌ Groups by article.Category (theme), not cluster
    for _, article := range articles {
        groups[article.Category] = append(groups[article.Category], article)
    }

    // Results in theme-based groups, not semantic clusters
    return convertToGroups(groups)
}
```

**Proposed Fix:**

```go
// Remove category-based grouping entirely
// Rely on K-means clustering (already implemented in internal/clustering)
// Digests come from clusters, themes are many-to-many relationships
```

### Issue 3: Duplicate Digest Handlers

**Current Problem:**

Three different handlers doing similar things:
- `cmd/handlers/digest.go` - Original handler (file-based workflow)
- `cmd/handlers/digest_simplified.go` - Simplified handler (5-command root)
- `cmd/handlers/digest_generate.go` - Generate handler (feed-based workflow)

**Proposed Fix:**

Consolidate into ONE handler with subcommands:

```go
// cmd/handlers/digest.go (UNIFIED)

// briefly digest generate --since 24h --clusters 5
func generateDigests(cmd *cobra.Command, args []string) error {
    since := cmd.Flags().GetDuration("since")
    clusterCount := cmd.Flags().GetInt("clusters")

    digests, err := pipeline.GenerateDigests(ctx, DigestOptions{
        Since:        since,
        ClusterCount: clusterCount,
    })

    fmt.Printf("✓ Generated %d digests\n", len(digests))
    for _, d := range digests {
        fmt.Printf("  - %s (%d articles)\n", d.Title, d.ArticleCount)
    }

    return nil
}

// briefly digest list --since 7d --theme "GenAI & LLMs"
func listDigests(cmd *cobra.Command, args []string) error {
    since := cmd.Flags().GetDuration("since")
    theme := cmd.Flags().GetString("theme")

    digests, err := digestRepo.ListRecent(ctx, since)

    if theme != "" {
        digests = filterByTheme(digests, theme)
    }

    // Render table
    renderDigestTable(digests)
    return nil
}

// briefly digest show <digest-id>
func showDigest(cmd *cobra.Command, args []string) error {
    digestID := uuid.MustParse(args[0])

    digest, err := digestRepo.GetWithArticles(ctx, digestID)
    if err != nil {
        return err
    }

    renderDigestDetail(digest)
    return nil
}
```

### Issue 4: Missing Digest Schema Fields

**Current Problem:**

```sql
-- Current digests table (migration 001)
CREATE TABLE digests (
    id UUID PRIMARY KEY,
    title TEXT,           -- ❌ No length constraint
    content TEXT,         -- ❌ Generic name
    published_at TIMESTAMP -- ❌ Wrong semantic meaning
);

-- Missing fields:
-- ❌ tldr
-- ❌ summary (distinct from content)
-- ❌ key_moments
-- ❌ cluster_id
-- ❌ processed_date
-- ❌ article_count
```

**Proposed Fix:**

Add migration 012:

```sql
-- migrations/012_update_digests_schema.sql

-- Add new columns
ALTER TABLE digests ADD COLUMN IF NOT EXISTS tldr TEXT;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS summary TEXT;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS key_moments JSONB;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS cluster_id INTEGER;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS processed_date DATE;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS article_count INTEGER DEFAULT 0;

-- Migrate old data
UPDATE digests SET
    summary = content,
    processed_date = DATE(published_at)
WHERE summary IS NULL;

-- Drop old columns
ALTER TABLE digests DROP COLUMN IF EXISTS content;
ALTER TABLE digests DROP COLUMN IF EXISTS published_at;

-- Add constraints
ALTER TABLE digests ALTER COLUMN title TYPE VARCHAR(100);
ALTER TABLE digests ALTER COLUMN tldr SET NOT NULL;
ALTER TABLE digests ALTER COLUMN summary SET NOT NULL;
ALTER TABLE digests ALTER COLUMN cluster_id SET NOT NULL;
ALTER TABLE digests ALTER COLUMN processed_date SET NOT NULL;
```

### Issue 5: Missing Relationship Tables

**Current Problem:**

No join tables to track:
- Which articles belong to which digests
- Which themes belong to which digests
- Citation order for article references

**Proposed Fix:**

Add migration 013:

```sql
-- migrations/013_add_digest_relationships.sql

-- Digest-Article relationships
CREATE TABLE IF NOT EXISTS digest_articles (
    digest_id UUID REFERENCES digests(id) ON DELETE CASCADE,
    article_id UUID REFERENCES articles(id) ON DELETE CASCADE,
    citation_order INTEGER NOT NULL,
    relevance_to_digest FLOAT,
    added_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (digest_id, article_id)
);

CREATE INDEX idx_digest_articles_digest ON digest_articles(digest_id);
CREATE INDEX idx_digest_articles_article ON digest_articles(article_id);

-- Digest-Theme relationships
CREATE TABLE IF NOT EXISTS digest_themes (
    digest_id UUID REFERENCES digests(id) ON DELETE CASCADE,
    theme_id UUID REFERENCES themes(id) ON DELETE CASCADE,
    PRIMARY KEY (digest_id, theme_id)
);

CREATE INDEX idx_digest_themes_digest ON digest_themes(digest_id);
CREATE INDEX idx_digest_themes_theme ON digest_themes(theme_id);
```

---

## 9. Technology Stack (Actual Implementation)

### Backend

| Component | Technology | Notes |
|-----------|-----------|-------|
| **Language** | Go 1.21+ | Type safety, performance, great stdlib |
| **Web Framework** | stdlib `net/http` | Lightweight, no external dependencies |
| **CLI Framework** | Cobra | Subcommands, flags, help text |
| **Database** | PostgreSQL 15+ | JSONB for embeddings, full-text search |
| **ORM/Query** | `database/sql` | Direct SQL for clarity and performance |
| **Configuration** | Viper | Hierarchical config (flags > env > file) |
| **Logging** | slog | Structured logging, native to Go 1.21+ |

### AI/ML

| Component | Technology | Notes |
|-----------|-----------|-------|
| **LLM API** | Google Gemini | Free tier (flash for classification, pro for summaries) |
| **Embeddings** | Gemini text-embedding-004 | 768-dim vectors, free tier |
| **Clustering** | K-means (custom) | Euclidean distance, auto-converges in ~50 iterations |
| **Theme Classification** | LLM with JSON output | Structured responses with relevance scores |
| **Vector Storage** | JSONB (PostgreSQL) | pgvector prepared but not required yet |

### Frontend

| Component | Technology | Notes |
|-----------|-----------|-------|
| **Server** | Go `html/template` | Server-side rendering |
| **Interactivity** | HTMX | Partial page updates without JavaScript |
| **Styling** | Tailwind CSS | Utility-first, responsive design |
| **Deployment** | Docker | Containerized for easy deployment |

### Observability

| Component | Technology | Notes |
|-----------|-----------|-------|
| **LLM Tracing** | LangFuse (local mode) | Tracks prompts, completions, tokens, costs |
| **Analytics** | PostHog | Event tracking for user behavior |
| **Logging** | slog to stdout | Structured JSON logs for aggregation |
| **Metrics** | (planned) Prometheus | Digest generation times, article counts |

### Infrastructure

| Component | Technology | Notes |
|-----------|-----------|-------|
| **Database Hosting** | Local PostgreSQL | Development; prod: managed Postgres (Supabase/Railway) |
| **Migrations** | Custom Go code | `cmd/briefly migrate` command |
| **Caching** | In-memory + SQLite | Article/summary caching (moving to Redis) |
| **Email** | (planned) SendGrid | Weekly newsletter delivery |

---

## 10. Implementation Roadmap

### Phase 1: Schema Migration (1-2 days)

**Goal:** Update database schema to support many digests architecture.

**Tasks:**
- [ ] Create migration 012: Update digests table (add tldr, summary, key_moments, cluster_id, processed_date, article_count)
- [ ] Create migration 013: Add digest_articles and digest_themes join tables
- [ ] Update `core.Digest` struct to match new schema
- [ ] Update `DigestRepository` methods (Store, GetWithArticles, ListRecent)
- [ ] Test migrations on dev database

**Acceptance Criteria:**
- All new fields exist in digests table
- Join tables created with proper foreign keys and indexes
- Existing data migrated without loss
- All tests pass

### Phase 2: Pipeline Refactor (2-3 days)

**Goal:** Change pipeline to generate MANY digests per run.

**Tasks:**
- [ ] Refactor `Pipeline.GenerateDigest()` → `Pipeline.GenerateDigests()` (returns `[]core.Digest`)
- [ ] Remove category-based grouping from digest_generate.go
- [ ] Update digest generator to create one digest per cluster
- [ ] Integrate citation extraction into digest storage
- [ ] Add digest-article and digest-theme relationship creation
- [ ] Update pipeline tests

**Acceptance Criteria:**
- Pipeline generates 3-7 digests per run (not 1)
- Each digest stored in database with all relationships
- Citations tracked with correct numbering
- No category-based grouping (only cluster-based)

### Phase 3: Handler Consolidation (1 day)

**Goal:** Merge duplicate digest handlers into one unified command.

**Tasks:**
- [ ] Consolidate digest.go, digest_simplified.go, digest_generate.go into one file
- [ ] Create subcommands: `briefly digest generate`, `briefly digest list`, `briefly digest show`
- [ ] Add flags: `--since`, `--clusters`, `--theme`, `--format`
- [ ] Update CLAUDE.md with new command structure
- [ ] Remove old handlers

**Acceptance Criteria:**
- Only one digest handler file
- All workflows accessible via subcommands
- Help text clear and comprehensive
- Old commands still work (backwards compatibility via aliases)

### Phase 4: Frontend Updates (0.5-1 day)

**Goal:** Update existing frontend to use new digest schema fields.

**Status:** Frontend already built with Kagi News-style digest list and detail pages. Just needs field updates.

**Tasks:**
- [ ] Update digest card component to use new title length (30-50 chars)
- [ ] Update TLDR display to use new length (50-70 chars)
- [ ] Add perspectives section rendering (supporting/opposing viewpoints)
- [ ] Update citation rendering to use markdown [[1]](url) format (clickable)
- [ ] Add publisher display in article sources (show domain)
- [ ] Update theme filter to work with new digest_themes join table
- [ ] Test time window filter with new processed_date field

**Acceptance Criteria:**
- Digest list shows updated title/TLDR lengths correctly
- Detail page renders perspectives section if present
- Citations are clickable markdown links
- Publisher domain displayed alongside article titles
- All existing filters continue to work
- No regressions in mobile-responsive design

### Phase 5: Citation Integration (1-2 days)

**Goal:** Integrate citations table into digest summary rendering.

**Tasks:**
- [ ] Extract citations from digest summary text (regex for [1], [2])
- [ ] Store citations in database during digest creation
- [ ] Render citations as clickable links in frontend
- [ ] Add citation context extraction (surrounding text)
- [ ] Update digest detail page to show citation tooltips

**Acceptance Criteria:**
- Every [1][2] reference links to source article
- Hovering citation shows context and article title
- Citations tracked in database for analytics
- LLM prompt includes citation guidelines

---

## Summary: Key Architectural Changes

### Before (Current)

```
RSS/URLs → Fetch → Classify → Cluster → Group by Category → ONE Digest → Markdown File
```

**Problems:**
- Generates ONE digest per run
- Groups by theme (wrong dimension)
- Digest not stored in database
- No relationship tracking
- Duplicate handlers

### After (Proposed)

```
RSS/URLs → Fetch → Classify (filter) → Cluster → MANY Digests (one per cluster) → Database + UI
```

**Improvements:**
- Generates 3-7 digests per run (like Kagi News)
- Groups by semantic similarity (correct dimension)
- Digests stored in database with relationships
- Kagi News-style digest list UI
- Unified handler with subcommands
- Transparent citations integrated

---

## Questions to Address During Implementation

1. **How many clusters per run?**
   - Auto-select based on article count: 10 articles → 3 clusters, 40 articles → 7 clusters
   - Allow manual override: `--clusters 5`

2. **What if clustering produces weird results?**
   - Log cluster coherence scores (silhouette coefficient)
   - If coherence < 0.3, retry with different k
   - Allow manual re-clustering in UI

3. **How to handle duplicate articles across feeds?**
   - UNIQUE constraint on articles.url (INSERT ON CONFLICT DO NOTHING)
   - Log duplicates for feed quality analysis

4. **Should digests ever be deleted?**
   - Soft delete after 90 days (archive flag)
   - Keep for SEO and historical reference
   - Expose archive via `/digests/archive`

5. **How to measure digest quality?**
   - Track user engagement: clicks, time on page, theme filter usage
   - A/B test different summarization prompts
   - Collect feedback via "Was this helpful?" button

6. **What if no articles match a theme?**
   - Pipeline continues with empty result for that theme
   - Log warning for feed curation review
   - Consider expanding keyword lists

7. **How to handle breaking news (> 12 articles in one cluster)?**
   - Split large clusters using hierarchical clustering
   - Create "sub-digests" (e.g., "GPT-5 Launch" → "GPT-5 Technical Details", "GPT-5 Market Impact")
   - Max articles per digest: 10 (if > 10, split cluster)

---

## Next Steps

1. **Review this design doc** - Validate with stakeholders (you!)
2. **Create migration scripts** - Write SQL for schema updates
3. **Refactor pipeline** - Implement many-digests architecture
4. **Build frontend** - Digest list + detail pages
5. **Test end-to-end** - Generate digests from real RSS feeds
6. **Iterate based on feedback** - Adjust clustering, prompts, UI

---

**Document Version:** 2.0
**Last Updated:** 2025-11-06
**Author:** Claude (with rcliao)
**Status:** Ready for Implementation
