// Package markdown provides utilities for parsing markdown citations
package markdown

import (
	"briefly/internal/core"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

// CitationReference represents a citation extracted from markdown
type CitationReference struct {
	Number  int    // Citation number [1], [2], [3]
	URL     string // Target URL
	Context string // Surrounding text (for context)
}

// ExtractCitations parses markdown text and extracts all citations
// Supports formats: [[N]](url) and [N](url)
func ExtractCitations(markdown string) []CitationReference {
	var citations []CitationReference

	// Pattern 1: [[N]](url) - double bracket format
	pattern1 := regexp.MustCompile(`\[\[(\d+)\]\]\(([^)]+)\)`)
	matches1 := pattern1.FindAllStringSubmatch(markdown, -1)
	for _, match := range matches1 {
		if len(match) >= 3 {
			var num int
			_, _ = fmt.Sscanf(match[1], "%d", &num)
			citations = append(citations, CitationReference{
				Number:  num,
				URL:     strings.TrimSpace(match[2]),
				Context: extractContext(markdown, match[0]),
			})
		}
	}

	// Pattern 2: [N](url) - single bracket format
	pattern2 := regexp.MustCompile(`\[(\d+)\]\(([^)]+)\)`)
	matches2 := pattern2.FindAllStringSubmatch(markdown, -1)
	for _, match := range matches2 {
		if len(match) >= 3 {
			var num int
			_, _ = fmt.Sscanf(match[1], "%d", &num)

			// Check if this citation was already found with double brackets
			alreadyExists := false
			for _, existing := range citations {
				if existing.Number == num && existing.URL == strings.TrimSpace(match[2]) {
					alreadyExists = true
					break
				}
			}

			if !alreadyExists {
				citations = append(citations, CitationReference{
					Number:  num,
					URL:     strings.TrimSpace(match[2]),
					Context: extractContext(markdown, match[0]),
				})
			}
		}
	}

	return citations
}

// extractContext extracts surrounding text around a citation for context
// Returns up to 100 characters before and after the citation
func extractContext(text string, citation string) string {
	index := strings.Index(text, citation)
	if index == -1 {
		return ""
	}

	start := index - 100
	if start < 0 {
		start = 0
	}

	end := index + len(citation) + 100
	if end > len(text) {
		end = len(text)
	}

	context := text[start:end]
	context = strings.TrimSpace(context)

	// Truncate at sentence boundaries if possible
	if start > 0 {
		if idx := strings.Index(context, ". "); idx > 0 && idx < 50 {
			context = context[idx+2:]
		}
	}

	if end < len(text) {
		if idx := strings.LastIndex(context, ". "); idx > len(context)-50 && idx > 0 {
			context = context[:idx+1]
		}
	}

	return context
}

// BuildCitationRecords converts citation references to core.Citation structs
// This is used to store citations in the database
func BuildCitationRecords(
	digestID string,
	citations []CitationReference,
	articleMap map[string]*core.Article, // Map of URL -> Article
) []core.Citation {
	records := make([]core.Citation, 0, len(citations))

	for _, ref := range citations {
		// Try to find the corresponding article
		article, found := articleMap[ref.URL]
		if !found {
			// Citation URL doesn't match any article - skip or log warning
			continue
		}

		citation := core.Citation{
			ID:             uuid.New().String(),
			ArticleID:      article.ID,
			URL:            ref.URL,
			Title:          article.Title,
			Publisher:      article.Publisher,
			AccessedDate:   time.Now().UTC(),
			CreatedAt:      time.Now().UTC(),
			DigestID:       &digestID,
			CitationNumber: &ref.Number,
			Context:        ref.Context,
		}

		// Set published date if available
		if !article.DatePublished.IsZero() {
			citation.PublishedDate = &article.DatePublished
		}

		records = append(records, citation)
	}

	return records
}

// InjectCitationURLs replaces citation placeholders with actual URLs
// Converts [[N]] -> [[N]](url) using the provided article map
// Only replaces citations that don't already have URLs
func InjectCitationURLs(markdown string, articles []core.Article) string {
	// Build map of citation number -> URL
	citationMap := make(map[int]string)
	for i, article := range articles {
		citationMap[i+1] = article.URL
	}

	result := markdown
	// Process each citation number
	for num, url := range citationMap {
		placeholder := fmt.Sprintf("[[%d]]", num)
		withURL := fmt.Sprintf("[[%d]](%s)", num, url)

		// Only replace if the citation exists as placeholder (no URL yet)
		// and doesn't already have the URL
		if strings.Contains(result, placeholder) && !strings.Contains(result, withURL) {
			// Make sure we're not replacing citations that already have a URL
			// by checking they're not followed by '('
			result = strings.ReplaceAll(result, placeholder+" ", withURL+" ")
			result = strings.ReplaceAll(result, placeholder+",", withURL+",")
			result = strings.ReplaceAll(result, placeholder+".", withURL+".")
			result = strings.ReplaceAll(result, placeholder+"\n", withURL+"\n")

			// Handle end of string
			if strings.HasSuffix(result, placeholder) {
				result = result[:len(result)-len(placeholder)] + withURL
			}
		}
	}

	return result
}

// ValidateCitations checks if all citations in markdown have corresponding articles
func ValidateCitations(markdown string, articles []core.Article) []string {
	citations := ExtractCitations(markdown)
	articleURLs := make(map[string]bool)

	for _, article := range articles {
		articleURLs[article.URL] = true
	}

	var warnings []string
	for _, citation := range citations {
		if !articleURLs[citation.URL] {
			warnings = append(warnings,
				fmt.Sprintf("Citation [%d] references unknown URL: %s", citation.Number, citation.URL))
		}
	}

	return warnings
}

// CountCitations returns the number of citations in markdown text
func CountCitations(markdown string) int {
	return len(ExtractCitations(markdown))
}

// FormatCitationNumber formats a citation number for display
// Example: 1 -> "[1]"
func FormatCitationNumber(num int) string {
	return fmt.Sprintf("[%d]", num)
}

// ParseCitationMarkdown extracts just the citation numbers from text
// Used for parsing perspectives that reference multiple citations
// Example: "Supporting evidence from [[1]], [[2]], and [[3]]" -> [1, 2, 3]
func ParseCitationNumbers(text string) []int {
	pattern := regexp.MustCompile(`\[?\[?(\d+)\]?\]?`)
	matches := pattern.FindAllStringSubmatch(text, -1)

	numbers := make([]int, 0, len(matches))
	seen := make(map[int]bool)

	for _, match := range matches {
		if len(match) >= 2 {
			var num int
			_, _ = fmt.Sscanf(match[1], "%d", &num)
			if !seen[num] && num > 0 {
				numbers = append(numbers, num)
				seen[num] = true
			}
		}
	}

	return numbers
}
