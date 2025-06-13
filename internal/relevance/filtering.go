package relevance

import (
	"context"
	"fmt"
	"sort"
)

// FilterResult represents the result of content filtering
type FilterResult struct {
	Content  Scorable `json:"content"`
	Score    Score    `json:"score"`
	Included bool     `json:"included"`
	Reason   string   `json:"reason"`
}

// FilterByThreshold filters content based on minimum relevance threshold
func FilterByThreshold(ctx context.Context, scorer Scorer, contents []Scorable, criteria Criteria) ([]FilterResult, error) {
	if len(contents) == 0 {
		return []FilterResult{}, nil
	}

	// Apply quality filters first
	filteredContents := applyQualityFilters(contents, criteria.Filters)

	// Score remaining content
	scores, err := scorer.ScoreBatch(ctx, filteredContents, criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to score content: %w", err)
	}

	// Create filter results
	results := make([]FilterResult, len(filteredContents))
	for i, content := range filteredContents {
		score := scores[i]
		included := score.Value >= criteria.Threshold
		reason := generateFilterReason(score, criteria.Threshold, included)

		results[i] = FilterResult{
			Content:  content,
			Score:    score,
			Included: included,
			Reason:   reason,
		}
	}

	return results, nil
}

// FilterForDigest filters and prioritizes content for digest generation with word budget
func FilterForDigest(ctx context.Context, scorer Scorer, contents []Scorable, criteria Criteria, maxWords int) ([]FilterResult, error) {
	// Get initial filtering results
	results, err := FilterByThreshold(ctx, scorer, contents, criteria)
	if err != nil {
		return nil, err
	}

	// Sort by relevance score (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score.Value > results[j].Score.Value
	})

	// Apply word budget constraints if specified
	if maxWords > 0 {
		results = applyWordBudget(results, maxWords)
	}

	return results, nil
}

// GetIncludedContent returns only the content that passed filtering
func GetIncludedContent(results []FilterResult) []Scorable {
	var included []Scorable
	for _, result := range results {
		if result.Included {
			included = append(included, result.Content)
		}
	}
	return included
}

// GetExcludedContent returns only the content that was filtered out
func GetExcludedContent(results []FilterResult) []FilterResult {
	var excluded []FilterResult
	for _, result := range results {
		if !result.Included {
			excluded = append(excluded, result)
		}
	}
	return excluded
}

// FilterStats provides statistics about filtering results
type FilterStats struct {
	TotalItems    int     `json:"total_items"`
	IncludedItems int     `json:"included_items"`
	ExcludedItems int     `json:"excluded_items"`
	AvgScore      float64 `json:"avg_score"`
	MaxScore      float64 `json:"max_score"`
	MinScore      float64 `json:"min_score"`
	Threshold     float64 `json:"threshold"`
}

// GetFilterStats calculates statistics from filter results
func GetFilterStats(results []FilterResult, threshold float64) FilterStats {
	if len(results) == 0 {
		return FilterStats{Threshold: threshold}
	}

	stats := FilterStats{
		TotalItems: len(results),
		Threshold:  threshold,
		MinScore:   1.0,
		MaxScore:   0.0,
	}

	totalScore := 0.0
	for _, result := range results {
		if result.Included {
			stats.IncludedItems++
		} else {
			stats.ExcludedItems++
		}

		score := result.Score.Value
		totalScore += score

		if score > stats.MaxScore {
			stats.MaxScore = score
		}
		if score < stats.MinScore {
			stats.MinScore = score
		}
	}

	stats.AvgScore = totalScore / float64(len(results))
	return stats
}

