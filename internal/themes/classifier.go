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

	"google.golang.org/genai"
)

// LLMClient interface for theme classification
type LLMClient interface {
	GenerateText(ctx context.Context, prompt string, options llm.TextGenerationOptions) (string, error)
}

// PostHogTracker interface for analytics tracking
type PostHogTracker interface {
	IsEnabled() bool
	TrackThemeClassification(ctx context.Context, articleID string, themeName string, relevance float64) error
}

// Classifier classifies articles into themes using LLM
type Classifier struct {
	llmClient LLMClient
	posthog   PostHogTracker
}

// NewClassifier creates a new theme classifier
func NewClassifier(llmClient LLMClient, posthog PostHogTracker) *Classifier {
	return &Classifier{
		llmClient: llmClient,
		posthog:   posthog,
	}
}

// NewClassifierWithClients creates a new theme classifier with concrete types (convenience method)
func NewClassifierWithClients(llmClient *llm.TracedClient, posthog *observability.PostHogClient) *Classifier {
	return NewClassifier(llmClient, posthog)
}

// ClassificationResult contains the results of theme classification
type ClassificationResult struct {
	ThemeID        string  // ID of the matched theme
	ThemeName      string  // Name of the matched theme
	RelevanceScore float64 // Relevance score (0.0-1.0)
	Reasoning      string  // Why this theme was chosen
	ReaderIntent   string  // Reader intent: "skim", "read", or "deep_dive"
}

// Reader intent constants
const (
	IntentSkim     = "skim"      // Industry news, announcements - just know it exists
	IntentRead     = "read"      // Practical tools, techniques - actionable for engineers
	IntentDeepDive = "deep_dive" // Research, architecture - optional for specialists
)

// Interface methods for sources.ThemeClassificationResult compatibility
func (c *ClassificationResult) GetThemeID() string {
	if c == nil {
		return ""
	}
	return c.ThemeID
}

func (c *ClassificationResult) GetThemeName() string {
	if c == nil {
		return ""
	}
	return c.ThemeName
}

func (c *ClassificationResult) GetRelevanceScore() float64 {
	if c == nil {
		return 0.0
	}
	return c.RelevanceScore
}

func (c *ClassificationResult) GetReasoning() string {
	if c == nil {
		return ""
	}
	return c.Reasoning
}

// CreateClassificationSchema creates a Gemini response schema for theme classification
// This ensures the LLM returns properly structured JSON without parsing issues
func CreateClassificationSchema() *genai.Schema {
	return &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"reader_intent": {
				Type:        genai.TypeString,
				Description: "Reader intent: 'skim' (news/announcements - just know it exists), 'read' (practical tools/techniques - actionable), or 'deep_dive' (research/architecture - optional for specialists)",
				Enum:        []string{"skim", "read", "deep_dive"},
			},
			"classifications": {
				Type:        genai.TypeArray,
				Description: "List of theme classifications with relevance scores",
				Items: &genai.Schema{
					Type: genai.TypeObject,
					Properties: map[string]*genai.Schema{
						"theme_name": {
							Type:        genai.TypeString,
							Description: "Exact name of the theme from the provided list",
						},
						"relevance_score": {
							Type:        genai.TypeNumber,
							Description: "Relevance score from 0.0 to 1.0 indicating how well the article matches this theme",
						},
						"reasoning": {
							Type:        genai.TypeString,
							Description: "Brief explanation (1-2 sentences) for why this theme was assigned with this score",
						},
					},
					Required: []string{"theme_name", "relevance_score", "reasoning"},
				},
			},
		},
		Required: []string{"reader_intent", "classifications"},
	}
}

