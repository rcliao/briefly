package pipeline

import (
	"briefly/internal/core"
	"briefly/internal/narrative"
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
	categorizer   ArticleCategorizer // NEW: Categorizes articles
	embedder      EmbeddingGenerator
	clusterer     TopicClusterer
	orderer       ArticleOrderer
	narrative     NarrativeGenerator
	renderer      MarkdownRenderer
	cache         CacheManager
	banner        BannerGenerator   // Optional
	citationTracker CitationTracker // Phase 1: Track citations for articles

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

	// Phase 1: Summary settings
	UseStructuredSummaries bool // Use structured summaries with sections (default: false)
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
		UseStructuredSummaries: false, // Default to simple summaries for backward compatibility
	}
}

// NewPipeline creates a new pipeline with all dependencies
func NewPipeline(
	parser URLParser,
	fetcher ContentFetcher,
	summarizer ArticleSummarizer,
	categorizer ArticleCategorizer,
	embedder EmbeddingGenerator,
	clusterer TopicClusterer,
	orderer ArticleOrderer,
	narrative NarrativeGenerator,
	renderer MarkdownRenderer,
	cache CacheManager,
	banner BannerGenerator,
	citationTracker CitationTracker,
	config *Config,
) *Pipeline {
	if config == nil {
		config = DefaultConfig()
	}

	return &Pipeline{
		parser:          parser,
		fetcher:         fetcher,
		summarizer:      summarizer,
		categorizer:     categorizer,
		embedder:        embedder,
		clusterer:       clusterer,
		orderer:         orderer,
		narrative:       narrative,
		renderer:        renderer,
		cache:           cache,
		banner:          banner,
		citationTracker: citationTracker,
		config:          config,
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

	// Step 2.5: Categorize articles (NEW)
	fmt.Printf("📁 Step 3/10: Categorizing articles...\n")
	articles, err = p.categorizeArticles(ctx, articles, summaries)
	if err != nil {
		// Non-fatal: log warning and continue with default category
		fmt.Printf("   ⚠️  Categorization failed: %v\n", err)
		fmt.Printf("   • Continuing with uncategorized articles\n\n")
	} else {
		fmt.Printf("   ✓ Categorized %d articles\n\n", len(articles))
	}

	// Step 3: Generate embeddings for clustering
	fmt.Printf("🧠 Step 4/10: Generating embeddings for clustering...\n")
	embeddings, err := p.generateEmbeddings(ctx, summaries)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}
	fmt.Printf("   ✓ Generated %d embeddings\n\n", len(embeddings))

	// Step 4: Cluster articles by topic
	fmt.Printf("🔗 Step 5/10: Clustering articles by topic...\n")
	clusters, err := p.clusterer.ClusterArticles(ctx, articles, summaries, embeddings)
	if err != nil {
		return nil, fmt.Errorf("failed to cluster articles: %w", err)
	}

	stats.ClustersGenerated = len(clusters)
	fmt.Printf("   ✓ Created %d topic clusters\n\n", stats.ClustersGenerated)

	// Step 5: Order articles within clusters
	fmt.Printf("📊 Step 6/10: Ordering articles within clusters...\n")
	orderedClusters, err := p.orderer.OrderClusters(ctx, clusters, articles)
	if err != nil {
		return nil, fmt.Errorf("failed to order articles: %w", err)
	}
	fmt.Printf("   ✓ Ordered %d clusters\n\n", len(orderedClusters))

	// Step 6: Build digest structure (before executive summary so we have correct article ordering)
	fmt.Printf("🔨 Step 7/10: Building digest structure...\n")
	digest := p.buildDigest(orderedClusters, articles, summaries, "")
	fmt.Printf("   ✓ Digest structure complete\n")
	fmt.Printf("   • Articles: %d, Summaries: %d\n\n", len(articles), len(summaries))

	// Step 7: Generate digest content (title, TL;DR, executive summary)
	fmt.Printf("📝 Step 8/10: Generating digest content (title, TL;DR, summary)...\n")
	digestContent, err := p.generateDigestContentFromDigest(ctx, digest)
	if err != nil {
		// Non-fatal: log and continue without generated content
		fmt.Printf("   ⚠️  Digest content generation failed: %v\n", err)
		fmt.Printf("   • Continuing with fallback title and empty TL;DR\n\n")
		digestContent.Title = fmt.Sprintf("Weekly Digest - %s", time.Now().Format("Jan 2, 2006"))
		digestContent.TLDRSummary = ""
		digestContent.KeyMoments = []string{}
		digestContent.ExecutiveSummary = ""
	} else {
		fmt.Printf("   ✓ Generated title: %s\n", digestContent.Title)
		fmt.Printf("   ✓ Generated TL;DR: %s\n", digestContent.TLDRSummary)
		fmt.Printf("   ✓ Generated %d key moments\n", len(digestContent.KeyMoments))
		fmt.Printf("   ✓ Generated summary (%d words)\n\n", len(digestContent.ExecutiveSummary)/5)
	}

	// Update digest with generated content
	digest.Metadata.Title = digestContent.Title
	digest.Metadata.TLDRSummary = digestContent.TLDRSummary
	digest.Metadata.KeyMoments = digestContent.KeyMoments
	digest.DigestSummary = digestContent.ExecutiveSummary

	// Step 8: Render markdown output
	fmt.Printf("✍️  Step 9/10: Rendering markdown output...\n")
	markdownPath, err := p.renderer.RenderDigest(ctx, digest, opts.OutputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to render digest: %w", err)
	}
	fmt.Printf("   ✓ Saved to %s\n\n", markdownPath)

	// Step 9: Optional banner generation
	var bannerPath string
	if opts.GenerateBanner && p.banner != nil {
		fmt.Printf("🎨 Step 10/10: Generating banner image...\n")
		bannerPath, err = p.banner.GenerateBanner(ctx, digest, opts.BannerStyle)
		if err != nil {
			// Non-fatal: log warning and continue without banner
			fmt.Printf("   ⚠️  Banner generation failed, continuing without it\n\n")
			bannerPath = ""
		} else {
			fmt.Printf("   ✓ Banner saved to %s\n\n", bannerPath)
		}
	} else {
		fmt.Printf("⏭️  Step 10/10: Skipping banner generation\n\n")
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

		// Track citation (Phase 1) - non-fatal if it fails
		if p.citationTracker != nil {
			_, err := p.citationTracker.TrackArticle(ctx, article)
			if err != nil {
				// Log warning but continue - citation tracking is not critical
				fmt.Printf("           ⚠️  Citation tracking failed: %v\n", err)
			}
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

// categorizeArticles assigns categories to all articles using LLM categorization
func (p *Pipeline) categorizeArticles(ctx context.Context, articles []core.Article, summaries []core.Summary) ([]core.Article, error) {
	if p.categorizer == nil {
		return articles, fmt.Errorf("categorizer not available")
	}

	// Build a map of article ID to summary for faster lookup
	summaryMap := summariesToMap(summaries)

	categorizedArticles := make([]core.Article, 0, len(articles))
	var failedCount int

	for i, article := range articles {
		fmt.Printf("   [%d/%d] Categorizing: %s\n", i+1, len(articles), article.Title)

		// Get corresponding summary
		summary, hasSummary := summaryMap[article.ID]
		var summaryPtr *core.Summary
		if hasSummary {
			summaryPtr = &summary
		}

		// Categorize article
		category, err := p.categorizer.CategorizeArticle(ctx, &article, summaryPtr)
		if err != nil {
			// Log error but continue with default category
			fmt.Printf("           ✗ Categorization failed: %v (using 'Miscellaneous')\n", err)
			category = "Miscellaneous"
			failedCount++
		} else {
			fmt.Printf("           ✓ Category: %s\n", category)
		}

		// Update article with category
		article.Category = category
		categorizedArticles = append(categorizedArticles, article)
	}

	if failedCount > 0 {
		fmt.Printf("   ⚠️  Warning: %d/%d articles failed to categorize\n", failedCount, len(articles))
	}

	return categorizedArticles, nil
}

// buildDigest constructs the final digest structure
// Groups articles by category first, then by cluster theme within each category
func (p *Pipeline) buildDigest(clusters []core.TopicCluster, articles []core.Article, summaries []core.Summary, executiveSummary string) *core.Digest {
	digest := &core.Digest{
		ID:            generateID(),
		DigestSummary: executiveSummary,
		DateGenerated: time.Now(),
	}

	// Build a map of article IDs to full articles for quick lookup
	articleMap := articlesToMap(articles)

	// Group articles by category
	categoryGroups := make(map[string][]core.Article)
	articleURLs := make([]string, 0, len(articles))

	// First pass: organize articles by category
	for _, cluster := range clusters {
		for _, articleID := range cluster.ArticleIDs {
			if article, found := articleMap[articleID]; found {
				category := article.Category
				if category == "" {
					category = "Miscellaneous"
				}
				categoryGroups[category] = append(categoryGroups[category], article)
				articleURLs = append(articleURLs, article.URL)
			}
		}
	}

	// Build article groups by category
	articleGroups := make([]core.ArticleGroup, 0, len(categoryGroups))
	for category, categoryArticles := range categoryGroups {
		group := core.ArticleGroup{
			Category: category,
			Theme:    category, // Use category as theme for now
			Articles: categoryArticles,
			Priority: p.getCategoryPriority(category),
		}
		articleGroups = append(articleGroups, group)
	}

	// Sort groups by priority (lower number = higher priority)
	// This ensures Platform Updates comes before Miscellaneous, etc.
	for i := 0; i < len(articleGroups); i++ {
		for j := i + 1; j < len(articleGroups); j++ {
			if articleGroups[i].Priority > articleGroups[j].Priority {
				articleGroups[i], articleGroups[j] = articleGroups[j], articleGroups[i]
			}
		}
	}

	digest.ArticleGroups = articleGroups
	digest.ArticleURLs = articleURLs
	digest.Summaries = summaries // Store summaries for rendering

	// Set metadata (title and TL;DR will be generated later)
	digest.Metadata = core.DigestMetadata{
		Title:         "", // Will be generated by narrative generator
		TLDRSummary:   "", // Will be generated by narrative generator
		DateGenerated: time.Now(),
		ArticleCount:  len(articles),
	}

	return digest
}

// getCategoryPriority returns the priority of a category for sorting
// Lower numbers appear first in the digest
func (p *Pipeline) getCategoryPriority(category string) int {
	// Map of category to priority (from categorization/categories.go default order)
	priorities := map[string]int{
		"Platform Updates": 1,
		"From the Field":   2,
		"Research":         3,
		"Tutorials":        4,
		"Analysis":         5,
		"Miscellaneous":    99,
	}

	if priority, found := priorities[category]; found {
		return priority
	}
	return 50 // Default priority for unknown categories
}

// generateExecutiveSummaryFromDigest generates executive summary using category-grouped articles
// This ensures article numbering in the prompt matches the final output
// generateDigestContentFromDigest generates title, TL;DR, and executive summary
func (p *Pipeline) generateDigestContentFromDigest(ctx context.Context, digest *core.Digest) (*narrative.DigestContent, error) {
	if len(digest.ArticleGroups) == 0 {
		return nil, fmt.Errorf("no article groups in digest")
	}

	// Convert ArticleGroups to TopicClusters for narrative generator
	clusters := make([]core.TopicCluster, 0, len(digest.ArticleGroups))
	for _, group := range digest.ArticleGroups {
		articleIDs := make([]string, 0, len(group.Articles))
		for _, article := range group.Articles {
			articleIDs = append(articleIDs, article.ID)
		}
		clusters = append(clusters, core.TopicCluster{
			Label:      group.Theme,
			Keywords:   []string{group.Category},
			ArticleIDs: articleIDs,
		})
	}

	// Build article and summary maps
	articleMap := make(map[string]core.Article)
	for _, group := range digest.ArticleGroups {
		for _, article := range group.Articles {
			articleMap[article.ID] = article
		}
	}

	summaryMap := summariesToMap(digest.Summaries)

	// Generate content using narrative generator
	type ContentGenerator interface {
		GenerateDigestContent(ctx context.Context, clusters []core.TopicCluster, articles map[string]core.Article, summaries map[string]core.Summary) (*narrative.DigestContent, error)
	}

	gen, ok := p.narrative.(ContentGenerator)
	if !ok {
		return nil, fmt.Errorf("narrative generator does not support digest content generation")
	}

	return gen.GenerateDigestContent(ctx, clusters, articleMap, summaryMap)
}

// checkArticleCache checks if an article and its summary are cached
func (p *Pipeline) checkArticleCache(url string) (*core.Article, *core.Summary, error) {
	if p.cache == nil {
		return nil, nil, fmt.Errorf("cache not available")
	}
	return p.cache.GetArticleWithSummary(url, p.config.CacheTTL)
}

// cacheArticle stores an article and its summary in cache
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
		// Map by article ID, not summary ID, for narrative generator
		// Summaries have ArticleIDs array, use first one
		if len(summary.ArticleIDs) > 0 {
			result[summary.ArticleIDs[0]] = summary
		}
	}
	return result
}

func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}