// InferDigestTheme attempts to infer the main theme from article titles and content
func InferDigestTheme(contents []Scorable) string {
	if len(contents) == 0 {
		return ""
	}

	// Collect all titles and extract common keywords
	var allText string
	for _, content := range contents {
		allText += content.GetTitle() + " "
	}

	// Simple keyword extraction for theme inference
	scorer := NewKeywordScorer()
	keywords := scorer.extractKeywords(allText)

	// Find most common technical terms that could indicate theme
	techTerms := map[string]string{
		"ai":         "artificial intelligence",
		"ml":         "machine learning",
		"llm":        "large language models",
		"security":   "cybersecurity",
		"crypto":     "cryptocurrency",
		"blockchain": "blockchain technology",
		"cloud":      "cloud computing",
		"kubernetes": "container orchestration",
		"docker":     "containerization",
		"go":         "Go programming",
		"rust":       "Rust programming",
		"python":     "Python development",
		"javascript": "JavaScript development",
		"react":      "React development",
		"api":        "API development",
		"database":   "database technology",
		"web":        "web development",
		"mobile":     "mobile development",
		"devops":     "DevOps",
		"startup":    "startup technology",
	}

	// Count occurrences and find most common theme
	termCounts := make(map[string]int)
	for _, keyword := range keywords {
		if theme, exists := techTerms[keyword]; exists {
			termCounts[theme]++
		}
	}

	// Find the most common theme
	maxCount := 0
	commonTheme := ""
	for theme, count := range termCounts {
		if count > maxCount {
			maxCount = count
			commonTheme = theme
		}
	}

	if commonTheme != "" {
		return commonTheme
	}

	// Fallback: use most common keyword
	if len(keywords) > 0 {
		return keywords[0]
	}

	return "technology" // Default theme
}

// applyQualityFilters applies quality filters to content before scoring
func applyQualityFilters(contents []Scorable, filters []Filter) []Scorable {
	if len(filters) == 0 {
		return contents
	}

	var filtered []Scorable
	for _, content := range contents {
		passesAllFilters := true
		for _, filter := range filters {
			if !filter.Apply(content) {
				passesAllFilters = false
				break
			}
		}
		if passesAllFilters {
			filtered = append(filtered, content)
		}
	}

	return filtered
}

// applyWordBudget applies word budget constraints to prioritize high-relevance content
func applyWordBudget(results []FilterResult, maxWords int) []FilterResult {
	// This is a simplified implementation - in practice, you'd want to:
	// 1. Estimate words per article based on content length
	// 2. Include articles until budget is exhausted
	// 3. Prioritize by relevance score

	// For now, we'll use a simple heuristic: assume average 100 words per article summary
	avgWordsPerArticle := 100
	maxArticles := maxWords / avgWordsPerArticle

	if maxArticles <= 0 {
		maxArticles = 1 // Always include at least one article
	}

	// Mark articles beyond the budget as excluded
	for i := range results {
		if i < maxArticles && results[i].Score.Value >= ThresholdImportant {
			results[i].Included = true
			results[i].Reason = fmt.Sprintf("Included: High relevance (%.2f) within word budget", results[i].Score.Value)
		} else if i >= maxArticles {
			results[i].Included = false
			results[i].Reason = fmt.Sprintf("Excluded: Over word budget (ranked #%d)", i+1)
		}
	}

	return results
}

// generateFilterReason creates human-readable reason for inclusion/exclusion
func generateFilterReason(score Score, threshold float64, included bool) string {
	if included {
		if score.Value >= ThresholdCritical {
			return fmt.Sprintf("🔥 Critical relevance (%.2f)", score.Value)
		} else if score.Value >= ThresholdImportant {
			return fmt.Sprintf("⭐ Important relevance (%.2f)", score.Value)
		} else {
			return fmt.Sprintf("✅ Above threshold (%.2f ≥ %.2f)", score.Value, threshold)
		}
	} else {
		if score.Value < ThresholdMinimum {
			return fmt.Sprintf("❌ Very low relevance (%.2f)", score.Value)
		} else {
			return fmt.Sprintf("💡 Below threshold (%.2f < %.2f)", score.Value, threshold)
		}
	}
}
