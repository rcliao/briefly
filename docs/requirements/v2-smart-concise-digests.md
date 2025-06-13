# Requirements Document: Briefly v2.0 - Smart Concise Digests

## Executive Summary

Transform Briefly from producing verbose, blog-style digests (2,100+ words) to intelligent, bite-sized content (200-500 words) that busy tech professionals can consume in 2-3 minutes. Implement unified relevance scoring architecture that benefits all commands, not just digest generation.

## Problem Statement

**Current Issues:**
- Digests are too long (158 lines ≈ 2,100+ words) for busy readers
- No content quality filtering - all fetched articles included regardless of relevance
- Duplicate relevance scoring logic between research and digest commands
- Executive summaries read like blog posts instead of actionable takeaways
- Alert system generates noise with 10+ low-value notifications

**Target State:**
- 200-500 word digests consumable in 2-3 minutes
- Intelligent content filtering drops irrelevant articles
- Unified relevance scoring shared across all commands
- Scannable, action-oriented format
- High-value alerts only

## Architecture Overview

### Current Relevance Scoring Landscape
- **Research Command**: Has TWO scoring systems
  - Basic Research (`internal/services/research.go`): Keyword + domain scoring
  - Deep Research (`internal/deepresearch/`): Advanced EmbeddingRanker with multi-factor scoring
- **Digest Command**: Zero relevance filtering - includes all articles
- **Problem**: Duplicate logic, no shared abstraction

### Proposed: Unified Relevance Architecture

```
New Package: internal/relevance/
├── interfaces.go          # Core abstractions
├── keyword_scorer.go      # Fast scoring for digest filtering
├── embedding_scorer.go    # Advanced scoring for research ranking  
├── hybrid_scorer.go       # Combined approach
└── profiles.go           # Predefined weight configurations
```

**Integration Points:**
- **Digest**: Filter articles before clustering, prioritize by relevance within word budget
- **Research**: Replace duplicate scoring logic with unified interface
- **TUI**: Rank cached articles by relevance to user queries

## Phase 1: Core Improvements (High Priority) ✅ **COMPLETED**

### REQ-1: Word Count Optimization ✅ **IMPLEMENTED**
**Target**: 200-500 words per digest (vs. current 2,100+ words)

**Implementation:** ✅ **COMPLETED**
- ✅ Updated `internal/templates/templates.go` MaxSummaryLength configurations to word-based limits
- ✅ Added word count tracking and display: "📊 342 words • ⏱️ 2m read"
- ✅ Implemented word distribution strategy:
  - Executive Summary: 100-150 words (truncated with `truncateToWordLimit`)
  - Article summaries: 15-25 words each (all formats optimized)
  - Action items: 5-8 words each in "⚡ Try This Week" section
- ✅ Added format-specific word targets: Brief (200), Standard (400), Newsletter (500), Email (400)

**Success Metrics:** ✅ **ACHIEVED**
- ✅ Consistent <500 word output with configurable `--max-words` flag
- ✅ <3 minute read time with `estimateReadTime()` calculation
- ✅ Word count displayed in digest header with `generateWordCountHeader()`

### REQ-2: Unified Relevance Scoring Architecture ✅ **IMPLEMENTED**
**Create reusable relevance abstraction serving all commands**

**Core Interface:**
```go
type Scorer interface {
    Score(ctx context.Context, content Scorable, criteria Criteria) (Score, error)
    ScoreBatch(ctx context.Context, contents []Scorable, criteria Criteria) ([]Score, error)
}

type Scorable interface {
    GetTitle() string
    GetContent() string  
    GetURL() string
    GetMetadata() map[string]interface{}
}

type Criteria struct {
    Query       string         // Main query/topic
    Keywords    []string       // Important keywords
    Weights     ScoringWeights // Configurable weights
    Context     string         // "digest", "research", "tui"
    Filters     []Filter       // Quality filters
}
```

**Implementation Strategy:** ✅ **COMPLETED**
1. ✅ Created `internal/relevance/` package with clean interfaces (`interfaces.go`)
2. ✅ Implemented KeywordScorer (fast, simple) for digest filtering (`keyword_scorer.go`)
3. 🔄 Extract and enhance existing research scoring logic (Phase 2 - REQ-5)
4. ✅ Created scoring profiles for different contexts (`profiles.go` with DigestWeights, ResearchWeights, InteractiveWeights)

