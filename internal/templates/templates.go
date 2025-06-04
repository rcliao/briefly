package templates

import (
	"briefly/internal/llm"
	"briefly/internal/render"
	"fmt"
	"sort"
	"strings"
	"time"
)

// DigestFormat represents different digest formats
type DigestFormat string

const (
	// FormatBrief creates a concise digest with key highlights only
	FormatBrief DigestFormat = "brief"
	// FormatStandard creates a balanced digest with summaries and key points
	FormatStandard DigestFormat = "standard"
	// FormatDetailed creates comprehensive digest with full summaries and analysis
	FormatDetailed DigestFormat = "detailed"
	// FormatNewsletter creates a newsletter-style digest optimized for sharing
	FormatNewsletter DigestFormat = "newsletter"
)

// DigestTemplate holds template configuration for different formats
type DigestTemplate struct {
	Format                    DigestFormat
	Title                     string
	IncludeSummaries          bool
	IncludeKeyInsights        bool
	IncludeActionItems        bool
	IncludeSourceLinks        bool
	IncludePromptCorner       bool // For newsletter format to include AI prompts section
	IncludeIndividualArticles bool // Whether to include the "Individual Articles" section
	IncludeTopicClustering    bool // Whether to group articles by topic clusters
	MaxSummaryLength          int  // 0 for no limit
	IntroductionText          string
	ConclusionText            string
	SectionSeparator          string
}

// GetTemplate returns a pre-configured template for the specified format
func GetTemplate(format DigestFormat) *DigestTemplate {
	switch format {
	case FormatBrief:
		return &DigestTemplate{
			Format:                    FormatBrief,
			Title:                     "Brief Digest",
			IncludeSummaries:          true,
			IncludeKeyInsights:        false,
			IncludeActionItems:        false,
			IncludeSourceLinks:        true,
			IncludePromptCorner:       false,
			IncludeIndividualArticles: false,
			IncludeTopicClustering:    false, // Keep simple for brief format
			MaxSummaryLength:          150,
			IntroductionText:          "Quick highlights from today's reading:",
			ConclusionText:            "",
			SectionSeparator:          "\n\n---\n\n",
		}
	case FormatStandard:
		return &DigestTemplate{
			Format:                    FormatStandard,
			Title:                     "Daily Digest",
			IncludeSummaries:          true,
			IncludeKeyInsights:        true,
			IncludeActionItems:        false,
			IncludeSourceLinks:        true,
			IncludePromptCorner:       false,
			IncludeIndividualArticles: true, // Enable to showcase topic clustering
			IncludeTopicClustering:    true, // Enable topic clustering for better organization
			MaxSummaryLength:          300,
			IntroductionText:          "Here's what's worth knowing from today's articles:",
			ConclusionText:            "",
			SectionSeparator:          "\n\n---\n\n",
		}
	case FormatDetailed:
		return &DigestTemplate{
			Format:                    FormatDetailed,
			Title:                     "Comprehensive Digest",
			IncludeSummaries:          true,
			IncludeKeyInsights:        true,
			IncludeActionItems:        true,
			IncludeSourceLinks:        true,
			IncludePromptCorner:       false,
			IncludeIndividualArticles: true, // Enable to showcase topic clustering
			IncludeTopicClustering:    true, // Enable topic clustering for detailed analysis
			MaxSummaryLength:          0,    // No limit
			IntroductionText:          "In-depth analysis of today's key articles:",
			ConclusionText:            "These insights provide a comprehensive view of current developments in the field.",
			SectionSeparator:          "\n\n---\n\n",
		}
	case FormatNewsletter:
		return &DigestTemplate{
			Format:                    FormatNewsletter,
			Title:                     "Weekly Newsletter",
			IncludeSummaries:          true,
			IncludeKeyInsights:        true,
			IncludeActionItems:        true,
			IncludeSourceLinks:        true,
			IncludePromptCorner:       true, // Enable prompt corner for newsletter format
			IncludeIndividualArticles: false,
			IncludeTopicClustering:    true, // Enable topic clustering for newsletter organization
			MaxSummaryLength:          250,
			IntroductionText:          "Welcome to this week's curated selection of insights! Here's what caught our attention:",
			ConclusionText:            "Thank you for reading! Forward this to colleagues who might find it valuable.",
			SectionSeparator:          "\n\n💡 **Key Insight**\n\n",
		}
	default:
		return GetTemplate(FormatStandard)
	}
}

