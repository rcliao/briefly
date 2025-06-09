package handlers

import (
	"briefly/internal/alerts"
	"briefly/internal/clustering"
	"briefly/internal/core"
	"briefly/internal/cost"
	"briefly/internal/fetch"
	"briefly/internal/llm"
	"briefly/internal/logger"
	"briefly/internal/messaging"
	"briefly/internal/render"
	"briefly/internal/sentiment"
	"briefly/internal/store"
	"briefly/internal/templates"
	"briefly/internal/tts"
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// NewDigestCmd creates the consolidated digest command with all format options
func NewDigestCmd() *cobra.Command {
	digestCmd := &cobra.Command{
		Use:   "digest [input-file|URL]",
		Short: "Generate digest with Smart Headlines from URLs or single article",
		Long: `Process URLs from a markdown file or summarize a single URL, with multiple output formats.

Features:
- Smart Headlines: Automatically generates compelling, content-based titles
- AI-powered insights: Sentiment analysis, alerts, and trend analysis
- Multiple formats: brief, standard, detailed, newsletter, email, slack, discord, audio, condensed
- Interactive my-take workflow: Add personal commentary
- Single article mode: Process just one URL

Examples:
  # Standard digest generation
  briefly digest input/links.md
  briefly digest --format newsletter --output digests input/links.md
  
  # Single article processing  
  briefly digest --single https://example.com/article
  
  # Multi-channel outputs
  briefly digest --format slack input/links.md
  briefly digest --format discord input/links.md  
  briefly digest --format audio input/links.md
  
  # Interactive workflow
  briefly digest --interactive input/links.md
  
  # Cost estimation
  briefly digest --dry-run input/links.md
  
  # List available formats
  briefly digest --list-formats`,
		Run: digestRunFunc,
	}

	// Add flags
	digestCmd.Flags().StringP("output", "o", "digests", "Output directory for digest file")
	digestCmd.Flags().StringP("format", "f", "standard", "Digest format: brief, standard, detailed, newsletter, email, slack, discord, audio, condensed")
	digestCmd.Flags().Bool("dry-run", false, "Estimate costs without making API calls")
	digestCmd.Flags().Bool("list-formats", false, "List available output formats")
	digestCmd.Flags().Bool("single", false, "Process single URL instead of input file")
	digestCmd.Flags().Bool("interactive", false, "Interactive my-take workflow")
	digestCmd.Flags().Bool("with-banner", false, "Generate AI banner image (future feature)")
	digestCmd.Flags().String("style-guide", "", "Path to personal style guide file")

	// Messaging platform flags
	digestCmd.Flags().String("slack-webhook", "", "Slack webhook URL for slack format")
	digestCmd.Flags().String("discord-webhook", "", "Discord webhook URL for discord format")
	digestCmd.Flags().String("messaging-format", "bullets", "Messaging format: bullets, summary, highlights")

	// TTS flags
	digestCmd.Flags().String("tts-provider", "openai", "TTS provider: openai, elevenlabs, google, mock")
	digestCmd.Flags().String("tts-voice", "alloy", "Voice for TTS generation")
	digestCmd.Flags().Float32("tts-speed", 1.0, "Speech speed (0.25-4.0)")
	digestCmd.Flags().Int("tts-max-articles", 10, "Maximum articles for TTS")

	return digestCmd
}

func digestRunFunc(cmd *cobra.Command, args []string) {
	// Handle list-formats flag
	if listFormats, _ := cmd.Flags().GetBool("list-formats"); listFormats {
		printAvailableFormats()
		return
	}

	// Validate arguments
	single, _ := cmd.Flags().GetBool("single")
	if single {
		if len(args) != 1 {
			fmt.Fprintf(os.Stderr, "Error: single mode requires exactly one URL argument\n")
			os.Exit(1)
		}
		if err := runSingleArticle(cmd, args[0]); err != nil {
			logger.Error("Failed to process single article", err)
			os.Exit(1)
		}
		return
	}

	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "Error: digest command requires exactly one input file argument\n")
		os.Exit(1)
	}

	inputFile := args[0]
	outputDir, _ := cmd.Flags().GetString("output")
	format, _ := cmd.Flags().GetString("format")

	// If no output directory specified via flag, try config then default
	if outputDir == "digests" { // Default value
		configOutputDir := viper.GetString("output.directory")
		if configOutputDir != "" {
			outputDir = configOutputDir
		}
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	interactive, _ := cmd.Flags().GetBool("interactive")

	if err := runDigest(cmd, inputFile, outputDir, format, dryRun, interactive); err != nil {
		logger.Error("Failed to generate digest", err)
		os.Exit(1)
	}
}

