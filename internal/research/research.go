package research

import (
	"briefly/internal/llm"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
)

// SearchProvider defines the interface for search API providers
type SearchProvider interface {
	Search(query string, maxResults int) ([]SearchResult, error)
	GetName() string
}

// SearchResult represents a search result from a search provider
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Snippet     string `json:"snippet"`
	Source      string `json:"source"`       // Domain name
	PublishedAt string `json:"published_at"` // Publication date if available
	Rank        int    `json:"rank"`         // Position in search results
}

// ResearchSession represents a deep research session
type ResearchSession struct {
	ID             string                 `json:"id"`
	Topic          string                 `json:"topic"`
	MaxDepth       int                    `json:"max_depth"`
	CurrentDepth   int                    `json:"current_depth"`
	Queries        []string               `json:"queries"`         // Generated search queries
	Results        []SearchResult         `json:"results"`         // All collected results
	DiscoveredURLs []string               `json:"discovered_urls"` // URLs to be processed
	StartedAt      time.Time              `json:"started_at"`
	CompletedAt    time.Time              `json:"completed_at"`
	Status         ResearchStatus         `json:"status"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// ResearchStatus represents the status of a research session
type ResearchStatus string

const (
	StatusInitialized ResearchStatus = "initialized"
	StatusInProgress  ResearchStatus = "in_progress"
	StatusCompleted   ResearchStatus = "completed"
	StatusFailed      ResearchStatus = "failed"
)

// DeepResearcher handles LLM-driven research sessions
type DeepResearcher struct {
	llmClient          *llm.Client
	searchProvider     SearchProvider
	maxResultsPerQuery int
}

// NewDeepResearcher creates a new deep researcher
func NewDeepResearcher(llmClient *llm.Client, searchProvider SearchProvider) *DeepResearcher {
	return &DeepResearcher{
		llmClient:          llmClient,
		searchProvider:     searchProvider,
		maxResultsPerQuery: 10, // Default to top 10 results per query
	}
}

// StartResearch begins a new deep research session
func (dr *DeepResearcher) StartResearch(topic string, maxDepth int) (*ResearchSession, error) {
	session := &ResearchSession{
		ID:             fmt.Sprintf("research-%d", time.Now().Unix()),
		Topic:          topic,
		MaxDepth:       maxDepth,
		CurrentDepth:   0,
		Queries:        []string{},
		Results:        []SearchResult{},
		DiscoveredURLs: []string{},
		StartedAt:      time.Now(),
		Status:         StatusInitialized,
		Metadata:       make(map[string]interface{}),
	}

	return session, nil
}

// ExecuteResearch performs the complete research process
func (dr *DeepResearcher) ExecuteResearch(session *ResearchSession) error {
	session.Status = StatusInProgress

	for session.CurrentDepth < session.MaxDepth {
		fmt.Printf("🔍 Research depth %d/%d for topic: %s\n",
			session.CurrentDepth+1, session.MaxDepth, session.Topic)

		// Generate search queries for current iteration
		queries, err := dr.generateSearchQueries(session.Topic, session.CurrentDepth, session.Results)
		if err != nil {
			session.Status = StatusFailed
			return fmt.Errorf("failed to generate search queries: %w", err)
		}

		session.Queries = append(session.Queries, queries...)

		// Execute searches
		for _, query := range queries {
			fmt.Printf("  🔎 Searching: %s\n", query)
			results, err := dr.searchProvider.Search(query, dr.maxResultsPerQuery)
			if err != nil {
				fmt.Printf("  ⚠️ Search failed for query '%s': %s\n", query, err)
				continue
			}

			// Add rank information and filter duplicates
			newResults := dr.filterAndRankResults(results, session.Results)
			session.Results = append(session.Results, newResults...)

			fmt.Printf("  ✅ Found %d new results\n", len(newResults))
		}

		session.CurrentDepth++

		// Add a small delay between iterations to be respectful to APIs
		time.Sleep(1 * time.Second)
	}

	// Extract URLs from results
	session.DiscoveredURLs = dr.extractURLsFromResults(session.Results)
	session.CompletedAt = time.Now()
	session.Status = StatusCompleted

	fmt.Printf("✅ Research completed: %d URLs discovered\n", len(session.DiscoveredURLs))

	return nil
}

// generateSearchQueries uses the LLM to generate relevant search queries
func (dr *DeepResearcher) generateSearchQueries(topic string, depth int, previousResults []SearchResult) ([]string, error) {
	// Check if LLM client is available
	if dr.llmClient == nil {
		fmt.Printf("  📝 Using template-based queries (no LLM available)\n")
		return dr.generateFallbackQueries(topic, depth), nil
	}

	// Use LLM to generate intelligent search queries based on topic and previous results
	prompt := dr.buildQueryGenerationPrompt(topic, depth, previousResults)

	fmt.Printf("  🤖 Generating search queries using LLM...\n")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	resp, err := dr.llmClient.GetGenaiModel().GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		fmt.Printf("  ⚠️ LLM query generation failed: %s\n", err)
		fmt.Printf("  🔄 Falling back to template-based queries\n")
		return dr.generateFallbackQueries(topic, depth), nil
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		fmt.Printf("  ⚠️ LLM returned no queries\n")
		fmt.Printf("  🔄 Falling back to template-based queries\n")
		return dr.generateFallbackQueries(topic, depth), nil
	}

	queriesPart := resp.Candidates[0].Content.Parts[0]
	queriesText, ok := queriesPart.(genai.Text)
	if !ok {
		fmt.Printf("  ⚠️ LLM returned unexpected format\n")
		fmt.Printf("  🔄 Falling back to template-based queries\n")
		return dr.generateFallbackQueries(topic, depth), nil
	}

	// Parse the numbered list of queries
	queries := dr.parseQueriesFromText(string(queriesText))

	// Ensure we have at least some fallback queries if LLM fails
	if len(queries) == 0 {
		fmt.Printf("  ⚠️ Could not parse LLM-generated queries\n")
		fmt.Printf("  🔄 Falling back to template-based queries\n")
		return dr.generateFallbackQueries(topic, depth), nil
	}

	fmt.Printf("  ✅ Generated %d LLM-based queries\n", len(queries))
	return queries, nil
}

// buildQueryGenerationPrompt creates a prompt for LLM to generate search queries
func (dr *DeepResearcher) buildQueryGenerationPrompt(topic string, depth int, previousResults []SearchResult) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf(`Generate %d specific search queries for the topic: "%s"

Requirements:
- Each query should be focused and specific
- Suitable for Google search
- Different perspectives (technical, business, practical)
- Return as a numbered list (1. First query 2. Second query etc.)

`, 3+depth, topic))

	if depth > 0 {
		builder.WriteString("Focus on advanced/specialized aspects since this is a deeper research iteration.\n\n")
	}

	if len(previousResults) > 0 && len(previousResults) <= 3 {
		builder.WriteString("Previous sources found:\n")
		for _, result := range previousResults {
			builder.WriteString(fmt.Sprintf("- %s\n", result.Source))
		}
		builder.WriteString("Generate queries that would find DIFFERENT sources.\n\n")
	}

	builder.WriteString("Queries:")

	return builder.String()
}

// parseQueriesFromText extracts search queries from LLM-generated text
func (dr *DeepResearcher) parseQueriesFromText(text string) []string {
	lines := strings.Split(text, "\n")
	var queries []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip intro text, headers, and markdown
		if strings.HasPrefix(line, "Here are") ||
			strings.HasPrefix(line, "Queries:") ||
			strings.HasPrefix(line, "**") ||
			strings.HasPrefix(line, "#") ||
			strings.Contains(line, "perspective") && len(line) > 50 {
			continue
		}

		// Remove numbering like "1. " or "- "
		line = strings.TrimPrefix(line, "- ")
		if len(line) > 2 && line[1] == '.' && line[0] >= '1' && line[0] <= '9' {
			line = strings.TrimSpace(line[2:])
		}

		// Clean up markdown and backticks
		line = strings.Trim(line, "`")
		line = strings.TrimPrefix(line, "**Technical:** ")
		line = strings.TrimPrefix(line, "**Business:** ")
		line = strings.TrimPrefix(line, "**Practical:** ")

		if line != "" && len(line) > 5 { // Ensure meaningful query length
			queries = append(queries, line)
		}
	}
	return queries
}

