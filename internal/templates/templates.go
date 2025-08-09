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
	// FormatScannableNewsletter creates a scannable newsletter format with prominent links
	FormatScannableNewsletter DigestFormat = "scannable"
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
	MaxSummaryLength          int  // 0 for no limit (in words for v2.0)
	MaxDigestWords            int  // v2.0: Maximum total words for entire digest (0 for no limit)
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
			MaxSummaryLength:          25,    // v2.0: 15-25 words per article summary
			MaxDigestWords:            200,   // v2.0: 200-word target for brief format
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
			MaxSummaryLength:          25,    // v2.0: 15-25 words per article summary
			MaxDigestWords:            400,   // v2.0: 400-word target for standard format
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
			MaxSummaryLength:          50,    // v2.0: Longer summaries for detailed format but still controlled
			MaxDigestWords:            0,     // No limit for detailed format
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
			MaxSummaryLength:          25,   // v2.0: 15-25 words per article summary
			MaxDigestWords:            800,  // v2.0: 800-word target for newsletter format to include more articles
			IntroductionText:          "Welcome to this week's curated selection of insights! Here's what caught our attention:",
			ConclusionText:            "Thank you for reading! Forward this to colleagues who might find it valuable.",
			SectionSeparator:          "\n\n---\n\n",
		}
	case FormatScannableNewsletter:
		return &DigestTemplate{
			Format:                    FormatScannableNewsletter,
			Title:                     "Briefly Bytes",
			IncludeSummaries:          true,
			IncludeKeyInsights:        false, // Remove AI insights for bite-sized format
			IncludeActionItems:        false, // Remove action items for bite-sized format
			IncludeSourceLinks:        true,
			IncludePromptCorner:       false, // Remove prompt corner for bite-sized format
			IncludeIndividualArticles: true,  // Enable individual articles in scannable format
			IncludeTopicClustering:    false, // Disable clustering for scannable - use flat structure
			IncludeBanner:             false, // Remove banner for bite-sized format
			MaxSummaryLength:          25,    // Allow complete sentences for bite-sized format
			MaxDigestWords:            400,   // Reduced word count for bite-sized format
			IntroductionText:          "This week's tech highlights - bite-sized for busy schedules:",
			ConclusionText:            "Keep learning, keep building 🚀",
			SectionSeparator:          "\n\n",
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
			MaxSummaryLength:          25,   // v2.0: 15-25 words per article summary
			MaxDigestWords:            400,  // v2.0: 400-word target for email format
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

// countWords counts words in a text string
func countWords(text string) int {
	if text == "" {
		return 0
	}
	words := strings.Fields(text)
	return len(words)
}

// estimateReadTime estimates read time based on word count (assuming 200 WPM)
func estimateReadTime(wordCount int) string {
	if wordCount == 0 {
		return "0m"
	}

	minutes := wordCount / 200 // 200 words per minute reading speed
	if minutes == 0 {
		return "<1m"
	}
	return fmt.Sprintf("%dm", minutes)
}

// generateWordCountHeader generates a header with word count and read time
func generateWordCountHeader(wordCount int) string {
	readTime := estimateReadTime(wordCount)
	return fmt.Sprintf("📊 %d words • ⏱️ %s read\n\n", wordCount, readTime)
}

// truncateToWordLimit truncates text to stay within word limit
func truncateToWordLimit(text string, maxWords int) string {
	if maxWords <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) <= maxWords {
		return text
	}

	return strings.Join(words[:maxWords], " ") + "..."
}

