package parser

import (
	"briefly/internal/core"
	"bufio"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// URL regex patterns
var (
	// Matches markdown links: [text](url)
	markdownLinkRegex = regexp.MustCompile(`\[([^\]]+)\]\((https?://[^\)]+)\)`)

	// Matches raw URLs in text
	rawURLRegex = regexp.MustCompile(`https?://[^\s)]+`)
)

// Parser handles URL extraction and validation from markdown files
type Parser struct {
	// Configuration options could be added here in the future
}

// NewParser creates a new Parser instance
func NewParser() *Parser {
	return &Parser{}
}

// ParseMarkdownFile reads a markdown file and extracts all URLs
// Returns a list of core.Link objects with deduplicated URLs
func (p *Parser) ParseMarkdownFile(filePath string) ([]core.Link, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	urls := p.ParseMarkdownContent(string(content))
	links := make([]core.Link, 0, len(urls))

	for _, u := range urls {
		links = append(links, core.Link{
			ID:        uuid.NewString(),
			URL:       u,
			DateAdded: time.Now().UTC(),
			Source:    "file:" + filePath,
		})
	}

	return links, nil
}

// ParseMarkdownContent extracts URLs from markdown content string
// Handles both markdown links [text](url) and raw URLs
// Returns deduplicated list of URLs in document order
func (p *Parser) ParseMarkdownContent(content string) []string {
	urlMap := make(map[string]bool)
	var urls []string

	// Process line by line to maintain document order
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()

		// First, check for markdown links [text](url)
		markdownMatches := markdownLinkRegex.FindAllStringSubmatch(line, -1)
		if len(markdownMatches) > 0 {
			// Extract markdown link URLs from this line
			for _, match := range markdownMatches {
				if len(match) >= 3 {
					url := match[2] // URL is the second capture group
					if p.isValidURL(url) {
						normalized := p.NormalizeURL(url)
						if !urlMap[normalized] {
							urlMap[normalized] = true
							urls = append(urls, normalized)
						}
					}
				}
			}
		} else {
			// If no markdown links, look for raw URLs
			rawMatches := rawURLRegex.FindAllString(line, -1)
			for _, rawURL := range rawMatches {
				if p.isValidURL(rawURL) {
					normalized := p.NormalizeURL(rawURL)
					if !urlMap[normalized] {
						urlMap[normalized] = true
						urls = append(urls, normalized)
					}
				}
			}
		}
	}

	return urls
}

// ValidateURL checks if a URL is valid and accessible
func (p *Parser) ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("empty URL")
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s (must be http or https)", parsed.Scheme)
	}

	if parsed.Host == "" {
		return fmt.Errorf("URL missing host")
	}

	return nil
}

// NormalizeURL removes tracking parameters and normalizes URL format
func (p *Parser) NormalizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL // Return original if parsing fails
	}

	// Remove common tracking parameters
	query := parsed.Query()
	trackingParams := []string{
		"utm_source", "utm_medium", "utm_campaign", "utm_term", "utm_content",
		"fbclid", "gclid", "msclkid",
		"ref", "source",
	}

	for _, param := range trackingParams {
		query.Del(param)
	}

	parsed.RawQuery = query.Encode()

	// Remove fragment (anchor) as it doesn't affect content
	parsed.Fragment = ""

	// Normalize trailing slash for consistency
	if parsed.Path != "" && parsed.Path != "/" {
		parsed.Path = strings.TrimSuffix(parsed.Path, "/")
	}

	return parsed.String()
}

// DeduplicateURLs removes duplicate URLs from a list
func (p *Parser) DeduplicateURLs(urls []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(urls))

	for _, url := range urls {
		normalized := p.NormalizeURL(url)
		if !seen[normalized] {
			seen[normalized] = true
			result = append(result, normalized)
		}
	}

	return result
}

// isValidURL performs basic validation without returning errors
func (p *Parser) isValidURL(rawURL string) bool {
	return p.ValidateURL(rawURL) == nil
}

// ParseFile is a convenience method that wraps ParseMarkdownFile
// Kept for backward compatibility with existing code
func ParseFile(filePath string) ([]core.Link, error) {
	parser := NewParser()
	return parser.ParseMarkdownFile(filePath)
}