package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"briefly/internal/logger"
)

// SerpAPIProvider implements Provider using SerpAPI (premium option)
type SerpAPIProvider struct {
	apiKey    string
	client    *http.Client
	rateLimit time.Duration
	lastCall  time.Time
}

// NewSerpAPIProvider creates a new SerpAPI search provider
func NewSerpAPIProvider(apiKey string) *SerpAPIProvider {
	return &SerpAPIProvider{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimit: 1 * time.Second, // SerpAPI has generous rate limits
	}
}

// GetName returns the name of this provider
func (s *SerpAPIProvider) GetName() string {
	return "SerpAPI"
}

// Search performs a search using SerpAPI
func (s *SerpAPIProvider) Search(ctx context.Context, query string, config Config) ([]Result, error) {
	// Respect rate limiting
	if elapsed := time.Since(s.lastCall); elapsed < s.rateLimit {
		time.Sleep(s.rateLimit - elapsed)
	}
	s.lastCall = time.Now()
	
	// Build API URL
	apiURL := "https://serpapi.com/search"
	params := url.Values{}
	params.Set("q", query)
	params.Set("engine", "google")
	params.Set("api_key", s.apiKey)
	params.Set("num", strconv.Itoa(config.MaxResults))
	
	// Add time filter if specified
	if config.SinceTime > 0 {
		days := int(config.SinceTime.Hours() / 24)
		switch {
		case days <= 1:
			params.Set("tbs", "qdr:d")
		case days <= 7:
			params.Set("tbs", "qdr:w")
		case days <= 30:
			params.Set("tbs", "qdr:m")
		case days <= 365:
			params.Set("tbs", "qdr:y")
		}
	}
	
	fullURL := apiURL + "?" + params.Encode()
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create SerpAPI request: %w", err)
	}
	
	// Execute request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute SerpAPI request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SerpAPI request failed with status: %d", resp.StatusCode)
	}
	
	// Parse JSON response
	var apiResponse struct {
		OrganicResults []struct {
			Title    string `json:"title"`
			Link     string `json:"link"`
			Snippet  string `json:"snippet"`
			Position int    `json:"position"`
		} `json:"organic_results"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse SerpAPI response: %w", err)
	}
	
	// Check for API errors
	if apiResponse.Error.Code != 0 {
		return nil, fmt.Errorf("SerpAPI error (%d): %s", apiResponse.Error.Code, apiResponse.Error.Message)
	}
	
	// Convert to Result format
	var results []Result
	for _, item := range apiResponse.OrganicResults {
		result := Result{
			URL:     item.Link,
			Title:   item.Title,
			Snippet: item.Snippet,
			Domain:  s.extractDomain(item.Link),
			Source:  "SerpAPI",
			Rank:    item.Position,
		}
		results = append(results, result)
	}
	
	logger.Info("SerpAPI search completed", "query", query, "results_found", len(results))
	
	return results, nil
}

// extractDomain extracts the domain name from a URL
func (s *SerpAPIProvider) extractDomain(urlStr string) string {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	
	domain := parsed.Hostname()
	// Remove www. prefix
	if strings.HasPrefix(domain, "www.") {
		domain = domain[4:]
	}
	
	return domain
}