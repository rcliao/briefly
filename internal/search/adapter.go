package search

import (
	"context"
	"time"
)

// LegacySearchProvider defines the old interface used by the research module
type LegacySearchProvider interface {
	Search(query string, maxResults int) ([]LegacySearchResult, error)
	GetName() string
}

// LegacySearchResult represents the old search result format
type LegacySearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Snippet     string `json:"snippet"`
	Source      string `json:"source"`       // Domain name
	PublishedAt string `json:"published_at"` // Publication date if available
	Rank        int    `json:"rank"`         // Position in search results
}

// LegacyProviderAdapter adapts the new Provider interface to the old LegacySearchProvider interface
type LegacyProviderAdapter struct {
	provider Provider
	timeout  time.Duration
}

// NewLegacyProviderAdapter creates an adapter that wraps a new Provider to implement LegacySearchProvider
func NewLegacyProviderAdapter(provider Provider) *LegacyProviderAdapter {
	return &LegacyProviderAdapter{
		provider: provider,
		timeout:  30 * time.Second,
	}
}

// Search implements LegacySearchProvider.Search using the new Provider interface
func (a *LegacyProviderAdapter) Search(query string, maxResults int) ([]LegacySearchResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), a.timeout)
	defer cancel()

	config := Config{
		MaxResults: maxResults,
		SinceTime:  0, // No time filter for legacy interface
		Language:   "en",
	}

	results, err := a.provider.Search(ctx, query, config)
	if err != nil {
		return nil, err
	}

	// Convert new Result format to legacy LegacySearchResult format
	legacyResults := make([]LegacySearchResult, len(results))
	for i, result := range results {
		legacyResults[i] = LegacySearchResult{
			Title:       result.Title,
			URL:         result.URL,
			Snippet:     result.Snippet,
			Source:      result.Domain,                           // Map Domain to Source
			PublishedAt: result.PublishedAt.Format("2006-01-02"), // Convert time to string
			Rank:        result.Rank,
		}
	}

	return legacyResults, nil
}

// GetName implements LegacySearchProvider.GetName
func (a *LegacyProviderAdapter) GetName() string {
	return a.provider.GetName()
}

// ModernProviderAdapter adapts the old LegacySearchProvider interface to the new Provider interface
type ModernProviderAdapter struct {
	legacyProvider LegacySearchProvider
}

// NewModernProviderAdapter creates an adapter that wraps a LegacySearchProvider to implement Provider
func NewModernProviderAdapter(legacyProvider LegacySearchProvider) *ModernProviderAdapter {
	return &ModernProviderAdapter{
		legacyProvider: legacyProvider,
	}
}

// Search implements Provider.Search using the old LegacySearchProvider interface
func (a *ModernProviderAdapter) Search(ctx context.Context, query string, config Config) ([]Result, error) {
	// Note: The legacy interface doesn't support context or advanced config,
	// so we ignore those and just use maxResults
	legacyResults, err := a.legacyProvider.Search(query, config.MaxResults)
	if err != nil {
		return nil, err
	}

	// Convert legacy LegacySearchResult format to new Result format
	results := make([]Result, len(legacyResults))
	for i, legacyResult := range legacyResults {
		// Parse the published date if available
		var publishedAt time.Time
		if legacyResult.PublishedAt != "" {
			if parsed, err := time.Parse("2006-01-02", legacyResult.PublishedAt); err == nil {
				publishedAt = parsed
			}
		}

		results[i] = Result{
			URL:         legacyResult.URL,
			Title:       legacyResult.Title,
			Snippet:     legacyResult.Snippet,
			Domain:      legacyResult.Source, // Map Source to Domain
			PublishedAt: publishedAt,
			Source:      a.legacyProvider.GetName(),
			Rank:        legacyResult.Rank,
		}
	}

	return results, nil
}

// GetName implements Provider.GetName
func (a *ModernProviderAdapter) GetName() string {
	return a.legacyProvider.GetName()
}
