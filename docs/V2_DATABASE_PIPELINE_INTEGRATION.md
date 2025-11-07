# v2.0 Database Pipeline Integration - COMPLETE

**Date:** 2025-11-06
**Status:** ✅ Integration Complete, Ready for Testing

---

## Summary

Successfully integrated v2.0 enhancements (citations, relationships, structured data) into the **database-driven digest pipeline** (`briefly digest generate`).

The file-based digest pipeline (`briefly digest [file.md]`) has been marked as **deprecated** and will be removed in a future version.

---

## What Was Changed

### 1. Deprecated File-Based Digest (v1.0)

**File:** `cmd/handlers/digest_simplified.go`

Added comprehensive deprecation notice:
```go
// ============================================================================
// ⚠️  DEPRECATED: This is the v1.0 file-based digest pipeline
// ============================================================================
//
// DO NOT MODIFY THIS FILE - It will be removed once v2.0 is fully validated.
// All new features should go into digest_generate.go (v2.0 database pipeline).
```

**User Impact:**
- Command still works but shows `[DEPRECATED]` in help text
- Users encouraged to migrate to `briefly aggregate` + `briefly digest generate`

---

### 2. Enhanced Database Pipeline (v2.0)

**File:** `cmd/handlers/digest_generate.go`

#### Changes Made:

**A. Added Citation Support:**
```go
// Import citation utilities
import "briefly/internal/markdown"

// Inject citations into summary markdown
summaryWithCitations := markdown.InjectCitationURLs(digestContent.ExecutiveSummary, articleList)
```

**B. Updated Digest Structure (v2.0 Fields):**
```go
digest := &core.Digest{
    ID:          uuid.NewString(),
    Title:       digestContent.Title,
    Summary:     summaryWithCitations,  // v2.0: [[N]](url) citations
    TLDRSummary: digestContent.TLDRSummary,  // v2.0
    Articles:    articleList,           // v2.0
    ProcessedDate: time.Now(),          // v2.0
    ArticleCount:  len(articles),       // v2.0

    // Legacy fields maintained for backward compatibility
    ArticleGroups: articleGroups,
    Summaries:     summaryList,
    // ...
}
```

**C. Switched to StoreWithRelationships():**
```go
// Build relationships
articleIDs := []string{...}  // All article IDs in digest
themeIDs := []string{...}    // All unique theme IDs

// Use v2.0 storage method (automatically extracts citations)
db.Digests().StoreWithRelationships(ctx, digest, articleIDs, themeIDs)
```

**What StoreWithRelationships() Does:**
1. Stores digest with v2.0 fields (Summary, TLDRSummary, etc.)
2. **Automatically extracts citations** from `Summary` field
3. Stores citation records in `citations` table
4. Creates `digest_articles` relationships
5. Creates `digest_themes` relationships
6. All within a single database transaction

---

## Architecture: v1.0 vs v2.0

### v1.0 (File-Based) - DEPRECATED

```
Input: Markdown file with URLs
   ↓
Direct processing (parse → fetch → cluster → digest)
   ↓
Output: Single markdown file
```

**Limitations:**
- No database storage
- No classification by theme
- No citation tracking
- No relationships
- Single consolidated digest

---

### v2.0 (Database-Driven) - ACTIVE

```
Phase 1: Aggregation
   RSS Feeds + Manual URLs
   ↓
   Fetch & Classify (LLM-based theme matching)
   ↓
   Store in Database (with ThemeID, relevance scores)

Phase 2: Digest Generation
   Query classified articles (by date range, theme)
   ↓
   Group by theme → Generate summaries
   ↓
   Build digest with citations [[N]](url)
   ↓
   StoreWithRelationships (digest + citations + article/theme links)
   ↓
   Output: Database + Markdown file
```

**Advantages:**
- Persistent storage
- LLM theme classification
- Citation extraction & tracking
- Article/theme relationships
- Multiple digests from same data
- Web UI rendering

---

## v2.0 Features Status

| Feature | Status | Notes |
|---------|--------|-------|
| **Database Schema** | ✅ Complete | v2.0 tables exist |
| **Data Structures** | ✅ Complete | `core.Digest` has v2.0 fields |
| **Citation Extraction** | ✅ Complete | `markdown.InjectCitationURLs()` |
| **Citation Storage** | ✅ Complete | `StoreWithRelationships()` auto-extracts |
| **Article Relationships** | ✅ Complete | `digest_articles` table |
| **Theme Relationships** | ✅ Complete | `digest_themes` table |
| **Frontend Rendering** | ✅ Complete | Templates support v2.0 fields |
| **Clickable Citations** | ✅ Complete | Converts to anchor links |
| **Theme Classification** | ✅ Complete | LLM-based in `aggregate` command |
| **HDBSCAN Clustering** | ✅ Ready | Can be enabled in pipeline |
| **KeyMoments (structured)** | 🔜 Future | Currently []string, needs v2.0 format |
| **Perspectives** | 🔜 Future | Needs LLM prompt update |

---

## Testing the v2.0 Pipeline

### Quick Test (5 minutes)