// generateFallbackQueries creates fallback queries when LLM fails
func (dr *DeepResearcher) generateFallbackQueries(topic string, depth int) []string {
	queries := []string{
		fmt.Sprintf("%s overview", topic),
		fmt.Sprintf("%s trends 2025", topic),
		fmt.Sprintf("%s best practices", topic),
	}

	// Add depth-specific queries
	if depth > 0 {
		queries = append(queries,
			fmt.Sprintf("%s case studies", topic),
			fmt.Sprintf("%s advanced techniques", topic),
		)
	}

	return queries
}

// filterAndRankResults removes duplicates and adds ranking information
func (dr *DeepResearcher) filterAndRankResults(newResults []SearchResult, existingResults []SearchResult) []SearchResult {
	// Create a map of existing URLs for quick lookup
	existingURLs := make(map[string]bool)
	for _, result := range existingResults {
		existingURLs[result.URL] = true
	}

	var filtered []SearchResult
	for i, result := range newResults {
		// Skip if URL already exists
		if existingURLs[result.URL] {
			continue
		}

		// Add ranking information
		result.Rank = i + 1
		filtered = append(filtered, result)
		existingURLs[result.URL] = true
	}

	return filtered
}

// extractURLsFromResults extracts unique URLs from search results
func (dr *DeepResearcher) extractURLsFromResults(results []SearchResult) []string {
	urlSet := make(map[string]bool)
	var urls []string

	for _, result := range results {
		if !urlSet[result.URL] {
			urls = append(urls, result.URL)
			urlSet[result.URL] = true
		}
	}

	return urls
}

