package pipeline

import (
	"briefly/internal/citations"
	"briefly/internal/clustering"
	"briefly/internal/core"
	"briefly/internal/fetch"
	"briefly/internal/llm"
	"briefly/internal/narrative"
	"briefly/internal/parser"
	"briefly/internal/persistence"
	"briefly/internal/render"
	"briefly/internal/store"
	"briefly/internal/vectorstore"
	"context"
	"fmt"
	"strings"
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

// SemanticClustererAdapter wraps internal/clustering/semantic (Phase 2)
// Uses pgvector HNSW index for fast similarity-based clustering
type SemanticClustererAdapter struct {
	clusterer *clustering.SemanticClusterer
}

// vectorSearcherWrapper wraps VectorStore to implement clustering.VectorSearcher
type vectorSearcherWrapper struct {
	store VectorStore
}

func (w *vectorSearcherWrapper) SearchSimilar(ctx context.Context, embedding []float64, limit int, threshold float64, excludeIDs []string) ([]clustering.SearchResult, error) {
	// Lower the threshold slightly to allow more connections
	// News articles are diverse, 0.5 (50%) similarity is reasonable
	adjustedThreshold := 0.5

	query := VectorSearchQuery{
		Embedding:           embedding,
		Limit:               limit,
		SimilarityThreshold: adjustedThreshold,
		IncludeArticle:      false,
		ExcludeIDs:          excludeIDs,
	}

	results, err := w.store.Search(ctx, query)
	if err != nil {
		return nil, err
	}

	// Convert to clustering.SearchResult
	searchResults := make([]clustering.SearchResult, len(results))
	for i, r := range results {
		searchResults[i] = clustering.SearchResult{
			ArticleID:  r.ArticleID,
			Similarity: r.Similarity,
		}
	}

	return searchResults, nil
}

func NewSemanticClustererAdapter(vectorStore VectorStore) *SemanticClustererAdapter {
	searcher := &vectorSearcherWrapper{store: vectorStore}
	// Enable tag-aware clustering for better digest quality
	clusterer := clustering.NewSemanticClusterer(searcher).WithTagAware(true)
	return &SemanticClustererAdapter{
		clusterer: clusterer,
	}
}

func (a *SemanticClustererAdapter) ClusterArticles(ctx context.Context, articles []core.Article, summaries []core.Summary, embeddings map[string][]float64) ([]core.TopicCluster, error) {
	if len(articles) == 0 {
		return nil, fmt.Errorf("no articles to cluster")
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings provided")
	}

	// Use semantic clustering (graph-based, no need to pre-determine k)
	return a.clusterer.ClusterArticles(ctx, articles, embeddings)
}

func (a *SemanticClustererAdapter) CalculateSimilarity(embedding1, embedding2 []float64) float64 {
	// Use cosine similarity from LLM package
	return llm.CosineSimilarity(embedding1, embedding2)
}

// LouvainClustererAdapter wraps internal/clustering/louvain
// Uses Louvain community detection for higher-quality clustering
// Key advantage: uses edge WEIGHTS (similarity scores) instead of binary connections
type LouvainClustererAdapter struct {
	clusterer *clustering.LouvainClusterer
}

// NewLouvainClustererAdapter creates a new Louvain clusterer adapter
func NewLouvainClustererAdapter(vectorStore VectorStore) *LouvainClustererAdapter {
	searcher := &vectorSearcherWrapper{store: vectorStore}
	clusterer := clustering.NewLouvainClusterer(searcher).
		WithTagAware(true).
		WithResolution(1.0).
		WithMinSimilarity(0.3).
		WithMaxNeighbors(10)
	return &LouvainClustererAdapter{clusterer: clusterer}
}

func (a *LouvainClustererAdapter) ClusterArticles(ctx context.Context, articles []core.Article, summaries []core.Summary, embeddings map[string][]float64) ([]core.TopicCluster, error) {
	if len(articles) == 0 {
		return nil, fmt.Errorf("no articles to cluster")
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings provided")
	}

	// Use Louvain community detection (optimizes modularity Q)
	return a.clusterer.ClusterArticles(ctx, articles, embeddings)
}

func (a *LouvainClustererAdapter) CalculateSimilarity(embedding1, embedding2 []float64) float64 {
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
	llmClient narrative.LLMClient
}

func NewNarrativeAdapter(llmClient narrative.LLMClient) *NarrativeAdapter {
	return &NarrativeAdapter{
		generator: narrative.NewGenerator(llmClient),
		llmClient: llmClient,
	}
}

func (a *NarrativeAdapter) GenerateExecutiveSummary(ctx context.Context, clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (string, error) {
	return a.generator.GenerateExecutiveSummary(ctx, clusters, articles, summaries)
}

func (a *NarrativeAdapter) GenerateDigestContent(ctx context.Context, clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (*narrative.DigestContent, error) {
	return a.generator.GenerateDigestContent(ctx, clusters, articles, summaries)
}

func (a *NarrativeAdapter) GenerateText(ctx context.Context, prompt string, options llm.TextGenerationOptions) (string, error) {
	return a.llmClient.GenerateText(ctx, prompt, options)
}

func (a *NarrativeAdapter) IdentifyClusterTheme(ctx context.Context, cluster core.TopicCluster, articles []core.Article) (string, error) {
	return a.generator.IdentifyClusterTheme(ctx, cluster, articles)
}

func (a *NarrativeAdapter) SelectTopArticles(cluster core.TopicCluster, articles []core.Article, n int) []core.Article {
	return a.generator.SelectTopArticles(cluster, articles, n)
}

// GenerateClusterSummary generates a comprehensive narrative for a single cluster using ALL articles
func (a *NarrativeAdapter) GenerateClusterSummary(ctx context.Context, cluster core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (*core.ClusterNarrative, error) {
	return a.generator.GenerateClusterSummary(ctx, cluster, articles, summaries)
}

// RendererAdapter wraps internal/render and templates
type RendererAdapter struct {
	// Will use existing render/templates packages
}

func NewRendererAdapter() *RendererAdapter {
	return &RendererAdapter{}
}

func (a *RendererAdapter) RenderDigest(ctx context.Context, digest *core.Digest, outputPath string) (string, error) {
	// Use the new category-based rendering with ArticleGroups
	// Build a map of article ID to summary
	summaryMap := make(map[string]*core.Summary)
	for i := range digest.Summaries {
		summary := &digest.Summaries[i]
		for _, articleID := range summary.ArticleIDs {
			summaryMap[articleID] = summary
		}
	}

	// Populate summaries in articles for rendering
	for i := range digest.ArticleGroups {
		for j := range digest.ArticleGroups[i].Articles {
			article := &digest.ArticleGroups[i].Articles[j]
			if summary, exists := summaryMap[article.ID]; exists {
				// Store summary in MyTake for rendering (legacy field)
				article.MyTake = summary.SummaryText
			}
		}
	}

	// Render using category-based template
	content, err := a.renderCategoryBasedDigest(digest)
	if err != nil {
		return "", fmt.Errorf("failed to render digest: %w", err)
	}

	// Write to file
	dateStr := digest.Metadata.DateGenerated.Format("2006-01-02")
	filename := fmt.Sprintf("digest_signal_%s.md", dateStr)

	if outputPath == "" {
		outputPath = "digests"
	}

	filePath, err := render.WriteDigestToFile(content, outputPath, filename)
	if err != nil {
		return "", fmt.Errorf("failed to write digest file: %w", err)
	}

	return filePath, nil
}

// renderCategoryBasedDigest renders digest grouped by categories
func (a *RendererAdapter) renderCategoryBasedDigest(digest *core.Digest) (string, error) {
	var content strings.Builder

	// Header
	content.WriteString(fmt.Sprintf("# %s\n\n", digest.Metadata.Title))

	// Article count and reading time
	articleCount := digest.Metadata.ArticleCount
	readTime := (articleCount * 2) / 3 // Rough estimate: 2 min per 3 articles
	if readTime < 1 {
		readTime = 1
	}
	content.WriteString(fmt.Sprintf("📊 %d sources • ⏱️ %dm read\n\n", articleCount, readTime))

	// Signal section (executive summary)
	if digest.DigestSummary != "" {
		content.WriteString("## 🔍 Signal\n\n")
		content.WriteString(digest.DigestSummary)
		content.WriteString("\n\n")
	}

	// Sources section grouped by category
	content.WriteString("## 📚 Sources\n\n")

	// Use global article numbering across all categories
	globalArticleNum := 1

	for _, group := range digest.ArticleGroups {
		if len(group.Articles) == 0 {
			continue
		}

		// Category header with icon
		categoryIcon := a.getCategoryIcon(group.Category)
		content.WriteString(fmt.Sprintf("### %s %s\n\n", categoryIcon, group.Category))

		// Articles in this category
		for i, article := range group.Articles {
			if i > 0 {
				content.WriteString("\n")
			}

			// Article title with global numbering
			content.WriteString(fmt.Sprintf("**[%d] %s**\n", globalArticleNum, article.Title))
			globalArticleNum++ // Increment global counter

			// Summary from MyTake (populated earlier)
			if article.MyTake != "" {
				content.WriteString(article.MyTake)
				content.WriteString("\n\n")
			}

			// Link
			content.WriteString(fmt.Sprintf("🔗 [Read more](%s)\n\n", article.URL))
		}
	}

	// Footer
	content.WriteString("---\n\n")
	content.WriteString("*Generated using hybrid AI processing*\n")

	return content.String(), nil
}

// getCategoryIcon returns the emoji icon for a category
func (a *RendererAdapter) getCategoryIcon(category string) string {
	icons := map[string]string{
		"Platform Updates": "📦",
		"From the Field":   "💭",
		"Research":         "📊",
		"Tutorials":        "🎓",
		"Analysis":         "🔍",
		"Miscellaneous":    "📌",
	}

	if icon, found := icons[category]; found {
		return icon
	}
	return "💡" // Default icon
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

func (a *LLMClientAdapter) GenerateText(ctx context.Context, prompt string, options llm.TextGenerationOptions) (string, error) {
	return a.client.GenerateText(ctx, prompt, options)
}

// LegacyLLMClientAdapter adapts the new LLM client interface to old interfaces that don't support options
// This is for backward compatibility with packages that haven't been updated yet
type LegacyLLMClientAdapter struct {
	client *llm.Client
}

func NewLegacyLLMClientAdapter(client *llm.Client) *LegacyLLMClientAdapter {
	return &LegacyLLMClientAdapter{
		client: client,
	}
}

// GenerateText implements the old interface without options parameter
func (a *LegacyLLMClientAdapter) GenerateText(ctx context.Context, prompt string) (string, error) {
	return a.client.GenerateText(ctx, prompt, llm.TextGenerationOptions{})
}

// CategorizerAdapter wraps internal/categorization to implement ArticleCategorizer
type CategorizerAdapter struct {
	categorizer interface {
		CategorizeArticle(ctx context.Context, article *core.Article, summary *core.Summary) (string, error)
	}
}

func NewCategorizerAdapter(categorizer interface {
	CategorizeArticle(ctx context.Context, article *core.Article, summary *core.Summary) (string, error)
}) *CategorizerAdapter {
	return &CategorizerAdapter{
		categorizer: categorizer,
	}
}

func (a *CategorizerAdapter) CategorizeArticle(ctx context.Context, article *core.Article, summary *core.Summary) (string, error) {
	return a.categorizer.CategorizeArticle(ctx, article, summary)
}

// CitationTrackerAdapter wraps internal/citations to implement CitationTracker
type CitationTrackerAdapter struct {
	tracker *citations.Tracker
}

func NewCitationTrackerAdapter(db persistence.Database) *CitationTrackerAdapter {
	return &CitationTrackerAdapter{
		tracker: citations.NewTracker(db),
	}
}

func (a *CitationTrackerAdapter) TrackArticle(ctx context.Context, article *core.Article) (*core.Citation, error) {
	return a.tracker.TrackArticle(ctx, article)
}

func (a *CitationTrackerAdapter) TrackBatch(ctx context.Context, articles []core.Article) (map[string]*core.Citation, error) {
	return a.tracker.TrackBatch(ctx, articles)
}

func (a *CitationTrackerAdapter) GetCitation(ctx context.Context, articleID string) (*core.Citation, error) {
	return a.tracker.GetCitation(ctx, articleID)
}

// VectorStoreAdapter wraps internal/vectorstore to implement pipeline.VectorStore
// Provides type conversion between vectorstore and pipeline types
type VectorStoreAdapter struct {
	store vectorstore.VectorStore
}

// NewVectorStoreAdapter creates a new vector store adapter
func NewVectorStoreAdapter(store vectorstore.VectorStore) *VectorStoreAdapter {
	return &VectorStoreAdapter{store: store}
}

func (a *VectorStoreAdapter) Store(ctx context.Context, articleID string, embedding []float64) error {
	return a.store.Store(ctx, articleID, embedding)
}

func (a *VectorStoreAdapter) Search(ctx context.Context, query VectorSearchQuery) ([]VectorSearchResult, error) {
	// Convert pipeline query to vectorstore query
	vsQuery := vectorstore.SearchQuery{
		Embedding:           query.Embedding,
		Limit:               query.Limit,
		SimilarityThreshold: query.SimilarityThreshold,
		IncludeArticle:      query.IncludeArticle,
		ExcludeIDs:          query.ExcludeIDs,
	}

	results, err := a.store.Search(ctx, vsQuery)
	if err != nil {
		return nil, err
	}

	// Convert vectorstore results to pipeline results
	return a.convertResults(results), nil
}

func (a *VectorStoreAdapter) SearchByTag(ctx context.Context, query VectorSearchQuery, tagID string) ([]VectorSearchResult, error) {
	vsQuery := vectorstore.SearchQuery{
		Embedding:           query.Embedding,
		Limit:               query.Limit,
		SimilarityThreshold: query.SimilarityThreshold,
		IncludeArticle:      query.IncludeArticle,
		ExcludeIDs:          query.ExcludeIDs,
	}

	results, err := a.store.SearchByTag(ctx, vsQuery, tagID)
	if err != nil {
		return nil, err
	}

	return a.convertResults(results), nil
}

func (a *VectorStoreAdapter) SearchByTags(ctx context.Context, query VectorSearchQuery, tagIDs []string) ([]VectorSearchResult, error) {
	vsQuery := vectorstore.SearchQuery{
		Embedding:           query.Embedding,
		Limit:               query.Limit,
		SimilarityThreshold: query.SimilarityThreshold,
		IncludeArticle:      query.IncludeArticle,
		ExcludeIDs:          query.ExcludeIDs,
	}

	results, err := a.store.SearchByTags(ctx, vsQuery, tagIDs)
	if err != nil {
		return nil, err
	}

	return a.convertResults(results), nil
}

func (a *VectorStoreAdapter) Delete(ctx context.Context, articleID string) error {
	return a.store.Delete(ctx, articleID)
}

func (a *VectorStoreAdapter) CreateIndex(ctx context.Context) error {
	return a.store.CreateIndex(ctx)
}

func (a *VectorStoreAdapter) GetStats(ctx context.Context) (*VectorStoreStats, error) {
	stats, err := a.store.GetStats(ctx)
	if err != nil {
		return nil, err
	}

	// Convert vectorstore stats to pipeline stats
	return &VectorStoreStats{
		TotalEmbeddings:     stats.TotalEmbeddings,
		EmbeddingDimensions: stats.EmbeddingDimensions,
		IndexType:           stats.IndexType,
		IndexSize:           stats.IndexSize,
		AvgSearchLatency:    stats.AvgSearchLatency,
	}, nil
}

// convertResults converts vectorstore.SearchResult to pipeline.VectorSearchResult
func (a *VectorStoreAdapter) convertResults(vsResults []vectorstore.SearchResult) []VectorSearchResult {
	results := make([]VectorSearchResult, len(vsResults))
	for i, vsr := range vsResults {
		results[i] = VectorSearchResult{
			ArticleID:  vsr.ArticleID,
			Similarity: vsr.Similarity,
			Article:    vsr.Article,
			TagIDs:     vsr.TagIDs,
			Distance:   vsr.Distance,
		}
	}
	return results
}