**Files Created:** ✅ **COMPLETED**
- ✅ `internal/relevance/interfaces.go` - Core abstractions (Scorer, Scorable, Criteria, Score)
- ✅ `internal/relevance/profiles.go` - Predefined weight configurations  
- ✅ `internal/relevance/keyword_scorer.go` - Fast keyword-based scoring implementation
- ✅ `internal/relevance/filtering.go` - Threshold filtering and word budget logic
- ✅ `internal/relevance/keyword_scorer_test.go` - Comprehensive test coverage

### REQ-3: Digest Content Filtering ✅ **IMPLEMENTED**
**Apply relevance scoring to filter articles before processing**

**Integration Point**: ✅ `cmd/handlers/digest.go::processArticles()` after summary generation

**Filtering Strategy:** ✅ **IMPLEMENTED**
- ✅ **🔥 Critical**: Relevance ≥ 0.8 (always include)
- ✅ **⭐ Important**: Relevance 0.6-0.8 (include if space permits)  
- ✅ **💡 Optional**: Relevance < 0.6 (exclude)
- ✅ **Word Budget**: Prioritize high-relevance when hitting word limits

**Technical Implementation:** ✅ **COMPLETED**
```go
// ✅ Implemented in cmd/handlers/digest.go::applyRelevanceFiltering()
relevanceScorer := relevance.NewKeywordScorer()
criteria := relevance.DefaultCriteria("digest", digestTheme)
criteria.Threshold = minRelevance // From --min-relevance flag

// Smart theme detection
digestTheme := relevance.InferDigestTheme(scorableContents)

// Filter with word budget management
if maxWords > 0 {
    filterResults, err = relevance.FilterForDigest(ctx, scorer, contents, criteria, maxWords)
} else {
    filterResults, err = relevance.FilterByThreshold(ctx, scorer, contents, criteria)
}
```

**Command Line Integration:** ✅ **COMPLETED**
- ✅ Added `--min-relevance` flag (default: 0.6)
- ✅ Added `--max-words` flag (0 for template default)
- ✅ Added `--enable-filtering` flag (default: true)
- ✅ Progress reporting with filtering statistics
- ✅ Graceful fallback: keeps top 3 articles if all filtered out

### REQ-4: Enhanced Actionability ✅ **IMPLEMENTED**
**Transform passive summaries into actionable insights**

**Format Requirements:** ✅ **IMPLEMENTED**
- ✅ Single action per article (5-8 words max)
- ✅ Specific, implementable recommendations
- ✅ Technology-aware suggestions

**Examples:** ✅ **IMPLEMENTED**
- ✅ "Test the mentioned API in a small project this week"
- ✅ "Evaluate React for your current tech stack"  
- ✅ "Audit one security practice in your current projects"
- ✅ "Profile and optimize one slow function in your codebase"
- ✅ "Add monitoring to one critical service endpoint"

**New Section**: ✅ "⚡ Try This Week" with 2-3 concrete tasks

**Technical Implementation:** ✅ **COMPLETED**
- ✅ Added `renderActionableSection()` function in `internal/templates/templates.go`
- ✅ Technology-specific action generation with `generateActionableItem()`
- ✅ Context-aware fallback actions with `generateFallbackActions()`
- ✅ Tool/library name extraction with pattern matching
- ✅ Pro tip integration: "Start with just one item - small actions lead to big results"
- ✅ Integrated into all action-enabled templates (detailed, newsletter, email)

## 🎯 Phase 1 Implementation Summary

### ✅ **Successfully Delivered**

**Transform Briefly from verbose blog-style digests (2,100+ words) to intelligent, bite-sized content (200-500 words)**

#### Key Achievements:
1. **📊 Word Count Reduction**: 75% reduction in digest length with precise word-based controls
2. **🎯 Smart Filtering**: Automatic relevance scoring filters articles to high-value content only
3. **⚡ Actionable Insights**: Every digest now includes specific, implementable recommendations
4. **🏗️ Unified Architecture**: Reusable relevance scoring system ready for Phase 2 expansion

#### Quantitative Results:
- ✅ **Length Reduction**: 200-500 words (75% reduction from current)
- ✅ **Read Time**: <3 minutes consistently measured with visual indicator
- ✅ **Relevance Quality**: Configurable thresholds with 0.6 default minimum relevance
- ✅ **Alert Quality**: Integration ready for streamlined alerts in Phase 2
- ✅ **Performance**: No degradation in digest generation time

