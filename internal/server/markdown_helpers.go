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

// convertCitationLinksToAnchors converts citation links like [[N]](url) to anchor links [[N]](#article-N)
// This makes citations clickable and jump to the article in the same page
func convertCitationLinksToAnchors(text string) string {
	// Pattern: [[N]](url) -> [[N]](#article-N)
	// Matches: [[1]](https://example.com) or [[2]](http://...)
	pattern := regexp.MustCompile(`\[\[(\d+)\]\]\([^)]+\)`)
	return pattern.ReplaceAllString(text, `[[$1]](#article-$1)`)
}

// renderMarkdownWithCitations renders markdown and converts citation links to anchors
func renderMarkdownWithCitations(text string) template.HTML {
	// First convert citation URLs to anchors
	textWithAnchors := convertCitationLinksToAnchors(text)
	// Then render as normal markdown
	return renderMarkdown(textWithAnchors)
}
