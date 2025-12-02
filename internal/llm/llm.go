package llm

import (
	"briefly/internal/core"
	"context"
	"fmt"
	"math"
	"os" // Added to fetch API key from environment variable
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
	"google.golang.org/genai"
)

const (
	// DefaultModel is the default Gemini model to use for summarization.
	DefaultModel = "gemini-flash-lite-latest" // Gemini Flash Lite (latest version)
	// DefaultEmbeddingModel is the default model for generating embeddings
	DefaultEmbeddingModel = "gemini-embedding-001"
	// DefaultEmbeddingDimensions is the output dimension for embeddings (Matryoshka)
	DefaultEmbeddingDimensions = int32(768)
	// SummarizeTextPromptTemplate is the template for the summarization prompt.
	SummarizeTextPromptTemplate = "Please summarize the following text concisely:\n\n---\n%s\n---"
	// SummarizeTextWithFormatPromptTemplate is the template for format-aware summarization.
	SummarizeTextWithFormatPromptTemplate = `Summarize the following text for a %s format. Focus on what matters and why it's relevant. Write only the summary, no meta-commentary or format explanations.

FORMAT REQUIREMENTS:
- Brief: 50-100 words, essential information only
- Standard: 100-200 words, key points with context  
- Detailed: 200-300 words, comprehensive analysis
- Newsletter: 150-250 words, engaging and shareable
- Scannable: 20-40 words, one complete sentence focusing on the core insight

Text to summarize:
%s`
)

// Client represents a client for interacting with an LLM.
// It can be expanded to include more configuration or methods.
type Client struct {
	apiKey    string
	modelName string
	gClient   *genai.Client // Store the main client (new SDK)
}

// TextGenerationOptions contains options for text generation
type TextGenerationOptions struct {
	MaxTokens      int32        // Maximum number of tokens to generate
	Temperature    float32      // Temperature for randomness (0.0 to 1.0)
	Model          string       // Model to use (optional, defaults to client's model)
	ResponseSchema *genai.Schema // Optional: Schema for structured output (Phase 1)
}

// NewClient creates a new LLM client.
// It supports multiple ways to get the API key (in order of preference):
// 1. Environment variable: GEMINI_API_KEY (or alternatives)
// 2. Viper configuration: gemini.api_key
func NewClient(modelName string) (*Client, error) {
	// Try to get API key from multiple sources for backward compatibility
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		// Try alternative environment variable names
		if apiKey = os.Getenv("GOOGLE_GEMINI_API_KEY"); apiKey == "" {
			if apiKey = os.Getenv("GOOGLE_AI_API_KEY"); apiKey == "" {
				apiKey = viper.GetString("gemini.api_key")
			}
		}
	}
	if apiKey == "" {
		return nil, fmt.Errorf("gemini API key is required. Set GEMINI_API_KEY environment variable or gemini.api_key in config file.\nGet your API key from: https://makersuite.google.com/app/apikey")
	}

	// Get model name from parameter, viper config, or default
	if modelName == "" {
		modelName = viper.GetString("gemini.model")
		if modelName == "" {
			modelName = DefaultModel
		}
	}

	ctx := context.Background()
	gClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	return &Client{
		apiKey:    apiKey,
		modelName: modelName,
		gClient:   gClient,
	}, nil
}

