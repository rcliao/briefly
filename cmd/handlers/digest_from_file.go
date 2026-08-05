package handlers

import (
	"briefly/internal/config"
	"briefly/internal/core"
	"briefly/internal/fetch"
	"briefly/internal/llm"
	"briefly/internal/logger"
	"briefly/internal/narrative"
	"briefly/internal/parser"
	"briefly/internal/store"
	"briefly/internal/summarize"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

// NewDigestFromFileCmd creates the digest from-file command for processing curated markdown files
func NewDigestFromFileCmd() *cobra.Command {
	var (
		outputDir      string
		numClusters    int
		noCache        bool
		themeThreshold float64
		outputFormat   string
	)

	cmd := &cobra.Command{
		Use:   "from-file <input.md>",
		Short: "Generate digest from curated markdown file",
		Long: `Generate a digest from a curated markdown file containing URLs.

This command:
  • Parses URLs from a markdown file
  • Fetches articles in parallel (HTML, PDF, YouTube)
  • Generates summaries using LLM
  • Runs a single editorial pass: topic grouping, per-article takeaways,
    must-read pick, executive summary
  • Renders scannable markdown (or Slack format with --format slack)
  • Reports any curated links that failed to fetch
  • No database persistence (lightweight, in-memory processing)

Perfect for weekly digests from manually curated URLs.

Examples:
  # Generate digest from curated file
  briefly digest from-file input/weekly.md

  # Custom output directory
  briefly digest from-file input/weekly.md --output my-digests

  # Disable caching (fresh fetch)
  briefly digest from-file input/weekly.md --no-cache

  # Generate Slack-optimized digest
  briefly digest from-file input/weekly.md --format slack`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDigestFromFile(cmd.Context(), args[0], outputDir, noCache, outputFormat)
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output", "o", "digests", "Output directory for digest file")
	cmd.Flags().IntVar(&numClusters, "clusters", 0, "Deprecated: topic grouping is now editorial, not k-means")
	cmd.Flags().BoolVar(&noCache, "no-cache", false, "Disable caching (fetch fresh content)")
	cmd.Flags().Float64Var(&themeThreshold, "theme-threshold", 0.4, "Deprecated: theme classification removed from this pipeline")
	cmd.Flags().StringVar(&outputFormat, "format", "markdown", "Output format: markdown (default), slack")
	_ = cmd.Flags().MarkDeprecated("clusters", "topic grouping is now done by the editorial LLM pass")
	_ = cmd.Flags().MarkDeprecated("theme-threshold", "theme classification was removed from this pipeline")

	return cmd
}

// fetchWorkers bounds fetch/summarize concurrency: fast without hammering
// hosts or the LLM API rate limits.
const fetchWorkers = 4

func runDigestFromFile(ctx context.Context, inputFile string, outputDir string, noCache bool, outputFormat string) error {
	startTime := time.Now()
	log := logger.Get()
	log.Info("Starting digest generation from file",
		"input_file", inputFile,
		"output_dir", outputDir,
		"no_cache", noCache,
		"format", outputFormat,
	)

	// Load configuration
	_, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := config.Get()

	// Validate input file
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file not found: %s", inputFile)
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Initialize LLM client
	modelName := cfg.AI.Gemini.Model
	if modelName == "" {
		modelName = "gemini-3-flash-preview"
	}

	fmt.Printf("🔧 Initializing AI client (model: %s)...\n", modelName)
	llmClient, err := llm.NewClient(modelName)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}
	defer llmClient.Close()

	// Initialize cache (unless disabled)
	var cache *store.Store
	if !noCache {
		cacheDir := cfg.Cache.Directory
		if cacheDir == "" {
			cacheDir = ".briefly-cache"
		}
		cache, err = store.NewStore(cacheDir)
		if err != nil {
			log.Warn("Failed to initialize cache, continuing without cache", "error", err)
		} else {
			defer cache.Close()
			fmt.Println("   ✓ Cache initialized")
		}
	}

	// Step 1: Parse URLs from markdown file
	fmt.Printf("\n📄 Step 1/5: Parsing URLs from %s...\n", inputFile)
	urlParser := parser.NewParser()
	links, err := urlParser.ParseMarkdownFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to parse markdown file: %w", err)
	}

	if len(links) == 0 {
		fmt.Println("⚠️  No URLs found in markdown file")
		return nil
	}

	fmt.Printf("   ✓ Found %d URLs\n", len(links))

	// Step 2: Fetch articles in parallel, preserving input order for citations
	fmt.Printf("\n🔍 Step 2/5: Fetching articles (%d workers)...\n", fetchWorkers)
	processor := fetch.NewContentProcessor()

	type fetchResult struct {
		article *core.Article
		failure *failedLink
	}
	fetchResults := make([]fetchResult, len(links))

	var wg sync.WaitGroup
	sem := make(chan struct{}, fetchWorkers)
	var cacheMu sync.Mutex

	for i, link := range links {
		wg.Add(1)
		go func(i int, link core.Link) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Check cache first
			var article *core.Article
			if cache != nil {
				cacheMu.Lock()
				cachedArticle, err := cache.GetCachedArticle(link.URL, 24*time.Hour)
				cacheMu.Unlock()
				if err == nil && cachedArticle != nil {
					article = cachedArticle
					// Re-extract with the current parser: older cache entries
					// carry duplicated/junk text from a previous extractor bug.
					if article.FetchedHTML != "" {
						if err := fetch.ParseArticleContent(article); err == nil {
							article.EstimatedReadMinutes = fetch.CalculateReadingTime(article)
						}
					} else if article.EstimatedReadMinutes == 0 {
						article.EstimatedReadMinutes = fetch.CalculateReadingTime(article)
					}
					fmt.Printf("   ✓ [%d/%d] Cache hit: %s\n", i+1, len(links), link.URL)
				}
			}

			if article == nil {
				fetchedArticle, err := processor.ProcessArticle(ctx, link.URL)
				if err != nil {
					log.Warn("Failed to fetch article", "url", link.URL, "error", err)
					fmt.Printf("   ⚠ [%d/%d] Fetch failed: %s (%v)\n", i+1, len(links), link.URL, err)
					fetchResults[i] = fetchResult{failure: &failedLink{URL: link.URL, Reason: fmt.Sprintf("fetch failed: %v", err)}}
					return
				}
				article = fetchedArticle

				if cache != nil {
					cacheMu.Lock()
					if err := cache.SaveArticle(article); err != nil {
						log.Warn("Failed to cache article", "url", link.URL, "error", err)
					}
					cacheMu.Unlock()
				}
				fmt.Printf("   ✓ [%d/%d] Fetched: %s\n", i+1, len(links), link.URL)
			}

			fetchResults[i] = fetchResult{article: article}
		}(i, link)
	}
	wg.Wait()

	articles := make([]core.Article, 0, len(links))
	failed := make([]failedLink, 0)
	for _, r := range fetchResults {
		switch {
		case r.article != nil:
			articles = append(articles, *r.article)
		case r.failure != nil:
			failed = append(failed, *r.failure)
		}
	}

	if len(articles) == 0 {
		fmt.Println("\n⚠️  No articles could be fetched")
		return nil
	}

	fmt.Printf("   ✓ Successfully fetched %d/%d articles\n", len(articles), len(links))

	// Step 3: Generate summaries in parallel
	fmt.Printf("\n📝 Step 3/5: Generating article summaries (%d workers)...\n", fetchWorkers)
	adapter := &llmClientAdapter{client: llmClient}
	summarizer := summarize.NewSummarizerWithDefaults(adapter)

	summaries := make([]*core.Summary, len(articles))
	for i := range articles {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			summary, err := summarizer.SummarizeArticle(ctx, &articles[i])
			if err != nil {
				log.Warn("Failed to generate summary", "article_id", articles[i].ID, "error", err)
				fmt.Printf("   ⚠ [%d/%d] Summary failed: %s (%v)\n", i+1, len(articles), articles[i].Title, err)
				summary = &core.Summary{
					ID:          uuid.NewString(),
					ArticleIDs:  []string{articles[i].ID},
					SummaryText: fmt.Sprintf("Summary for: %s", articles[i].Title),
					ModelUsed:   "fallback",
				}
			} else {
				fmt.Printf("   ✓ [%d/%d] Summarized: %s\n", i+1, len(articles), articles[i].Title)
			}
			summaries[i] = summary
		}(i)
	}
	wg.Wait()

	articleSummaries := make(map[string]*core.Summary, len(articles))
	for i := range articles {
		articleSummaries[articles[i].ID] = summaries[i]
	}

	// Step 4: Single editorial pass — topic grouping, one-liners, must-read,
	// executive summary. Replaces theme classification + embeddings + k-means
	// + per-cluster narratives from the previous pipeline.
	fmt.Printf("\n✨ Step 4/5: Running editorial pass (grouping, takeaways, must-read)...\n")

	editorial, err := generateEditorialDigest(ctx, llmClient, articles, articleSummaries)
	if err != nil {
		log.Warn("Editorial pass failed, using fallback structure", "error", err)
		fmt.Printf("   ⚠ Editorial pass failed (%v), using fallback structure\n", err)
		editorial = fallbackEditorialDigest(articles, articleSummaries)
	}

	fmt.Printf("   ✓ %s\n", editorial.Title)
	for _, topic := range editorial.Topics {
		fmt.Printf("      %s %s (%d articles)\n", topic.Emoji, topic.Name, len(topic.Citations))
	}

	// Handle Slack format using editorial topics as clusters
	if outputFormat == "slack" {
		articleMap := make(map[string]core.Article, len(articles))
		summaryMap := make(map[string]core.Summary, len(articles))
		for i := range articles {
			articleMap[articles[i].ID] = articles[i]
			summaryMap[articles[i].ID] = *summaries[i]
		}
		clusters := make([]core.TopicCluster, 0, len(editorial.Topics))
		for _, topic := range editorial.Topics {
			ids := make([]string, 0, len(topic.Citations))
			for _, c := range topic.Citations {
				ids = append(ids, articles[c-1].ID)
			}
			clusters = append(clusters, core.TopicCluster{Label: topic.Name, ArticleIDs: ids})
		}
		narrativeGen := narrative.NewGenerator(&narrativeLLMAdapter{client: llmClient})
		return generateSlackDigest(ctx, narrativeGen, clusters, articleMap, summaryMap, articles, outputDir, startTime, inputFile, len(links))
	}

	// Step 5: Render markdown
	fmt.Printf("\n📄 Step 5/5: Rendering markdown digest...\n")

	now := time.Now()
	content := renderEditorialDigest(editorial, articles, failed, now)
	outputPath := fmt.Sprintf("%s/digest_%s.md", outputDir, now.Format("2006-01-02"))
	if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write digest: %w", err)
	}

	fmt.Printf("   ✓ Saved: %s\n", outputPath)

	duration := time.Since(startTime)
	fmt.Printf("\n✅ Successfully generated digest!\n")
	fmt.Printf("   Title: %s\n", editorial.Title)
	fmt.Printf("   Input file: %s\n", inputFile)
	fmt.Printf("   Total URLs: %d\n", len(links))
	fmt.Printf("   Articles included: %d\n", len(articles))
	if len(failed) > 0 {
		fmt.Printf("   ⚠️  Links not included: %d (listed in digest footer)\n", len(failed))
		for _, f := range failed {
			fmt.Printf("      - %s (%s)\n", f.URL, f.Reason)
		}
	}
	fmt.Printf("   Output file: %s\n", outputPath)
	fmt.Printf("   Duration: %s\n", duration.Round(time.Millisecond))

	fmt.Println("\n💡 Next steps:")
	fmt.Println("   • Review the digest:", outputPath)
	fmt.Println("   • Edit and refine as needed")
	fmt.Println("   • Share on LinkedIn or your preferred platform")

	return nil
}

