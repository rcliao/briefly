package search

import (
	"context"
	"fmt"
	"time"
)

// MockProvider implements Provider for testing purposes
type MockProvider struct {
	name    string
	results []Result
}

// NewMockProvider creates a new mock search provider
func NewMockProvider() *MockProvider {
	return &MockProvider{
		name: "Mock",
		results: []Result{
			{
				URL:     "https://example.com/article1",
				Title:   "Example Article 1",
				Snippet: "This is a mock search result for testing purposes.",
				Domain:  "example.com",
				Source:  "Mock",
				Rank:    1,
			},
			{
				URL:     "https://test.org/article2",
				Title:   "Test Article 2", 
				Snippet: "Another mock search result with different content.",
				Domain:  "test.org",
				Source:  "Mock",
				Rank:    2,
			},
			{
				URL:     "https://demo.net/article3",
				Title:   "Demo Article 3",
				Snippet: "Third mock result to simulate multiple search results.",
				Domain:  "demo.net",
				Source:  "Mock",
				Rank:    3,
			},
		},
	}
}

// GetName returns the name of this provider
func (m *MockProvider) GetName() string {
	return m.name
}

// Search returns mock search results
func (m *MockProvider) Search(ctx context.Context, query string, config Config) ([]Result, error) {
	// Simulate some processing time
	time.Sleep(100 * time.Millisecond)
	
	// Limit results based on config
	maxResults := config.MaxResults
	if maxResults <= 0 || maxResults > len(m.results) {
		maxResults = len(m.results)
	}
	
	// Create copies of results with query-specific modifications
	results := make([]Result, maxResults)
	for i := 0; i < maxResults; i++ {
		result := m.results[i]
		// Modify title to include query for demonstration
		result.Title = fmt.Sprintf("%s (for query: %s)", result.Title, query)
		results[i] = result
	}
	
	return results, nil
}

// SetResults allows customization of mock results for testing
func (m *MockProvider) SetResults(results []Result) {
	m.results = results
}

// SetName allows customization of provider name for testing
func (m *MockProvider) SetName(name string) {
	m.name = name
}