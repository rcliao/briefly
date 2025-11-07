# Architectural Redesign: Pipeline Consolidation

## Problem Statement

**Issue:** Duplicate digest generation logic in two places causing:
- Updates being applied to wrong code path
- Inconsistent behavior between implementations
- Difficult maintenance and testing
- Unclear architecture boundaries

**Root Causes:**
1. Business logic embedded in command handlers (violation of clean architecture)
2. No single source of truth for digest generation
3. Pipeline abstraction exists but is bypassed
4. No integration tests to catch divergence

## Current Architecture (Problematic)

```
cmd/handlers/digest_generate.go (363 lines)
├─ runDigestGenerate()
└─ generateDigestsWithClustering() ← ACTUAL IMPLEMENTATION
    ├─ Direct clustering calls
    ├─ Direct narrative calls
    ├─ Direct database operations
    └─ Inline quality metrics (after our fix)

internal/pipeline/pipeline.go (930 lines)
└─ GenerateDigests() ← DEAD CODE (never called)
    ├─ Same clustering logic
    ├─ Same narrative logic
    └─ Quality gates integration
```

**Evidence:**
- `pipeline.GenerateDigests()` only referenced in docs (not code)
- `digest_generate.go` contains 270+ lines of business logic
- Recent improvement applied to wrong file first

## Proposed Architecture (Clean)

### Layered Architecture

```
┌─────────────────────────────────────────────────┐
│  Presentation Layer (cmd/handlers)              │
│  - Thin wrappers around use cases               │
│  - Parse flags, format output                    │
│  - NO business logic                             │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────┐
│  Application Layer (internal/usecases)          │
│  - DigestGenerationUseCase                       │
│  - Orchestrates pipeline with dependencies       │
│  - Single entry point                            │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────┐
│  Domain Layer (internal/pipeline)                │
│  - Pipeline orchestrator (business rules)        │
│  - Quality gates                                 │
│  - Clustering strategy selection                 │
└────────────────┬────────────────────────────────┘
                 │
┌────────────────▼────────────────────────────────┐
│  Infrastructure Layer                            │
│  - persistence (repositories)                    │
│  - llm (LLM clients)                             │
│  - clustering (algorithms)                       │
│  - narrative (generators)                        │
└─────────────────────────────────────────────────┘
```

### Key Principles

1. **Single Responsibility**: Each layer has one reason to change
2. **Dependency Inversion**: High-level modules don't depend on low-level
3. **Interface Segregation**: Small, focused interfaces
4. **DRY**: One implementation of digest generation

## Implementation Plan

### Step 1: Create Use Case Layer (NEW)

**File:** `internal/usecases/generate_digest.go`

```go
package usecases

import (
    "context"
    "briefly/internal/pipeline"
    "briefly/internal/persistence"
    "briefly/internal/llm"
)

// DigestGenerationUseCase orchestrates digest generation from database
type DigestGenerationUseCase struct {
    db       persistence.Database
    pipeline *pipeline.Pipeline
}

// GenerateDigestsFromDB is the single entry point for digest generation
func (uc *DigestGenerationUseCase) GenerateDigestsFromDB(
    ctx context.Context,
    opts DigestGenerationOptions,
) (*DigestGenerationResult, error) {
    // 1. Query articles from database
    // 2. Generate summaries (check cache)
    // 3. Generate embeddings
    // 4. Run pipeline with quality gates
    // 5. Store results
    // 6. Return structured result
}

type DigestGenerationOptions struct {
    SinceDays   int
    ThemeFilter string
    OutputDir   string
    MinArticles int
}

type DigestGenerationResult struct {
    Digests       []*core.Digest
    TotalArticles int
    Duration      time.Duration
    QualityReport *QualityReport
}
```

### Step 2: Refactor Pipeline (CONSOLIDATE)

**File:** `internal/pipeline/pipeline.go`

Keep existing pipeline but:
- Remove `GenerateDigests()` (dead code)
- Add `GenerateDigestsFromArticles()` (consolidate logic)
- Ensure quality gates are integrated
- Add observability hooks

```go
// GenerateDigestsFromArticles is the core digest generation logic
// Called by use case layer after articles are fetched
func (p *Pipeline) GenerateDigestsFromArticles(
    ctx context.Context,
    articles []core.Article,
    options GenerationOptions,
) ([]*core.Digest, error) {
    // 1. Generate summaries
    // 2. Generate embeddings (full content)
    // 3. Run clustering with optimal K
    // 4. Generate cluster narratives
    // 5. Run quality gates (clustering, narrative)
    // 6. Generate digest content with self-critique
    // 7. Run quality gates (digest)
    // 8. Return digests
}
```

### Step 3: Simplify Command Handler (THIN WRAPPER)

**File:** `cmd/handlers/digest_generate.go` (reduce from 830 to ~150 lines)

```go
func runDigestGenerate(ctx context.Context, sinceDays int, ...) error {
    // 1. Load config
    cfg := config.Get()

    // 2. Connect to database
    db, err := persistence.NewPostgresDB(cfg.Database.ConnectionString)
    if err != nil {
        return err
    }
    defer db.Close()

    // 3. Create dependencies
    llmClient, err := llm.NewClient(cfg.AI.Gemini.Model)
    if err != nil {
        return err
    }
    defer llmClient.Close()

    pipeline := buildPipeline(llmClient)

    // 4. Create use case and execute
    useCase := usecases.NewDigestGenerationUseCase(db, pipeline)
    result, err := useCase.GenerateDigestsFromDB(ctx, usecases.DigestGenerationOptions{
        SinceDays:   sinceDays,
        ThemeFilter: themeFilter,
        OutputDir:   outputDir,
        MinArticles: minArticles,
    })

    // 5. Format and display results
    displayResults(result)

    return nil
}
```

