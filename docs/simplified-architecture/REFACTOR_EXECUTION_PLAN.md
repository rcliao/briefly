# Breaking Refactor Execution Plan
## Simplification of Briefly Digest Application

**Status**: Planning Phase
**Target**: Align implementation with actual user workflow
**Impact**: Breaking changes, ~35-40% code reduction

---

## Executive Summary

### Current State
- **29 internal packages** (~32,686 lines of code)
- **8 command handlers** (digest, summarize, research, tui, cache, unified, etc.)
- **Complex architecture** with features not aligned to actual usage
- **Over-engineered** for the core use case

### Target State (Simplified Architecture)
- **11 focused components** (estimated ~18,000-20,000 lines)
- **2 primary commands** (digest, read)
- **Linear pipeline** matching weekly workflow
- **Purpose-built** for quality digest generation

### Key Metrics
- **Code Reduction**: ~35-40% (removing 10,794+ lines of unused features)
- **Package Reduction**: 29 → ~12 packages (59% reduction)
- **Command Simplification**: 8 → 2 primary commands (75% reduction)
- **Complexity Reduction**: Removes 14 unused packages entirely

---

## User Workflow Analysis

### Primary Use Case: Weekly Digest (95% of usage)
1. Collect URLs in markdown file (manual)
2. Run application to fetch & summarize articles
3. Cluster articles by topic similarity
4. Generate executive summary from top 3 per cluster
5. Render markdown digest (LinkedIn-ready)
6. Optional: Generate banner image
7. Copy/paste to LinkedIn (manual)

### Secondary Use Case: Quick Read (5% of usage)
- Single URL → Quick summary with key points

### Features NOT in Workflow (to be removed)
- ❌ Deep research pipeline (manual curation preferred)
- ❌ RSS feed tracking (manual collection)
- ❌ Email/Slack/Discord integration (LinkedIn only)
- ❌ TTS/Audio generation (not used)
- ❌ Terminal UI (not part of workflow)
- ❌ Interactive workflows (automated pipeline preferred)
- ❌ Sentiment analysis, alerts, trends (overengineered)
- ❌ Complex relevance scoring (simple clustering sufficient)

---

## Architecture Gap Analysis

### Component Mapping: Current → Simplified

