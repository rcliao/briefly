package pipeline

import (
	"briefly/internal/core"
	"context"
	"fmt"
	"time"
)

// Pipeline orchestrates the end-to-end digest generation workflow
// It coordinates all components according to the simplified architecture
type Pipeline struct {
	// Core components
	parser        URLParser
	fetcher       ContentFetcher
	summarizer    ArticleSummarizer
	embedder      EmbeddingGenerator
	clusterer     TopicClusterer
	orderer       ArticleOrderer
	narrative     NarrativeGenerator
	renderer      MarkdownRenderer
	cache         CacheManager
	banner        BannerGenerator // Optional

	// Configuration
	config *Config
}

// Config holds pipeline configuration
type Config struct {
	// Cache settings
	CacheEnabled bool
	CacheTTL     time.Duration

	// Processing settings
	MaxConcurrentFetches int
	RetryAttempts        int
	RequestTimeout       time.Duration

	// Output settings
	OutputFormat    string // Always "markdown" for now
	GenerateBanner  bool
	BannerStyle     string

	// Quality settings
	MinArticleLength  int     // Minimum chars for valid article
	MinSummaryQuality float64 // 0-1 quality threshold
}

// DefaultConfig returns sensible default configuration
func DefaultConfig() *Config {
	return &Config{
		CacheEnabled:         true,
		CacheTTL:             7 * 24 * time.Hour, // 7 days
		MaxConcurrentFetches: 5,
		RetryAttempts:        3,
		RequestTimeout:       30 * time.Second,
		OutputFormat:         "markdown",
		GenerateBanner:       false,
		BannerStyle:          "tech",
		MinArticleLength:     100,
		MinSummaryQuality:    0.5,
	}
}

// NewPipeline creates a new pipeline with all dependencies
func NewPipeline(
	parser URLParser,
	fetcher ContentFetcher,
	summarizer ArticleSummarizer,
	embedder EmbeddingGenerator,
	clusterer TopicClusterer,
	orderer ArticleOrderer,
	narrative NarrativeGenerator,
	renderer MarkdownRenderer,
	cache CacheManager,
	banner BannerGenerator,
	config *Config,
) *Pipeline {
	if config == nil {
		config = DefaultConfig()
	}

	return &Pipeline{
		parser:     parser,
		fetcher:    fetcher,
		summarizer: summarizer,
		embedder:   embedder,
		clusterer:  clusterer,
		orderer:    orderer,
		narrative:  narrative,
		renderer:   renderer,
		cache:      cache,
		banner:     banner,
		config:     config,
	}
}

// DigestOptions configures digest generation
type DigestOptions struct {
	InputFile      string
	OutputPath     string
	GenerateBanner bool
	BannerStyle    string
	DryRun         bool
}

// DigestResult contains the output of digest generation
type DigestResult struct {
	Digest       *core.Digest
	MarkdownPath string
	BannerPath   string
	Stats        ProcessingStats
}

// ProcessingStats tracks pipeline execution metrics
type ProcessingStats struct {
	TotalURLs          int
	SuccessfulArticles int
	FailedArticles     int
	CacheHits          int
	CacheMisses        int
	ClustersGenerated  int
	ProcessingTime     time.Duration
	StartTime          time.Time
	EndTime            time.Time
}

