package trends

import (
	"briefly/internal/core"
	"fmt"
	"sort"
	"strings"
	"time"
)

// TrendMetric represents a specific metric being tracked over time
type TrendMetric struct {
	Name          string    `json:"name"`           // Metric name (e.g., "topic_frequency", "article_count")
	Value         float64   `json:"value"`          // Current value
	PreviousValue float64   `json:"previous_value"` // Previous period value
	Change        float64   `json:"change"`         // Absolute change
	ChangePercent float64   `json:"change_percent"` // Percentage change
	Period        string    `json:"period"`         // Time period (e.g., "week", "month")
	LastUpdated   time.Time `json:"last_updated"`   // When this metric was last calculated
}

// TrendReport contains analysis of trends across different time periods
type TrendReport struct {
	ID          string        `json:"id"`           // Unique identifier
	Period      string        `json:"period"`       // Analysis period (week-over-week, month-over-month)
	StartDate   time.Time     `json:"start_date"`   // Start of current period
	EndDate     time.Time     `json:"end_date"`     // End of current period
	Metrics     []TrendMetric `json:"metrics"`      // Collection of trend metrics
	TopicTrends []TopicTrend  `json:"topic_trends"` // Topic-specific trends
	KeyFindings []string      `json:"key_findings"` // Notable findings
	GeneratedAt time.Time     `json:"generated_at"` // When the report was generated
}

// TopicTrend represents trending information for a specific topic
type TopicTrend struct {
	Topic           string   `json:"topic"`             // Topic name
	CurrentCount    int      `json:"current_count"`     // Articles in current period
	PreviousCount   int      `json:"previous_count"`    // Articles in previous period
	Change          int      `json:"change"`            // Change in article count
	ChangePercent   float64  `json:"change_percent"`    // Percentage change
	Keywords        []string `json:"keywords"`          // Trending keywords for this topic
	IsNewTopic      bool     `json:"is_new_topic"`      // True if topic is new this period
	IsEmergingTopic bool     `json:"is_emerging_topic"` // True if showing significant growth
	Examples        []string `json:"examples"`          // Example article titles
}

// TrendAnalyzer handles trend analysis and reporting
type TrendAnalyzer struct {
	// Can be extended with configuration options
}

// NewTrendAnalyzer creates a new trend analyzer
func NewTrendAnalyzer() *TrendAnalyzer {
	return &TrendAnalyzer{}
}

// AnalyzeWeeklyTrends generates a week-over-week trend report
func (ta *TrendAnalyzer) AnalyzeWeeklyTrends(currentWeekDigests []core.Digest, previousWeekDigests []core.Digest) (*TrendReport, error) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -7) // One week ago

	report := &TrendReport{
		ID:          fmt.Sprintf("weekly-trend-%d", now.Unix()),
		Period:      "week-over-week",
		StartDate:   startDate,
		EndDate:     now,
		GeneratedAt: now,
	}

	// Analyze article count trends
	currentCount := len(currentWeekDigests)
	previousCount := len(previousWeekDigests)

	articleCountMetric := TrendMetric{
		Name:          "article_count",
		Value:         float64(currentCount),
		PreviousValue: float64(previousCount),
		Change:        float64(currentCount - previousCount),
		Period:        "week",
		LastUpdated:   now,
	}

	if previousCount > 0 {
		articleCountMetric.ChangePercent = (float64(currentCount-previousCount) / float64(previousCount)) * 100
	}

	report.Metrics = append(report.Metrics, articleCountMetric)

	// Analyze topic trends
	currentTopics := ta.extractTopicsFromDigests(currentWeekDigests)
	previousTopics := ta.extractTopicsFromDigests(previousWeekDigests)

	topicTrends := ta.compareTopicFrequencies(currentTopics, previousTopics)
	report.TopicTrends = topicTrends

	// Generate key findings
	report.KeyFindings = ta.generateKeyFindings(report.Metrics, report.TopicTrends)

	return report, nil
}

// AnalyzeMonthlyTrends generates a month-over-month trend report
func (ta *TrendAnalyzer) AnalyzeMonthlyTrends(currentMonthDigests []core.Digest, previousMonthDigests []core.Digest) (*TrendReport, error) {
	now := time.Now()
	startDate := now.AddDate(0, -1, 0) // One month ago

	report := &TrendReport{
		ID:          fmt.Sprintf("monthly-trend-%d", now.Unix()),
		Period:      "month-over-month",
		StartDate:   startDate,
		EndDate:     now,
		GeneratedAt: now,
	}

	// Similar analysis to weekly but with different time frames
	currentCount := len(currentMonthDigests)
	previousCount := len(previousMonthDigests)

	articleCountMetric := TrendMetric{
		Name:          "article_count",
		Value:         float64(currentCount),
		PreviousValue: float64(previousCount),
		Change:        float64(currentCount - previousCount),
		Period:        "month",
		LastUpdated:   now,
	}

	if previousCount > 0 {
		articleCountMetric.ChangePercent = (float64(currentCount-previousCount) / float64(previousCount)) * 100
	}

	report.Metrics = append(report.Metrics, articleCountMetric)

	// Analyze topic trends
	currentTopics := ta.extractTopicsFromDigests(currentMonthDigests)
	previousTopics := ta.extractTopicsFromDigests(previousMonthDigests)

	topicTrends := ta.compareTopicFrequencies(currentTopics, previousTopics)
	report.TopicTrends = topicTrends

	// Generate key findings
	report.KeyFindings = ta.generateKeyFindings(report.Metrics, report.TopicTrends)

	return report, nil
}