func printAvailableFormats() {
	fmt.Println("Available digest formats:")
	fmt.Println("  brief      - Short summaries with key points")
	fmt.Println("  standard   - Standard digest with full summaries")
	fmt.Println("  detailed   - Detailed digest with analysis")
	fmt.Println("  newsletter - Newsletter format with executive summary")
	fmt.Println("  condensed  - Bite-size format for 30-second reading")
	fmt.Println("  email      - HTML email format")
	fmt.Println("  slack      - Slack messaging format")
	fmt.Println("  discord    - Discord messaging format")
	fmt.Println("  audio      - Text-to-Speech audio generation")
}

func runSingleArticle(cmd *cobra.Command, url string) error {
	logger.Info("Processing single article", "url", url)
	fmt.Printf("Processing single article: %s\n", url)

	// Initialize LLM client
	llmClient, err := llm.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	defer llmClient.Close()

	// Create link from URL
	link := core.Link{URL: url}

	// Fetch article
	article, err := fetch.FetchArticle(link)
	if err != nil {
		return fmt.Errorf("failed to fetch article: %w", err)
	}

	// Clean article content
	if err := fetch.CleanArticleHTML(&article); err != nil {
		return fmt.Errorf("failed to clean article HTML: %w", err)
	}

	// Get format for summarization
	format, _ := cmd.Flags().GetString("format")
	if format == "slack" || format == "discord" || format == "audio" {
		format = "standard" // Use standard format for LLM processing
	}

	// Generate summary
	summary, err := llmClient.SummarizeArticleTextWithFormat(article, format)
	if err != nil {
		return fmt.Errorf("failed to summarize article: %w", err)
	}

	// Output to terminal
	fmt.Printf("\n✅ %s\n\n", article.Title)
	fmt.Printf("Summary: %s\n\n", summary.SummaryText)
	fmt.Printf("Source: %s\n", url)

	return nil
}

func runDigest(cmd *cobra.Command, inputFile, outputDir, format string, dryRun, interactive bool) error {
	logger.Info("Starting digest generation", "input_file", inputFile, "format", format, "dry_run", dryRun, "interactive", interactive)

	// Read links from input file
	links, err := fetch.ReadLinksFromFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read links from file: %w", err)
	}

	logger.Info("Found links", "count", len(links))
	if len(links) == 0 {
		logger.Warn("No valid links found in input file")
		return nil
	}

	if dryRun {
		return runCostEstimation(links)
	}

	// Process articles and generate digest
	digestItems, processedArticles, err := processArticles(links, format)
	if err != nil {
		return fmt.Errorf("failed to process articles: %w", err)
	}

	if len(digestItems) == 0 {
		logger.Warn("No articles were successfully processed")
		fmt.Println("⚠️  No articles were successfully processed")
		return nil
	}

	// Generate insights
	insightsData, err := generateInsights(processedArticles)
	if err != nil {
		logger.Warn("Failed to generate insights", "error", err)
	}

	// Load style guide if specified
	styleGuide, err := loadStyleGuide(cmd)
	if err != nil {
		logger.Warn("Failed to load style guide", "error", err)
	}

	// Generate final digest based on format
	digestPath, err := generateOutput(cmd, digestItems, processedArticles, insightsData, outputDir, format, styleGuide)
	if err != nil {
		return fmt.Errorf("failed to generate output: %w", err)
	}

	// Handle interactive my-take workflow
	if interactive && digestPath != "" {
		return handleInteractiveMyTake(digestPath, styleGuide)
	}

	fmt.Printf("✅ %s digest generated: %s\n", format, digestPath)
	return nil
}

func runCostEstimation(links []core.Link) error {
	logger.Info("Dry run mode - performing cost estimation", "links_count", len(links))

	model := viper.GetString("gemini.model")
	if model == "" {
		model = "gemini-2.5-flash-preview-05-20"
	}

	estimate, err := cost.EstimateDigestCost(links, model)
	if err != nil {
		return fmt.Errorf("failed to estimate costs: %w", err)
	}

	fmt.Print(estimate.FormatEstimate())
	return nil
}