// GenerateDigest executes the full digest generation pipeline
func (p *Pipeline) GenerateDigest(ctx context.Context, opts DigestOptions) (*DigestResult, error) {
	startTime := time.Now()
	stats := ProcessingStats{
		StartTime: startTime,
	}

	// Step 1: Parse URLs from markdown file
	fmt.Printf("📄 Step 1/9: Parsing URLs from %s...\n", opts.InputFile)
	links, err := p.parser.ParseMarkdownFile(opts.InputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URLs: %w", err)
	}

	stats.TotalURLs = len(links)
	if stats.TotalURLs == 0 {
		return nil, fmt.Errorf("no valid URLs found in input file")
	}
	fmt.Printf("   ✓ Found %d URLs\n\n", stats.TotalURLs)

	// Step 2: Fetch and summarize articles (with caching)
	fmt.Printf("🔍 Step 2/9: Fetching and summarizing articles...\n")
	articles, summaries, err := p.processArticles(ctx, links, &stats)
	if err != nil {
		return nil, fmt.Errorf("failed to process articles: %w", err)
	}

	if len(articles) == 0 {
		return nil, fmt.Errorf("no articles were successfully processed")
	}

	stats.SuccessfulArticles = len(articles)
	stats.FailedArticles = stats.TotalURLs - stats.SuccessfulArticles
	fmt.Printf("   ✓ Successfully processed %d/%d articles\n", stats.SuccessfulArticles, stats.TotalURLs)
	fmt.Printf("   • Cache hits: %d, Cache misses: %d\n\n", stats.CacheHits, stats.CacheMisses)

	// Step 3: Generate embeddings for clustering
	fmt.Printf("🧠 Step 3/9: Generating embeddings for clustering...\n")
	embeddings, err := p.generateEmbeddings(ctx, summaries)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}
	fmt.Printf("   ✓ Generated %d embeddings\n\n", len(embeddings))

	// Step 4: Cluster articles by topic
	fmt.Printf("🔗 Step 4/9: Clustering articles by topic...\n")
	clusters, err := p.clusterer.ClusterArticles(ctx, articles, summaries, embeddings)
	if err != nil {
		return nil, fmt.Errorf("failed to cluster articles: %w", err)
	}

	stats.ClustersGenerated = len(clusters)
	fmt.Printf("   ✓ Created %d topic clusters\n\n", stats.ClustersGenerated)

	// Step 5: Order articles within clusters
	fmt.Printf("📊 Step 5/9: Ordering articles within clusters...\n")
	orderedClusters, err := p.orderer.OrderClusters(ctx, clusters, articles)
	if err != nil {
		return nil, fmt.Errorf("failed to order articles: %w", err)
	}
	fmt.Printf("   ✓ Ordered %d clusters\n\n", len(orderedClusters))

	// Step 6: Generate executive summary
	fmt.Printf("📝 Step 6/9: Generating executive summary...\n")
	executiveSummary, err := p.narrative.GenerateExecutiveSummary(ctx, orderedClusters, articlesToMap(articles), summariesToMap(summaries))
	if err != nil {
		// Non-fatal: log and continue without executive summary
		fmt.Printf("   ⚠️  Executive summary generation failed, continuing without it\n\n")
		executiveSummary = ""
	} else {
		fmt.Printf("   ✓ Generated executive summary (%d words)\n\n", len(executiveSummary)/5)
	}

	// Step 7: Build digest structure
	fmt.Printf("🔨 Step 7/9: Building digest structure...\n")
	digest := p.buildDigest(orderedClusters, articles, summaries, executiveSummary)
	fmt.Printf("   ✓ Digest structure complete\n\n")

	// Step 8: Render markdown output
	fmt.Printf("✍️  Step 8/9: Rendering markdown output...\n")
	markdownPath, err := p.renderer.RenderDigest(ctx, digest, opts.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to render digest: %w", err)
	}
	fmt.Printf("   ✓ Saved to %s\n\n", markdownPath)

	// Step 9: Optional banner generation
	var bannerPath string
	if opts.GenerateBanner && p.banner != nil {
		fmt.Printf("🎨 Step 9/9: Generating banner image...\n")
		bannerPath, err = p.banner.GenerateBanner(ctx, digest, opts.BannerStyle)
		if err != nil {
			// Non-fatal: log warning and continue without banner
			fmt.Printf("   ⚠️  Banner generation failed, continuing without it\n\n")
			bannerPath = ""
		} else {
			fmt.Printf("   ✓ Banner saved to %s\n\n", bannerPath)
		}
	} else {
		fmt.Printf("⏭️  Step 9/9: Skipping banner generation\n\n")
	}

	stats.EndTime = time.Now()
	stats.ProcessingTime = stats.EndTime.Sub(startTime)

	return &DigestResult{
		Digest:       digest,
		MarkdownPath: markdownPath,
		BannerPath:   bannerPath,
		Stats:        stats,
	}, nil
}