// TopicGroup represents a group of articles with the same topic cluster
type TopicGroup struct {
	TopicCluster  string
	Articles      []render.DigestData
	AvgConfidence float64
}

// GroupArticlesByTopic groups articles by their topic clusters and sorts them by confidence
func GroupArticlesByTopic(digestItems []render.DigestData) []TopicGroup {
	if len(digestItems) == 0 {
		return []TopicGroup{}
	}

	// Group articles by topic cluster
	topicMap := make(map[string][]render.DigestData)
	confidenceMap := make(map[string][]float64)

	for _, item := range digestItems {
		topicCluster := item.TopicCluster
		if topicCluster == "" {
			topicCluster = "General" // Default cluster for uncategorized items
		}

		topicMap[topicCluster] = append(topicMap[topicCluster], item)
		confidenceMap[topicCluster] = append(confidenceMap[topicCluster], item.TopicConfidence)
	}

	// Convert to TopicGroup slice and calculate average confidence
	var groups []TopicGroup
	for cluster, articles := range topicMap {
		// Calculate average confidence for this topic
		var totalConfidence float64
		for _, conf := range confidenceMap[cluster] {
			totalConfidence += conf
		}
		avgConfidence := totalConfidence / float64(len(articles))

		groups = append(groups, TopicGroup{
			TopicCluster:  cluster,
			Articles:      articles,
			AvgConfidence: avgConfidence,
		})
	}

	// Sort groups by average confidence (descending)
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].AvgConfidence > groups[j].AvgConfidence
	})

	return groups
}

// renderArticlesSection renders the articles section with optional topic clustering
func renderArticlesSection(digestItems []render.DigestData, template *DigestTemplate) string {
	var content strings.Builder

	if !template.IncludeIndividualArticles {
		return content.String()
	}

	content.WriteString("## Individual Articles\n\n")

	if template.IncludeTopicClustering {
		// Group articles by topic clusters
		topicGroups := GroupArticlesByTopic(digestItems)

		for groupIdx, group := range topicGroups {
			if groupIdx > 0 {
				content.WriteString("\n\n")
			}

			// Topic cluster header
			content.WriteString(fmt.Sprintf("### 📑 %s\n\n", group.TopicCluster))

			// Articles in this topic group
			for i, item := range group.Articles {
				if i > 0 {
					content.WriteString(template.SectionSeparator)
				}

				// Article title with sentiment emoji
				title := item.Title
				if item.SentimentEmoji != "" {
					title = fmt.Sprintf("%s %s", item.SentimentEmoji, title)
				}

				if template.IncludeSourceLinks {
					content.WriteString(fmt.Sprintf("#### %s\n\n", title))
				} else {
					content.WriteString(fmt.Sprintf("#### %s\n\n", title))
				}

				// Topic confidence indicator (if high confidence)
				if item.TopicConfidence > 0.7 {
					content.WriteString(fmt.Sprintf("*Topic relevance: %.0f%%*\n\n", item.TopicConfidence*100))
				}

				// Alert indicator (if article triggered alerts)
				if item.AlertTriggered && len(item.AlertConditions) > 0 {
					content.WriteString(fmt.Sprintf("🚨 **Alert:** %s\n\n", strings.Join(item.AlertConditions, ", ")))
				}

				// Summary
				if template.IncludeSummaries && item.SummaryText != "" {
					summary := item.SummaryText
					if template.MaxSummaryLength > 0 && len(summary) > template.MaxSummaryLength {
						summary = summary[:template.MaxSummaryLength] + "..."
					}
					content.WriteString(fmt.Sprintf("%s\n\n", summary))
				}

				// Key insights (if template supports and data available)
				if template.IncludeKeyInsights && item.MyTake != "" {
					content.WriteString(fmt.Sprintf("**Key Insight:** %s\n\n", item.MyTake))
				}

				// Source link
				if template.IncludeSourceLinks {
					content.WriteString(fmt.Sprintf("🔗 [Read more](%s)\n\n", item.URL))
				}
			}
		}
	} else {
		// Traditional flat article listing
		for i, item := range digestItems {
			if i > 0 {
				content.WriteString(template.SectionSeparator)
			}

			// Article title and source
			if template.IncludeSourceLinks {
				content.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, item.Title))
			} else {
				content.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, item.Title))
			}

			// Summary
			if template.IncludeSummaries && item.SummaryText != "" {
				summary := item.SummaryText
				if template.MaxSummaryLength > 0 && len(summary) > template.MaxSummaryLength {
					summary = summary[:template.MaxSummaryLength] + "..."
				}
				content.WriteString(fmt.Sprintf("%s\n\n", summary))
			}

			// Key insights (if template supports and data available)
			if template.IncludeKeyInsights && item.MyTake != "" {
				content.WriteString(fmt.Sprintf("**Key Insight:** %s\n\n", item.MyTake))
			}

			// Footnote citation
			content.WriteString(fmt.Sprintf("[^%d]: %s\n\n", i+1, item.URL))
		}
	}

	return content.String()
}

