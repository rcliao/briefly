package relevance

import (
	"context"
)

// Scorer defines the interface for relevance scoring implementations
type Scorer interface {
	// Score calculates relevance score for a single piece of content
	Score(ctx context.Context, content Scorable, criteria Criteria) (Score, error)

	// ScoreBatch calculates relevance scores for multiple pieces of content
	ScoreBatch(ctx context.Context, contents []Scorable, criteria Criteria) ([]Score, error)
}

// Scorable represents content that can be scored for relevance
type Scorable interface {
	GetTitle() string
	GetContent() string
	GetURL() string
	GetMetadata() map[string]interface{}
}

// Criteria defines the parameters for relevance scoring
type Criteria struct {
	Query     string         `json:"query"`     // Main query/topic
	Keywords  []string       `json:"keywords"`  // Important keywords
	Weights   ScoringWeights `json:"weights"`   // Configurable weights
	Context   string         `json:"context"`   // "digest", "research", "tui"
	Filters   []Filter       `json:"filters"`   // Quality filters
	Threshold float64        `json:"threshold"` // Minimum relevance threshold
}

// ScoringWeights defines the relative importance of different scoring factors
type ScoringWeights struct {
	ContentRelevance float64 `json:"content_relevance"` // Weight for content match (0.0-1.0)
	TitleRelevance   float64 `json:"title_relevance"`   // Weight for title match (0.0-1.0)
	Authority        float64 `json:"authority"`         // Weight for source authority (0.0-1.0)
	Recency          float64 `json:"recency"`           // Weight for content freshness (0.0-1.0)
	Quality          float64 `json:"quality"`           // Weight for content quality (0.0-1.0)
}

// Score represents the result of relevance scoring
type Score struct {
	Value      float64                `json:"value"`      // Overall relevance score (0.0-1.0)
	Confidence float64                `json:"confidence"` // Confidence in the score (0.0-1.0)
	Factors    map[string]float64     `json:"factors"`    // Individual factor scores
	Reasoning  string                 `json:"reasoning"`  // Explanation of the score
	Metadata   map[string]interface{} `json:"metadata"`   // Additional scoring metadata
}

// Filter defines content quality filters
type Filter interface {
	Apply(content Scorable) bool
	Name() string
	Description() string
}

// FilterFunc is a function that implements the Filter interface
type FilterFunc struct {
	FilterName string
	FilterDesc string
	Fn         func(Scorable) bool
}

func (f FilterFunc) Apply(content Scorable) bool {
	return f.Fn(content)
}

func (f FilterFunc) Name() string {
	return f.FilterName
}

func (f FilterFunc) Description() string {
	return f.FilterDesc
}

// ScoringMethod represents different scoring algorithm types
type ScoringMethod string

const (
	MethodKeyword   ScoringMethod = "keyword"   // Fast keyword-based scoring
	MethodEmbedding ScoringMethod = "embedding" // Advanced embedding-based scoring
	MethodHybrid    ScoringMethod = "hybrid"    // Combined keyword + embedding approach
)

// RelevanceThreshold defines relevance level categories
const (
	ThresholdCritical  = 0.8 // 🔥 Critical: Always include
	ThresholdImportant = 0.6 // ⭐ Important: Include if space permits
	ThresholdOptional  = 0.4 // 💡 Optional: Usually exclude
	ThresholdMinimum   = 0.2 // Minimum viable relevance
)

// ArticleAdapter adapts core.Article to implement Scorable interface
type ArticleAdapter struct {
	Title    string
	Content  string
	URL      string
	Metadata map[string]interface{}
}

func (a ArticleAdapter) GetTitle() string {
	return a.Title
}

func (a ArticleAdapter) GetContent() string {
	return a.Content
}

func (a ArticleAdapter) GetURL() string {
	return a.URL
}

func (a ArticleAdapter) GetMetadata() map[string]interface{} {
	return a.Metadata
}

// ResearchResultAdapter adapts research results to implement Scorable interface
type ResearchResultAdapter struct {
	Title    string
	Content  string
	URL      string
	Metadata map[string]interface{}
}

func (r ResearchResultAdapter) GetTitle() string {
	return r.Title
}

func (r ResearchResultAdapter) GetContent() string {
	return r.Content
}

func (r ResearchResultAdapter) GetURL() string {
	return r.URL
}

func (r ResearchResultAdapter) GetMetadata() map[string]interface{} {
	return r.Metadata
}