// AnalyzeArticleTrends generates trend analysis directly from articles
func (ta *TrendAnalyzer) AnalyzeArticleTrends(currentArticles []core.Article, previousArticles []core.Article) (*TrendReport, error) {
	now := time.Now()
	startDate := now.AddDate(0, 0, -7) // One week ago for default

	report := &TrendReport{
		ID:          fmt.Sprintf("article-trend-%d", now.Unix()),
		Period:      "article-comparison",
		StartDate:   startDate,
		EndDate:     now,
		GeneratedAt: now,
	}

	// Analyze article count trends
	currentCount := len(currentArticles)
	previousCount := len(previousArticles)

	articleCountMetric := TrendMetric{
		Name:          "article_count",
		Value:         float64(currentCount),
		PreviousValue: float64(previousCount),
		Change:        float64(currentCount - previousCount),
		Period:        "comparison",
		LastUpdated:   now,
	}

	if previousCount > 0 {
		articleCountMetric.ChangePercent = (float64(currentCount-previousCount) / float64(previousCount)) * 100
	}

	report.Metrics = append(report.Metrics, articleCountMetric)

	// Extract topics from articles using topic clusters and content
	currentTopics := ta.extractTopicsFromArticles(currentArticles)
	previousTopics := ta.extractTopicsFromArticles(previousArticles)

	topicTrends := ta.compareTopicFrequencies(currentTopics, previousTopics)
	report.TopicTrends = topicTrends

	// Generate key findings
	report.KeyFindings = ta.generateKeyFindings(report.Metrics, report.TopicTrends)

	return report, nil
}

// extractTopicsFromDigests extracts topics and their frequencies from digests
func (ta *TrendAnalyzer) extractTopicsFromDigests(digests []core.Digest) map[string]int {
	topics := make(map[string]int)

	for _, digest := range digests {
		// Extract topics from digest content - this is a simple implementation
		// In a more sophisticated version, this could use the topic clustering data
		content := strings.ToLower(digest.Content)

		// Simple keyword extraction (can be enhanced with NLP)
		keywords := ta.extractKeywords(content)
		for _, keyword := range keywords {
			topics[keyword]++
		}
	}

	return topics
}

// extractTopicsFromArticles extracts topics and their frequencies from articles
func (ta *TrendAnalyzer) extractTopicsFromArticles(articles []core.Article) map[string]int {
	topics := make(map[string]int)

	for _, article := range articles {
		// Use topic cluster if available
		if article.TopicCluster != "" {
			topics[article.TopicCluster]++
		} else {
			// Fallback to keyword extraction from content
			content := strings.ToLower(article.Title + " " + article.CleanedText)
			keywords := ta.extractKeywords(content)
			for _, keyword := range keywords {
				topics[keyword]++
			}
		}
	}

	return topics
}

// extractKeywords performs simple keyword extraction
func (ta *TrendAnalyzer) extractKeywords(content string) []string {
	// This is a simple implementation - in production, you'd want more sophisticated NLP
	commonTechKeywords := []string{
		"ai", "artificial intelligence", "machine learning", "ml", "llm", "gpt",
		"blockchain", "cryptocurrency", "bitcoin", "ethereum", "web3", "nft",
		"cloud", "aws", "azure", "google cloud", "kubernetes", "docker",
		"react", "javascript", "typescript", "python", "golang", "rust",
		"api", "microservices", "database", "sql", "nosql", "redis",
		"security", "cybersecurity", "privacy", "gdpr", "compliance",
		"startup", "funding", "venture capital", "ipo", "acquisition",
		"remote work", "productivity", "collaboration", "management",
	}

	var foundKeywords []string
	for _, keyword := range commonTechKeywords {
		if strings.Contains(content, keyword) {
			foundKeywords = append(foundKeywords, keyword)
		}
	}

	return foundKeywords
}

