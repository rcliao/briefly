# Migration Plan: Current → Many-Digests Architecture

**Goal:** Transform the current single-digest architecture into a many-digests-per-run architecture (like Kagi News).

**Timeline:** 7-10 days

**Risk Level:** Medium (breaking changes to database schema and API)

---

## Pre-Migration Checklist

- [ ] **Backup production database** (if applicable)
- [ ] **Tag current codebase** as `v3.0-before-many-digests`
- [ ] **Create migration branch**: `git checkout -b feature/many-digests-architecture`
- [ ] **Document current digest count**: Check how many digests exist in current DB
- [ ] **Review all places where `core.Digest` is used** (grep for references)

---

## Phase 1: Database Schema Migration (Days 1-2)

### Step 1.1: Create Migration 012 (Update Digests Table)

**File:** `internal/persistence/migrations/012_update_digests_schema.sql`

```sql
-- migrations/012_update_digests_schema.sql
-- Add new columns to digests table

BEGIN;

-- Add new required fields
ALTER TABLE digests ADD COLUMN IF NOT EXISTS tldr TEXT;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS summary TEXT;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS key_moments JSONB;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS cluster_id INTEGER;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS processed_date DATE;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS article_count INTEGER DEFAULT 0;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS pipeline_run_id UUID;

-- Migrate existing data (if any digests exist)
UPDATE digests SET
    summary = COALESCE(content, ''),
    processed_date = DATE(COALESCE(published_at, created_at)),
    cluster_id = 0,  -- default cluster for old digests
    article_count = 0,
    tldr = SUBSTRING(COALESCE(content, ''), 1, 200)  -- generate TLDR from content
WHERE summary IS NULL;

-- Drop old columns (after data migration)
ALTER TABLE digests DROP COLUMN IF EXISTS content;
ALTER TABLE digests DROP COLUMN IF EXISTS published_at;

-- Adjust column types
ALTER TABLE digests ALTER COLUMN title TYPE VARCHAR(100);

-- Add NOT NULL constraints (after data migration)
ALTER TABLE digests ALTER COLUMN tldr SET NOT NULL;
ALTER TABLE digests ALTER COLUMN summary SET NOT NULL;
ALTER TABLE digests ALTER COLUMN cluster_id SET NOT NULL;
ALTER TABLE digests ALTER COLUMN processed_date SET NOT NULL;

-- Add indexes
CREATE INDEX IF NOT EXISTS idx_digests_date ON digests(processed_date DESC);
CREATE INDEX IF NOT EXISTS idx_digests_cluster ON digests(cluster_id);

COMMIT;
```

**Run migration:**

```bash
briefly migrate --target 012
```

**Verify:**

```sql
-- Check schema
\d digests

-- Verify data migrated
SELECT id, title, tldr, cluster_id, processed_date FROM digests LIMIT 5;
```

---

### Step 1.2: Create Migration 013 (Add Relationship Tables)

**File:** `internal/persistence/migrations/013_add_digest_relationships.sql`

```sql
-- migrations/013_add_digest_relationships.sql
-- Create join tables for digest relationships

BEGIN;

-- Digest-Article relationships (many-to-many)
CREATE TABLE IF NOT EXISTS digest_articles (
    digest_id UUID REFERENCES digests(id) ON DELETE CASCADE,
    article_id UUID REFERENCES articles(id) ON DELETE CASCADE,
    citation_order INTEGER NOT NULL,
    relevance_to_digest FLOAT,
    added_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (digest_id, article_id)
);

-- Digest-Theme relationships (many-to-many)
CREATE TABLE IF NOT EXISTS digest_themes (
    digest_id UUID REFERENCES digests(id) ON DELETE CASCADE,
    theme_id UUID REFERENCES themes(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (digest_id, theme_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_digest_articles_digest ON digest_articles(digest_id);
CREATE INDEX IF NOT EXISTS idx_digest_articles_article ON digest_articles(article_id);
CREATE INDEX IF NOT EXISTS idx_digest_articles_citation ON digest_articles(citation_order);
CREATE INDEX IF NOT EXISTS idx_digest_themes_digest ON digest_themes(digest_id);
CREATE INDEX IF NOT EXISTS idx_digest_themes_theme ON digest_themes(theme_id);

COMMIT;
```

**Run migration:**

```bash
briefly migrate --target 013
```

**Verify:**

```sql
-- Check tables exist
\dt digest_*

-- Check indexes
\di digest_*
```

---

### Step 1.3: Update Go Structs

**File:** `internal/core/core.go`

Update `Digest` struct to match new schema:

```go
// core/core.go

type Digest struct {
    ID            uuid.UUID     `json:"id"`
    Title         string        `json:"title"`         // < 100 chars
    TLDR          string        `json:"tldr"`          // One sentence
    Summary       string        `json:"summary"`       // 2-3 paragraphs with [1][2] citations
    KeyMoments    []KeyMoment   `json:"key_moments"`   // Important quotes
    ClusterID     int           `json:"cluster_id"`    // Which K-means cluster
    ProcessedDate time.Time     `json:"processed_date"`
    ArticleCount  int           `json:"article_count"`
    PipelineRunID *uuid.UUID    `json:"pipeline_run_id,omitempty"`
    CreatedAt     time.Time     `json:"created_at"`
    UpdatedAt     time.Time     `json:"updated_at"`

    // Relationships (loaded separately)
    Articles      []Article     `json:"articles,omitempty"`
    Themes        []Theme       `json:"themes,omitempty"`
}

type KeyMoment struct {
    Quote            string    `json:"quote"`
    ArticleID        uuid.UUID `json:"article_id"`
    CitationNumber   int       `json:"citation_number"`
}
```

**Remove deprecated fields:**

```go
// DELETE these fields from Digest struct:
// - Content string (replaced by Summary)
// - PublishedAt time.Time (replaced by ProcessedDate)
// - DigestSummary string (use Summary instead)
```

**Search and replace usage:**

```bash
# Find all usages of deprecated fields
grep -r "\.Content" internal/ cmd/ | grep -i digest
grep -r "\.PublishedAt" internal/ cmd/ | grep -i digest
grep -r "DigestSummary" internal/ cmd/

# Replace in code (manual review)
```

---

## Phase 2: Repository Layer Updates (Day 3)

### Step 2.1: Update DigestRepository Interface

**File:** `internal/persistence/interfaces.go`

```go
// persistence/interfaces.go

type DigestRepository interface {
    // Create stores a new digest with all relationships
    Create(ctx context.Context, digest *core.Digest, articles []core.Article, themes []core.Theme) error

    // GetByID retrieves a digest by ID
    GetByID(ctx context.Context, id uuid.UUID) (*core.Digest, error)

    // GetWithArticles retrieves a digest with all related articles
    GetWithArticles(ctx context.Context, id uuid.UUID) (*core.Digest, error)

    // ListRecent retrieves digests created since the given duration
    ListRecent(ctx context.Context, since time.Duration) ([]core.Digest, error)

    // ListByTheme retrieves digests for a specific theme
    ListByTheme(ctx context.Context, themeID uuid.UUID, since time.Duration) ([]core.Digest, error)

    // ListByDateRange retrieves digests within a date range
    ListByDateRange(ctx context.Context, startDate, endDate time.Time) ([]core.Digest, error)

    // Update updates an existing digest
    Update(ctx context.Context, digest *core.Digest) error

    // Delete soft-deletes a digest
    Delete(ctx context.Context, id uuid.UUID) error
}
```

---

### Step 2.2: Implement DigestRepository Methods

**File:** `internal/persistence/postgres_digest_repo.go`

Create new file with full implementation:

```go
// persistence/postgres_digest_repo.go

package persistence

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/rcliao/briefly/internal/core"
)

type PostgresDigestRepository struct {
    db *sql.DB
}

func NewPostgresDigestRepository(db *sql.DB) *PostgresDigestRepository {
    return &PostgresDigestRepository{db: db}
}

// Create stores a new digest with all relationships
func (r *PostgresDigestRepository) Create(ctx context.Context, digest *core.Digest, articles []core.Article, themes []core.Theme) error {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback()

    // 1. Marshal key_moments to JSONB
    keyMomentsJSON, err := json.Marshal(digest.KeyMoments)
    if err != nil {
        return fmt.Errorf("marshal key_moments: %w", err)
    }

    // 2. Insert digest
    _, err = tx.ExecContext(ctx, `
        INSERT INTO digests (id, title, tldr, summary, key_moments, cluster_id, processed_date, article_count, pipeline_run_id, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    `, digest.ID, digest.Title, digest.TLDR, digest.Summary, keyMomentsJSON, digest.ClusterID, digest.ProcessedDate, digest.ArticleCount, digest.PipelineRunID, digest.CreatedAt, digest.UpdatedAt)
    if err != nil {
        return fmt.Errorf("insert digest: %w", err)
    }

    // 3. Insert digest-article relationships
    for i, article := range articles {
        _, err = tx.ExecContext(ctx, `
            INSERT INTO digest_articles (digest_id, article_id, citation_order, relevance_to_digest)
            VALUES ($1, $2, $3, $4)
        `, digest.ID, article.ID, i+1, 1.0) // citation_order starts at 1
        if err != nil {
            return fmt.Errorf("insert digest_article: %w", err)
        }
    }

    // 4. Insert digest-theme relationships
    uniqueThemes := extractUniqueThemes(articles, themes)
    for _, theme := range uniqueThemes {
        _, err = tx.ExecContext(ctx, `
            INSERT INTO digest_themes (digest_id, theme_id)
            VALUES ($1, $2)
            ON CONFLICT DO NOTHING
        `, digest.ID, theme.ID)
        if err != nil {
            return fmt.Errorf("insert digest_theme: %w", err)
        }
    }

    return tx.Commit()
}

// GetWithArticles retrieves a digest with all related articles
func (r *PostgresDigestRepository) GetWithArticles(ctx context.Context, id uuid.UUID) (*core.Digest, error) {
    query := `
        SELECT
            d.id, d.title, d.tldr, d.summary, d.key_moments, d.cluster_id,
            d.processed_date, d.article_count, d.pipeline_run_id,
            d.created_at, d.updated_at,
            COALESCE(
                json_agg(
                    json_build_object(
                        'id', a.id,
                        'url', a.url,
                        'title', a.title,
                        'published_at', a.published_at,
                        'citation_order', da.citation_order
                    ) ORDER BY da.citation_order
                ) FILTER (WHERE a.id IS NOT NULL),
                '[]'
            ) as articles,
            COALESCE(
                json_agg(
                    DISTINCT jsonb_build_object(
                        'id', t.id,
                        'name', t.name
                    )
                ) FILTER (WHERE t.id IS NOT NULL),
                '[]'
            ) as themes
        FROM digests d
        LEFT JOIN digest_articles da ON d.id = da.digest_id
        LEFT JOIN articles a ON da.article_id = a.id
        LEFT JOIN digest_themes dt ON d.id = dt.digest_id
        LEFT JOIN themes t ON dt.theme_id = t.id
        WHERE d.id = $1
        GROUP BY d.id
    `

    var digest core.Digest
    var keyMomentsJSON []byte
    var articlesJSON []byte
    var themesJSON []byte

    err := r.db.QueryRowContext(ctx, query, id).Scan(
        &digest.ID, &digest.Title, &digest.TLDR, &digest.Summary, &keyMomentsJSON,
        &digest.ClusterID, &digest.ProcessedDate, &digest.ArticleCount,
        &digest.PipelineRunID, &digest.CreatedAt, &digest.UpdatedAt,
        &articlesJSON, &themesJSON,
    )
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("digest not found")
    }
    if err != nil {
        return nil, fmt.Errorf("query digest: %w", err)
    }

    // Unmarshal JSON fields
    if err := json.Unmarshal(keyMomentsJSON, &digest.KeyMoments); err != nil {
        return nil, fmt.Errorf("unmarshal key_moments: %w", err)
    }
    if err := json.Unmarshal(articlesJSON, &digest.Articles); err != nil {
        return nil, fmt.Errorf("unmarshal articles: %w", err)
    }
    if err := json.Unmarshal(themesJSON, &digest.Themes); err != nil {
        return nil, fmt.Errorf("unmarshal themes: %w", err)
    }

    return &digest, nil
}

// ListRecent retrieves digests created since the given duration
func (r *PostgresDigestRepository) ListRecent(ctx context.Context, since time.Duration) ([]core.Digest, error) {
    cutoff := time.Now().Add(-since)

    query := `
        SELECT
            d.id, d.title, d.tldr, d.cluster_id, d.processed_date,
            d.article_count, d.created_at,
            COALESCE(
                json_agg(
                    DISTINCT jsonb_build_object('id', t.id, 'name', t.name)
                ) FILTER (WHERE t.id IS NOT NULL),
                '[]'
            ) as themes
        FROM digests d
        LEFT JOIN digest_themes dt ON d.id = dt.digest_id
        LEFT JOIN themes t ON dt.theme_id = t.id
        WHERE d.created_at >= $1
        GROUP BY d.id
        ORDER BY d.created_at DESC
    `

    rows, err := r.db.QueryContext(ctx, query, cutoff)
    if err != nil {
        return nil, fmt.Errorf("query digests: %w", err)
    }
    defer rows.Close()

    digests := []core.Digest{}
    for rows.Next() {
        var d core.Digest
        var themesJSON []byte

        err := rows.Scan(&d.ID, &d.Title, &d.TLDR, &d.ClusterID,
            &d.ProcessedDate, &d.ArticleCount, &d.CreatedAt, &themesJSON)
        if err != nil {
            return nil, fmt.Errorf("scan digest: %w", err)
        }

        if err := json.Unmarshal(themesJSON, &d.Themes); err != nil {
            return nil, fmt.Errorf("unmarshal themes: %w", err)
        }

        digests = append(digests, d)
    }

    return digests, nil
}

// Helper: Extract unique themes from articles
func extractUniqueThemes(articles []core.Article, themes []core.Theme) []core.Theme {
    themeMap := make(map[uuid.UUID]core.Theme)

    // If themes provided, use those
    for _, theme := range themes {
        themeMap[theme.ID] = theme
    }

    // Otherwise, extract from article-theme relationships
    // (assumes articles have Themes field loaded)
    if len(themes) == 0 {
        for _, article := range articles {
            for _, theme := range article.Themes {
                themeMap[theme.ID] = theme
            }
        }
    }

    uniqueThemes := []core.Theme{}
    for _, theme := range themeMap {
        uniqueThemes = append(uniqueThemes, theme)
    }

    return uniqueThemes
}
```

**Testing:**

```bash
# Unit tests for repository
go test ./internal/persistence -run TestDigestRepository -v
```

---

## Phase 3: Pipeline Refactor (Days 4-5)

### Step 3.1: Refactor Pipeline to Generate Many Digests

**File:** `internal/pipeline/pipeline.go`

**Current signature (WRONG):**

```go
func (p *Pipeline) GenerateDigest(ctx context.Context, opts DigestOptions) (*core.Digest, error)
```

**New signature (CORRECT):**

```go
func (p *Pipeline) GenerateDigests(ctx context.Context, opts DigestOptions) ([]core.Digest, error)
```

**Implementation:**

```go
// pipeline/pipeline.go

func (p *Pipeline) GenerateDigests(ctx context.Context, opts DigestOptions) ([]core.Digest, error) {
    logger := p.logger.With("operation", "generate_digests")

    // Step 1: Aggregate articles from sources
    logger.Info("Step 1/8: Aggregating articles", "since", opts.Since)
    articles, err := p.aggregator.Aggregate(ctx, opts.Since)
    if err != nil {
        return nil, fmt.Errorf("aggregate: %w", err)
    }
    logger.Info("Articles aggregated", "count", len(articles))

    // Step 2: Classify and filter by theme
    logger.Info("Step 2/8: Classifying articles by theme")
    relevantArticles := []core.Article{}
    for i, article := range articles {
        classifications, err := p.themeClassifier.ClassifyArticle(ctx, &article, opts.RelevanceThreshold)
        if err != nil {
            logger.Warn("Classification failed", "article", article.URL, "error", err)
            continue
        }

        if len(classifications) > 0 {
            article.Themes = classificationsToThemes(classifications)
            relevantArticles = append(relevantArticles, article)
            logger.Debug("Article classified", "article", i+1, "themes", len(classifications))
        }
    }
    logger.Info("Articles classified", "relevant", len(relevantArticles), "filtered", len(articles)-len(relevantArticles))

    if len(relevantArticles) < 3 {
        return nil, fmt.Errorf("insufficient articles for clustering (need 3+, got %d)", len(relevantArticles))
    }

    // Step 3: Summarize articles
    logger.Info("Step 3/8: Summarizing articles")
    summaries := make(map[uuid.UUID]core.Summary)
    for i, article := range relevantArticles {
        summary, err := p.summarizer.SummarizeArticle(ctx, &article)
        if err != nil {
            logger.Warn("Summarization failed", "article", article.URL, "error", err)
            continue
        }
        summaries[article.ID] = *summary
        logger.Debug("Article summarized", "article", i+1, "summary_length", len(summary.SummaryText))
    }

    // Step 4: Generate embeddings
    logger.Info("Step 4/8: Generating embeddings")
    embeddings := make(map[uuid.UUID][]float64)
    for articleID, summary := range summaries {
        embedding, err := p.embeddingGenerator.GenerateEmbedding(ctx, summary.SummaryText)
        if err != nil {
            logger.Warn("Embedding generation failed", "article_id", articleID, "error", err)
            continue
        }
        embeddings[articleID] = embedding
    }
    logger.Info("Embeddings generated", "count", len(embeddings))

    // Step 5: Cluster articles by similarity
    logger.Info("Step 5/8: Clustering articles", "target_clusters", opts.ClusterCount)
    clusters, err := p.clusterer.ClusterArticles(ctx, relevantArticles, embeddings, opts.ClusterCount)
    if err != nil {
        return nil, fmt.Errorf("cluster: %w", err)
    }
    logger.Info("Clustering complete", "clusters", len(clusters))

    // Step 6: Generate digest summaries (one per cluster)
    logger.Info("Step 6/8: Generating digest summaries")
    digests := []core.Digest{}
    pipelineRunID := uuid.New() // Track this run

    for i, cluster := range clusters {
        logger.Info("Generating digest", "cluster", i+1, "articles", len(cluster.ArticleIDs))

        // Get articles for this cluster
        clusterArticles := filterArticlesByIDs(relevantArticles, cluster.ArticleIDs)
        clusterSummaries := filterSummariesByArticleIDs(summaries, cluster.ArticleIDs)

        // Generate digest
        digest, err := p.digestGenerator.GenerateDigest(ctx, cluster, clusterArticles, clusterSummaries)
        if err != nil {
            logger.Warn("Digest generation failed", "cluster", cluster.Label, "error", err)
            continue
        }

        digest.PipelineRunID = &pipelineRunID
        digest.ProcessedDate = time.Now()
        digest.CreatedAt = time.Now()
        digest.UpdatedAt = time.Now()

        // Step 7: Store in database
        logger.Info("Step 7/8: Storing digest", "title", digest.Title)
        err = p.digestRepo.Create(ctx, digest, clusterArticles, extractThemes(clusterArticles))
        if err != nil {
            logger.Error("Failed to store digest", "title", digest.Title, "error", err)
            return nil, fmt.Errorf("store digest: %w", err)
        }

        digests = append(digests, *digest)
        logger.Info("Digest stored", "id", digest.ID, "title", digest.Title)
    }

    // Step 8: Render output (optional)
    if opts.RenderMarkdown {
        logger.Info("Step 8/8: Rendering markdown files")
        for _, digest := range digests {
            outputPath := fmt.Sprintf("%s/digest_%s_%s.md", opts.OutputPath, digest.ProcessedDate.Format("2006-01-02"), sanitizeFilename(digest.Title))
            _, err := p.renderer.RenderDigest(ctx, &digest, digest.Articles)
            if err != nil {
                logger.Warn("Render failed", "digest", digest.Title, "error", err)
            }
            logger.Debug("Markdown rendered", "path", outputPath)
        }
    }

    logger.Info("Pipeline complete", "digests_generated", len(digests))
    return digests, nil
}
```

---

### Step 3.2: Remove Category-Based Grouping

**Files to update:**

1. `cmd/handlers/digest_generate.go` - Remove `groupArticlesByCategory()`
2. `internal/pipeline/adapters.go` - Remove category-based logic

**Search and delete:**

```bash
# Find all category grouping references
grep -r "groupArticlesByCategory" .
grep -r "ArticleGroup" . | grep -v "test"

# Delete functions manually
```

---

## Phase 4: Handler Consolidation (Day 6)

### Step 4.1: Consolidate Digest Handlers

**Create new unified handler:** `cmd/handlers/digest_unified.go`

```go
// cmd/handlers/digest_unified.go

package handlers

import (
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/spf13/cobra"
)

var digestCmd = &cobra.Command{
    Use:   "digest",
    Short: "Manage digests",
    Long:  "Generate, list, and view digests from aggregated articles.",
}

// briefly digest generate --since 24h --clusters 5
var digestGenerateCmd = &cobra.Command{
    Use:   "generate",
    Short: "Generate digests from recent articles",
    RunE:  runDigestGenerate,
}

var (
    sinceDuration       time.Duration
    clusterCount        int
    relevanceThreshold  float64
    renderMarkdown      bool
    outputPath          string
)

func init() {
    digestGenerateCmd.Flags().DurationVar(&sinceDuration, "since", 24*time.Hour, "Process articles from last N duration (e.g., 24h, 7d)")
    digestGenerateCmd.Flags().IntVar(&clusterCount, "clusters", 0, "Number of clusters (0 = auto-select)")
    digestGenerateCmd.Flags().Float64Var(&relevanceThreshold, "min-relevance", 0.4, "Minimum relevance score (0.0-1.0)")
    digestGenerateCmd.Flags().BoolVar(&renderMarkdown, "render", false, "Render markdown files")
    digestGenerateCmd.Flags().StringVar(&outputPath, "output", "digests", "Output directory for markdown files")

    digestCmd.AddCommand(digestGenerateCmd)
}

func runDigestGenerate(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()

    // Auto-select cluster count if not specified
    if clusterCount == 0 {
        clusterCount = 5 // reasonable default
    }

    fmt.Printf("Generating digests (since: %s, clusters: %d)...\n", sinceDuration, clusterCount)

    // Build pipeline
    pipe, err := buildPipeline(ctx)
    if err != nil {
        return fmt.Errorf("build pipeline: %w", err)
    }

    // Generate digests
    digests, err := pipe.GenerateDigests(ctx, pipeline.DigestOptions{
        Since:               sinceDuration,
        ClusterCount:        clusterCount,
        RelevanceThreshold:  relevanceThreshold,
        RenderMarkdown:      renderMarkdown,
        OutputPath:          outputPath,
    })
    if err != nil {
        return fmt.Errorf("generate digests: %w", err)
    }

    // Print summary
    fmt.Printf("\n✓ Generated %d digests:\n", len(digests))
    for i, digest := range digests {
        fmt.Printf("  %d. %s (%d articles)\n", i+1, digest.Title, digest.ArticleCount)
        fmt.Printf("     ID: %s\n", digest.ID)
        fmt.Printf("     Themes: %s\n", formatThemes(digest.Themes))
    }

    return nil
}

// briefly digest list --since 7d --theme "GenAI & LLMs"
var digestListCmd = &cobra.Command{
    Use:   "list",
    Short: "List recent digests",
    RunE:  runDigestList,
}

var (
    listSince time.Duration
    themeFilter string
)

func init() {
    digestListCmd.Flags().DurationVar(&listSince, "since", 7*24*time.Hour, "List digests from last N duration")
    digestListCmd.Flags().StringVar(&themeFilter, "theme", "", "Filter by theme name")

    digestCmd.AddCommand(digestListCmd)
}

func runDigestList(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()

    // Get digest repository
    digestRepo := getDigestRepository(ctx)

    // Query digests
    digests, err := digestRepo.ListRecent(ctx, listSince)
    if err != nil {
        return fmt.Errorf("list digests: %w", err)
    }

    // Filter by theme if specified
    if themeFilter != "" {
        digests = filterDigestsByTheme(digests, themeFilter)
    }

    // Print table
    fmt.Printf("Digests (last %s):\n\n", listSince)
    printDigestTable(digests)

    return nil
}

// briefly digest show <digest-id>
var digestShowCmd = &cobra.Command{
    Use:   "show <digest-id>",
    Short: "Show digest details",
    Args:  cobra.ExactArgs(1),
    RunE:  runDigestShow,
}

func init() {
    digestCmd.AddCommand(digestShowCmd)
}

func runDigestShow(cmd *cobra.Command, args []string) error {
    ctx := cmd.Context()
    digestID, err := uuid.Parse(args[0])
    if err != nil {
        return fmt.Errorf("invalid digest ID: %w", err)
    }

    // Get digest with articles
    digestRepo := getDigestRepository(ctx)
    digest, err := digestRepo.GetWithArticles(ctx, digestID)
    if err != nil {
        return fmt.Errorf("get digest: %w", err)
    }

    // Render digest
    renderDigestDetail(digest)

    return nil
}
```

**Delete old handlers:**

```bash
# After verifying new handler works
rm cmd/handlers/digest.go
rm cmd/handlers/digest_simplified.go
rm cmd/handlers/digest_generate.go
```

---

## Phase 5: Frontend Implementation (Days 7-8)

### Step 5.1: Create Digest List Page

**File:** `internal/server/digest_handlers.go`

```go
// server/digest_handlers.go

func (h *DigestHandler) ListDigests(w http.ResponseWriter, r *http.Request) {
    ctx := r.Context()

    // Parse query params
    themeFilter := r.URL.Query().Get("theme")
    timeWindow := r.URL.Query().Get("since")

    since := parseDuration(timeWindow, 24*time.Hour)

    // Query digests
    digests, err := h.digestRepo.ListRecent(ctx, since)
    if err != nil {
        http.Error(w, err.Error(), 500)
        return
    }

    // Filter by theme
    if themeFilter != "" {
        digests = filterByTheme(digests, themeFilter)
    }

    // Render template
    h.templates.ExecuteTemplate(w, "digest_list.html", map[string]interface{}{
        "Digests":    digests,
        "Theme":      themeFilter,
        "TimeWindow": timeWindow,
    })
}
```

**Template:** `internal/server/templates/digest_list.html`

```html
<!DOCTYPE html>
<html>
<head>
    <title>Briefly - GenAI Digests</title>
    <script src="https://unpkg.com/htmx.org@1.9.0"></script>
    <link href="https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css" rel="stylesheet">
</head>
<body class="bg-gray-50">
    <div class="container mx-auto px-4 py-8">
        <header class="mb-8">
            <h1 class="text-4xl font-bold text-gray-900">Briefly</h1>
            <p class="text-gray-600">GenAI News Digests</p>
        </header>

        <!-- Filters -->
        <div class="mb-6 flex gap-4">
            <select name="theme" class="border rounded px-4 py-2" hx-get="/digests" hx-trigger="change" hx-target="#digest-list" hx-include="[name='since']">
                <option value="">All Themes</option>
                {{range .Themes}}
                <option value="{{.Name}}" {{if eq .Name $.Theme}}selected{{end}}>{{.Name}}</option>
                {{end}}
            </select>

            <select name="since" class="border rounded px-4 py-2" hx-get="/digests" hx-trigger="change" hx-target="#digest-list" hx-include="[name='theme']">
                <option value="24h" {{if eq $.TimeWindow "24h"}}selected{{end}}>Last 24 hours</option>
                <option value="7d" {{if eq $.TimeWindow "7d"}}selected{{end}}>Last 7 days</option>
                <option value="30d" {{if eq $.TimeWindow "30d"}}selected{{end}}>Last 30 days</option>
            </select>
        </div>

        <!-- Digest List -->
        <div id="digest-list" class="space-y-4">
            {{range .Digests}}
            <div class="bg-white rounded-lg shadow p-6 cursor-pointer hover:shadow-lg transition"
                 hx-get="/digests/{{.ID}}"
                 hx-push-url="true"
                 hx-target="body">
                <div class="flex justify-between items-start mb-2">
                    <h2 class="text-2xl font-semibold text-gray-900">{{.Title}}</h2>
                    <span class="text-sm text-gray-500">{{.ArticleCount}} articles</span>
                </div>
                <p class="text-gray-700 mb-4">{{.TLDR}}</p>
                <div class="flex justify-between items-center">
                    <div class="flex gap-2">
                        {{range .Themes}}
                        <span class="bg-blue-100 text-blue-800 text-xs px-2 py-1 rounded">🏷 {{.Name}}</span>
                        {{end}}
                    </div>
                    <span class="text-sm text-gray-500">{{timeAgo .CreatedAt}}</span>
                </div>
            </div>
            {{end}}
        </div>
    </div>
</body>
</html>
```

---

### Step 5.2: Create Digest Detail Page

**Template:** `internal/server/templates/digest_detail.html`

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}} - Briefly</title>
    <script src="https://unpkg.com/htmx.org@1.9.0"></script>
    <link href="https://cdn.jsdelivr.net/npm/tailwindcss@2.2.19/dist/tailwind.min.css" rel="stylesheet">