// renderInsightsSection renders the insights section with sentiment, alerts, trends, and research
func renderInsightsSection(digestItems []render.DigestData, template *DigestTemplate, overallSentiment string, alertsSummary string, trendsSummary string, researchSuggestions []string) string {
	var content strings.Builder

	// Only include insights for templates that support them
	if !template.IncludeKeyInsights && template.Format != FormatDetailed && template.Format != FormatNewsletter {
		return content.String()
	}

	content.WriteString("## 🧠 AI-Powered Insights\n\n")
	content.WriteString("*Leveraging sentiment analysis, alert monitoring, trend analysis, and research suggestions powered by AI.*\n\n")

	// 1. Sentiment Analysis Summary
	if overallSentiment != "" {
		content.WriteString("### 📊 Sentiment Analysis\n\n")
		content.WriteString(overallSentiment)
		content.WriteString("\n\n")

		// Add individual article sentiment indicators
		sentimentCount := make(map[string]int)
		for _, item := range digestItems {
			if item.SentimentLabel != "" {
				sentimentCount[item.SentimentLabel]++
			}
		}

		if len(sentimentCount) > 0 {
			content.WriteString("**Article Sentiment Distribution:**\n")
			for sentiment, count := range sentimentCount {
				emoji := getSentimentEmoji(sentiment)
				content.WriteString(fmt.Sprintf("- %s %s: %d articles\n", emoji, strings.Title(sentiment), count))
			}
			content.WriteString("\n")
		}
	}

	// 2. Alert Summary
	if alertsSummary != "" {
		content.WriteString("### 🚨 Alert Monitoring\n\n")
		content.WriteString(alertsSummary)
		content.WriteString("\n\n")
	} else {
		// Show that monitoring is active even if no alerts
		alertCount := 0
		for _, item := range digestItems {
			if item.AlertTriggered {
				alertCount++
			}
		}
		if alertCount == 0 {
			content.WriteString("### ✅ Alert Monitoring\n\n")
			content.WriteString("No alerts triggered for this digest. All articles passed through standard monitoring criteria.\n\n")
		}
	}

	// 3. Trend Analysis
	if trendsSummary != "" {
		content.WriteString("### 📈 Trend Analysis\n\n")
		content.WriteString(trendsSummary)
		content.WriteString("\n\n")
	}

	// 4. Research Suggestions
	if len(researchSuggestions) > 0 {
		content.WriteString("### 🔍 Research Suggestions\n\n")
		content.WriteString("*AI-generated queries for deeper exploration of these topics:*\n\n")

		// Deduplicate and limit research suggestions
		uniqueSuggestions := make(map[string]bool)
		var limitedSuggestions []string
		for _, suggestion := range researchSuggestions {
			if !uniqueSuggestions[suggestion] && len(limitedSuggestions) < 8 {
				uniqueSuggestions[suggestion] = true
				limitedSuggestions = append(limitedSuggestions, suggestion)
			}
		}

		for i, suggestion := range limitedSuggestions {
			content.WriteString(fmt.Sprintf("%d. %s\n", i+1, suggestion))
		}
		content.WriteString("\n")
	}

	return content.String()
}

// getSentimentEmoji returns the appropriate emoji for a sentiment label
func getSentimentEmoji(sentimentLabel string) string {
	switch strings.ToLower(sentimentLabel) {
	case "positive", "very positive":
		return "😊"
	case "negative", "very negative":
		return "😟"
	case "neutral":
		return "😐"
	case "mixed":
		return "🤔"
	default:
		return "📄"
	}
}

