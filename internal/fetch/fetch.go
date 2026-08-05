package fetch

import (
	"briefly/internal/core"
	"bufio"
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

// urlRegex finds HTTP/HTTPS URLs
var urlRegex = regexp.MustCompile(`https?://[^\s)]+`)

// filePathRegex finds file paths (including file:// URLs and relative paths) - currently unused but kept for potential future use
//var filePathRegex = regexp.MustCompile(`(?:file://)?(?:[./])?[^\s]+\.(?:pdf|html|htm)`)

// allContentRegex finds all supported content (URLs and file paths)
var allContentRegex = regexp.MustCompile(`(?:https?://[^\s)]+|(?:file://)?(?:[./])?[^\s]+\.(?:pdf|html|htm))`)

// ReadLinksFromFile reads a list of URLs from a text file.
// It expects URLs to be on lines, potentially prefixed (e.g., in a markdown list).
func ReadLinksFromFile(filePath string) ([]core.Link, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open link file %s: %w", filePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close file: %s\n", err)
		}
	}()

	var links []core.Link
	scanner := bufio.NewScanner(file)
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		// Attempt to find URLs and file paths in the line
		foundContent := allContentRegex.FindAllString(line, -1)

		for _, content := range foundContent {
			var isValid bool
			var contentURL string

			// Check if it's a URL or file path
			if strings.HasPrefix(content, "http://") || strings.HasPrefix(content, "https://") {
				// Validate HTTP/HTTPS URL
				parsedURL, err := url.ParseRequestURI(content)
				if err != nil {
					fmt.Printf("Skipping invalid URL on line %d: %s (%s)\\n", lineNumber, content, err)
					continue
				}
				if parsedURL.Scheme == "http" || parsedURL.Scheme == "https" {
					isValid = true
					contentURL = content
				}
			} else {
				// Handle file paths and file:// URLs
				if strings.HasPrefix(content, "file://") {
					contentURL = content
					isValid = true
				} else {
					// Relative or absolute file path
					contentURL = content
					isValid = true
				}
			}

			if !isValid {
				continue
			}

			// Check if this content has already been added to avoid duplicates from the same file
			alreadyAdded := false
			for _, l := range links {
				if l.URL == contentURL {
					alreadyAdded = true
					break
				}
			}
			if alreadyAdded {
				fmt.Printf("Skipping duplicate content from file: %s\\n", contentURL)
				continue
			}

			links = append(links, core.Link{
				ID:        uuid.NewString(),
				URL:       contentURL,
				DateAdded: time.Now().UTC(),   // Use UTC for consistency
				Source:    "file:" + filePath, // More specific source
			})
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading link file %s: %w", filePath, err)
	}

	return links, nil
}

// browserUserAgent mimics a real browser; hosts like Substack block Go's default UA.
const browserUserAgent = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36"

// maxFetchBytes caps response bodies to avoid pathological pages.
const maxFetchBytes = 10 << 20 // 10 MB

// FetchArticle fetches the content from a given core.Link and returns a core.Article.
// It currently only fetches the raw HTML content.
func FetchArticle(link core.Link) (core.Article, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	fetchOnce := func() (*http.Response, error) {
		req, err := http.NewRequest(http.MethodGet, link.URL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", browserUserAgent)
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			_ = resp.Body.Close()
			return nil, fmt.Errorf("status code %d", resp.StatusCode)
		}
		return resp, nil
	}

	resp, err := fetchOnce()
	if err != nil {
		// One retry: transient errors and rate limits are common on first hit
		time.Sleep(2 * time.Second)
		resp, err = fetchOnce()
	}
	if err != nil {
		return core.Article{}, fmt.Errorf("failed to fetch URL %s: %w", link.URL, err)
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxFetchBytes))
	if err != nil {
		return core.Article{}, fmt.Errorf("failed to read response body from %s: %w", link.URL, err)
	}

	article := core.Article{
		ID:          uuid.NewString(),
		URL:         link.URL, // Set the URL field
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
	doc.Find("script, style, nav, footer, header, aside, form, iframe, noscript, svg, button, .sidebar, #sidebar, .ad, .advertisement, .popup, .modal, .cookie-banner, .comments, #comments").Remove()

	// Attempt to find the main content container. Pages can contain several
	// matches (comment cards, related-post teasers), so take the largest one.
	mainContentSelectors := []string{
		"article", "main", ".main-content", ".entry-content", ".post-content", ".post-body", ".article-body",
		"[role='main']",
		".content", "#content",
	}

	var container *goquery.Selection
	for _, selector := range mainContentSelectors {
		var best *goquery.Selection
		bestLen := 0
		doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
			if l := len(strings.TrimSpace(s.Text())); l > bestLen {
				best, bestLen = s, l
			}
		})
		// Require some substance so a match on an empty wrapper doesn't win
		if best != nil && bestLen > 200 {
			container = best
			break
		}
	}
	if container == nil {
		container = doc.Find("body")
	}

	// Extract text from leaf block elements only. Selecting "div" here caused
	// massive duplication (every nested div re-emits all descendant text).
	var textBuilder strings.Builder
	seen := make(map[string]bool)
	container.Find("p, h1, h2, h3, h4, h5, h6, li, blockquote, pre, td").Each(func(_ int, item *goquery.Selection) {
		// Skip elements that contain other block elements (e.g. li wrapping a nested list)
		if item.Find("p, li, blockquote, pre").Length() > 0 {
			return
		}
		text := strings.TrimSpace(item.Text())
		if text == "" || seen[text] {
			return
		}
		seen[text] = true
		textBuilder.WriteString(text)
		textBuilder.WriteString("\n\n")
	})

	// Fallback: container had no block elements at all — take its raw text
	if textBuilder.Len() == 0 {
		textBuilder.WriteString(strings.TrimSpace(container.Text()))
	}

	// Collapse runs of blank lines and normalize whitespace
	newlineRegex := regexp.MustCompile(`\n{3,}`)
	cleanedText := newlineRegex.ReplaceAllString(textBuilder.String(), "\n\n")
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
