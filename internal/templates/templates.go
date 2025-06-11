package templates

import (
	"briefly/internal/core"
	"briefly/internal/email"
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
	// FormatEmail creates HTML email format
	FormatEmail DigestFormat = "email"
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
	IncludeBanner             bool // Whether to include banner image
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
			IncludeBanner:             false, // Keep minimal for brief format
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
			IncludeIndividualArticles: true,  // Enable to showcase topic clustering
			IncludeTopicClustering:    true,  // Enable topic clustering for better organization
			IncludeBanner:             false, // Standard format keeps simple
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
			IncludeIndividualArticles: true,  // Enable to showcase topic clustering
			IncludeTopicClustering:    true,  // Enable topic clustering for detailed analysis
			IncludeBanner:             false, // Detailed format focuses on content
			MaxSummaryLength:          0,     // No limit
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
			IncludeBanner:             true, // Enable banner for newsletter format
			MaxSummaryLength:          250,
			IntroductionText:          "Welcome to this week's curated selection of insights! Here's what caught our attention:",
			ConclusionText:            "Thank you for reading! Forward this to colleagues who might find it valuable.",
			SectionSeparator:          "\n\n💡 **Key Insight**\n\n",
		}
	case FormatEmail:
		return &DigestTemplate{
			Format:                    FormatEmail,
			Title:                     "Email Digest",
			IncludeSummaries:          true,
			IncludeKeyInsights:        true,
			IncludeActionItems:        true,
			IncludeSourceLinks:        true,
			IncludePromptCorner:       false,
			IncludeIndividualArticles: true,
			IncludeTopicClustering:    true, // Enable topic clustering for email organization
			IncludeBanner:             true, // Enable banner for email format
			MaxSummaryLength:          300,
			IntroductionText:          "Here's your personalized digest with today's most important insights:",
			ConclusionText:            "Stay informed and keep exploring!",
			SectionSeparator:          "\n\n---\n\n",
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
				// Add content type icon first, then sentiment emoji
				if item.ContentIcon != "" {
					title = fmt.Sprintf("%s %s", item.ContentIcon, title)
				}
				if item.SentimentEmoji != "" {
					title = fmt.Sprintf("%s %s", item.SentimentEmoji, title)
				}

				if template.IncludeSourceLinks {
					content.WriteString(fmt.Sprintf("#### %s\n\n", title))
				} else {
					content.WriteString(fmt.Sprintf("#### %s\n\n", title))
				}

				// Content type metadata (for non-HTML content)
				if item.ContentType != "html" && item.ContentType != "" {
					var metadata []string
					if item.ContentLabel != "" {
						metadata = append(metadata, item.ContentLabel)
					}
					if item.Duration > 0 {
						minutes := item.Duration / 60
						seconds := item.Duration % 60
						metadata = append(metadata, fmt.Sprintf("%d:%02d", minutes, seconds))
					}
					if item.Channel != "" {
						metadata = append(metadata, fmt.Sprintf("by %s", item.Channel))
					}
					if item.PageCount > 0 {
						metadata = append(metadata, fmt.Sprintf("%d pages", item.PageCount))
					}
					if len(metadata) > 0 {
						content.WriteString(fmt.Sprintf("*%s*\n\n", strings.Join(metadata, " • ")))
					}
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

			// Article title with content type indicator
			titleWithIcon := item.Title
			if item.ContentIcon != "" {
				titleWithIcon = fmt.Sprintf("%s %s", item.ContentIcon, item.Title)
			}

			if template.IncludeSourceLinks {
				content.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, titleWithIcon))
			} else {
				content.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, titleWithIcon))
			}

			// Content type metadata (for non-HTML content)
			if item.ContentType != "html" && item.ContentType != "" {
				var metadata []string
				if item.ContentLabel != "" {
					metadata = append(metadata, item.ContentLabel)
				}
				if item.Duration > 0 {
					minutes := item.Duration / 60
					seconds := item.Duration % 60
					metadata = append(metadata, fmt.Sprintf("%d:%02d", minutes, seconds))
				}
				if item.Channel != "" {
					metadata = append(metadata, fmt.Sprintf("by %s", item.Channel))
				}
				if item.PageCount > 0 {
					metadata = append(metadata, fmt.Sprintf("%d pages", item.PageCount))
				}
				if len(metadata) > 0 {
					content.WriteString(fmt.Sprintf("*%s*\n\n", strings.Join(metadata, " • ")))
				}
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

// renderAlertsSection renders just the alerts section
func renderAlertsSection(digestItems []render.DigestData, alertsSummary string) string {
	var content strings.Builder

	if alertsSummary != "" {
		content.WriteString(alertsSummary)
		content.WriteString("\n")
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

	return content.String()
}

// renderInsightsSection renders the insights section with sentiment, trends, and research (alerts moved to separate section)
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
				// Capitalize first letter manually to avoid deprecated strings.Title
				capitalized := strings.ToUpper(sentiment[:1]) + sentiment[1:]
				content.WriteString(fmt.Sprintf("- %s %s: %d articles\n", emoji, capitalized, count))
			}
			content.WriteString("\n")
		}
	}

	// 2. Trend Analysis (alerts moved to separate section above)
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