func processArticles(links []core.Link, format string) ([]render.DigestData, []core.Article, error) {
	// Initialize cache store
	cacheStore, err := store.NewStore(".briefly-cache")
	if err != nil {
		logger.Error("Failed to initialize cache store", err)
		fmt.Printf("⚠️  Cache disabled due to error: %s\n", err)
		cacheStore = nil
	} else {
		defer func() {
			if err := cacheStore.Close(); err != nil {
				logger.Error("Failed to close cache store", err)
			}
		}()
		logger.Info("Cache store initialized")
	}

	// Initialize LLM client
	llmClient, err := llm.NewClient("")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	defer llmClient.Close()

	var digestItems []render.DigestData
	var processedArticles []core.Article
	cacheHits := 0
	cacheMisses := 0

	// Process each link
	for i, link := range links {
		logger.Info("Processing link", "index", i+1, "total", len(links), "url", link.URL)
		fmt.Printf("Processing %d/%d: %s\n", i+1, len(links), link.URL)

		var article core.Article
		var summary core.Summary
		var usedCache bool

		// Try to get article from cache first
		if cacheStore != nil {
			cachedArticle, err := cacheStore.GetCachedArticle(link.URL, 24*time.Hour)
			if err != nil {
				logger.Error("Cache lookup error", err)
			} else if cachedArticle != nil {
				article = *cachedArticle
				usedCache = true
				cacheHits++
				fmt.Printf("📦 Using cached article: %s\n", cachedArticle.Title)
			}
		}

		// Fetch article if not in cache
		if !usedCache {
			fetchedArticle, err := fetch.FetchArticle(link)
			if err != nil {
				logger.Error("Failed to fetch article", err, "url", link.URL)
				fmt.Printf("❌ Failed to fetch: %s\n", link.URL)
				continue
			}
			article = fetchedArticle
			cacheMisses++

			// Clean article content
			err = fetch.CleanArticleHTML(&article)
			if err != nil {
				logger.Error("Failed to clean article HTML", err, "url", link.URL)
				fmt.Printf("❌ Failed to parse: %s\n", link.URL)
				continue
			}

			// Cache the cleaned article
			if cacheStore != nil {
				article.ID = uuid.NewString()
				article.LinkID = link.URL
				article.DateFetched = time.Now().UTC()

				if err := cacheStore.CacheArticle(article); err != nil {
					logger.Error("Failed to cache article", err)
				}
			}
		}

		// Check for cached summary
		contentHash := fmt.Sprintf("%d-%s-%s", len(article.CleanedText), link.URL, format)
		var summaryFromCache bool

		if cacheStore != nil && usedCache {
			cachedSummary, err := cacheStore.GetCachedSummary(link.URL, contentHash, 7*24*time.Hour)
			if err != nil {
				logger.Error("Summary cache lookup error", err)
			} else if cachedSummary != nil {
				summary = *cachedSummary
				summaryFromCache = true
				fmt.Printf("📦 Using cached summary\n")
			}
		}

		// Generate summary if not cached
		if !summaryFromCache {
			generatedSummary, err := llmClient.SummarizeArticleTextWithFormat(article, format)
			if err != nil {
				logger.Error("Failed to summarize article", err, "url", link.URL)
				fmt.Printf("❌ Failed to summarize: %s\n", link.URL)
				continue
			}
			summary = generatedSummary
			summary.ID = uuid.NewString()
			summary.DateGenerated = time.Now().UTC()

			// Cache the summary
			if cacheStore != nil {
				if err := cacheStore.CacheSummary(summary, link.URL, contentHash); err != nil {
					logger.Error("Failed to cache summary", err)
				}
			}
		}

		// Create digest item
		digestItem := render.DigestData{
			Title:           article.Title,
			URL:             link.URL,
			SummaryText:     summary.SummaryText,
			MyTake:          article.MyTake,
			TopicCluster:    "",  // Will be populated after clustering
			TopicConfidence: 0.0, // Will be populated after clustering
		}

		digestItems = append(digestItems, digestItem)
		processedArticles = append(processedArticles, article)
		logger.Info("Successfully processed article", "title", article.Title)
		fmt.Printf("✅ %s\n", article.Title)
	}

	// Display cache statistics
	if cacheStore != nil {
		fmt.Printf("\n📊 Cache Statistics: %d hits, %d misses (%.1f%% hit rate)\n",
			cacheHits, cacheMisses, float64(cacheHits)/float64(cacheHits+cacheMisses)*100)
	}

	// Perform clustering if we have enough articles
	if len(digestItems) > 1 && llmClient != nil {
		if err := performClustering(digestItems, processedArticles, llmClient, cacheStore); err != nil {
			logger.Warn("Failed to perform clustering", "error", err)
		}
	}

	return digestItems, processedArticles, nil
}

