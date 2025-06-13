package services

import (
	"briefly/internal/core"
	"briefly/internal/llm"
	"context"
	"fmt"
	"sort"
	"strings"
)

// ResultClusterer handles automatic categorization and organization of research results
type ResultClusterer struct {
	llmClient *llm.Client
}

// NewResultClusterer creates a new result clusterer
func NewResultClusterer(llmClient *llm.Client) *ResultClusterer {
	return &ResultClusterer{
		llmClient: llmClient,
	}
}

// ClusterCategory represents a categorized group of research results
type ClusterCategory struct {
	Name        string                `json:"name"`
	Description string                `json:"description"`
	Results     []core.ResearchResult `json:"results"`
	Quality     float64               `json:"quality"`  // Average relevance of results in cluster
	Density     float64               `json:"density"`  // How well-populated this cluster is
	Priority    int                   `json:"priority"` // Display order priority
}

// ClusteringResult contains the organized research results
type ClusteringResult struct {
	Categories         []ClusterCategory `json:"categories"`
	OverallQuality     float64           `json:"overall_quality"`
	CoverageGaps       []string          `json:"coverage_gaps"`       // Missing information areas
	TotalCategorized   int               `json:"total_categorized"`   // Number of results categorized
	UncategorizedCount int               `json:"uncategorized_count"` // Results that didn't fit categories
}

// ClusterResults automatically categorizes research results into meaningful groups
func (rc *ResultClusterer) ClusterResults(ctx context.Context, query string, results []core.ResearchResult) (*ClusteringResult, error) {
	if len(results) == 0 {
		return &ClusteringResult{
			Categories:         []ClusterCategory{},
			OverallQuality:     0.0,
			CoverageGaps:       []string{"No results found to analyze"},
			TotalCategorized:   0,
			UncategorizedCount: 0,
		}, nil
	}

	// Generate category definitions based on query and results
	categories, err := rc.generateCategoryDefinitions(ctx, query, results)
	if err != nil {
		return nil, fmt.Errorf("failed to generate categories: %w", err)
	}

	// Categorize results into clusters
	clusteredCategories := rc.categorizeResults(results, categories)

	// Calculate cluster quality metrics
	rc.calculateClusterMetrics(clusteredCategories)

	// Sort categories by priority and quality
	sort.Slice(clusteredCategories, func(i, j int) bool {
		if clusteredCategories[i].Priority != clusteredCategories[j].Priority {
			return clusteredCategories[i].Priority < clusteredCategories[j].Priority
		}
		return clusteredCategories[i].Quality > clusteredCategories[j].Quality
	})

	// Identify coverage gaps
	gaps := rc.identifyCoverageGaps(ctx, query, clusteredCategories)

	// Calculate overall statistics
	totalCategorized := 0
	overallQuality := 0.0
	for _, category := range clusteredCategories {
		totalCategorized += len(category.Results)
		overallQuality += category.Quality * float64(len(category.Results))
	}
	if totalCategorized > 0 {
		overallQuality /= float64(totalCategorized)
	}

	uncategorizedCount := len(results) - totalCategorized

	return &ClusteringResult{
		Categories:         clusteredCategories,
		OverallQuality:     overallQuality,
		CoverageGaps:       gaps,
		TotalCategorized:   totalCategorized,
		UncategorizedCount: uncategorizedCount,
	}, nil
}

