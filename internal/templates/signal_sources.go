package templates

import (
	"fmt"
	"strings"
	"time"

	"briefly/internal/core"
)

// RenderSignalSources renders the new v3.0 Signal + Sources format
func RenderSignalSources(digest *core.Digest, maxWords int) (string, error) {
	if digest == nil {
		return "", fmt.Errorf("digest cannot be nil")
	}

	var content strings.Builder

	// Header with title and metadata
	title := digest.Metadata.Title
	if title == "" && digest.Signal.Content != "" {
		title = generateSignalTitle(digest.Signal.Content)
	}
	if title == "" {
		title = fmt.Sprintf("Signal Digest - %s", time.Now().Format("Jan 2, 2006"))
	}

	content.WriteString(fmt.Sprintf("# %s\n\n", title))

	// Word count and reading time (if available)
	if digest.Metadata.WordCount > 0 {
		readTime := estimateReadingTime(digest.Metadata.WordCount)
		content.WriteString(fmt.Sprintf("📊 %d words • ⏱️ %dm read\n\n", digest.Metadata.WordCount, readTime))
	}

	// The Signal section - main insight (60-80 words max)
	if digest.Signal.Content != "" {
		content.WriteString("## 🎯 The Signal\n\n")
		signalContent := digest.Signal.Content
		if maxWords > 0 {
			signalContent = truncateToWordLimitSignal(signalContent, 80) // Max 80 words for signal
		}
		content.WriteString(fmt.Sprintf("%s\n\n", signalContent))

		// Add implications if available
		if len(digest.Signal.Implications) > 0 {
			content.WriteString(fmt.Sprintf("**What it means:** %s\n\n", digest.Signal.Implications[0]))
		}

		content.WriteString("---\n\n")
	}

	// Article Groups - organized sources
	if len(digest.ArticleGroups) > 0 {
		content.WriteString(renderArticleGroups(digest.ArticleGroups))
	}

	// Action Items section
	if len(digest.Signal.Actions) > 0 {
		content.WriteString(renderActionItems(digest.Signal.Actions))
	}

	// Simple engagement question
	content.WriteString("## 💭 Your Take?\n\n")
	content.WriteString(generateEngagementQuestion(digest.ArticleGroups))
	content.WriteString("\n\n")

	// Enforce word limit if specified
	result := content.String()
	if maxWords > 0 {
		result = enforceWordLimit(result, maxWords)
	}

	return result, nil
}

// renderArticleGroups renders the Sources section with grouped articles
func renderArticleGroups(groups []core.ArticleGroup) string {
	var content strings.Builder

	for _, group := range groups {
		if len(group.Articles) == 0 {
			continue
		}

		// Group header
		content.WriteString(fmt.Sprintf("## %s\n\n", group.Category))

		// Group summary (if available and meaningful)
		if group.Summary != "" && group.Summary != fmt.Sprintf("%d articles covering related topics", len(group.Articles)) {
			content.WriteString(fmt.Sprintf("*%s*\n\n", group.Summary))
		}

		// Articles in this group
		for _, article := range group.Articles {
			content.WriteString(renderArticleEntry(article))
		}

		content.WriteString("\n")
	}

	return content.String()
}

// renderArticleEntry renders a single article entry
func renderArticleEntry(article core.Article) string {
	var entry strings.Builder

	// Article title with quality indicator
	qualityIndicator := getQualityIndicator(article.QualityScore)
	entry.WriteString(fmt.Sprintf("**%s %s**\n\n", qualityIndicator, article.Title))

	// Brief summary from existing data or generate basic one
	summary := generateBasicSummary(article)
	entry.WriteString(fmt.Sprintf("%s\n\n", summary))

	// Link with clean formatting
	entry.WriteString(fmt.Sprintf("🔗 [Read more](%s)\n\n", article.URL))

	return entry.String()
}

// renderActionItems renders the action items section
func renderActionItems(actions []core.ActionItem) string {
	if len(actions) == 0 {
		return ""
	}

	var content strings.Builder
	content.WriteString("## ⚡ Try This Week\n\n")

	for i, action := range actions {
		if i >= 3 { // Max 3 actions to keep focused
			break
		}

		effortIcon := getEffortIcon(action.Effort)
		content.WriteString(fmt.Sprintf("- %s %s\n", effortIcon, action.Description))
	}

	content.WriteString("\n")
	return content.String()
}

// Helper functions

func generateSignalTitle(signalContent string) string {
	// Extract key phrases for title generation
	words := strings.Fields(signalContent)
	if len(words) > 5 {
		return strings.Join(words[:5], " ") + "..."
	}
	return strings.Join(words, " ")
}

func estimateReadingTime(wordCount int) int {
	// Average reading speed: 250 words per minute
	minutes := wordCount / 250
	if minutes == 0 {
		return 1
	}
	return minutes
}

func truncateToWordLimitSignal(text string, maxWords int) string {
	words := strings.Fields(text)
	if len(words) <= maxWords {
		return text
	}
	return strings.Join(words[:maxWords], " ") + "..."
}

