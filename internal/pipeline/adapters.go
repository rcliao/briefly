package pipeline

import (
	"briefly/internal/clustering"
	"briefly/internal/core"
	"briefly/internal/fetch"
	"briefly/internal/llm"
	"briefly/internal/narrative"
	"briefly/internal/parser"
	"briefly/internal/render"
	"briefly/internal/store"
	"briefly/internal/templates"
	"context"
	"fmt"
	"time"
)

// ParserAdapter wraps internal/parser to implement URLParser
type ParserAdapter struct {
	parser *parser.Parser
}

func NewParserAdapter() *ParserAdapter {
	return &ParserAdapter{
		parser: parser.NewParser(),
	}
}

func (a *ParserAdapter) ParseMarkdownFile(filePath string) ([]core.Link, error) {
	return a.parser.ParseMarkdownFile(filePath)
}

func (a *ParserAdapter) ParseMarkdownContent(content string) []string {
	return a.parser.ParseMarkdownContent(content)
}

func (a *ParserAdapter) ValidateURL(url string) error {
	return a.parser.ValidateURL(url)
}

func (a *ParserAdapter) NormalizeURL(url string) string {
	return a.parser.NormalizeURL(url)
}

// FetcherAdapter wraps internal/fetch to implement ContentFetcher
type FetcherAdapter struct {
	processor *fetch.ContentProcessor
}

func NewFetcherAdapter() *FetcherAdapter {
	return &FetcherAdapter{
		processor: fetch.NewContentProcessor(),
	}
}

func (a *FetcherAdapter) FetchArticle(ctx context.Context, url string) (*core.Article, error) {
	return a.processor.ProcessArticle(ctx, url)
}

// LLMAdapter wraps internal/llm for embedding generation
type LLMAdapter struct {
	client *llm.Client
}

func NewLLMAdapter(client *llm.Client) *LLMAdapter {
	return &LLMAdapter{
		client: client,
	}
}

func (a *LLMAdapter) GenerateEmbedding(ctx context.Context, text string) ([]float64, error) {
	return a.client.GenerateEmbedding(text)
}

func (a *LLMAdapter) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float64, error) {
	embeddings := make([][]float64, 0, len(texts))
	for _, text := range texts {
		emb, err := a.client.GenerateEmbedding(text)
		if err != nil {
			return nil, fmt.Errorf("failed to generate embedding: %w", err)
		}
		embeddings = append(embeddings, emb)
	}
	return embeddings, nil
}

// ClustererAdapter wraps internal/clustering
type ClustererAdapter struct {
	clusterer *clustering.KMeansClusterer
}

func NewClustererAdapter() *ClustererAdapter {
	return &ClustererAdapter{
		clusterer: clustering.NewKMeansClusterer(),
	}
}

func (a *ClustererAdapter) ClusterArticles(ctx context.Context, articles []core.Article, summaries []core.Summary, embeddings map[string][]float64) ([]core.TopicCluster, error) {
	// Build a map of article ID to summary for embedding lookup
	// Since summaries can have multiple article IDs, we need to check ArticleIDs array
	articleToSummaryID := make(map[string]string)
	for _, summary := range summaries {
		// A summary can summarize multiple articles (ArticleIDs is an array)
		// For digest pipeline, we generate 1 summary per article, so use first ID
		if len(summary.ArticleIDs) > 0 {
			for _, articleID := range summary.ArticleIDs {
				articleToSummaryID[articleID] = summary.ID
			}
		}
	}

	// Populate embeddings into articles
	articlesWithEmbeddings := make([]core.Article, len(articles))
	for i, article := range articles {
		articlesWithEmbeddings[i] = article

		// Find the corresponding summary ID and embedding
		if summaryID, exists := articleToSummaryID[article.ID]; exists {
			if embedding, hasEmbedding := embeddings[summaryID]; hasEmbedding {
				articlesWithEmbeddings[i].Embedding = embedding
			}
		}
	}

	// Determine optimal number of clusters (2-5 based on article count)
	numClusters := min(5, max(2, len(articles)/3))

	// Use KMeans clustering on articles with embeddings
	return a.clusterer.Cluster(articlesWithEmbeddings, numClusters)
}

func (a *ClustererAdapter) CalculateSimilarity(embedding1, embedding2 []float64) float64 {
	// Use cosine similarity from LLM package
	return llm.CosineSimilarity(embedding1, embedding2)
}

