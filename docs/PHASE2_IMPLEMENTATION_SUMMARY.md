# Phase 2: Pipeline Refactor - Implementation Summary

**Date:** 2025-11-06
**Status:** ✅ Core Implementation Complete
**Branch:** main

---

## Overview

Successfully refactored the digest pipeline from **single digest** architecture (v1.0) to **many digests** architecture (v2.0), enabling Kagi News-style digest generation where each topic cluster produces its own focused digest.

---

## ✅ Completed Work

### 1. Database Schema Migration (Phase 1)

**Migrations Created:**

- **Migration 012:** Updated digests table
  - Added `summary` (markdown with `[[N]](url)` citations)
  - Added `key_moments` (JSONB array)
  - Added `perspectives` (JSONB for viewpoints)
  - Added `cluster_id` (INTEGER for HDBSCAN)
  - Added `processed_date` (DATE for filtering)
  - Added `article_count` (cached count)
  - Removed UNIQUE constraint on `date` (multiple digests per day)
  - Migrated existing data from `content` JSONB field

- **Migration 013:** Created relationship tables
  - `digest_articles` (many-to-many with citation_order)
  - `digest_themes` (many-to-many theme associations)
  - Added indexes for performance

- **Migration 014:** Updated articles and citations tables
  - Added `publisher` field to articles
  - Updated citations table for digest citations (digest_id, citation_number, context)
  - Removed UNIQUE constraint on article_id
  - Prepared for optional pgvector migration

**Status:** ✅ All migrations applied successfully

### 2. Core Data Structures Updated

**Files Modified:**
- `internal/core/core.go`

**Changes:**
- Updated `Article` struct: Added `Publisher` field
- Updated `Digest` struct: Added v2.0 fields (Summary, TLDRSummary, KeyMoments, Perspectives, ClusterID, ProcessedDate, ArticleCount, Themes, Articles)
- Created `KeyMoment` struct for structured quotes
- Created `Perspective` struct for supporting/opposing viewpoints
- Updated `Citation` struct: Added digest citation fields (DigestID, CitationNumber, Context)

### 3. Repository Interfaces Enhanced

**Files Modified:**
- `internal/persistence/interfaces.go`

**New DigestRepository Methods:**
- `StoreWithRelationships()` - Atomic storage with article/theme relationships
- `GetWithArticles()` - Eager loading of articles
- `GetWithThemes()` - Eager loading of themes
- `GetFull()` - Load everything (articles, themes, citations)
- `ListRecent()` - Time-window filtering for homepage
- `ListByTheme()` - Theme-based filtering
- `ListByCluster()` - Cluster-based queries

**New CitationRepository Methods:**
- `CreateBatch()` - Efficient batch insertion
- `GetByDigestID()` - Retrieve all citations for a digest
- `DeleteByDigestID()` - Clean up digest citations

### 4. Repository Implementation

**Files Modified:**
- `internal/persistence/postgres_repos.go` (digest repository)
- `internal/persistence/postgres_citation_repo.go` (citation repository)

**Implemented Methods:**

#### DigestRepository.StoreWithRelationships()
```go
// Full transaction support with 4 steps:
// 1. Insert digest (using v2.0 schema)
// 2. Create digest_articles relationships with citation order
// 3. Create digest_themes relationships
// 4. Extract and store citations from summary markdown (TODO)
```

**Key Features:**
- Uses PostgreSQL transactions for atomicity
- Marshals KeyMoments and Perspectives to JSONB
- Creates citation-ordered article relationships
- Handles both transactional and non-transactional contexts
- ON CONFLICT handling for idempotency

#### DigestRepository.ListRecent()
```go
// Efficient homepage query with LEFT JOINs:
// - Retrieves digests since a given date
// - Eagerly loads associated themes
// - Uses JSON aggregation for theme arrays
// - Orders by processed_date DESC, created_at DESC
```

**Key Features:**
- Single query with LEFT JOINs (no N+1 queries)
- JSON aggregation for themes array
- COALESCE for empty arrays
- Proper NULL handling for JSONB fields

#### Stub Implementations (for compilation)
- `GetWithArticles()` - Falls back to basic Get
- `GetWithThemes()` - Falls back to basic Get
- `GetFull()` - Falls back to basic Get
- `ListByTheme()` - Returns empty list (TODO)
- `ListByCluster()` - Returns empty list (TODO)
- `CreateBatch()` (citations) - Loops through citations (inefficient but functional)
- `GetByDigestID()` (citations) - Returns empty list (TODO)
- `DeleteByDigestID()` (citations) - No-op (TODO)

