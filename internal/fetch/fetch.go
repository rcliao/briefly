package fetch

import (
	"bufio"
	"briefly/internal/core"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery" // Added for HTML parsing
	"github.com/google/uuid"
)

// urlRegex is a simple regex to find URLs.
// This can be refined to be more comprehensive if needed.
var urlRegex = regexp.MustCompile(`https?://[^\s)]+`)

// ReadLinksFromFile reads a list of URLs from a text file.
// It expects URLs to be on lines, potentially prefixed (e.g., in a markdown list).
func ReadLinksFromFile(filePath string) ([]core.Link, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open link file %s: %w", filePath, err)
	}
	defer file.Close()

	var links []core.Link
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Attempt to find a URL in the line
		foundURLs := urlRegex.FindAllString(line, -1)

		for _, textURL := range foundURLs {
			// Trim potential markdown list characters or other noise if necessary,
			// though the regex should capture a clean URL.
			// For example, if lines are like "- https://example.com", textURL should be "https://example.com"

			// Validate URL
			parsedURL, err := url.ParseRequestURI(textURL)
			if err != nil {
				fmt.Printf("Skipping invalid URL on line %d: %s (%s)\\n", lineNumber, textURL, err)
				continue
			}

			// Further check: ensure scheme is http or https
			if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
				fmt.Printf("Skipping URL with non-http(s) scheme on line %d: %s\\n", lineNumber, textURL)
				continue
			}

			// Check if this link (URL) has already been added to avoid duplicates from the same file
			alreadyAdded := false
			for _, l := range links {
				if l.URL == textURL {
					alreadyAdded = true
					break
				}
			}
			if alreadyAdded {
				fmt.Printf("Skipping duplicate URL from file: %s\\n", textURL)
				continue
			}

			links = append(links, core.Link{
				ID:        uuid.NewString(),
				URL:       textURL,
				DateAdded: time.Now().UTC(), // Use UTC for consistency
				Source:    "file:" + filePath, // More specific source
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading link file %s: %w", filePath, err)
	}

	return links, nil
}

// FetchArticle fetches the content from a given core.Link and returns a core.Article.
// It currently only fetches the raw HTML content.
func FetchArticle(link core.Link) (core.Article, error) {
	resp, err := http.Get(link.URL)
	if err != nil {
		return core.Article{}, fmt.Errorf("failed to fetch URL %s: %w", link.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return core.Article{}, fmt.Errorf("failed to fetch URL %s: status code %d", link.URL, resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return core.Article{}, fmt.Errorf("failed to read response body from %s: %w", link.URL, err)
	}

	article := core.Article{
		ID:          uuid.NewString(),
		LinkID:      link.ID,
		FetchedHTML: string(bodyBytes),
		DateFetched: time.Now().UTC(),
		Title:       extractTitle(string(bodyBytes), link.URL), // Extract title
		// CleanedText will be populated by a subsequent parsing step
	}

	return article, nil
}

// extractTitle tries to extract the title from HTML content.
func extractTitle(htmlContent string, sourceURL string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		fmt.Printf("Error creating goquery document for title extraction from %s: %v\\n", sourceURL, err)
		return ""
	}

	// Try common title tags
	title := doc.Find("head title").First().Text()
	if title != "" {
		return strings.TrimSpace(title)
	}

	// Fallback to OpenGraph title
	ogTitle, _ := doc.Find("meta[property='og:title']").Attr("content")
	if ogTitle != "" {
		return strings.TrimSpace(ogTitle)
	}
	
	// Fallback to h1
	h1Title := doc.Find("h1").First().Text()
	if h1Title != "" {
		return strings.TrimSpace(h1Title)
	}

	// Further fallbacks can be added if needed
	return "" // Return empty if no title found
}

// ParseArticleContent extracts the main textual content from HTML and removes boilerplate.
// It updates the CleanedText and potentially Title field of the provided article.
func ParseArticleContent(article *core.Article) error {
	if article.FetchedHTML == "" {
		return fmt.Errorf("article ID %s has no FetchedHTML to parse", article.ID)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(article.FetchedHTML))
	if err != nil {
		return fmt.Errorf("failed to create goquery document for article %s: %w", article.ID, err)
	}

	// Remove common non-content elements
	// This list is similar to the one in main.go, can be expanded.
	doc.Find("script, style, nav, footer, header, aside, form, iframe, noscript, .sidebar, #sidebar, .ad, .advertisement, .popup, .modal, .cookie-banner").Remove()
	
	// Attempt to find main content using common selectors (inspired by main.go)
	var textBuilder strings.Builder
	mainContentSelectors := []string{
		"article", "main", ".main-content", ".entry-content", ".post-content", ".post-body", ".article-body", // Common semantic tags and classes
		"[role='main']",                                                                                      // ARIA role
		".content", "#content", // Generic content containers
		// Add more specific selectors if common patterns are observed in target sites
	}

	foundMainContent := false
	for _, selector := range mainContentSelectors {
		doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			// Extract text and preserve paragraph breaks better by adding newlines after block elements
			s.Find("p, h1, h2, h3, h4, h5, h6, li, blockquote, pre, div").Each(func(_ int, item *goquery.Selection) {
				textBuilder.WriteString(strings.TrimSpace(item.Text()))
				textBuilder.WriteString("\\n\\n") // Add double newline to simulate paragraph breaks
			})
		})
		if textBuilder.Len() > 0 {
			foundMainContent = true
			break 
		}
	}

	// If no specific main content found, get all text from the body, then try to clean it
	if !foundMainContent {
		doc.Find("body").Find("p, h1, h2, h3, h4, h5, h6, li, blockquote, pre, div").Each(func(_ int, item *goquery.Selection) {
			textBuilder.WriteString(strings.TrimSpace(item.Text()))
			textBuilder.WriteString("\\n\\n")
		})
	}
	
	fullText := textBuilder.String()

	// Basic cleaning:
	// 1. Replace multiple newlines with a single newline.
	// 2. Trim leading/trailing whitespace from the result.
	// This is a simplified version of the cleaning in main.go's extractTextFromHTML
	newlineRegex := regexp.MustCompile(`(\\n\\s*){2,}`)
	cleanedText := newlineRegex.ReplaceAllString(fullText, "\\n")
	cleanedText = strings.TrimSpace(cleanedText)

	article.CleanedText = cleanedText

	// If title was not extracted during fetch, try again from parsed doc
	if article.Title == "" {
		article.Title = extractTitle(article.FetchedHTML, article.LinkID) // LinkID used as a stand-in for URL here
	}
	if article.Title == "" && len(cleanedText) > 0 { // Fallback title from first few words of content
		words := strings.Fields(cleanedText)
		if len(words) > 10 {
			article.Title = strings.Join(words[:10], " ") + "..."
		} else {
			article.Title = strings.Join(words, " ")
		}
	}


	if strings.TrimSpace(article.CleanedText) == "" {
		// It's not necessarily an error if no text is extracted, could be a non-article page.
		// Consider logging this as a warning if desired.
		fmt.Printf("Warning: No text extracted from article with LinkID %s after cleaning\\n", article.LinkID)
	}

	return nil
}

// CleanArticleHTML is a wrapper around ParseArticleContent for consistency with the digest command
func CleanArticleHTML(article *core.Article) error {
	return ParseArticleContent(article)
}

// TODO: Add functions for cleaning HTML, extracting title, etc.
