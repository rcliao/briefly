// Package tags provides multi-label tag classification for articles (Phase 1)
// Tags enable fine-grained clustering within themes (5 themes → 50+ tags)
package tags

import (
	"briefly/internal/core"
	"briefly/internal/llm"
	"briefly/internal/observability"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

// LLMClient interface for tag classification
type LLMClient interface {
	GenerateText(ctx context.Context, prompt string, options llm.TextGenerationOptions) (string, error)
}

// PostHogTracker interface for analytics tracking
type PostHogTracker interface {
	IsEnabled() bool
	TrackEvent(ctx context.Context, event string, properties map[string]interface{}) error
}

// Classifier classifies articles into multiple tags using LLM (multi-label classification)
type Classifier struct {
	llmClient LLMClient
	posthog   PostHogTracker
}

// NewClassifier creates a new tag classifier
func NewClassifier(llmClient LLMClient, posthog PostHogTracker) *Classifier {
	return &Classifier{
		llmClient: llmClient,
		posthog:   posthog,
	}
}

// NewClassifierWithClients creates a new tag classifier with concrete types (convenience method)
func NewClassifierWithClients(llmClient *llm.TracedClient, posthog *observability.PostHogClient) *Classifier {
	return NewClassifier(llmClient, posthog)
}

// TagClassificationResult contains a single tag classification
type TagClassificationResult struct {
	TagID          string  // ID of the matched tag (e.g., "tag-llm")
	TagName        string  // Name of the matched tag (e.g., "Large Language Models")
	RelevanceScore float64 // Relevance score (0.0-1.0)
	Reasoning      string  // Why this tag was chosen
}

// ClassificationResult contains all tag classifications for an article
type ClassificationResult struct {
	ArticleID string                    // Article being classified
	Tags      []TagClassificationResult // Assigned tags (3-5 recommended)
	ThemeID   string                    // Parent theme (for filtering)
}

// CreateTagClassificationSchema creates a Gemini response schema for multi-label tag classification
// This ensures the LLM returns properly structured JSON without parsing issues
func CreateTagClassificationSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"tags": {
				Type:        genai.TypeArray,
				Description: "List of 3-5 most relevant tags with scores (multi-label classification)",
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"tag_name": {
							Type:        genai.TypeString,
							Description: "Exact name of the tag from the provided list",
						},
						"relevance_score": {
							Type:        genai.TypeNumber,
							Description: "Relevance score from 0.0 to 1.0 indicating tag relevance",
						},
						"reasoning": {
							Type:        genai.TypeString,
							Description: "Brief explanation (1 sentence) for why this tag was assigned",
						},
					},
					Required: []string{"tag_name", "relevance_score", "reasoning"},
				},
			},
		},
		Required: []string{"tags"},
	}
}

// ClassifyArticle classifies an article with multiple tags (multi-label)
// Returns 3-5 most relevant tags above the minimum threshold
// Uses article summary (not full content) for better classification accuracy
func (c *Classifier) ClassifyArticle(ctx context.Context, article core.Article, summary *core.Summary, tags []core.Tag, minRelevance float64) (*ClassificationResult, error) {
	if len(tags) == 0 {
		return &ClassificationResult{
			ArticleID: article.ID,
			Tags:      []TagClassificationResult{},
		}, nil
	}

	// Build the classification prompt (uses summary for better accuracy)
	prompt := c.buildClassificationPrompt(article, summary, tags)

	// Create schema for structured output
	schema := CreateTagClassificationSchema()

	// Use the LLM to classify with structured output (guaranteed valid JSON)
	response, err := c.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		Temperature:    0.3, // Low temperature for consistent classification
		MaxTokens:      1500,
		ResponseSchema: schema, // Structured output eliminates JSON parsing errors
	})
	if err != nil {
		return nil, fmt.Errorf("failed to classify article tags: %w", err)
	}

	// Parse the LLM response
	results, err := c.parseClassificationResponse(response, tags)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tag classification response: %w", err)
	}

	// Filter by minimum relevance and limit to top 5 tags
	var filtered []TagClassificationResult
	for _, result := range results {
		if result.RelevanceScore >= minRelevance && len(filtered) < 5 {
			filtered = append(filtered, result)

			// Track tag classification in PostHog
			if c.posthog != nil && c.posthog.IsEnabled() {
				_ = c.posthog.TrackEvent(ctx, "tag_classification", map[string]interface{}{
					"article_id":      article.ID,
					"tag_id":          result.TagID,
					"tag_name":        result.TagName,
					"relevance_score": result.RelevanceScore,
					"theme_id":        article.ThemeID,
				})
			}
		}
	}

	return &ClassificationResult{
		ArticleID: article.ID,
		Tags:      filtered,
		ThemeID:   getThemeID(article),
	}, nil
}