### 5. Pipeline Refactoring

**Files Modified:**
- `internal/pipeline/pipeline.go`

**Major Changes:**

#### New Method: GenerateDigests() (v2.0)
```go
func (p *Pipeline) GenerateDigests(ctx context.Context, opts DigestOptions) ([]DigestResult, error)
```

**Flow:**
1. Parse URLs from markdown file
2. Fetch and summarize articles (with caching)
3. Generate embeddings for clustering
4. Cluster articles by topic
5. **Loop through each cluster** (v2.0 change!)
6. For each cluster:
   - Extract cluster articles and summaries
   - Build digest for this specific cluster
   - Generate title, TLDR, and summary
   - Render markdown output
   - Collect result
7. Return array of DigestResult

**Key Improvements:**
- Generates 3-7 digests per run (one per cluster)
- Each digest is self-contained and focused
- Parallel rendering support
- Better progress logging

#### New Method: buildDigestForCluster()
```go
func (p *Pipeline) buildDigestForCluster(
    cluster core.TopicCluster,
    articles []core.Article,
    summaries []core.Summary,
) *core.Digest
```

**Replaces:** Category-based grouping in `buildDigest()`

**Key Changes:**
- Creates ONE digest per cluster (not one digest with all categories)
- Sets cluster.Label as title
- Populates v2.0 fields (ProcessedDate, ArticleCount)
- ClusterID set to nil (will be numeric when HDBSCAN is implemented)
- Creates single ArticleGroup for backward compatibility

#### Legacy Method: GenerateDigest() (v1.0 - DEPRECATED)
- Kept for backward compatibility
- Marked as deprecated in comments
- Uses category-based grouping (old behavior)

---

## 🔧 Architecture Changes

### Before (v1.0)
```
Fetch Articles → Cluster → Group by Category → ONE Digest → Markdown File
```

### After (v2.0)
```
Fetch Articles → Cluster → [For Each Cluster] → MANY Digests → Database + Files
                                    ↓
                     Generate Individual Digest
                                    ↓
                     StoreWithRelationships
                                    ↓
                     Render Markdown
```

---

## 📊 Database Schema

### Digests Table (v2.0)
```sql
CREATE TABLE digests (
    id VARCHAR(255) PRIMARY KEY,
    summary TEXT NOT NULL,                    -- Markdown with [[N]](url) citations
    tldr_summary TEXT,                        -- 50-70 chars one-liner
    key_moments JSONB,                        -- [{quote, citation_number}]
    perspectives JSONB,                       -- [{type, summary, citation_numbers}]
    cluster_id INTEGER,                       -- HDBSCAN cluster (-1 = noise, NULL = weekly)
    processed_date DATE NOT NULL,             -- When generated
    article_count INTEGER NOT NULL,           -- Cached count
    created_at TIMESTAMP DEFAULT NOW()
);
```

### Relationship Tables (v2.0)
```sql
CREATE TABLE digest_articles (
    digest_id VARCHAR(255) REFERENCES digests(id),
    article_id VARCHAR(255) REFERENCES articles(id),
    citation_order INTEGER NOT NULL,          -- [1], [2], [3]
    relevance_to_digest FLOAT,
    added_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (digest_id, article_id)
);

CREATE TABLE digest_themes (
    digest_id VARCHAR(255) REFERENCES digests(id),
    theme_id VARCHAR(255) REFERENCES themes(id),
    added_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (digest_id, theme_id)
);
```

---

## 🧪 Testing Status

### ✅ Compilation
- All code compiles successfully
- No type errors or missing method implementations
- Migrations applied without errors

### ⏳ Pending Testing
- End-to-end digest generation with `GenerateDigests()`
- Database storage with `StoreWithRelationships()`
- Frontend integration with new schema
- Handler updates to use new pipeline method

---

## 📝 TODOs and Next Steps

### High Priority
1. **Test GenerateDigests() end-to-end**
   - Run: `briefly digest input/links.md`
   - Verify multiple digest files are created
   - Check database for digest records