// renderBannerSection renders the banner image section for formats that support it
func renderBannerSection(banner *core.BannerImage, template *DigestTemplate, format string) string {
	if banner == nil || !template.IncludeBanner {
		return ""
	}

	var content strings.Builder

	switch format {
	case "markdown":
		// Markdown format with image and alt text
		if banner.AltText != "" {
			content.WriteString(fmt.Sprintf("![%s](%s)\n\n", banner.AltText, banner.ImageURL))
		} else {
			content.WriteString(fmt.Sprintf("![AI-generated banner](%s)\n\n", banner.ImageURL))
		}

		// Optional: Add themes as subtle text
		if len(banner.Themes) > 0 {
			content.WriteString(fmt.Sprintf("*Featured themes: %s*\n\n", strings.Join(banner.Themes, ", ")))
		}

	case "html":
		// HTML format for email templates
		content.WriteString(fmt.Sprintf(`<img src="%s" alt="%s" style="width: 100%%; max-width: 600px; height: auto; border-radius: 8px; margin-bottom: 20px;" />`,
			banner.ImageURL, banner.AltText))
		content.WriteString("\n\n")

	case "plain":
		// Plain text fallback
		content.WriteString("🎨 **Banner Image Available**\n")
		if len(banner.Themes) > 0 {
			content.WriteString(fmt.Sprintf("Themes: %s\n", strings.Join(banner.Themes, ", ")))
		}
		content.WriteString(fmt.Sprintf("View at: %s\n\n", banner.ImageURL))

	default:
		// Default to markdown format
		return renderBannerSection(banner, template, "markdown")
	}

	return content.String()
}

// renderReferencesSection renders the references section with numbered citations
func renderReferencesSection(digestItems []render.DigestData) string {
	if len(digestItems) == 0 {
		return ""
	}

	var content strings.Builder
	content.WriteString("## References\n\n")

	for i, item := range digestItems {
		content.WriteString(fmt.Sprintf("[%d] %s\n", i+1, item.URL))
		if item.Title != "" {
			content.WriteString(fmt.Sprintf("    *%s*\n", item.Title))
		}
		content.WriteString("\n")
	}

	return content.String()
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

	// References section
	referencesSection := renderReferencesSection(digestItems)
	if referencesSection != "" {
		content.WriteString("\n\n---\n\n")
		content.WriteString(referencesSection)
	}

	// Write to file and return both content and path
	filePath, err := render.WriteDigestToFile(content.String(), outputDir, filename)
	return content.String(), filePath, err
}

// RenderWithBanner renders a digest with banner image support
func RenderWithBanner(digestItems []render.DigestData, outputDir string, finalDigest string, digestMyTake string, template *DigestTemplate, customTitle string, banner *core.BannerImage) (string, string, error) {
	return RenderWithBannerAndInsights(digestItems, outputDir, finalDigest, digestMyTake, template, customTitle, "", "", "", []string{}, banner)
}