</head>
<body class="bg-gray-50">
    <div class="container mx-auto px-4 py-8 max-w-4xl">
        <div class="mb-4">
            <a href="/digests" class="text-blue-600 hover:underline">← Back to Digests</a>
        </div>

        <article class="bg-white rounded-lg shadow p-8">
            <!-- Header -->
            <header class="mb-6">
                <h1 class="text-4xl font-bold text-gray-900 mb-2">{{.Title}}</h1>
                <div class="flex justify-between items-center text-sm text-gray-600">
                    <div class="flex gap-2">
                        {{range .Themes}}
                        <span class="bg-blue-100 text-blue-800 px-2 py-1 rounded">🏷 {{.Name}}</span>
                        {{end}}
                    </div>
                    <span>{{.ProcessedDate.Format "Jan 2, 2006"}} • {{.ArticleCount}} articles</span>
                </div>
            </header>

            <!-- TLDR -->
            <section class="mb-8 bg-blue-50 p-4 rounded">
                <h2 class="text-sm font-semibold text-blue-900 mb-2">TL;DR</h2>
                <p class="text-gray-800">{{.TLDR}}</p>
            </section>

            <!-- Summary -->
            <section class="mb-8 prose max-w-none">
                <h2 class="text-2xl font-semibold mb-4">Summary</h2>
                <div class="text-gray-800 leading-relaxed">
                    {{.Summary | renderWithCitations}}
                </div>
            </section>

            <!-- Key Moments -->
            {{if .KeyMoments}}
            <section class="mb-8">
                <h2 class="text-2xl font-semibold mb-4">Key Moments</h2>
                <ul class="space-y-3">
                    {{range .KeyMoments}}
                    <li class="flex items-start">
                        <span class="text-blue-600 mr-3">•</span>
                        <div>
                            <p class="text-gray-800">"{{.Quote}}"</p>
                            <span class="text-sm text-gray-600">[{{.CitationNumber}}]</span>
                        </div>
                    </li>
                    {{end}}
                </ul>
            </section>
            {{end}}

            <!-- Sources -->
            <section>
                <h2 class="text-2xl font-semibold mb-4">Sources ({{.ArticleCount}} articles)</h2>
                <div class="space-y-4">
                    {{range .Articles}}
                    <div class="border-l-4 border-blue-500 pl-4">
                        <div class="flex items-start justify-between">
                            <div>
                                <span class="text-sm font-semibold text-gray-600">[{{.CitationOrder}}]</span>
                                <a href="{{.URL}}" target="_blank" class="text-blue-600 hover:underline font-medium">
                                    {{.Title}}
                                </a>
                            </div>
                        </div>
                        <p class="text-sm text-gray-600 mt-1">{{.URL}}</p>
                        <p class="text-xs text-gray-500 mt-1">{{.PublishedAt.Format "Jan 2, 2006"}}</p>
                    </div>
                    {{end}}
                </div>
            </section>
        </article>
    </div>