### Step 4: Add Integration Tests (PREVENT REGRESSION)

**File:** `test/integration/digest_generation_test.go`

```go
func TestDigestGeneration_EndToEnd(t *testing.T) {
    // Test actual command execution
    // Verify quality gates run
    // Verify self-critique runs
    // Verify metrics are calculated
}

func TestDigestGeneration_QualityImprovements(t *testing.T) {
    // Verify all Phase 1-5 improvements are active
    // Check for vague phrases detection
    // Check for specificity scoring
    // Check for self-critique execution
}
```

## Migration Strategy

### Option A: Big Bang (Risky)
- Rewrite everything at once
- High risk of breaking existing functionality
- Faster to complete

### Option B: Strangler Fig Pattern (Recommended)
1. Create new use case layer alongside existing code
2. Add feature flag to switch between implementations
3. Test new implementation thoroughly
4. Gradually migrate to new architecture
5. Remove old code once confident

```go
// Feature flag approach
if cfg.Features.UseNewPipeline {
    // New architecture
    useCase := usecases.NewDigestGenerationUseCase(db, pipeline)
    result, err := useCase.GenerateDigestsFromDB(ctx, opts)
} else {
    // Old architecture (deprecated)
    digests, err := generateDigestsWithClustering(ctx, db, llmClient, articles)
}
```

### Option C: Incremental Refactoring (Safest)
1. Extract business logic from handler to functions
2. Move functions to pipeline package
3. Create use case wrapper
4. Update handler to use use case
5. Add tests at each step

**Recommended:** Option C (Incremental)

## Benefits of Redesign

### Immediate Benefits
1. ✅ **Single Source of Truth** - One implementation to maintain
2. ✅ **Clear Architecture** - Obvious where code belongs
3. ✅ **Easier Testing** - Use cases are unit-testable
4. ✅ **Better Separation** - Concerns properly separated

### Long-Term Benefits
1. ✅ **Extensibility** - Easy to add new digest types
2. ✅ **Maintainability** - Clear boundaries and responsibilities
3. ✅ **Testability** - Integration tests catch regressions
4. ✅ **Observability** - Single place to add metrics/tracing

## Prevention Mechanisms

### 1. Code Organization Rules

```
cmd/handlers/           ← MAX 150 lines per file
├─ Parse CLI flags
├─ Create dependencies
├─ Call use case
└─ Format output

internal/usecases/      ← NEW: 200-300 lines per file
├─ Orchestrate workflows
├─ Transaction management
└─ Error handling

internal/pipeline/      ← Core business logic
├─ Digest generation
├─ Quality gates
└─ Clustering strategy

internal/*/             ← Domain services
├─ clustering
├─ narrative
├─ quality
└─ summarize
```

### 2. Linting Rules

Add to `.golangci.yml`:

```yaml
linters-settings:
  funlen:
    lines: 100
    statements: 50

  cyclop:
    max-complexity: 15

  maintidx:
    under: 20

issues:
  exclude-rules:
    # Command handlers should be thin (max 150 lines)
    - path: cmd/handlers/
      linters:
        - funlen
      threshold: 150
```

### 3. Architecture Tests

**File:** `test/architecture/structure_test.go`

```go
func TestCommandHandlersAreThin(t *testing.T) {
    // Fail if any handler exceeds 200 lines
    // Fail if handlers contain business logic
}

func TestUseCasesExist(t *testing.T) {
    // Verify use cases for all major operations
}

func TestNoBusinessLogicInHandlers(t *testing.T) {
    // Parse AST and check for complex logic
}
```

### 4. Documentation

Update `CLAUDE.md`:

```markdown
## Architecture Principles

1. **Command Handlers**: Thin wrappers (max 150 lines)
   - Parse flags
   - Create dependencies
   - Call use case
   - Format output

2. **Use Cases**: Orchestration layer (200-300 lines)
   - Single entry points for features
   - Transaction management
   - Error handling

3. **Pipeline**: Core business logic
   - Digest generation
   - Quality gates
   - Strategy selection

4. **Domain Services**: Specialized operations
   - clustering, narrative, quality, etc.
```

### 5. PR Checklist

Add to `.github/PULL_REQUEST_TEMPLATE.md`:

```markdown
## Checklist

- [ ] Business logic is NOT in command handlers
- [ ] Changes applied to use case layer (not handlers)
- [ ] Integration tests updated
- [ ] CLAUDE.md updated if architecture changed
```

## Action Items

### Immediate (This Session)
- [x] Document the problem
- [ ] Create feature flag for new architecture
- [ ] Create `internal/usecases/generate_digest.go` skeleton
- [ ] Add integration test that verifies improvements run

### Short Term (Next Week)
- [ ] Implement use case layer
- [ ] Migrate digest generation to pipeline
- [ ] Add architecture tests
- [ ] Update documentation

### Long Term (This Month)
- [ ] Full migration to new architecture
- [ ] Remove duplicate code
- [ ] Add comprehensive integration tests
- [ ] Establish coding standards

## Success Metrics

1. **Code Duplication**: 0 duplicate implementations
2. **Handler Size**: All handlers < 200 lines
3. **Test Coverage**: 80%+ on use cases and pipeline
4. **Architectural Violations**: 0 business logic in handlers

## Conclusion

This redesign will:
- Prevent future errors like the one we encountered
- Make the codebase more maintainable
- Enable easier testing and validation
- Establish clear architectural boundaries

**Next Step:** Implement incremental refactoring (Option C) starting with use case layer.