func performClustering(digestItems []render.DigestData, processedArticles []core.Article, llmClient *llm.Client, cacheStore *store.Store) error {
	fmt.Println("\n🧮 Generating embeddings and clustering articles...")

	// Collect articles for clustering
	var articlesForClustering []core.Article
	for i, item := range digestItems {
		if i < len(processedArticles) {
			// Generate embedding for the article if not already present
			if len(processedArticles[i].Embedding) == 0 {
				embedding, err := llmClient.GenerateEmbeddingForArticle(processedArticles[i])
				if err != nil {
					logger.Warn("Failed to generate embedding for article", "title", item.Title, "error", err)
					continue
				}
				processedArticles[i].Embedding = embedding

				// Update the article in cache with embedding
				if cacheStore != nil {
					if err := cacheStore.CacheArticle(processedArticles[i]); err != nil {
						logger.Warn("Failed to update article cache with embedding", "error", err)
					}
				}
			}
			articlesForClustering = append(articlesForClustering, processedArticles[i])
		}
	}

	// Perform clustering if we have enough articles
	if len(articlesForClustering) >= 2 {
		fmt.Printf("🔍 Clustering %d articles...\n", len(articlesForClustering))

		// Auto-detect optimal number of clusters (max 5 for readability)
		maxClusters := len(articlesForClustering) / 2
		if maxClusters > 5 {
			maxClusters = 5
		}
		if maxClusters < 2 {
			maxClusters = 2
		}

		optimalClusters, err := clustering.AutoDetectOptimalClusters(articlesForClustering, maxClusters)
		if err != nil {
			logger.Warn("Failed to detect optimal clusters, using 3 clusters", "error", err)
			optimalClusters = 3
		}

		// Perform clustering
		clusterer := clustering.NewKMeansClusterer()
		clusters, err := clusterer.Cluster(articlesForClustering, optimalClusters)
		if err != nil {
			return fmt.Errorf("failed to cluster articles: %w", err)
		}

		fmt.Printf("✅ Created %d topic clusters\n", len(clusters))

		// Update articles with cluster assignments
		for _, cluster := range clusters {
			for _, articleID := range cluster.ArticleIDs {
				for j := range articlesForClustering {
					if articlesForClustering[j].ID == articleID {
						articlesForClustering[j].TopicCluster = cluster.Label
						articlesForClustering[j].TopicConfidence = 0.8
						break
					}
				}
			}
		}

		// Update cache with cluster assignments
		if cacheStore != nil {
			for _, article := range articlesForClustering {
				if article.TopicCluster != "" {
					if err := cacheStore.CacheArticle(article); err != nil {
						logger.Warn("Failed to update article cache with cluster", "error", err)
					}
				}
			}
		}

		// Print cluster summary
		for _, cluster := range clusters {
			fmt.Printf("  📂 %s: %d articles\n", cluster.Label, len(cluster.ArticleIDs))
		}

		// Update digestItems with cluster information
		for i := range digestItems {
			if i < len(articlesForClustering) {
				digestItems[i].TopicCluster = articlesForClustering[i].TopicCluster
				digestItems[i].TopicConfidence = articlesForClustering[i].TopicConfidence
			}
		}

		// Update processedArticles reference
		copy(processedArticles, articlesForClustering)
	}

	return nil
}

type InsightsData struct {
	OverallSentiment    sentiment.DigestSentiment
	TriggeredAlerts     []alerts.Alert
	ResearchSuggestions []string
	InsightsContent     string
}

func generateInsights(processedArticles []core.Article) (*InsightsData, error) {
	if len(processedArticles) == 0 {
		return &InsightsData{}, nil
	}

	fmt.Println("\n🧠 Generating insights...")

	var insightsContent strings.Builder
	insights := &InsightsData{}

	// Initialize analyzers
	sentimentAnalyzer := sentiment.NewSentimentAnalyzer()
	alertManager := alerts.NewAlertManager()

	// 1. Sentiment Analysis
	fmt.Printf("📊 Analyzing sentiment...\n")
	digestSentiment, err := sentimentAnalyzer.AnalyzeDigest(processedArticles, "digest-"+time.Now().Format("20060102"))
	if err == nil {
		insights.OverallSentiment = *digestSentiment
		sentimentSection := sentimentAnalyzer.FormatSentimentSummary(digestSentiment)
		insightsContent.WriteString(sentimentSection)
		insightsContent.WriteString("\n")
	}

	// 2. Alert Evaluation
	fmt.Printf("🚨 Evaluating alerts...\n")
	alertContext := alerts.AlertContext{
		Articles:      processedArticles,
		EstimatedCost: 0.0,
	}

	// Get current topics for alert context
	var currentTopics []string
	for _, article := range processedArticles {
		if article.TopicCluster != "" {
			currentTopics = append(currentTopics, article.TopicCluster)
		}
	}
	alertContext.CurrentTopics = currentTopics

	insights.TriggeredAlerts = alertManager.CheckConditions(alertContext)
	if len(insights.TriggeredAlerts) > 0 {
		alertSection := alertManager.FormatAlertsSection(insights.TriggeredAlerts)
		insightsContent.WriteString(alertSection)
		insightsContent.WriteString("\n")
		fmt.Printf("   ⚠️ %d alerts triggered\n", len(insights.TriggeredAlerts))
	} else {
		fmt.Printf("   ✅ No alerts triggered\n")
	}

	// 3. Research Query Generation (using LLM client)
	fmt.Printf("🔍 Generating research queries...\n")
	llmClient, err := llm.NewClient("")
	if err == nil {
		defer llmClient.Close()
		for _, article := range processedArticles {
			queries, err := llmClient.GenerateResearchQueries(article, 3)
			if err == nil {
				insights.ResearchSuggestions = append(insights.ResearchSuggestions, queries...)
			}
		}
	}

	insights.InsightsContent = insightsContent.String()
	fmt.Printf("✅ Insights generation complete\n")

	return insights, nil
}

