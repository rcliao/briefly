// Package themes provides theme classification for articles (Phase 0)
package themes

import (
	"briefly/internal/core"
	"briefly/internal/llm"
	"briefly/internal/observability"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// Classifier classifies articles into themes using LLM
type Classifier struct {
	llmClient *llm.TracedClient
	posthog   *observability.PostHogClient
}

// NewClassifier creates a new theme classifier
func NewClassifier(llmClient *llm.TracedClient, posthog *observability.PostHogClient) *Classifier {
	return &Classifier{
		llmClient: llmClient,
		posthog:   posthog,
	}
}

// ClassificationResult contains the results of theme classification
type ClassificationResult struct {
	ThemeID        string  // ID of the matched theme
	ThemeName      string  // Name of the matched theme
	RelevanceScore float64 // Relevance score (0.0-1.0)
	Reasoning      string  // Why this theme was chosen
}

// ClassifyArticle classifies an article against a list of themes
// Returns a map of theme_id -> relevance_score for all themes above the minimum threshold
func (c *Classifier) ClassifyArticle(ctx context.Context, article core.Article, themes []core.Theme, minRelevance float64) ([]ClassificationResult, error) {
	if len(themes) == 0 {
		return []ClassificationResult{}, nil
	}

	// Build the classification prompt
	prompt := c.buildClassificationPrompt(article, themes)

	// Use the LLM to classify
	response, err := c.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		Temperature: 0.3, // Low temperature for more consistent classification
		MaxTokens:   1000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to classify article: %w", err)
	}

	// Parse the LLM response
	results, err := c.parseClassificationResponse(response, themes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse classification response: %w", err)
	}

	// Filter by minimum relevance
	var filtered []ClassificationResult
	for _, result := range results {
		if result.RelevanceScore >= minRelevance {
			filtered = append(filtered, result)

			// Track classification in PostHog
			if c.posthog != nil && c.posthog.IsEnabled() {
				_ = c.posthog.TrackThemeClassification(ctx, article.ID, result.ThemeName, result.RelevanceScore)
			}
		}
	}

	return filtered, nil
}

// GetBestMatch returns the single best matching theme for an article
func (c *Classifier) GetBestMatch(ctx context.Context, article core.Article, themes []core.Theme, minRelevance float64) (*ClassificationResult, error) {
	results, err := c.ClassifyArticle(ctx, article, themes, minRelevance)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return nil, nil // No match above threshold
	}

	// Return the highest scoring theme
	best := results[0]
	for _, result := range results[1:] {
		if result.RelevanceScore > best.RelevanceScore {
			best = result
		}
	}

	return &best, nil
}

// buildClassificationPrompt creates the prompt for LLM classification
func (c *Classifier) buildClassificationPrompt(article core.Article, themes []core.Theme) string {
	var sb strings.Builder

	sb.WriteString("You are a content classification expert. Analyze the following article and determine which theme(s) it belongs to.\n\n")

	sb.WriteString("ARTICLE:\n")
	sb.WriteString("Title: ")
	sb.WriteString(article.Title)
	sb.WriteString("\n\n")

	// Truncate content if too long (keep first 2000 chars)
	content := article.CleanedText
	if len(content) > 2000 {
		content = content[:2000] + "..."
	}
	sb.WriteString("Content: ")
	sb.WriteString(content)
	sb.WriteString("\n\n")

	sb.WriteString("AVAILABLE THEMES:\n")
	for i, theme := range themes {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, theme.Name))
		if theme.Description != "" {
			sb.WriteString(fmt.Sprintf("   Description: %s\n", theme.Description))
		}
		if len(theme.Keywords) > 0 {
			sb.WriteString(fmt.Sprintf("   Keywords: %s\n", strings.Join(theme.Keywords, ", ")))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("TASK:\n")
	sb.WriteString("For each theme, provide a relevance score from 0.0 to 1.0 indicating how well the article matches that theme.\n")
	sb.WriteString("Also provide a brief reasoning for your classification.\n\n")

	sb.WriteString("Respond in JSON format:\n")
	sb.WriteString("{\n")
	sb.WriteString("  \"classifications\": [\n")
	sb.WriteString("    {\n")
	sb.WriteString("      \"theme_name\": \"Theme Name\",\n")
	sb.WriteString("      \"relevance_score\": 0.85,\n")
	sb.WriteString("      \"reasoning\": \"Brief explanation\"\n")
	sb.WriteString("    }\n")
	sb.WriteString("  ]\n")
	sb.WriteString("}\n\n")

	sb.WriteString("Important:\n")
	sb.WriteString("- Only include themes with relevance_score > 0.1\n")
	sb.WriteString("- Be honest and conservative with scores\n")
	sb.WriteString("- Consider both the title and content\n")
	sb.WriteString("- Match against theme keywords and descriptions\n")

	return sb.String()
}

// parseClassificationResponse parses the LLM JSON response into results
func (c *Classifier) parseClassificationResponse(response string, themes []core.Theme) ([]ClassificationResult, error) {
	// Extract JSON from response (sometimes LLMs add markdown code blocks)
	response = strings.TrimSpace(response)
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	// Parse JSON
	var parsed struct {
		Classifications []struct {
			ThemeName      string  `json:"theme_name"`
			RelevanceScore float64 `json:"relevance_score"`
			Reasoning      string  `json:"reasoning"`
		} `json:"classifications"`
	}

	if err := json.Unmarshal([]byte(response), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w\nResponse: %s", err, response)
	}

	// Map theme names to theme IDs
	themeMap := make(map[string]core.Theme)
	for _, theme := range themes {
		themeMap[theme.Name] = theme
		// Also support lowercase matching
		themeMap[strings.ToLower(theme.Name)] = theme
	}

	// Build results
	var results []ClassificationResult
	for _, classification := range parsed.Classifications {
		// Find matching theme
		theme, ok := themeMap[classification.ThemeName]
		if !ok {
			// Try lowercase
			theme, ok = themeMap[strings.ToLower(classification.ThemeName)]
			if !ok {
				// Skip unknown themes
				continue
			}
		}

		results = append(results, ClassificationResult{
			ThemeID:        theme.ID,
			ThemeName:      theme.Name,
			RelevanceScore: classification.RelevanceScore,
			Reasoning:      classification.Reasoning,
		})
	}

	return results, nil
}

// ClassifyBatch classifies multiple articles in batch
func (c *Classifier) ClassifyBatch(ctx context.Context, articles []core.Article, themes []core.Theme, minRelevance float64) (map[string][]ClassificationResult, error) {
	results := make(map[string][]ClassificationResult)

	for _, article := range articles {
		classifications, err := c.ClassifyArticle(ctx, article, themes, minRelevance)
		if err != nil {
			// Log error but continue with other articles
			fmt.Printf("Warning: Failed to classify article %s: %v\n", article.ID, err)
			continue
		}

		if len(classifications) > 0 {
			results[article.ID] = classifications
		}
	}

	return results, nil
}