func getQualityIndicator(score float64) string {
	if score >= 0.8 {
		return "🔥"
	} else if score >= 0.6 {
		return "⭐"
	}
	return "💡"
}

func generateBasicSummary(article core.Article) string {
	// Generate a basic summary from article content
	if len(article.CleanedText) > 200 {
		words := strings.Fields(article.CleanedText)
		if len(words) > 25 {
			return strings.Join(words[:25], " ") + "..."
		}
		return article.CleanedText[:200] + "..."
	}

	// Fallback to a generic summary
	return "Article covering developments in the field."
}

func getEffortIcon(effort string) string {
	switch effort {
	case "low":
		return "🟢"
	case "medium":
		return "🟡"
	case "high":
		return "🔴"
	default:
		return "⚪"
	}
}

func generateEngagementQuestion(groups []core.ArticleGroup) string {
	if len(groups) == 0 {
		return "What's your take on these developments?"
	}

	// Generate question based on first group
	firstGroup := groups[0]
	if strings.Contains(firstGroup.Category, "Breaking") {
		return "Which of these breaking developments will have the biggest impact?"
	} else if strings.Contains(firstGroup.Category, "Tools") {
		return "Have you tried any of these tools? What's been your experience?"
	} else if strings.Contains(firstGroup.Category, "Business") {
		return "How do you see these business trends affecting your industry?"
	}

	return "What patterns do you see emerging from these articles?"
}

func enforceWordLimit(content string, maxWords int) string {
	words := strings.Fields(content)
	if len(words) <= maxWords {
		return content
	}

	// Truncate at word boundary and add indication
	truncated := strings.Join(words[:maxWords], " ")

	// Try to end at a natural break (paragraph, sentence)
	if lastNewline := strings.LastIndex(truncated, "\n\n"); lastNewline > len(truncated)/2 {
		truncated = truncated[:lastNewline]
	} else if lastPeriod := strings.LastIndex(truncated, ". "); lastPeriod > len(truncated)/2 {
		truncated = truncated[:lastPeriod+1]
	}

	return truncated + "\n\n*[Content truncated to meet word limit]*"
}

// IsSignalSourcesFormat checks if a digest uses the new Signal+Sources format
func IsSignalSourcesFormat(digest *core.Digest) bool {
	// Check if digest has the new v3.0 structure
	return digest.Signal.Content != "" && len(digest.ArticleGroups) > 0
}

// ConvertLegacyToSignalSources converts legacy digest to Signal+Sources format
func ConvertLegacyToSignalSources(digest *core.Digest) error {
	if IsSignalSourcesFormat(digest) {
		return nil // Already in new format
	}

	// Convert legacy digest to new format
	if digest.Content != "" && digest.Signal.Content == "" {
		// Extract signal from legacy content (basic implementation)
		lines := strings.Split(digest.Content, "\n")
		var signalContent string

		for _, line := range lines {
			if strings.HasPrefix(line, "## Executive Summary") && len(lines) > 1 {
				// Take next few lines as signal content
				signalContent = extractExecutiveSummary(lines)
				break
			}
		}

		if signalContent == "" && len(digest.Content) > 100 {
			// Fallback: use first paragraph
			paragraphs := strings.Split(digest.Content, "\n\n")
			if len(paragraphs) > 0 {
				signalContent = paragraphs[0]
			}
		}

		// Create basic signal
		digest.Signal = core.Signal{
			ID:            fmt.Sprintf("signal_%d", time.Now().Unix()),
			Content:       signalContent,
			Theme:         "Legacy Conversion",
			DateGenerated: time.Now(),
		}

		// Create basic article groups from legacy URL list
		if len(digest.ArticleURLs) > 0 {
			group := core.ArticleGroup{
				Category: "📖 Articles",
				Theme:    "Legacy Content",
				Summary:  fmt.Sprintf("%d articles from legacy digest", len(digest.ArticleURLs)),
				Priority: 1,
			}

			// Convert URLs to basic articles
			for i, url := range digest.ArticleURLs {
				article := core.Article{
					ID:           fmt.Sprintf("legacy_%d", i),
					URL:          url,
					Title:        fmt.Sprintf("Article %d", i+1),
					QualityScore: 0.7, // Assume decent quality for legacy
				}
				group.Articles = append(group.Articles, article)
			}

			digest.ArticleGroups = []core.ArticleGroup{group}
		}
	}

	return nil
}

func extractExecutiveSummary(lines []string) string {
	inSummary := false
	var summaryLines []string

	for _, line := range lines {
		if strings.HasPrefix(line, "## Executive Summary") {
			inSummary = true
			continue
		}
		if inSummary && strings.HasPrefix(line, "## ") {
			break // End of summary section
		}
		if inSummary && strings.TrimSpace(line) != "" {
			summaryLines = append(summaryLines, strings.TrimSpace(line))
		}
	}

	if len(summaryLines) > 0 {
		return strings.Join(summaryLines, " ")
	}
	return ""
}