func loadStyleGuide(cmd *cobra.Command) (string, error) {
	styleGuidePath, _ := cmd.Flags().GetString("style-guide")
	if styleGuidePath == "" {
		// Try to get from config
		styleGuidePath = viper.GetString("style.default_guide")
	}

	if styleGuidePath == "" {
		return "", nil // No style guide specified
	}

	// Check if file exists
	if _, err := os.Stat(styleGuidePath); os.IsNotExist(err) {
		return "", fmt.Errorf("style guide file not found: %s", styleGuidePath)
	}

	// Read style guide content
	content, err := os.ReadFile(styleGuidePath)
	if err != nil {
		return "", fmt.Errorf("failed to read style guide: %w", err)
	}

	return string(content), nil
}

func generateOutput(cmd *cobra.Command, digestItems []render.DigestData, processedArticles []core.Article, insightsData *InsightsData, outputDir, format, styleGuide string) (string, error) {
	switch format {
	case "slack":
		return generateSlackOutput(cmd, digestItems, insightsData)
	case "discord":
		return generateDiscordOutput(cmd, digestItems, insightsData)
	case "audio":
		return generateTTSOutput(cmd, digestItems, outputDir)
	case "condensed":
		return generateCondensedOutput(digestItems, outputDir, insightsData)
	default:
		return generateStandardOutput(digestItems, processedArticles, insightsData, outputDir, format, styleGuide)
	}
}

func generateSlackOutput(cmd *cobra.Command, digestItems []render.DigestData, insightsData *InsightsData) (string, error) {
	webhookURL, _ := cmd.Flags().GetString("slack-webhook")
	if webhookURL == "" {
		return "", fmt.Errorf("slack webhook URL required for slack format (use --slack-webhook)")
	}

	msgFormat, _ := cmd.Flags().GetString("messaging-format")

	// Validate message format
	var messageFormat messaging.MessageFormat
	switch msgFormat {
	case "bullets":
		messageFormat = messaging.FormatBullets
	case "summary":
		messageFormat = messaging.FormatSummary
	case "highlights":
		messageFormat = messaging.FormatHighlights
	default:
		messageFormat = messaging.FormatBullets
	}

	// Create messaging client
	client := messaging.NewMessagingClient(webhookURL, "")

	// Generate title
	title := fmt.Sprintf("Digest - %s", time.Now().Format("Jan 2, 2006"))

	// Send message
	err := client.SendMessage(messaging.PlatformSlack, digestItems, title, messageFormat, true)
	if err != nil {
		return "", fmt.Errorf("failed to send Slack message: %w", err)
	}

	fmt.Printf("✅ Slack message sent successfully\n")
	return "slack://sent", nil
}

func generateDiscordOutput(cmd *cobra.Command, digestItems []render.DigestData, insightsData *InsightsData) (string, error) {
	webhookURL, _ := cmd.Flags().GetString("discord-webhook")
	if webhookURL == "" {
		return "", fmt.Errorf("discord webhook URL required for discord format (use --discord-webhook)")
	}

	msgFormat, _ := cmd.Flags().GetString("messaging-format")

	// Validate message format
	var messageFormat messaging.MessageFormat
	switch msgFormat {
	case "bullets":
		messageFormat = messaging.FormatBullets
	case "summary":
		messageFormat = messaging.FormatSummary
	case "highlights":
		messageFormat = messaging.FormatHighlights
	default:
		messageFormat = messaging.FormatBullets
	}

	// Create messaging client
	client := messaging.NewMessagingClient("", webhookURL)

	// Generate title
	title := fmt.Sprintf("Digest - %s", time.Now().Format("Jan 2, 2006"))

	// Send message
	err := client.SendMessage(messaging.PlatformDiscord, digestItems, title, messageFormat, true)
	if err != nil {
		return "", fmt.Errorf("failed to send Discord message: %w", err)
	}

	fmt.Printf("✅ Discord message sent successfully\n")
	return "discord://sent", nil
}

