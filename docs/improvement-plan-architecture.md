# Architecture Improvement Plan

**Date:** 2025-11-07
**Issue:** Quality improvements applied to wrong code path (pipeline.go instead of digest_generate.go)
**Status:** Documented, Pending Implementation

---

## Executive Summary

During implementation of digest quality improvements (Phases 1-5), we discovered that updates were applied to `internal/pipeline/pipeline.go` which is **never actually called** by the CLI. The real implementation is in `cmd/handlers/digest_generate.go`, which contains 830 lines of business logic that should be in a use case/pipeline layer.

**Impact:**
- Self-critique and quality gates didn't run initially
- Wasted development time fixing wrong code path
- Architectural debt discovered
- Risk of future similar issues

**Resolution:**
- Identified duplicate implementations
- Applied fixes to correct file (`digest_generate.go`)
- Documented architectural redesign plan
- Created this improvement roadmap

---

## Problem Statement

### What Happened

1. **Phase 1-5 Implementations**: Built quality improvements (self-critique, quality gates, metrics)
2. **Integration**: Added to `internal/pipeline/pipeline.go:749-903`
3. **Testing**: User ran `./briefly digest generate --since 1`
4. **Discovery**: Improvements didn't run - wrong code path modified
5. **Root Cause**: Command handler bypasses pipeline, has own implementation

### The Duplicate Implementations

```
❌ UPDATED (But Never Called)
internal/pipeline/pipeline.go
├─ generateDigestContentWithNarratives() [Lines 749-786]
│  └─ Calls GenerateDigestContentWithCritique() ← Added self-critique
├─ Quality gates integration [Lines 198-233]
└─ storeQualityMetrics() [Lines 871-903]

✅ ACTUALLY USED (Missed Initially)
cmd/handlers/digest_generate.go
├─ generateDigestsWithClustering() [Lines 363-636]
│  └─ Called GenerateDigestContent() ← OLD METHOD (no critique)
└─ No quality gates
└─ No quality metrics
```

### Why This Was Confusing

1. **Both looked legitimate** - Well-structured, documented code
2. **No clear ownership** - Unclear which is "source of truth"
3. **No integration tests** - Would have caught the divergence
4. **Documentation referenced pipeline.go** - CLAUDE.md showed it as the main flow
5. **Reasonable assumption** - Pipeline should be the orchestrator

---

## Current Architecture Issues

### Issue 1: Business Logic in Command Handlers (Anti-Pattern)

**File:** `cmd/handlers/digest_generate.go` (830 lines)

**Contains:**
- Article fetching logic (40 lines)
- Summary generation logic (45 lines)
- Embedding generation logic (60 lines)
- Clustering logic (80 lines)
- Narrative generation logic (50 lines)
- Digest assembly logic (120 lines)
- Theme grouping logic (70 lines)
- Database operations (90 lines)

**Should Be:**
```go
func runDigestGenerate(...) error {
    // 1. Parse flags (10 lines)
    // 2. Create dependencies (20 lines)
    // 3. Call use case (5 lines)
    // 4. Format output (20 lines)
    // Total: ~55 lines
}
```

### Issue 2: Duplicate Pipeline Logic

**Evidence:**
- `pipeline.GenerateDigests()` - 180 lines, never called
- `digest_generate.generateDigestsWithClustering()` - 270 lines, actually used
- ~70% code duplication between them

### Issue 3: No Single Source of Truth

When we want to add a feature, which file do we modify?
- ❓ `internal/pipeline/pipeline.go`? (looks correct)
- ❓ `cmd/handlers/digest_generate.go`? (actually works)
- ❗ Both? (creates inconsistency)

### Issue 4: No Protection Against Regression

- No integration tests for digest generation
- No architecture validation tests
- No linting rules for handler complexity
- No documentation of code organization rules

---

## Root Cause Analysis

### Organizational Root Causes

1. **Unclear Architecture Boundaries**
   - No clear rule: "Business logic goes in X"
   - Command handlers evolved to contain logic
   - Pipeline was refactored but not integrated

2. **Missing Abstraction Layer**
   - No "use case" layer between handlers and pipeline
   - Handlers directly orchestrate multiple services
   - Tight coupling between CLI and business logic

3. **Insufficient Testing**
   - No integration tests for CLI commands
   - No tests verifying improvements are active
   - Manual testing only (discovered by user)

4. **Documentation-Code Drift**
   - `CLAUDE.md` describes pipeline architecture
   - Actual code doesn't use pipeline
   - Diagrams show ideal, not actual state

