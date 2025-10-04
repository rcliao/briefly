# Reference Numbering Implementation Plan

## Quick Fix vs. Proper Solution

### Option 1: Quick Fix (1-2 hours)
Simply ensure the LLM receives articles in the SAME order as the Sources section will display them.

**Changes needed:**
1. In `cmd/handlers/digest.go`, before building `combinedSummaries`:
   - Apply the same categorization logic that Sources section uses
   - Sort articles by category order
   - Build summaries with this final order

### Option 2: Proper Architectural Fix (4-6 hours)
Implement the Single Source of Truth pattern with proper abstraction.

## Recommended Implementation: Quick Fix First

Since the system is in active use, let's fix the immediate problem first:

### Step 1: Create Ordering Helper (30 min)

```go
// cmd/handlers/digest.go

// orderItemsForSources orders items the same way Sources section will display them
func orderItemsForSources(items []render.DigestData, format string) []render.DigestData {
    if format != "signal" {
        return items // Only signal format has special ordering currently
    }
    
    // Use the same categorization as Sources section
    categoryGroups := groupSignalItemsByCategory(items)
    categoryOrder := []string{
        "🔥 Breaking & Hot", 
        "🛠️ Tools & Platforms", 
        "📊 Analysis & Research", 
        "💰 Business & Economics", 
        "💡 Additional Items",
    }
    
    // Build ordered list
    ordered := []render.DigestData{}
    for _, category := range categoryOrder {
        if categoryItems, exists := categoryGroups[category]; exists {
            ordered = append(ordered, categoryItems...)
        }
    }
    
    return ordered
}
```

### Step 2: Update Combined Summaries Building (15 min)

```go
// In runDigest function, around line 1252

// Order items for consistent references
orderedItems := orderItemsForSources(digestItems, format)

// Build combined summaries with ordered items
var combinedSummaries strings.Builder
if categorized {
    // Use orderedItems instead of digestItems
    categoryGroups := groupSignalItemsByCategory(orderedItems)
    // ... rest remains same but uses orderedItems
} else {
    // Use orderedItems for flat format too
    for i, item := range orderedItems {
        combinedSummaries.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, item.Title))
        // ...
    }
}
```

### Step 3: Update Template to Use Same Order (15 min)

```go
// Ensure templates receive the same ordered items
if format == "signal" {
    // Pass orderedItems instead of digestItems
    renderedContent, digestPath, renderErr = templates.RenderSignalStyleDigest(
        orderedItems, // Use ordered items
        outputDir, 
        finalDigest, 
        template, 
        generatedTitle
    )
}
```

### Step 4: Remove Fix Function (5 min)
- Remove `fixReferenceNumbersForSignal` function
- Remove the call to this function
- Remove debug logging

### Step 5: Test (30 min)
1. Test with small 3-article digest
2. Test with larger 15+ article digest
3. Verify references match in all formats

## Future Improvement: Proper Architecture

After the quick fix is working, consider implementing the full solution:

### 1. Create Dedicated Ordering Module
```go
// internal/ordering/ordering.go
package ordering

type OrderedItem struct {
    Data     render.DigestData
    Position int
    Category string
}

type Orderer interface {
    Order(items []render.DigestData) []OrderedItem
}

type SignalOrderer struct{}
type StandardOrderer struct{}
// etc.
```

### 2. Factory Pattern for Format-Specific Ordering
```go
func GetOrderer(format string) Orderer {
    switch format {
    case "signal":
        return &SignalOrderer{}
    case "newsletter":
        return &NewsletterOrderer{}
    default:
        return &StandardOrderer{}
    }
}
```

### 3. Update All Components to Use Ordered Items
- LLM input builder
- Template renderers
- MyTake system
- Research components

## Testing Strategy

### Unit Tests
```go
func TestSignalOrdering(t *testing.T) {
    items := []render.DigestData{
        {Title: "AI Tool", URL: "..."},      // Should be Tools category
        {Title: "MIT Report", URL: "..."},   // Should be Analysis category
        {Title: "Breaking News", URL: "..."}, // Should be Breaking category
    }
    
    ordered := orderItemsForSources(items, "signal")
    
    // Breaking should come first
    assert.Contains(t, ordered[0].Title, "Breaking")
    // Tools should come second
    assert.Contains(t, ordered[1].Title, "Tool")
    // Analysis should come third
    assert.Contains(t, ordered[2].Title, "Report")
}
```

### Integration Tests
1. Generate digest with known articles
2. Parse the output
3. Verify each reference points to correct article
4. Check both Signal section and Sources section

## Timeline

### Quick Fix (Recommended)
- Implementation: 1-2 hours
- Testing: 30 minutes
- Deployment: Immediate

### Full Architecture
- Design review: 2 hours
- Implementation: 4 hours
- Testing: 2 hours
- Migration: 1 hour
- Total: ~9 hours

## Recommendation

**Start with the Quick Fix** to solve the immediate problem, then plan the proper architectural improvement for a future sprint. The quick fix will:
1. Solve the reference mismatch immediately
2. Be easy to test and verify
3. Not break existing functionality
4. Provide a working baseline for future improvements

The key insight remains: **Order articles ONCE before LLM processing, and use that same order everywhere.**