</body>
</html>
```

---

## Phase 6: Testing & Validation (Days 9-10)

### Step 6.1: End-to-End Test

```bash
# 1. Run migrations
briefly migrate

# 2. Add some feeds (if not already added)
briefly feed add "https://openai.com/blog/rss.xml" --name "OpenAI Blog"
briefly feed add "https://techcrunch.com/category/artificial-intelligence/feed/" --name "TechCrunch AI"

# 3. Aggregate articles (last 7 days for testing)
briefly aggregate --since 168h --min-relevance 0.4

# 4. Generate digests
briefly digest generate --since 168h --clusters 5

# Expected output:
# ✓ Generated 5 digests:
#   1. GPT-5 Launch (8 articles)
#      ID: 123e4567-e89b-12d3-a456-426614174000
#      Themes: GenAI & LLMs, Cloud & DevOps
#   2. Claude 3.5 Updates (6 articles)
#      ...

# 5. List digests
briefly digest list --since 7d

# 6. Show digest detail
briefly digest show 123e4567-e89b-12d3-a456-426614174000

# 7. Start web server
briefly serve

# 8. Open browser: http://localhost:8080/digests
```

---

### Step 6.2: Validation Checklist

**Database:**
- [ ] Migrations 012-013 applied successfully
- [ ] Digests table has all new fields
- [ ] Join tables (digest_articles, digest_themes) exist with indexes
- [ ] Old data migrated without loss

**Pipeline:**
- [ ] Generates 3-7 digests per run (not 1)
- [ ] Each digest stored in database with correct relationships
- [ ] Citations tracked with correct numbering
- [ ] Themes extracted from articles and linked to digests

**CLI:**
- [ ] `briefly digest generate` works
- [ ] `briefly digest list` shows digests
- [ ] `briefly digest show <id>` displays full digest

**Frontend:**
- [ ] `/digests` page shows digest list
- [ ] Theme filter works
- [ ] Time window filter works
- [ ] `/digests/:id` shows digest detail with articles
- [ ] Citations clickable and link to sources

**Performance:**
- [ ] Digest list page loads < 100ms
- [ ] Digest detail page loads < 200ms
- [ ] Pipeline generates 5 digests in < 5 minutes

---

## Rollback Plan (If Needed)

### Quick Rollback (Database)

```sql
-- Rollback migration 013
DROP TABLE IF EXISTS digest_articles CASCADE;
DROP TABLE IF EXISTS digest_themes CASCADE;

