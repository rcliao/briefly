# Reference Numbering Solution Architecture

## Problem Summary
The current system has mismatched reference numbers because:
1. LLM receives articles in processing order (1,2,3...)
2. Sources section displays articles in categorized order (Tools, Analysis, etc.)
3. Reference numbers don't match between the two

## Proposed Solution: Single Source of Truth

### Core Architecture Change

**Principle: Determine final article order ONCE before LLM processing**

```
┌─────────────────┐
│ Process Articles│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│Categorize & Sort│ ← Single ordering decision point
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Build Summaries │ ← Use final order with positions
│  with Final Order│
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  LLM Generates  │ ← References match final positions
│  Digest Text    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Render Sources  │ ← Use same order as LLM input
│  Section        │
└─────────────────┘
```

## Implementation Details

### 1. Create Unified Ordering Function

```go
// internal/render/ordering.go
package render

type OrderedDigestItem struct {
    render.DigestData
    Position int      // Final position in digest
    Category string   // Category for grouping
}

// OrderArticlesForDigest determines the final order once
func OrderArticlesForDigest(items []DigestData, format string) []OrderedDigestItem {
    // 1. Categorize items
    categorized := categorizeItems(items)
    
    // 2. Define category order based on format
    categoryOrder := getCategoryOrder(format)
    
    // 3. Build ordered list with final positions
    ordered := []OrderedDigestItem{}
    position := 1
    
    for _, category := range categoryOrder {
        if items, exists := categorized[category]; exists {
            for _, item := range items {
                ordered = append(ordered, OrderedDigestItem{
                    DigestData: item,
                    Position:   position,
                    Category:   category,
                })
                position++
            }
        }
    }
    
    return ordered
}
```

### 2. Update Combined Summaries Building

```go
// cmd/handlers/digest.go
func buildCombinedSummaries(orderedItems []render.OrderedDigestItem) string {
    var combinedSummaries strings.Builder
    
    // Group by category for better LLM context
    currentCategory := ""
    for _, item := range orderedItems {
        if item.Category != currentCategory {
            combinedSummaries.WriteString(fmt.Sprintf("\n**Category: %s**\n", item.Category))
            currentCategory = item.Category
        }
        
        // Use FINAL position, not loop index
        combinedSummaries.WriteString(fmt.Sprintf("%d. **%s**\n", item.Position, item.Title))
        combinedSummaries.WriteString(fmt.Sprintf("   Summary: %s\n", item.SummaryText))
        combinedSummaries.WriteString(fmt.Sprintf("   Reference URL: %s\n\n", item.URL))
    }
    
    return combinedSummaries.String()
}
```

### 3. Update Template Rendering

```go
// internal/templates/templates.go
func RenderSignalStyleDigest(orderedItems []render.OrderedDigestItem, ...) {
    // Use the pre-ordered items directly
    for _, item := range orderedItems {
        content.WriteString(fmt.Sprintf("**[%d] %s**\n", item.Position, item.Title))
        // ... rest of rendering
    }
}
```

## Benefits of This Approach

1. **Single Source of Truth**: Article order is determined once
2. **No Post-Processing**: References are correct from the start
3. **Consistent Across Formats**: All digest formats use same ordering logic
4. **Maintainable**: Changes to ordering logic happen in one place
5. **Predictable**: LLM always sees articles in their final order

## Migration Path

### Phase 1: Add New Ordering System
1. Create `OrderedDigestItem` struct
2. Implement `OrderArticlesForDigest` function
3. Add unit tests for ordering logic

### Phase 2: Update LLM Input
1. Modify `buildCombinedSummaries` to use ordered items
2. Ensure LLM prompt includes position information
3. Test with small digests

### Phase 3: Update Templates
1. Modify all template functions to use ordered items
2. Remove old categorization logic from templates
3. Remove `fixReferenceNumbersForSignal` function

### Phase 4: Cleanup
1. Remove old ordering/categorization code
2. Update documentation
3. Add integration tests

## Alternative Approach: Reference by URL

Instead of numbered references, use URL-based references:

```markdown
The MIT report reveals 95% failure rate [mit-report]...
Meta freezes AI hiring [meta-freeze]...

[mit-report]: https://fortune.com/2025/08/18/mit-report...
[meta-freeze]: https://telegraph.co.uk/business/2025/08/21/zuckerberg...
```

Benefits:
- No ordering dependencies
- References always correct
- More semantic

Drawbacks:
- Longer reference markers
- Less conventional format
- May confuse readers

## Recommendation

Implement the **Single Source of Truth** approach:
1. It's cleaner architecturally
2. Minimal changes to existing code
3. References will always be correct
4. Works with all digest formats

The key insight is: **Order articles once, before LLM processing, and use that order everywhere.**