// generateContent is a helper that wraps the new SDK's GenerateContent call
func (c *Client) generateContent(ctx context.Context, prompt string) (string, error) {
	contents := []*genai.Content{{
		Parts: []*genai.Part{{Text: prompt}},
		Role:  "user",
	}}

	resp, err := c.gClient.Models.GenerateContent(ctx, c.modelName, contents, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	// Use the Text() helper from the new SDK (returns string only)
	text := resp.Text()
	if text == "" {
		return "", fmt.Errorf("empty response from model")
	}

	return text, nil
}

// generateContentWithModel is a helper that uses a specific model
func (c *Client) generateContentWithModel(ctx context.Context, modelName, prompt string) (string, error) {
	contents := []*genai.Content{{
		Parts: []*genai.Part{{Text: prompt}},
		Role:  "user",
	}}

	resp, err := c.gClient.Models.GenerateContent(ctx, modelName, contents, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate content: %w", err)
	}

	text := resp.Text()
	if text == "" {
		return "", fmt.Errorf("empty response from model")
	}

	return text, nil
}

// SummarizeArticleText takes an Article object, extracts its CleanedText,
// and returns a Summary object.
func (c *Client) SummarizeArticleText(article core.Article) (core.Summary, error) {
	if article.CleanedText == "" {
		return core.Summary{}, fmt.Errorf("article ID %s has no CleanedText to summarize", article.ID)
	}

	prompt := fmt.Sprintf(SummarizeTextPromptTemplate, article.CleanedText)

	ctx := context.Background()
	summaryText, err := c.generateContent(ctx, prompt)
	if err != nil {
		return core.Summary{}, fmt.Errorf("failed to generate content for article ID %s: %w", article.ID, err)
	}

	// Populate the Summary struct
	summary := core.Summary{
		ArticleIDs:   []string{article.ID},
		SummaryText:  summaryText,
		ModelUsed:    c.modelName,
		Instructions: SummarizeTextPromptTemplate,
	}

	return summary, nil
}

// SummarizeArticleTextWithFormat takes an Article object, extracts its CleanedText,
// and returns a Summary object with format-specific guidance.
func (c *Client) SummarizeArticleTextWithFormat(article core.Article, format string) (core.Summary, error) {
	if article.CleanedText == "" {
		return core.Summary{}, fmt.Errorf("article ID %s has no CleanedText to summarize", article.ID)
	}

	prompt := fmt.Sprintf(SummarizeTextWithFormatPromptTemplate, format, article.CleanedText)

	ctx := context.Background()
	summaryText, err := c.generateContent(ctx, prompt)
	if err != nil {
		return core.Summary{}, fmt.Errorf("failed to generate content for article ID %s: %w", article.ID, err)
	}

	// Populate the Summary struct
	summary := core.Summary{
		ArticleIDs:   []string{article.ID},
		SummaryText:  summaryText,
		ModelUsed:    c.modelName,
		Instructions: fmt.Sprintf("Format-aware summarization for %s format", format),
	}

	return summary, nil
}

// SummarizeArticleWithKeyMoments creates a summary with key moments in a specific format
func (c *Client) SummarizeArticleWithKeyMoments(article core.Article) (core.Summary, error) {
	if article.CleanedText == "" {
		return core.Summary{}, fmt.Errorf("article ID %s has no CleanedText to summarize", article.ID)
	}

	// Custom prompt for summary with key moments
	keyMomentsPrompt := `Please analyze the following article and provide a summary in this exact format:

# Executive Summary

[Write a concise 2-3 sentence summary that captures the main topic and key takeaway]

# Key Insights

## 💡 [Short descriptive title for first insight]
> "[Select a powerful, concise quote (1-2 sentences max) from the article]"

**Why it matters:** [One clear sentence explaining the significance]

## 📊 [Short descriptive title for second insight]
> "[Another impactful quote from the article]"

**Why it matters:** [One clear sentence explaining the significance]

## 🚀 [Short descriptive title for third insight]
> "[Another important quote from the article]"

**Why it matters:** [One clear sentence explaining the significance]

## 🔍 [Short descriptive title for fourth insight]
> "[Final key quote from the article]"

**Why it matters:** [One clear sentence explaining the significance]

Instructions:
- Use EXACT, concise quotes (1-2 sentences maximum) from the article
- Create descriptive titles that categorize each insight (e.g., "Performance Breakthrough", "New Feature Launch", "Market Impact")
- Keep explanations to one clear, punchy sentence
- Use appropriate emojis for insight categories: 💡 (concepts), 📊 (data/metrics), 🚀 (launches/features), 🔍 (analysis), ⚡ (speed/performance), 🔒 (security/privacy), 💰 (business/cost)
- Select 3-4 most impactful moments that represent different aspects of the story

Article Content:
%s`

	prompt := fmt.Sprintf(keyMomentsPrompt, article.CleanedText)

	ctx := context.Background()
	summaryText, err := c.generateContent(ctx, prompt)
	if err != nil {
		return core.Summary{}, fmt.Errorf("failed to generate content for article ID %s: %w", article.ID, err)
	}

	// Create the Summary object
	summary := core.Summary{
		ArticleIDs:    []string{article.ID},
		SummaryText:   summaryText,
		ModelUsed:     c.modelName,
		Instructions:  "Article summarization with key moments and explanations",
		DateGenerated: time.Now().UTC(),
	}

	return summary, nil
}

// GenerateWhyItMatters generates team context-aware "Why it matters" insights for articles
func (c *Client) GenerateWhyItMatters(articles []core.Article, teamContext string) (map[string]string, error) {
	if len(articles) == 0 {
		return nil, fmt.Errorf("no articles provided")
	}

	// Template for team context-aware insights
	whyItMattersPrompt := `%s

For each link below, provide a one-sentence "Why it matters" that connects to our context:

%s

Format each response as:
"Why it matters: [Your insight here - how it relates to our stack, challenges, or interests]"

Make each insight specific and actionable for our team.`

	// Prepare article list
	var articleList strings.Builder
	for i, article := range articles {
		articleList.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, article.Title))
		articleList.WriteString(fmt.Sprintf("   Summary: %s\n", article.CleanedText[:min(500, len(article.CleanedText))]))
		articleList.WriteString(fmt.Sprintf("   URL: %s\n\n", article.LinkID))
	}

	prompt := fmt.Sprintf(whyItMattersPrompt, teamContext, articleList.String())

	ctx := context.Background()
	content, err := c.generateContent(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate why it matters insights: %w", err)
	}

	// Parse the response to extract insights for each article
	insights := parseWhyItMattersResponse(content, articles)
	return insights, nil
}

// GenerateWhyItMattersSingle generates a "Why it matters" insight for a single article
func (c *Client) GenerateWhyItMattersSingle(article core.Article, teamContext string) (string, error) {
	// Template for single article insight
	singleInsightPrompt := `%s

Article: **%s**
Summary: %s
URL: %s

Provide a one-sentence "Why it matters" insight that connects this article to our team's context:

Format: "Why it matters: [Your specific insight about relevance to our stack, challenges, or interests]"`

	// Truncate content for single article processing
	content := article.CleanedText
	if len(content) > 1000 {
		content = content[:1000] + "..."
	}

	prompt := fmt.Sprintf(singleInsightPrompt, teamContext, article.Title, content, article.LinkID)

	ctx := context.Background()
	responseContent, err := c.generateContent(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate single insight: %w", err)
	}

	// Extract the insight from the response
	insight := extractInsightFromResponse(responseContent)
	return insight, nil
}

// GenerateTeamRelevanceScore generates a relevance score for an article based on team context
func (c *Client) GenerateTeamRelevanceScore(article core.Article, teamContext string) (float64, string, error) {
	relevancePrompt := `%s

Article: **%s**
Summary: %s

Rate this article's relevance to our team on a scale of 0.0 to 1.0, where:
- 0.0-0.3: Low relevance (general interest only)
- 0.4-0.6: Medium relevance (somewhat applicable to our work)  
- 0.7-0.9: High relevance (directly applicable to our challenges/stack)
- 0.9-1.0: Critical relevance (must-read for our current priorities)

Respond in this exact format:
Relevance Score: [0.0-1.0]
Reasoning: [One sentence explaining why this score was assigned]`

	content := article.CleanedText
	if len(content) > 800 {
		content = content[:800] + "..."
	}

	prompt := fmt.Sprintf(relevancePrompt, teamContext, article.Title, content)

	ctx := context.Background()
	responseContent, err := c.generateContent(ctx, prompt)
	if err != nil {
		return 0.0, "", fmt.Errorf("failed to generate relevance score: %w", err)
	}

	// Parse score and reasoning from response
	score, reasoning := parseRelevanceResponse(responseContent)
	return score, reasoning, nil
}

