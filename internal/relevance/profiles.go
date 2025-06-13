package relevance

// Pre-configured scoring weight profiles for different use cases

var (
	// DigestWeights optimized for digest content filtering
	// High content relevance, moderate title weight, low authority/recency importance
	DigestWeights = ScoringWeights{
		ContentRelevance: 0.6, // High content match weight
		TitleRelevance:   0.3, // Medium title weight
		Authority:        0.1, // Low source authority weight
		Recency:          0.0, // Time not critical for digests
		Quality:          0.0, // Quality handled by content relevance
	}

	// ResearchWeights optimized for research result ranking
	// Balanced content, high authority, some recency consideration
	ResearchWeights = ScoringWeights{
		ContentRelevance: 0.4, // Balanced content weight
		TitleRelevance:   0.2, // Lower title weight
		Authority:        0.3, // High authority weight for research
		Recency:          0.1, // Some recency consideration
		Quality:          0.0, // Quality embedded in authority
	}

	// ResearchV2Weights optimized for enhanced research v2 requirements
	// Multi-dimensional relevancy assessment with technical depth and competitive value
	ResearchV2Weights = ScoringWeights{
		ContentRelevance: 0.30, // Core topic match
		TitleRelevance:   0.15, // Moderate title importance
		Authority:        0.20, // Source credibility
		Recency:          0.15, // Publication freshness
		Quality:          0.20, // Technical depth + competitive value
	}

	// CompetitiveAnalysisWeights optimized for competitive intelligence
	// High authority and quality focus with competitive value emphasis
	CompetitiveAnalysisWeights = ScoringWeights{
		ContentRelevance: 0.25, // Moderate content relevance
		TitleRelevance:   0.15, // Lower title weight
		Authority:        0.30, // Very high authority weight
		Recency:          0.20, // Recent competitive intelligence
		Quality:          0.10, // Competitive comparison richness
	}

	// TechnicalDeepDiveWeights optimized for technical assessment
	// High content relevance with technical depth emphasis
	TechnicalDeepDiveWeights = ScoringWeights{
		ContentRelevance: 0.35, // High technical content match
		TitleRelevance:   0.15, // Moderate title importance
		Authority:        0.25, // Technical source credibility
		Recency:          0.10, // Technical content less time-sensitive
		Quality:          0.15, // Technical detail richness
	}

	// InteractiveWeights optimized for TUI/interactive browsing
	// High title importance for quick scanning, balanced content relevance
	InteractiveWeights = ScoringWeights{
		ContentRelevance: 0.5, // Balanced relevance
		TitleRelevance:   0.4, // High title importance for scanning
		Authority:        0.1, // Low authority weight
		Recency:          0.0, // Time irrelevant for browsing
		Quality:          0.0, // Quality not critical for interactive
	}

	// NewsWeights optimized for news/current events
	// High recency, balanced content, moderate authority
	NewsWeights = ScoringWeights{
		ContentRelevance: 0.4, // Balanced content relevance
		TitleRelevance:   0.2, // Moderate title weight
		Authority:        0.2, // Moderate source authority
		Recency:          0.2, // High recency for news
		Quality:          0.0, // Quality handled by authority
	}

	// EducationalWeights optimized for learning/tutorial content
	// Very high content relevance, high quality, moderate authority
	EducationalWeights = ScoringWeights{
		ContentRelevance: 0.5, // Very high content match
		TitleRelevance:   0.2, // Moderate title weight
		Authority:        0.2, // Moderate authority for credibility
		Recency:          0.0, // Time not critical for educational content
		Quality:          0.1, // Some quality consideration
	}
)

// GetWeightsForContext returns appropriate scoring weights based on context
func GetWeightsForContext(context string) ScoringWeights {
	switch context {
	case "digest":
		return DigestWeights
	case "research", "deep_research":
		return ResearchWeights
	case "research_v2":
		return ResearchV2Weights
	case "competitive", "competitive_analysis":
		return CompetitiveAnalysisWeights
	case "technical", "technical_analysis":
		return TechnicalDeepDiveWeights
	case "tui", "interactive", "browse":
		return InteractiveWeights
	case "news", "feed":
		return NewsWeights
	case "educational", "learning", "tutorial":
		return EducationalWeights
	default:
		// Default to digest weights for unknown contexts
		return DigestWeights
	}
}

// DefaultCriteria creates default criteria for a given context
func DefaultCriteria(context, query string) Criteria {
	weights := GetWeightsForContext(context)

	// Set default thresholds based on context
	threshold := ThresholdOptional
	switch context {
	case "digest":
		threshold = ThresholdImportant // Higher bar for digest inclusion
	case "research":
		threshold = ThresholdOptional // Lower bar for research exploration
	case "tui":
		threshold = ThresholdMinimum // Lowest bar for interactive browsing
	}

	return Criteria{
		Query:     query,
		Keywords:  []string{}, // Will be extracted from query
		Weights:   weights,
		Context:   context,
		Filters:   []Filter{}, // No filters by default
		Threshold: threshold,
	}
}

// CommonFilters provides commonly used content filters
func CommonFilters() map[string]Filter {
	return map[string]Filter{
		"min_content_length": FilterFunc{
			FilterName: "min_content_length",
			FilterDesc: "Filters out content with less than 100 characters",
			Fn: func(content Scorable) bool {
				return len(content.GetContent()) >= 100
			},
		},
		"has_title": FilterFunc{
			FilterName: "has_title",
			FilterDesc: "Filters out content without a title",
			Fn: func(content Scorable) bool {
				return len(content.GetTitle()) > 0
			},
		},
		"valid_url": FilterFunc{
			FilterName: "valid_url",
			FilterDesc: "Filters out content without a valid URL",
			Fn: func(content Scorable) bool {
				url := content.GetURL()
				return len(url) > 0 && (len(url) > 7) // Basic URL validation
			},
		},
		"no_spam_domains": FilterFunc{
			FilterName: "no_spam_domains",
			FilterDesc: "Filters out content from known spam domains",
			Fn: func(content Scorable) bool {
				url := content.GetURL()
				spamDomains := []string{"example.com", "test.com", "spam.com"}
				for _, domain := range spamDomains {
					if contains(url, domain) {
						return false
					}
				}
				return true
			},
		},
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