// Helper functions for min/max
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// OrdererAdapter wraps internal/ordering
type OrdererAdapter struct{}

func NewOrdererAdapter() *OrdererAdapter {
	return &OrdererAdapter{}
}

func (a *OrdererAdapter) OrderClusters(ctx context.Context, clusters []core.TopicCluster, articles []core.Article) ([]core.TopicCluster, error) {
	// Use existing ordering logic if available
	// For now, return clusters as-is (can enhance later)
	return clusters, nil
}

func (a *OrdererAdapter) OrderArticlesInCluster(cluster *core.TopicCluster, articles []core.Article) error {
	// Order by signal strength or quality score
	// For now, maintain current order
	return nil
}

// NarrativeAdapter wraps internal/narrative
type NarrativeAdapter struct {
	generator *narrative.Generator
}

func NewNarrativeAdapter(llmClient narrative.LLMClient) *NarrativeAdapter {
	return &NarrativeAdapter{
		generator: narrative.NewGenerator(llmClient),
	}
}

func (a *NarrativeAdapter) GenerateExecutiveSummary(ctx context.Context, clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (string, error) {
	return a.generator.GenerateExecutiveSummary(ctx, clusters, articles, summaries)
}

func (a *NarrativeAdapter) IdentifyClusterTheme(ctx context.Context, cluster core.TopicCluster, articles []core.Article) (string, error) {
	return a.generator.IdentifyClusterTheme(ctx, cluster, articles)
}

func (a *NarrativeAdapter) SelectTopArticles(cluster core.TopicCluster, articles []core.Article, n int) []core.Article {
	return a.generator.SelectTopArticles(cluster, articles, n)
}

// RendererAdapter wraps internal/render and templates
type RendererAdapter struct {
	// Will use existing render/templates packages
}

func NewRendererAdapter() *RendererAdapter {
	return &RendererAdapter{}
}

func (a *RendererAdapter) RenderDigest(ctx context.Context, digest *core.Digest, outputPath string) (string, error) {
	// Convert digest to DigestData format for compatibility with existing templates
	digestItems := make([]render.DigestData, 0)

	// Build a map of article ID to summary
	summaryMap := make(map[string]*core.Summary)
	for i := range digest.Summaries {
		summary := &digest.Summaries[i]
		for _, articleID := range summary.ArticleIDs {
			summaryMap[articleID] = summary
		}
	}

	// Extract articles from ArticleGroups
	for _, group := range digest.ArticleGroups {
		for _, article := range group.Articles {
			// Find summary for this article
			summaryText := ""
			if summary, exists := summaryMap[article.ID]; exists {
				summaryText = summary.SummaryText
			}

			item := render.DigestData{
				Title:           article.Title,
				URL:             article.URL,
				SummaryText:     summaryText,
				TopicCluster:    article.TopicCluster,
				TopicConfidence: article.ClusterConfidence,
				ContentType:     string(article.ContentType),
			}

			// Set content type icons
			switch article.ContentType {
			case core.ContentTypeYouTube:
				item.ContentIcon = "🎥"
				item.ContentLabel = "Video"
				item.Duration = article.Duration
				item.Channel = article.Channel
			case core.ContentTypePDF:
				item.ContentIcon = "📄"
				item.ContentLabel = "PDF"
				item.PageCount = article.PageCount
			default:
				item.ContentIcon = "🔗"
				item.ContentLabel = "Article"
			}

			digestItems = append(digestItems, item)
		}
	}

	// Configure template for newsletter format
	template := &templates.DigestTemplate{
		Format:                    templates.FormatNewsletter,
		Title:                     digest.Metadata.Title,
		IncludeSummaries:          true,
		IncludeKeyInsights:        true,
		IncludeSourceLinks:        true,
		IncludeIndividualArticles: true,
		IncludeTopicClustering:    true,
	}

	// RenderSignalStyleDigest returns (content, filePath, error)
	_, filePath, err := templates.RenderSignalStyleDigest(digestItems, outputPath, digest.DigestSummary, template, digest.Metadata.Title)
	if err != nil {
		return "", fmt.Errorf("failed to render digest: %w", err)
	}

	return filePath, nil
}

func (a *RendererAdapter) RenderQuickRead(ctx context.Context, article *core.Article, summary *core.Summary) (string, error) {
	// Format quick read output
	markdown := fmt.Sprintf("# %s\n\n", article.Title)
	markdown += fmt.Sprintf("**Source:** %s\n\n", article.URL)
	markdown += fmt.Sprintf("## Summary\n\n%s\n\n", summary.SummaryText)

	if len(summary.SummaryText) > 0 && summary.SummaryText != "" {
		markdown += "## Key Points\n\n"
		// Extract key points from summary if structured
		markdown += summary.SummaryText + "\n"
	}

	return markdown, nil
}

func (a *RendererAdapter) FormatForLinkedIn(markdown string) string {
	// Apply LinkedIn-specific formatting
	// For now, return as-is (can enhance later with character limits, etc.)
	return markdown
}

// CacheAdapter wraps internal/store
type CacheAdapter struct {
	store *store.Store
}

func NewCacheAdapter(cacheDir string) (*CacheAdapter, error) {
	s, err := store.NewStore(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	return &CacheAdapter{
		store: s,
	}, nil
}

func (a *CacheAdapter) GetArticleWithSummary(url string, ttl time.Duration) (*core.Article, *core.Summary, error) {
	article, err := a.store.GetCachedArticle(url, ttl)
	if err != nil {
		return nil, nil, err
	}

	// Check if article was found in cache
	if article == nil {
		return nil, nil, nil // Cache miss - no article found
	}

	// Generate content hash for summary lookup
	contentHash := fmt.Sprintf("%x", article.CleanedText)

	summary, err := a.store.GetCachedSummary(article.URL, contentHash, ttl)
	if err != nil {
		return nil, nil, err
	}

	return article, summary, nil
}

func (a *CacheAdapter) StoreArticleWithSummary(article *core.Article, summary *core.Summary, ttl time.Duration) error {
	if err := a.store.CacheArticle(*article); err != nil {
		return fmt.Errorf("failed to cache article: %w", err)
	}

	// Generate content hash
	contentHash := fmt.Sprintf("%x", article.CleanedText)

	if err := a.store.CacheSummary(*summary, article.URL, contentHash); err != nil {
		return fmt.Errorf("failed to cache summary: %w", err)
	}

	return nil
}

func (a *CacheAdapter) GetCachedArticle(url string, ttl time.Duration) (*core.Article, error) {
	return a.store.GetCachedArticle(url, ttl)
}

func (a *CacheAdapter) CacheArticle(article *core.Article, ttl time.Duration) error {
	return a.store.CacheArticle(*article)
}

func (a *CacheAdapter) Clear() error {
	return a.store.ClearCache()
}

func (a *CacheAdapter) Stats() (*core.CacheStats, error) {
	storeStats, err := a.store.GetCacheStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache stats: %w", err)
	}
	// Convert store.CacheStats to core.CacheStats
	return &core.CacheStats{
		ArticleCount:  storeStats.ArticleCount,
		SummaryCount:  storeStats.SummaryCount,
		DigestCount:   storeStats.DigestCount,
		FeedCount:     storeStats.FeedCount,
		FeedItemCount: storeStats.FeedItemCount,
		CacheSize:     storeStats.CacheSize,
		LastUpdated:   storeStats.LastUpdated,
	}, nil
}

func (a *CacheAdapter) Close() error {
	return a.store.Close()
}

// BannerAdapter wraps internal/visual
type BannerAdapter struct {
	// Visual package will be used when banner generation is needed
}

func NewBannerAdapter() *BannerAdapter {
	return &BannerAdapter{}
}

func (a *BannerAdapter) GenerateBanner(ctx context.Context, digest *core.Digest, style string) (string, error) {
	// Use existing visual package when available
	// For now, return empty path (banner generation is optional)
	return "", fmt.Errorf("banner generation not yet implemented in simplified architecture")
}

func (a *BannerAdapter) AnalyzeThemes(digest *core.Digest) ([]core.ContentTheme, error) {
	// Theme analysis for banners
	return nil, fmt.Errorf("theme analysis not yet implemented in simplified architecture")
}

// LLMClientAdapter implements narrative.LLMClient for narrative generation
type LLMClientAdapter struct {
	client *llm.Client
}

func NewLLMClientAdapter(client *llm.Client) *LLMClientAdapter {
	return &LLMClientAdapter{
		client: client,
	}
}

func (a *LLMClientAdapter) GenerateText(ctx context.Context, prompt string) (string, error) {
	return a.client.GenerateText(ctx, prompt, llm.TextGenerationOptions{})
}