// QuickReadOptions configures quick read operation
type QuickReadOptions struct {
	URL string
}

// QuickReadResult contains the output of quick read
type QuickReadResult struct {
	Article     *core.Article
	Summary     *core.Summary
	Markdown    string
	WasCached   bool
	ProcessTime time.Duration
}

// QuickRead processes a single URL and returns a summary
func (p *Pipeline) QuickRead(ctx context.Context, opts QuickReadOptions) (*QuickReadResult, error) {
	startTime := time.Now()

	// Step 1: Check cache
	cachedArticle, cachedSummary, err := p.checkQuickReadCache(opts.URL)
	if err == nil && cachedArticle != nil && cachedSummary != nil {
		// Render cached result
		markdown, err := p.renderer.RenderQuickRead(ctx, cachedArticle, cachedSummary)
		if err != nil {
			return nil, fmt.Errorf("failed to render cached summary: %w", err)
		}

		return &QuickReadResult{
			Article:     cachedArticle,
			Summary:     cachedSummary,
			Markdown:    markdown,
			WasCached:   true,
			ProcessTime: time.Since(startTime),
		}, nil
	}

	// Step 2: Fetch article
	article, err := p.fetcher.FetchArticle(ctx, opts.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch article: %w", err)
	}

	// Step 3: Summarize
	summary, err := p.summarizer.SummarizeArticle(ctx, article)
	if err != nil {
		return nil, fmt.Errorf("failed to summarize article: %w", err)
	}

	// Step 4: Cache result
	if p.config.CacheEnabled {
		_ = p.cacheQuickRead(article, summary)
	}

	// Step 5: Render
	markdown, err := p.renderer.RenderQuickRead(ctx, article, summary)
	if err != nil {
		return nil, fmt.Errorf("failed to render summary: %w", err)
	}

	return &QuickReadResult{
		Article:     article,
		Summary:     summary,
		Markdown:    markdown,
		WasCached:   false,
		ProcessTime: time.Since(startTime),
	}, nil
}

// processArticles fetches and summarizes all articles with cache support
func (p *Pipeline) processArticles(ctx context.Context, links []core.Link, stats *ProcessingStats) ([]core.Article, []core.Summary, error) {
	articles := make([]core.Article, 0, len(links))
	summaries := make([]core.Summary, 0, len(links))

	// Process each link (TODO: Add concurrency control)
	for i, link := range links {
		fmt.Printf("   [%d/%d] Processing: %s\n", i+1, len(links), link.URL)

		// Check cache first
		if p.config.CacheEnabled {
			cachedArticle, cachedSummary, err := p.checkArticleCache(link.URL)
			if err == nil && cachedArticle != nil && cachedSummary != nil {
				fmt.Printf("           ✓ Cache hit\n")
				articles = append(articles, *cachedArticle)
				summaries = append(summaries, *cachedSummary)
				stats.CacheHits++
				continue
			}
		}

		stats.CacheMisses++

		// Fetch article
		article, err := p.fetcher.FetchArticle(ctx, link.URL)
		if err != nil {
			// Log error but continue with other articles
			fmt.Printf("           ✗ Fetch failed: %v\n", err)
			continue
		}

		// Validate article quality
		if len(article.CleanedText) < p.config.MinArticleLength {
			// Skip articles that are too short
			fmt.Printf("           ✗ Article too short (%d chars)\n", len(article.CleanedText))
			continue
		}

		// Summarize article
		summary, err := p.summarizer.SummarizeArticle(ctx, article)
		if err != nil {
			// Log error but continue with other articles
			fmt.Printf("           ✗ Summarization failed: %v\n", err)
			continue
		}

		// Cache result
		if p.config.CacheEnabled {
			_ = p.cacheArticle(article, summary)
		}

		fmt.Printf("           ✓ Fetched and summarized\n")
		articles = append(articles, *article)
		summaries = append(summaries, *summary)
	}

	return articles, summaries, nil
}

