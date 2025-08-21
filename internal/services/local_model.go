package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"briefly/internal/core"
)

// ollamaService implements LocalModelService using Ollama
type ollamaService struct {
	baseURL    string
	httpClient *http.Client
	model      string
}

// NewOllamaService creates a new Ollama local model service
func NewOllamaService(baseURL string, model string) LocalModelService {
	if baseURL == "" {
		baseURL = "http://localhost:11434" // Default Ollama endpoint
	}
	if model == "" {
		model = "llama3.2:3b" // Default lightweight model
	}

	return &ollamaService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		model: model,
	}
}

// IsAvailable checks if Ollama is running and accessible
func (s *ollamaService) IsAvailable(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/api/tags", nil)
	if err != nil {
		return false, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("ollama not accessible: %w", err)
	}
	defer resp.Body.Close()

	return resp.StatusCode == 200, nil
}

// Initialize ensures the required model is available
func (s *ollamaService) Initialize(ctx context.Context) error {
	// Check if Ollama is available
	available, err := s.IsAvailable(ctx)
	if err != nil {
		return fmt.Errorf("ollama initialization failed: %w", err)
	}
	if !available {
		return fmt.Errorf("ollama service not available at %s", s.baseURL)
	}

	// Check if our model is available
	hasModel, err := s.hasModel(ctx, s.model)
	if err != nil {
		return fmt.Errorf("failed to check model availability: %w", err)
	}

	if !hasModel {
		return fmt.Errorf("model %s not available in Ollama - please run: ollama pull %s", s.model, s.model)
	}

	return nil
}

// CategorizeContent uses local model to categorize articles
func (s *ollamaService) CategorizeContent(ctx context.Context, content string) (string, float64, error) {
	prompt := fmt.Sprintf(`Categorize this article into ONE of these categories:
🔥 Breaking & Hot - breaking news, announcements, releases
🛠️ Tools & Platforms - tools, github repos, libraries, frameworks  
📊 Analysis & Research - studies, research, reports, analysis
💰 Business & Economics - business, market, cost, pricing, money
💡 Additional Items - everything else

Article: %s

Respond with only the category name (including emoji).`, truncateContent(content, 500))

	response, err := s.generateCompletion(ctx, prompt)
	if err != nil {
		return "💡 Additional Items", 0.5, err
	}

	category := strings.TrimSpace(response)
	confidence := 0.8 // Local models get decent confidence for categorization

	// Validate category
	validCategories := []string{
		"🔥 Breaking & Hot",
		"🛠️ Tools & Platforms", 
		"📊 Analysis & Research",
		"💰 Business & Economics",
		"💡 Additional Items",
	}

	for _, valid := range validCategories {
		if strings.Contains(category, valid) {
			return valid, confidence, nil
		}
	}

	// Fallback to default category
	return "💡 Additional Items", 0.5, nil
}

// FilterByQuality applies quality filtering using local model
func (s *ollamaService) FilterByQuality(ctx context.Context, articles []core.Article, threshold float64) ([]core.Article, error) {
	var filtered []core.Article

	for _, article := range articles {
		score, err := s.evaluateQuality(ctx, article)
		if err != nil {
			// On error, use basic heuristic
			score = s.calculateBasicQualityScore(article)
		}

		article.QualityScore = score
		if score >= threshold {
			filtered = append(filtered, article)
		}
	}

	return filtered, nil
}

// ClusterArticles groups articles using local model
func (s *ollamaService) ClusterArticles(ctx context.Context, articles []core.Article) ([]core.ArticleGroup, error) {
	if len(articles) == 0 {
		return []core.ArticleGroup{}, nil
	}

	// For Phase 3, use enhanced local clustering
	groups := make(map[string][]core.Article)

	for _, article := range articles {
		category, _, err := s.CategorizeContent(ctx, article.Title+". "+truncateContent(article.CleanedText, 200))
		if err != nil {
			category = "💡 Additional Items" // Fallback
		}
		
		article.TopicCluster = category
		groups[category] = append(groups[category], article)
	}

	// Convert to ArticleGroups
	var articleGroups []core.ArticleGroup
	priority := 1

	for category, categoryArticles := range groups {
		theme, err := s.generateTheme(ctx, categoryArticles)
		if err != nil {
			theme = category // Fallback to category name
		}

		summary, err := s.generateGroupSummary(ctx, categoryArticles)
		if err != nil {
			summary = fmt.Sprintf("%d articles in this category", len(categoryArticles))
		}

		group := core.ArticleGroup{
			Category: category,
			Theme:    theme,
			Articles: categoryArticles,
			Summary:  summary,
			Priority: priority,
		}
		articleGroups = append(articleGroups, group)
		priority++
	}

	return articleGroups, nil
}

// AnalyzeComplexity determines task complexity for routing decisions
func (s *ollamaService) AnalyzeComplexity(ctx context.Context, content string) (float64, error) {
	prompt := fmt.Sprintf(`Analyze the complexity of this task on a scale of 0.0 to 1.0:
- 0.0-0.3: Simple (categorization, basic analysis)
- 0.4-0.6: Medium (summarization, pattern recognition)  
- 0.7-1.0: Complex (synthesis, reasoning, creative tasks)

Task: %s

Respond with only a decimal number between 0.0 and 1.0.`, truncateContent(content, 300))

	response, err := s.generateCompletion(ctx, prompt)
	if err != nil {
		return 0.5, err // Default to medium complexity
	}

	var complexity float64
	if _, parseErr := fmt.Sscanf(strings.TrimSpace(response), "%f", &complexity); parseErr != nil {
		return 0.5, fmt.Errorf("failed to parse complexity: %w", parseErr)
	}

	// Clamp to valid range
	if complexity < 0.0 {
		complexity = 0.0
	}
	if complexity > 1.0 {
		complexity = 1.0
	}

	return complexity, nil
}