// parseWhyItMattersResponse parses the LLM response to extract insights for each article
func parseWhyItMattersResponse(response string, articles []core.Article) map[string]string {
	insights := make(map[string]string)
	lines := strings.Split(response, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Why it matters:") {
			insight := strings.TrimSpace(strings.TrimPrefix(line, "Why it matters:"))
			// For now, use a simple mapping approach
			// In production, this could be more sophisticated
			if len(articles) > 0 {
				// Map to first unprocessed article
				for _, article := range articles {
					if _, exists := insights[article.ID]; !exists {
						insights[article.ID] = insight
						break
					}
				}
			}
		}
	}

	return insights
}

// extractInsightFromResponse extracts the insight text from the LLM response
func extractInsightFromResponse(response string) string {
	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Why it matters:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Why it matters:"))
		}
	}
	// Fallback: return the whole response if format is unexpected
	return strings.TrimSpace(response)
}

// parseRelevanceResponse parses relevance score and reasoning from LLM response
func parseRelevanceResponse(response string) (float64, string) {
	lines := strings.Split(response, "\n")
	var score float64
	var reasoning string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Relevance Score:") {
			scoreStr := strings.TrimSpace(strings.TrimPrefix(line, "Relevance Score:"))
			if parsedScore, err := strconv.ParseFloat(scoreStr, 64); err == nil {
				score = parsedScore
			}
		} else if strings.HasPrefix(line, "Reasoning:") {
			reasoning = strings.TrimSpace(strings.TrimPrefix(line, "Reasoning:"))
		}
	}

	// Ensure score is within valid range
	if score < 0.0 {
		score = 0.0
	} else if score > 1.0 {
		score = 1.0
	}

	if reasoning == "" {
		reasoning = "No specific reasoning provided"
	}

	return score, reasoning
}

// RegenerateDigestWithMyTake regenerates an entire digest incorporating the user's personal take
func (c *Client) RegenerateDigestWithMyTake(originalDigest, myTake, teamContext, styleGuide string) (string, error) {
	if originalDigest == "" {
		return "", fmt.Errorf("original digest content cannot be empty")
	}

	if myTake == "" {
		return originalDigest, nil // No changes needed
	}

	// Create comprehensive regeneration prompt
	regenerationPrompt := `I need you to regenerate this digest by incorporating my personal take and insights. The goal is to create a cohesive, enhanced version that weaves my perspective throughout rather than just appending it.

ORIGINAL DIGEST:
%s

MY PERSONAL TAKE:
%s

%s

%s

INSTRUCTIONS:
1. Integrate my personal take naturally throughout the digest, not just as an appendix
2. Enhance insights and conclusions based on my perspective
3. Maintain the original structure and format while improving content quality
4. Keep all original links and references intact
5. Make the enhanced digest feel cohesive and authoritative
6. If I've provided contradictory or additional insights, weave them into the relevant sections
7. Enhance the executive summary to reflect my perspective
8. Maintain professional tone while incorporating my voice

Generate the complete enhanced digest that feels like a collaborative effort between AI analysis and human insight.`

	// Prepare context sections
	var contextSection, styleSection string

	if teamContext != "" {
		contextSection = fmt.Sprintf("TEAM CONTEXT:\n%s\n", teamContext)
	}

	if styleGuide != "" {
		styleSection = fmt.Sprintf("STYLE GUIDE:\n%s\n", styleGuide)
	}

	prompt := fmt.Sprintf(regenerationPrompt, originalDigest, myTake, contextSection, styleSection)

	ctx := context.Background()
	content, err := c.generateContent(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to regenerate digest: %w", err)
	}

	return content, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Close cleans up resources used by the client
func (c *Client) Close() {
	// New SDK client doesn't require explicit close
}

// GetGenaiClient returns the underlying genai client for direct use by other packages
func (c *Client) GetGenaiClient() *genai.Client {
	return c.gClient
}

// GetModelName returns the model name used by this client
func (c *Client) GetModelName() string {
	return c.modelName
}

// CategorizeArticle categorizes an article using LLM analysis
func (c *Client) CategorizeArticle(ctx context.Context, article core.Article, categories map[string]Category) (CategoryResult, error) {
	if article.CleanedText == "" && article.Title == "" {
		return CategoryResult{}, fmt.Errorf("article has no content to categorize")
	}

	// Build category descriptions for the prompt
	var categoryDescriptions []string
	for id, category := range categories {
		categoryDescriptions = append(categoryDescriptions,
			fmt.Sprintf("%s (%s %s): %s", id, category.Emoji, category.Name, category.Description))
	}

	prompt := fmt.Sprintf(`Analyze this article and categorize it into one of the following categories. Consider the title, content, urgency, and relevance.

AVAILABLE CATEGORIES:
%s

ARTICLE TO CATEGORIZE:
Title: %s
Content: %s

Respond with EXACTLY this format:
CATEGORY: [category_id]
CONFIDENCE: [0.0-1.0]
REASONING: [brief explanation in one sentence]

Be precise and specific in your categorization.`,
		strings.Join(categoryDescriptions, "\n"),
		article.Title,
		article.CleanedText)

	content, err := c.generateContent(ctx, prompt)
	if err != nil {
		return CategoryResult{}, fmt.Errorf("failed to categorize article: %w", err)
	}

	return parseCategorizeResponse(content, categories)
}

// Category represents a content category (imported from categorization package)
type Category struct {
	ID          string
	Name        string
	Emoji       string
	Priority    int
	Description string
}

// CategoryResult holds categorization results
type CategoryResult struct {
	Category   Category
	Confidence float64
	Reasoning  string
	Source     string
}

// parseCategorizeResponse parses LLM response for article categorization
func parseCategorizeResponse(response string, categories map[string]Category) (CategoryResult, error) {
	lines := strings.Split(response, "\n")

	var categoryID string
	var confidence float64
	var reasoning string

	// Parse each line for the expected format
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "CATEGORY:") {
			categoryID = strings.TrimSpace(strings.TrimPrefix(line, "CATEGORY:"))
		} else if strings.HasPrefix(line, "CONFIDENCE:") {
			confidenceStr := strings.TrimSpace(strings.TrimPrefix(line, "CONFIDENCE:"))
			if parsed, err := strconv.ParseFloat(confidenceStr, 64); err == nil {
				confidence = parsed
			}
		} else if strings.HasPrefix(line, "REASONING:") {
			reasoning = strings.TrimSpace(strings.TrimPrefix(line, "REASONING:"))
		}
	}

	// Validate and get category
	category, exists := categories[categoryID]
	if !exists {
		// Fallback to monitoring category
		category = Category{
			ID:          "monitoring",
			Name:        "Worth Monitoring",
			Emoji:       "🔍",
			Priority:    6,
			Description: "Emerging trends and topics worth exploring",
		}
		confidence = 0.5
		reasoning = "LLM returned unknown category, using fallback"
	}

	// Ensure reasonable confidence bounds
	if confidence < 0.0 {
		confidence = 0.0
	} else if confidence > 1.0 {
		confidence = 1.0
	}

	if reasoning == "" {
		reasoning = "LLM-based categorization"
	}

	return CategoryResult{
		Category:   category,
		Confidence: confidence,
		Reasoning:  reasoning,
		Source:     "llm-based",
	}, nil
}