| Simplified Component | Current Package(s) | Action |
|----------------------|-------------------|--------|
| 1. URL Parser | cmd/handlers/* | **Simplify**: Extract to dedicated parser |
| 2. Content Fetcher | internal/fetch | **Keep**: Already well-structured |
| 3. HTML Extractor | internal/fetch/html.go | **Keep**: Working well |
| 4. PDF Extractor | internal/fetch/pdf.go | **Keep**: Working well |
| 5. YouTube Extractor | internal/fetch/youtube.go | **Keep**: Working well |
| 6. Article Summarizer | internal/llm + scattered | **Refactor**: Centralize logic |
| 7. Embedding Generator | internal/llm | **Keep**: Already exists |
| 8. Topic Clusterer | internal/clustering | **Keep**: Core feature |
| 9. Article Reorderer | internal/ordering | **Keep**: Recently added |
| 10. Executive Summary Generator | scattered/missing | **Create**: New component |
| 11. Markdown Renderer | internal/render + templates | **Simplify**: Merge, LinkedIn-focused |
| 12. Cache Manager | internal/store | **Keep**: Essential for performance |
| 13. Banner Generator | internal/visual | **Keep**: User requested feature |

### Packages to KEEP & REFACTOR

#### ✅ Core Pipeline (Keep with simplification)
- **internal/core** → Core data structures (Article, Summary, Cluster, Digest)
- **internal/fetch** → Content fetching and extraction (well-structured)
- **internal/llm** → LLM operations (centralize summarization logic)
- **internal/clustering** → Topic clustering (core feature)
- **internal/ordering** → Article reordering (recently added)
- **internal/store** → Caching (essential for performance)
- **internal/visual** → Banner generation (user-requested)

#### 🔧 Infrastructure (Keep with minor changes)
- **internal/config** → Configuration management (needed)
- **internal/logger** → Logging (needed)
- **internal/cost** → Cost estimation (nice to have)

#### 🔀 Merge/Simplify
- **internal/render** + **internal/templates** → Merge into `internal/render`
- **internal/services** → Simplify to match new architecture
- **internal/summaries** → Merge into `internal/llm`

### Packages to REMOVE (14 packages, ~10,794 lines)

#### ❌ Research Features (not in workflow)
- **internal/deepresearch** → Replaced by manual curation
- **internal/research** → Not used
- **internal/search** → Not needed

#### ❌ Distribution Channels (LinkedIn only)
- **internal/email** → Not used (LinkedIn posting is manual)
- **internal/messaging** → Slack/Discord not needed
- **internal/tts** → Audio not in workflow

#### ❌ Interactive/UI Features (automated workflow)
- **internal/interactive** → Not in workflow
- **internal/tui** → Not used

#### ❌ Over-Engineered Analytics
- **internal/alerts** → Not needed
- **internal/sentiment** → Not in workflow
- **internal/trends** → Not needed
- **internal/relevance** → Overengineered, simple clustering sufficient

#### ❌ Redundant Features
- **internal/categorization** → Redundant with clustering
- **internal/feeds** → RSS not in workflow

---

## Refactoring Phases

### Phase 1: Foundation & Architecture Setup
**Goal**: Establish new structure without breaking existing functionality
**Duration**: 2-3 sessions

#### 1.1 Create New Package Structure
```
internal/
├── core/           # Core data structures (Article, Summary, Cluster, Digest)
├── parser/         # NEW: URL parsing and validation
├── fetch/          # Content fetching (keep existing)
├── extract/        # Content extraction (refactor from fetch/)
├── summarize/      # NEW: Centralized summarization logic
├── cluster/        # Rename from clustering/
├── order/          # Rename from ordering/
├── narrative/      # NEW: Executive summary generation
├── render/         # Merged: render/ + templates/
├── cache/          # Rename from store/
├── banner/         # Rename from visual/
├── config/         # Keep
├── logger/         # Keep
└── cost/           # Keep (optional)
```

#### 1.2 Create Facade Layer
- Create `internal/pipeline/` package with new simplified API
- Implement pipeline orchestration matching data-flow.md
- Keep old packages working temporarily for testing

#### 1.3 Update Core Data Structures
- Review and simplify `internal/core/core.go`
- Ensure alignment with `data-model.yaml`
- Add missing structures (ClusterTheme, DigestMetadata, etc.)

### Phase 2: Command Simplification
**Goal**: Reduce to 2 primary commands
**Duration**: 1-2 sessions

#### 2.1 Create New Command Structure
```
cmd/
├── briefly/main.go
└── handlers/
    ├── root.go      # Root command with global flags
    ├── digest.go    # NEW: Simplified digest command
    └── read.go      # NEW: Quick read command
```

#### 2.2 Implement Digest Command
- Single command: `briefly digest <input.md> [--with-banner]`
- Flags: `--output`, `--cache`, `--dry-run`, `--with-banner`
- No format flags (LinkedIn markdown only)
- Leverage pipeline facade from Phase 1

#### 2.3 Implement Read Command
- Single command: `briefly read <url>`
- Quick summary output
- Cache-first approach

#### 2.4 Deprecate Old Commands
- Remove: research, unified, tui handlers
- Keep cache command for utility: `briefly cache stats|clear`

### Phase 3: Package Removal & Cleanup
**Goal**: Remove unused packages and dependencies
**Duration**: 1 session

#### 3.1 Remove Feature Packages (14 packages)
```bash
rm -rf internal/alerts
rm -rf internal/categorization
rm -rf internal/deepresearch
rm -rf internal/email
rm -rf internal/feeds
rm -rf internal/interactive
rm -rf internal/messaging
rm -rf internal/relevance
rm -rf internal/research
rm -rf internal/search
rm -rf internal/sentiment
rm -rf internal/trends
rm -rf internal/tts
rm -rf internal/tui
rm -rf internal/summaries  # Merge into llm
```

#### 3.2 Update Dependencies
- Run `go mod tidy` to remove unused dependencies
- Update imports across remaining files
- Remove feature-specific configuration from config/

#### 3.3 Clean Command Handlers
```bash
rm cmd/handlers/research.go
rm cmd/handlers/unified.go
rm cmd/handlers/tui.go
# Keep: root.go, digest.go (new), cache.go
```

### Phase 4: Integration & Testing
**Goal**: Ensure pipeline works end-to-end
**Duration**: 2-3 sessions

#### 4.1 Integration Testing
- Test with user's actual markdown files (input/*.md)
- Verify cache behavior
- Test error handling and graceful degradation
- Validate LinkedIn-ready output format

#### 4.2 Performance Validation
- Benchmark against targets from data-flow.md:
  - Single article: < 5 seconds ✓
  - Full digest (10 articles): < 30 seconds ✓
  - Cache hit ratio: > 80% ✓
  - Clustering: < 1 second ✓

#### 4.3 Update Tests
- Remove tests for deleted packages
- Update integration tests for new pipeline
- Ensure >80% coverage for core pipeline

#### 4.4 Update Documentation
- Update CLAUDE.md with new commands
- Update README.md (if exists)
- Document new architecture in docs/

### Phase 5: Polish & Finalization
**Goal**: Production-ready simplified application
**Duration**: 1 session

#### 5.1 Configuration Cleanup
- Simplify .briefly.yaml to match new features
- Remove unused config sections
- Update config validation

#### 5.2 Error Messages & UX
- Clear, actionable error messages
- Progress indicators for long operations
- Help text and examples

#### 5.3 Final Code Review
- Ensure consistency across packages
- Remove dead code and unused functions
- Verify all imports are necessary

---

## Detailed Component Refactoring

### New Components to Create

#### 1. internal/parser/ (NEW)
**Purpose**: Extract and validate URLs from markdown
**Files**:
- `parser.go`: Main parsing logic
- `parser_test.go`: Unit tests

**Key Functions**:
```go
func ParseMarkdownFile(path string) ([]string, error)
func ParseMarkdownContent(content string) []string
func ValidateURL(url string) error
func NormalizeURL(url string) string
func DeduplicateURLs(urls []string) []string
```

#### 2. internal/summarize/ (NEW)
**Purpose**: Centralized article summarization
**Files**:
- `summarizer.go`: Summarization logic
- `prompts.go`: LLM prompts
- `summarizer_test.go`: Unit tests

**Key Functions**:
```go
func SummarizeArticle(ctx context.Context, article *core.Article) (*core.Summary, error)
func GenerateKeyPoints(content string) ([]string, error)
func ExtractTitle(content string) string
```

**Migration**: Extract logic from internal/llm and handlers

#### 3. internal/narrative/ (NEW)
**Purpose**: Generate executive summaries from clusters
**Files**:
- `generator.go`: Narrative generation
- `prompts.go`: LLM prompts for storytelling
- `generator_test.go`: Unit tests

**Key Functions**:
```go
func GenerateExecutiveSummary(ctx context.Context, clusters []*core.Cluster) (string, error)
func IdentifyClusterThemes(cluster *core.Cluster) string
func SelectTopArticles(cluster *core.Cluster, n int) []*core.Article
```

#### 4. internal/pipeline/ (NEW)
**Purpose**: Orchestrate end-to-end digest generation
**Files**:
- `pipeline.go`: Main orchestration
- `digest.go`: Digest workflow
- `quick_read.go`: Single article workflow
- `pipeline_test.go`: Integration tests

**Key Functions**:
```go
func NewPipeline(config *config.Config) *Pipeline
func (p *Pipeline) GenerateDigest(ctx context.Context, inputFile string, opts DigestOptions) (*core.Digest, error)
func (p *Pipeline) QuickRead(ctx context.Context, url string) (*core.Summary, error)
```

### Packages to Refactor

#### internal/render/ (Merge templates/)
**Changes**:
- Merge internal/templates/ into internal/render/
- Single responsibility: Markdown output
- Remove multi-format support (email, slack, discord, audio)
- Focus on LinkedIn-optimized markdown

**Structure**:
```
internal/render/
├── renderer.go       # Main rendering logic
├── linkedin.go       # LinkedIn-specific formatting
├── templates.go      # Markdown templates (merged from templates/)
└── renderer_test.go
```

#### internal/llm/ (Simplify)
**Changes**:
- Remove scattered summarization logic → Move to internal/summarize/
- Keep: Embedding generation, LLM client
- Simplify: Single LLM provider (Gemini), remove complex fallback logic

#### internal/services/ (Simplify)
**Changes**:
- Reduce to 3 core services:
  1. DigestService (pipeline orchestration)
  2. CacheService (from store)
  3. BannerService (from visual)
- Remove: ResearchService, MessagingService, TTSService, etc.

---

## Migration Strategy

### Breaking Changes
This is a **breaking refactor** with the following impacts:

#### Commands
- **REMOVED**: `briefly research`, `briefly send-slack`, `briefly send-discord`, `briefly generate-tts`, `briefly tui`, `briefly unified`
- **CHANGED**: `briefly digest` (simplified flags, single output format)
- **NEW**: `briefly read` (replaces `briefly summarize` with simpler interface)
- **KEPT**: `briefly cache stats|clear`

#### Configuration
- **REMOVED** config sections:
  - `email.*`
  - `messaging.*`
  - `tts.*`
  - `research.*`
  - `alerts.*`
  - `trends.*`
  - `feeds.*`
- **SIMPLIFIED** config sections:
  - `output.formats` → removed (markdown only)
  - `ai.*` → simplified (Gemini only)

#### API (if any external consumers)
- All internal packages moved/renamed
- Service interfaces completely changed

### Backward Compatibility Strategy
**None.** This is a clean break for simplification.

**Rationale**:
- User is the primary/only user
- Current complexity doesn't match actual usage
- Clean slate enables better architecture

### Rollback Plan
1. Tag current version: `git tag v2.0-before-simplification`
2. Create refactor branch: `git checkout -b simplify-architecture`
3. Keep old code accessible via tags
4. If needed, revert: `git revert <commit-range>`

---

## Testing Strategy

### Unit Tests
- **Maintain coverage**: Keep >80% coverage during refactor
- **Test new components**: All new packages need comprehensive tests
- **Remove obsolete tests**: Delete tests for removed packages

### Integration Tests
- **End-to-end digest**: Test full pipeline with real markdown files
- **Quick read**: Test single URL workflow
- **Cache behavior**: Verify cache hit/miss scenarios
- **Error handling**: Test graceful degradation

### Test Files to Update
```
test/integration/digest_test.go       # Update for new pipeline
test/integration/multiformat_test.go  # REMOVE (single format now)
test/mocks/services_mock.go           # Simplify mocks
```

### User Acceptance Testing
- Run against actual input files in `input/` directory
- Verify output matches expected format
- Validate LinkedIn compatibility
- Test banner generation (optional feature)

---

## Risk Assessment & Mitigation

### High Risk Areas

#### 1. LLM Integration
**Risk**: Centralized summarization logic breaks existing behavior
**Mitigation**:
- Extract and test prompts separately
- Compare outputs with old system
- Keep LLM client interface stable

#### 2. Clustering Algorithm
**Risk**: Changes to clustering affect output quality
**Mitigation**:
- Benchmark against existing clustering
- Test with diverse article sets
- Validate similarity thresholds

#### 3. Cache Compatibility
**Risk**: Cache schema changes break existing cache
**Mitigation**:
- Version cache schema
- Implement migration if needed
- Document cache clearing if necessary

### Medium Risk Areas

#### 4. Content Extraction
**Risk**: Refactoring extractors breaks content parsing
**Mitigation**:
- Keep extractor logic mostly unchanged
- Test with diverse content types (HTML, PDF, YouTube)
- Maintain fallback logic

#### 5. Configuration Management
**Risk**: Config changes break existing setups
**Mitigation**:
- Document all config changes
- Provide migration guide
- Validate config on startup

### Low Risk Areas

#### 6. Command Interface
**Risk**: New commands confuse workflow
**Mitigation**:
- Provide clear help text
- Document migration in CLAUDE.md
- User is primary stakeholder (direct feedback)

---

## Success Criteria

### Functional Requirements
- ✅ Weekly digest generation works end-to-end
- ✅ Quick read command works for single URLs
- ✅ Clustering groups articles correctly
- ✅ Executive summary is coherent and story-driven
- ✅ Output is LinkedIn-ready markdown
- ✅ Banner generation works (optional)
- ✅ Cache reduces redundant API calls

### Non-Functional Requirements
- ✅ Code reduced by 35-40% (~12,000 lines removed)
- ✅ Package count reduced from 29 → ~12 (59% reduction)
- ✅ Command count reduced from 8 → 2 primary commands
- ✅ Single article summary < 5 seconds
- ✅ Full digest (10 articles) < 30 seconds
- ✅ Cache hit ratio > 80%
- ✅ Test coverage > 80%

### Quality Requirements
- ✅ Clear linear data flow (no spaghetti)
- ✅ Single responsibility per component
- ✅ Minimal dependencies between packages
- ✅ Comprehensive error handling
- ✅ LinkedIn-optimized output format
- ✅ Documentation updated

---

## Timeline Estimate

| Phase | Tasks | Estimated Duration |
|-------|-------|-------------------|
| Phase 1: Foundation | New structure, facade, data models | 2-3 sessions |
| Phase 2: Commands | Simplify to 2 commands | 1-2 sessions |
| Phase 3: Cleanup | Remove 14 packages | 1 session |
| Phase 4: Integration | Testing and validation | 2-3 sessions |
| Phase 5: Polish | Documentation, UX, final review | 1 session |
| **TOTAL** | | **7-10 sessions** |

**Assumptions**:
- 1 session = 2-4 hours of focused work
- No major blockers or surprises
- LLM behavior remains consistent
- User available for UAT feedback

---

## Next Steps

### Immediate Actions
1. **Review this plan** with user for approval
2. **Tag current version**: `git tag v2.0-before-simplification`
3. **Create refactor branch**: `git checkout -b simplify-architecture`
4. **Begin Phase 1**: Start with new package structure

### Phase 1 First Tasks
1. Create `internal/parser/` package with URL parsing logic
2. Create `internal/pipeline/` package with orchestration facade
3. Create `internal/narrative/` package for executive summaries
4. Test new components in isolation before integration

---

## Appendix

### Reference Documents
- `docs/simplified-architecture/data-flow.md` - Pipeline specification
- `docs/simplified-architecture/components.md` - Component details
- `docs/simplified-architecture/data-model.yaml` - Data structures
- `docs/simplified-architecture/api-contracts.yaml` - Interface contracts
- `docs/simplified-architecture/verification-criteria.md` - Testing specs

### Current Package Analysis
```
Total Packages: 29
Total Lines: ~32,686

KEEP (12 packages, ~18,000 lines):
- core, fetch, llm, clustering, ordering, store, render, templates, visual
- config, logger, cost

REMOVE (14 packages, ~10,794 lines):
- alerts, categorization, deepresearch, email, feeds, interactive
- messaging, relevance, research, search, sentiment, trends, tts, tui

MERGE/REFACTOR (3 packages):
- services (simplify)
- render + templates (merge)
- summaries → llm (merge)
```

### Key Architectural Principles
1. **Simplicity First**: Remove anything not directly supporting user workflow
2. **Linear Pipeline**: Clear data flow matching user's mental model
3. **Quality Over Features**: Excellent digest generation is the only goal
4. **Purpose-Built**: Tool optimized for weekly LinkedIn digest creation
5. **Graceful Degradation**: Partial failures don't stop digest generation

---

**Document Version**: 1.0
**Last Updated**: 2025-09-29
**Author**: Architecture Refactor Planning
**Status**: Ready for Review & Approval