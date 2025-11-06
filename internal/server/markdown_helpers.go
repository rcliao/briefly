package server

import (
	"html/template"
	"regexp"
	"strings"

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

// extractKeyMoments parses a digest summary to extract key moments/highlights.
// Handles multiple formats:
// - Numbered lists: "1. **Actor → verb** description"
// - Bullet points: "- text" or "* text"
// - Bold lines: "**text**"
// Returns up to 5 key moments found in the summary.
func extractKeyMoments(summary string) []string {
	if summary == "" {
		return []string{}
	}

	var keyMoments []string

	// Split summary into lines
	lines := strings.Split(summary, "\n")

	// Regex patterns for key moments
	// Pattern for numbered lists with bold: "1. **Actor → verb** description [See #X]"
	numberedBoldPattern := regexp.MustCompile(`^\d+\.\s+\*\*(.+?)\*\*\s*(.+?)(?:\s*\[See #\d+\])?$`)
	// Pattern for standalone bold: "**Bold text**"
	boldPattern := regexp.MustCompile(`^\*\*(.+?)\*\*`)
	// Pattern for bullet points: "- text" or "* text"
	bulletPattern := regexp.MustCompile(`^[\-\*]\s+(.+)$`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check for numbered list with bold (e.g., "1. **Anthropic → releases** Claude Code...")
		if matches := numberedBoldPattern.FindStringSubmatch(line); len(matches) > 2 {
			// Combine bold part and description
			keyMoment := matches[1] + " " + matches[2]
			keyMoments = append(keyMoments, keyMoment)
			continue
		}

		// Check for standalone bold pattern
		if matches := boldPattern.FindStringSubmatch(line); len(matches) > 1 {
			keyMoments = append(keyMoments, matches[1])
			continue
		}

		// Check for bullet pattern
		if matches := bulletPattern.FindStringSubmatch(line); len(matches) > 1 {
			keyMoments = append(keyMoments, matches[1])
			continue
		}
	}

	// Limit to 5 key moments
	if len(keyMoments) > 5 {
		keyMoments = keyMoments[:5]
	}

	return keyMoments
}

// truncateSummary truncates a summary to a specified character length for preview.
// Adds "..." if truncated. Used for digest list preview text.
func truncateSummary(text string, maxChars int) string {
	if len(text) <= maxChars {
		return text
	}

	// Find last space before maxChars to avoid cutting words
	truncated := text[:maxChars]
	lastSpace := strings.LastIndex(truncated, " ")
	if lastSpace > 0 {
		truncated = truncated[:lastSpace]
	}

	return truncated + "..."
}