// compareTopicFrequencies compares topic frequencies between two periods
func (ta *TrendAnalyzer) compareTopicFrequencies(current, previous map[string]int) []TopicTrend {
	var trends []TopicTrend

	// Track all topics from both periods
	allTopics := make(map[string]bool)
	for topic := range current {
		allTopics[topic] = true
	}
	for topic := range previous {
		allTopics[topic] = true
	}

	// Calculate trends for each topic
	for topic := range allTopics {
		currentCount := current[topic]
		previousCount := previous[topic]

		trend := TopicTrend{
			Topic:         topic,
			CurrentCount:  currentCount,
			PreviousCount: previousCount,
			Change:        currentCount - previousCount,
			IsNewTopic:    previousCount == 0 && currentCount > 0,
		}

		if previousCount > 0 {
			trend.ChangePercent = (float64(currentCount-previousCount) / float64(previousCount)) * 100
			trend.IsEmergingTopic = trend.ChangePercent > 50 // 50% growth threshold
		} else if currentCount > 0 {
			trend.ChangePercent = 100 // New topic
			trend.IsEmergingTopic = true
		}

		// Only include topics with meaningful activity
		if currentCount > 0 || previousCount > 0 {
			trends = append(trends, trend)
		}
	}

	// Sort by change magnitude (descending)
	sort.Slice(trends, func(i, j int) bool {
		return abs(trends[i].Change) > abs(trends[j].Change)
	})

	return trends
}

// generateKeyFindings creates human-readable findings from the trend data
func (ta *TrendAnalyzer) generateKeyFindings(metrics []TrendMetric, topicTrends []TopicTrend) []string {
	var findings []string

	// Article count findings
	for _, metric := range metrics {
		if metric.Name == "article_count" {
			if metric.Change > 0 {
				findings = append(findings, fmt.Sprintf("Article volume increased by %d articles (%.1f%% growth)",
					int(metric.Change), metric.ChangePercent))
			} else if metric.Change < 0 {
				findings = append(findings, fmt.Sprintf("Article volume decreased by %d articles (%.1f%% decline)",
					int(-metric.Change), -metric.ChangePercent))
			} else {
				findings = append(findings, "Article volume remained stable")
			}
		}
	}

	// Topic findings
	var emergingTopics []string
	var decliningTopics []string
	var newTopics []string

	for _, trend := range topicTrends {
		if trend.IsNewTopic {
			newTopics = append(newTopics, trend.Topic)
		} else if trend.IsEmergingTopic && trend.Change > 0 {
			emergingTopics = append(emergingTopics, trend.Topic)
		} else if trend.Change < -2 { // Significant decline
			decliningTopics = append(decliningTopics, trend.Topic)
		}
	}

	if len(newTopics) > 0 {
		findings = append(findings, fmt.Sprintf("New topics emerged: %s", strings.Join(newTopics[:min(len(newTopics), 3)], ", ")))
	}

	if len(emergingTopics) > 0 {
		findings = append(findings, fmt.Sprintf("Trending topics: %s", strings.Join(emergingTopics[:min(len(emergingTopics), 3)], ", ")))
	}

	if len(decliningTopics) > 0 {
		findings = append(findings, fmt.Sprintf("Declining topics: %s", strings.Join(decliningTopics[:min(len(decliningTopics), 3)], ", ")))
	}

	if len(findings) == 0 {
		findings = append(findings, "No significant trends detected in this period")
	}

	return findings
}

// FormatReport generates a human-readable report
func (ta *TrendAnalyzer) FormatReport(report *TrendReport) string {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("# Trend Analysis Report\n"))
	builder.WriteString(fmt.Sprintf("**Period:** %s\n", report.Period))
	builder.WriteString(fmt.Sprintf("**Analysis Window:** %s to %s\n",
		report.StartDate.Format("2006-01-02"), report.EndDate.Format("2006-01-02")))
	builder.WriteString(fmt.Sprintf("**Generated:** %s\n\n", report.GeneratedAt.Format("2006-01-02 15:04")))

	// Key findings
	builder.WriteString("## Key Findings\n")
	for _, finding := range report.KeyFindings {
		builder.WriteString(fmt.Sprintf("- %s\n", finding))
	}
	builder.WriteString("\n")

	// Metrics
	builder.WriteString("## Metrics\n")
	for _, metric := range report.Metrics {
		changeIndicator := "→"
		if metric.Change > 0 {
			changeIndicator = "↗"
		} else if metric.Change < 0 {
			changeIndicator = "↘"
		}

		builder.WriteString(fmt.Sprintf("- **%s**: %.0f %s %.0f (%.1f%%)\n",
			strings.Title(strings.ReplaceAll(metric.Name, "_", " ")),
			metric.PreviousValue, changeIndicator, metric.Value, metric.ChangePercent))
	}
	builder.WriteString("\n")

	// Topic trends
	builder.WriteString("## Topic Trends\n")
	for i, trend := range report.TopicTrends {
		if i >= 10 { // Limit to top 10 trends
			break
		}

		status := ""
		if trend.IsNewTopic {
			status = " (New)"
		} else if trend.IsEmergingTopic {
			status = " (Trending)"
		}

		changeIndicator := "→"
		if trend.Change > 0 {
			changeIndicator = "↗"
		} else if trend.Change < 0 {
			changeIndicator = "↘"
		}

		builder.WriteString(fmt.Sprintf("- **%s**%s: %d %s %d articles",
			strings.Title(trend.Topic), status, trend.PreviousCount, changeIndicator, trend.CurrentCount))

		if trend.ChangePercent != 0 {
			builder.WriteString(fmt.Sprintf(" (%.1f%%)", trend.ChangePercent))
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

// Helper functions
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