// RenderWithTemplate renders a digest using the specified template
func RenderWithTemplate(digestItems []render.DigestData, outputDir string, finalDigest string, template *DigestTemplate) (string, error) {
	return RenderWithTemplateAndMyTake(digestItems, outputDir, finalDigest, "", template)
}

// RenderWithTemplateAndMyTake renders a digest using the specified template and includes digest-level my-take
func RenderWithTemplateAndMyTake(digestItems []render.DigestData, outputDir string, finalDigest string, digestMyTake string, template *DigestTemplate) (string, error) {
	return RenderWithTemplateAndMyTakeWithTitle(digestItems, outputDir, finalDigest, digestMyTake, template, "")
}

// RenderWithTemplateAndMyTakeWithTitle renders a digest using the specified template, includes digest-level my-take, and allows custom title
func RenderWithTemplateAndMyTakeWithTitle(digestItems []render.DigestData, outputDir string, finalDigest string, digestMyTake string, template *DigestTemplate, customTitle string) (string, error) {
	dateStr := time.Now().UTC().Format("2006-01-02")
	filename := fmt.Sprintf("digest_%s_%s.md", strings.ToLower(string(template.Format)), dateStr)

	if outputDir == "" {
		outputDir = "digests"
	}

	var content strings.Builder

	// Header - use custom title if provided, otherwise use template title
	title := template.Title
	if customTitle != "" {
		title = customTitle
	}
	content.WriteString(fmt.Sprintf("# %s - %s\n\n", title, dateStr))

	// Introduction
	if template.IntroductionText != "" {
		content.WriteString(fmt.Sprintf("%s\n\n", template.IntroductionText))
	}

	// Final digest summary (if provided)
	if finalDigest != "" {
		content.WriteString("## Executive Summary\n\n")
		content.WriteString(finalDigest)
		content.WriteString("\n\n")
	}

	// Process each article using helper function
	content.WriteString(renderArticlesSection(digestItems, template))

	// Conclusion
	if template.ConclusionText != "" {
		content.WriteString(template.SectionSeparator)
		content.WriteString(template.ConclusionText)
		content.WriteString("\n")
	}

	// Prompt Corner (for newsletter format)
	if template.IncludePromptCorner && finalDigest != "" {
		promptCorner, err := llm.GeneratePromptCorner(finalDigest)
		if err == nil && promptCorner != "" {
			content.WriteString("\n\n---\n\n")
			content.WriteString("## 🎯 Prompt Corner\n\n")
			content.WriteString(promptCorner)
			content.WriteString("\n")
		}
		// If prompt generation fails, we continue without it to not break the digest
	}

	// Digest-level My Take (if provided)
	if digestMyTake != "" {
		content.WriteString("\n\n---\n\n")
		content.WriteString("## My Take\n\n")
		content.WriteString(digestMyTake)
		content.WriteString("\n")
	}

	// Write to file
	return render.WriteDigestToFile(content.String(), outputDir, filename)
}

// RenderWithTemplateAndMyTakeReturnContent renders a digest using the specified template and includes digest-level my-take,
// returning both the rendered content and the file path
func RenderWithTemplateAndMyTakeReturnContent(digestItems []render.DigestData, outputDir string, finalDigest string, digestMyTake string, template *DigestTemplate) (string, string, error) {
	return RenderWithTemplateAndMyTakeReturnContentWithTitle(digestItems, outputDir, finalDigest, digestMyTake, template, "")
}

