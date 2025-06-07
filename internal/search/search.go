package search

import (
	"context"
	"time"
)

// Provider defines the unified interface for search providers
// This interface supports both simple searches and context-aware searches
type Provider interface {
	// Search performs a search with configuration
	Search(ctx context.Context, query string, config Config) ([]Result, error)
	
	// GetName returns the name of the search provider
	GetName() string
}

// Config holds configuration for search requests
type Config struct {
	MaxResults int           // Maximum number of results to return
	SinceTime  time.Duration // Only return results newer than this duration
	Language   string        // Language preference (e.g., "en", "es")
}

// Result represents a unified search result
type Result struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Snippet     string    `json:"snippet"`
	Domain      string    `json:"domain"`
	PublishedAt time.Time `json:"published_at,omitempty"`
	Source      string    `json:"source"` // Provider-specific source identifier
	Rank        int       `json:"rank"`   // Position in search results
}

// ProviderType represents the type of search provider
type ProviderType string

const (
	ProviderTypeDuckDuckGo     ProviderType = "duckduckgo"
	ProviderTypeGoogle         ProviderType = "google"
	ProviderTypeSerpAPI        ProviderType = "serpapi"
	ProviderTypeMock           ProviderType = "mock"
)

// ProviderFactory creates search providers based on type and configuration
type ProviderFactory struct{}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
}

// CreateProvider creates a search provider of the specified type
func (f *ProviderFactory) CreateProvider(providerType ProviderType, config map[string]string) (Provider, error) {
	switch providerType {
	case ProviderTypeDuckDuckGo:
		return NewDuckDuckGoProvider(), nil
	case ProviderTypeGoogle:
		apiKey, exists := config["api_key"]
		if !exists {
			return nil, ErrMissingAPIKey
		}
		searchID, exists := config["search_id"] 
		if !exists {
			return nil, ErrMissingSearchID
		}
		return NewGoogleProvider(apiKey, searchID), nil
	case ProviderTypeSerpAPI:
		apiKey, exists := config["api_key"]
		if !exists {
			return nil, ErrMissingAPIKey
		}
		return NewSerpAPIProvider(apiKey), nil
	case ProviderTypeMock:
		return NewMockProvider(), nil
	default:
		return nil, ErrUnsupportedProvider
	}
}

// GetAvailableProviders returns a list of available provider types
func (f *ProviderFactory) GetAvailableProviders() []ProviderType {
	return []ProviderType{
		ProviderTypeDuckDuckGo,
		ProviderTypeGoogle,
		ProviderTypeSerpAPI,
		ProviderTypeMock,
	}
}