package categorization

import (
	"briefly/internal/core"
	"context"
	"fmt"
	"strings"
)

// LLMClient defines the interface for LLM operations needed by the categorizer
type LLMClient interface {
	// GenerateText generates text from a prompt
	GenerateText(ctx context.Context, prompt string) (string, error)
}

// Categorizer assigns articles to categories using LLM
type Categorizer struct {
	llmClient  LLMClient
	categories []Category
}

// NewCategorizer creates a new article categorizer
func NewCategorizer(llmClient LLMClient, categories []Category) *Categorizer {
	if len(categories) == 0 {
		categories = DefaultCategories()
	}

	return &Categorizer{
		llmClient:  llmClient,
		categories: categories,
	}
}

// CategorizeArticle assigns a category to an article based on its content
func (c *Categorizer) CategorizeArticle(ctx context.Context, article *core.Article, summary *core.Summary) (string, error) {
	if article == nil {
		return "", fmt.Errorf("article cannot be nil")
	}

	// Build categorization prompt
	prompt := c.buildCategorizationPrompt(article, summary)

	// Get LLM response
	response, err := c.llmClient.GenerateText(ctx, prompt)
	if err != nil {
		// Fallback to heuristic categorization on error
		return c.heuristicCategorize(article, summary), nil
	}

	// Parse and validate category
	category := c.parseCategory(response)
	if category == "" {
		// Fallback to heuristic if LLM response is invalid
		return c.heuristicCategorize(article, summary), nil
	}

	return category, nil
}

// buildCategorizationPrompt creates the LLM prompt for categorization
func (c *Categorizer) buildCategorizationPrompt(article *core.Article, summary *core.Summary) string {
	var prompt strings.Builder

	prompt.WriteString("Categorize this article into ONE of the following categories:\n\n")

	// List available categories
	for _, cat := range c.categories {
		prompt.WriteString(fmt.Sprintf("- **%s**: %s\n", cat.Name, cat.Description))
	}

	prompt.WriteString("\n**Article Information:**\n")
	prompt.WriteString(fmt.Sprintf("Title: %s\n", article.Title))
	prompt.WriteString(fmt.Sprintf("URL: %s\n", article.URL))

	if summary != nil && summary.SummaryText != "" {
		summaryText := summary.SummaryText
		if len(summaryText) > 500 {
			summaryText = summaryText[:500] + "..."
		}
		prompt.WriteString(fmt.Sprintf("Summary: %s\n", summaryText))
	}

	prompt.WriteString("\n**Instructions:**\n")
	prompt.WriteString("1. Analyze the title, URL, and summary\n")
	prompt.WriteString("2. Determine which category best fits this article\n")
	prompt.WriteString("3. Return ONLY the category name, nothing else\n")
	prompt.WriteString("4. If unsure, prefer 'Miscellaneous'\n")
	prompt.WriteString("\n**Category:**\n")

	return prompt.String()
}

// parseCategory extracts the category name from LLM response
func (c *Categorizer) parseCategory(response string) string {
	// Clean up response
	response = strings.TrimSpace(response)
	response = strings.Trim(response, "\"'`")

	// Check if response matches any category name
	for _, cat := range c.categories {
		if strings.EqualFold(response, cat.Name) {
			return cat.Name
		}
	}

	// Try case-insensitive substring match
	responseLower := strings.ToLower(response)
	for _, cat := range c.categories {
		catLower := strings.ToLower(cat.Name)
		if strings.Contains(responseLower, catLower) {
			return cat.Name
		}
	}

	// No match found
	return ""
}

// heuristicCategorize provides fallback categorization using simple heuristics
func (c *Categorizer) heuristicCategorize(article *core.Article, summary *core.Summary) string {
	titleLower := strings.ToLower(article.Title)
	urlLower := strings.ToLower(article.URL)

	// Platform Updates: Look for product/feature announcement patterns
	platformKeywords := []string{
		"introducing", "announcing", "launch", "release", "available now",
		"new feature", "update", "version", "beta", "generally available",
	}
	for _, keyword := range platformKeywords {
		if strings.Contains(titleLower, keyword) {
			return "Platform Updates"
		}
	}

	// Check URL patterns for platform updates
	if strings.Contains(urlLower, "/blog") || strings.Contains(urlLower, "/news") ||
		strings.Contains(urlLower, "/changelog") || strings.Contains(urlLower, "/release") {
		if strings.Contains(urlLower, "anthropic.com") ||
			strings.Contains(urlLower, "openai.com") ||
			strings.Contains(urlLower, "google") ||
			strings.Contains(urlLower, "microsoft.com") {
			return "Platform Updates"
		}
	}

	// From the Field: Look for practitioner content
	fieldKeywords := []string{
		"how i", "my workflow", "my experience", "in production",
		"lessons learned", "our approach", "what we learned",
	}
	for _, keyword := range fieldKeywords {
		if strings.Contains(titleLower, keyword) {
			return "From the Field"
		}
	}

	// Research: Look for academic/research patterns
	researchKeywords := []string{
		"paper", "study", "research", "arxiv", "analysis of",
		"empirical", "systematic", "survey",
	}
	for _, keyword := range researchKeywords {
		if strings.Contains(titleLower, keyword) || strings.Contains(urlLower, keyword) {
			return "Research"
		}
	}

	// Tutorials: Look for educational content
	tutorialKeywords := []string{
		"how to", "guide", "tutorial", "walkthrough",
		"getting started", "introduction to", "learn",
	}
	for _, keyword := range tutorialKeywords {
		if strings.Contains(titleLower, keyword) {
			return "Tutorials"
		}
	}

	// Analysis: Look for deep dives and opinion pieces
	analysisKeywords := []string{
		"analysis", "deep dive", "why", "understanding",
		"explained", "breakdown", "review",
	}
	for _, keyword := range analysisKeywords {
		if strings.Contains(titleLower, keyword) {
			return "Analysis"
		}
	}

	// Default to Miscellaneous
	return "Miscellaneous"
}

// GetCategories returns the list of categories this categorizer uses
func (c *Categorizer) GetCategories() []Category {
	return c.categories
}