// generateCategoryDefinitions creates category templates based on research requirements
func (rc *ResultClusterer) generateCategoryDefinitions(ctx context.Context, query string, results []core.ResearchResult) ([]ClusterCategory, error) {
	// Sample results for LLM analysis (max 10 to avoid token limits)
	sampleResults := results
	if len(results) > 10 {
		sampleResults = results[:10]
	}

	// Build context from sample results
	var contextBuilder strings.Builder
	contextBuilder.WriteString("Sample research results:\n")
	for i, result := range sampleResults {
		contextBuilder.WriteString(fmt.Sprintf("%d. %s\n   %s\n", i+1, result.Title, result.Snippet))
	}

	prompt := fmt.Sprintf(`Based on the research query "%s" and the sample results below, define 6 meaningful categories to organize research findings.

%s

Create categories that follow the research v2 framework:
1. Overview - General introduction and background information
2. Competitive Analysis - Direct comparisons, market positioning, alternatives  
3. Technical Details - Architecture, implementation, performance specifics
4. Use Cases - Real-world applications, case studies, examples
5. Limitations - Known issues, constraints, criticisms, challenges
6. Recent Developments - Latest updates, roadmap items, news

For each category, provide:
- Category name (2-4 words)  
- Brief description of what belongs in this category
- Keywords that indicate content belongs here

Format as JSON:
{
  "categories": [
    {
      "name": "Overview",
      "description": "General background and introductory information",
      "keywords": ["overview", "introduction", "background", "basics"]
    }
  ]
}`, query, contextBuilder.String())

	response, err := rc.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		MaxTokens:   800,
		Temperature: 0.5,
		Model:       "gemini-1.5-flash",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate category definitions: %w", err)
	}

	// Parse response and create default categories if parsing fails
	categories := rc.parseCategories(response)
	if len(categories) == 0 {
		categories = rc.getDefaultCategories()
	}

	return categories, nil
}

// parseCategories attempts to parse LLM response into categories (simplified implementation)
func (rc *ResultClusterer) parseCategories(response string) []ClusterCategory {
	// For now, return default categories - a full JSON parser would go here
	// This is a simplified implementation for the demo
	return rc.getDefaultCategories()
}

// getDefaultCategories returns the standard research v2 categories
func (rc *ResultClusterer) getDefaultCategories() []ClusterCategory {
	return []ClusterCategory{
		{
			Name:        "Overview",
			Description: "General introduction, background information, and foundational concepts",
			Priority:    1,
		},
		{
			Name:        "Competitive Analysis",
			Description: "Direct comparisons, market positioning, alternatives, and competitive intelligence",
			Priority:    2,
		},
		{
			Name:        "Technical Details",
			Description: "Architecture, implementation specifics, performance data, and technical specifications",
			Priority:    3,
		},
		{
			Name:        "Use Cases",
			Description: "Real-world applications, case studies, examples, and practical implementations",
			Priority:    4,
		},
		{
			Name:        "Limitations",
			Description: "Known issues, constraints, criticisms, challenges, and potential problems",
			Priority:    5,
		},
		{
			Name:        "Recent Developments",
			Description: "Latest updates, recent news, roadmap items, and emerging trends",
			Priority:    6,
		},
	}
}

// categorizeResults assigns research results to appropriate categories
func (rc *ResultClusterer) categorizeResults(results []core.ResearchResult, categories []ClusterCategory) []ClusterCategory {
	// Create category classification keywords
	categoryKeywords := map[string][]string{
		"Overview": {
			"overview", "introduction", "what is", "basics", "fundamental", "guide",
			"explained", "understanding", "beginner", "getting started", "definition",
		},
		"Competitive Analysis": {
			"vs", "versus", "comparison", "compare", "alternative", "competitor",
			"market", "share", "positioning", "rivals", "competing", "against",
			"advantages", "disadvantages", "pros", "cons", "better", "worse",
		},
		"Technical Details": {
			"architecture", "implementation", "technical", "api", "code", "engineering",
			"performance", "benchmark", "scalability", "algorithm", "framework",
			"infrastructure", "design", "specification", "documentation", "sdk",
		},
		"Use Cases": {
			"use case", "example", "case study", "application", "implementation",
			"real world", "practical", "scenario", "solution", "project", "deployment",
			"success story", "tutorial", "how to", "guide", "walkthrough",
		},
		"Limitations": {
			"limitation", "problem", "issue", "challenge", "difficulty", "constraint",
			"drawback", "disadvantage", "weakness", "criticism", "fail", "error",
			"bug", "vulnerability", "risk", "concern", "downside", "negative",
		},
		"Recent Developments": {
			"new", "latest", "recent", "update", "announcement", "release", "roadmap",
			"future", "upcoming", "development", "news", "2024", "2023", "beta",
			"preview", "launch", "version", "improvement", "feature", "enhancement",
		},
	}

	// Initialize categories with empty results
	for i := range categories {
		categories[i].Results = []core.ResearchResult{}
	}

	// Categorize each result
	for _, result := range results {
		bestCategory := rc.findBestCategory(result, categories, categoryKeywords)
		if bestCategory != -1 {
			categories[bestCategory].Results = append(categories[bestCategory].Results, result)
		}
	}

	return categories
}

