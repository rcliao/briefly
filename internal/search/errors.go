package search

import "errors"

var (
	// ErrMissingAPIKey is returned when a required API key is not provided
	ErrMissingAPIKey = errors.New("API key is required")
	
	// ErrMissingSearchID is returned when a required search ID is not provided  
	ErrMissingSearchID = errors.New("search ID is required")
	
	// ErrUnsupportedProvider is returned when an unsupported provider type is specified
	ErrUnsupportedProvider = errors.New("unsupported search provider")
	
	// ErrNoResults is returned when a search returns no results
	ErrNoResults = errors.New("no search results found")
	
	// ErrRateLimited is returned when rate limits are exceeded
	ErrRateLimited = errors.New("rate limit exceeded")
	
	// ErrProviderUnavailable is returned when a provider service is unavailable
	ErrProviderUnavailable = errors.New("search provider is currently unavailable")
)