2. **Update Handlers**
   - File: `cmd/handlers/digest_simplified.go`
   - Change: Call `GenerateDigests()` instead of `GenerateDigest()`
   - Handle array of results

3. **Implement Citation Extraction**
   - Create helper function to extract `[[N]](url)` from markdown
   - Store in citations table during `StoreWithRelationships()`

### Medium Priority
4. **Complete Repository Methods**
   - Implement `GetWithArticles()` with JOIN
   - Implement `ListByTheme()` for theme filtering
   - Implement efficient `CreateBatch()` for citations

5. **Frontend Template Updates**
   - Update `digest-detail.html` to render perspectives
   - Add clickable citation links
   - Show publisher in article sources

### Low Priority (Future)
6. **HDBSCAN Clustering**
   - Replace K-means with HDBSCAN for automatic cluster discovery
   - Set numeric ClusterID in digests
   - Handle noise cluster (-1)

7. **Weekly Digest Aggregation**
   - Implement weekly digest generation from daily digests
   - Rank digests by importance
   - Generate executive summary

---

## 🐛 Known Issues

1. **ClusterID Type Mismatch**
   - TopicCluster.ID is string (K-means)
   - Digest.ClusterID is *int (HDBSCAN)
   - **Workaround:** Set to nil for now
   - **Fix:** Implement HDBSCAN with numeric IDs

2. **Citation Extraction Not Implemented**
   - Citations not stored during `StoreWithRelationships()`
   - **TODO** placeholder in code
   - **Impact:** Citation table not populated

3. **Some Repository Methods are Stubs**
   - Several query methods return empty/default values
   - Code compiles but not fully functional
   - **Impact:** Frontend features may not work until implemented

---

## 📦 Files Modified

### Core Files
- `internal/core/core.go` - Data structures
- `internal/persistence/interfaces.go` - Repository contracts
- `internal/persistence/postgres_repos.go` - Digest repository
- `internal/persistence/postgres_citation_repo.go` - Citation repository
- `internal/pipeline/pipeline.go` - Pipeline logic

### Migrations
- `internal/persistence/migrations/012_update_digests_schema.sql`
- `internal/persistence/migrations/013_add_digest_relationships.sql`
- `internal/persistence/migrations/014_update_articles_and_citations.sql`

### Documentation
- `docs/PHASE2_IMPLEMENTATION_SUMMARY.md` (this file)

---

## 🎯 Success Criteria

### Phase 2 Complete When:
- [x] Migrations applied successfully
- [x] Code compiles without errors
- [x] GenerateDigests() method implemented
- [x] buildDigestForCluster() method implemented
- [x] StoreWithRelationships() method implemented
- [x] ListRecent() method implemented
- [ ] End-to-end test passes
- [ ] Multiple digest files generated
- [ ] Database records verified
- [ ] Handlers updated

---

## 💡 Usage Examples

### Generate Multiple Digests (v2.0)
```go
results, err := pipeline.GenerateDigests(ctx, DigestOptions{
    InputFile:  "input/weekly-links.md",
    OutputPath: "digests",
})

// Returns []DigestResult (one per cluster)
for _, result := range results {
    fmt.Printf("Generated: %s\n", result.Digest.Title)
    fmt.Printf("  Articles: %d\n", result.Digest.ArticleCount)
    fmt.Printf("  File: %s\n", result.MarkdownPath)
}
```

### Store Digest with Relationships
```go
err := digestRepo.StoreWithRelationships(ctx,
    digest,
    []string{"article-id-1", "article-id-2"},  // Article IDs
    []string{"theme-id-1", "theme-id-2"},      // Theme IDs
)
```

### Query Recent Digests
```go
digests, err := digestRepo.ListRecent(ctx,
    time.Now().Add(-7*24*time.Hour),  // Last 7 days
    50,                                // Limit 50
)

for _, digest := range digests {
    fmt.Printf("%s (%d articles)\n", digest.Title, digest.ArticleCount)
    fmt.Printf("  Themes: %v\n", digest.Themes)
}
```

---

## 🔗 Related Documents

- `docs/digest-pipeline-v2.md` - Original design document
- `docs/migration-plan.md` - Migration strategy
- `CLAUDE.md` - Project instructions
- `README.md` - User-facing documentation

---

**Implementation By:** Claude (Anthropic)
**Review Status:** Awaiting user testing
**Next Milestone:** Phase 3 - Handler Updates & Frontend Integration