// GenerateLinksFile creates a markdown file with discovered URLs
func (dr *DeepResearcher) GenerateLinksFile(session *ResearchSession, outputPath string) error {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# Deep Research Results: %s\n\n", session.Topic))
	builder.WriteString(fmt.Sprintf("**Research Session:** %s\n", session.ID))
	builder.WriteString(fmt.Sprintf("**Completed:** %s\n", session.CompletedAt.Format("2006-01-02 15:04")))
	builder.WriteString(fmt.Sprintf("**Depth:** %d iterations\n", session.MaxDepth))
	builder.WriteString(fmt.Sprintf("**Queries Used:** %d\n", len(session.Queries)))
	builder.WriteString(fmt.Sprintf("**URLs Found:** %d\n\n", len(session.DiscoveredURLs)))

	builder.WriteString("## Search Queries Used\n\n")
	for i, query := range session.Queries {
		builder.WriteString(fmt.Sprintf("%d. %s\n", i+1, query))
	}
	builder.WriteString("\n")

	builder.WriteString("## Discovered URLs\n\n")
	builder.WriteString("*Note: These URLs were discovered through deep research and can be used as input for digest generation.*\n\n")

	// Group results by source domain for better organization
	domainGroups := make(map[string][]SearchResult)
	for _, result := range session.Results {
		domain := dr.extractDomain(result.URL)
		domainGroups[domain] = append(domainGroups[domain], result)
	}

	for domain, results := range domainGroups {
		builder.WriteString(fmt.Sprintf("### %s\n\n", domain))
		for _, result := range results {
			builder.WriteString(fmt.Sprintf("- [%s](%s)\n", result.Title, result.URL))
			if result.Snippet != "" {
				builder.WriteString(fmt.Sprintf("  *%s*\n", result.Snippet))
			}
			builder.WriteString("\n")
		}
	}

	// Write to file
	content := builder.String()
	return writeToFile(outputPath, content)
}

// extractDomain extracts the domain name from a URL
func (dr *DeepResearcher) extractDomain(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "unknown"
	}
	return parsedURL.Host
}

// CreateLinksForDigest creates a simple URL list file for digest processing
func (dr *DeepResearcher) CreateLinksForDigest(session *ResearchSession, outputPath string) error {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# Deep Research: %s\n\n", session.Topic))
	builder.WriteString("<!-- Generated by deep research - source=deep_research -->\n\n")

	for _, url := range session.DiscoveredURLs {
		builder.WriteString(fmt.Sprintf("%s\n", url))
	}

	return writeToFile(outputPath, builder.String())
}

// MockSearchProvider implements a mock search provider for testing
type MockSearchProvider struct {
	name string
}

// NewMockSearchProvider creates a new mock search provider
func NewMockSearchProvider() *MockSearchProvider {
	return &MockSearchProvider{name: "Mock Search"}
}

// Search implements the SearchProvider interface with mock data
func (msp *MockSearchProvider) Search(query string, maxResults int) ([]SearchResult, error) {
	// Generate mock results based on the query
	results := []SearchResult{
		{
			Title:   fmt.Sprintf("Understanding %s: A Comprehensive Guide", query),
			URL:     fmt.Sprintf("https://example.com/guide-%s", strings.ReplaceAll(query, " ", "-")),
			Snippet: fmt.Sprintf("This comprehensive guide covers everything you need to know about %s, including best practices and latest trends.", query),
			Source:  "example.com",
		},
		{
			Title:   fmt.Sprintf("Latest Trends in %s for 2025", query),
			URL:     fmt.Sprintf("https://techblog.com/trends-%s-2025", strings.ReplaceAll(query, " ", "-")),
			Snippet: fmt.Sprintf("Explore the cutting-edge developments and emerging trends in %s that are shaping the industry.", query),
			Source:  "techblog.com",
		},
		{
			Title:   fmt.Sprintf("Case Study: Implementing %s at Scale", query),
			URL:     fmt.Sprintf("https://research.org/case-study-%s", strings.ReplaceAll(query, " ", "-")),
			Snippet: fmt.Sprintf("Real-world case study demonstrating how organizations successfully implement %s solutions.", query),
			Source:  "research.org",
		},
	}

	// Limit results to maxResults
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results, nil
}