// GenerateText generates text using the LLM with specified options
func (c *Client) GenerateText(ctx context.Context, prompt string, options TextGenerationOptions) (string, error) {
	if prompt == "" {
		return "", fmt.Errorf("prompt cannot be empty")
	}

	// Determine which model to use
	modelName := c.modelName
	if options.Model != "" {
		modelName = options.Model
	}

	// Build contents
	contents := []*genai.Content{{
		Parts: []*genai.Part{{Text: prompt}},
		Role:  "user",
	}}

	// Build config if options are provided
	var config *genai.GenerateContentConfig
	if options.MaxTokens > 0 || options.Temperature > 0 || options.ResponseSchema != nil {
		config = &genai.GenerateContentConfig{}
		if options.MaxTokens > 0 {
			maxTokens := int32(options.MaxTokens)
			config.MaxOutputTokens = maxTokens
		}
		if options.Temperature > 0 {
			temp := float32(options.Temperature)
			config.Temperature = &temp
		}
		// Phase 1: Structured output support
		if options.ResponseSchema != nil {
			config.ResponseMIMEType = "application/json"
			config.ResponseSchema = options.ResponseSchema
		}
	}

	// Generate content
	resp, err := c.gClient.Models.GenerateContent(ctx, modelName, contents, config)
	if err != nil {
		return "", fmt.Errorf("failed to generate text: %w", err)
	}

	text := resp.Text()
	if text == "" {
		return "", fmt.Errorf("empty response from LLM")
	}

	return text, nil
}

// GenerateSummary is a simpler function, more aligned with the original request,
// that takes text and returns a summary string.
// It uses the GEMINI_API_KEY environment variable and the default model.
func GenerateSummary(textContent string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}

	prompt := fmt.Sprintf(SummarizeTextPromptTemplate, textContent)
	contents := []*genai.Content{{
		Parts: []*genai.Part{{Text: prompt}},
		Role:  "user",
	}}

	resp, err := client.Models.GenerateContent(ctx, DefaultModel, contents, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate content for summarization: %w", err)
	}

	text := resp.Text()
	if text == "" {
		return "", fmt.Errorf("empty summary response")
	}

	return text, nil
}

// RegenerateDigestWithMyTake takes an existing digest and a personal take,
// then uses the LLM to regenerate the entire digest incorporating the personal perspective throughout
func RegenerateDigestWithMyTake(digestContent, myTake, format string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Create a sophisticated prompt that asks the LLM to rewrite the digest
	// incorporating the personal perspective throughout
	prompt := fmt.Sprintf(`You are helping to rewrite a digest to incorporate the author's personal perspective throughout the content.

Here is the original digest:
---
%s
---

Here is the author's personal take/perspective:
---
%s
---

Please rewrite the entire digest incorporating the author's personal voice and perspective throughout. The goal is to:

1. Maintain all the factual information and structure from the original digest
2. Weave the author's perspective naturally throughout the content (not just at the end)
3. Use the author's voice and tone based on their "take"
4. Keep the same format (%s) but make it feel like the author wrote it with their personal insights
5. Make it feel cohesive and natural, not like separate sections were bolted together
6. Preserve any important details, links, and key insights from the original
7. The result should feel like the author's personal commentary and analysis of the topics

Do not add a separate "My Take" section - instead, integrate the perspective throughout the entire digest naturally.

Please provide the complete rewritten digest:`, digestContent, myTake, format)

	contents := []*genai.Content{{
		Parts: []*genai.Part{{Text: prompt}},
		Role:  "user",
	}}

	resp, err := client.Models.GenerateContent(ctx, DefaultModel, contents, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate content for digest regeneration: %w", err)
	}

	text := resp.Text()
	if text == "" {
		return "", fmt.Errorf("empty response")
	}

	return text, nil
}