func generateTTSOutput(cmd *cobra.Command, digestItems []render.DigestData, outputDir string) (string, error) {
	provider, _ := cmd.Flags().GetString("tts-provider")
	voice, _ := cmd.Flags().GetString("tts-voice")
	speed, _ := cmd.Flags().GetFloat32("tts-speed")
	maxArticles, _ := cmd.Flags().GetInt("tts-max-articles")

	// Validate provider
	if provider != "openai" && provider != "elevenlabs" && provider != "google" && provider != "mock" {
		return "", fmt.Errorf("unsupported TTS provider: %s (supported: openai, elevenlabs, google, mock)", provider)
	}

	// Limit articles for TTS
	if len(digestItems) > maxArticles {
		digestItems = digestItems[:maxArticles]
		fmt.Printf("ℹ️  Limited to %d articles for TTS generation\n", maxArticles)
	}

	// Create TTS configuration
	voiceConfig := tts.TTSVoice{ID: voice, Name: voice}
	config := tts.TTSConfig{
		Provider:  tts.TTSProvider(provider),
		Voice:     voiceConfig,
		Speed:     float64(speed),
		OutputDir: outputDir,
	}

	// Generate TTS
	fmt.Printf("🎵 Generating TTS audio using %s provider...\n", provider)

	// Create TTS client and generate audio
	client := tts.NewTTSClient(&config)
	ttsText := tts.PrepareTTSText(digestItems, "Weekly Digest", true, maxArticles)
	filename := fmt.Sprintf("digest_%s.mp3", time.Now().Format("2006-01-02"))
	outputPath, err := client.GenerateAudio(ttsText, filename)
	if err != nil {
		return "", fmt.Errorf("failed to generate TTS: %w", err)
	}

	fmt.Printf("✅ TTS audio generated: %s\n", outputPath)
	return outputPath, nil
}

func generateCondensedOutput(digestItems []render.DigestData, outputDir string, insightsData *InsightsData) (string, error) {
	// Implement condensed format according to Sprint 1 requirements
	// This should be a truly bite-size format (150-200 words, 30-second read)

	fmt.Printf("📝 Generating condensed digest format...\n")

	// Create condensed template data
	condensedData := struct {
		Title        string
		Date         string
		Items        []render.DigestData
		ReadingTime  string
		ArticleCount int
	}{
		Title:        generateCondensedTitle(digestItems),
		Date:         time.Now().Format("Jan 2"),
		Items:        digestItems[:min(5, len(digestItems))], // Limit to 5 items max
		ReadingTime:  "30 sec",
		ArticleCount: len(digestItems),
	}

	// Generate condensed content
	var content strings.Builder
	content.WriteString(fmt.Sprintf("# %s - Week of %s\n\n", condensedData.Title, condensedData.Date))
	content.WriteString("## 🎯 This Week's Picks\n\n")

	for i, item := range condensedData.Items {
		if i >= 3 { // Limit to 3 key items for true condensed format
			break
		}

		// Determine category emoji
		category := determineCategory(item.Title, item.SummaryText)

		// Create one-liner insight (max 60 chars)
		insight := createOneLineInsight(item.SummaryText)

		// Create actionable takeaway
		takeaway := createActionableTakeaway(item.SummaryText)

		content.WriteString(fmt.Sprintf("**%s %s**: %s\n", category, "Category", insight))
		content.WriteString(fmt.Sprintf("→ %s\n\n", takeaway))
	}

	content.WriteString("## 🚀 Try This\n")
	content.WriteString(generateCallToAction(digestItems))
	content.WriteString("\n\n---\n")
	content.WriteString(fmt.Sprintf("*%d articles, %s read • Forward to your team*\n",
		condensedData.ArticleCount, condensedData.ReadingTime))

	// Write to file
	filename := fmt.Sprintf("digest_condensed_%s.md", time.Now().Format("2006-01-02"))
	filepath := filepath.Join(outputDir, filename)

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	if err := os.WriteFile(filepath, []byte(content.String()), 0644); err != nil {
		return "", fmt.Errorf("failed to write condensed digest: %w", err)
	}

	return filepath, nil
}

