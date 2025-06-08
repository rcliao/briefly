package deepresearch

import (
	"context"
	"fmt"
	"time"

	"briefly/internal/core"
	"briefly/internal/search"
)

// ResearchBrief represents the output of a deep research operation
type ResearchBrief struct {
	ID               string                 `json:"id"`
	Topic            string                 `json:"topic"`
	ExecutiveSummary string                 `json:"executive_summary"`
	DetailedFindings []DetailedFinding      `json:"detailed_findings"`
	OpenQuestions    []string               `json:"open_questions"`
	Sources          []Source               `json:"sources"`
	SubQueries       []string               `json:"sub_queries"`
	GeneratedAt      time.Time              `json:"generated_at"`
	Config           ResearchConfig         `json:"config"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// DetailedFinding represents a specific finding from the research
type DetailedFinding struct {
	Topic      string  `json:"topic"`
	Content    string  `json:"content"`
	Citations  []int   `json:"citations"` // References to Sources array indices
	Confidence float64 `json:"confidence"`
}

// Source represents a research source with citation information
type Source struct {
	ID          string    `json:"id"`
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Domain      string    `json:"domain"`
	Content     string    `json:"content"`
	RetrievedAt time.Time `json:"retrieved_at"`
	Relevance   float64   `json:"relevance"`
	Type        string    `json:"type"` // "news", "paper", "blog", "repo", etc.
}

// ResearchConfig holds configuration for deep research operations
type ResearchConfig struct {
	MaxSources     int           `json:"max_sources"`
	SinceTime      time.Duration `json:"since_time"`
	Model          string        `json:"model"`
	SearchProvider string        `json:"search_provider"`
	UseJavaScript  bool          `json:"use_javascript"`
	RefreshCache   bool          `json:"refresh_cache"`
	OutputHTML     bool          `json:"output_html"`
}

// Planner interface defines the topic decomposition functionality
type Planner interface {
	DecomposeTopicSubQueries(ctx context.Context, topic string) ([]string, error)
}

// SearchProvider is an alias for the shared search provider interface
type SearchProvider = search.Provider

// SearchConfig is an alias for the shared search config
type SearchConfig = search.Config

// SearchResult is an alias for the shared search result
type SearchResult = search.Result

// ContentFetcher interface defines content retrieval functionality
type ContentFetcher interface {
	FetchContent(ctx context.Context, url string, useJS bool) (*core.Article, error)
}

// Ranker interface defines content ranking functionality
type Ranker interface {
	RankSources(ctx context.Context, sources []Source, topic string) ([]Source, error)
}

// Synthesizer interface defines brief generation functionality
type Synthesizer interface {
	SynthesizeBrief(ctx context.Context, topic string, sources []Source, subQueries []string) (*ResearchBrief, error)
}

// ResearchEngine orchestrates the deep research process
type ResearchEngine struct {
	planner     Planner
	searcher    SearchProvider
	fetcher     ContentFetcher
	ranker      Ranker
	synthesizer Synthesizer
}

// NewResearchEngine creates a new research engine with the provided components
func NewResearchEngine(planner Planner, searcher SearchProvider, fetcher ContentFetcher, ranker Ranker, synthesizer Synthesizer) *ResearchEngine {
	return &ResearchEngine{
		planner:     planner,
		searcher:    searcher,
		fetcher:     fetcher,
		ranker:      ranker,
		synthesizer: synthesizer,
	}
}

// Research performs a complete deep research operation
func (e *ResearchEngine) Research(ctx context.Context, topic string, config ResearchConfig) (*ResearchBrief, error) {
	// Step 1: Decompose topic into sub-queries
	subQueries, err := e.planner.DecomposeTopicSubQueries(ctx, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to decompose topic: %w", err)
	}

	// Step 2: Search for content using each sub-query
	var allSources []Source
	searchConfig := SearchConfig{
		MaxResults: config.MaxSources / len(subQueries), // Distribute max sources across queries
		SinceTime:  config.SinceTime,
		Language:   "en",
	}

	for _, query := range subQueries {
		results, err := e.searcher.Search(ctx, query, searchConfig)
		if err != nil {
			// Log error but continue with other queries
			continue
		}

		// Step 3: Fetch content for each search result
		for _, result := range results {
			article, err := e.fetcher.FetchContent(ctx, result.URL, config.UseJavaScript)
			if err != nil {
				// Log error but continue with other sources
				continue
			}

			source := Source{
				ID:          article.ID,
				URL:         result.URL,
				Title:       result.Title,
				Domain:      result.Domain,
				Content:     article.CleanedText,
				RetrievedAt: article.DateFetched,
				Type:        inferSourceType(result.Domain),
			}
			allSources = append(allSources, source)
		}
	}

	// Step 4: Rank and filter sources
	rankedSources, err := e.ranker.RankSources(ctx, allSources, topic)
	if err != nil {
		return nil, fmt.Errorf("failed to rank sources: %w", err)
	}

	// Limit to max sources
	if len(rankedSources) > config.MaxSources {
		rankedSources = rankedSources[:config.MaxSources]
	}

	// Step 5: Synthesize the research brief
	brief, err := e.synthesizer.SynthesizeBrief(ctx, topic, rankedSources, subQueries)
	if err != nil {
		return nil, fmt.Errorf("failed to synthesize brief: %w", err)
	}

	brief.Config = config
	return brief, nil
}

// inferSourceType attempts to categorize the source type based on domain
func inferSourceType(domain string) string {
	switch {
	case contains(domain, []string{"arxiv.org", "doi.org", "pubmed.ncbi.nlm.nih.gov"}):
		return "paper"
	case contains(domain, []string{"github.com", "gitlab.com", "bitbucket.org"}):
		return "repo"
	case contains(domain, []string{"news", "cnn", "bbc", "reuters", "ap", "nytimes"}):
		return "news"
	case contains(domain, []string{"blog", "medium.com", "substack.com"}):
		return "blog"
	default:
		return "web"
	}
}

// contains checks if any of the substrings are present in the main string
func contains(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