// RenderWithTemplateAndMyTakeReturnContentWithTitle renders a digest using the specified template, includes digest-level my-take, and allows custom title,
// returning both the rendered content and the file path
func RenderWithTemplateAndMyTakeReturnContentWithTitle(digestItems []render.DigestData, outputDir string, finalDigest string, digestMyTake string, template *DigestTemplate, customTitle string) (string, string, error) {
	dateStr := time.Now().UTC().Format("2006-01-02")
	filename := fmt.Sprintf("digest_%s_%s.md", strings.ToLower(string(template.Format)), dateStr)

	if outputDir == "" {
		outputDir = "digests"
	}

	var content strings.Builder

	// Header - use custom title if provided, otherwise use template title
	title := template.Title
	if customTitle != "" {
		title = customTitle
	}
	content.WriteString(fmt.Sprintf("# %s - %s\n\n", title, dateStr))

	// Introduction
	if template.IntroductionText != "" {
		content.WriteString(fmt.Sprintf("%s\n\n", template.IntroductionText))
	}

	// Final digest summary (if provided)
	if finalDigest != "" {
		content.WriteString("## Executive Summary\n\n")
		content.WriteString(finalDigest)
		content.WriteString("\n\n")
	}

	// Process each article using helper function
	content.WriteString(renderArticlesSection(digestItems, template))

	// Conclusion
	if template.ConclusionText != "" {
		content.WriteString(template.SectionSeparator)
		content.WriteString(template.ConclusionText)
		content.WriteString("\n")
	}

	// Prompt Corner (for newsletter format)
	if template.IncludePromptCorner && finalDigest != "" {
		promptCorner, err := llm.GeneratePromptCorner(finalDigest)
		if err == nil && promptCorner != "" {
			content.WriteString("\n\n---\n\n")
			content.WriteString("## 🎯 Prompt Corner\n\n")
			content.WriteString(promptCorner)
			content.WriteString("\n")
		}
		// If prompt generation fails, we continue without it to not break the digest
	}

	// Digest-level My Take (if provided)
	if digestMyTake != "" {
		content.WriteString("\n\n---\n\n")
		content.WriteString("## My Take\n\n")
		content.WriteString(digestMyTake)
		content.WriteString("\n")
	}

	// Write to file and return both content and path
	filePath, err := render.WriteDigestToFile(content.String(), outputDir, filename)
	return content.String(), filePath, err
}

// RenderWithInsights renders a digest with comprehensive insights data
func RenderWithInsights(digestItems []render.DigestData, outputDir string, finalDigest string, digestMyTake string, template *DigestTemplate, customTitle string, overallSentiment string, alertsSummary string, trendsSummary string, researchSuggestions []string) (string, string, error) {
	dateStr := time.Now().UTC().Format("2006-01-02")
	filename := fmt.Sprintf("digest_%s_%s.md", strings.ToLower(string(template.Format)), dateStr)

	if outputDir == "" {
		outputDir = "digests"
	}

	var content strings.Builder

	// Header - use custom title if provided, otherwise use template title
	title := template.Title
	if customTitle != "" {
		title = customTitle
	}
	content.WriteString(fmt.Sprintf("# %s - %s\n\n", title, dateStr))

	// Introduction
	if template.IntroductionText != "" {
		content.WriteString(fmt.Sprintf("%s\n\n", template.IntroductionText))
	}

	// Final digest summary (if provided)
	if finalDigest != "" {
		content.WriteString("## Executive Summary\n\n")
		content.WriteString(finalDigest)
		content.WriteString("\n\n")
	}

	// AI-Powered Insights Section (new for v0.4)
	insightsSection := renderInsightsSection(digestItems, template, overallSentiment, alertsSummary, trendsSummary, researchSuggestions)
	if insightsSection != "" {
		content.WriteString(insightsSection)
		content.WriteString("\n")
	}

	// Process each article using helper function
	content.WriteString(renderArticlesSection(digestItems, template))

	// Conclusion
	if template.ConclusionText != "" {
		content.WriteString(template.SectionSeparator)
		content.WriteString(template.ConclusionText)
		content.WriteString("\n")
	}

	// Prompt Corner (for newsletter format)
	if template.IncludePromptCorner && finalDigest != "" {
		promptCorner, err := llm.GeneratePromptCorner(finalDigest)
		if err == nil && promptCorner != "" {
			content.WriteString("\n\n---\n\n")
			content.WriteString("## 🎯 Prompt Corner\n\n")
			content.WriteString(promptCorner)
			content.WriteString("\n")
		}
		// If prompt generation fails, we continue without it to not break the digest
	}

	// Digest-level My Take (if provided)
	if digestMyTake != "" {
		content.WriteString("\n\n---\n\n")
		content.WriteString("## My Take\n\n")
		content.WriteString(digestMyTake)
		content.WriteString("\n")
	}

	// Write to file and return both content and path
	filePath, err := render.WriteDigestToFile(content.String(), outputDir, filename)
	return content.String(), filePath, err
}

// GetAvailableFormats returns a list of all available format names
func GetAvailableFormats() []string {
	return []string{
		string(FormatBrief),
		string(FormatStandard),
		string(FormatDetailed),
		string(FormatNewsletter),
	}
}