// GeneratePromptCorner generates interesting prompts based on digest content
// that readers can copy and use with any LLM (ChatGPT, Gemini, Claude, etc.)
func GeneratePromptCorner(digestContent string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}

	prompt := fmt.Sprintf(`Based on the following digest content, create at most 3 interesting and practical prompts that readers can copy and paste into any LLM (ChatGPT, Gemini, Claude, etc.).

The prompts should be:
- Directly inspired by the topics covered in the digest
- Practical and actionable for developers, tech professionals, or curious learners
- Self-contained (no need for additional context)
- Simple to copy and paste
- Encouraging exploration and learning

Format the output as a clean markdown section with:
- A brief intro sentence
- Each prompt in a code block for easy copying
- A short description after each prompt explaining what it's for

Here's the digest content:
---
%s
---

Please generate the Prompt Corner section:`, digestContent)

	contents := []*genai.Content{{
		Parts: []*genai.Part{{Text: prompt}},
		Role:  "user",
	}}

	resp, err := client.Models.GenerateContent(ctx, DefaultModel, contents, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate content for prompt corner: %w", err)
	}

	text := resp.Text()
	if text == "" {
		return "", fmt.Errorf("empty response")
	}

	return text, nil
}

// GenerateDigestTitle creates a compelling Smart Headline for a digest based on the content
func (c *Client) GenerateDigestTitle(digestContent string, format string) (string, error) {
	if digestContent == "" {
		return "", fmt.Errorf("cannot generate title for empty digest content")
	}

	// Create a prompt that asks for a compelling Smart Headline based on the digest content
	prompt := fmt.Sprintf(`Generate a compelling Smart Headline for the following digest content. This headline will be the main title of the digest and should capture readers' attention while accurately representing the content.

REQUIREMENTS:
- Must be under 80 characters
- Should capture the core themes, insights, or trends from the content
- Be engaging and informative to encourage reading
- Avoid generic words like "Update", "News", or "Summary"
- Focus on the most impactful or surprising element from the content
- Align with '%s' format style:
  * Brief: Direct and action-oriented
  * Standard: Clear and informative
  * Detailed: Analytical and comprehensive
  * Newsletter: Engaging and shareable
  * Email: Personal and relevant

DIGEST CONTENT:
---
%s
---

Generate only the Smart Headline text, without quotes or additional formatting:`, format, digestContent)

	ctx := context.Background()
	titleText, err := c.generateContent(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate title: %w", err)
	}

	// Clean up the title - remove any unwanted characters or formatting
	titleStr := strings.TrimSpace(titleText)
	titleStr = strings.Trim(titleStr, "\"'") // Remove quotes if present

	return titleStr, nil
}

// GenerateDigestTitle creates a compelling Smart Headline for a digest based on the content
// It uses the GEMINI_API_KEY environment variable and the default model.
func GenerateDigestTitle(digestContent string, format string) (string, error) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", fmt.Errorf("failed to create Gemini client: %w", err)
	}

	// Create a prompt that asks for a compelling Smart Headline based on the digest content
	prompt := fmt.Sprintf(`Generate a compelling Smart Headline for the following digest content. This headline will be the main title of the digest and should capture readers' attention while accurately representing the content.

REQUIREMENTS:
- Must be under 80 characters
- Should capture the core themes, insights, or trends from the content
- Be engaging and informative to encourage reading
- Avoid generic words like "Update", "News", or "Summary"
- Focus on the most impactful or surprising element from the content
- Align with '%s' format style:
  * Brief: Direct and action-oriented
  * Standard: Clear and informative
  * Detailed: Analytical and comprehensive
  * Newsletter: Engaging and shareable
  * Email: Personal and relevant

DIGEST CONTENT:
---
%s
---

Generate only the Smart Headline text, without quotes or additional formatting:`, format, digestContent)

	contents := []*genai.Content{{
		Parts: []*genai.Part{{Text: prompt}},
		Role:  "user",
	}}

	resp, err := client.Models.GenerateContent(ctx, DefaultModel, contents, nil)
	if err != nil {
		return "", fmt.Errorf("failed to generate title: %w", err)
	}

	titleText := resp.Text()
	if titleText == "" {
		return "", fmt.Errorf("empty title response")
	}

	// Clean up the title - remove any unwanted characters or formatting
	titleStr := strings.TrimSpace(titleText)
	titleStr = strings.Trim(titleStr, "\"'") // Remove quotes if present

	return titleStr, nil
}

// GenerateEmbedding generates a vector embedding for the given text using Gemini's embedding model
// Uses gemini-embedding-001 with Matryoshka to output 768 dimensions for compatibility
func (c *Client) GenerateEmbedding(text string) ([]float64, error) {
	ctx := context.Background()

	// Build content for embedding
	contents := []*genai.Content{{
		Parts: []*genai.Part{{Text: text}},
		Role:  "user",
	}}

	// Configure embedding with 768 dimensions using Matryoshka
	dims := DefaultEmbeddingDimensions
	config := &genai.EmbedContentConfig{
		OutputDimensionality: &dims,
	}

	resp, err := c.gClient.Models.EmbedContent(ctx, DefaultEmbeddingModel, contents, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if resp == nil || len(resp.Embeddings) == 0 || resp.Embeddings[0] == nil {
		return nil, fmt.Errorf("no embedding values returned from API")
	}

	// Convert float32 to float64
	values := resp.Embeddings[0].Values
	embedding := make([]float64, len(values))
	for i, val := range values {
		embedding[i] = float64(val)
	}

	return embedding, nil
}

// GenerateEmbeddingForArticle generates an embedding for an article's content
func (c *Client) GenerateEmbeddingForArticle(article core.Article) ([]float64, error) {
	// Combine title and content for better embedding representation
	text := article.Title + "\n\n" + article.CleanedText

	// Truncate if text is too long (embedding models have token limits)
	if len(text) > 8000 { // Conservative limit for gemini-embedding-001
		text = text[:8000]
	}

	return c.GenerateEmbedding(text)
}

// GenerateEmbeddingForSummary generates an embedding for a summary's content
func (c *Client) GenerateEmbeddingForSummary(summary core.Summary) ([]float64, error) {
	return c.GenerateEmbedding(summary.SummaryText)
}

// CosineSimilarity calculates the cosine similarity between two embeddings
func CosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}