// ClassifyWithinTheme classifies an article using only tags from a specific theme
// This enables hierarchical classification: theme → tags within theme
func (c *Classifier) ClassifyWithinTheme(ctx context.Context, article core.Article, summary *core.Summary, themeID string, allTags []core.Tag, minRelevance float64) (*ClassificationResult, error) {
	// Filter tags to only those belonging to the theme
	var themeTags []core.Tag
	for _, tag := range allTags {
		if tag.ThemeID != nil && *tag.ThemeID == themeID {
			themeTags = append(themeTags, tag)
		}
	}

	if len(themeTags) == 0 {
		// No tags for this theme, return empty result
		return &ClassificationResult{
			ArticleID: article.ID,
			Tags:      []TagClassificationResult{},
			ThemeID:   themeID,
		}, nil
	}

	// Classify using filtered tags
	result, err := c.ClassifyArticle(ctx, article, summary, themeTags, minRelevance)
	if err != nil {
		return nil, err
	}

	result.ThemeID = themeID
	return result, nil
}

// GetTagMap returns a map of tag_id -> relevance_score for easy database storage
func (c *ClassificationResult) GetTagMap() map[string]float64 {
	tagMap := make(map[string]float64)
	for _, tag := range c.Tags {
		tagMap[tag.TagID] = tag.RelevanceScore
	}
	return tagMap
}

// buildClassificationPrompt creates the prompt for multi-label tag classification
// Uses article summary (not full content) for better signal-to-noise ratio
func (c *Classifier) buildClassificationPrompt(article core.Article, summary *core.Summary, tags []core.Tag) string {
	var sb strings.Builder

	sb.WriteString("You are an expert at multi-label content tagging. Analyze the article and assign 3-5 most relevant tags.\n\n")

	sb.WriteString("ARTICLE:\n")
	sb.WriteString("Title: ")
	sb.WriteString(article.Title)
	sb.WriteString("\n\n")

	// Use summary if available (better signal), otherwise truncate content
	var contentForClassification string
	if summary != nil && summary.SummaryText != "" {
		contentForClassification = summary.SummaryText
	} else {
		contentForClassification = article.CleanedText
		if len(contentForClassification) > 2000 {
			contentForClassification = contentForClassification[:2000] + "..."
		}
	}

	sb.WriteString("Summary: ")
	sb.WriteString(contentForClassification)
	sb.WriteString("\n\n")

	sb.WriteString("AVAILABLE TAGS:\n")
	for i, tag := range tags {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, tag.Name))
		if tag.Description != "" {
			sb.WriteString(fmt.Sprintf("   Description: %s\n", tag.Description))
		}
		if len(tag.Keywords) > 0 {
			sb.WriteString(fmt.Sprintf("   Keywords: %s\n", strings.Join(tag.Keywords, ", ")))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("TASK:\n")
	sb.WriteString("Select 3-5 tags that best describe this article, with relevance scores.\n\n")

	sb.WriteString("GUIDELINES:\n")
	sb.WriteString("- Return exactly 3-5 tags (multi-label classification)\n")
	sb.WriteString("- Prioritize tags with the highest relevance to the article's main topic\n")
	sb.WriteString("- Relevance scores should be between 0.0 and 1.0\n")
	sb.WriteString("- The top tag should have a score ≥ 0.7 if it's a strong match\n")
	sb.WriteString("- Only include tags with relevance_score ≥ 0.4\n")
	sb.WriteString("- Use exact tag names from the list above\n")
	sb.WriteString("- Match against tag keywords, description, and article content\n")
	sb.WriteString("- Be specific: prefer granular tags over generic ones\n\n")

	sb.WriteString("OUTPUT FORMAT:\n")
	sb.WriteString("Return JSON with 3-5 tags, ordered by relevance (highest first).\n")

	return sb.String()
}

