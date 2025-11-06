# Command Consolidation Proposal

## Problem

Current workflow requires 2 commands for RSS-based digests:
```bash
briefly aggregate --since 24    # Fetch from feeds
briefly digest generate --since 7  # Generate digest
```

This is confusing and creates friction.

## Proposed Streamlined Structure

### Option A: Single Unified Command (Recommended)

```bash
# Fetch from feeds + generate digest in one step
briefly digest --from-feeds --since 7

# Manual URL input (current behavior)
briefly digest input/links.md

# Advanced: fetch without generating
briefly digest --from-feeds --aggregate-only
```

**Implementation:**
- `digest` command detects `--from-feeds` flag
- Runs aggregate step internally (fetch + classify)
- Immediately generates digest from classified articles
- Single command, single output

### Option B: Keep Separate but Make Clear

```bash
# Renamed for clarity
briefly fetch-articles --since 24    # Step 1: Fetch from RSS
briefly generate-digest --since 7     # Step 2: Create digest

# Or use subcommands
briefly article fetch --since 24
briefly article digest --since 7
```

### Option C: Make Aggregate Auto-Generate (Simplest)

```bash
# This does BOTH: fetch + generate digest
briefly aggregate --since 24

# Add flag to skip digest generation
briefly aggregate --since 24 --no-digest
```

## Recommendation: Option A

**Rationale:**
1. `digest` is the primary user intent - "I want a digest"
2. Source (`--from-feeds` vs file) is just an input option
3. Single command = simpler mental model
4. Consistent with existing `digest [file.md]` pattern

## Migration Path

### Phase 1: Add `--from-feeds` to digest command
```go
// cmd/handlers/digest_simplified.go
cmd.Flags().Bool("from-feeds", false, "Generate digest from RSS feed articles")
cmd.Flags().IntVar(&sinceDays, "since", 7, "Include articles from last N days (with --from-feeds)")
```

### Phase 2: Deprecate standalone `aggregate`
- Mark `aggregate` as deprecated in help text
- Add warning: "Use 'briefly digest --from-feeds' instead"
- Keep command functional for 2-3 releases

### Phase 3: Remove `aggregate` command
- Remove from codebase after deprecation period
- Update all documentation

## Benefits

1. **Simpler UX**: One command for RSS workflow
2. **Less confusion**: Clear intent - "generate digest"
3. **Consistent**: Both RSS and file inputs use same command
4. **Flexible**: Can still aggregate without digest if needed

## Implementation Checklist

- [ ] Add `--from-feeds` flag to digest command
- [ ] Implement internal aggregate logic in digest handler
- [ ] Add `--aggregate-only` flag for advanced users
- [ ] Update help text and docs
- [ ] Add deprecation warning to `aggregate` command
- [ ] Update README and examples
- [ ] Test both workflows (RSS + file)