// GenerateResearchQueries generates search queries for deep research based on article content
func (c *Client) GenerateResearchQueries(article core.Article, depth int) ([]string, error) {
	if article.CleanedText == "" {
		return nil, fmt.Errorf("article ID %s has no CleanedText for research query generation", article.ID)
	}

	prompt := fmt.Sprintf(`Based on the following article, generate %d highly specific and targeted search queries that would help discover related content, background information, and follow-up research. 

The queries should be:
- Specific enough to find relevant, high-quality sources
- Diverse in perspective (technical, business, historical, competitive analysis)
- Actionable for someone researching this topic further
- Not too broad or generic

Article Title: %s

Article Content:
%s

Return the queries as a simple numbered list (1. Query one 2. Query two, etc.) without any additional formatting:`, depth, article.Title, article.CleanedText)

	ctx := context.Background()
	queriesText, err := c.generateContent(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate research queries for article ID %s: %w", article.ID, err)
	}

	// Parse the numbered list of queries
	lines := strings.Split(queriesText, "\n")
	var queries []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove numbering like "1. " or "- "
		line = strings.TrimPrefix(line, "- ")
		if len(line) > 2 && line[1] == '.' && line[0] >= '1' && line[0] <= '9' {
			line = strings.TrimSpace(line[2:])
		}
		if line != "" {
			queries = append(queries, line)
		}
	}

	return queries, nil
}

// GenerateDigestResearchQueries generates comprehensive research queries based on the entire digest content
// This creates follow-up research directions based on themes and patterns across all articles
func (c *Client) GenerateDigestResearchQueries(digestContent string, teamContext string, articleTitles []string) ([]string, error) {
	articlesContext := strings.Join(articleTitles, "\n- ")

	prompt := fmt.Sprintf(`Based on this digest content and team context, generate 5-7 strategic research queries that would help the team dive deeper into the themes and trends identified across all articles.

DIGEST CONTENT:
%s

TEAM CONTEXT:
%s

ARTICLES COVERED:
- %s

REQUIREMENTS:
- Focus on cross-article themes and patterns rather than individual articles
- Generate queries that would help uncover future trends and opportunities
- Consider the team's context and current challenges
- Include both technical and strategic research directions
- Make queries specific enough to be actionable but broad enough to uncover new insights
- Mix different research angles: competitive analysis, technical deep-dives, market trends, case studies

FORMAT: Return as a numbered list (1. Query one 2. Query two, etc.) without additional formatting:`, digestContent, teamContext, articlesContext)

	ctx := context.Background()
	queriesText, err := c.generateContent(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate digest research queries: %w", err)
	}

	// Parse the numbered list of queries
	lines := strings.Split(queriesText, "\n")
	var queries []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove numbering like "1. " or "- "
		line = strings.TrimPrefix(line, "- ")
		if len(line) > 2 && line[1] == '.' && line[0] >= '1' && line[0] <= '9' {
			line = strings.TrimSpace(line[2:])
		}
		if line != "" {
			queries = append(queries, line)
		}
	}

	return queries, nil
}

// GenerateTrendAnalysisPrompt creates a prompt for analyzing trends between current and previous articles
func (c *Client) GenerateTrendAnalysisPrompt(currentTopics []string, previousTopics []string, timeframe string) string {
	currentTopicsStr := strings.Join(currentTopics, ", ")
	previousTopicsStr := strings.Join(previousTopics, ", ")

	return fmt.Sprintf(`Analyze the trends and changes in topics between two time periods:

CURRENT PERIOD (%s):
Topics: %s

PREVIOUS PERIOD:
Topics: %s

Please provide a trend analysis that includes:
1. **New Emerging Topics**: Topics that appeared in current period but not in previous
2. **Declining Topics**: Topics that were prominent before but less so now
3. **Consistent Themes**: Topics that remain important across both periods
4. **Notable Shifts**: Any significant changes in focus or emphasis

Format your response as a brief, insightful analysis (150-200 words) suitable for inclusion in a digest.`,
		timeframe, currentTopicsStr, previousTopicsStr)
}