#### Technical Foundation:
- ✅ **New Package**: `internal/relevance/` with 5 modules and comprehensive test coverage
- ✅ **Enhanced Core**: Added `RelevanceScore` field to `core.Article` structure  
- ✅ **Smart Templates**: Word-optimized templates with actionable sections
- ✅ **CLI Integration**: 3 new command flags with backward compatibility
- ✅ **Quality Assurance**: All tests passing, linting clean, production-ready

#### User Experience Improvements:
- ✅ **Scannable Format**: Easy-to-consume 2-3 minute digests
- ✅ **Focused Content**: Only relevant, high-value articles included
- ✅ **Clear Actions**: Specific next steps instead of passive summaries
- ✅ **Progress Transparency**: Filtering statistics and excluded article reporting
- ✅ **Flexible Control**: Granular control over relevance thresholds and word limits

**Ready for Phase 2**: Research command unification, scoring profiles, and alert streamlining.

## Phase 2: Cross-Command Integration (Medium Priority)

### REQ-5: Research Command Unification
**Consolidate duplicate scoring logic with shared interface**

**Current State**: Two separate scoring implementations
**Target State**: Single interface serving all research types

**Migration Strategy:**
1. Extract scoring from `internal/services/research.go`
2. Enhance `internal/deepresearch/` EmbeddingRanker to implement new interface
3. Create HybridScorer combining keyword and embedding approaches
4. Maintain backward compatibility

**Benefits:**
- Consistent scoring across all research operations
- Improved ranking quality through unified algorithms
- Reduced code duplication and maintenance burden

### REQ-6: Scoring Profiles for Different Contexts
**Predefined weight configurations optimized for each use case**

```go
// internal/relevance/profiles.go
var (
    DigestWeights = ScoringWeights{
        ContentRelevance: 0.6,  // High content match weight
        TitleRelevance:   0.3,  // Medium title weight
        Authority:        0.1,  // Low source authority weight
        Recency:         0.0,   // Time not critical
    }
    
    ResearchWeights = ScoringWeights{
        ContentRelevance: 0.4,  // Balanced content weight
        TitleRelevance:   0.2,  // Lower title weight  
        Authority:        0.3,  // High authority weight
        Recency:         0.1,   // Some recency consideration
    }
    
    InteractiveWeights = ScoringWeights{
        ContentRelevance: 0.5,  // Balanced relevance
        TitleRelevance:   0.4,  // High title importance
        Authority:        0.1,  // Low authority weight
        Recency:         0.0,   // Time irrelevant
    }
)
```

### REQ-7: Alert System Streamlining
**Reduce alert noise while maintaining value**

**Current Problem**: Often 10+ low-value alerts with duplicates
**Target State**: Max 2-3 high-value, relevance-filtered alerts

**Implementation:**
- Apply relevance scoring to alert content
- Consolidate similar alerts (remove "LLMs & Related" duplicates)
- Remove low-value INFO alerts and timestamps
- Only show HIGH and CRITICAL priority alerts

## Phase 3: Advanced Features (Lower Priority)

### REQ-8: TUI Command Relevance Integration
**Add relevance-based content discovery to interactive interface**

**Features:**
- Rank cached articles by relevance to user queries
- Surface relevant content based on reading patterns
- Real-time relevance threshold adjustment
- Content discovery suggestions

### REQ-9: Adaptive Scoring Capabilities
**Learning and optimization features**

**Capabilities:**
- Context learning: Adjust weights based on user feedback
- Content type awareness: Different scoring for news vs. tutorials vs. research
- Temporal relevance: Factor content freshness based on topic type
- Performance optimization: Batch processing and smart caching

## Non-Functional Requirements

### NFR-1: Architectural Simplicity
**Maintain clean, maintainable architecture**

**Principles:**
- **Interface-First Design**: Commands depend on interfaces, not implementations
- **Strategy Pattern**: Easy to swap scoring algorithms without changing command logic
- **Minimal Code Footprint**: Single `internal/relevance/` package
- **Plugin Architecture**: New scoring methods can be added without modifying existing code

### NFR-2: Backward Compatibility
**Ensure smooth migration without breaking existing functionality**

