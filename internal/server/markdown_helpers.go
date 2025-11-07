package server

import (
	"html/template"
	"regexp"

	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// renderMarkdown converts markdown text to HTML, returning it as template.HTML for safe rendering.
// Configures parser with common extensions and HTML options for external links.
func renderMarkdown(text string) template.HTML {
	if text == "" {
		return template.HTML("")
	}

	// Configure markdown parser with common extensions
	extensions := parser.CommonExtensions | parser.AutoHeadingIDs
	mdParser := parser.NewWithExtensions(extensions)

	// Configure HTML renderer with external link handling
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	renderer := html.NewRenderer(html.RendererOptions{
		Flags: htmlFlags,
	})

	// Convert markdown to HTML
	htmlBytes := markdown.ToHTML([]byte(text), mdParser, renderer)

	return template.HTML(htmlBytes)
}

// convertCitationLinksToAnchors converts citation markers like [N] to clickable anchor links
// This makes citations clickable and jump to the article in the same page
func convertCitationLinksToAnchors(text string) string {
	// Pattern 1: [[N]](url) -> [[N]](#article-N) (double bracket format)
	// Matches: [[1]](https://example.com) or [[2]](http://...)
	pattern1 := regexp.MustCompile(`\[\[(\d+)\]\]\([^)]+\)`)
	text = pattern1.ReplaceAllString(text, `[[$1]](#article-$1)`)

	// Pattern 2: [N] -> [N](#article-N) (single bracket format - most common)
	// Matches: [1], [2], [3], etc. but NOT [1](url) markdown links
	// Match [number] followed by space, punctuation, or end of string (not opening paren)
	pattern2 := regexp.MustCompile(`\[(\d+)\]([\s.,;:!?\-\n]|$)`)
	text = pattern2.ReplaceAllString(text, `[$1](#article-$1)$2`)

	return text
}

// renderMarkdownWithCitations renders markdown and converts citation links to anchors
func renderMarkdownWithCitations(text string) template.HTML {
	// First convert citation URLs to anchors
	textWithAnchors := convertCitationLinksToAnchors(text)
	// Then render as normal markdown
	return renderMarkdown(textWithAnchors)
}
