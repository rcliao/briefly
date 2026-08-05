package llm

import (
	"briefly/internal/core"
	"briefly/internal/observability"
	"context"
	"time"
)

// TracedClient wraps an LLM Client with PostHog analytics tracking
type TracedClient struct {
	client  *Client
	posthog *observability.PostHogClient
}

// NewTracedClient creates a new traced LLM client
func NewTracedClient(modelName string, posthog *observability.PostHogClient) (*TracedClient, error) {
	client, err := NewClient(modelName)
	if err != nil {
		return nil, err
	}

	return &TracedClient{
		client:  client,
		posthog: posthog,
	}, nil
}

// GetUnderlyingClient returns the underlying untraced client (for methods that don't need tracing)
func (tc *TracedClient) GetUnderlyingClient() *Client {
	return tc.client
}

// trackLLMCall records an LLM call in PostHog when enabled; failures are logged nowhere
// by design — analytics must never break the pipeline.
func (tc *TracedClient) trackLLMCall(ctx context.Context, model, operation string, tokens int, latencyMs int64) {
	if tc.posthog != nil && tc.posthog.IsEnabled() {
		_ = tc.posthog.TrackLLMCall(ctx, model, operation, tokens, latencyMs, 0)
	}
}

// GenerateText generates text with tracking
func (tc *TracedClient) GenerateText(ctx context.Context, prompt string, options TextGenerationOptions) (string, error) {
	startTime := time.Now()
	result, err := tc.client.GenerateText(ctx, prompt, options)

	model := options.Model
	if model == "" {
		model = tc.client.modelName
	}
	tc.trackLLMCall(ctx, model, "text_generation", estimateTokens(prompt, result), time.Since(startTime).Milliseconds())

	return result, err
}

// GenerateEmbedding generates embeddings with tracking
func (tc *TracedClient) GenerateEmbedding(text string) ([]float64, error) {
	startTime := time.Now()
	result, err := tc.client.GenerateEmbedding(text)
	tc.trackLLMCall(context.Background(), DefaultEmbeddingModel, "embedding", estimateTokens(text, ""), time.Since(startTime).Milliseconds())
	return result, err
}

// SummarizeArticleText summarizes article with tracking
func (tc *TracedClient) SummarizeArticleText(article core.Article) (core.Summary, error) {
	startTime := time.Now()
	result, err := tc.client.SummarizeArticleText(article)
	tc.trackLLMCall(context.Background(), tc.client.modelName, "summarization", estimateTokens(article.CleanedText, result.SummaryText), time.Since(startTime).Milliseconds())
	return result, err
}

// SummarizeArticleTextWithFormat summarizes article with format and tracking
func (tc *TracedClient) SummarizeArticleTextWithFormat(article core.Article, format string) (core.Summary, error) {
	startTime := time.Now()
	result, err := tc.client.SummarizeArticleTextWithFormat(article, format)
	tc.trackLLMCall(context.Background(), tc.client.modelName, "summarization", estimateTokens(article.CleanedText, result.SummaryText), time.Since(startTime).Milliseconds())
	return result, err
}

// CategorizeArticle categorizes article with tracking (used for theme classification)
func (tc *TracedClient) CategorizeArticle(ctx context.Context, article core.Article, categories map[string]Category) (CategoryResult, error) {
	startTime := time.Now()
	result, err := tc.client.CategorizeArticle(ctx, article, categories)
	tc.trackLLMCall(ctx, tc.client.modelName, "categorization", estimateTokens(article.CleanedText, result.Category.Name), time.Since(startTime).Milliseconds())
	return result, err
}

// Close closes the underlying client and flushes analytics
func (tc *TracedClient) Close() {
	tc.client.Close()

	if tc.posthog != nil && tc.posthog.IsEnabled() {
		_ = tc.posthog.Flush()
	}
}

// estimateTokens provides a rough estimate of token count
// This is a simple approximation: ~4 characters per token for English text
func estimateTokens(prompt, completion string) int {
	return (len(prompt) + len(completion)) / 4
}

// Passthrough methods that don't need special tracing
// These delegate directly to the underlying client

func (tc *TracedClient) SummarizeArticleWithKeyMoments(article core.Article) (core.Summary, error) {
	return tc.client.SummarizeArticleWithKeyMoments(article)
}

func (tc *TracedClient) GenerateWhyItMatters(articles []core.Article, teamContext string) (map[string]string, error) {
	return tc.client.GenerateWhyItMatters(articles, teamContext)
}

func (tc *TracedClient) GenerateWhyItMattersSingle(article core.Article, teamContext string) (string, error) {
	return tc.client.GenerateWhyItMattersSingle(article, teamContext)
}

func (tc *TracedClient) GenerateTeamRelevanceScore(article core.Article, teamContext string) (float64, string, error) {
	return tc.client.GenerateTeamRelevanceScore(article, teamContext)
}

func (tc *TracedClient) RegenerateDigestWithMyTake(originalDigest, myTake, teamContext, styleGuide string) (string, error) {
	return tc.client.RegenerateDigestWithMyTake(originalDigest, myTake, teamContext, styleGuide)
}

func (tc *TracedClient) GenerateDigestTitle(digestContent string, format string) (string, error) {
	return tc.client.GenerateDigestTitle(digestContent, format)
}

func (tc *TracedClient) GenerateEmbeddingForArticle(article core.Article) ([]float64, error) {
	return tc.client.GenerateEmbeddingForArticle(article)
}

func (tc *TracedClient) GenerateEmbeddingForSummary(summary core.Summary) ([]float64, error) {
	return tc.client.GenerateEmbeddingForSummary(summary)
}

func (tc *TracedClient) GenerateResearchQueries(article core.Article, depth int) ([]string, error) {
	return tc.client.GenerateResearchQueries(article, depth)
}

func (tc *TracedClient) GenerateDigestResearchQueries(digestContent string, teamContext string, articleTitles []string) ([]string, error) {
	return tc.client.GenerateDigestResearchQueries(digestContent, teamContext, articleTitles)
}

func (tc *TracedClient) GenerateTrendAnalysisPrompt(currentTopics []string, previousTopics []string, timeframe string) string {
	return tc.client.GenerateTrendAnalysisPrompt(currentTopics, previousTopics, timeframe)
}

func (tc *TracedClient) GenerateFinalDigest(combinedSummaries, format string) (string, error) {
	return tc.client.GenerateFinalDigest(combinedSummaries, format)
}

func (tc *TracedClient) GenerateStructuredDigest(combinedSummaries, format string, alertsSummary string, overallSentiment string, researchSuggestions []string) (string, error) {
	return tc.client.GenerateStructuredDigest(combinedSummaries, format, alertsSummary, overallSentiment, researchSuggestions)
}

func (tc *TracedClient) AnalyzeSentimentWithEmoji(text string) (float64, string, string, error) {
	return tc.client.AnalyzeSentimentWithEmoji(text)
}

func (tc *TracedClient) AnalyzeYouTubeVideo(ctx context.Context, videoURL, videoTitle, channelName string) (string, error) {
	return tc.client.AnalyzeYouTubeVideo(ctx, videoURL, videoTitle, channelName)
}

func (tc *TracedClient) StartChatSession(ctx context.Context, initialContext string) (*ChatSession, error) {
	return tc.client.StartChatSession(ctx, initialContext)
}

func (tc *TracedClient) SendChatMessage(ctx context.Context, session *ChatSession, message string) (string, error) {
	return tc.client.SendChatMessage(ctx, session, message)
}