### Technical Root Causes

1. **Dead Code Not Removed**
   - `pipeline.GenerateDigests()` looks active but isn't
   - No warnings about unused methods
   - Easy to assume it's the right place

2. **No Dependency Injection**
   - Handlers create their own dependencies
   - Can't easily swap implementations
   - Hard to test in isolation

3. **Lack of Interface Contracts**
   - No clear contract for "digest generation"
   - Multiple implementations diverge
   - No compile-time enforcement

---

## Proposed Solution

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────┐
│  cmd/handlers/digest_generate.go (~150 lines)           │
│  ├─ Parse CLI flags                                     │
│  ├─ Load config                                         │
│  ├─ Create dependencies (DI container)                  │
│  ├─ Execute: useCase.GenerateDigestsFromDB()            │
│  └─ Format and display results                          │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│  internal/usecases/generate_digest.go (NEW)             │
│  ├─ DigestGenerationUseCase                             │
│  ├─ GenerateDigestsFromDB() - Main entry point          │
│  ├─ Transaction management                              │
│  ├─ Error handling and logging                          │
│  └─ Coordinates: DB → Pipeline → Storage                │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│  internal/pipeline/pipeline.go (CONSOLIDATED)           │
│  ├─ GenerateDigestsFromArticles() - Core logic          │
│  ├─ Quality gates integration                           │
│  ├─ Self-critique always-on                             │
│  ├─ Clustering strategy selection                       │
│  └─ Metrics calculation                                 │
└────────────────────┬────────────────────────────────────┘
                     │