// parseClassificationResponse parses the LLM JSON response into tag results
// With structured output (ResponseSchema), the response is guaranteed to be valid JSON
func (c *Classifier) parseClassificationResponse(response string, tags []core.Tag) ([]TagClassificationResult, error) {
	// Strip markdown code blocks if present (backward compatibility)
	cleanResponse := strings.TrimSpace(response)
	if strings.HasPrefix(cleanResponse, "```json") {
		cleanResponse = strings.TrimPrefix(cleanResponse, "```json")
		cleanResponse = strings.TrimPrefix(cleanResponse, "```")
		cleanResponse = strings.TrimSuffix(cleanResponse, "```")
		cleanResponse = strings.TrimSpace(cleanResponse)
	}

	var parsed struct {
		Tags []struct {
			TagName        string  `json:"tag_name"`
			RelevanceScore float64 `json:"relevance_score"`
			Reasoning      string  `json:"reasoning"`
		} `json:"tags"`
	}

	if err := json.Unmarshal([]byte(cleanResponse), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w\nResponse: %s", err, response)
	}

	// Map tag names to tag objects
	tagMap := make(map[string]core.Tag)
	for _, tag := range tags {
		tagMap[tag.Name] = tag
		// Also support lowercase and normalized matching
		tagMap[strings.ToLower(tag.Name)] = tag
		tagMap[normalizeTagName(tag.Name)] = tag
	}

	// Build results
	var results []TagClassificationResult
	for _, classification := range parsed.Tags {
		// Find matching tag
		tag, ok := tagMap[classification.TagName]
		if !ok {
			// Try lowercase
			tag, ok = tagMap[strings.ToLower(classification.TagName)]
			if !ok {
				// Try normalized
				tag, ok = tagMap[normalizeTagName(classification.TagName)]
				if !ok {
					// Skip unknown tags
					continue
				}
			}
		}

		results = append(results, TagClassificationResult{
			TagID:          tag.ID,
			TagName:        tag.Name,
			RelevanceScore: classification.RelevanceScore,
			Reasoning:      classification.Reasoning,
		})
	}

	return results, nil
}

// ClassifyBatch classifies multiple articles in batch (multi-label for each)
func (c *Classifier) ClassifyBatch(ctx context.Context, articles []core.Article, summaries map[string]*core.Summary, tags []core.Tag, minRelevance float64) (map[string]*ClassificationResult, error) {
	results := make(map[string]*ClassificationResult)

	for _, article := range articles {
		summary := summaries[article.ID]
		classification, err := c.ClassifyArticle(ctx, article, summary, tags, minRelevance)
		if err != nil {
			// Log error but continue with other articles
			fmt.Printf("Warning: Failed to classify article %s: %v\n", article.ID, err)
			continue
		}

		if len(classification.Tags) > 0 {
			results[article.ID] = classification
		}
	}

	return results, nil
}

// ClassifyBatchWithinTheme classifies multiple articles within a specific theme
// Enables hierarchical batch processing: group by theme → classify with theme tags
func (c *Classifier) ClassifyBatchWithinTheme(ctx context.Context, articles []core.Article, summaries map[string]*core.Summary, themeID string, allTags []core.Tag, minRelevance float64) (map[string]*ClassificationResult, error) {
	results := make(map[string]*ClassificationResult)

	for _, article := range articles {
		summary := summaries[article.ID]
		classification, err := c.ClassifyWithinTheme(ctx, article, summary, themeID, allTags, minRelevance)
		if err != nil {
			fmt.Printf("Warning: Failed to classify article %s within theme %s: %v\n", article.ID, themeID, err)
			continue
		}

		if len(classification.Tags) > 0 {
			results[article.ID] = classification
		}
	}

	return results, nil
}

// Helper functions

// getThemeID safely extracts theme ID from article
func getThemeID(article core.Article) string {
	if article.ThemeID != nil {
		return *article.ThemeID
	}
	return ""
}

// normalizeTagName normalizes tag names for matching (e.g., "RAG & Retrieval" → "rag retrieval")
func normalizeTagName(name string) string {
	// Remove special characters and convert to lowercase
	normalized := strings.ToLower(name)
	normalized = strings.ReplaceAll(normalized, "&", "")
	normalized = strings.ReplaceAll(normalized, "-", " ")
	normalized = strings.TrimSpace(normalized)
	// Collapse multiple spaces
	for strings.Contains(normalized, "  ") {
		normalized = strings.ReplaceAll(normalized, "  ", " ")
	}
	return normalized
}
