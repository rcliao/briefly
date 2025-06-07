package deepresearch

import (
	"briefly/internal/search"
)

// Re-export search providers from the shared search module for convenience
var (
	NewDuckDuckGoSearchProvider     = search.NewDuckDuckGoProvider
	NewGoogleCustomSearchProvider  = search.NewGoogleProvider  
	NewSerpAPISearchProvider       = search.NewSerpAPIProvider
)