func generateStandardOutput(digestItems []render.DigestData, processedArticles []core.Article, insightsData *InsightsData, outputDir, format, styleGuide string) (string, error) {
	// Initialize LLM client for final digest generation
	llmClient, err := llm.NewClient("")
	if err != nil {
		return "", fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	defer llmClient.Close()

	// Determine which template to use
	var digestFormat templates.DigestFormat
	switch format {
	case "brief":
		digestFormat = templates.FormatBrief
	case "detailed":
		digestFormat = templates.FormatDetailed
	case "newsletter":
		digestFormat = templates.FormatNewsletter
	case "email":
		digestFormat = templates.FormatEmail
	case "standard", "":
		digestFormat = templates.FormatStandard
	default:
		logger.Warn("Unknown format, using standard", "format", format)
		digestFormat = templates.FormatStandard
	}

	template := templates.GetTemplate(digestFormat)

	// Generate final digest summary using LLM
	fmt.Printf("Generating final digest summary using %s format...\n", format)
	logger.Info("Generating final digest summary", "format", format)

	// Prepare combined summaries for final digest
	var combinedSummaries strings.Builder
	for i, item := range digestItems {
		combinedSummaries.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, item.Title))
		combinedSummaries.WriteString(fmt.Sprintf("   Summary: %s\n", item.SummaryText))
		combinedSummaries.WriteString(fmt.Sprintf("   Reference URL: %s\n\n", item.URL))
	}

	// Include style guide in LLM prompt if provided
	var finalDigest string
	if styleGuide != "" {
		// Enhanced prompt with style guide
		finalDigest, err = generateDigestWithStyleGuide(llmClient, combinedSummaries.String(), format, *template, styleGuide)
	} else {
		// Standard digest generation
		finalDigest, err = generateStandardDigest(llmClient, combinedSummaries.String(), format, *template)
	}

	var generatedTitle string
	if err != nil {
		logger.Error("Failed to generate final digest", err)
		fmt.Printf("⚠️  Failed to generate final digest summary, using individual summaries: %s\n", err)
		finalDigest = ""
	}

	// Generate Smart Headline
	fmt.Printf("🎯 Generating Smart Headline...\n")
	contentForTitle := finalDigest
	if contentForTitle == "" {
		contentForTitle = combinedSummaries.String()
	}

	title, titleErr := llmClient.GenerateDigestTitle(contentForTitle, format)
	if titleErr != nil {
		logger.Error("Failed to generate digest title", titleErr)
		fmt.Printf("   ⚠️ Smart Headline generation failed, using default\n")
		generatedTitle = template.Title
	} else {
		generatedTitle = title
		fmt.Printf("   ✅ Smart Headline: \"%s\"\n", generatedTitle)
		logger.Info("Digest title generated", "title", generatedTitle)
	}

	// Prepare insights data for rendering
	var overallSentimentText, alertsSummaryText, trendsSummaryText string
	if insightsData != nil {
		overallSentimentText = fmt.Sprintf("Overall: %s (%.2f)",
			insightsData.OverallSentiment.OverallEmoji, insightsData.OverallSentiment.OverallScore.Overall)

		if len(insightsData.TriggeredAlerts) > 0 {
			alertManager := alerts.NewAlertManager()
			alertsSummaryText = alertManager.FormatAlertsSection(insightsData.TriggeredAlerts)
		} else {
			alertsSummaryText = "### ✅ Alert Monitoring\nNo alerts triggered for this digest. All articles passed through standard monitoring criteria."
		}
	}

	// Generate digest output
	var renderedContent, digestPath string
	var renderErr error

	if format == "email" {
		renderedContent, digestPath, renderErr = templates.RenderHTMLEmail(digestItems, outputDir, finalDigest, generatedTitle, overallSentimentText, alertsSummaryText, trendsSummaryText, insightsData.ResearchSuggestions, "default")
	} else {
		renderedContent, digestPath, renderErr = templates.RenderWithInsights(digestItems, outputDir, finalDigest, "", template, generatedTitle, overallSentimentText, alertsSummaryText, trendsSummaryText, insightsData.ResearchSuggestions)
	}

	if renderErr != nil {
		return "", fmt.Errorf("failed to render digest with template: %w", renderErr)
	}

	// Cache the digest
	cacheStore, err := store.NewStore(".briefly-cache")
	if err == nil {
		defer func() { _ = cacheStore.Close() }()
		digestID := uuid.NewString()
		var articleURLs []string
		for _, item := range digestItems {
			articleURLs = append(articleURLs, item.URL)
		}

		model := viper.GetString("gemini.model")
		if model == "" {
			model = "gemini-2.5-flash-preview-05-20"
		}

		if cacheErr := cacheStore.CacheDigestWithFormat(digestID, generatedTitle, renderedContent, finalDigest, format, articleURLs, model); cacheErr != nil {
			logger.Error("Failed to cache digest", cacheErr)
		}
	}

	logger.Info("Digest generated successfully", "path", digestPath, "format", format, "title", generatedTitle)
	return digestPath, nil
}

