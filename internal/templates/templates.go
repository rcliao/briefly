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
	// FormatSignal creates Signal+Sources format with concise insights
	FormatSignal DigestFormat = "signal"
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

	// LinkedIn optimization fields
	IncludeLinkedInHook     bool   // Whether to include LinkedIn hook at top
	IncludeGameChanger      bool   // Whether to include "This Week's Game-Changer" section
	IncludeDiscussionPrompt bool   // Whether to include engagement question at end
	IncludeTryThisWeek      bool   // Whether to include actionable items section
	LinkedInHookPattern     string // Pattern for hook generation (Pattern1, Pattern2, Pattern3)
	GameChangerFormat       string // Format template for game-changer section
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
			IncludeDiscussionPrompt:   true,  // Enable discussion prompt for engagement
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
			IncludeDiscussionPrompt:   true,  // Enable discussion prompt for engagement
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
			IncludeDiscussionPrompt:   true,  // Enable discussion prompt for engagement
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
			IncludeDiscussionPrompt:   true, // Enable discussion prompt for engagement
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
			IncludeDiscussionPrompt:   true,  // Enable discussion prompt for LinkedIn engagement
			IncludeLinkedInHook:       true,  // Enable LinkedIn hook for engagement
			IncludeGameChanger:        true,  // Enable Game-Changer section for LinkedIn
			MaxSummaryLength:          22,    // Strict 20-25 word range (target 22 words)
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
			IncludeDiscussionPrompt:   true, // Enable discussion prompt for engagement
			MaxSummaryLength:          25,   // v2.0: 15-25 words per article summary
			MaxDigestWords:            400,  // v2.0: 400-word target for email format
			IntroductionText:          "Here's your personalized digest with today's most important insights:",
			ConclusionText:            "Stay informed and keep exploring!",
			SectionSeparator:          "\n\n---\n\n",
		}
	case FormatSignal:
		return &DigestTemplate{
			Format:                    FormatSignal,
			Title:                     "Signal Digest",
			IncludeSummaries:          false, // Signal format uses structured insights
			IncludeKeyInsights:        true,
			IncludeActionItems:        true,
			IncludeSourceLinks:        true,
			IncludePromptCorner:       false,
			IncludeIndividualArticles: false, // Signal format groups sources differently
			IncludeTopicClustering:    true,  // Essential for Signal+Sources format
			IncludeBanner:             false, // Keep Signal format clean and focused
			IncludeDiscussionPrompt:   false, // Signal format has action items instead
			MaxSummaryLength:          0,     // No traditional summaries in Signal format
			MaxDigestWords:            80,    // Signal insight should be 60-80 words
			IntroductionText:          "",    // Signal format has its own structure
			ConclusionText:            "",    // Signal format uses action items for conclusion
			SectionSeparator:          "\n\n",
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
// truncateToCompleteSentence was removed as unused

// truncateToScannableFormat provides strict 20-25 word enforcement for scannable format
// Prioritizes readability while enforcing word limits more strictly than truncateToCompleteSentence
func truncateToScannableFormat(text string, minWords, maxWords int) string {
	if maxWords <= 0 {
		return text
	}

	words := strings.Fields(text)
	wordCount := len(words)

	// If already within range, return as-is
	if wordCount >= minWords && wordCount <= maxWords {
		return text
	}

	// If too short, return as-is (can't artificially expand)
	if wordCount < minWords {
		return text
	}

	// If too long, try to find optimal truncation point
	// First attempt: look for complete sentence within range
	for i := maxWords; i >= minWords; i-- {
		candidate := strings.Join(words[:i], " ")

		// Check if this ends with proper punctuation
		if strings.HasSuffix(candidate, ".") || strings.HasSuffix(candidate, "!") || strings.HasSuffix(candidate, "?") {
			return candidate
		}
	}

	// Second attempt: look for clause boundaries (commas, semicolons)
	for i := maxWords; i >= minWords; i-- {
		candidate := strings.Join(words[:i], " ")

		if strings.HasSuffix(candidate, ",") || strings.HasSuffix(candidate, ";") {
			// Remove trailing comma/semicolon and add period
			return strings.TrimSuffix(strings.TrimSuffix(candidate, ","), ";") + "."
		}
	}

	// Final fallback: hard truncate at maxWords with proper ending
	truncated := strings.Join(words[:maxWords], " ")

	// Remove any trailing punctuation and add ellipsis if we cut mid-sentence
	truncated = strings.TrimRight(truncated, ".,;:!?")

	// If the next word (if exists) would complete a thought, hint at continuation
	if wordCount > maxWords {
		return truncated + "..."
	}

	return truncated + "."
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
					content.WriteString(fmt.Sprintf("%s\n\n", formatScannableLink(item.URL)))
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
	categoryOrder := []string{
		"🔥 Breaking & Hot",
		"🚀 Product Updates",
		"🤖 AI & Machine Learning",
		"🔒 Security & Privacy",
		"🛠️ Dev Tools & Techniques",
		"☁️ Infrastructure & Cloud",
		"📊 Research & Analysis",
		"💡 Ideas & Inspiration",
		"🔍 Worth Monitoring",
	}

	for _, item := range digestItems {
		if item.MyTake != "" {
			// Try to extract category from MyTake (format: "🔥 Breaking & Hot | insight" or "🔥 Breaking & Hot")
			parts := strings.Split(item.MyTake, " | ")
			if len(parts) >= 1 {
				categoryName := strings.TrimSpace(parts[0])
				if categoryName != "" {
					// Check if this matches any of our expected categories
					for _, expectedCategory := range categoryOrder {
						if categoryName == expectedCategory ||
							strings.HasSuffix(expectedCategory, strings.TrimSpace(strings.TrimLeft(categoryName, "🔥🚀🤖🔒🛠☁️📊💡🔍"))) {
							categorized = true
							categoryGroups[expectedCategory] = append(categoryGroups[expectedCategory], item)
							break
						}
					}
				}
			}
		}
	}

	if categorized {
		content.WriteString("## 📖 Featured Articles\n\n")

		// Track article numbering across categories
		articleNum := 1

		// Render articles grouped by category in priority order
		for _, categoryName := range categoryOrder {
			if articles, exists := categoryGroups[categoryName]; exists && len(articles) > 0 {
				content.WriteString(fmt.Sprintf("### %s\n\n", categoryName))

				// Sort articles within category by priority score
				sortedArticles := SortByPriority(articles)

				for i, item := range sortedArticles {
					if i > 0 {
						content.WriteString("\n")
					}

					// Get content type emoji
					contentEmoji := getContentTypeEmoji(item.ContentType, item.Title)

					// Article title with number and content type emoji
					content.WriteString(fmt.Sprintf("**[%d] %s %s**\n\n", articleNum, contentEmoji, item.Title))
					articleNum++

					// Key insight (summary) - simplified to just the summary
					if template.IncludeSummaries && item.SummaryText != "" {
						summary := item.SummaryText
						if template.MaxSummaryLength > 0 {
							// Use strict word enforcement for scannable format (20-25 words)
							summary = truncateToScannableFormat(summary, 20, template.MaxSummaryLength)
						}
						content.WriteString(fmt.Sprintf("%s\n\n", summary))
					}

					// Link with clear call to action and reference URL
					if template.IncludeSourceLinks {
						content.WriteString(fmt.Sprintf("%s\n", formatScannableLink(item.URL)))
						content.WriteString(fmt.Sprintf("*Reference: %s*\n\n", item.URL))
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
				content.WriteString(fmt.Sprintf("**[%d] %s %s**\n\n", articleNum, contentEmoji, item.Title))
				articleNum++

				if template.IncludeSummaries && item.SummaryText != "" {
					summary := item.SummaryText
					if template.MaxSummaryLength > 0 {
						// Use strict word enforcement for scannable format (20-25 words)
						summary = truncateToScannableFormat(summary, 20, template.MaxSummaryLength)
					}
					content.WriteString(fmt.Sprintf("%s\n\n", summary))
				}

				if template.IncludeSourceLinks {
					content.WriteString(fmt.Sprintf("%s\n", formatScannableLink(item.URL)))
					content.WriteString(fmt.Sprintf("*Reference: %s*\n\n", item.URL))
				}
			}
		}
	} else {
		// Fallback to original format if not categorized
		content.WriteString("## 📖 Featured Articles\n\n")

		// Sort all articles by priority score when not categorized
		sortedItems := SortByPriority(digestItems)

		for i, item := range sortedItems {
			if i > 0 {
				content.WriteString("\n")
			}

			// Get content type emoji
			contentEmoji := getContentTypeEmoji(item.ContentType, item.Title)

			// Article title with number and content type emoji
			content.WriteString(fmt.Sprintf("### [%d] %s %s\n\n", i+1, contentEmoji, item.Title))

			// Key insight (summary) - simplified to just the summary
			if template.IncludeSummaries && item.SummaryText != "" {
				summary := item.SummaryText
				if template.MaxSummaryLength > 0 {
					// Use strict word enforcement for scannable format (20-25 words)
					summary = truncateToScannableFormat(summary, 20, template.MaxSummaryLength)
				}
				content.WriteString(fmt.Sprintf("%s\n\n", summary))
			}

			// Link with clear call to action and reference URL
			if template.IncludeSourceLinks {
				content.WriteString(fmt.Sprintf("%s\n", formatScannableLink(item.URL)))
				content.WriteString(fmt.Sprintf("*Reference: %s*\n\n", item.URL))
			}
		}
	}

	return content.String()
}

// calculatePriorityScore computes a priority score for article ordering in scannable format
// Score is based on multiple factors: sentiment, alerts, topic confidence, content type, etc.
func calculatePriorityScore(item render.DigestData) float64 {
	score := 0.5 // Base score

	// Alert factor (highest priority) - articles that triggered alerts are most important
	if item.AlertTriggered {
		score += 0.3
	}

	// Sentiment factor - positive articles get slight boost, negative get attention too
	sentimentBoost := 0.0
	switch item.SentimentLabel {
	case "positive":
		sentimentBoost = 0.1
	case "negative":
		sentimentBoost = 0.05 // Negative news can be important too
	case "neutral":
		sentimentBoost = 0.0
	}
	score += sentimentBoost

	// Topic confidence factor - higher confidence means better categorization
	if item.TopicConfidence > 0 {
		score += item.TopicConfidence * 0.15 // Scale confidence (0-1) to contribute up to 0.15
	}

	// Content type factor - certain content types are more valuable
	switch item.ContentType {
	case "pdf": // Research papers, whitepapers are high value
		score += 0.1
	case "youtube": // Video content often has unique insights
		score += 0.05
	default: // Web articles
		score += 0.0
	}

	// Title length factor - longer titles often indicate more substantial content
	titleWords := len(strings.Fields(item.Title))
	if titleWords >= 8 { // Substantial titles
		score += 0.05
	} else if titleWords <= 3 { // Very short titles might be less informative
		score -= 0.05
	}

	// Research queries factor - articles that generated more research queries are more interesting
	if len(item.ResearchQueries) > 0 {
		// Scale based on number of research queries (max 5)
		queryBoost := float64(len(item.ResearchQueries)) * 0.02
		if queryBoost > 0.1 {
			queryBoost = 0.1
		}
		score += queryBoost
	}

	// Ensure score stays within bounds [0.0, 1.0]
	if score > 1.0 {
		score = 1.0
	} else if score < 0.0 {
		score = 0.0
	}

	return score
}

// SortByPriority sorts DigestData items by priority score (highest first)
func SortByPriority(items []render.DigestData) []render.DigestData {
	// Calculate priority scores if not already set
	for i := range items {
		if items[i].PriorityScore == 0.0 { // Only calculate if not already set
			items[i].PriorityScore = calculatePriorityScore(items[i])
		}
	}

	// Create a copy to avoid modifying the original slice
	sorted := make([]render.DigestData, len(items))
	copy(sorted, items)

	// Simple bubble sort by priority score (descending)
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].PriorityScore < sorted[j].PriorityScore {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}

// generateLinkedInHook creates an engaging 2-3 line hook for LinkedIn posts
func generateLinkedInHook(digestItems []render.DigestData, pattern string) string {
	if len(digestItems) == 0 {
		return "This week's tech highlights worth your attention 👇"
	}

	// Sort by priority to get the most impactful items
	sortedItems := SortByPriority(digestItems)
	topItem := sortedItems[0]

	// Extract key themes from top articles
	var themes []string
	for i, item := range sortedItems {
		if i >= 3 { // Only look at top 3 for themes
			break
		}

		// Extract technology names and themes from titles
		title := strings.ToLower(item.Title)
		if strings.Contains(title, "ai") || strings.Contains(title, "artificial intelligence") {
			themes = append(themes, "AI")
		}
		if strings.Contains(title, "gemini") || strings.Contains(title, "claude") || strings.Contains(title, "gpt") {
			themes = append(themes, "LLM development")
		}
		if strings.Contains(title, "coding") || strings.Contains(title, "code") || strings.Contains(title, "developer") {
			themes = append(themes, "coding tools")
		}
		if strings.Contains(title, "google") || strings.Contains(title, "anthropic") || strings.Contains(title, "openai") {
			themes = append(themes, "big tech moves")
		}
	}

	// Generate hook based on pattern
	switch pattern {
	case "Pattern1":
		// "X happened this week that changes Y"
		if len(themes) > 0 {
			return fmt.Sprintf("%s breakthroughs this week that change how engineers work.\n\n%s just made complex tasks instant. Here's what you need to know 👇",
				themes[0], extractProductName(topItem.Title))
		}
		return fmt.Sprintf("%s just changed the game for engineering teams.\n\nHere's what happened this week that you can't ignore 👇",
			extractProductName(topItem.Title))

	case "Pattern2":
		// "The gap between [leader] and [followers] just widened"
		leader := extractCompanyName(topItem.Title)
		if leader != "" {
			return fmt.Sprintf("The gap between %s and everyone else just widened.\n\nWhile competitors catch up, here's what engineering leaders are already using 👇", leader)
		}
		return fmt.Sprintf("The gap between AI leaders and followers just widened.\n\n%s is proof. Here's this week's developments 👇",
			extractProductName(topItem.Title))

	case "Pattern3":
		// "While everyone talks about X, Y quietly shipped Z"
		if len(sortedItems) >= 2 {
			return fmt.Sprintf("While everyone talks about %s, %s quietly shipped game-changing updates.\n\nHere's what you missed this week 👇",
				extractTopic(sortedItems[1].Title), extractProductName(topItem.Title))
		}
		return "While everyone talks about AI hype, real engineering breakthroughs happened this week.\n\nHere's what actually matters 👇"

	default:
		// Default engaging hook
		return fmt.Sprintf("%s developments this week that engineering teams are already using.\n\nFrom %s to practical applications - here's what you need to know 👇",
			themes[0], extractProductName(topItem.Title))
	}
}

// extractProductName extracts product name from article title for hooks
func extractProductName(title string) string {
	title = strings.ToLower(title)

	// Common product patterns
	products := []string{
		"gemini", "claude", "chatgpt", "copilot", "jules",
		"deep think", "opus", "gpt-4", "llama", "midjourney",
	}

	for _, product := range products {
		if strings.Contains(title, product) {
			// Capitalize first letter
			return strings.ToUpper(string(product[0])) + product[1:]
		}
	}

	// Fall back to first significant word
	words := strings.Fields(title)
	for _, word := range words {
		if len(word) > 3 && !isStopWord(word) {
			return strings.ToUpper(string(word[0])) + word[1:]
		}
	}

	return "AI tools"
}

// extractCompanyName extracts company name from article title for hooks
func extractCompanyName(title string) string {
	title = strings.ToLower(title)

	companies := []string{
		"google", "anthropic", "openai", "microsoft", "meta",
		"apple", "amazon", "nvidia", "deepmind",
	}

	for _, company := range companies {
		if strings.Contains(title, company) {
			return strings.ToUpper(string(company[0])) + company[1:]
		}
	}

	return ""
}

// extractTopic extracts general topic from title for contrast hooks
func extractTopic(title string) string {
	title = strings.ToLower(title)

	if strings.Contains(title, "ai") || strings.Contains(title, "artificial intelligence") {
		return "AI"
	}
	if strings.Contains(title, "code") || strings.Contains(title, "coding") || strings.Contains(title, "programming") {
		return "coding"
	}
	if strings.Contains(title, "cloud") || strings.Contains(title, "infrastructure") {
		return "cloud infrastructure"
	}

	return "tech developments"
}

// isStopWord checks if word should be ignored in product name extraction
func isStopWord(word string) bool {
	stopWords := []string{
		"the", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for",
		"of", "with", "by", "is", "are", "was", "were", "has", "have", "had",
		"now", "new", "just", "here", "this", "that", "your", "you", "can",
	}

	word = strings.ToLower(word)
	for _, stopWord := range stopWords {
		if word == stopWord {
			return true
		}
	}
	return false
}

// extractDomainFromURL extracts the domain name from a URL for source attribution
func extractDomainFromURL(url string) string {
	// Remove protocol
	domain := strings.TrimPrefix(url, "https://")
	domain = strings.TrimPrefix(domain, "http://")

	// Remove www prefix
	domain = strings.TrimPrefix(domain, "www.")

	// Extract domain before first slash
	if slashIndex := strings.Index(domain, "/"); slashIndex != -1 {
		domain = domain[:slashIndex]
	}

	// Clean up common subdomains for better readability
	if strings.HasPrefix(domain, "blog.") {
		domain = strings.TrimPrefix(domain, "blog.")
	} else if strings.HasPrefix(domain, "docs.") {
		domain = strings.TrimPrefix(domain, "docs.")
	}

	return domain
}

// selectGameChanger identifies the most impactful article for "This Week's Game-Changer" section
func selectGameChanger(digestItems []render.DigestData) *render.DigestData {
	if len(digestItems) == 0 {
		return nil
	}

	// Sort by priority to get the most impactful items
	sortedItems := SortByPriority(digestItems)

	// Look for the highest impact item that's not just breaking news
	for _, item := range sortedItems {
		// Skip items with generic or unclear summaries
		if strings.Contains(strings.ToLower(item.SummaryText), "no identifiable information") ||
			len(item.SummaryText) < 50 {
			continue
		}

		// Prefer items with clear practical impact
		title := strings.ToLower(item.Title)
		summary := strings.ToLower(item.SummaryText)

		// Boost score for items with practical impact keywords
		impactScore := item.PriorityScore

		if strings.Contains(title, "launch") || strings.Contains(title, "available") ||
			strings.Contains(title, "release") || strings.Contains(summary, "now available") {
			impactScore += 0.2 // Production-ready tools
		}

		if strings.Contains(summary, "developer") || strings.Contains(summary, "coding") ||
			strings.Contains(summary, "programming") || strings.Contains(summary, "engineer") {
			impactScore += 0.15 // Direct relevance to engineers
		}

		if item.AlertTriggered {
			impactScore += 0.1 // Alert-worthy developments
		}

		// Update the item's priority score for comparison
		item.PriorityScore = impactScore

		// Return the first high-impact item
		if impactScore > 0.6 {
			return &item
		}
	}

	// Fallback to highest priority item
	return &sortedItems[0]
}

// formatGameChanger formats the game-changer section with winner details
func formatGameChanger(gameChanger *render.DigestData) string {
	if gameChanger == nil {
		return ""
	}

	var content strings.Builder
	content.WriteString("## 🔥 This Week's Game-Changer\n\n")

	// Winner - use the actual article title (shortened if needed)
	winnerTitle := gameChanger.Title
	if len(winnerTitle) > 60 {
		// Truncate long titles but preserve meaning
		winnerTitle = winnerTitle[:57] + "..."
	}
	content.WriteString(fmt.Sprintf("**Winner:** %s\n\n", winnerTitle))

	// Why It Matters - extract key benefit from summary
	whyItMatters := generateWhyItMatters(gameChanger.SummaryText)
	content.WriteString(fmt.Sprintf("**Why It Matters:** %s\n\n", whyItMatters))

	// Try It - generate actionable suggestion
	tryIt := generateTryItSuggestion(gameChanger.Title, gameChanger.SummaryText)
	content.WriteString(fmt.Sprintf("**Try It:** %s\n\n", tryIt))

	// Reality Check - add balanced perspective
	realityCheck := generateRealityCheck(gameChanger.SummaryText)
	content.WriteString(fmt.Sprintf("**Reality Check:** %s\n\n", realityCheck))

	// User Take - add personal commentary if available
	if gameChanger.UserTakeText != "" {
		content.WriteString(fmt.Sprintf("**Your Take:** %s\n\n", gameChanger.UserTakeText))
	}

	return content.String()
}

// generateWhyItMatters extracts the key benefit for engineers from article summary
func generateWhyItMatters(summary string) string {
	summary = strings.ToLower(summary)

	// Look for key impact phrases
	if strings.Contains(summary, "parallel thinking") || strings.Contains(summary, "complex") {
		return "Parallel thinking for complex code = junior dev tasks automated"
	}
	if strings.Contains(summary, "coding") && strings.Contains(summary, "assistant") {
		return "AI coding assistance reaches production-ready quality"
	}
	if strings.Contains(summary, "beta") || strings.Contains(summary, "available") {
		return "Production-ready tool for immediate team adoption"
	}
	if strings.Contains(summary, "improve") && (strings.Contains(summary, "coding") || strings.Contains(summary, "development")) {
		return "Significant productivity gains for development workflows"
	}
	if strings.Contains(summary, "breakthrough") || strings.Contains(summary, "advance") {
		return "Major capability leap that changes what's possible"
	}

	// Fallback to generic but engaging message
	return "Game-changing capability that engineering teams can use today"
}

// generateTryItSuggestion creates actionable advice for testing the tool/feature
func generateTryItSuggestion(title, summary string) string {
	title = strings.ToLower(title)
	summary = strings.ToLower(summary)

	if strings.Contains(title, "deep think") || strings.Contains(summary, "complex") {
		return "Upload your gnarliest legacy codebase section and ask for refactoring suggestions"
	}
	if strings.Contains(title, "jules") || strings.Contains(summary, "coding agent") {
		return "Connect to your GitHub and let it handle a simple PR workflow"
	}
	if strings.Contains(title, "claude") && strings.Contains(summary, "coding") {
		return "Test it on a complex debugging session with multi-file context"
	}
	if strings.Contains(title, "copilot") || strings.Contains(summary, "assistant") {
		return "Try it on your most challenging algorithm implementation this week"
	}
	if strings.Contains(summary, "text-to-speech") || strings.Contains(summary, "voice") {
		return "Generate voice narration for your technical documentation"
	}

	// Extract product name and create generic suggestion
	product := extractProductName(title)
	return fmt.Sprintf("Test %s on your current project's biggest technical challenge", product)
}

// generateRealityCheck provides balanced perspective on limitations
func generateRealityCheck(summary string) string {
	summary = strings.ToLower(summary)

	if strings.Contains(summary, "beta") || strings.Contains(summary, "preview") {
		return "Still in beta - expect some rough edges but worth exploring"
	}
	if strings.Contains(summary, "ultra") || strings.Contains(summary, "subscription") {
		return "Requires premium subscription - evaluate ROI for your team first"
	}
	if strings.Contains(summary, "agent") || strings.Contains(summary, "autonomous") {
		return "Great for exploration, still needs human validation for production"
	}
	if strings.Contains(summary, "improvement") || strings.Contains(summary, "upgrade") {
		return "Incremental improvement - valuable but not revolutionary"
	}

	// Default balanced view
	return "Promising technology - test thoroughly before production deployment"
}

// generateDiscussionPrompt creates an engaging question to drive LinkedIn engagement
func generateDiscussionPrompt(digestItems []render.DigestData) string {
	if len(digestItems) == 0 {
		return "What's the biggest AI development you're testing in your workflow this week? Share your experience below 👇"
	}

	// Sort by priority to get the most discussion-worthy items
	sortedItems := SortByPriority(digestItems)

	// Look for controversial, interesting, or practical topics
	for _, item := range sortedItems {
		title := strings.ToLower(item.Title)
		summary := strings.ToLower(item.SummaryText)

		// Agent/automation discussions
		if strings.Contains(title, "agent") || strings.Contains(summary, "autonomous") {
			product := extractProductName(item.Title)
			return fmt.Sprintf("%s claims to handle entire workflows autonomously.\n\nWho's already using AI agents for real work? What's working and what still needs human oversight?", product)
		}

		// Coding tools discussions
		if strings.Contains(summary, "coding") || strings.Contains(summary, "developer") {
			return "AI coding tools are getting impressive results in demos.\n\nWhat's your experience with them on real projects? Where do they excel vs. fall short?"
		}

		// Performance/speed claims
		if strings.Contains(summary, "10x") || strings.Contains(summary, "faster") || strings.Contains(summary, "speed") {
			return "Another week, another \"10x faster\" AI tool claim.\n\nWhich tools have actually made your team measurably more productive? Looking for real examples."
		}

		// Beta/new releases
		if strings.Contains(summary, "beta") || strings.Contains(summary, "launch") {
			product := extractProductName(item.Title)
			return fmt.Sprintf("%s just launched publicly after beta testing.\n\nWho tried it during beta? How does it compare to alternatives for your use cases?", product)
		}

		// Subscription/pricing model
		if strings.Contains(summary, "subscription") || strings.Contains(summary, "premium") {
			return "More AI tools moving to premium tiers and higher pricing.\n\nHow do you evaluate ROI on AI subscriptions for your team? What's your decision framework?"
		}
	}

	// Topic-based fallbacks
	themes := extractTopicThemes(sortedItems)
	if len(themes) > 0 {
		switch themes[0] {
		case "AI":
			return "AI capabilities are advancing rapidly, but adoption varies widely.\n\nWhat's the biggest gap between AI demos and production reality in your experience?"
		case "coding tools":
			return "Engineering teams are experimenting with more AI coding assistants.\n\nWhich tools have stuck in your workflow vs. which were just hype? Why?"
		case "big tech moves":
			return "Big tech companies are racing to ship AI features.\n\nWhich company's AI strategy do you think will win for enterprise adoption? Why?"
		default:
			return "This week brought several interesting tech developments.\n\nWhich one are you most likely to try with your team? What's your evaluation process?"
		}
	}

	// Default engaging question
	return "Another week of rapid AI development across the industry.\n\nWhat's the most interesting tool or development you're considering for your workflow? Why that one?"
}

// extractTopicThemes extracts common themes from top articles for discussion prompts
func extractTopicThemes(digestItems []render.DigestData) []string {
	var themes []string
	themeCount := make(map[string]int)

	for i, item := range digestItems {
		if i >= 5 { // Only check top 5 articles
			break
		}

		title := strings.ToLower(item.Title)
		summary := strings.ToLower(item.SummaryText)

		if strings.Contains(title, "ai") || strings.Contains(summary, "artificial intelligence") {
			themeCount["AI"]++
		}
		if strings.Contains(title, "code") || strings.Contains(summary, "coding") || strings.Contains(summary, "developer") {
			themeCount["coding tools"]++
		}
		if strings.Contains(title, "google") || strings.Contains(title, "anthropic") || strings.Contains(title, "openai") {
			themeCount["big tech moves"]++
		}
		if strings.Contains(title, "agent") || strings.Contains(summary, "autonomous") {
			themeCount["AI agents"]++
		}
	}

	// Sort themes by frequency
	type themeFreq struct {
		theme string
		count int
	}

	var sortedThemes []themeFreq
	for theme, count := range themeCount {
		sortedThemes = append(sortedThemes, themeFreq{theme, count})
	}

	// Simple sort by count (descending)
	for i := 0; i < len(sortedThemes)-1; i++ {
		for j := i + 1; j < len(sortedThemes); j++ {
			if sortedThemes[i].count < sortedThemes[j].count {
				sortedThemes[i], sortedThemes[j] = sortedThemes[j], sortedThemes[i]
			}
		}
	}

	// Extract theme names
	for _, tf := range sortedThemes {
		themes = append(themes, tf.theme)
	}

	return themes
}

// formatScannableLink formats links with source attribution for scannable format
func formatScannableLink(url string) string {
	domain := extractDomainFromURL(url)

	// Format with domain and subtle styling
	return fmt.Sprintf("🔗 [Read more](%s) *(%s)*", url, domain)
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

// capitalizeFirst capitalizes the first letter of a string
// capitalizeFirst removed as unused

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
// renderReferencesSection removed as unused

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

	// Discussion prompt section for LinkedIn engagement
	if template.IncludeDiscussionPrompt && len(digestItems) > 0 {
		content.WriteString("\n\n## 💭 Your Take?\n\n")
		content.WriteString(generateDiscussionPrompt(digestItems))
		content.WriteString("\n\n")
	}

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

	// References removed - now included in Featured Articles section with numbering

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
		// Allow longer descriptions for better comprehensiveness
		if len(cleaned) > 200 {
			return cleaned[:197] + "..."
		}
		return cleaned
	}
	// Allow longer single-sentence summaries
	if len(summary) > 200 {
		return summary[:197] + "..."
	}
	return summary
}

// extractCoreConceptDescription extracts technical concept from summary
func extractCoreConceptDescription(summary string) string {
	// Look for technical keywords and extract relevant sentences
	summary = strings.TrimSpace(summary)
	// Allow longer concept descriptions for better clarity
	if len(summary) > 250 {
		return summary[:247] + "..."
	}
	return summary
}

// extractImplementationDescription extracts what was built/solved from summary
func extractImplementationDescription(summary string) string {
	// Similar to extractCoreConceptDescription but focused on implementations
	summary = strings.TrimSpace(summary)
	// Allow longer implementation descriptions for better detail
	if len(summary) > 250 {
		return summary[:247] + "..."
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

	// References removed - now included in article listings with numbering

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

	// LinkedIn Hook (LinkedIn optimization)
	if template.IncludeLinkedInHook && len(digestItems) > 0 {
		linkedInHook := generateLinkedInHook(digestItems, "Pattern1")
		if linkedInHook != "" {
			content.WriteString(linkedInHook)
			content.WriteString("\n\n")
		}
	}

	// Final digest summary (if provided)
	if finalDigest != "" {
		content.WriteString("## Executive Summary\n\n")
		// v2.0: For scannable format, allow full executive summary without truncation
		// Other formats can have reasonable limits to maintain digestibility
		executiveSummary := finalDigest
		if template.MaxDigestWords > 0 && template.Format != FormatScannableNewsletter {
			// Allow generous executive summary length for most formats, but not scannable
			executiveSummary = truncateToWordLimit(finalDigest, 250) // Increased from 150 to 250 words
		}
		// For scannable format, use the complete executive summary without truncation
		content.WriteString(executiveSummary)
		content.WriteString("\n\n")
	}

	// Game-Changer section (LinkedIn optimization)
	if template.IncludeGameChanger && len(digestItems) > 0 {
		gameChanger := selectGameChanger(digestItems)
		if gameChanger != nil {
			content.WriteString(formatGameChanger(gameChanger))
			content.WriteString("---\n\n")
		}
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

	// Discussion prompt section for LinkedIn engagement
	if template.IncludeDiscussionPrompt && len(digestItems) > 0 {
		content.WriteString("\n\n## 💭 Your Take?\n\n")
		content.WriteString(generateDiscussionPrompt(digestItems))
		content.WriteString("\n\n")
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

	// References removed - now included in Featured Articles section with numbering

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
		string(FormatSignal),
	}
}

// RenderSignalStyleDigest renders a Signal-style digest using existing data structures
// This is a Phase 4 bridge function until full Signal+Sources implementation
func RenderSignalStyleDigest(digestItems []render.DigestData, outputDir string, finalDigest string, template *DigestTemplate, customTitle string) (string, string, error) {
	var content strings.Builder

	// Generate Signal-style title
	title := customTitle
	if title == "" {
		title = generateSignalStyleTitle(digestItems)
	}

	// Signal Header
	content.WriteString(fmt.Sprintf("# %s\n\n", title))
	content.WriteString(fmt.Sprintf("📊 %d sources • ⏱️ 2m read\n\n", len(digestItems)))

	// Signal Insight (60-80 words)
	signalInsight := generateSignalInsight(digestItems, finalDigest)
	content.WriteString("## 🔍 Signal\n\n")
	content.WriteString(signalInsight)
	content.WriteString("\n\n")

	// Implications
	implications := generateSignalImplications(digestItems)
	if len(implications) > 0 {
		content.WriteString("### 💡 Implications\n\n")
		for _, implication := range implications {
			content.WriteString(fmt.Sprintf("- %s\n", implication))
		}
		content.WriteString("\n")
	}

	// Action Items
	actions := generateSignalActions(digestItems)
	if len(actions) > 0 {
		content.WriteString("### 🎯 Actions\n\n")
		for _, action := range actions {
			content.WriteString(fmt.Sprintf("- **%s** (%s)\n", action.Description, action.Timeline))
		}
		content.WriteString("\n")
	}

	// Sources Section
	content.WriteString("## 📚 Sources\n\n")

	// Use items in the exact order they were passed - they are already ordered consistently with LLM input
	// Track categories we've already shown to avoid duplicate headers
	categoriesShown := make(map[string]bool)
	currentCategory := ""
	referenceNumber := 1

	for _, item := range digestItems {
		// Determine category for this item using the same logic as ordering
		category := extractCategoryFromItem(item)

		// Add category header only if we haven't shown this category yet, or if it's different from current
		if category != currentCategory {
			if !categoriesShown[category] {
				content.WriteString(fmt.Sprintf("### %s\n\n", category))
				categoriesShown[category] = true
				currentCategory = category
			}
		}

		// Render the item with its sequential reference number
		content.WriteString(fmt.Sprintf("**[%d] %s**\n", referenceNumber, item.Title))
		referenceNumber++

		// Use full summary for better comprehensiveness, only truncate if extremely long
		summaryText := item.SummaryText
		if len(strings.Fields(summaryText)) > 100 {
			summaryText = truncateToWords(summaryText, 100)
		}
		content.WriteString(fmt.Sprintf("%s\n\n", summaryText))
		content.WriteString(fmt.Sprintf("🔗 [Read more](%s)\n\n", item.URL))
	}

	// Footer
	content.WriteString("---\n\n")
	content.WriteString("*Generated using hybrid AI processing*\n")

	// Write to file
	filename := fmt.Sprintf("digest_signal_%s.md", time.Now().Format("2006-01-02"))
	filePath, err := render.WriteDigestToFile(content.String(), outputDir, filename)

	return content.String(), filePath, err
}

// Helper functions for Signal-style digest

func generateSignalStyleTitle(digestItems []render.DigestData) string {
	if len(digestItems) == 0 {
		return "Signal Digest"
	}

	// Extract key themes from article titles
	themes := extractSignalThemes(digestItems)
	if len(themes) > 0 {
		return fmt.Sprintf("Signal: %s", strings.Join(themes[:min(2, len(themes))], " & "))
	}

	return fmt.Sprintf("Signal: %d Key Developments", len(digestItems))
}

func generateSignalInsight(digestItems []render.DigestData, finalDigest string) string {
	if finalDigest != "" && len(finalDigest) > 100 {
		// Use full digest content for signal without truncation
		// Only ensure it's well-formatted
		words := strings.Fields(finalDigest)
		if len(words) > 150 {
			// Only truncate if extremely long (150+ words), but use sentence boundaries
			return strings.Join(words[:150], " ") + "."
		}
		return finalDigest
	}

	// Generate basic insight from article titles
	if len(digestItems) == 0 {
		return "No articles to analyze."
	}

	themes := extractSignalThemes(digestItems)
	insight := fmt.Sprintf("Today's developments highlight %d key areas", len(digestItems))

	if len(themes) > 0 {
		insight += fmt.Sprintf(" including %s", strings.Join(themes, ", "))
	}

	insight += ". These changes signal evolving priorities in technology adoption and strategic decision-making across the industry."

	return insight
}

func generateSignalImplications(digestItems []render.DigestData) []string {
	implications := []string{}

	if len(digestItems) > 3 {
		implications = append(implications, "Multiple concurrent developments suggest accelerating change")
	}

	// Check for AI-related content
	aiCount := 0
	for _, item := range digestItems {
		if strings.Contains(strings.ToLower(item.Title), "ai") || strings.Contains(strings.ToLower(item.SummaryText), "ai") {
			aiCount++
		}
	}

	if aiCount > 0 {
		implications = append(implications, "AI integration remains a key strategic priority")
	}

	// Default implications
	if len(implications) == 0 {
		implications = append(implications, "Technology landscape continues evolving rapidly")
	}

	return implications[:min(3, len(implications))]
}

func generateSignalActions(digestItems []render.DigestData) []SignalAction {
	actions := []SignalAction{}

	// Generate actions based on content
	if len(digestItems) > 0 {
		actions = append(actions, SignalAction{
			Description: "Review highlighted developments for strategic relevance",
			Timeline:    "this week",
		})
	}

	if len(digestItems) > 2 {
		actions = append(actions, SignalAction{
			Description: "Assess impact on current technology roadmap",
			Timeline:    "this month",
		})
	}

	return actions[:min(3, len(actions))]
}

func extractSignalThemes(digestItems []render.DigestData) []string {
	themes := make(map[string]int)

	for _, item := range digestItems {
		words := strings.Fields(strings.ToLower(item.Title))
		for _, word := range words {
			if len(word) > 4 && !isSignalStopWord(word) {
				themes[word]++
			}
		}
	}

	// Get most common themes
	var sortedThemes []string
	for theme, count := range themes {
		if count >= 1 {
			sortedThemes = append(sortedThemes, theme)
		}
	}

	return sortedThemes[:min(3, len(sortedThemes))]
}

func isSignalStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "from": true,
		"this": true, "that": true, "your": true, "you": true, "are": true,
		"how": true, "why": true, "what": true, "when": true, "where": true,
	}
	return stopWords[word]
}

// groupSignalItemsByCategory groups digest items by category for Signal format
// groupSignalItemsByCategory removed as unused

// extractCategoryFromItem extracts category from digest item
func extractCategoryFromItem(item render.DigestData) string {
	// Try to extract from MyTake first (which may contain category info)
	if item.MyTake != "" && strings.Contains(item.MyTake, "🔥") {
		return "🔥 Breaking & Hot"
	}
	if item.MyTake != "" && strings.Contains(item.MyTake, "🛠️") {
		return "🛠️ Tools & Platforms"
	}
	if item.MyTake != "" && strings.Contains(item.MyTake, "📊") {
		return "📊 Analysis & Research"
	}
	if item.MyTake != "" && strings.Contains(item.MyTake, "💰") {
		return "💰 Business & Economics"
	}

	// Fallback to keyword-based categorization
	title := strings.ToLower(item.Title)
	summary := strings.ToLower(item.SummaryText)

	if containsAny(title, []string{"breaking", "urgent", "announcement", "released", "launched"}) {
		return "🔥 Breaking & Hot"
	}
	if containsAny(title, []string{"tool", "github", "framework", "library", "sdk"}) ||
		containsAny(summary, []string{"developer", "coding", "programming"}) {
		return "🛠️ Tools & Platforms"
	}
	if containsAny(title, []string{"analysis", "research", "study", "report", "survey"}) {
		return "📊 Analysis & Research"
	}
	if containsAny(title, []string{"business", "market", "revenue", "funding", "cost"}) {
		return "💰 Business & Economics"
	}

	return "💡 Additional Items"
}

func truncateToWords(text string, maxWords int) string {
	words := strings.Fields(text)
	if len(words) <= maxWords {
		return text
	}

	// Try to find a sentence boundary near the word limit
	truncated := strings.Join(words[:maxWords], " ")

	// Look for the last complete sentence within the truncated text
	lastPeriod := strings.LastIndex(truncated, ". ")
	lastQuestion := strings.LastIndex(truncated, "? ")
	lastExclamation := strings.LastIndex(truncated, "! ")

	// Find the latest sentence ending
	lastSentence := lastPeriod
	if lastQuestion > lastSentence {
		lastSentence = lastQuestion
	}
	if lastExclamation > lastSentence {
		lastSentence = lastExclamation
	}

	// If we found a sentence boundary and it's at least 70% of the target length, use it
	if lastSentence > 0 && lastSentence > len(truncated)*7/10 {
		return truncated[:lastSentence+1]
	}

	// Otherwise, truncate at word boundary with ellipsis
	return truncated + "..."
}

// SignalAction represents an action item in Signal format
type SignalAction struct {
	Description string
	Timeline    string
}

// min returns the smaller of two integers (utility function)
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
