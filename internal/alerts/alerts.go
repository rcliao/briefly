package alerts

import (
	"briefly/internal/core"
	"fmt"
	"strings"
	"time"
)

// AlertLevel represents the severity of an alert
type AlertLevel int

const (
	AlertLevelInfo AlertLevel = iota
	AlertLevelWarning
	AlertLevelCritical
)

func (al AlertLevel) String() string {
	switch al {
	case AlertLevelInfo:
		return "INFO"
	case AlertLevelWarning:
		return "WARNING"
	case AlertLevelCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// AlertCondition defines a condition that can trigger an alert
type AlertCondition struct {
	ID          string                 `json:"id"`          // Unique identifier
	Name        string                 `json:"name"`        // Human-readable name
	Description string                 `json:"description"` // Description of what triggers this alert
	Type        AlertConditionType     `json:"type"`        // Type of condition
	Level       AlertLevel             `json:"level"`       // Alert severity level
	Enabled     bool                   `json:"enabled"`     // Whether this condition is active
	Config      map[string]interface{} `json:"config"`      // Configuration parameters
	CreatedAt   time.Time              `json:"created_at"`  // When the condition was created
}

// AlertConditionType defines different types of alert conditions
type AlertConditionType string

const (
	ConditionKeywordMatch         AlertConditionType = "keyword_match"
	ConditionTopicEmergence       AlertConditionType = "topic_emergence"
	ConditionVolumeChange         AlertConditionType = "volume_change"
	ConditionCostThreshold        AlertConditionType = "cost_threshold"
	ConditionSentimentShift       AlertConditionType = "sentiment_shift"
	ConditionSourceFailure        AlertConditionType = "source_failure"
	ConditionMarketImpact         AlertConditionType = "market_impact"
	ConditionRegulatoryPolicy     AlertConditionType = "regulatory_policy"
	ConditionLeadershipPersonnel  AlertConditionType = "leadership_personnel"
	ConditionTechnologyDisruption AlertConditionType = "technology_disruption"
	ConditionCompetitiveIntel     AlertConditionType = "competitive_intelligence"
	ConditionGeographicImpact     AlertConditionType = "geographic_impact"
	ConditionMultipleSource       AlertConditionType = "multiple_source"
	ConditionSentimentExtreme     AlertConditionType = "sentiment_extreme"
	ConditionTimeSensitive        AlertConditionType = "time_sensitive"
	ConditionIndustryDisruption   AlertConditionType = "industry_disruption"
)

// Alert represents a triggered alert
type Alert struct {
	ID             string                 `json:"id"`              // Unique identifier
	ConditionID    string                 `json:"condition_id"`    // ID of the condition that triggered this
	Level          AlertLevel             `json:"level"`           // Alert severity
	Title          string                 `json:"title"`           // Alert title
	Message        string                 `json:"message"`         // Detailed alert message
	Context        map[string]interface{} `json:"context"`         // Additional context data
	TriggeredAt    time.Time              `json:"triggered_at"`    // When the alert was triggered
	Acknowledged   bool                   `json:"acknowledged"`    // Whether the alert has been acknowledged
	AcknowledgedAt time.Time              `json:"acknowledged_at"` // When it was acknowledged
}

// AlertManager handles alert conditions and notifications
type AlertManager struct {
	conditions []AlertCondition
	alerts     []Alert
}

// NewAlertManager creates a new alert manager with default conditions
func NewAlertManager() *AlertManager {
	am := &AlertManager{
		conditions: []AlertCondition{},
		alerts:     []Alert{},
	}

	// Add default alert conditions
	am.addDefaultConditions()

	return am
}

// addDefaultConditions sets up useful default alert conditions
func (am *AlertManager) addDefaultConditions() {
	// High-priority keyword matches
	am.conditions = append(am.conditions, AlertCondition{
		ID:          "high-priority-keywords",
		Name:        "High Priority Keywords",
		Description: "Alert when articles mention critical business keywords",
		Type:        ConditionKeywordMatch,
		Level:       AlertLevelWarning,
		Enabled:     true,
		Config: map[string]interface{}{
			"keywords": []string{
				"security breach", "data leak", "vulnerability", "hack", "cyberattack",
				"outage", "downtime", "incident", "emergency", "critical",
				"acquisition", "merger", "ipo", "funding", "layoffs",
				"breaking", "urgent", "alert", "warning", "crisis",
			},
			"case_insensitive": true,
		},
		CreatedAt: time.Now(),
	})

	// Topic emergence detection
	am.conditions = append(am.conditions, AlertCondition{
		ID:          "new-topic-emergence",
		Name:        "New Topic Emergence",
		Description: "Alert when a completely new topic appears in articles",
		Type:        ConditionTopicEmergence,
		Level:       AlertLevelInfo,
		Enabled:     true,
		Config: map[string]interface{}{
			"min_articles": 3, // Minimum articles to consider a topic significant
		},
		CreatedAt: time.Now(),
	})

	// Volume change detection
	am.conditions = append(am.conditions, AlertCondition{
		ID:          "volume-spike",
		Name:        "Article Volume Spike",
		Description: "Alert when article volume increases significantly",
		Type:        ConditionVolumeChange,
		Level:       AlertLevelInfo,
		Enabled:     true,
		Config: map[string]interface{}{
			"threshold_percent": 50.0, // 50% increase triggers alert
		},
		CreatedAt: time.Now(),
	})

	// Cost threshold (disabled for reader-focused digests)
	am.conditions = append(am.conditions, AlertCondition{
		ID:          "cost-threshold",
		Name:        "Cost Threshold",
		Description: "Alert when estimated processing costs exceed threshold",
		Type:        ConditionCostThreshold,
		Level:       AlertLevelWarning,
		Enabled:     false, // Disabled for readers
		Config: map[string]interface{}{
			"threshold_usd": 5.0, // $5 threshold
		},
		CreatedAt: time.Now(),
	})

	// Market Impact Alerts
	am.conditions = append(am.conditions, AlertCondition{
		ID:          "market-impact",
		Name:        "Market Impact",
		Description: "Alert when articles mention significant market movements or financial disruptions",
		Type:        ConditionMarketImpact,
		Level:       AlertLevelWarning,
		Enabled:     true,
		Config: map[string]interface{}{
			"keywords": []string{
				"market crash", "stock plunge", "market volatility", "trading halt",
				"sector decline", "bubble burst", "market correction", "bear market",
				"recession", "economic downturn", "financial crisis", "currency collapse",
				"interest rate hike", "fed raises rates", "inflation surge", "gdp decline",
			},
			"case_insensitive": true,
		},
		CreatedAt: time.Now(),
	})

	// Regulatory & Policy Alerts
	am.conditions = append(am.conditions, AlertCondition{
		ID:          "regulatory-policy",
		Name:        "Regulatory & Policy Changes",
		Description: "Alert when government policy changes or new regulations are announced",
		Type:        ConditionRegulatoryPolicy,
		Level:       AlertLevelWarning,
		Enabled:     true,
		Config: map[string]interface{}{
			"keywords": []string{
				"new regulation", "policy change", "compliance deadline", "government announcement",
				"regulatory approval", "sanctions", "new law", "legal requirement",
				"antitrust", "privacy regulation", "data protection", "tax reform",
				"trade policy", "import ban", "export restriction", "regulatory framework",
			},
			"case_insensitive": true,
		},
		CreatedAt: time.Now(),
	})

	// Leadership & Personnel Alerts
	am.conditions = append(am.conditions, AlertCondition{
		ID:          "leadership-personnel",
		Name:        "Leadership & Personnel Changes",
		Description: "Alert when significant leadership changes or key appointments occur",
		Type:        ConditionLeadershipPersonnel,
		Level:       AlertLevelInfo,
		Enabled:     true,
		Config: map[string]interface{}{
			"keywords": []string{
				"ceo resigns", "ceo steps down", "new ceo", "chief executive",
				"cto appointed", "cfo leaves", "founder departs", "president resigns",
				"board member", "executive team", "leadership change", "succession",
				"interim ceo", "acting ceo", "new chief", "executive appointment",
			},
			"case_insensitive": true,
		},
		CreatedAt: time.Now(),
	})

	// Technology Disruption Alerts
	am.conditions = append(am.conditions, AlertCondition{
		ID:          "technology-disruption",
		Name:        "Technology Disruption",
		Description: "Alert when breakthrough technologies or major tech changes are announced",
		Type:        ConditionTechnologyDisruption,
		Level:       AlertLevelInfo,
		Enabled:     true,
		Config: map[string]interface{}{
			"keywords": []string{
				"breakthrough", "game changer", "revolutionary technology", "paradigm shift",
				"obsolete", "deprecated", "major update", "next generation",
				"artificial intelligence", "quantum computing", "blockchain revolution",
				"autonomous vehicles", "gene editing", "renewable energy breakthrough",
			},
			"case_insensitive": true,
		},
		CreatedAt: time.Now(),
	})

	// Competitive Intelligence Alerts
	am.conditions = append(am.conditions, AlertCondition{
		ID:          "competitive-intelligence",
		Name:        "Competitive Intelligence",
		Description: "Alert when partnerships, acquisitions, or competitive moves are announced",
		Type:        ConditionCompetitiveIntel,
		Level:       AlertLevelInfo,
		Enabled:     true,
		Config: map[string]interface{}{
			"keywords": []string{
				"strategic partnership", "acquires", "acquisition", "merger", "joint venture",
				"strategic alliance", "partnership agreement", "collaboration",
				"competitor", "market share", "competitive advantage", "buyout",
				"hostile takeover", "spinoff", "divestiture", "strategic investment",
			},
			"case_insensitive": true,
		},
		CreatedAt: time.Now(),
	})

	// Geographic Impact Alerts
	am.conditions = append(am.conditions, AlertCondition{
		ID:          "geographic-impact",
		Name:        "Geographic Impact",
		Description: "Alert when natural disasters or geopolitical events affect business operations",
		Type:        ConditionGeographicImpact,
		Level:       AlertLevelWarning,
		Enabled:     true,
		Config: map[string]interface{}{
			"keywords": []string{
				"earthquake", "hurricane", "tsunami", "flood", "wildfire",
				"supply chain disruption", "trade war", "border closure", "embargo",
				"natural disaster", "geopolitical crisis", "war", "conflict",
				"political instability", "coup", "election results", "diplomatic crisis",
			},
			"case_insensitive": true,
		},
		CreatedAt: time.Now(),
	})

	// Time-Sensitive Content Alerts
	am.conditions = append(am.conditions, AlertCondition{
		ID:          "time-sensitive",
		Name:        "Time-Sensitive Content",
		Description: "Alert when articles mention deadlines, urgent timelines, or expiring opportunities",
		Type:        ConditionTimeSensitive,
		Level:       AlertLevelWarning,
		Enabled:     true,
		Config: map[string]interface{}{
			"keywords": []string{
				"deadline", "expires", "last chance", "urgent", "time-sensitive",
				"final call", "limited time", "expires soon", "act now",
				"closing date", "registration deadline", "submission deadline",
				"offer expires", "window closing", "time running out",
			},
			"case_insensitive": true,
		},
		CreatedAt: time.Now(),
	})

	// Industry Disruption Alerts
	am.conditions = append(am.conditions, AlertCondition{
		ID:          "industry-disruption",
		Name:        "Industry Disruption",
		Description: "Alert when major industry changes or business model shifts occur",
		Type:        ConditionIndustryDisruption,
		Level:       AlertLevelCritical,
		Enabled:     true,
		Config: map[string]interface{}{
			"keywords": []string{
				"industry disruption", "business model", "transforms industry",
				"changes everything", "industry revolution", "disruptive innovation",
				"market transformation", "new business model", "industry shift",
				"digital transformation", "industry overhaul", "reimagining",
			},
			"case_insensitive": true,
		},
		CreatedAt: time.Now(),
	})

	// Multiple Source Validation Alerts
	am.conditions = append(am.conditions, AlertCondition{
		ID:          "multiple-source",
		Name:        "Multiple Source Validation",
		Description: "Alert when the same story is reported by multiple sources (indicates high importance)",
		Type:        ConditionMultipleSource,
		Level:       AlertLevelInfo,
		Enabled:     true,
		Config: map[string]interface{}{
			"min_sources":          3,   // Minimum number of sources
			"similarity_threshold": 0.7, // Topic similarity threshold
		},
		CreatedAt: time.Now(),
	})

	// Sentiment Extremes Alerts
	am.conditions = append(am.conditions, AlertCondition{
		ID:          "sentiment-extreme",
		Name:        "Sentiment Extremes",
		Description: "Alert when articles have extremely positive or negative sentiment",
		Type:        ConditionSentimentExtreme,
		Level:       AlertLevelInfo,
		Enabled:     true,
		Config: map[string]interface{}{
			"positive_threshold": 0.8,  // Very positive sentiment threshold
			"negative_threshold": -0.8, // Very negative sentiment threshold
		},
		CreatedAt: time.Now(),
	})
}

// CheckConditions evaluates all enabled alert conditions against the provided context
func (am *AlertManager) CheckConditions(ctx AlertContext) []Alert {
	var triggeredAlerts []Alert

	for _, condition := range am.conditions {
		if !condition.Enabled {
			continue
		}

		alert := am.evaluateCondition(condition, ctx)
		if alert != nil {
			triggeredAlerts = append(triggeredAlerts, *alert)
			am.alerts = append(am.alerts, *alert)
		}
	}

	return triggeredAlerts
}

// AlertContext contains data for alert evaluation
type AlertContext struct {
	Articles        []core.Article         `json:"articles"`
	Digests         []core.Digest          `json:"digests"`
	CurrentTopics   []string               `json:"current_topics"`
	PreviousTopics  []string               `json:"previous_topics"`
	EstimatedCost   float64                `json:"estimated_cost"`
	ProcessingStats map[string]interface{} `json:"processing_stats"`
}

// EvaluateAlerts processes a list of articles and checks for triggered alert conditions
func (am *AlertManager) EvaluateAlerts(articles []core.Article) ([]Alert, error) {
	if len(articles) == 0 {
		return []Alert{}, nil
	}

	// Build alert context from articles
	ctx := AlertContext{
		Articles:       articles,
		Digests:        []core.Digest{}, // Empty for now - could be populated if needed
		CurrentTopics:  am.extractTopicsFromArticles(articles),
		PreviousTopics: []string{}, // Would need historical data
		EstimatedCost:  0.0,        // Could calculate based on article count
		ProcessingStats: map[string]interface{}{
			"article_count": len(articles),
			"processed_at":  time.Now(),
		},
	}

	// Check all conditions
	triggeredAlerts := am.CheckConditions(ctx)

	return triggeredAlerts, nil
}

// extractTopicsFromArticles extracts topic keywords from article titles and content
func (am *AlertManager) extractTopicsFromArticles(articles []core.Article) []string {
	topicSet := make(map[string]bool)

	for _, article := range articles {
		// Extract from title
		titleWords := strings.Fields(strings.ToLower(article.Title))
		for _, word := range titleWords {
			// Skip common words and focus on meaningful terms
			if len(word) > 3 && !isCommonWord(word) {
				topicSet[word] = true
			}
		}

		// Extract from cleaned text if available (first few words to avoid overload)
		if article.CleanedText != "" {
			// Take first 200 characters to get key terms without overwhelming
			text := article.CleanedText
			if len(text) > 200 {
				text = text[:200]
			}
			textWords := strings.Fields(strings.ToLower(text))
			for _, word := range textWords {
				if len(word) > 3 && !isCommonWord(word) {
					topicSet[word] = true
				}
			}
		}
	}

	// Convert to slice
	var topics []string
	for topic := range topicSet {
		topics = append(topics, topic)
	}

	return topics
}

// isCommonWord checks if a word is too common to be considered a topic
func isCommonWord(word string) bool {
	commonWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true,
		"not": true, "you": true, "all": true, "can": true, "had": true,
		"her": true, "was": true, "one": true, "our": true, "out": true,
		"day": true, "get": true, "has": true, "him": true, "his": true,
		"how": true, "man": true, "new": true, "now": true, "old": true,
		"see": true, "two": true, "way": true, "who": true, "boy": true,
		"did": true, "its": true, "let": true, "put": true, "say": true,
		"she": true, "too": true, "use": true, "with": true, "from": true,
		"have": true, "they": true, "will": true, "about": true, "could": true,
		"there": true, "other": true, "would": true, "which": true,
	}

	return commonWords[word]
}

// evaluateCondition checks if a specific condition is triggered
func (am *AlertManager) evaluateCondition(condition AlertCondition, ctx AlertContext) *Alert {
	switch condition.Type {
	case ConditionKeywordMatch:
		return am.checkKeywordMatch(condition, ctx)
	case ConditionTopicEmergence:
		return am.checkTopicEmergence(condition, ctx)
	case ConditionVolumeChange:
		return am.checkVolumeChange(condition, ctx)
	case ConditionCostThreshold:
		return am.checkCostThreshold(condition, ctx)
	case ConditionMarketImpact:
		return am.checkKeywordMatch(condition, ctx)
	case ConditionRegulatoryPolicy:
		return am.checkKeywordMatch(condition, ctx)
	case ConditionLeadershipPersonnel:
		return am.checkKeywordMatch(condition, ctx)
	case ConditionTechnologyDisruption:
		return am.checkKeywordMatch(condition, ctx)
	case ConditionCompetitiveIntel:
		return am.checkKeywordMatch(condition, ctx)
	case ConditionGeographicImpact:
		return am.checkKeywordMatch(condition, ctx)
	case ConditionTimeSensitive:
		return am.checkKeywordMatch(condition, ctx)
	case ConditionIndustryDisruption:
		return am.checkKeywordMatch(condition, ctx)
	case ConditionMultipleSource:
		return am.checkMultipleSource(condition, ctx)
	case ConditionSentimentExtreme:
		return am.checkSentimentExtreme(condition, ctx)
	default:
		return nil
	}
}

// checkKeywordMatch evaluates keyword matching conditions
func (am *AlertManager) checkKeywordMatch(condition AlertCondition, ctx AlertContext) *Alert {
	keywords, ok := condition.Config["keywords"].([]string)
	if !ok {
		return nil
	}

	caseInsensitive, _ := condition.Config["case_insensitive"].(bool)

	var matchedArticles []string
	var matchedKeywords []string

	for _, article := range ctx.Articles {
		content := article.CleanedText + " " + article.Title
		if caseInsensitive {
			content = strings.ToLower(content)
		}

		for _, keyword := range keywords {
			searchKeyword := keyword
			if caseInsensitive {
				searchKeyword = strings.ToLower(keyword)
			}

			if strings.Contains(content, searchKeyword) {
				matchedArticles = append(matchedArticles, article.Title)
				matchedKeywords = append(matchedKeywords, keyword)
			}
		}
	}

	if len(matchedArticles) > 0 {
		alert := &Alert{
			ID:          fmt.Sprintf("keyword-match-%d", time.Now().Unix()),
			ConditionID: condition.ID,
			Level:       condition.Level,
			Title:       "High Priority Keywords Detected",
			Message: fmt.Sprintf("Found %d articles containing priority keywords: %s",
				len(matchedArticles), strings.Join(uniqueStrings(matchedKeywords), ", ")),
			Context: map[string]interface{}{
				"matched_articles": matchedArticles,
				"matched_keywords": uniqueStrings(matchedKeywords),
			},
			TriggeredAt: time.Now(),
		}
		return alert
	}

	return nil
}

// checkTopicEmergence evaluates topic emergence conditions
func (am *AlertManager) checkTopicEmergence(condition AlertCondition, ctx AlertContext) *Alert {
	// Note: min_articles configuration is available but not currently used in this implementation

	// Find topics that are new (present in current but not in previous)
	var newTopics []string
	for _, currentTopic := range ctx.CurrentTopics {
		isNew := true
		for _, previousTopic := range ctx.PreviousTopics {
			if currentTopic == previousTopic {
				isNew = false
				break
			}
		}
		if isNew {
			newTopics = append(newTopics, currentTopic)
		}
	}

	if len(newTopics) > 0 {
		alert := &Alert{
			ID:          fmt.Sprintf("topic-emergence-%d", time.Now().Unix()),
			ConditionID: condition.ID,
			Level:       condition.Level,
			Title:       "New Topics Emerged",
			Message: fmt.Sprintf("Detected %d new topics: %s",
				len(newTopics), strings.Join(newTopics, ", ")),
			Context: map[string]interface{}{
				"new_topics": newTopics,
			},
			TriggeredAt: time.Now(),
		}
		return alert
	}

	return nil
}

// checkVolumeChange evaluates volume change conditions
func (am *AlertManager) checkVolumeChange(condition AlertCondition, ctx AlertContext) *Alert {
	threshold, _ := condition.Config["threshold_percent"].(float64)
	if threshold == 0 {
		threshold = 50.0
	}

	currentCount := len(ctx.Articles)

	// Get previous count from processing stats
	previousCount := 0
	if stats, ok := ctx.ProcessingStats["previous_article_count"].(int); ok {
		previousCount = stats
	}

	if previousCount > 0 {
		changePercent := (float64(currentCount-previousCount) / float64(previousCount)) * 100

		if changePercent > threshold {
			alert := &Alert{
				ID:          fmt.Sprintf("volume-spike-%d", time.Now().Unix()),
				ConditionID: condition.ID,
				Level:       condition.Level,
				Title:       "Article Volume Spike Detected",
				Message: fmt.Sprintf("Article volume increased by %.1f%% (%d → %d articles)",
					changePercent, previousCount, currentCount),
				Context: map[string]interface{}{
					"current_count":  currentCount,
					"previous_count": previousCount,
					"change_percent": changePercent,
				},
				TriggeredAt: time.Now(),
			}
			return alert
		}
	}

	return nil
}

// checkCostThreshold evaluates cost threshold conditions
func (am *AlertManager) checkCostThreshold(condition AlertCondition, ctx AlertContext) *Alert {
	threshold, _ := condition.Config["threshold_usd"].(float64)
	if threshold == 0 {
		threshold = 5.0
	}

	if ctx.EstimatedCost > threshold {
		alert := &Alert{
			ID:          fmt.Sprintf("cost-threshold-%d", time.Now().Unix()),
			ConditionID: condition.ID,
			Level:       condition.Level,
			Title:       "Cost Threshold Exceeded",
			Message: fmt.Sprintf("Estimated processing cost $%.2f exceeds threshold $%.2f",
				ctx.EstimatedCost, threshold),
			Context: map[string]interface{}{
				"estimated_cost": ctx.EstimatedCost,
				"threshold":      threshold,
			},
			TriggeredAt: time.Now(),
		}
		return alert
	}

	return nil
}

// checkMultipleSource evaluates multiple source validation conditions
func (am *AlertManager) checkMultipleSource(condition AlertCondition, ctx AlertContext) *Alert {
	minSources, _ := condition.Config["min_sources"].(int)
	if minSources == 0 {
		minSources = 3
	}

	// Group articles by similar topics/keywords
	topicGroups := make(map[string][]string)

	for _, article := range ctx.Articles {
		// Extract key terms from title for grouping
		title := strings.ToLower(article.Title)
		words := strings.Fields(title)

		// Create topic signature from meaningful words
		var topicWords []string
		for _, word := range words {
			if len(word) > 3 && !isCommonWord(word) {
				topicWords = append(topicWords, word)
			}
		}

		if len(topicWords) > 0 {
			// Use first few words as topic signature
			topicKey := strings.Join(topicWords[:min(len(topicWords), 3)], " ")
			topicGroups[topicKey] = append(topicGroups[topicKey], article.Title)
		}
	}

	// Check if any topic appears in multiple articles from different sources
	for topic, articles := range topicGroups {
		if len(articles) >= minSources {
			alert := &Alert{
				ID:          fmt.Sprintf("multiple-source-%d", time.Now().Unix()),
				ConditionID: condition.ID,
				Level:       condition.Level,
				Title:       "Multiple Source Validation",
				Message:     fmt.Sprintf("Story covered by %d sources: %s", len(articles), topic),
				Context: map[string]interface{}{
					"topic":            topic,
					"source_count":     len(articles),
					"matched_articles": articles,
				},
				TriggeredAt: time.Now(),
			}
			return alert
		}
	}

	return nil
}

// checkSentimentExtreme evaluates sentiment extreme conditions
func (am *AlertManager) checkSentimentExtreme(condition AlertCondition, ctx AlertContext) *Alert {
	positiveThreshold, _ := condition.Config["positive_threshold"].(float64)
	if positiveThreshold == 0 {
		positiveThreshold = 0.8
	}

	negativeThreshold, _ := condition.Config["negative_threshold"].(float64)
	if negativeThreshold == 0 {
		negativeThreshold = -0.8
	}

	var extremeArticles []string
	var sentimentType string
	var averageScore float64
	var count int

	for _, article := range ctx.Articles {
		// Check if article has sentiment score (this would come from sentiment analysis)
		// For now, we'll check if the article object has sentiment fields
		if article.Title != "" { // Placeholder - in real implementation, check article.SentimentScore
			// This is a simplified check - in practice, you'd get sentiment from the article
			// For demonstration, we'll trigger on certain emotional keywords
			title := strings.ToLower(article.Title)
			content := strings.ToLower(article.CleanedText)

			// Check for extremely positive indicators
			positiveWords := []string{"amazing", "incredible", "revolutionary", "breakthrough", "spectacular", "outstanding"}
			negativeWords := []string{"catastrophic", "devastating", "disaster", "crisis", "collapse", "failure"}

			for _, word := range positiveWords {
				if strings.Contains(title, word) || strings.Contains(content, word) {
					extremeArticles = append(extremeArticles, article.Title)
					sentimentType = "extremely positive"
					averageScore += positiveThreshold
					count++
					break
				}
			}

			for _, word := range negativeWords {
				if strings.Contains(title, word) || strings.Contains(content, word) {
					extremeArticles = append(extremeArticles, article.Title)
					sentimentType = "extremely negative"
					averageScore += negativeThreshold
					count++
					break
				}
			}
		}
	}

	if count > 0 {
		averageScore = averageScore / float64(count)
		alert := &Alert{
			ID:          fmt.Sprintf("sentiment-extreme-%d", time.Now().Unix()),
			ConditionID: condition.ID,
			Level:       condition.Level,
			Title:       "Sentiment Extremes Detected",
			Message:     fmt.Sprintf("%d articles with %s sentiment (avg: %.2f)", count, sentimentType, averageScore),
			Context: map[string]interface{}{
				"sentiment_type":   sentimentType,
				"article_count":    count,
				"average_score":    averageScore,
				"matched_articles": extremeArticles,
			},
			TriggeredAt: time.Now(),
		}
		return alert
	}

	return nil
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AddCondition adds a new alert condition
func (am *AlertManager) AddCondition(condition AlertCondition) {
	am.conditions = append(am.conditions, condition)
}

// GetConditions returns all alert conditions
func (am *AlertManager) GetConditions() []AlertCondition {
	return am.conditions
}

// GetDefaultConditions returns the default alert conditions
func (am *AlertManager) GetDefaultConditions() []AlertCondition {
	return am.conditions
}

// GetAlerts returns all alerts (optionally filtered by level)
func (am *AlertManager) GetAlerts(level *AlertLevel) []Alert {
	if level == nil {
		return am.alerts
	}

	var filtered []Alert
	for _, alert := range am.alerts {
		if alert.Level == *level {
			filtered = append(filtered, alert)
		}
	}
	return filtered
}

// AcknowledgeAlert marks an alert as acknowledged
func (am *AlertManager) AcknowledgeAlert(alertID string) error {
	for i, alert := range am.alerts {
		if alert.ID == alertID {
			am.alerts[i].Acknowledged = true
			am.alerts[i].AcknowledgedAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("alert with ID %s not found", alertID)
}

// FormatAlert creates a human-readable representation of an alert
func (am *AlertManager) FormatAlert(alert Alert) string {
	levelEmoji := map[AlertLevel]string{
		AlertLevelInfo:     "ℹ️",
		AlertLevelWarning:  "⚠️",
		AlertLevelCritical: "🚨",
	}

	emoji := levelEmoji[alert.Level]
	timestamp := alert.TriggeredAt.Format("2006-01-02 15:04")

	return fmt.Sprintf("%s **%s** [%s]\n%s\n*Triggered at %s*",
		emoji, alert.Title, alert.Level.String(), alert.Message, timestamp)
}

// FormatAlertsSection creates a formatted section for inclusion in digests
func (am *AlertManager) FormatAlertsSection(alerts []Alert) string {
	if len(alerts) == 0 {
		return ""
	}

	var builder strings.Builder
	builder.WriteString("## 🚨 Alerts\n\n")

	for _, alert := range alerts {
		builder.WriteString(am.FormatAlert(alert))
		builder.WriteString("\n\n")
	}

	return builder.String()
}

// Helper functions

// uniqueStrings removes duplicates from a string slice
func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