// fallbackEditorialDigest builds a minimal but complete digest structure when
// the editorial LLM call fails: one topic, one-liners from summaries.
func fallbackEditorialDigest(articles []core.Article, summaries map[string]*core.Summary) *editorialDigest {
	d := &editorialDigest{
		Title: fmt.Sprintf("This Week in GenAI — %d Links", len(articles)),
	}
	citations := make([]int, 0, len(articles))
	for i := range articles {
		citations = append(citations, i+1)
	}
	d.Topics = []editorialTopic{{Name: "This Week", Emoji: "🗞️", Citations: citations}}
	normalizeEditorialDigest(d, articles, summaries)
	return d
}

// generateSlackDigest handles Slack format digest generation
func generateSlackDigest(ctx context.Context, narrativeGen *narrative.Generator, clusters []core.TopicCluster, articleMap map[string]core.Article, summaryMap map[string]core.Summary, articles []core.Article, outputDir string, startTime time.Time, inputFile string, totalLinks int) error {
	log := logger.Get()

	fmt.Printf("\n📱 Step 8/9: Generating Slack-formatted digest...\n")

	slackContent, err := narrativeGen.GenerateSlackDigest(ctx, clusters, articleMap, summaryMap)
	if err != nil {
		log.Error("Failed to generate Slack digest", "error", err)
		return fmt.Errorf("failed to generate Slack digest: %w", err)
	}

	fmt.Printf("   ✓ Generated Slack digest: %s\n", slackContent.WeekRange)
	fmt.Printf("      Big 3: %d items\n", len(slackContent.Big3))
	fmt.Printf("      Also on radar: %d items\n", len(slackContent.AlsoOnRadar))
	fmt.Printf("      Thread content: %d items\n", len(slackContent.ThreadContent))

	// Step 9: Render Slack format
	fmt.Printf("\n📄 Step 9/9: Rendering Slack markdown...\n")

	output := renderSlackFormat(slackContent, articles, clusters)

	// Save to file
	timestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("digest_slack_%s.md", timestamp)
	outputPath := fmt.Sprintf("%s/%s", outputDir, filename)

	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write Slack digest: %w", err)
	}

	fmt.Printf("   ✓ Saved: %s\n", outputPath)

	duration := time.Since(startTime)

	// Print summary
	fmt.Printf("\n✅ Successfully generated Slack digest!\n")
	fmt.Printf("   Week: %s\n", slackContent.WeekRange)
	fmt.Printf("   Input file: %s\n", inputFile)
	fmt.Printf("   Total URLs: %d\n", totalLinks)
	fmt.Printf("   Articles fetched: %d\n", len(articles))
	fmt.Printf("   Output file: %s\n", outputPath)
	fmt.Printf("   Duration: %s\n", duration.Round(time.Millisecond))

	fmt.Println("\n💡 Next steps:")
	fmt.Println("   • Copy the main content to Slack")
	fmt.Println("   • Post thread content as replies")
	fmt.Println("   • File:", outputPath)

	return nil
}