// RenderWithInsights renders a digest with comprehensive insights data (backward compatibility)
func RenderWithInsights(digestItems []render.DigestData, outputDir string, finalDigest string, digestMyTake string, template *DigestTemplate, customTitle string, overallSentiment string, alertsSummary string, trendsSummary string, researchSuggestions []string) (string, string, error) {
	return RenderWithBannerAndInsights(digestItems, outputDir, finalDigest, digestMyTake, template, customTitle, overallSentiment, alertsSummary, trendsSummary, researchSuggestions, nil)
}

// RenderWithBannerAndInsights renders a digest with both banner and insights data
func RenderWithBannerAndInsights(digestItems []render.DigestData, outputDir string, finalDigest string, digestMyTake string, template *DigestTemplate, customTitle string, overallSentiment string, alertsSummary string, trendsSummary string, researchSuggestions []string, banner *core.BannerImage) (string, string, error) {
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

	// Banner image (if provided and template supports it)
	bannerSection := renderBannerSection(banner, template, "markdown")
	if bannerSection != "" {
		content.WriteString(bannerSection)
	}

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

	// Alert Monitoring Section (moved up for prominence)
	alertsSection := renderAlertsSection(digestItems, alertsSummary)
	if alertsSection != "" {
		content.WriteString(alertsSection)
		content.WriteString("\n")
	}

	// AI-Powered Insights Section (without alerts, which are now shown above)
	insightsSection := renderInsightsSection(digestItems, template, overallSentiment, "", trendsSummary, researchSuggestions)
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

	// References section
	referencesSection := renderReferencesSection(digestItems)
	if referencesSection != "" {
		content.WriteString("\n\n---\n\n")
		content.WriteString(referencesSection)
	}

	// Write to file and return both content and path
	filePath, err := render.WriteDigestToFile(content.String(), outputDir, filename)
	return content.String(), filePath, err
}

// RenderHTMLEmail renders a digest as HTML email
func RenderHTMLEmail(digestItems []render.DigestData, outputDir string, finalDigest string, customTitle string, overallSentiment string, alertsSummary string, trendsSummary string, researchSuggestions []string, emailStyle string) (string, string, error) {
	return RenderHTMLEmailWithBanner(digestItems, outputDir, finalDigest, customTitle, overallSentiment, alertsSummary, trendsSummary, researchSuggestions, emailStyle, nil)
}

// RenderHTMLEmailWithBanner renders a digest as HTML email with banner support
func RenderHTMLEmailWithBanner(digestItems []render.DigestData, outputDir string, finalDigest string, customTitle string, overallSentiment string, alertsSummary string, trendsSummary string, researchSuggestions []string, emailStyle string, banner *core.BannerImage) (string, string, error) {
	template := GetTemplate(FormatEmail)

	// Choose email template style
	var emailTemplate *email.EmailTemplate
	switch emailStyle {
	case "newsletter":
		emailTemplate = email.GetNewsletterEmailTemplate()
	case "minimal":
		emailTemplate = email.GetMinimalEmailTemplate()
	default:
		emailTemplate = email.GetDefaultEmailTemplate()
	}

	// Convert digest data to email format
	title := customTitle
	if title == "" {
		title = template.Title
	}

	emailData := email.ConvertDigestToEmailWithBanner(
		digestItems,
		title,
		template.IntroductionText,
		finalDigest,
		template.ConclusionText,
		overallSentiment,
		alertsSummary,
		trendsSummary,
		researchSuggestions,
		banner,
	)

	// Render HTML email
	htmlContent, err := email.RenderHTMLEmail(emailData, emailTemplate)
	if err != nil {
		return "", "", fmt.Errorf("failed to render HTML email: %w", err)
	}

	// Write HTML file
	dateStr := time.Now().UTC().Format("2006-01-02")
	filename := fmt.Sprintf("digest_email_%s.html", dateStr)

	if outputDir == "" {
		outputDir = "digests"
	}

	filePath, err := email.WriteHTMLEmail(htmlContent, outputDir, filename)
	if err != nil {
		return "", "", fmt.Errorf("failed to write HTML email: %w", err)
	}

	return htmlContent, filePath, nil
}

// GetAvailableFormats returns a list of all available format names
func GetAvailableFormats() []string {
	return []string{
		string(FormatBrief),
		string(FormatStandard),
		string(FormatDetailed),
		string(FormatNewsletter),
		string(FormatEmail),
	}
}