```bash
# 1. Add RSS feed
./briefly feed add https://hnrss.org/newest

# 2. Aggregate articles (fetch + classify)
./briefly aggregate --since 24 --max-articles 10

# 3. Generate digest from classified articles
./briefly digest generate --since 1

# 4. View digest
./briefly digest list
DIGEST_ID=$(./briefly digest list --limit 1 | tail -1 | awk '{print $1}')
./briefly digest show $DIGEST_ID

# 5. Test web UI
./briefly serve --port 8080
open http://localhost:8080/digests/$DIGEST_ID
```

### What to Verify

**In CLI Output (`digest show`):**
- [ ] Digest has Title and TL;DR
- [ ] Summary contains citations like `[[1]](https://example.com)`
- [ ] Article count matches expected

**In Database:**
```sql
-- Check digest stored
SELECT id, title, tldr_summary, article_count FROM digests ORDER BY processed_date DESC LIMIT 1;

-- Check citations extracted
SELECT * FROM citations WHERE digest_id = '<digest-id>';

-- Check relationships
SELECT COUNT(*) FROM digest_articles WHERE digest_id = '<digest-id>';
SELECT COUNT(*) FROM digest_themes WHERE digest_id = '<digest-id>';
```

**In Web UI (`http://localhost:8080/digests/<id>`):**
- [ ] Citations render as clickable links `[[1]]`
- [ ] Clicking citation jumps to article on same page
- [ ] Article cards show citation badges
- [ ] Summary markdown renders properly

---

## Migration Path

### For Existing Users (v1.0 → v2.0)

**Old Workflow:**
```bash
# Create markdown file with URLs
echo "https://example.com/article1" > input/links.md
echo "https://example.com/article2" >> input/links.md

# Generate digest
./briefly digest input/links.md
```

**New Workflow (v2.0):**
```bash
# Option 1: Use RSS feeds (recommended)
./briefly feed add https://example.com/rss
./briefly aggregate --since 24
./briefly digest generate --since 7

# Option 2: Submit manual URLs
./briefly manual-url add https://example.com/article1
./briefly manual-url add https://example.com/article2
./briefly aggregate --since 1
./briefly digest generate --since 1
```

---

## Known Limitations

### Currently NOT Implemented in v2.0

1. **Per-Cluster Digests**
   - Current: Generates ONE digest with all classified articles
   - v2.0 Design: Generate MANY digests (one per cluster/topic)
   - Reason: Requires additional clustering step before digest generation
   - Future: Implement in v2.1

2. **Structured KeyMoments**
   - Current: `KeyMoments` are `[]string` (simple text)
   - v2.0 Design: `[]KeyMoment` with Quote + CitationNumber
   - Reason: Requires LLM prompt update to generate structured output
   - Future: Implement in v2.1

3. **Perspectives**
   - Current: Not generated by narrative generator
   - v2.0 Design: Supporting/opposing viewpoints with citations
   - Reason: Requires LLM prompt update
   - Future: Implement in v2.1

### Workarounds

**For Multiple Digests:**
Run `digest generate` with theme filter:
```bash
./briefly digest generate --theme "AI & Machine Learning" --since 7
./briefly digest generate --theme "Cloud & DevOps" --since 7
```

---

## Next Steps

### Immediate (Ready to Test)

1. **Test End-to-End** - Use Quick Test above
2. **Verify Citations** - Check database and web UI
3. **Validate Storage** - Confirm StoreWithRelationships works

### Short-Term (v2.1 Enhancements)

1. **Update Narrative Generator**
   - Generate structured KeyMoments (Quote + CitationNumber)
   - Generate Perspectives (supporting/opposing with citations)
   - Update LLM prompts for structured JSON output

2. **Per-Cluster Digests**
   - Generate one digest per topic cluster
   - Store each as separate digest in database
   - Update web UI to show digest list by date

3. **Citation Improvements**
   - Validate citation numbers match article count
   - Handle missing citations gracefully
   - Add citation analytics (most-cited articles)

---

## File Changes Summary

**Modified:**
1. `cmd/handlers/digest_simplified.go` - Added deprecation notice
2. `cmd/handlers/digest_generate.go` - v2.0 integration (citations, relationships)

**No Changes Required:**
- `internal/markdown/citations.go` - Already complete
- `internal/persistence/postgres_repos.go` - StoreWithRelationships already implemented
- `web/templates/` - Frontend already supports v2.0 fields
- Database schema - Already has v2.0 tables

**Build Status:** ✅ Compiles successfully

---

## Conclusion

The v2.0 database pipeline is now **fully integrated with citations and relationships**. The infrastructure is complete and ready for production testing.

**Key Achievement:** Citations are automatically extracted and stored when using the database-driven workflow!

**Migration Complete:**
- ✅ v1.0 file-based pipeline marked deprecated
- ✅ v2.0 database pipeline has citation support
- ✅ StoreWithRelationships properly wired up
- ✅ Frontend ready to render citations
- 🔜 Ready for end-to-end testing

**Recommendation:** Focus testing on `briefly aggregate` + `briefly digest generate` workflow, as this is the future architecture! 🚀