// SlackMessageChunk represents a chunked message for Slack
type SlackMessageChunk struct {
	Title   string // e.g., "Thread 1/3", "Thread 2/3"
	Content string
}

// SlackChunkLimit is the max characters per Slack message (leaving buffer for formatting)
const SlackChunkLimit = 3000

// renderSlackFormat renders SlackDigestContent to Slack mrkdwn format with chunked thread content
func renderSlackFormat(content *narrative.SlackDigestContent, articles []core.Article, clusters []core.TopicCluster) string {
	var out strings.Builder

	// Build article URL map (1-based citation number -> URL)
	articleURLs := buildArticleURLMap(articles, clusters)

	// Header
	out.WriteString(fmt.Sprintf("🤖 *AI Weekly* — %s\n\n", content.WeekRange))

	// Big 3 Section
	out.WriteString("*🔥 This Week's Big 3*\n\n")
	for _, item := range content.Big3 {
		url := getArticleURL(articleURLs, item.ArticleNum)
		out.WriteString(fmt.Sprintf("*%s* — %s\n%s\n\n", item.Headline, item.Editorial, url))
	}

	// Separator
	out.WriteString("---\n")

	// Also on my radar
	out.WriteString("*📌 Also on my radar* (links in thread)\n")
	for _, item := range content.AlsoOnRadar {
		out.WriteString(fmt.Sprintf("- %s\n", item.Title))
	}

	// Chunk thread content for Slack message limits
	chunks := chunkThreadContent(content.ThreadContent, articleURLs, SlackChunkLimit)

	// Thread content (chunked for multiple messages)
	for i, chunk := range chunks {
		if len(chunks) > 1 {
			out.WriteString(fmt.Sprintf("\n---\n*🧵 Thread %d/%d*\n\n", i+1, len(chunks)))
		} else {
			out.WriteString("\n---\n*🧵 Thread: More Details*\n\n")
		}
		out.WriteString(chunk)
	}

	return out.String()
}

