package search

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"briefly/internal/logger"
)

// DuckDuckGoProvider implements the Provider interface using DuckDuckGo
type DuckDuckGoProvider struct {
	client    *http.Client
	userAgent string
	rateLimit time.Duration
	lastCall  time.Time
}

// NewDuckDuckGoProvider creates a new DuckDuckGo search provider
func NewDuckDuckGoProvider() *DuckDuckGoProvider {
	return &DuckDuckGoProvider{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		userAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
		rateLimit: 2 * time.Second, // Be respectful with rate limiting
	}
}

// GetName returns the name of this provider
func (d *DuckDuckGoProvider) GetName() string {
	return "DuckDuckGo"
}

// Search performs a search using DuckDuckGo and returns results
func (d *DuckDuckGoProvider) Search(ctx context.Context, query string, config Config) ([]Result, error) {
	// Respect rate limiting
	if elapsed := time.Since(d.lastCall); elapsed < d.rateLimit {
		time.Sleep(d.rateLimit - elapsed)
	}
	d.lastCall = time.Now()

	// Build search URL
	searchURL := d.buildSearchURL(query, config)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", d.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("DNT", "1")

	// Execute request
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search request failed with status: %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Debug: Save response body to inspect HTML structure
	bodyStr := string(body)
	if len(bodyStr) < 1000 {
		logger.Debug("DuckDuckGo response too short", "query", query, "response_length", len(bodyStr), "response_preview", bodyStr[:min(200, len(bodyStr))])
	} else {
		logger.Debug("DuckDuckGo response received", "query", query, "response_length", len(bodyStr))
	}

	// Check for CAPTCHA or blocking
	if strings.Contains(bodyStr, "captcha") || strings.Contains(bodyStr, "Captcha") || strings.Contains(bodyStr, "blocked") {
		logger.Debug("DuckDuckGo CAPTCHA detected", "query", query)
		return nil, fmt.Errorf("DuckDuckGo search blocked by CAPTCHA - try again later or use Google Custom Search")
	}

	// Parse results from HTML
	results := d.parseSearchResults(bodyStr, config.MaxResults)

	logger.Info("DuckDuckGo search completed", "query", query, "results_found", len(results))

	return results, nil
}

// buildSearchURL constructs the DuckDuckGo search URL with parameters
func (d *DuckDuckGoProvider) buildSearchURL(query string, config Config) string {
	baseURL := "https://html.duckduckgo.com/html/"
	params := url.Values{}

	// Add time filter if specified
	if config.SinceTime > 0 {
		days := int(config.SinceTime.Hours() / 24)
		switch {
		case days <= 1:
			params.Set("df", "d") // Past day
		case days <= 7:
			params.Set("df", "w") // Past week
		case days <= 30:
			params.Set("df", "m") // Past month
		case days <= 365:
			params.Set("df", "y") // Past year
		}
	}

	params.Set("q", query)
	params.Set("b", "0")      // Start from first result
	params.Set("kl", "us-en") // Language/region
	params.Set("s", "0")      // Safe search off

	return baseURL + "?" + params.Encode()
}

// parseSearchResults extracts search results from DuckDuckGo HTML response
func (d *DuckDuckGoProvider) parseSearchResults(html string, maxResults int) []Result {
	var results []Result

	// Regular expressions for parsing DuckDuckGo HTML results
	// Note: These patterns may need adjustment if DuckDuckGo changes their HTML structure
	resultPattern := regexp.MustCompile(`<div class="result[^"]*"[^>]*>(.*?)</div>`)
	titlePattern := regexp.MustCompile(`<a[^>]*class="result__a"[^>]*href="([^"]*)"[^>]*>(.*?)</a>`)
	snippetPattern := regexp.MustCompile(`<a[^>]*class="result__snippet"[^>]*>(.*?)</a>`)

	resultMatches := resultPattern.FindAllStringSubmatch(html, -1)

	for i, match := range resultMatches {
		if i >= maxResults {
			break
		}

		resultHTML := match[1]

		// Extract title and URL
		titleMatch := titlePattern.FindStringSubmatch(resultHTML)
		if len(titleMatch) < 3 {
			continue
		}

		rawURL := titleMatch[1]
		title := d.cleanHTMLText(titleMatch[2])

		// Extract snippet
		snippetMatch := snippetPattern.FindStringSubmatch(resultHTML)
		snippet := ""
		if len(snippetMatch) >= 2 {
			snippet = d.cleanHTMLText(snippetMatch[1])
		}

		// Decode URL (DuckDuckGo uses redirect URLs)
		finalURL := d.extractFinalURL(rawURL)
		if finalURL == "" {
			continue
		}

		// Extract domain
		domain := d.extractDomain(finalURL)

		result := Result{
			URL:     finalURL,
			Title:   title,
			Snippet: snippet,
			Domain:  domain,
			Source:  "DuckDuckGo",
			Rank:    i + 1,
		}

		results = append(results, result)
	}

	return results
}

// extractFinalURL extracts the actual URL from DuckDuckGo's redirect URL
func (d *DuckDuckGoProvider) extractFinalURL(redirectURL string) string {
	// DuckDuckGo uses URLs like: /l/?uddg=https%3A//example.com/...&rut=...
	if strings.HasPrefix(redirectURL, "/l/?") {
		parsed, err := url.Parse(redirectURL)
		if err != nil {
			return ""
		}

		uddg := parsed.Query().Get("uddg")
		if uddg != "" {
			decoded, err := url.QueryUnescape(uddg)
			if err != nil {
				return ""
			}
			return decoded
		}
	}

	// If it's already a full URL, return as-is
	if strings.HasPrefix(redirectURL, "http") {
		return redirectURL
	}

	return ""
}

// extractDomain extracts the domain name from a URL
func (d *DuckDuckGoProvider) extractDomain(urlStr string) string {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}

	domain := parsed.Hostname()
	// Remove www. prefix
	domain = strings.TrimPrefix(domain, "www.")

	return domain
}

// cleanHTMLText removes HTML tags and decodes HTML entities
func (d *DuckDuckGoProvider) cleanHTMLText(text string) string {
	// Remove HTML tags
	tagPattern := regexp.MustCompile(`<[^>]*>`)
	text = tagPattern.ReplaceAllString(text, "")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")
	text = strings.ReplaceAll(text, "&nbsp;", " ")

	// Clean up whitespace
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	return text
}
