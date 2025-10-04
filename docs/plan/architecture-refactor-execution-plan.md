# Architecture Refactor Execution Plan: Pre-LLM Categorization

## Overview
Transform the digest generation pipeline to apply sorting and clustering BEFORE LLM processing, ensuring consistent reference numbering and clean separation of concerns.

## Current State vs Target State

### Current (Problematic) Flow
```
Articles → Process → LLM Generate → Template Categorize → Render
                                   ↑ (Reference mismatch occurs here)
```

### Target (Clean) Flow  
```
Articles → Process → Categorize/Sort → LLM Generate → Template Render
                     ↑ (Single source of truth established here)
```

## Implementation Phases

### Phase 1: Core Data Structures and Ordering Service (2-3 hours)

#### 1.1 Create New Data Structures
**File: `internal/ordering/types.go`**
```go
type OrderedDigestItem struct {
    DigestData    render.DigestData
    FinalPosition int
    Category      string
    ClusterInfo   *clustering.TopicCluster
    PriorityScore float64
    UserSelected  bool
}

type DigestStructure struct {
    OrderedItems   []OrderedDigestItem
    CategoryGroups map[string][]OrderedDigestItem
    Metadata       DigestMetadata
    FormatRules    FormatRules
}

type FormatRules struct {
    Format            string
    CategoryOrder     []string
    MaxItemsPerCategory int
    UseClustering     bool
    EnableUserSelection bool
}
```

#### 1.2 Implement Article Ordering Service
**File: `internal/ordering/service.go`**
```go
type Service interface {
    CreateOrderedStructure(ctx context.Context, articles []render.DigestData, format string) (*DigestStructure, error)
    ApplyUserSelection(structure *DigestStructure, selected []int) error
    ValidateStructure(structure *DigestStructure) error
}

func NewService(categorizer CategorizationService, clusterer ClusteringService) Service
```

#### 1.3 Category Assignment Logic
**File: `internal/ordering/categorization.go`**
- Move existing categorization logic from templates
- Create consistent category rules across formats
- Implement confidence scoring for category assignments

**Tasks:**
- [ ] Create `internal/ordering/` package
- [ ] Implement `OrderedDigestItem` and `DigestStructure`
- [ ] Build ordering service with categorization
- [ ] Write unit tests for ordering logic
- [ ] Test with sample articles

### Phase 2: Structured Summary Builder (1-2 hours)

#### 2.1 LLM Input Builder
**File: `internal/summaries/builder.go`**
```go
type StructuredBuilder interface {
    BuildLLMInput(structure *DigestStructure, format string) (string, map[int]string, error)
    FormatCategorySection(items []OrderedDigestItem, category string) string
    GenerateReferenceMap(structure *DigestStructure) map[int]string
}
```

#### 2.2 Update LLM Integration
**File: `cmd/handlers/digest.go`**
- Replace `combinedSummaries` building with `StructuredBuilder`
- Ensure LLM receives articles in final order with correct positions
- Maintain reference number consistency

**Tasks:**
- [ ] Create structured summary builder
- [ ] Update `generateStandardOutput()` to use new builder
- [ ] Test LLM input generation with ordered structure
- [ ] Verify reference numbers in LLM output

### Phase 3: Template System Updates (1-2 hours)

#### 3.1 Order-Aware Template Rendering
**File: `internal/templates/ordered_renderer.go`**
```go
func RenderWithOrderedStructure(structure *DigestStructure, llmContent string, template *DigestTemplate) (string, string, error)
func RenderSourcesFromStructure(structure *DigestStructure) string
func RenderCategoriesFromStructure(structure *DigestStructure) string
```

#### 3.2 Update Existing Templates
- Modify `RenderSignalStyleDigest` to use `DigestStructure`
- Update other template functions to respect pre-ordering
- Remove redundant categorization logic

**Tasks:**
- [ ] Create order-aware rendering functions
- [ ] Update signal template to use DigestStructure
- [ ] Update other digest templates
- [ ] Remove old categorization code from templates
- [ ] Test template rendering with new structure

### Phase 4: Integration and Migration (1 hour)

#### 4.1 Update Main Pipeline
**File: `cmd/handlers/digest.go`**
```go
// New flow in generateStandardOutput:
1. Create DigestStructure using OrderingService
2. Build LLM input using StructuredBuilder  
3. Generate digest with LLM
4. Render using order-aware templates
```

#### 4.2 Configuration Updates
**File: `internal/config/config.go`**
- Add format-specific ordering rules
- Configure category definitions
- Set clustering preferences

**Tasks:**
- [ ] Integrate ordering service into main pipeline
- [ ] Replace old ordering logic with new system
- [ ] Update configuration for format-specific rules
- [ ] Test end-to-end digest generation
- [ ] Verify reference consistency across formats

### Phase 5: Cleanup and Testing (30 minutes)

#### 5.1 Remove Deprecated Code
- Remove `orderItemsForSources()` function (quick fix)
- Remove old categorization helpers
- Clean up unused imports

#### 5.2 Comprehensive Testing
- Test all digest formats (signal, newsletter, standard, email)
- Verify reference numbers match between LLM text and sources
- Test with large article sets (15+ articles)
- Performance testing

**Tasks:**
- [ ] Remove quick fix functions
- [ ] Add integration tests
- [ ] Test all supported formats
- [ ] Performance validation
- [ ] Documentation updates

## Implementation Timeline

**Total Estimated Time: 5-8 hours**

1. **Phase 1**: 2-3 hours (Core structures and ordering)
2. **Phase 2**: 1-2 hours (Summary builder) 
3. **Phase 3**: 1-2 hours (Template updates)
4. **Phase 4**: 1 hour (Integration)
5. **Phase 5**: 30 minutes (Cleanup)

## Risk Mitigation

### Backward Compatibility
- Keep existing template functions during transition
- Test with current digest formats before removing old code
- Maintain API compatibility for external callers

### Performance Considerations
- Categorization happens once instead of multiple times
- Caching category assignments for repeated use
- Minimal impact on overall processing time

### Quality Assurance
- Reference numbers must match between LLM and sources 100%
- Category assignments should be consistent and logical
- All existing digest formats must continue working

## Success Criteria

✅ **Reference Accuracy**: LLM references like `[6]` match Sources section `[6]` exactly
✅ **Format Consistency**: All digest formats use same ordering logic  
✅ **Performance**: No regression in processing time
✅ **Maintainability**: Clean separation of concerns with single source of truth
✅ **User Experience**: No visible changes to digest quality or format

## Testing Strategy

### Unit Tests
```go
func TestOrderingService_CreateStructure(t *testing.T)
func TestStructuredBuilder_GenerateReferences(t *testing.T)  
func TestTemplateRenderer_PreserveOrder(t *testing.T)
```

### Integration Tests
```go
func TestEndToEndDigestGeneration(t *testing.T)
func TestReferenceConsistency(t *testing.T)
func TestMultipleFormats(t *testing.T)
```

### Manual Verification
1. Generate signal digest with 10+ articles
2. Verify each `[N]` reference in signal text matches article `[N]` in Sources
3. Test with different input article orders
4. Ensure categories are logical and consistent

This plan transforms the architecture to eliminate reference numbering issues while maintaining clean, maintainable code structure.