// GenerateFinalDigest creates a comprehensive digest summary from multiple article summaries
func (c *Client) GenerateFinalDigest(combinedSummaries, format string) (string, error) {
	if combinedSummaries == "" {
		return "", fmt.Errorf("cannot generate digest from empty summaries")
	}

	// Create format-specific style requirements
	var styleRequirements string
	switch format {
	case "brief":
		styleRequirements = "Concise and to-the-point (150-300 words total). Focus only on the most essential insights."
	case "standard":
		styleRequirements = "Balanced and informative (300-500 words). Include key points with sufficient context."
	case "detailed":
		styleRequirements = "Comprehensive and analytical (500-800 words). Provide thorough analysis and deeper insights."
	case "newsletter":
		styleRequirements = "Engaging and shareable (400-600 words). Use a conversational tone with compelling insights."
	case "scannable":
		styleRequirements = "Scannable and link-focused (300-400 words). Create a brief overview that highlights key themes without burying the individual articles."
	case "email":
		styleRequirements = "Personal and relevant (300-500 words). Write as if addressing a colleague directly."
	default:
		styleRequirements = "Clear and informative. Include key points with sufficient context."
	}

	// Create a comprehensive prompt for digest generation based on format
	prompt := fmt.Sprintf(`You are creating a comprehensive digest summary that synthesizes multiple article summaries into a cohesive, engaging overview. This will be the Executive Summary section of a digest.

FORMAT: %s
STYLE REQUIREMENTS for %s format:
%s

STRUCTURAL REQUIREMENTS:
- Write in well-structured paragraphs with natural line breaks
- Use 2-4 focused paragraphs instead of one long block of text
- Start with an engaging hook or compelling insight
- Each paragraph should focus on a specific theme or group of related insights
- End with forward-looking implications or actionable takeaways
- Break up dense content for better readability

Your task is to:
1. Read through all the individual article summaries
2. Identify 2-3 key themes that connect the articles
3. Create a cohesive narrative that weaves these themes together
4. Structure the content into digestible, thematic paragraphs
5. Write in a %s style that matches the format requirements above
6. INCLUDE CITATIONS: When referencing specific insights, include citations [1], [2], etc.

INDIVIDUAL ARTICLE SUMMARIES:
---
%s
---

Please generate a well-structured Executive Summary that:

**PARAGRAPH 1**: Open with the most compelling insight, trend, or theme that emerges from these articles. Include a brief subject line or hook that captures attention.

**PARAGRAPH 2-3**: Develop 1-2 additional key themes, showing how the articles connect and what patterns emerge. Focus on practical implications and insights.

**PARAGRAPH 4**: Close with forward-looking takeaways, actionable insights, or what these developments mean for readers.

CITATION REQUIREMENTS:
- Use numbered citations [1], [2], [3], etc. when referencing specific articles
- Place citations immediately after claims from specific sources
- Ensure credibility through proper attribution

Generate the structured Executive Summary with proper paragraph breaks and citations:`, format, format, styleRequirements, format, combinedSummaries)

	ctx := context.Background()
	digestText, err := c.generateContent(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate final digest: %w", err)
	}

	return digestText, nil
}

// GenerateStructuredDigest creates a comprehensive, cohesive digest using structural guidelines
// This replaces the fragmented approach of building separate sections
func (c *Client) GenerateStructuredDigest(combinedSummaries, format string, alertsSummary string, overallSentiment string, researchSuggestions []string) (string, error) {
	if combinedSummaries == "" {
		return "", fmt.Errorf("cannot generate digest from empty summaries")
	}

	// Create format-specific parameters
	var wordTarget, structuralGuidance string
	switch format {
	case "brief":
		wordTarget = "150-200 words total"
		structuralGuidance = "Write 2 focused paragraphs: (1) Main insight with key finding, (2) One actionable takeaway"
	case "standard":
		wordTarget = "300-400 words total"
		structuralGuidance = "Structure in 3-4 paragraphs: (1) Hook with main theme, (2-3) Key insights with practical implications, (4) Forward-looking conclusion"
	case "newsletter":
		wordTarget = "400-500 words total"
		structuralGuidance = "Create engaging newsletter content: (1) Attention-grabbing opener, (2-3) Key themes with practical insights, (4) Actionable takeaways, (5) Reader-focused conclusion"
	case "scannable":
		wordTarget = "200-300 words total"
		structuralGuidance = "Create scannable overview: (1) Brief engaging opener about main theme, (2-3) Key patterns and insights that connect the articles, (3) What readers should focus on"
	case "detailed":
		wordTarget = "500-700 words total"
		structuralGuidance = "Comprehensive analysis: (1) Executive overview, (2-3) Detailed theme analysis, (4) Implications and trends, (5) Actionable recommendations"
	default:
		wordTarget = "300-400 words total"
		structuralGuidance = "Structure in 3-4 paragraphs with clear theme progression"
	}

	// Prepare context information
	alertsContext := ""
	if alertsSummary != "" {
		alertsContext = fmt.Sprintf("\n\nALERT CONTEXT:\n%s", alertsSummary)
	}

	sentimentContext := ""
	if overallSentiment != "" {
		sentimentContext = fmt.Sprintf("\n\nSENTIMENT ANALYSIS:\n%s", overallSentiment)
	}

	researchContext := ""
	if len(researchSuggestions) > 0 {
		researchContext = fmt.Sprintf("\n\nRELATED RESEARCH AREAS:\n%s", strings.Join(researchSuggestions[:3], "; ")) // Top 3 suggestions
	}

	// Check if articles are categorized by looking for "Category:" in the combined summaries
	categorizedPrompt := ""
	if strings.Contains(combinedSummaries, "**Category:") {
		categorizedPrompt = `

CATEGORIZATION CONTEXT:
The articles have been organized into categories (Breaking & Hot, Product Updates, Dev Tools & Techniques, etc.). Use this categorization to:
- Identify thematic patterns across categories
- Connect related insights within and between categories
- Highlight the most significant developments from high-priority categories
- Create smooth transitions between different topic areas`
	}

	prompt := fmt.Sprintf(`You are creating a cohesive, professionally-written digest that synthesizes multiple article summaries into one flowing narrative. Generate the COMPLETE digest content as a unified piece of writing.

TARGET FORMAT: %s (%s)
STRUCTURAL GUIDANCE: %s%s

YOUR TASK:
Create a complete digest that flows naturally from insight to insight, avoiding the disconnected "section-by-section" approach. The content should feel like a single, coherent piece of analysis written by a knowledgeable professional.

WRITING REQUIREMENTS:
- Write in a professional but accessible tone
- Use natural transitions between paragraphs
- Include specific citations [1], [2], [3] when referencing articles
- Integrate any alerts or sentiment insights naturally into the narrative
- Provide actionable takeaways embedded in the analysis
- Maintain consistent voice throughout

CONTENT INTEGRATION:
- Weave insights together thematically rather than listing them separately
- Connect related findings across different articles
- Highlight patterns and implications naturally
- Include practical takeaways as part of the narrative flow%s%s%s

INDIVIDUAL ARTICLE SUMMARIES:
---
%s
---

Generate a complete, cohesive digest that reads as a unified analysis rather than disconnected sections. Start with an engaging hook and build a narrative that connects the key insights from these articles into a compelling story about the trends and implications for readers.

IMPORTANT: Generate ONLY the main content - no headers, no "## Executive Summary" titles, just the flowing narrative content that will form the heart of the digest. Ensure the content is complete and stays within the target word count without abrupt truncation.`, format, wordTarget, structuralGuidance, categorizedPrompt, alertsContext, sentimentContext, researchContext, combinedSummaries)

	ctx := context.Background()
	digestText, err := c.generateContent(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to generate structured digest: %w", err)
	}

	return digestText, nil
}

