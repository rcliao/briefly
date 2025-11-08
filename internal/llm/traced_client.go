package llm

import (
	"briefly/internal/core"
	"briefly/internal/observability"
	"context"
	"time"

	"github.com/google/generative-ai-go/genai"
)

// TracedClient wraps an LLM Client with LangFuse tracing
type TracedClient struct {
	client   *Client
	langfuse *observability.LangFuseClient
	posthog  *observability.PostHogClient
}

// NewTracedClient creates a new traced LLM client
func NewTracedClient(modelName string, langfuse *observability.LangFuseClient, posthog *observability.PostHogClient) (*TracedClient, error) {
	client, err := NewClient(modelName)
	if err != nil {
		return nil, err
	}

	return &TracedClient{
		client:   client,
		langfuse: langfuse,
		posthog:  posthog,
	}, nil
}

// GetUntracedClient returns the underlying untrace client (for methods that don't need tracing)
func (tc *TracedClient) GetUnderlyingClient() *Client {
	return tc.client
}

// GenerateText generates text with tracing
func (tc *TracedClient) GenerateText(ctx context.Context, prompt string, options TextGenerationOptions) (string, error) {
	if !tc.langfuse.IsEnabled() {
		// No tracing, call directly
		return tc.client.GenerateText(ctx, prompt, options)
	}

	// Create trace
	trace, err := tc.langfuse.CreateTrace(ctx, observability.TraceOptions{
		Name: "text_generation",
		Tags: []string{"llm", "generation"},
	})
	if err != nil {
		// If tracing fails, continue without it
		return tc.client.GenerateText(ctx, prompt, options)
	}

	// Track start time
	startTime := time.Now()

	// Call the underlying method
	result, err := tc.client.GenerateText(ctx, prompt, options)

	// Calculate latency
	latencyMs := time.Since(startTime).Milliseconds()

	// Track generation in LangFuse
	if trace != nil {
		model := options.Model
		if model == "" {
			model = tc.client.modelName
		}

		_ = tc.langfuse.TrackGeneration(trace, observability.GenerationOptions{
			Model:       model,
			Prompt:      prompt,
			Completion:  result,
			Temperature: options.Temperature,
			MaxTokens:   options.MaxTokens,
			TotalTokens: estimateTokens(prompt, result),
			LatencyMs:   latencyMs,
		})
	}

	// Track in PostHog for analytics
	if tc.posthog.IsEnabled() {
		model := options.Model
		if model == "" {
			model = tc.client.modelName
		}
		_ = tc.posthog.TrackLLMCall(ctx, model, "text_generation", estimateTokens(prompt, result), latencyMs, 0)
	}

	return result, err
}

// GenerateEmbedding generates embeddings with tracing
func (tc *TracedClient) GenerateEmbedding(text string) ([]float64, error) {
	ctx := context.Background()

	if !tc.langfuse.IsEnabled() {
		return tc.client.GenerateEmbedding(text)
	}

	// Create trace
	trace, err := tc.langfuse.CreateTrace(ctx, observability.TraceOptions{
		Name: "embedding_generation",
		Tags: []string{"llm", "embedding"},
	})
	if err != nil {
		return tc.client.GenerateEmbedding(text)
	}

	startTime := time.Now()
	result, err := tc.client.GenerateEmbedding(text)
	latencyMs := time.Since(startTime).Milliseconds()

	if trace != nil {
		_ = tc.langfuse.TrackGeneration(trace, observability.GenerationOptions{
			Model:       DefaultEmbeddingModel,
			Prompt:      text,
			Completion:  "embedding_vector",
			TotalTokens: estimateTokens(text, ""),
			LatencyMs:   latencyMs,
		})
	}

	if tc.posthog.IsEnabled() {
		_ = tc.posthog.TrackLLMCall(ctx, DefaultEmbeddingModel, "embedding", estimateTokens(text, ""), latencyMs, 0)
	}

	return result, err
}