// ClassifyArticle classifies an article against a list of themes
// Returns a map of theme_id -> relevance_score for all themes above the minimum threshold
func (c *Classifier) ClassifyArticle(ctx context.Context, article core.Article, themes []core.Theme, minRelevance float64) ([]ClassificationResult, error) {
	if len(themes) == 0 {
		return []ClassificationResult{}, nil
	}

	// Build the classification prompt
	prompt := c.buildClassificationPrompt(article, themes)

	// Create schema for structured output
	schema := CreateClassificationSchema()

	// Use the LLM to classify with structured output (guaranteed valid JSON)
	response, err := c.llmClient.GenerateText(ctx, prompt, llm.TextGenerationOptions{
		Temperature:    0.3, // Low temperature for more consistent classification
		MaxTokens:      2000, // Increased to ensure complete JSON output
		ResponseSchema: schema, // Phase 1: Structured output eliminates JSON parsing errors
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

	sb.WriteString("You are a content classification expert helping senior software engineers stay current on GenAI.\n")
	sb.WriteString("Analyze the following article and determine its theme(s) AND reader intent.\n\n")

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

	sb.WriteString("READER INTENT DEFINITIONS:\n")
	sb.WriteString("- \"skim\": Industry news, announcements, partnerships, funding rounds.\n")
	sb.WriteString("  Reader just needs to know it exists. Headline + 1-sentence context is enough.\n")
	sb.WriteString("  Examples: Company X raises $100M, Partnership announced, New product launched\n\n")
	sb.WriteString("- \"read\": Practical tools, techniques, tutorials, actionable for engineers.\n")
	sb.WriteString("  Reader should spend 5-10 minutes understanding the details.\n")
	sb.WriteString("  Examples: New coding tool released, How-to guide, Best practices article\n\n")
	sb.WriteString("- \"deep_dive\": Research papers, architectural deep-dives, technical analysis.\n")
	sb.WriteString("  Optional for specialists. Reader may want to bookmark for weekend reading.\n")
	sb.WriteString("  Examples: Academic paper, System design breakdown, In-depth technical analysis\n\n")

	sb.WriteString("TASK:\n")
	sb.WriteString("1. Determine the READER INTENT (skim, read, or deep_dive) - think about how a busy senior engineer would consume this\n")
	sb.WriteString("2. For each theme, provide a relevance score (0.0-1.0) and brief reasoning\n\n")

	sb.WriteString("Guidelines:\n")
	sb.WriteString("- Only include themes with relevance_score > 0.1\n")
	sb.WriteString("- Be honest and conservative with scores\n")
	sb.WriteString("- Consider both the title and content\n")
	sb.WriteString("- Match against theme keywords and descriptions\n")
	sb.WriteString("- Use the exact theme name from the list above\n")
	sb.WriteString("- For intent: Consider article length, depth, and practical applicability\n\n")

	sb.WriteString("OUTPUT FORMAT:\n")
	sb.WriteString("Provide your response as structured JSON following the schema.\n")

	return sb.String()
}

// parseClassificationResponse parses the LLM JSON response into results
// With structured output (ResponseSchema), the response is guaranteed to be valid JSON
func (c *Classifier) parseClassificationResponse(response string, themes []core.Theme) ([]ClassificationResult, error) {
	// Strip markdown code blocks if present (for backward compatibility and testing)
	cleanResponse := strings.TrimSpace(response)
	if strings.HasPrefix(cleanResponse, "```json") {
		// Remove ```json prefix and ``` suffix
		cleanResponse = strings.TrimPrefix(cleanResponse, "```json")
		cleanResponse = strings.TrimPrefix(cleanResponse, "```")
		cleanResponse = strings.TrimSuffix(cleanResponse, "```")
		cleanResponse = strings.TrimSpace(cleanResponse)
	}

	var parsed struct {
		ReaderIntent    string `json:"reader_intent"`
		Classifications []struct {
			ThemeName      string  `json:"theme_name"`
			RelevanceScore float64 `json:"relevance_score"`
			Reasoning      string  `json:"reasoning"`
		} `json:"classifications"`
	}

	if err := json.Unmarshal([]byte(cleanResponse), &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w\nResponse: %s", err, response)
	}

	// Validate and normalize reader intent
	readerIntent := strings.ToLower(strings.TrimSpace(parsed.ReaderIntent))
	if readerIntent != IntentSkim && readerIntent != IntentRead && readerIntent != IntentDeepDive {
		// Default to "read" if invalid
		readerIntent = IntentRead
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
			ReaderIntent:   readerIntent, // Include intent with each result
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