// findBestCategory determines the best category for a research result
func (rc *ResultClusterer) findBestCategory(result core.ResearchResult, categories []ClusterCategory, categoryKeywords map[string][]string) int {
	text := strings.ToLower(result.Title + " " + result.Snippet)
	bestScore := 0.0
	bestCategory := -1

	for i, category := range categories {
		score := 0.0
		keywords := categoryKeywords[category.Name]

		for _, keyword := range keywords {
			if strings.Contains(text, keyword) {
				// Weight keywords based on where they appear
				if strings.Contains(strings.ToLower(result.Title), keyword) {
					score += 2.0 // Title matches are more important
				} else {
					score += 1.0 // Snippet matches
				}
			}
		}

		// Normalize score by number of keywords to avoid bias toward categories with more keywords
		if len(keywords) > 0 {
			score = score / float64(len(keywords))
		}

		if score > bestScore {
			bestScore = score
			bestCategory = i
		}
	}

	// Only assign if we have a meaningful match (threshold)
	if bestScore > 0.1 {
		return bestCategory
	}

	return -1 // No good category match
}

// calculateClusterMetrics computes quality and density metrics for each cluster
func (rc *ResultClusterer) calculateClusterMetrics(categories []ClusterCategory) {
	for i := range categories {
		category := &categories[i]

		if len(category.Results) == 0 {
			category.Quality = 0.0
			category.Density = 0.0
			continue
		}

		// Calculate average relevance score
		totalRelevance := 0.0
		for _, result := range category.Results {
			totalRelevance += result.Relevance
		}
		category.Quality = totalRelevance / float64(len(category.Results))

		// Calculate density (how well-populated this cluster is)
		// Higher density for clusters with more high-quality results
		highQualityCount := 0
		for _, result := range category.Results {
			if result.Relevance > 0.6 {
				highQualityCount++
			}
		}

		if len(category.Results) > 0 {
			category.Density = float64(highQualityCount) / float64(len(category.Results))
		}
	}
}

// identifyCoverageGaps analyzes clusters to identify missing information areas
func (rc *ResultClusterer) identifyCoverageGaps(ctx context.Context, query string, categories []ClusterCategory) []string {
	var gaps []string

	// Check for empty or low-quality categories
	for _, category := range categories {
		if len(category.Results) == 0 {
			gaps = append(gaps, fmt.Sprintf("No %s information found", category.Name))
		} else if category.Quality < 0.5 {
			gaps = append(gaps, fmt.Sprintf("Limited high-quality %s content", category.Name))
		}
	}

	// Check for specific information types that should be present
	expectedContent := map[string][]string{
		"technical":   {"api", "documentation", "architecture", "performance"},
		"competitive": {"comparison", "alternative", "vs", "competitor"},
		"practical":   {"example", "tutorial", "guide", "implementation"},
	}

	for contentType, keywords := range expectedContent {
		found := false
		for _, category := range categories {
			for _, result := range category.Results {
				text := strings.ToLower(result.Title + " " + result.Snippet)
				for _, keyword := range keywords {
					if strings.Contains(text, keyword) {
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			gaps = append(gaps, fmt.Sprintf("Missing %s information", contentType))
		}
	}

	// Limit gaps to most important ones
	if len(gaps) > 5 {
		gaps = gaps[:5]
	}

	return gaps
}