// SummarizeArticleText summarizes article with tracing
func (tc *TracedClient) SummarizeArticleText(article core.Article) (core.Summary, error) {
	ctx := context.Background()

	if !tc.langfuse.IsEnabled() {
		return tc.client.SummarizeArticleText(article)
	}

	trace, err := tc.langfuse.CreateTrace(ctx, observability.TraceOptions{
		Name: "article_summarization",
		Tags: []string{"llm", "summarization"},
		Metadata: map[string]string{
			"article_id":  article.ID,
			"article_url": article.URL,
		},
	})
	if err != nil {
		return tc.client.SummarizeArticleText(article)
	}

	startTime := time.Now()
	result, err := tc.client.SummarizeArticleText(article)
	latencyMs := time.Since(startTime).Milliseconds()

	if trace != nil {
		_ = tc.langfuse.TrackGeneration(trace, observability.GenerationOptions{
			Model:       tc.client.modelName,
			Prompt:      article.CleanedText,
			Completion:  result.SummaryText,
			TotalTokens: estimateTokens(article.CleanedText, result.SummaryText),
			LatencyMs:   latencyMs,
		})
	}

	if tc.posthog.IsEnabled() {
		_ = tc.posthog.TrackLLMCall(ctx, tc.client.modelName, "summarization", estimateTokens(article.CleanedText, result.SummaryText), latencyMs, 0)
	}

	return result, err
}

// SummarizeArticleTextWithFormat summarizes article with format and tracing
func (tc *TracedClient) SummarizeArticleTextWithFormat(article core.Article, format string) (core.Summary, error) {
	ctx := context.Background()

	if !tc.langfuse.IsEnabled() {
		return tc.client.SummarizeArticleTextWithFormat(article, format)
	}

	trace, err := tc.langfuse.CreateTrace(ctx, observability.TraceOptions{
		Name: "article_summarization_formatted",
		Tags: []string{"llm", "summarization", format},
		Metadata: map[string]string{
			"article_id":  article.ID,
			"article_url": article.URL,
			"format":      format,
		},
	})
	if err != nil {
		return tc.client.SummarizeArticleTextWithFormat(article, format)
	}

	startTime := time.Now()
	result, err := tc.client.SummarizeArticleTextWithFormat(article, format)
	latencyMs := time.Since(startTime).Milliseconds()

	if trace != nil {
		_ = tc.langfuse.TrackGeneration(trace, observability.GenerationOptions{
			Model:       tc.client.modelName,
			Prompt:      article.CleanedText,
			Completion:  result.SummaryText,
			TotalTokens: estimateTokens(article.CleanedText, result.SummaryText),
			LatencyMs:   latencyMs,
		})
	}

	if tc.posthog.IsEnabled() {
		_ = tc.posthog.TrackLLMCall(ctx, tc.client.modelName, "summarization", estimateTokens(article.CleanedText, result.SummaryText), latencyMs, 0)
	}

	return result, err
}

// CategorizeArticle categorizes article with tracing (used for theme classification)
func (tc *TracedClient) CategorizeArticle(ctx context.Context, article core.Article, categories map[string]Category) (CategoryResult, error) {
	if !tc.langfuse.IsEnabled() {
		return tc.client.CategorizeArticle(ctx, article, categories)
	}

	trace, err := tc.langfuse.CreateTrace(ctx, observability.TraceOptions{
		Name: "article_categorization",
		Tags: []string{"llm", "categorization", "classification"},
		Metadata: map[string]string{
			"article_id":  article.ID,
			"article_url": article.URL,
		},
	})
	if err != nil {
		return tc.client.CategorizeArticle(ctx, article, categories)
	}

	startTime := time.Now()
	result, err := tc.client.CategorizeArticle(ctx, article, categories)
	latencyMs := time.Since(startTime).Milliseconds()

	if trace != nil {
		// Build completion string from result
		completion := result.Category.Name
		if result.Reasoning != "" {
			completion += " (reason: " + result.Reasoning + ")"
		}

		_ = tc.langfuse.TrackGeneration(trace, observability.GenerationOptions{
			Model:       tc.client.modelName,
			Prompt:      article.CleanedText,
			Completion:  completion,
			TotalTokens: estimateTokens(article.CleanedText, completion),
			LatencyMs:   latencyMs,
		})
	}

	if tc.posthog.IsEnabled() {
		_ = tc.posthog.TrackLLMCall(ctx, tc.client.modelName, "categorization", estimateTokens(article.CleanedText, result.Category.Name), latencyMs, 0)
	}

	return result, err
}

// Close closes both the underlying client and flushes observability
func (tc *TracedClient) Close() {
	tc.client.Close()

	if tc.langfuse != nil && tc.langfuse.IsEnabled() {
		_ = tc.langfuse.Flush()
	}

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

func (tc *TracedClient) GetGenaiModel() *genai.GenerativeModel {
	return tc.client.GetGenaiModel()
}

func (tc *TracedClient) GenerateDigestTitle(digestContent string, format string) (string, error) {
	return tc.client.GenerateDigestTitle(digestContent, format)
}

func (tc *TracedClient) GenerateEmbeddingForArticle(article core.Article) ([]float64, error) {
	// This could use tracing, but for simplicity we'll just pass through
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