// GetName returns the name of the search provider
func (msp *MockSearchProvider) GetName() string {
	return msp.name
}

// Helper function to write content to file
func writeToFile(path, content string) error {
	// For now, this would need to be implemented using the file system
	// In a real implementation, you'd use os.WriteFile or similar
	fmt.Printf("Would write to file: %s\nContent length: %d bytes\n", path, len(content))
	return nil
}

// SerpAPISearchProvider implements SearchProvider using SerpAPI
// This is a skeleton implementation - you'd need to add actual SerpAPI integration
type SerpAPISearchProvider struct {
	apiKey string
	client *http.Client
}

// NewSerpAPISearchProvider creates a new SerpAPI search provider
func NewSerpAPISearchProvider(apiKey string) *SerpAPISearchProvider {
	return &SerpAPISearchProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Search implements the SearchProvider interface using SerpAPI
func (sap *SerpAPISearchProvider) Search(query string, maxResults int) ([]SearchResult, error) {
	baseURL := "https://serpapi.com/search"
	params := url.Values{}
	params.Add("q", query)
	params.Add("api_key", sap.apiKey)
	params.Add("num", fmt.Sprintf("%d", maxResults))
	params.Add("engine", "google")

	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Create request with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	resp, err := sap.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search API returned status %d", resp.StatusCode)
	}

	// Parse response (this would need to match SerpAPI's actual response format)
	var apiResponse struct {
		OrganicResults []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic_results"`
		Error string `json:"error,omitempty"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	// Check for API errors
	if apiResponse.Error != "" {
		return nil, fmt.Errorf("search API error: %s", apiResponse.Error)
	}

	// Convert to our SearchResult format
	var results []SearchResult
	for i, organic := range apiResponse.OrganicResults {
		result := SearchResult{
			Title:   organic.Title,
			URL:     organic.Link,
			Snippet: organic.Snippet,
			Source:  extractDomainFromURL(organic.Link),
			Rank:    i + 1,
		}
		results = append(results, result)
	}

	return results, nil
}

// GetName returns the name of the search provider
func (sap *SerpAPISearchProvider) GetName() string {
	return "SerpAPI"
}

// Helper function to extract domain from URL
func extractDomainFromURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "unknown"
	}
	return parsedURL.Host
}

// GoogleCustomSearchProvider implements SearchProvider using Google Custom Search API
type GoogleCustomSearchProvider struct {
	apiKey   string
	searchID string
	client   *http.Client
}

// NewGoogleCustomSearchProvider creates a new Google Custom Search provider
func NewGoogleCustomSearchProvider(apiKey, searchID string) *GoogleCustomSearchProvider {
	return &GoogleCustomSearchProvider{
		apiKey:   apiKey,
		searchID: searchID,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// Search implements the SearchProvider interface using Google Custom Search API
func (gcsp *GoogleCustomSearchProvider) Search(query string, maxResults int) ([]SearchResult, error) {
	baseURL := "https://www.googleapis.com/customsearch/v1"
	params := url.Values{}
	params.Add("key", gcsp.apiKey)
	params.Add("cx", gcsp.searchID)
	params.Add("q", query)
	params.Add("num", fmt.Sprintf("%d", min(maxResults, 10))) // Google CSE allows max 10 results per request

	requestURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	// Create request with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	resp, err := gcsp.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search API returned status %d", resp.StatusCode)
	}

	// Parse response (Google Custom Search API format)
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
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	// Check for API errors
	if apiResponse.Error.Code != 0 {
		return nil, fmt.Errorf("search API error (%d): %s", apiResponse.Error.Code, apiResponse.Error.Message)
	}

	// Convert to our SearchResult format
	var results []SearchResult
	for i, item := range apiResponse.Items {
		result := SearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: item.Snippet,
			Source:  extractDomainFromURL(item.Link),
			Rank:    i + 1,
		}
		results = append(results, result)
	}

	return results, nil
}

// GetName returns the name of the search provider
func (gcsp *GoogleCustomSearchProvider) GetName() string {
	return "Google Custom Search"
}

// Helper function to get minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
