# Unused Packages Analysis

## Summary
As part of the simplified architecture refactor, 18 packages (approximately 18,797 lines of code) have been identified as unused and can be safely removed.

## Package Usage Analysis

### ✅ Used Packages (14)
These packages are actively used by the simplified architecture:

1. **config** - Configuration management
2. **llm** - LLM client for Gemini API
3. **logger** - Structured logging
4. **pipeline** - New orchestration layer (Phase 2)
5. **clustering** - Topic clustering with K-means
6. **core** - Core data structures
7. **fetch** - Content fetching (HTML, PDF, YouTube)
8. **narrative** - Executive summary generation (Phase 1)
9. **parser** - URL parsing from markdown (Phase 1)
10. **render** - Output formatting
11. **store** - SQLite caching
12. **summarize** - Centralized summarization (Phase 3)
13. **templates** - Digest format templates
14. **email** - HTML email templates (used by templates)

### ❌ Unused Packages (18)
These packages are not used by the simplified weekly digest workflow:

1. **alerts** - Alert monitoring system (v2.0 feature, not in core workflow)
2. **categorization** - Article categorization (replaced by clustering)
3. **cost** - API cost estimation (not in simplified workflow)
4. **deepresearch** - Multi-stage deep research pipeline (advanced feature)
5. **feeds** - RSS feed processing (not in user's workflow)
6. **interactive** - Interactive article selection mode (not in simplified workflow)
7. **messaging** - Slack/Discord integration (not in core workflow)
8. **ordering** - Article ordering (stubbed in pipeline, not implemented)
9. **relevance** - Relevance scoring system (not in simplified workflow)
10. **research** - Research query generation (not in core workflow)
11. **search** - Web search integration (not in user's workflow)
12. **sentiment** - Sentiment analysis (v2.0 feature, not in core workflow)
13. **services** - Service layer interfaces (replaced by pipeline interfaces)
14. **summaries** - Legacy summary handling (replaced by summarize package)
15. **trends** - Trend analysis (v2.0 feature, not in core workflow)
16. **tts** - Text-to-speech generation (not in core workflow)
17. **tui** - Terminal UI browser (not in simplified workflow)
18. **visual** - Banner generation (stubbed for future, not currently implemented)

## Dependency Analysis

### Direct Dependencies (from simplified commands)
```
cmd/handlers/digest_simplified.go → config, llm, logger, pipeline
cmd/handlers/read_simplified.go → config, llm, logger, pipeline
cmd/handlers/root_simplified.go → config
```

### Transitive Dependencies (from pipeline)
```
internal/pipeline → clustering, core, fetch, narrative, parser, render, store, summarize, templates
internal/templates → email
```

## Removal Strategy

### Phase 5.1: Safe Removal
Remove packages with no external dependencies first:
- alerts, cost, deepresearch, feeds, interactive, messaging, ordering
- relevance, research, search, sentiment, summaries, trends, tts, tui, visual

### Phase 5.2: Verification
1. Remove all 18 packages
2. Run `go mod tidy` to clean dependencies
3. Build application: `go build ./cmd/briefly`
4. Run tests: `go test ./...`
5. Verify commands work: `./briefly digest --help`

## Expected Impact

### Lines of Code Reduction
- Before: ~32 internal packages
- After: ~14 internal packages
- Reduction: 56% fewer packages
- Lines removed: ~18,797 lines

### Maintenance Benefits
- Simpler codebase focused on core workflow
- Faster builds and tests
- Easier onboarding for contributors
- Reduced cognitive load

### User Workflow Alignment
The remaining packages directly support the user's stated workflow:
1. Collect URLs (manual) → **parser**
2. Fetch & summarize articles → **fetch**, **summarize**, **llm**
3. Cluster by topic → **clustering**
4. Generate executive summary → **narrative**
5. Render digest → **templates**, **render**
6. Optional: Generate banner → (future: visual)
7. Copy to LinkedIn (manual)

## Notes
- **visual** package: Kept for potential future banner implementation, but currently stubbed
- **ordering** package: Currently stubbed in pipeline, not using actual ordering logic
- **email** package: Kept because templates package uses it for HTML email rendering