// truncateToCompleteSentence truncates text to complete sentences within word limit
func truncateToCompleteSentence(text string, maxWords int) string {
	if maxWords <= 0 {
		return text
	}

	words := strings.Fields(text)
	if len(words) <= maxWords {
		return text
	}

	// Find the last complete sentence within the word limit
	truncated := strings.Join(words[:maxWords], " ")
	
	// Look for sentence endings
	sentences := strings.FieldsFunc(truncated, func(r rune) bool {
		return r == '.' || r == '!' || r == '?'
	})
	
	if len(sentences) > 0 {
		// Return the first complete sentence
		firstSentence := strings.TrimSpace(sentences[0])
		if firstSentence != "" {
			return firstSentence + "."
		}
	}
	
	// If no complete sentence found, fall back to word limit with ellipsis
	return strings.Join(words[:maxWords], " ") + "..."
}

// renderArticlesSection renders the articles section with optional topic clustering
func renderArticlesSection(digestItems []render.DigestData, template *DigestTemplate) string {
	var content strings.Builder

	if !template.IncludeIndividualArticles {
		return content.String()
	}

	// Use scannable format for the new scannable newsletter format
	if template.Format == FormatScannableNewsletter {
		return renderScannableArticlesSection(digestItems, template)
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
					// v2.0: Use word-based truncation instead of character-based
					if template.MaxSummaryLength > 0 {
						summary = truncateToWordLimit(summary, template.MaxSummaryLength)
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
				// v2.0: Use word-based truncation instead of character-based
				if template.MaxSummaryLength > 0 {
					summary = truncateToWordLimit(summary, template.MaxSummaryLength)
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

// renderScannableArticlesSection renders articles in a scannable newsletter format
func renderScannableArticlesSection(digestItems []render.DigestData, template *DigestTemplate) string {
	var content strings.Builder

	// Check if articles are categorized (look for category info in MyTake)
	categorized := false
	categoryGroups := make(map[string][]render.DigestData)
	categoryOrder := []string{"🔥 Breaking & Hot", "🚀 Product Updates", "🛠️ Dev Tools & Techniques", "📊 Research & Analysis", "💡 Ideas & Inspiration", "🔍 Worth Monitoring"}

	for _, item := range digestItems {
		if item.MyTake != "" && strings.Contains(item.MyTake, " ") {
			// Try to extract category from MyTake (format: "🔥 Breaking & Hot | insight")
			parts := strings.Split(item.MyTake, " | ")
			if len(parts) >= 1 {
				categoryName := strings.TrimSpace(parts[0])
				if categoryName != "" && strings.Contains(categoryName, " ") {
					categorized = true
					categoryGroups[categoryName] = append(categoryGroups[categoryName], item)
				}
			}
		}
	}

	if categorized {
		content.WriteString("## 📖 Featured Articles\n\n")
		
		// Render articles grouped by category in priority order
		for _, categoryName := range categoryOrder {
			if articles, exists := categoryGroups[categoryName]; exists && len(articles) > 0 {
				content.WriteString(fmt.Sprintf("### %s\n\n", categoryName))
				
				for i, item := range articles {
					if i > 0 {
						content.WriteString("\n")
					}
					
					// Get content type emoji
					contentEmoji := getContentTypeEmoji(item.ContentType, item.Title)
					
					// Article title with content type emoji
					content.WriteString(fmt.Sprintf("**%s %s**\n\n", contentEmoji, item.Title))
					
					// Key insight (summary) - simplified to just the summary
					if template.IncludeSummaries && item.SummaryText != "" {
						summary := item.SummaryText
						if template.MaxSummaryLength > 0 {
							summary = truncateToCompleteSentence(summary, template.MaxSummaryLength)
						}
						content.WriteString(fmt.Sprintf("%s\n\n", summary))
					}
					
					// Link with clear call to action  
					if template.IncludeSourceLinks {
						content.WriteString(fmt.Sprintf("🔗 [Read more](%s)\n\n", item.URL))
					}
				}
			}
		}
		
		// Handle any uncategorized items
		uncategorized := []render.DigestData{}
		for _, item := range digestItems {
			found := false
			for _, articles := range categoryGroups {
				for _, catItem := range articles {
					if catItem.URL == item.URL {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if !found {
				uncategorized = append(uncategorized, item)
			}
		}
		
		if len(uncategorized) > 0 {
			content.WriteString("### 🔍 Additional Items\n\n")
			for i, item := range uncategorized {
				if i > 0 {
					content.WriteString("\n")
				}
				contentEmoji := getContentTypeEmoji(item.ContentType, item.Title)
				content.WriteString(fmt.Sprintf("**%s %s**\n\n", contentEmoji, item.Title))
				
				if template.IncludeSummaries && item.SummaryText != "" {
					summary := item.SummaryText
					if template.MaxSummaryLength > 0 {
						summary = truncateToCompleteSentence(summary, template.MaxSummaryLength)
					}
					content.WriteString(fmt.Sprintf("%s\n\n", summary))
				}
				
				if template.IncludeSourceLinks {
					content.WriteString(fmt.Sprintf("🔗 [Read more](%s)\n\n", item.URL))
				}
			}
		}
	} else {
		// Fallback to original format if not categorized
		content.WriteString("## 📖 Featured Articles\n\n")

		for i, item := range digestItems {
			if i > 0 {
				content.WriteString("\n")
			}

			// Get content type emoji
			contentEmoji := getContentTypeEmoji(item.ContentType, item.Title)

			// Article title with content type emoji
			content.WriteString(fmt.Sprintf("### %s %s\n\n", contentEmoji, item.Title))

			// Key insight (summary) - simplified to just the summary
			if template.IncludeSummaries && item.SummaryText != "" {
				summary := item.SummaryText
				if template.MaxSummaryLength > 0 {
					summary = truncateToCompleteSentence(summary, template.MaxSummaryLength)
				}
				content.WriteString(fmt.Sprintf("%s\n\n", summary))
			}

			// Link with clear call to action
			if template.IncludeSourceLinks {
				content.WriteString(fmt.Sprintf("🔗 [Read more](%s)\n\n", item.URL))
			}
		}
	}

	return content.String()
}

// getContentTypeEmoji returns appropriate emoji based on content type and title
func getContentTypeEmoji(contentType, title string) string {
	titleLower := strings.ToLower(title)

	// Check for video content
	if contentType == "youtube" || strings.Contains(titleLower, "video") {
		return "🎥"
	}

	// Check for PDF/documentation
	if contentType == "pdf" || strings.Contains(titleLower, "guide") || strings.Contains(titleLower, "documentation") {
		return "📄"
	}

	// Check for tools/products
	if strings.Contains(titleLower, "tool") || strings.Contains(titleLower, "platform") || strings.Contains(titleLower, "app") {
		return "🛠️"
	}

	// Check for research/studies
	if strings.Contains(titleLower, "research") || strings.Contains(titleLower, "study") || strings.Contains(titleLower, "analysis") {
		return "📊"
	}

	// Check for tutorials/how-to
	if strings.Contains(titleLower, "how to") || strings.Contains(titleLower, "tutorial") || strings.Contains(titleLower, "guide") {
		return "📚"
	}

	// Check for news/announcements
	if strings.Contains(titleLower, "announcement") || strings.Contains(titleLower, "release") || strings.Contains(titleLower, "launch") {
		return "📢"
	}

	// Check for AI/ML specific content
	if strings.Contains(titleLower, "ai") || strings.Contains(titleLower, "machine learning") || strings.Contains(titleLower, "llm") {
		return "🤖"
	}

	// Check for performance/optimization
	if strings.Contains(titleLower, "performance") || strings.Contains(titleLower, "optimization") || strings.Contains(titleLower, "speed") {
		return "⚡"
	}

	// Check for security
	if strings.Contains(titleLower, "security") || strings.Contains(titleLower, "privacy") || strings.Contains(titleLower, "vulnerability") {
		return "🔒"
	}

	// Default to hot/trending content
	return "🔥"
}

// generateWhyItMatters creates a default "why it matters" statement when MyTake is not available
func generateWhyItMatters(item render.DigestData) string {
	titleLower := strings.ToLower(item.Title)

	// AI/ML related
	if strings.Contains(titleLower, "ai") || strings.Contains(titleLower, "llm") || strings.Contains(titleLower, "machine learning") {
		return "Key development in AI that could impact how we work with intelligent systems"
	}

	// Tools/platforms
	if strings.Contains(titleLower, "tool") || strings.Contains(titleLower, "platform") {
		return "New tool that could enhance productivity and development workflows"
	}

	// Security
	if strings.Contains(titleLower, "security") || strings.Contains(titleLower, "vulnerability") {
		return "Security insight important for protecting systems and data"
	}

	// Performance/optimization
	if strings.Contains(titleLower, "performance") || strings.Contains(titleLower, "optimization") {
		return "Performance insight that could improve system efficiency"
	}

	// Research/studies
	if strings.Contains(titleLower, "research") || strings.Contains(titleLower, "study") {
		return "Research findings that provide data-driven insights for decision making"
	}

	// Generic fallback
	return "Important development worth understanding for staying current in tech"
}

// capitalizeFirst capitalizes the first letter of a string
func capitalizeFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
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
			return "" // Don't show alerts section if there are no alerts
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
	content.WriteString("*Here's what the data tells us about the themes and patterns in this week's content:*\n\n")

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

// renderActionableSection renders the "Try This Week" section with specific, actionable recommendations
func renderActionableSection(digestItems []render.DigestData, template *DigestTemplate) string {
	var content strings.Builder

	// Only include actionable section for formats that support it
	if !template.IncludeActionItems {
		return content.String()
	}

	content.WriteString("## ⚡ Try This Week\n\n")
	content.WriteString("*Actionable takeaways from this digest - pick one to implement:*\n\n")

	var actions []string

	// Extract actionable insights from articles
	for i, item := range digestItems {
		if i >= 3 { // Limit to top 3 articles to keep focused
			break
		}

		action := generateActionableItem(item)
		if action != "" {
			actions = append(actions, action)
		}
	}

	// If we don't have enough specific actions, add some general ones
	if len(actions) < 2 {
		actions = append(actions, generateFallbackActions(digestItems)...)
	}

	// Limit to 2-3 actions to keep focused
	if len(actions) > 3 {
		actions = actions[:3]
	}

	// Render action items
	for i, action := range actions {
		if i >= 3 { // Hard limit to 3 actions
			break
		}
		content.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, action))
	}

	content.WriteString("\n*💡 Pro tip: Start with just one item - small actions lead to big results.*\n\n")

	return content.String()
}

// generateActionableItem creates a specific, actionable item from an article
func generateActionableItem(item render.DigestData) string {
	title := strings.ToLower(item.Title)
	summary := strings.ToLower(item.SummaryText)

	// Technology-specific actionable recommendations
	if strings.Contains(title, "api") || strings.Contains(summary, "api") {
		return "Test the mentioned API in a small project this week"
	}

	if strings.Contains(title, "tool") || strings.Contains(title, "library") {
		return fmt.Sprintf("Evaluate %s for your current tech stack", extractToolName(item.Title))
	}

	if strings.Contains(title, "security") || strings.Contains(summary, "security") {
		return "Audit one security practice in your current projects"
	}

	if strings.Contains(title, "performance") || strings.Contains(summary, "optimization") {
		return "Profile and optimize one slow function in your codebase"
	}

	if strings.Contains(title, "testing") || strings.Contains(summary, "test") {
		return "Add tests for one untested module in your project"
	}

	if strings.Contains(title, "docker") || strings.Contains(title, "container") {
		return "Containerize one service in your development environment"
	}

	if strings.Contains(title, "ai") || strings.Contains(title, "llm") || strings.Contains(title, "ml") {
		return "Experiment with AI integration in your next feature"
	}

	if strings.Contains(title, "database") || strings.Contains(summary, "database") {
		return "Optimize one slow database query in your application"
	}

	if strings.Contains(title, "monitoring") || strings.Contains(summary, "observability") {
		return "Add monitoring to one critical service endpoint"
	}

	if strings.Contains(title, "deployment") || strings.Contains(summary, "deploy") {
		return "Automate one manual deployment step in your workflow"
	}

	// Framework-specific actions
	if strings.Contains(title, "react") {
		return "Refactor one React component using best practices from the article"
	}

	if strings.Contains(title, "kubernetes") || strings.Contains(title, "k8s") {
		return "Review your Kubernetes resource limits and requests"
	}

	if strings.Contains(title, "go") && !strings.Contains(title, "google") {
		return "Apply Go performance patterns to your current project"
	}

	if strings.Contains(title, "rust") {
		return "Explore Rust for your next system-level component"
	}

	// Generic actionable item based on content
	if strings.Contains(summary, "improve") || strings.Contains(summary, "better") {
		return "Implement one improvement technique from this article"
	}

	if strings.Contains(summary, "learn") || strings.Contains(summary, "tutorial") {
		return "Complete the tutorial or example mentioned in the article"
	}

	// Default action
	return fmt.Sprintf("Research %s for potential application in your work", extractKeyTerm(item.Title))
}

// generateFallbackActions provides general actionable items when specific ones can't be generated
func generateFallbackActions(digestItems []render.DigestData) []string {
	actions := []string{
		"Refactor one function using patterns from this week's reading",
		"Share one insight from these articles with your team",
		"Bookmark one tool mentioned for future evaluation",
		"Update one dependency in your current project",
		"Write documentation for one undocumented feature",
	}

	// Try to make it more specific if we can detect patterns
	hasAI := false
	hasPerf := false
	hasTools := false

	for _, item := range digestItems {
		titleLower := strings.ToLower(item.Title)
		if strings.Contains(titleLower, "ai") || strings.Contains(titleLower, "llm") {
			hasAI = true
		}
		if strings.Contains(titleLower, "performance") || strings.Contains(titleLower, "optimization") {
			hasPerf = true
		}
		if strings.Contains(titleLower, "tool") || strings.Contains(titleLower, "library") {
			hasTools = true
		}
	}

	var contextualActions []string
	if hasAI {
		contextualActions = append(contextualActions, "Explore one AI use case for your current project")
	}
	if hasPerf {
		contextualActions = append(contextualActions, "Benchmark one performance-critical operation")
	}
	if hasTools {
		contextualActions = append(contextualActions, "Evaluate one new tool mentioned in the articles")
	}

	// Return contextual actions if available, otherwise fallback
	if len(contextualActions) > 0 {
		return contextualActions
	}

	return actions[:2] // Return first 2 generic actions
}

// extractToolName attempts to extract tool/library name from title
func extractToolName(title string) string {
	// Simple extraction - look for common patterns
	words := strings.Fields(title)
	for _, word := range words {
		// Skip common articles and prepositions
		if len(word) > 2 && !isCommonWord(word) {
			// Look for tool-like words (often capitalized or have specific patterns)
			if strings.Contains(strings.ToLower(word), "js") ||
				strings.Contains(strings.ToLower(word), "lib") ||
				word[0] >= 'A' && word[0] <= 'Z' {
				return word
			}
		}
	}
	return "this technology"
}

// extractKeyTerm extracts a key technical term from the title
func extractKeyTerm(title string) string {
	titleLower := strings.ToLower(title)

	// Technical terms to look for
	techTerms := []string{
		"kubernetes", "docker", "react", "vue", "angular", "node", "python", "go", "rust",
		"typescript", "javascript", "api", "microservices", "serverless", "cloud", "aws",
		"azure", "gcp", "database", "postgresql", "mysql", "mongodb", "redis", "graphql",
		"machine learning", "ai", "llm", "neural", "blockchain", "crypto", "security",
		"devops", "cicd", "testing", "monitoring", "performance", "optimization",
	}

	for _, term := range techTerms {
		if strings.Contains(titleLower, term) {
			return term
		}
	}

	// If no specific term found, return generic
	return "the concepts discussed"
}

// isCommonWord checks if word is a common article/preposition to skip
func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "how": true, "why": true, "what": true, "when": true,
		"where": true, "is": true, "are": true, "was": true, "were": true, "be": true,
		"been": true, "being": true, "have": true, "has": true, "had": true, "will": true,
		"would": true, "could": true, "should": true, "may": true, "might": true, "can": true,
		"this": true, "that": true, "these": true, "those": true, "your": true, "you": true,
	}
	return commonWords[strings.ToLower(word)]
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

	// Use all references
	referenceCount := len(digestItems)

	for i := 0; i < referenceCount; i++ {
		item := digestItems[i]
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

// RenderTeamFriendlyBrief renders the new team-focused brief format with emoji categorization
func RenderTeamFriendlyBrief(digestItems []render.DigestData, outputDir string, customTitle string, teamContext string) (string, string, error) {
	dateStr := time.Now().UTC().Format("2006-01-02")
	filename := fmt.Sprintf("digest_brief_%s.md", dateStr)

	if outputDir == "" {
		outputDir = "digests"
	}

	var content strings.Builder

	// Header
	title := "Weekly Tech Radar"
	if customTitle != "" {
		title = customTitle
	}
	content.WriteString(fmt.Sprintf("# 📚 %s - %s\n\n", title, time.Now().Format("Jan 2, 2006")))

	// Categorize articles
	categories := categorizeArticlesForBrief(digestItems)

	// Product Launches & Major Updates
	if len(categories.ProductLaunches) > 0 {
		content.WriteString("## 🎯 **Product Launches & Major Updates**\n\n")
		for _, item := range categories.ProductLaunches {
			content.WriteString(fmt.Sprintf("- **%s** - [Link](%s)\n", item.Title, item.URL))
			content.WriteString(fmt.Sprintf("  What: %s\n", extractOneLineDescription(item.SummaryText)))
			if item.MyTake != "" {
				content.WriteString(fmt.Sprintf("  Why it matters: %s\n\n", item.MyTake))
			} else {
				content.WriteString("  Why it matters: New development in the ecosystem\n\n")
			}
		}
	}

	// Engineering Deep Dives
	if len(categories.EngineeringDeepDives) > 0 {
		content.WriteString("## 🛠️ **Engineering Deep Dives**\n\n")
		for _, item := range categories.EngineeringDeepDives {
			content.WriteString(fmt.Sprintf("- **%s** - [Link](%s)\n", item.Title, item.URL))
			content.WriteString(fmt.Sprintf("  What: %s\n", extractCoreConceptDescription(item.SummaryText)))
			if item.MyTake != "" {
				content.WriteString(fmt.Sprintf("  Why it matters: %s\n\n", item.MyTake))
			} else {
				content.WriteString("  Why it matters: Technical insights for our development approach\n\n")
			}
		}
	}

	// Interesting Implementations
	if len(categories.InterestingImplementations) > 0 {
		content.WriteString("## 💡 **Interesting Implementations**\n\n")
		for _, item := range categories.InterestingImplementations {
			content.WriteString(fmt.Sprintf("- **%s** - [Link](%s)\n", item.Title, item.URL))
			content.WriteString(fmt.Sprintf("  What: %s\n", extractImplementationDescription(item.SummaryText)))
			if item.MyTake != "" {
				content.WriteString(fmt.Sprintf("  Why it matters: %s\n\n", item.MyTake))
			} else {
				content.WriteString("  Why it matters: Lessons we can apply to our projects\n\n")
			}
		}
	}

	// Worth Exploring
	if len(categories.WorthExploring) > 0 {
		content.WriteString("## 🔍 **Worth Exploring**\n\n")
		for _, item := range categories.WorthExploring {
			description := extractOneLineDescription(item.SummaryText)
			content.WriteString(fmt.Sprintf("- **%s** - [Link](%s) → %s\n", item.Title, item.URL, description))
		}
		content.WriteString("\n")
	}

	// Footer
	content.WriteString("---\n")
	content.WriteString(fmt.Sprintf("*Generated with team context • %d articles • Forward to your team*\n", len(digestItems)))

	// Write to file
	filePath, err := render.WriteDigestToFile(content.String(), outputDir, filename)
	return content.String(), filePath, err
}

// ArticleCategories holds categorized articles for brief format
type ArticleCategories struct {
	ProductLaunches            []render.DigestData
	EngineeringDeepDives       []render.DigestData
	InterestingImplementations []render.DigestData
	WorthExploring             []render.DigestData
}

// categorizeArticlesForBrief categorizes articles based on title and content keywords
func categorizeArticlesForBrief(digestItems []render.DigestData) ArticleCategories {
	categories := ArticleCategories{}

	for _, item := range digestItems {
		titleLower := strings.ToLower(item.Title)
		summaryLower := strings.ToLower(item.SummaryText)

		// Product launches & updates
		if containsAny(titleLower, []string{"launch", "release", "announce", "unveil", "version", "update"}) ||
			containsAny(summaryLower, []string{"launched", "released", "announced", "new version", "updated"}) {
			categories.ProductLaunches = append(categories.ProductLaunches, item)
		} else if containsAny(titleLower, []string{"deep dive", "guide", "tutorial", "how to", "engineering", "architecture", "system", "design", "performance"}) ||
			containsAny(summaryLower, []string{"technical", "engineering", "architecture", "system design", "performance", "optimization"}) {
			categories.EngineeringDeepDives = append(categories.EngineeringDeepDives, item)
		} else if containsAny(titleLower, []string{"built", "implementation", "project", "case study", "building"}) ||
			containsAny(summaryLower, []string{"implementation", "built", "developed", "created", "project"}) {
			categories.InterestingImplementations = append(categories.InterestingImplementations, item)
		} else {
			// Default to worth exploring
			categories.WorthExploring = append(categories.WorthExploring, item)
		}
	}

	return categories
}

// containsAny checks if text contains any of the given keywords
func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

// extractOneLineDescription extracts a concise one-line description from summary
func extractOneLineDescription(summary string) string {
	sentences := strings.Split(summary, ".")
	if len(sentences) > 0 && len(sentences[0]) > 0 {
		cleaned := strings.TrimSpace(sentences[0])
		if len(cleaned) > 80 {
			return cleaned[:77] + "..."
		}
		return cleaned
	}
	if len(summary) > 80 {
		return summary[:77] + "..."
	}
	return summary
}

// extractCoreConceptDescription extracts technical concept from summary
func extractCoreConceptDescription(summary string) string {
	// Look for technical keywords and extract relevant sentences
	summary = strings.TrimSpace(summary)
	if len(summary) > 100 {
		return summary[:97] + "..."
	}
	return summary
}

// extractImplementationDescription extracts what was built/solved from summary
func extractImplementationDescription(summary string) string {
	// Similar to extractCoreConceptDescription but focused on implementations
	summary = strings.TrimSpace(summary)
	if len(summary) > 100 {
		return summary[:97] + "..."
	}
	return summary
}

// RenderWithStructuredContent renders a digest using the new cohesive LLM-generated approach
func RenderWithStructuredContent(digestItems []render.DigestData, outputDir string, structuredContent string, template *DigestTemplate, customTitle string, banner *core.BannerImage) (string, string, error) {
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

	// Word count and read time statistics
	wordCount := countWords(structuredContent)
	if wordCount > 0 {
		wordCountHeader := generateWordCountHeader(wordCount)
		content.WriteString(wordCountHeader)
	}

	// Banner image (if provided and template supports it)
	bannerSection := renderBannerSection(banner, template, "markdown")
	if bannerSection != "" {
		content.WriteString(bannerSection)
	}

	// Introduction
	if template.IntroductionText != "" {
		content.WriteString(fmt.Sprintf("%s\n\n", template.IntroductionText))
	}

	// Main structured content (LLM-generated cohesive digest)
	if structuredContent != "" {
		content.WriteString(structuredContent)
		content.WriteString("\n\n")
	}

	// Conclusion
	if template.ConclusionText != "" {
		content.WriteString("\n\n---\n\n")
		content.WriteString(template.ConclusionText)
		content.WriteString("\n")
	}

	// Prompt Corner (for newsletter format)
	if template.IncludePromptCorner && structuredContent != "" {
		promptCorner, err := llm.GeneratePromptCorner(structuredContent)
		if err == nil && promptCorner != "" {
			content.WriteString("\n\n---\n\n")
			content.WriteString("## 🎯 Prompt Corner\n\n")
			content.WriteString(promptCorner)
			content.WriteString("\n")
		}
	}

	// References section (up to 7 references)
	referencesSection := renderReferencesSection(digestItems)
	if referencesSection != "" {
		content.WriteString("\n\n---\n\n")
		content.WriteString(referencesSection)
	}

	// Write to file and return both content and path
	filePath, err := render.WriteDigestToFile(content.String(), outputDir, filename)
	return content.String(), filePath, err
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
		// v2.0: Limit executive summary to 100-150 words
		executiveSummary := finalDigest
		if template.MaxDigestWords > 0 {
			executiveSummary = truncateToWordLimit(finalDigest, 150) // Max 150 words for executive summary
		}
		content.WriteString(executiveSummary)
		content.WriteString("\n\n")
	}

	// Alert Monitoring Section (moved up for prominence) - skip for scannable format
	if template.Format != FormatScannableNewsletter {
		alertsSection := renderAlertsSection(digestItems, alertsSummary)
		if alertsSection != "" {
			content.WriteString("## 🚨 Alerts\n\n")
			content.WriteString(alertsSection)
			content.WriteString("\n")
		}
	}

	// AI-Powered Insights Section (without alerts, which are now shown above) - skip for scannable format
	if template.Format != FormatScannableNewsletter {
		insightsSection := renderInsightsSection(digestItems, template, overallSentiment, "", trendsSummary, researchSuggestions)
		if insightsSection != "" {
			content.WriteString(insightsSection)
			content.WriteString("\n")
		}
	}

	// v2.0: Try This Week section for actionable recommendations - skip for scannable format
	if template.Format != FormatScannableNewsletter {
		actionSection := renderActionableSection(digestItems, template)
		if actionSection != "" {
			content.WriteString(actionSection)
			content.WriteString("\n")
		}
	}

	// Process each article using helper function (only for detailed formats)
	articlesSection := renderArticlesSection(digestItems, template)
	if articlesSection != "" {
		content.WriteString("\n\n---\n\n")
		content.WriteString(articlesSection)
	}

	// Conclusion
	if template.ConclusionText != "" {
		content.WriteString("\n\n")
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

	// v2.0: Add word count and read time statistics
	digestContent := content.String()
	wordCount := countWords(digestContent)
	if template.MaxDigestWords > 0 && wordCount > 0 {
		// Insert word count header after the title
		lines := strings.Split(digestContent, "\n")
		if len(lines) > 2 {
			wordCountHeader := generateWordCountHeader(wordCount)
			// Insert after the title (line 0) and empty line (line 1)
			newContent := strings.Join(lines[:2], "\n") + "\n" + wordCountHeader + strings.Join(lines[2:], "\n")
			digestContent = newContent
		}
	}

	// Write to file and return both content and path
	filePath, err := render.WriteDigestToFile(digestContent, outputDir, filename)
	return digestContent, filePath, err
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
		string(FormatScannableNewsletter),
		string(FormatEmail),
	}
}