// generateEmbeddings creates vector embeddings for all summaries
func (p *Pipeline) generateEmbeddings(ctx context.Context, summaries []core.Summary) (map[string][]float64, error) {
	embeddings := make(map[string][]float64)
	var failedCount int

	for i, summary := range summaries {
		fmt.Printf("   [%d/%d] Generating embedding for summary %s\n", i+1, len(summaries), summary.ID)

		embedding, err := p.embedder.GenerateEmbedding(ctx, summary.SummaryText)
		if err != nil {
			// Log error but continue with other summaries
			fmt.Printf("           ✗ Embedding generation failed: %v\n", err)
			failedCount++
			continue
		}

		embeddings[summary.ID] = embedding
		fmt.Printf("           ✓ Embedding generated (%d dimensions)\n", len(embedding))
	}

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("failed to generate any embeddings (all %d attempts failed)", failedCount)
	}

	if failedCount > 0 {
		fmt.Printf("   ⚠️  Warning: %d/%d embeddings failed to generate\n", failedCount, len(summaries))
	}

	return embeddings, nil
}

// buildDigest constructs the final digest structure
func (p *Pipeline) buildDigest(clusters []core.TopicCluster, articles []core.Article, summaries []core.Summary, executiveSummary string) *core.Digest {
	digest := &core.Digest{
		ID:            generateID(),
		DigestSummary: executiveSummary,
		DateGenerated: time.Now(),
	}

	// Build article groups from clusters
	articleGroups := make([]core.ArticleGroup, 0, len(clusters))
	articleURLs := make([]string, 0, len(articles))

	for _, cluster := range clusters {
		group := core.ArticleGroup{
			Theme:    cluster.Label,
			Articles: []core.Article{},
		}

		// Add articles from this cluster
		for _, articleID := range cluster.ArticleIDs {
			for _, article := range articles {
				if article.ID == articleID {
					group.Articles = append(group.Articles, article)
					articleURLs = append(articleURLs, article.URL)
					break
				}
			}
		}

		articleGroups = append(articleGroups, group)
	}

	digest.ArticleGroups = articleGroups
	digest.ArticleURLs = articleURLs

	// Set metadata
	digest.Metadata = core.DigestMetadata{
		Title:         fmt.Sprintf("Weekly Digest - %s", time.Now().Format("2006-01-02")),
		DateGenerated: time.Now(),
		ArticleCount:  len(articles),
	}

	return digest
}

// Cache helper methods

func (p *Pipeline) checkArticleCache(url string) (*core.Article, *core.Summary, error) {
	if p.cache == nil {
		return nil, nil, fmt.Errorf("cache not available")
	}
	return p.cache.GetArticleWithSummary(url, p.config.CacheTTL)
}

func (p *Pipeline) cacheArticle(article *core.Article, summary *core.Summary) error {
	if p.cache == nil {
		return nil
	}
	return p.cache.StoreArticleWithSummary(article, summary, p.config.CacheTTL)
}

func (p *Pipeline) checkQuickReadCache(url string) (*core.Article, *core.Summary, error) {
	if p.cache == nil {
		return nil, nil, fmt.Errorf("cache not available")
	}
	return p.cache.GetArticleWithSummary(url, 24*time.Hour) // 24 hour TTL for quick reads
}

func (p *Pipeline) cacheQuickRead(article *core.Article, summary *core.Summary) error {
	if p.cache == nil {
		return nil
	}
	return p.cache.StoreArticleWithSummary(article, summary, 24*time.Hour)
}

// Helper functions

func articlesToMap(articles []core.Article) map[string]core.Article {
	result := make(map[string]core.Article)
	for _, article := range articles {
		result[article.ID] = article
	}
	return result
}

func summariesToMap(summaries []core.Summary) map[string]core.Summary {
	result := make(map[string]core.Summary)
	for _, summary := range summaries {
		result[summary.ID] = summary
	}
	return result
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}