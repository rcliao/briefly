package pipeline

import (
	"briefly/internal/core"
	"briefly/internal/llm"
	"briefly/internal/observability"
	"briefly/internal/persistence"
	"briefly/internal/themes"
	"context"
	"fmt"
)

// ThemeCategorizer classifies articles into themes using LLM-based classification
type ThemeCategorizer struct {
	db         persistence.Database
	classifier *themes.Classifier
}

// NewThemeCategorizer creates a new theme-based categorizer
func NewThemeCategorizer(db persistence.Database, llmClient *llm.TracedClient, posthog *observability.PostHogClient) *ThemeCategorizer {
	return &ThemeCategorizer{
		db:         db,
		classifier: themes.NewClassifierWithClients(llmClient, posthog),
	}
}

// CategorizeArticle classifies an article into a theme category
// This implements the simple ArticleCategorizer interface (returns string)
// but also enriches the article with theme_id and relevance_score
func (tc *ThemeCategorizer) CategorizeArticle(ctx context.Context, article *core.Article, summary *core.Summary) (string, error) {
	// Get all enabled themes from database
	enabledThemes, err := tc.db.Themes().ListEnabled(ctx)
	if err != nil {
		return "Uncategorized", fmt.Errorf("failed to fetch themes: %w", err)
	}

	if len(enabledThemes) == 0 {
		// No themes configured, return default
		return "Uncategorized", nil
	}

	// Use the classifier to get the best matching theme
	// Minimum relevance threshold of 0.4 (40%)
	bestMatch, err := tc.classifier.GetBestMatch(ctx, *article, enabledThemes, 0.4)
	if err != nil {
		return "Uncategorized", fmt.Errorf("failed to classify article: %w", err)
	}

	if bestMatch == nil {
		// No theme matched above threshold
		return "Uncategorized", nil
	}

	// Enrich the article with theme information
	// This will be stored in the database when the article is saved
	article.TopicCluster = bestMatch.ThemeName // Use existing field for category
	// Note: We'll need to add theme_id and theme_relevance_score fields to articles table
	// For now, store the score in ClusterConfidence
	article.ClusterConfidence = bestMatch.RelevanceScore

	// Return the theme name for the digest categorization
	return bestMatch.ThemeName, nil
}

// CategorizeArticles processes multiple articles in batch
// This is a helper method for pipeline efficiency
func (tc *ThemeCategorizer) CategorizeArticles(ctx context.Context, articles []core.Article, summaries []core.Summary) ([]core.Article, error) {
	// Get all enabled themes once
	enabledThemes, err := tc.db.Themes().ListEnabled(ctx)
	if err != nil {
		return articles, fmt.Errorf("failed to fetch themes: %w", err)
	}

	if len(enabledThemes) == 0 {
		// No themes configured, return articles as-is
		return articles, nil
	}

	// Build a map of article ID to summary for quick lookup
	summaryMap := make(map[string]*core.Summary)
	for i := range summaries {
		for _, articleID := range summaries[i].ArticleIDs {
			summaryMap[articleID] = &summaries[i]
		}
	}

	// Classify each article
	for i := range articles {
		article := &articles[i]

		// Get corresponding summary (if exists)
		summary := summaryMap[article.ID]
		if summary == nil {
			// Skip articles without summaries
			article.TopicCluster = "Uncategorized"
			continue
		}

		// Classify the article
		bestMatch, err := tc.classifier.GetBestMatch(ctx, *article, enabledThemes, 0.4)
		if err != nil {
			// Log error but continue with other articles
			fmt.Printf("   ⚠️  Failed to classify article %s: %v\n", article.Title, err)
			article.TopicCluster = "Uncategorized"
			continue
		}

		if bestMatch == nil {
			// No theme matched above threshold
			article.TopicCluster = "Uncategorized"
			continue
		}

		// Enrich the article with theme information
		article.TopicCluster = bestMatch.ThemeName
		article.ClusterConfidence = bestMatch.RelevanceScore
	}

	return articles, nil
}