┌────────────────────▼────────────────────────────────────┐
│  Domain Services (existing)                             │
│  ├─ internal/clustering                                 │
│  ├─ internal/narrative                                  │
│  ├─ internal/quality                                    │
│  ├─ internal/summarize                                  │
│  └─ internal/persistence                                │
└─────────────────────────────────────────────────────────┘
```

### Key Principles

1. **Single Source of Truth**: One implementation of digest generation
2. **Thin Handlers**: Command handlers are < 150 lines
3. **Use Case Layer**: Orchestrates workflows, handles transactions
4. **Pipeline Layer**: Core business logic and rules
5. **Testability**: Every layer independently testable

---

## Implementation Roadmap

### Phase 1: Foundation (Week 1)

**Goal:** Create new architecture alongside existing code

#### Tasks

1. **Create Use Case Layer Structure**
   ```
   internal/usecases/
   ├── generate_digest.go       # Main use case
   ├── generate_digest_test.go  # Unit tests
   └── interfaces.go             # Contracts
   ```

2. **Define Interface Contracts**
   ```go
   type DigestGenerationUseCase interface {
       GenerateDigestsFromDB(ctx, opts) (*Result, error)
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

3. **Create Integration Test Suite**
   ```
   test/integration/
   ├── digest_generation_test.go    # End-to-end tests
   ├── quality_improvements_test.go # Verify improvements run
   └── fixtures/                    # Test data
   ```

4. **Add Feature Flag**
   ```go
   // config/config.go
   type FeaturesConfig struct {
       UseNewDigestPipeline bool `yaml:"use_new_digest_pipeline"`
   }
   ```

#### Success Criteria
- [ ] Use case package created with interfaces
- [ ] Integration test framework established
- [ ] Feature flag functional
- [ ] Zero impact on existing functionality

### Phase 2: Migration (Week 2)

**Goal:** Move logic from handler to use case

#### Tasks

1. **Extract Article Fetching**
   - Move from `digest_generate.go:queryClassifiedArticles()`
   - To `usecases.GenerateDigestsFromDB()` step 1

2. **Extract Summary Generation**
   - Move from `digest_generate.go:generateDigestsWithClustering()`
   - To `pipeline.GenerateDigestsFromArticles()` step 1

3. **Extract Clustering Logic**
   - Move from `digest_generate.go` (lines 420-480)
   - To `pipeline.GenerateDigestsFromArticles()` step 2

4. **Extract Narrative Generation**
   - Move from `digest_generate.go` (lines 500-540)
   - To `pipeline.GenerateDigestsFromArticles()` step 3

5. **Integrate Quality Improvements**
   - Quality gates after clustering
   - Self-critique in narrative generation
   - Quality metrics calculation

6. **Update Handler to Use Case**
   ```go
   // cmd/handlers/digest_generate.go (NEW - 150 lines)
   func runDigestGenerate(...) error {
       // Create use case
       useCase := usecases.NewDigestGenerationUseCase(db, pipeline)

       // Execute (behind feature flag)
       if cfg.Features.UseNewDigestPipeline {
           result, err := useCase.GenerateDigestsFromDB(ctx, opts)
       } else {
           result, err := oldGenerateDigestsWithClustering(...)
       }

       // Display results
       displayResults(result)
   }
   ```

#### Success Criteria
- [ ] Use case contains all orchestration logic
- [ ] Handler is < 200 lines
- [ ] Feature flag works (can toggle between implementations)
- [ ] All integration tests pass with new implementation
- [ ] Quality improvements verified in tests

### Phase 3: Validation (Week 3)

**Goal:** Ensure parity and quality

#### Tasks

1. **Run A/B Testing**
   - Generate digests with both implementations
   - Compare outputs for consistency
   - Verify quality metrics identical

2. **Performance Testing**
   - Measure execution time
   - Check memory usage
   - Identify bottlenecks

3. **Integration Testing**
   - Test all CLI flag combinations
   - Test error scenarios
   - Test database failures

4. **Documentation Update**
   - Update `CLAUDE.md` architecture section
   - Update `docs/data-flow.md`
   - Add architecture decision records

#### Success Criteria
- [ ] New implementation produces identical results
- [ ] Performance is equal or better
- [ ] All edge cases handled
- [ ] Documentation accurate

### Phase 4: Cutover (Week 4)

**Goal:** Switch to new implementation, remove old code

#### Tasks

1. **Enable New Pipeline by Default**
   ```yaml
   # .briefly.yaml
   features:
     use_new_digest_pipeline: true
   ```

2. **Monitor Production Usage**
   - Watch for errors
   - Collect user feedback
   - Monitor quality metrics

3. **Remove Old Implementation**
   - Delete `generateDigestsWithClustering()`
   - Remove feature flag (after 1 week stability)
   - Clean up dead code in `pipeline.go`

4. **Add Architecture Guards**
   - Linting rules for handler complexity
   - Architecture tests
   - Pre-commit hooks

#### Success Criteria
- [ ] New pipeline runs in production
- [ ] Zero user-reported issues
- [ ] Old code removed
- [ ] Architecture guards in place

### Phase 5: Hardening (Week 5)

**Goal:** Prevent future occurrences

#### Tasks

1. **Add Architecture Tests**
   ```go
   // test/architecture/structure_test.go
   func TestCommandHandlersAreThin(t *testing.T)
   func TestBusinessLogicInPipeline(t *testing.T)
   func TestNoDuplicateImplementations(t *testing.T)
   ```

2. **Configure Linters**
   ```yaml
   # .golangci.yml
   linters-settings:
     funlen:
       lines: 150  # Max lines for handlers

     cyclop:
       max-complexity: 15

     maintidx:
       under: 20  # Maintainability index
   ```

3. **Update Development Guidelines**
   ```markdown
   # CONTRIBUTING.md

   ## Architecture Rules

   1. Command handlers (cmd/handlers/*):
      - Max 150 lines
      - No business logic
      - Parse flags → call use case → format output

   2. Use cases (internal/usecases/*):
      - Orchestrate workflows
      - Transaction management
      - Error handling

   3. Pipeline (internal/pipeline/*):
      - Core business logic
      - Strategy selection
      - Quality gates
   ```

4. **Add PR Checklist**
   ```markdown
   # .github/PULL_REQUEST_TEMPLATE.md

   ## Architecture Compliance
   - [ ] No business logic in command handlers
   - [ ] Changes applied to correct layer
   - [ ] Integration tests updated
   - [ ] Architecture tests pass
   ```

5. **Set Up Continuous Monitoring**
   - Code complexity metrics
   - Test coverage tracking
   - Architecture violation alerts

#### Success Criteria
- [ ] Architecture tests prevent violations
- [ ] Linters catch issues in CI
- [ ] Documentation clear and comprehensive
- [ ] Team aligned on standards

---

## Prevention Mechanisms

### 1. Code Organization Rules

**Enforce via Linting:**
```yaml
# .golangci.yml
rules:
  - id: handler-complexity
    path: "cmd/handlers/*.go"
    max-lines: 150

  - id: no-business-logic-in-handlers
    path: "cmd/handlers/*.go"
    patterns:
      - "type.*UseCase"     # Should not define use cases
      - "func.*Cluster"     # Should not cluster
      - "func.*Generate.*Summary"  # Should not generate summaries
```

**Directory Structure:**
```
cmd/handlers/          ← Thin wrappers (max 150 lines)
internal/usecases/     ← Orchestration (200-300 lines)
internal/pipeline/     ← Core business logic
internal/*/            ← Domain services
```

### 2. Integration Tests (Catch Regressions)

```go
// test/integration/digest_generation_test.go

func TestDigestGeneration_QualityImprovementsActive(t *testing.T) {
    // Verify self-critique runs
    output := runCommand("digest", "generate", "--since", "1")

    assert.Contains(t, output, "Running self-critique refinement pass")
    assert.Contains(t, output, "Quality improved by critique pass")
    assert.Contains(t, output, "Quality Metrics:")
}

func TestDigestGeneration_QualityGatesRun(t *testing.T) {
    // Verify quality gates execute
    output := runCommand("digest", "generate", "--since", "1")

    assert.Contains(t, output, "Clustering Quality Gate")
    assert.Contains(t, output, "Narrative Quality Gate")
}
```

### 3. Architecture Tests (Enforce Structure)

```go
// test/architecture/structure_test.go

func TestHandlersAreThin(t *testing.T) {
    handlers := findGoFiles("cmd/handlers")

    for _, file := range handlers {
        lines := countLines(file)
        assert.Less(t, lines, 200,
            "Handler %s is too complex (%d lines)", file, lines)
    }
}

func TestNoBusinessLogicInHandlers(t *testing.T) {
    handlers := findGoFiles("cmd/handlers")

    for _, file := range handlers {
        ast := parseFile(file)

        // Check for clustering calls
        assert.NotContains(t, ast, "ClusterArticles",
            "Handler should not call clustering directly")

        // Check for LLM calls
        assert.NotContains(t, ast, "GenerateText",
            "Handler should not call LLM directly")
    }
}
```

### 4. Documentation Standards

**Update CLAUDE.md:**
```markdown
## Architecture Principles

### Command Handlers (cmd/handlers/*)
- **Purpose:** CLI interface only
- **Max Size:** 150 lines
- **Responsibilities:**
  1. Parse flags and arguments
  2. Load configuration
  3. Create dependencies (DI)
  4. Call use case method
  5. Format and display results
- **Forbidden:**
  - Business logic
  - Direct service calls
  - Complex orchestration

### Use Cases (internal/usecases/*)
- **Purpose:** Application workflows
- **Size:** 200-300 lines
- **Responsibilities:**
  1. Coordinate multiple services
  2. Transaction management
  3. Error handling
  4. Logging and metrics
- **Example:** DigestGenerationUseCase

### Pipeline (internal/pipeline/*)
- **Purpose:** Core business logic
- **Responsibilities:**
  1. Digest generation algorithm
  2. Quality gates
  3. Strategy selection
  4. Clustering orchestration
```

### 5. Pre-commit Hooks

```bash
#!/bin/bash
# .git/hooks/pre-commit

echo "Running architecture checks..."

# Check handler complexity
for file in cmd/handlers/*.go; do
    lines=$(wc -l < "$file")
    if [ $lines -gt 200 ]; then
        echo "ERROR: $file is too complex ($lines lines)"
        exit 1
    fi
done

# Run architecture tests
go test ./test/architecture/...

echo "✓ Architecture checks passed"
```

---

## Rollback Plan

### If Issues Arise

**Phase 2-3 (Feature Flag Active):**
1. Disable new pipeline: `use_new_digest_pipeline: false`
2. No code changes needed
3. Old implementation continues working

**Phase 4+ (After Old Code Removed):**
1. Revert merge commit
2. Restore old implementation from git history
3. Hotfix deployed within 1 hour

### Risk Mitigation

1. **Gradual Rollout:**
   - Week 1: Internal testing only
   - Week 2: 10% of users (feature flag)
   - Week 3: 50% of users
   - Week 4: 100% of users

2. **Monitoring:**
   - Error rate tracking
   - Quality metrics comparison
   - Performance monitoring
   - User feedback collection

3. **Escape Hatch:**
   - Feature flag remains for 2 weeks after 100% rollout
   - Can instantly rollback if issues found

---

## Success Metrics

### Objective Metrics

1. **Code Organization:**
   - [ ] Handler LOC: < 150 lines per file
   - [ ] Zero business logic in handlers
   - [ ] Single implementation of digest generation

2. **Test Coverage:**
   - [ ] Integration tests: 80%+ coverage
   - [ ] Architecture tests: 100% pass rate
   - [ ] Quality improvement verification: All tests pass

3. **Code Quality:**
   - [ ] Cyclomatic complexity: < 15 per function
   - [ ] Maintainability index: > 20
   - [ ] Code duplication: < 3%

4. **Performance:**
   - [ ] Execution time: ± 5% of current
   - [ ] Memory usage: No increase
   - [ ] API calls: Same or fewer

### Subjective Metrics

1. **Developer Experience:**
   - [ ] New developers understand architecture in < 1 hour
   - [ ] Clear where to add features
   - [ ] Easy to write tests

2. **Maintenance:**
   - [ ] Bug fix time reduced 50%
   - [ ] Feature addition time reduced 30%
   - [ ] Refactoring confidence increased

---

## Lessons Learned

### What Went Wrong

1. ❌ **Assumed pipeline was used** without verifying
2. ❌ **No integration tests** to catch the error
3. ❌ **Documentation didn't match reality**
4. ❌ **Multiple implementations** without clear ownership
5. ❌ **No architecture enforcement** (linting, tests)

### What Went Right

1. ✅ **User caught the issue** through manual testing
2. ✅ **Quick diagnosis** once issue reported
3. ✅ **Fixed both paths** to ensure consistency
4. ✅ **Documented the problem** comprehensively
5. ✅ **Created improvement plan** to prevent recurrence

### Key Takeaways

1. **Verify assumptions with tests** - Don't assume code is called
2. **One source of truth** - Eliminate duplicate implementations
3. **Architecture matters** - Clean architecture prevents issues
4. **Test at all levels** - Unit, integration, architecture tests
5. **Document actual state** - Keep docs in sync with code

---

## Next Steps

### Immediate Actions (This Week)

1. **Review this plan** with team
2. **Prioritize phases** based on urgency
3. **Assign owners** to each phase
4. **Set up tracking** (GitHub project board)

### Decision Points

**Decision 1: When to start?**
- Option A: Immediately (start Phase 1 now)
- Option B: After current sprint (2 weeks)
- Option C: Next quarter (3 months)

**Decision 2: How aggressive?**
- Option A: Full migration (5 weeks)
- Option B: Gradual migration (3 months)
- Option C: Minimal fixes only (1 week)

**Decision 3: Testing strategy?**
- Option A: TDD - write tests first
- Option B: Parallel - write tests during development
- Option C: After - write tests post-implementation

### Approval Required

- [ ] Architecture approach approved
- [ ] Timeline agreed upon
- [ ] Resources allocated
- [ ] Risks acknowledged

---

## Appendix: Current vs. Proposed Code Comparison

### Current: digest_generate.go (830 lines)

```go
func runDigestGenerate(...) error {
    // Load config (20 lines)

    // Connect to DB (30 lines)

    // Query articles (50 lines)

    // Generate summaries (60 lines)

    // Generate embeddings (70 lines)

    // Run clustering (100 lines)

    // Generate narratives (80 lines)

    // Build digests (120 lines)

    // Store in DB (90 lines)

    // Save markdown (60 lines)

    // Display results (150 lines)
}
```

### Proposed: digest_generate.go (150 lines)

```go
func runDigestGenerate(...) error {
    // Load config
    cfg := config.Get()

    // Create dependencies
    db := createDB(cfg)
    llm := createLLM(cfg)
    pipeline := createPipeline(llm)

    // Execute use case
    useCase := usecases.NewDigestGenerationUseCase(db, pipeline)
    result, err := useCase.GenerateDigestsFromDB(ctx, opts)
    if err != nil {
        return fmt.Errorf("digest generation failed: %w", err)
    }

    // Display results
    displayResults(result)

    return nil
}
```

### New: usecases/generate_digest.go (300 lines)

```go
func (uc *DigestGenerationUseCase) GenerateDigestsFromDB(
    ctx context.Context,
    opts DigestGenerationOptions,
) (*DigestGenerationResult, error) {
    // 1. Query articles
    // 2. Generate summaries (with cache)
    // 3. Generate embeddings
    // 4. Run pipeline with quality gates
    // 5. Store results
    // 6. Return structured result
}
```

### Updated: pipeline/pipeline.go (600 lines)

```go
func (p *Pipeline) GenerateDigestsFromArticles(
    ctx context.Context,
    articles []core.Article,
    options GenerationOptions,
) ([]*core.Digest, error) {
    // 1. Generate summaries
    // 2. Generate embeddings (full content)
    // 3. Run clustering with optimal K
    // 4. Quality gate: clustering
    // 5. Generate cluster narratives
    // 6. Quality gate: narratives
    // 7. Generate digest with self-critique
    // 8. Quality gate: digest
    // 9. Return digests
}
```

---

**End of Document**

*This is a living document. Update as architecture evolves.*
