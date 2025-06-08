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

// GoogleProvider implements Provider using Google Custom Search API
type GoogleProvider struct {
	apiKey    string
	searchID  string
	client    *http.Client
	rateLimit time.Duration
	lastCall  time.Time
}

// NewGoogleProvider creates a new Google Custom Search provider
func NewGoogleProvider(apiKey, searchID string) *GoogleProvider {
	return &GoogleProvider{
		apiKey:    apiKey,
		searchID:  searchID,
		client:    &http.Client{Timeout: 30 * time.Second},
		rateLimit: 100 * time.Millisecond, // Google CSE has generous rate limits
	}
}

// GetName returns the name of this provider
func (g *GoogleProvider) GetName() string {
	return "Google Custom Search"
}

// Search performs a search using Google Custom Search API
func (g *GoogleProvider) Search(ctx context.Context, query string, config Config) ([]Result, error) {
	// Respect rate limiting
	if elapsed := time.Since(g.lastCall); elapsed < g.rateLimit {
		time.Sleep(g.rateLimit - elapsed)
	}
	g.lastCall = time.Now()

	// Build API URL
	baseURL := "https://www.googleapis.com/customsearch/v1"
	params := url.Values{}
	params.Set("key", g.apiKey)
	params.Set("cx", g.searchID)
	params.Set("q", query)
	params.Set("num", strconv.Itoa(min(config.MaxResults, 10))) // Google CSE allows max 10 results per request

	// Add time filter if specified
	if config.SinceTime > 0 {
		days := int(config.SinceTime.Hours() / 24)
		switch {
		case days <= 1:
			params.Set("sort", "date:r:"+formatDateFilter(time.Now().AddDate(0, 0, -1)))
		case days <= 7:
			params.Set("sort", "date:r:"+formatDateFilter(time.Now().AddDate(0, 0, -7)))
		case days <= 30:
			params.Set("sort", "date:r:"+formatDateFilter(time.Now().AddDate(0, 0, -30)))
		case days <= 365:
			params.Set("sort", "date:r:"+formatDateFilter(time.Now().AddDate(0, 0, -365)))
		}
	}

	fullURL := baseURL + "?" + params.Encode()

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Google CSE request: %w", err)
	}

	// Execute request
	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute Google CSE request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google CSE request failed with status: %d", resp.StatusCode)
	}

	// Parse JSON response
	var apiResponse struct {
		Items []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"items"`
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse Google CSE response: %w", err)
	}

	// Check for API errors
	if apiResponse.Error.Code != 0 {
		return nil, fmt.Errorf("google CSE API error (%d): %s", apiResponse.Error.Code, apiResponse.Error.Message)
	}

	// Convert to Result format
	var results []Result
	for i, item := range apiResponse.Items {
		result := Result{
			URL:     item.Link,
			Title:   item.Title,
			Snippet: item.Snippet,
			Domain:  g.extractDomain(item.Link),
			Source:  "Google",
			Rank:    i + 1,
		}
		results = append(results, result)
	}

	logger.Info("Google Custom Search completed", "query", query, "results_found", len(results))

	return results, nil
}

// extractDomain extracts the domain name from a URL
func (g *GoogleProvider) extractDomain(urlStr string) string {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	domain := parsed.Hostname()
	// Remove www. prefix
	domain = strings.TrimPrefix(domain, "www.")

	return domain
}

// formatDateFilter formats a time for Google CSE date filtering
func formatDateFilter(t time.Time) string {
	return t.Format("20060102")
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