func handleInteractiveMyTake(digestPath, styleGuide string) error {
	fmt.Printf("\n🎯 Interactive My-Take Workflow\n")
	fmt.Printf("Digest generated: %s\n", digestPath)

	// Prompt user for review
	fmt.Print("Review and add your take? [Y/n]: ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "n" || response == "no" {
		fmt.Println("Skipping my-take workflow")
		return nil
	}

	// Open digest in editor for review
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim" // Default fallback
	}

	fmt.Printf("Opening digest in %s for review...\n", editor)
	cmd := exec.Command(editor, digestPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	// Prompt for personal take
	fmt.Print("\nYour take on this week's content: ")
	myTake, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read my-take input: %w", err)
	}

	myTake = strings.TrimSpace(myTake)
	if myTake == "" {
		fmt.Println("No take provided, skipping regeneration")
		return nil
	}

	// Regenerate digest with my-take
	fmt.Printf("Regenerating digest with your take...\n")

	// Read current digest content
	content, err := os.ReadFile(digestPath)
	if err != nil {
		return fmt.Errorf("failed to read digest file: %w", err)
	}

	// Generate new filename with my-take suffix
	dir := filepath.Dir(digestPath)
	filename := filepath.Base(digestPath)
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)
	newFilename := fmt.Sprintf("%s_with_my_take%s", nameWithoutExt, ext)
	newPath := filepath.Join(dir, newFilename)

	// Regenerate with LLM including my-take and style guide
	llmClient, err := llm.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	defer llmClient.Close()

	regeneratedContent, err := regenerateDigestWithMyTake(llmClient, string(content), myTake, styleGuide)
	if err != nil {
		return fmt.Errorf("failed to regenerate digest with my-take: %w", err)
	}

	// Write regenerated content
	if err := os.WriteFile(newPath, []byte(regeneratedContent), 0644); err != nil {
		return fmt.Errorf("failed to write regenerated digest: %w", err)
	}

	fmt.Printf("✅ Regenerated with your take: %s\n", newPath)
	return nil
}

// Helper functions for condensed format
func generateCondensedTitle(digestItems []render.DigestData) string {
	// Simple title generation - could be enhanced with LLM later
	return "Dev Insights & Quick Wins"
}

func determineCategory(title, summary string) string {
	// Simple category detection - could be enhanced with LLM
	titleLower := strings.ToLower(title)
	summaryLower := strings.ToLower(summary)

	if strings.Contains(titleLower, "dev") || strings.Contains(summaryLower, "development") {
		return "🔧"
	} else if strings.Contains(titleLower, "research") || strings.Contains(summaryLower, "study") {
		return "📚"
	} else {
		return "💡"
	}
}

func createOneLineInsight(summary string) string {
	// Extract first sentence or create summary
	sentences := strings.Split(summary, ".")
	if len(sentences) > 0 && len(sentences[0]) <= 60 {
		return strings.TrimSpace(sentences[0])
	}

	// Truncate if too long
	if len(summary) > 60 {
		return summary[:57] + "..."
	}
	return summary
}

func createActionableTakeaway(summary string) string {
	// Simple extraction - could be enhanced with LLM
	if strings.Contains(summary, "should") {
		parts := strings.Split(summary, "should")
		if len(parts) > 1 {
			return "Should" + strings.Split(parts[1], ".")[0]
		}
	}
	return "Worth exploring for your next project"
}

func generateCallToAction(digestItems []render.DigestData) string {
	// Simple CTA generation - could be enhanced
	return "Try implementing one of these insights in your current project"
}

func generateDigestWithStyleGuide(llmClient *llm.Client, content, format string, template templates.DigestTemplate, styleGuide string) (string, error) {
	// This would need to be implemented in the LLM client
	// For now, fall back to standard generation
	return generateStandardDigest(llmClient, content, format, template)
}

func generateStandardDigest(llmClient *llm.Client, content, format string, template templates.DigestTemplate) (string, error) {
	// Implementation would call the existing LLM digest generation
	// This is a placeholder - the actual implementation would need to be moved from the old root.go
	return "", fmt.Errorf("digest generation not yet implemented in new architecture")
}

func regenerateDigestWithMyTake(llmClient *llm.Client, originalContent, myTake, styleGuide string) (string, error) {
	// This would need to be implemented to regenerate digest with personal take
	// For now, append my-take to original content
	return originalContent + "\n\n## My Take\n\n" + myTake, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