-- Rollback migration 012
ALTER TABLE digests DROP COLUMN IF EXISTS tldr;
ALTER TABLE digests DROP COLUMN IF EXISTS summary;
ALTER TABLE digests DROP COLUMN IF EXISTS key_moments;
ALTER TABLE digests DROP COLUMN IF EXISTS cluster_id;
ALTER TABLE digests DROP COLUMN IF EXISTS processed_date;
ALTER TABLE digests DROP COLUMN IF EXISTS article_count;
ALTER TABLE digests DROP COLUMN IF EXISTS pipeline_run_id;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS content TEXT;
ALTER TABLE digests ADD COLUMN IF NOT EXISTS published_at TIMESTAMP;
```

### Code Rollback

```bash
# Checkout previous version
git checkout v3.0-before-many-digests

# Restore database
pg_restore -d briefly backup.dump
```

---

## Post-Migration Tasks

- [ ] Update CLAUDE.md with new command structure
- [ ] Update README.md with examples
- [ ] Write blog post explaining the architecture change
- [ ] Create demo video showing digest list UI
- [ ] Tag release: `v3.1-many-digests-architecture`
- [ ] Deploy to production (if applicable)
- [ ] Monitor for errors in first 48 hours

---

## Success Metrics

**Before Migration:**
- 1 digest per pipeline run
- No digest list UI
- Manual markdown file workflow

**After Migration:**
- 3-7 digests per pipeline run
- Kagi News-style digest list UI
- Digests stored in database with relationships
- Theme and time filtering working
- Citation tracking integrated

---

**Migration Status:** ⏳ Ready to Execute
**Estimated Completion:** 7-10 days
**Breaking Changes:** Yes (schema + API)
**Rollback Available:** Yes