**Requirements:**
- Existing functionality continues to work unchanged
- Gradual migration: Commands adopt relevance scoring incrementally
- Feature flags: Relevance scoring can be disabled for debugging
- Default relevance=1.0 for existing cached articles

### NFR-3: Performance Considerations
**Maintain system performance while adding intelligence**

**Implementation:**
- **Caching Strategy**: Relevance scores cached with articles (7-day TTL)
- **Batch Processing**: Score multiple articles in single LLM call when possible
- **Cost Impact**: Minimal - piggyback on existing summarization calls
- **Embeddings Reuse**: Share embedding computations across commands

## Implementation Priority Matrix

| Phase | Effort | Impact | Components |
|-------|--------|--------|------------|
| P0 | Low | High | Word count limits, basic filtering |
| P1 | Medium | High | Relevance interface, digest integration |
| P2 | Medium | Medium | Research unification, alert streamlining |
| P3 | High | Low | TUI integration, adaptive features |

## Success Metrics

### Quantitative Metrics
- **Length Reduction**: 200-500 words (75% reduction from current)
- **Read Time**: <3 minutes consistently measured
- **Relevance Quality**: >80% of included articles score ≥0.6 relevance
- **Alert Quality**: <3 high-value alerts per digest
- **Performance**: No degradation in digest generation time

### Qualitative Metrics
- User feedback on digest focus and actionability
- Reduced user complaints about digest length
- Increased engagement with recommended actions
- Improved perception of digest value

## Technical Implementation Plan

### Files to Modify
- **New Package**: `internal/relevance/` - Core relevance scoring abstraction
- **Core Data**: `internal/core/core.go` - Add RelevanceScore field to Article
- **Digest Logic**: `cmd/handlers/digest.go` - Add relevance scoring integration
- **Templates**: `internal/templates/templates.go` - Add word count controls
- **Research**: `internal/services/research.go` - Migrate to unified interface
- **Deep Research**: `internal/deepresearch/ranker.go` - Implement new interface

### Configuration Updates
```yaml
# .briefly.yaml additions
digest:
  max_words: 400
  min_relevance_threshold: 0.6
  
templates:
  newsletter:
    max_words: 500
    min_relevance_threshold: 0.7
  slack:
    max_words: 200
    min_relevance_threshold: 0.8

relevance:
  scoring_method: "keyword"  # keyword, embedding, hybrid
  cache_ttl: "168h"         # 7 days
  batch_size: 10            # Articles per scoring batch
```

### CLI Enhancements
- Add `--min-relevance` flag for testing different thresholds
- Add `--max-words` flag to override template defaults
- Include relevance filtering in `--dry-run` cost estimation
- Add relevance debugging with `--verbose` flag

## Migration Strategy

### Phase 1: Foundation (Sprint 1)
1. Create `internal/relevance/` package with core interfaces
2. Implement KeywordScorer for fast, simple relevance scoring
3. Add RelevanceScore field to core data structures
4. Integrate basic filtering in digest command

### Phase 2: Enhancement (Sprint 2)
1. Extract existing research scoring logic to new interface
2. Implement EmbeddingScorer for advanced relevance scoring
3. Create HybridScorer combining multiple approaches
4. Add word count controls and display

### Phase 3: Optimization (Sprint 3)
1. Add TUI command relevance integration
2. Implement alert system improvements
3. Add adaptive learning capabilities
4. Performance optimization and caching enhancements

## Risk Mitigation

### Technical Risks
- **LLM API Costs**: Mitigate through batching and caching
- **Performance Impact**: Mitigate through async processing and caching
- **Scoring Accuracy**: Mitigate through multiple scoring algorithms and user feedback

### User Experience Risks
- **Over-Filtering**: Provide relevance threshold controls
- **Loss of Context**: Maintain "deep dive" links for full articles
- **Change Resistance**: Gradual rollout with feature flags

## Conclusion

This requirements document establishes a foundation for transforming Briefly into a truly intelligent, concise digest system. The unified relevance scoring architecture serves as a force multiplier, improving not just digest quality but enhancing research, TUI, and future commands through shared, reusable intelligence.

The phased approach ensures steady progress while maintaining system stability and user experience. Success will be measured not just in reduced word count, but in increased user engagement and actionable value delivered to busy tech professionals.