// GenerateBasicSummary creates simple summaries using local model
func (s *ollamaService) GenerateBasicSummary(ctx context.Context, content string, maxWords int) (string, error) {
	prompt := fmt.Sprintf(`Summarize this article in exactly %d words or less. Be concise and focus on key points:

%s

Summary:`, maxWords, truncateContent(content, 1000))

	response, err := s.generateCompletion(ctx, prompt)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(response), nil
}

// Helper methods

// generateCompletion makes a completion request to Ollama
func (s *ollamaService) generateCompletion(ctx context.Context, prompt string) (string, error) {
	requestBody := map[string]interface{}{
		"model":  s.model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.3, // Low temperature for consistent results
			"num_ctx":     2048, // Context window
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/api/generate", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama request failed: %d - %s", resp.StatusCode, string(body))
	}

	var response struct {
		Response string `json:"response"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	return response.Response, nil
}

// hasModel checks if a model is available in Ollama
func (s *ollamaService) hasModel(ctx context.Context, model string) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", s.baseURL+"/api/tags", nil)
	if err != nil {
		return false, err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	for _, m := range result.Models {
		if strings.HasPrefix(m.Name, model) {
			return true, nil
		}
	}

	return false, nil
}

// evaluateQuality uses local model to evaluate article quality
func (s *ollamaService) evaluateQuality(ctx context.Context, article core.Article) (float64, error) {
	prompt := fmt.Sprintf(`Evaluate the quality of this article on a scale of 0.0 to 1.0 based on:
- Content depth and usefulness
- Writing quality and clarity  
- Technical accuracy (if applicable)
- Relevance to professional/technical audience

Title: %s
Content: %s

Respond with only a decimal number between 0.0 and 1.0.`, article.Title, truncateContent(article.CleanedText, 800))

	response, err := s.generateCompletion(ctx, prompt)
	if err != nil {
		return 0.0, err
	}

	var quality float64
	if _, parseErr := fmt.Sscanf(strings.TrimSpace(response), "%f", &quality); parseErr != nil {
		return 0.0, fmt.Errorf("failed to parse quality score: %w", parseErr)
	}

	// Clamp to valid range
	if quality < 0.0 {
		quality = 0.0
	}
	if quality > 1.0 {
		quality = 1.0
	}

	return quality, nil
}

// generateTheme creates a theme for a group of articles
func (s *ollamaService) generateTheme(ctx context.Context, articles []core.Article) (string, error) {
	if len(articles) == 0 {
		return "General", nil
	}

	var titles []string
	for _, article := range articles {
		titles = append(titles, article.Title)
	}

	prompt := fmt.Sprintf(`Generate a brief theme (3-5 words) that describes these article titles:

%s

Theme:`, strings.Join(titles, "\n"))

	response, err := s.generateCompletion(ctx, prompt)
	if err != nil {
		return "General", err
	}

	theme := strings.TrimSpace(response)
	if len(theme) > 50 {
		theme = theme[:47] + "..." // Truncate if too long
	}

	return theme, nil
}

// generateGroupSummary creates a summary for a group of articles
func (s *ollamaService) generateGroupSummary(ctx context.Context, articles []core.Article) (string, error) {
	if len(articles) == 0 {
		return "", nil
	}

	if len(articles) == 1 {
		return fmt.Sprintf("Article about %s", articles[0].Title), nil
	}

	var titles []string
	for i, article := range articles {
		if i >= 3 {
			break // Limit to first 3 for context
		}
		titles = append(titles, article.Title)
	}

	prompt := fmt.Sprintf(`Write a 1-sentence summary (max 50 words) describing what these %d articles have in common:

%s

Summary:`, len(articles), strings.Join(titles, "\n"))

	response, err := s.generateCompletion(ctx, prompt)
	if err != nil {
		return fmt.Sprintf("%d related articles", len(articles)), err
	}

	summary := strings.TrimSpace(response)
	if len(summary) > 200 {
		summary = summary[:197] + "..." // Truncate if too long
	}

	return summary, nil
}

// calculateBasicQualityScore provides fallback quality scoring
func (s *ollamaService) calculateBasicQualityScore(article core.Article) float64 {
	score := 0.0

	// Title quality (25% weight)
	if len(article.Title) > 10 && len(article.Title) < 200 {
		score += 0.25
	}

	// Content length (25% weight)
	if len(article.CleanedText) > 500 && len(article.CleanedText) < 10000 {
		score += 0.25
	}

	// URL quality (25% weight)
	if !isSpamDomain(article.URL) {
		score += 0.25
	}

	// Content type bonus (25% weight)
	if article.ContentType == core.ContentTypeHTML {
		score += 0.25
	}

	return score
}

// Utility functions

func truncateContent(content string, maxChars int) string {
	if len(content) <= maxChars {
		return content
	}
	return content[:maxChars] + "..."
}

func isSpamDomain(url string) bool {
	spamDomains := []string{"spam.com", "fake.com", "clickbait.com"}
	for _, domain := range spamDomains {
		if strings.Contains(url, domain) {
			return true
		}
	}
	return false
}