// chunkThreadContent splits thread items into chunks that fit within Slack's character limit
func chunkThreadContent(items []narrative.ThreadItem, articleURLs map[int]string, maxChars int) []string {
	if len(items) == 0 {
		return []string{}
	}

	chunks := make([]string, 0)
	var currentChunk strings.Builder
	itemIndex := 1

	for _, item := range items {
		url := getArticleURL(articleURLs, item.ArticleNum)
		itemContent := fmt.Sprintf("[%d] *%s*\n%s\n%s\n\n", itemIndex, item.Title, item.Explanation, url)

		// Check if adding this item would exceed the limit
		if currentChunk.Len()+len(itemContent) > maxChars && currentChunk.Len() > 0 {
			// Save current chunk and start new one
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()
		}

		currentChunk.WriteString(itemContent)
		itemIndex++
	}

	// Don't forget the last chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	return chunks
}

// buildArticleURLMap creates a map from citation number (1-based) to article URL
func buildArticleURLMap(articles []core.Article, clusters []core.TopicCluster) map[int]string {
	urlMap := make(map[int]string)
	articleNum := 1

	for _, cluster := range clusters {
		for _, articleID := range cluster.ArticleIDs {
			for _, article := range articles {
				if article.ID == articleID {
					urlMap[articleNum] = article.URL
					articleNum++
					break
				}
			}
		}
	}

	return urlMap
}

// getArticleURL safely retrieves URL for citation number
func getArticleURL(urlMap map[int]string, articleNum int) string {
	if url, found := urlMap[articleNum]; found {
		return url
	}
	return fmt.Sprintf("[Article %d URL not found]", articleNum)
}
