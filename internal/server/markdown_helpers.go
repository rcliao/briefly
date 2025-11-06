package server

import (
	"html/template"

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