// AnalyzeSentimentWithEmoji analyzes the sentiment of text and returns score, label, and emoji
func (c *Client) AnalyzeSentimentWithEmoji(text string) (float64, string, string, error) {
	prompt := fmt.Sprintf(`Analyze the sentiment of the following text and respond with EXACTLY this format:

SENTIMENT_SCORE: [number between -1.0 and 1.0, where -1.0 = very negative, 0.0 = neutral, 1.0 = very positive]
SENTIMENT_LABEL: [one word: positive, negative, or neutral]
SENTIMENT_EMOJI: [single emoji that best represents the sentiment]

Text to analyze:
%s

Remember: Respond with EXACTLY the format above, nothing else.`, text)

	ctx := context.Background()
	resultText, err := c.generateContent(ctx, prompt)
	if err != nil {
		return 0, "", "", fmt.Errorf("failed to analyze sentiment: %w", err)
	}

	// Parse the response
	lines := strings.Split(resultText, "\n")
	var score = 0.0
	var label = "neutral"
	var emoji = "😐"

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "SENTIMENT_SCORE:") {
			scoreStr := strings.TrimSpace(strings.TrimPrefix(line, "SENTIMENT_SCORE:"))
			if s, err := fmt.Sscanf(scoreStr, "%f", &score); err != nil || s != 1 {
				score = 0.0 // default neutral
			}
		} else if strings.HasPrefix(line, "SENTIMENT_LABEL:") {
			label = strings.TrimSpace(strings.TrimPrefix(line, "SENTIMENT_LABEL:"))
		} else if strings.HasPrefix(line, "SENTIMENT_EMOJI:") {
			emoji = strings.TrimSpace(strings.TrimPrefix(line, "SENTIMENT_EMOJI:"))
		}
	}

	// Clamp score to valid range
	if score < -1.0 {
		score = -1.0
	} else if score > 1.0 {
		score = 1.0
	}

	// Validate label
	if label != "positive" && label != "negative" && label != "neutral" {
		label = "neutral"
	}

	return score, label, emoji, nil
}

// AnalyzeYouTubeVideo creates intelligent content based on video metadata
func (c *Client) AnalyzeYouTubeVideo(ctx context.Context, videoURL, videoTitle, channelName string) (string, error) {
	// Create an intelligent summary based on video metadata using Gemini's knowledge
	prompt := fmt.Sprintf(`Based on the YouTube video metadata provided below, generate a comprehensive and intelligent summary that explains what this video likely covers and why it might be valuable.

Video Title: "%s"
Channel: "%s"
Video URL: %s

Using your knowledge about this channel, topic area, and typical content patterns, provide:

1. **Likely Content Overview**: What this video probably discusses based on the title and channel
2. **Key Topics**: Main subjects and themes likely covered
3. **Target Audience**: Who would benefit from watching this video
4. **Technical Relevance**: How this content might be relevant to technical professionals
5. **Context & Implications**: Why this topic matters in the current landscape

Write this as a 200-300 word summary that helps readers understand the value and content of the video without needing to watch it. Be specific and insightful based on the title and channel context.

Focus on being informative and helpful rather than speculative. Draw on your knowledge of the subject matter suggested by the title and the reputation/focus of the channel.`, videoTitle, channelName, videoURL)

	content, err := c.generateContent(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("failed to analyze YouTube video: %w", err)
	}

	return content, nil
}

// ChatSession represents an active chat session with the LLM
// Uses the new SDK pattern with manual history management
type ChatSession struct {
	client    *Client
	history   []*genai.Content
	context   string
	modelName string
}

// StartChatSession initializes a new chat session with the given context
func (c *Client) StartChatSession(ctx context.Context, initialContext string) (*ChatSession, error) {
	// Initialize history with system context
	history := []*genai.Content{{
		Parts: []*genai.Part{{Text: initialContext}},
		Role:  "user",
	}}

	// Get initial response to establish context
	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr(float32(0.7)),
	}

	resp, err := c.gClient.Models.GenerateContent(ctx, c.modelName, history, config)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize chat session: %w", err)
	}

	// Add assistant response to history
	assistantText := resp.Text()
	history = append(history, &genai.Content{
		Parts: []*genai.Part{{Text: assistantText}},
		Role:  "model",
	})

	return &ChatSession{
		client:    c,
		history:   history,
		context:   initialContext,
		modelName: c.modelName,
	}, nil
}

// SendChatMessage sends a message to the chat session and returns the response
func (c *Client) SendChatMessage(ctx context.Context, session *ChatSession, message string) (string, error) {
	if session == nil {
		return "", fmt.Errorf("invalid chat session")
	}

	// Add user message to history
	session.history = append(session.history, &genai.Content{
		Parts: []*genai.Part{{Text: message}},
		Role:  "user",
	})

	// Configure for chat
	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr(float32(0.7)),
	}

	// Send the full history
	resp, err := c.gClient.Models.GenerateContent(ctx, session.modelName, session.history, config)
	if err != nil {
		return "", fmt.Errorf("failed to send message: %w", err)
	}

	responseText := resp.Text()
	if responseText == "" {
		return "", fmt.Errorf("empty response from model")
	}

	// Add assistant response to history
	session.history = append(session.history, &genai.Content{
		Parts: []*genai.Part{{Text: responseText}},
		Role:  "model",
	})

	return strings.TrimSpace(responseText), nil
}
