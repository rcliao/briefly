/*
Copyright © 2025 Your Name

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"briefly/internal/alerts"
	"briefly/internal/clustering"
	"briefly/internal/core"
	"briefly/internal/cost"
	"briefly/internal/deepresearch"
	"briefly/internal/feeds"
	"briefly/internal/fetch"
	"briefly/internal/llm"
	"briefly/internal/logger"
	"briefly/internal/messaging"
	"briefly/internal/render"
	"briefly/internal/research"
	"briefly/internal/search"
	"briefly/internal/sentiment"
	"briefly/internal/store"
	"briefly/internal/templates"
	"briefly/internal/trends"
	"briefly/internal/tts"
	"briefly/internal/tui"
	"briefly/llmclient"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "briefly",
	Short: "Briefly is a CLI tool for fetching, summarizing, and managing articles.",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.briefly.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Load .env file if it exists (for local development)
	envFile := ".env"
	if _, err := os.Stat(envFile); err == nil {
		if err := godotenv.Load(envFile); err != nil {
			fmt.Printf("Warning: Error loading .env file: %v\n", err)
		}
	}

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in current directory and home directory
		viper.AddConfigPath(".")  // Current directory
		viper.AddConfigPath(home) // Home directory
		viper.SetConfigType("yaml")
		viper.SetConfigName(".briefly")
	}

	// Automatically bind environment variables
	viper.AutomaticEnv()

	// Set defaults for configuration
	viper.SetDefault("gemini.api_key", "")
	viper.SetDefault("gemini.model", "gemini-2.5-flash-preview-05-20")
	viper.SetDefault("output.directory", "digests")

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the Briefly Terminal User Interface",
	Long:  `Launch the Briefly TUI to browse and manage articles and summaries.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Launching TUI...")
		tui.StartTUI()
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// tuiCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// tuiCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

var digestCmd = &cobra.Command{
	Use:   "digest [input-file]",
	Short: "Generate a digest with Smart Headlines from URLs in a markdown file",
	Long: `Process URLs from a markdown file, fetch articles, summarize them using Gemini,
and generate a markdown digest file with an AI-powered Smart Headline.

Features:
- Smart Headlines: Automatically generates compelling, content-based titles
- AI-powered insights: Sentiment analysis, alerts, and trend analysis
- Multiple formats: brief, standard, detailed, newsletter, email

Example:
  briefly digest input/2025-05-30.md
  briefly digest --format newsletter --output digests input/2025-05-30.md
  briefly digest --dry-run input/2025-05-30.md`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
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

		if err := runDigest(inputFile, outputDir, format, dryRun); err != nil {
			logger.Error("Failed to generate digest", err)
			os.Exit(1)
		}
	},
}

func runDigest(inputFile, outputDir, format string, dryRun bool) error {
	logger.Info("Starting digest generation", "input_file", inputFile, "format", format, "dry_run", dryRun)

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
		logger.Info("Dry run mode - performing cost estimation", "links_count", len(links))

		// Get model from config
		model := viper.GetString("gemini.model")
		if model == "" {
			model = "gemini-2.5-flash-preview-05-20"
		}

		// Generate detailed cost estimate
		estimate, err := cost.EstimateDigestCost(links, model)
		if err != nil {
			return fmt.Errorf("failed to estimate costs: %w", err)
		}

		// Display formatted estimate
		fmt.Print(estimate.FormatEstimate())

		return nil
	}

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
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	defer llmClient.Close()

	// Initialize Insights modules
	sentimentAnalyzer := sentiment.NewSentimentAnalyzer()
	trendAnalyzer := trends.NewTrendAnalyzer()
	alertManager := alerts.NewAlertManager()

	var digestItems []render.DigestData
	var processedArticles []core.Article // Track articles for clustering and insights
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
			cachedArticle, err := cacheStore.GetCachedArticle(link.URL, 24*time.Hour) // Cache for 24 hours
			if err != nil {
				logger.Error("Cache lookup error", err)
			} else if cachedArticle != nil {
				// Use cached article directly
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
				// The article is already a core.Article, just update its metadata for caching
				article.ID = uuid.NewString()
				article.LinkID = link.URL // Use URL as LinkID for compatibility
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
			cachedSummary, err := cacheStore.GetCachedSummary(link.URL, contentHash, 7*24*time.Hour) // Cache summaries for 7 days
			if err != nil {
				logger.Error("Summary cache lookup error", err)
			} else if cachedSummary != nil {
				// Use cached summary directly
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

			// Set summary metadata
			summary.ID = uuid.NewString()
			summary.DateGenerated = time.Now().UTC()

			// Cache the summary
			if cacheStore != nil {
				if err := cacheStore.CacheSummary(summary, link.URL, contentHash); err != nil {
					logger.Error("Failed to cache summary", err)
				}
			}
		}

		// Create digest item (topic cluster info will be added later after clustering)
		digestItem := render.DigestData{
			Title:           article.Title,
			URL:             link.URL,
			SummaryText:     summary.SummaryText,
			MyTake:          article.MyTake,
			TopicCluster:    "",  // Will be populated after clustering
			TopicConfidence: 0.0, // Will be populated after clustering
		}

		digestItems = append(digestItems, digestItem)
		processedArticles = append(processedArticles, article) // Track for clustering
		logger.Info("Successfully processed article", "title", article.Title)
		fmt.Printf("✅ %s\n", article.Title)
	}

	// Display cache statistics
	if cacheStore != nil {
		fmt.Printf("\n📊 Cache Statistics: %d hits, %d misses (%.1f%% hit rate)\n",
			cacheHits, cacheMisses, float64(cacheHits)/float64(cacheHits+cacheMisses)*100)
	}

	// Generate embeddings and perform topic clustering
	if len(digestItems) > 1 && llmClient != nil {
		fmt.Println("\n🧮 Generating embeddings and clustering articles...")

		// Collect articles for clustering (those that were processed successfully)
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
				logger.Warn("Failed to cluster articles", "error", err)
			} else {
				fmt.Printf("✅ Created %d topic clusters\n", len(clusters))

				// Update articles with cluster assignments
				for _, cluster := range clusters {
					for _, articleID := range cluster.ArticleIDs {
						for j := range articlesForClustering {
							if articlesForClustering[j].ID == articleID {
								articlesForClustering[j].TopicCluster = cluster.Label
								articlesForClustering[j].TopicConfidence = 0.8 // Default confidence
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
			}
		}

		processedArticles = articlesForClustering
	}

	// Generate insights for the digest
	var insightsContent strings.Builder
	var overallSentiment sentiment.DigestSentiment
	var triggeredAlerts []alerts.Alert
	var researchSuggestions []string

	if len(processedArticles) > 0 {
		fmt.Println("\n🧠 Generating insights...")

		// 1. Sentiment Analysis with enhanced data population
		fmt.Printf("📊 Analyzing sentiment...\n")
		digestSentiment, err := sentimentAnalyzer.AnalyzeDigest(processedArticles, "digest-"+time.Now().Format("20060102"))
		if err != nil {
			logger.Warn("Failed to analyze digest sentiment", "error", err)
		} else {
			overallSentiment = *digestSentiment
			sentimentSection := sentimentAnalyzer.FormatSentimentSummary(digestSentiment)
			insightsContent.WriteString(sentimentSection)
			insightsContent.WriteString("\n")

			// Update digestItems with comprehensive sentiment data
			for i, articleSentiment := range digestSentiment.ArticleSentiments {
				if i < len(digestItems) {
					digestItems[i].SentimentScore = articleSentiment.Score.Overall
					digestItems[i].SentimentLabel = string(articleSentiment.Classification)
					digestItems[i].SentimentEmoji = articleSentiment.Emoji

					// Update the corresponding article in processedArticles
					if i < len(processedArticles) {
						processedArticles[i].SentimentScore = articleSentiment.Score.Overall
						processedArticles[i].SentimentLabel = string(articleSentiment.Classification)
						processedArticles[i].SentimentEmoji = articleSentiment.Emoji
					}
				}
			}
		}

		// 2. Alert Evaluation with enhanced data population
		fmt.Printf("🚨 Evaluating alerts...\n")
		alertContext := alerts.AlertContext{
			Articles:      processedArticles,
			EstimatedCost: 0.0, // Could integrate with cost estimation
		}

		// Get current topics for alert context
		var currentTopics []string
		for _, article := range processedArticles {
			if article.TopicCluster != "" {
				currentTopics = append(currentTopics, article.TopicCluster)
			}
		}
		alertContext.CurrentTopics = currentTopics

		triggeredAlerts = alertManager.CheckConditions(alertContext)
		if len(triggeredAlerts) > 0 {
			alertSection := alertManager.FormatAlertsSection(triggeredAlerts)
			insightsContent.WriteString(alertSection)
			insightsContent.WriteString("\n")

			// Update digestItems and articles with alert information
			for i := range digestItems {
				var alertConditions []string
				alertTriggered := false

				// Check if this article triggered any alerts by examining alert context
				for _, alert := range triggeredAlerts {
					if matchedArticles, ok := alert.Context["matched_articles"].([]string); ok {
						for _, matchedTitle := range matchedArticles {
							if matchedTitle == digestItems[i].Title {
								alertTriggered = true
								alertConditions = append(alertConditions, alert.Title)
							}
						}
					}
				}

				digestItems[i].AlertTriggered = alertTriggered
				digestItems[i].AlertConditions = alertConditions

				// Update corresponding article
				if i < len(processedArticles) {
					processedArticles[i].AlertTriggered = alertTriggered
					processedArticles[i].AlertConditions = alertConditions
				}
			}

			fmt.Printf("   ⚠️ %d alerts triggered\n", len(triggeredAlerts))
		} else {
			fmt.Printf("   ✅ No alerts triggered\n")
		}

		// 3. Research Query Generation
		fmt.Printf("🔍 Generating research queries...\n")
		for i := range digestItems {
			if i < len(processedArticles) {
				queries, err := llmClient.GenerateResearchQueries(processedArticles[i], 3) // Generate 3 queries per article
				if err != nil {
					logger.Warn("Failed to generate research queries", "article", processedArticles[i].Title, "error", err)
				} else {
					digestItems[i].ResearchQueries = queries
					processedArticles[i].ResearchQueries = queries
					researchSuggestions = append(researchSuggestions, queries...)
				}
			}
		}

		// 4. Trend Analysis (only if we have cached data for comparison)
		if cacheStore != nil {
			fmt.Printf("📈 Analyzing trends...\n")

			// Get articles from previous week for comparison
			previousWeekArticles, err := cacheStore.GetArticlesByDateRange(
				time.Now().AddDate(0, 0, -14), // Two weeks ago
				time.Now().AddDate(0, 0, -7),  // One week ago
			)
			if err == nil && len(previousWeekArticles) > 0 {
				// Use the updated trend analysis that works directly with articles
				trendReport, err := trendAnalyzer.AnalyzeArticleTrends(processedArticles, previousWeekArticles)
				if err != nil {
					logger.Warn("Failed to generate trend report", "error", err)
				} else {
					trendSection := trendAnalyzer.FormatReport(trendReport)
					insightsContent.WriteString("## 📈 Trend Analysis\n\n")
					insightsContent.WriteString(trendSection)
					insightsContent.WriteString("\n")
				}
			} else {
				fmt.Printf("   ℹ️ Not enough historical data for trend analysis\n")
			}
		}

		fmt.Printf("✅ Insights generation complete\n")
	}

	// Generate digest
	if len(digestItems) > 0 {
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

		// Prepare combined summaries and sources for final digest
		var combinedSummaries strings.Builder
		for i, item := range digestItems {
			combinedSummaries.WriteString(fmt.Sprintf("%d. **%s**\n", i+1, item.Title))
			combinedSummaries.WriteString(fmt.Sprintf("   Summary: %s\n", item.SummaryText))
			combinedSummaries.WriteString(fmt.Sprintf("   Reference URL: %s\n\n", item.URL))
		}

		// Get API key from environment or config (prefer environment)
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			apiKey = viper.GetString("gemini.api_key")
		}
		model := viper.GetString("gemini.model")
		if model == "" {
			model = "gemini-2.5-flash-preview-05-20"
		}

		// Generate final digest using llmclient with template information
		var maxLengthStr string
		if template.MaxSummaryLength == 0 {
			maxLengthStr = "no limit"
		} else {
			maxLengthStr = fmt.Sprintf("%d characters", template.MaxSummaryLength)
		}

		// Build key features description
		var features []string
		if template.IncludeSummaries {
			features = append(features, "summaries")
		}
		if template.IncludeKeyInsights {
			features = append(features, "key insights")
		}
		if template.IncludeActionItems {
			features = append(features, "action items")
		}
		if template.IncludeSourceLinks {
			features = append(features, "source links")
		}
		keyFeatures := strings.Join(features, ", ")

		finalDigest, err := llmclient.GenerateFinalDigestWithTemplate(
			apiKey,
			model,
			combinedSummaries.String(),
			string(template.Format),
			template.Title, // This is the generic template title, will be replaced by dynamic one later for rendering
			maxLengthStr,
			keyFeatures,
		)

		var generatedTitle string
		var contentForTitleGeneration string

		if err != nil {
			logger.Error("Failed to generate final digest", err)
			fmt.Printf("⚠️  Failed to generate final digest summary, using individual summaries: %s\n", err)
			contentForTitleGeneration = combinedSummaries.String() // Use combined summaries for title if final digest failed

			// Generate Smart Headline even in fallback
			fmt.Printf("🎯 Generating Smart Headline...\n")
			if llmClient != nil {
				title, titleErr := llmClient.GenerateDigestTitle(contentForTitleGeneration, format)
				if titleErr != nil {
					logger.Error("Failed to generate digest title in fallback", titleErr)
					fmt.Printf("   ⚠️ Smart Headline generation failed, using default\n")
					generatedTitle = template.Title // Fallback to template's generic title if specific generation fails
				} else {
					generatedTitle = title
					fmt.Printf("   ✅ Smart Headline: \"%s\"\n", generatedTitle)
					logger.Info("Digest title generated in fallback", "title", generatedTitle)
				}
			} else {
				fmt.Printf("   ⚠️ LLM client not available, using default title\n")
				generatedTitle = template.Title // Fallback if llmClient is nil
			}

			// Pass generatedTitle to the rendering function

			// Prepare insights data for rendering
			var overallSentimentText string
			var alertsSummaryText string
			var trendsSummaryText string

			if len(processedArticles) > 0 {
				// Format overall sentiment
				overallSentimentText = fmt.Sprintf("Overall: %s (%.2f)",
					overallSentiment.OverallEmoji, overallSentiment.OverallScore.Overall)

				// Format alerts summary
				if len(triggeredAlerts) > 0 {
					alertsSummaryText = alertManager.FormatAlertsSection(triggeredAlerts)
				} else {
					alertsSummaryText = "### ✅ Alert Monitoring\nNo alerts triggered for this digest. All articles passed through standard monitoring criteria."
				}
			}

			var renderedContent, digestPath string
			var renderErr error

			if format == "email" {
				// Handle email format specially
				renderedContent, digestPath, renderErr = templates.RenderHTMLEmail(digestItems, outputDir, "", generatedTitle, overallSentimentText, alertsSummaryText, trendsSummaryText, researchSuggestions, "default")
			} else {
				// Handle other formats with standard markdown rendering
				renderedContent, digestPath, renderErr = templates.RenderWithInsights(digestItems, outputDir, "", "", template, generatedTitle, overallSentimentText, alertsSummaryText, trendsSummaryText, researchSuggestions)
			}

			if renderErr != nil {
				return fmt.Errorf("failed to render digest with template: %w", renderErr)
			}

			// Cache the digest with actual content if store is available
			if cacheStore != nil {
				digestID := uuid.NewString()
				var articleURLs []string
				for _, item := range digestItems {
					articleURLs = append(articleURLs, item.URL)
				}
				// Pass generatedTitle to caching function
				if cacheErr := cacheStore.CacheDigestWithFormat(digestID, generatedTitle, renderedContent, "", format, articleURLs, model); cacheErr != nil {
					logger.Error("Failed to cache digest", cacheErr)
				}
			}

			logger.Info("Digest generated successfully with template (no final summary)", "path", digestPath, "format", format, "title", generatedTitle)
			fmt.Printf("✅ %s digest generated: %s\n", format, digestPath)
		} else {
			contentForTitleGeneration = finalDigest // Use final digest for title generation

			// Generate Smart Headline
			fmt.Printf("🎯 Generating Smart Headline...\n")
			if llmClient != nil {
				title, titleErr := llmClient.GenerateDigestTitle(contentForTitleGeneration, format)
				if titleErr != nil {
					logger.Error("Failed to generate digest title", titleErr)
					fmt.Printf("   ⚠️ Smart Headline generation failed, using default\n")
					generatedTitle = template.Title // Fallback to template's generic title
				} else {
					generatedTitle = title
					fmt.Printf("   ✅ Smart Headline: \"%s\"\n", generatedTitle)
					logger.Info("Digest title generated", "title", generatedTitle)
				}
			} else {
				fmt.Printf("   ⚠️ LLM client not available, using default title\n")
				generatedTitle = template.Title // Fallback if llmClient is nil
			}

			// Render digest with template and final summary, passing the generatedTitle

			// Prepare insights data for rendering
			var overallSentimentText string
			var alertsSummaryText string
			var trendsSummaryText string

			if len(processedArticles) > 0 {
				// Format overall sentiment
				overallSentimentText = fmt.Sprintf("Overall: %s (%.2f)",
					overallSentiment.OverallEmoji, overallSentiment.OverallScore.Overall)

				// Format alerts summary
				if len(triggeredAlerts) > 0 {
					alertsSummaryText = alertManager.FormatAlertsSection(triggeredAlerts)
				} else {
					alertsSummaryText = "### ✅ Alert Monitoring\nNo alerts triggered for this digest. All articles passed through standard monitoring criteria."
				}
			}

			var renderedContent, digestPath string
			var renderErr error

			if format == "email" {
				// Handle email format specially
				renderedContent, digestPath, renderErr = templates.RenderHTMLEmail(digestItems, outputDir, finalDigest, generatedTitle, overallSentimentText, alertsSummaryText, trendsSummaryText, researchSuggestions, "default")
			} else {
				// Handle other formats with standard markdown rendering
				renderedContent, digestPath, renderErr = templates.RenderWithInsights(digestItems, outputDir, finalDigest, "", template, generatedTitle, overallSentimentText, alertsSummaryText, trendsSummaryText, researchSuggestions)
			}

			if renderErr != nil {
				return fmt.Errorf("failed to render digest with template: %w", renderErr)
			}

			// Cache the digest with actual content if store is available
			if cacheStore != nil {
				digestID := uuid.NewString()
				var articleURLs []string
				for _, item := range digestItems {
					articleURLs = append(articleURLs, item.URL)
				}
				// Pass generatedTitle to caching function
				if cacheErr := cacheStore.CacheDigestWithFormat(digestID, generatedTitle, renderedContent, finalDigest, format, articleURLs, model); cacheErr != nil {
					logger.Error("Failed to cache digest", cacheErr)
				}
			}

			logger.Info("Digest generated successfully with template and final summary", "path", digestPath, "format", format, "title", generatedTitle)
			fmt.Printf("✅ %s digest with executive summary generated: %s\n", format, digestPath)
		}
	} else {
		logger.Warn("No articles were successfully processed")
		fmt.Println("⚠️  No articles were successfully processed")
	}

	return nil
}

func init() {
	rootCmd.AddCommand(digestCmd)
	rootCmd.AddCommand(summarizeCmd)
	digestCmd.Flags().StringP("output", "o", "digests", "Output directory for digest file")
	digestCmd.Flags().Bool("dry-run", false, "Estimate costs without making API calls")
	digestCmd.Flags().StringP("format", "f", "standard", "Digest format: brief, standard, detailed, newsletter")
}

// Add cache management commands
var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage the article and summary cache",
	Long:  `Inspect, clean, and manage the SQLite cache for articles and summaries.`,
}

var listFormatsCmd = &cobra.Command{
	Use:   "formats",
	Short: "List available digest formats",
	Long:  `Display all available digest formats with their descriptions.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Available Digest Formats:")
		fmt.Println("========================")
		fmt.Println()

		formats := map[string]string{
			"brief":      "Concise digest with key highlights only",
			"standard":   "Balanced digest with summaries and key points",
			"detailed":   "Comprehensive digest with full summaries and analysis",
			"newsletter": "Newsletter-style digest optimized for sharing",
			"email":      "HTML email format with responsive design",
		}

		for format, description := range formats {
			fmt.Printf("• %-12s %s\n", format+":", description)
		}

		fmt.Println()
		fmt.Printf("Usage: briefly digest --format <format> input.md\n")
	},
}

var cacheStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show cache statistics",
	Long:  `Display statistics about the article and summary cache.`,
	Run: func(cmd *cobra.Command, args []string) {
		cacheStore, err := store.NewStore(".briefly-cache")
		if err != nil {
			fmt.Printf("Error opening cache: %s\n", err)
			return
		}
		defer func() {
			if err := cacheStore.Close(); err != nil {
				logger.Error("Failed to close cache store", err)
			}
		}()

		stats, err := cacheStore.GetCacheStats()
		if err != nil {
			fmt.Printf("Error getting cache stats: %s\n", err)
			return
		}

		fmt.Println("Cache Statistics:")
		fmt.Println("================")
		fmt.Printf("Articles: %d\n", stats.ArticleCount)
		fmt.Printf("Summaries: %d\n", stats.SummaryCount)
		fmt.Printf("Digests: %d\n", stats.DigestCount)
		fmt.Printf("Cache size: %.2f MB\n", float64(stats.CacheSize)/(1024*1024))
		fmt.Printf("Last updated: %s\n", stats.LastUpdated.Format("2006-01-02 15:04:05"))

		// RSS Feed Statistics
		fmt.Println("\nRSS Feed Statistics:")
		fmt.Println("===================")
		fmt.Printf("Total feeds: %d\n", stats.FeedCount)
		fmt.Printf("Active feeds: %d\n", stats.ActiveFeedCount)
		fmt.Printf("Feed items: %d\n", stats.FeedItemCount)
		fmt.Printf("Processed items: %d\n", stats.ProcessedItemCount)
		if stats.FeedItemCount > 0 {
			processingRate := float64(stats.ProcessedItemCount) / float64(stats.FeedItemCount) * 100
			fmt.Printf("Processing rate: %.1f%%\n", processingRate)
		}

		// Topic Clustering Statistics
		if len(stats.TopicClusters) > 0 {
			fmt.Println("\nTopic Clusters:")
			fmt.Println("==============")
			totalClustered := 0
			for cluster, count := range stats.TopicClusters {
				fmt.Printf("• %-20s %d items\n", cluster, count)
				totalClustered += count
			}
			fmt.Printf("\nTotal clustered items: %d\n", totalClustered)
		}
	},
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear the cache",
	Long:  `Remove all cached articles, summaries, and digests.`,
	Run: func(cmd *cobra.Command, args []string) {
		confirm, _ := cmd.Flags().GetBool("confirm")
		if !confirm {
			fmt.Println("This will delete all cached data. Use --confirm to proceed.")
			return
		}

		cacheStore, err := store.NewStore(".briefly-cache")
		if err != nil {
			fmt.Printf("Error opening cache: %s\n", err)
			return
		}
		defer func() { _ = cacheStore.Close() }()

		if err := cacheStore.ClearCache(); err != nil {
			fmt.Printf("Error clearing cache: %s\n", err)
			return
		}

		fmt.Println("✅ Cache cleared successfully")
	},
}

var myTakeCmd = &cobra.Command{
	Use:   "my-take",
	Short: "Manage my-take for digests",
	Long:  `Add or edit your personal take on generated digests.`,
}

var addMyTakeCmd = &cobra.Command{
	Use:   "add [digest-id] [my-take-text]",
	Short: "Add your take to a digest",
	Long: `Add your personal take to a digest. If no digest ID is provided, shows recent digests to choose from.
If no my-take text is provided, opens an editor for input.

Example:
  briefly my-take add 12345678-abcd-1234-abcd-123456789abc "This is my personal take on the digest"
  briefly my-take add 12345678-abcd-1234-abcd-123456789abc  # Opens editor
  briefly my-take add  # Shows list of recent digests`,
	Run: func(cmd *cobra.Command, args []string) {
		cacheStore, err := store.NewStore(".briefly-cache")
		if err != nil {
			fmt.Printf("Error opening cache: %s\n", err)
			return
		}
		defer func() { _ = cacheStore.Close() }()

		if len(args) == 0 {
			// Show recent digests
			digests, err := cacheStore.GetLatestDigests(10)
			if err != nil {
				fmt.Printf("Error retrieving digests: %s\n", err)
				return
			}

			if len(digests) == 0 {
				fmt.Println("No digests found. Generate a digest first with 'briefly digest'.")
				return
			}

			fmt.Println("Recent Digests:")
			fmt.Println("===============")
			for i, digest := range digests {
				myTakeStatus := "❌ No take"
				if digest.MyTake != "" {
					myTakeStatus = "✅ Has take"
				}
				fmt.Printf("%d. %s (%s) [%s] - %s\n",
					i+1,
					digest.ID[:8],
					digest.Format,
					digest.DateGenerated.Format("2006-01-02 15:04"),
					myTakeStatus)
			}
			fmt.Println("\nUse: briefly my-take add <digest-id> [\"my take text\"]")
			return
		}

		digestID := args[0]
		digest, err := cacheStore.FindDigestByPartialID(digestID)
		if err != nil {
			fmt.Printf("Error retrieving digest: %s\n", err)
			return
		}

		if digest == nil {
			fmt.Printf("Digest with ID starting with '%s' not found.\n", digestID)
			return
		}

		fmt.Printf("Adding my-take to digest: %s (%s)\n", digest.Title, digest.Format)
		if digest.MyTake != "" {
			fmt.Printf("Current take: %s\n\n", digest.MyTake)
		}

		var myTake string

		if len(args) >= 2 {
			// My-take provided as argument
			myTake = strings.Join(args[1:], " ")
		} else {
			// No my-take provided, use a simple prompt
			fmt.Print("Enter your take: ")

			// Read input from stdin using os.Stdin
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Error reading input: %s\n", err)
				return
			}
			myTake = strings.TrimSpace(input)
		}

		if myTake == "" {
			fmt.Println("No take provided. Cancelled.")
			return
		}

		err = cacheStore.UpdateDigestMyTake(digest.ID, myTake)
		if err != nil {
			fmt.Printf("Error updating my-take: %s\n", err)
			return
		}

		fmt.Println("✅ My-take added successfully!")
		fmt.Printf("Your take: %s\n", myTake)
	},
}

var listMyTakeCmd = &cobra.Command{
	Use:   "list",
	Short: "List digests with my-take",
	Long:  `List all digests and show which ones have your personal take added.`,
	Run: func(cmd *cobra.Command, args []string) {
		cacheStore, err := store.NewStore(".briefly-cache")
		if err != nil {
			fmt.Printf("Error opening cache: %s\n", err)
			return
		}
		defer func() { _ = cacheStore.Close() }()

		digests, err := cacheStore.GetLatestDigests(20)
		if err != nil {
			fmt.Printf("Error retrieving digests: %s\n", err)
			return
		}

		if len(digests) == 0 {
			fmt.Println("No digests found.")
			return
		}

		fmt.Println("All Digests:")
		fmt.Println("============")
		for _, digest := range digests {
			myTakeStatus := "❌"
			if digest.MyTake != "" {
				myTakeStatus = "✅"
			}
			fmt.Printf("%s %s (%s) - %s [%s]\n",
				myTakeStatus,
				digest.ID[:8],
				digest.Format,
				digest.DateGenerated.Format("2006-01-02 15:04"),
				digest.Title)

			if digest.MyTake != "" {
				fmt.Printf("   Take: %s\n", digest.MyTake)
			}
			fmt.Println()
		}
	},
}

var regenerateCmd = &cobra.Command{
	Use:   "regenerate [digest-id]",
	Short: "Regenerate a digest with my-take included",
	Long: `Regenerate a digest file including your personal take. This creates a new markdown file with your take included.

Example:
  briefly my-take regenerate 12345678-abcd-1234-abcd-123456789abc`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		digestID := args[0]

		cacheStore, err := store.NewStore(".briefly-cache")
		if err != nil {
			fmt.Printf("Error opening cache: %s\n", err)
			return
		}
		defer func() { _ = cacheStore.Close() }()

		digest, err := cacheStore.FindDigestByPartialID(digestID)
		if err != nil {
			fmt.Printf("Error retrieving digest: %s\n", err)
			return
		}

		if digest == nil {
			fmt.Printf("Digest with ID starting with '%s' not found.\n", digestID)
			return
		}

		if digest.MyTake == "" {
			fmt.Printf("No my-take found for digest %s. Add one first with 'briefly my-take add %s'\n", digest.ID[:8], digest.ID)
			return
		}

		fmt.Printf("Regenerating digest with your personal voice integrated throughout...\n")

		// Use LLM to regenerate the entire digest with my-take integrated
		regeneratedContent, err := llm.RegenerateDigestWithMyTake(digest.Content, digest.MyTake, digest.Format)
		if err != nil {
			fmt.Printf("Error regenerating digest with LLM: %s\n", err)
			return
		}

		// Create output file with timestamp to avoid overwriting
		outputDir := "digests"
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				logger.Error("Failed to create output directory", err)
				fmt.Printf("❌ Failed to create directory %s: %s\n", outputDir, err)
			}
		}

		timestamp := time.Now().Format("2006-01-02_15-04-05")
		filename := fmt.Sprintf("digest_%s_with_my_take_%s.md", digest.Format, timestamp)
		outputPath := fmt.Sprintf("%s/%s", outputDir, filename)

		err = os.WriteFile(outputPath, []byte(regeneratedContent), 0644)
		if err != nil {
			fmt.Printf("Error writing regenerated digest: %s\n", err)
			return
		}

		fmt.Printf("✅ Digest regenerated with your voice integrated: %s\n", outputPath)

		// Show a preview of the regenerated content
		previewLength := 200
		if len(regeneratedContent) < previewLength {
			previewLength = len(regeneratedContent)
		}
		fmt.Printf("Preview: %s...\n", regeneratedContent[:previewLength])
	},
}

// Feed management commands
var feedCmd = &cobra.Command{
	Use:   "feed",
	Short: "Manage RSS/Atom feeds",
	Long:  `Add, list, and manage RSS/Atom feeds for automatic content discovery.`,
}

var addFeedCmd = &cobra.Command{
	Use:   "add [feed-url]",
	Short: "Add a new RSS/Atom feed",
	Long: `Add a new RSS/Atom feed for monitoring. The feed will be validated before adding.

Example:
  briefly feed add https://example.com/feed.xml
  briefly feed add https://blog.example.com/rss`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		feedURL := args[0]

		cacheStore, err := store.NewStore(".briefly-cache")
		if err != nil {
			fmt.Printf("Error opening cache: %s\n", err)
			return
		}
		defer func() { _ = cacheStore.Close() }()

		feedManager := feeds.NewFeedManager()

		// Validate the feed URL first
		fmt.Printf("Validating feed: %s\n", feedURL)
		if err := feedManager.ValidateFeedURL(feedURL); err != nil {
			fmt.Printf("Error: Invalid feed URL: %s\n", err)
			return
		}

		// Fetch feed to get metadata
		parsedFeed, err := feedManager.FetchFeed(feedURL, "", "")
		if err != nil {
			fmt.Printf("Error fetching feed: %s\n", err)
			return
		}

		if parsedFeed.NotModified {
			fmt.Println("Error: Could not retrieve feed content")
			return
		}

		// Update feed metadata
		parsedFeed.Feed.LastFetched = time.Now().UTC()
		parsedFeed.Feed.LastModified = parsedFeed.LastModified
		parsedFeed.Feed.ETag = parsedFeed.ETag

		// Add feed to database
		err = cacheStore.AddFeed(parsedFeed.Feed)
		if err != nil {
			fmt.Printf("Error adding feed: %s\n", err)
			return
		}

		// Add discovered items
		itemCount := 0
		for _, item := range parsedFeed.Items {
			if err := cacheStore.AddFeedItem(item); err == nil {
				itemCount++
			}
		}

		fmt.Printf("✅ Feed added successfully!\n")
		fmt.Printf("Feed: %s\n", parsedFeed.Feed.Title)
		fmt.Printf("Description: %s\n", parsedFeed.Feed.Description)
		fmt.Printf("Items discovered: %d\n", itemCount)
	},
}

var listFeedsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured feeds",
	Long:  `Display all RSS/Atom feeds with their status and recent activity.`,
	Run: func(cmd *cobra.Command, args []string) {
		cacheStore, err := store.NewStore(".briefly-cache")
		if err != nil {
			fmt.Printf("Error opening cache: %s\n", err)
			return
		}
		defer func() { _ = cacheStore.Close() }()

		activeOnly, _ := cmd.Flags().GetBool("active-only")
		feeds, err := cacheStore.GetFeeds(activeOnly)
		if err != nil {
			fmt.Printf("Error retrieving feeds: %s\n", err)
			return
		}

		if len(feeds) == 0 {
			if activeOnly {
				fmt.Println("No active feeds found. Add feeds with 'briefly feed add <url>'")
			} else {
				fmt.Println("No feeds found. Add feeds with 'briefly feed add <url>'")
			}
			return
		}

		fmt.Println("RSS/Atom Feeds:")
		fmt.Println("===============")
		for _, feed := range feeds {
			status := "✅ Active"
			if !feed.Active {
				status = "❌ Inactive"
			}

			errorInfo := ""
			if feed.ErrorCount > 0 {
				errorInfo = fmt.Sprintf(" (⚠️  %d errors)", feed.ErrorCount)
			}

			fmt.Printf("%s %s%s\n", status, feed.Title, errorInfo)
			fmt.Printf("   URL: %s\n", feed.URL)
			if feed.Description != "" {
				fmt.Printf("   Description: %s\n", feed.Description)
			}
			if !feed.LastFetched.IsZero() {
				fmt.Printf("   Last fetched: %s\n", feed.LastFetched.Format("2006-01-02 15:04"))
			}
			fmt.Printf("   ID: %s\n", feed.ID[:8])
			fmt.Println()
		}
	},
}

var pullFeedsCmd = &cobra.Command{
	Use:   "pull",
	Short: "Pull latest items from all active feeds",
	Long:  `Fetch the latest items from all active RSS/Atom feeds and add new items to the processing queue.`,
	Run: func(cmd *cobra.Command, args []string) {
		cacheStore, err := store.NewStore(".briefly-cache")
		if err != nil {
			fmt.Printf("Error opening cache: %s\n", err)
			return
		}
		defer func() { _ = cacheStore.Close() }()

		feedManager := feeds.NewFeedManager()

		// Get all active feeds
		activeFeeds, err := cacheStore.GetFeeds(true)
		if err != nil {
			fmt.Printf("Error retrieving feeds: %s\n", err)
			return
		}

		if len(activeFeeds) == 0 {
			fmt.Println("No active feeds configured. Add feeds with 'briefly feed add <url>'")
			return
		}

		fmt.Printf("Pulling from %d active feeds...\n", len(activeFeeds))

		totalNewItems := 0
		successCount := 0

		for _, feed := range activeFeeds {
			fmt.Printf("Fetching: %s\n", feed.Title)

			parsedFeed, err := feedManager.FetchFeed(feed.URL, feed.LastModified, feed.ETag)
			if err != nil {
				fmt.Printf("  ❌ Error: %s\n", err)
				if updateErr := cacheStore.UpdateFeedError(feed.ID, err.Error()); updateErr != nil {
					logger.Error("Failed to update feed error", updateErr)
				}
				continue
			}

			if parsedFeed.NotModified {
				fmt.Printf("  ✅ No new content\n")
				successCount++
				continue
			}

			// Update feed metadata
			err = cacheStore.UpdateFeed(feed.ID, parsedFeed.Feed.Title, parsedFeed.Feed.Description,
				parsedFeed.LastModified, parsedFeed.ETag, time.Now().UTC())
			if err != nil {
				fmt.Printf("  ⚠️  Warning: Could not update feed metadata: %s\n", err)
			}

			// Add new items
			newItems := 0
			for _, item := range parsedFeed.Items {
				if err := cacheStore.AddFeedItem(item); err == nil {
					newItems++
				}
			}

			fmt.Printf("  ✅ Found %d new items\n", newItems)
			totalNewItems += newItems
			successCount++
		}

		fmt.Printf("\n📊 Summary: %d feeds processed, %d new items discovered\n", successCount, totalNewItems)

		if totalNewItems > 0 {
			fmt.Println("Use 'briefly feed items' to see unprocessed items")
		}
	},
}

var manageFeedCmd = &cobra.Command{
	Use:   "manage [feed-id]",
	Short: "Enable, disable, or remove a feed",
	Long: `Manage individual feeds by enabling, disabling, or removing them.

Example:
  briefly feed manage 12345678  # Interactive management
  briefly feed manage 12345678 --disable
  briefly feed manage 12345678 --remove`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		feedID := args[0]
		disable, _ := cmd.Flags().GetBool("disable")
		enable, _ := cmd.Flags().GetBool("enable")
		remove, _ := cmd.Flags().GetBool("remove")

		cacheStore, err := store.NewStore(".briefly-cache")
		if err != nil {
			fmt.Printf("Error opening cache: %s\n", err)
			return
		}
		defer func() { _ = cacheStore.Close() }()

		// Get all feeds to find matching ID
		allFeeds, err := cacheStore.GetFeeds(false)
		if err != nil {
			fmt.Printf("Error retrieving feeds: %s\n", err)
			return
		}

		var targetFeed *core.Feed
		for _, feed := range allFeeds {
			if strings.HasPrefix(feed.ID, feedID) {
				targetFeed = &feed
				break
			}
		}

		if targetFeed == nil {
			fmt.Printf("Feed with ID %s not found\n", feedID)
			return
		}

		if remove {
			fmt.Printf("Are you sure you want to remove feed '%s'? (y/N): ", targetFeed.Title)
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(input)) != "y" {
				fmt.Println("Cancelled.")
				return
			}

			err = cacheStore.DeleteFeed(targetFeed.ID)
			if err != nil {
				fmt.Printf("Error removing feed: %s\n", err)
				return
			}
			fmt.Printf("✅ Feed '%s' removed\n", targetFeed.Title)
			return
		}

		if disable {
			err = cacheStore.SetFeedActive(targetFeed.ID, false)
			if err != nil {
				fmt.Printf("Error disabling feed: %s\n", err)
				return
			}
			fmt.Printf("✅ Feed '%s' disabled\n", targetFeed.Title)
			return
		}

		if enable {
			err = cacheStore.SetFeedActive(targetFeed.ID, true)
			if err != nil {
				fmt.Printf("Error enabling feed: %s\n", err)
				return
			}
			fmt.Printf("✅ Feed '%s' enabled\n", targetFeed.Title)
			return
		}

		// Interactive mode
		fmt.Printf("Feed: %s\n", targetFeed.Title)
		fmt.Printf("URL: %s\n", targetFeed.URL)
		fmt.Printf("Status: %s\n", map[bool]string{true: "Active", false: "Inactive"}[targetFeed.Active])
		fmt.Printf("Error count: %d\n", targetFeed.ErrorCount)
		if targetFeed.LastError != "" {
			fmt.Printf("Last error: %s\n", targetFeed.LastError)
		}
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  briefly feed manage", feedID, "--enable    # Enable feed")
		fmt.Println("  briefly feed manage", feedID, "--disable   # Disable feed")
		fmt.Println("  briefly feed manage", feedID, "--remove    # Remove feed")
	},
}

var feedItemsCmd = &cobra.Command{
	Use:   "items",
	Short: "List unprocessed feed items",
	Long:  `Display feed items that have been discovered but not yet processed into articles.`,
	Run: func(cmd *cobra.Command, args []string) {
		cacheStore, err := store.NewStore(".briefly-cache")
		if err != nil {
			fmt.Printf("Error opening cache: %s\n", err)
			return
		}
		defer func() { _ = cacheStore.Close() }()

		limit, _ := cmd.Flags().GetInt("limit")
		items, err := cacheStore.GetUnprocessedFeedItems(limit)
		if err != nil {
			fmt.Printf("Error retrieving feed items: %s\n", err)
			return
		}

		if len(items) == 0 {
			fmt.Println("No unprocessed feed items found.")
			fmt.Println("Pull feeds with 'briefly feed pull' to discover new content.")
			return
		}

		fmt.Printf("Unprocessed Feed Items (%d):\n", len(items))
		fmt.Println("============================")
		for i, item := range items {
			fmt.Printf("%d. %s\n", i+1, item.Title)
			fmt.Printf("   Link: %s\n", item.Link)
			if !item.Published.IsZero() {
				fmt.Printf("   Published: %s\n", item.Published.Format("2006-01-02 15:04"))
			}
			fmt.Printf("   Discovered: %s\n", item.DateDiscovered.Format("2006-01-02 15:04"))
			fmt.Println()
		}

		fmt.Printf("Use these links in your digest input files or mark them as processed.\n")
	},
}

var summarizeCmd = &cobra.Command{
	Use:   "summarize [url]",
	Short: "Quickly summarize an internet article",
	Long: `Fetch and summarize a single article from a URL. Outputs an executive summary
and key highlights directly to the command line.

Example:
  briefly summarize https://example.com/article
  briefly summarize https://blog.example.org/interesting-post`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]

		if err := runSummarize(url); err != nil {
			logger.Error("Failed to summarize article", err)
			os.Exit(1)
		}
	},
}

func runSummarize(url string) error {
	// Create a link object from the URL
	link := core.Link{
		ID:        "", // Will be set by uuid
		URL:       url,
		DateAdded: time.Now().UTC(),
		Source:    "command-line",
	}

	fmt.Printf("🔍 Fetching article from: %s\n", url)

	// Fetch the article
	article, err := fetch.FetchArticle(link)
	if err != nil {
		return fmt.Errorf("failed to fetch article: %w", err)
	}

	// Parse and clean the content
	if err := fetch.CleanArticleHTML(&article); err != nil {
		return fmt.Errorf("failed to parse article content: %w", err)
	}

	if strings.TrimSpace(article.CleanedText) == "" {
		return fmt.Errorf("no readable content found in the article")
	}

	fmt.Printf("📄 Processing article: %s\n", article.Title)
	fmt.Printf("📊 Content length: %d characters\n\n", len(article.CleanedText))

	// Create LLM client
	llmClient, err := llm.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}
	defer llmClient.Close()

	// Generate summary with custom prompt for quick summarization
	summary, err := generateQuickSummary(llmClient, article)
	if err != nil {
		return fmt.Errorf("failed to generate summary: %w", err)
	}

	// Display the results with improved formatting
	fmt.Println()
	fmt.Printf("📰 %s\n", article.Title)
	fmt.Printf("🔗 %s\n", url)
	fmt.Printf("📅 Summarized on %s\n", time.Now().Format("January 2, 2006"))
	fmt.Printf("📊 %d characters | ⏱️  %s model\n", len(article.CleanedText), summary.ModelUsed)
	fmt.Println()
	fmt.Println(strings.Repeat("─", 80))
	fmt.Println()
	fmt.Print(summary.SummaryText)
	fmt.Println()
	fmt.Println(strings.Repeat("─", 80))
	fmt.Printf("✨ Summary generated in %.1f seconds\n", time.Since(time.Now()).Seconds())

	return nil
}

func generateQuickSummary(llmClient *llm.Client, article core.Article) (core.Summary, error) {
	// Use the new method that generates summaries with key moments
	summary, err := llmClient.SummarizeArticleWithKeyMoments(article)
	if err != nil {
		return core.Summary{}, fmt.Errorf("failed to generate summary with key moments: %w", err)
	}

	return summary, nil
}

func init() {
	// Add cache commands
	rootCmd.AddCommand(cacheCmd)
	rootCmd.AddCommand(listFormatsCmd)
	rootCmd.AddCommand(myTakeCmd)
	rootCmd.AddCommand(feedCmd)

	// Add v0.4 Insights commands
	rootCmd.AddCommand(insightsCmd)
	rootCmd.AddCommand(trendsCmd)
	rootCmd.AddCommand(sentimentCmd)
	rootCmd.AddCommand(researchCmd)

	// Add v1.0 Multi-Channel commands
	rootCmd.AddCommand(sendSlackCmd)
	rootCmd.AddCommand(sendDiscordCmd)
	rootCmd.AddCommand(generateTTSCmd)

	// Add deep-research command
	rootCmd.AddCommand(deepResearchCmd)

	// Insights subcommands
	insightsCmd.AddCommand(alertsCmd)
	alertsCmd.AddCommand(listAlertsCmd)
	alertsCmd.AddCommand(testAlertsCmd)

	cacheCmd.AddCommand(cacheStatsCmd)
	cacheCmd.AddCommand(cacheClearCmd)

	myTakeCmd.AddCommand(addMyTakeCmd)
	myTakeCmd.AddCommand(listMyTakeCmd)
	myTakeCmd.AddCommand(regenerateCmd)

	feedCmd.AddCommand(addFeedCmd)
	feedCmd.AddCommand(listFeedsCmd)
	feedCmd.AddCommand(pullFeedsCmd)
	feedCmd.AddCommand(manageFeedCmd)
	feedCmd.AddCommand(feedItemsCmd)

	// Feed command flags
	listFeedsCmd.Flags().Bool("active-only", false, "Show only active feeds")
	manageFeedCmd.Flags().Bool("disable", false, "Disable the feed")
	manageFeedCmd.Flags().Bool("enable", false, "Enable the feed")
	manageFeedCmd.Flags().Bool("remove", false, "Remove the feed")
	feedItemsCmd.Flags().Int("limit", 50, "Maximum number of items to display")

	// Research command flags
	researchCmd.Flags().Int("depth", 2, "Number of research iterations")
	researchCmd.Flags().Int("max-results", 10, "Maximum results per search query")
	researchCmd.Flags().String("output", "", "Output file for discovered links")
	researchCmd.Flags().String("provider", "auto", "Search provider: auto, google, serpapi, mock")

	// Messaging command flags
	sendSlackCmd.Flags().String("webhook", "", "Slack webhook URL (or set SLACK_WEBHOOK_URL)")
	sendSlackCmd.Flags().String("message-format", "bullets", "Message format: bullets, summary, highlights")
	sendSlackCmd.Flags().Bool("include-sentiment", true, "Include sentiment emojis")

	sendDiscordCmd.Flags().String("webhook", "", "Discord webhook URL (or set DISCORD_WEBHOOK_URL)")
	sendDiscordCmd.Flags().String("message-format", "bullets", "Message format: bullets, summary, highlights")
	sendDiscordCmd.Flags().Bool("include-sentiment", true, "Include sentiment emojis")

	// TTS command flags
	generateTTSCmd.Flags().String("provider", "mock", "TTS provider: openai, elevenlabs, mock")
	generateTTSCmd.Flags().String("voice", "", "Voice ID (provider-specific)")
	generateTTSCmd.Flags().String("api-key", "", "API key for TTS provider (or set OPENAI_API_KEY/ELEVENLABS_API_KEY)")
	generateTTSCmd.Flags().Float64("speed", 1.0, "Speech speed (0.5-2.0)")
	generateTTSCmd.Flags().Int("max-articles", 10, "Maximum articles to include in audio (0 for all)")
	generateTTSCmd.Flags().Bool("include-summaries", true, "Include article summaries in audio")
	generateTTSCmd.Flags().String("output", "audio", "Output directory for audio files")

	// Deep research command flags
	deepResearchCmd.Flags().String("since", "21d", "Time filter for search results (e.g., 1d, 7d, 30d)")
	deepResearchCmd.Flags().Int("max-sources", 25, "Maximum number of sources to include")
	deepResearchCmd.Flags().Bool("html", false, "Generate HTML output in addition to Markdown")
	deepResearchCmd.Flags().String("model", "gemini-2.5-flash-preview-05-20", "LLM model to use for synthesis")
	deepResearchCmd.Flags().Bool("refresh", false, "Force refresh cache, bypass existing content")
	deepResearchCmd.Flags().Bool("javascript", false, "Use headless browser for JavaScript-heavy pages")
	deepResearchCmd.Flags().String("search-provider", "duckduckgo", "Search provider: duckduckgo, serpapi, google, mock")

	cacheClearCmd.Flags().Bool("confirm", false, "Confirm cache deletion")
}

// Insights commands for v0.4
var insightsCmd = &cobra.Command{
	Use:   "insights",
	Short: "Access insights features (trends, alerts, sentiment)",
	Long:  `Access advanced insights features including trend analysis, alert monitoring, and sentiment analysis.`,
}

var trendsCmd = &cobra.Command{
	Use:   "trends [period]",
	Short: "Generate trend analysis reports",
	Long: `Analyze trends in your content over time. Available periods: weekly, monthly.

Examples:
  briefly trends weekly   # Generate weekly trend report
  briefly trends monthly  # Generate monthly trend report`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		period := args[0]
		if period != "weekly" && period != "monthly" {
			fmt.Printf("Error: period must be 'weekly' or 'monthly'\n")
			return
		}

		if err := runTrends(period); err != nil {
			logger.Error("Failed to generate trends report", err)
			os.Exit(1)
		}
	},
}

var alertsCmd = &cobra.Command{
	Use:   "alerts",
	Short: "Manage alert conditions and triggers",
	Long:  `Configure and manage alert conditions that trigger notifications during digest processing.`,
}

var listAlertsCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured alert conditions",
	Long:  `Display all configured alert conditions with their status and configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runListAlerts(); err != nil {
			logger.Error("Failed to list alerts", err)
			os.Exit(1)
		}
	},
}

var testAlertsCmd = &cobra.Command{
	Use:   "test [input-file]",
	Short: "Test alert conditions against content",
	Long: `Test configured alert conditions against articles from an input file.

Example:
  briefly alerts test input/2025-06-03.md`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]
		if err := runTestAlerts(inputFile); err != nil {
			logger.Error("Failed to test alerts", err)
			os.Exit(1)
		}
	},
}

var sentimentCmd = &cobra.Command{
	Use:   "sentiment [input-file]",
	Short: "Analyze sentiment of articles",
	Long: `Perform sentiment analysis on articles and display results with emoji indicators.

Example:
  briefly sentiment input/2025-06-03.md`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]
		if err := runSentiment(inputFile); err != nil {
			logger.Error("Failed to analyze sentiment", err)
			os.Exit(1)
		}
	},
}

var researchCmd = &cobra.Command{
	Use:   "research [topic]",
	Short: "Perform deep research on a topic",
	Long: `Use LLM-driven research to discover relevant articles and content for a topic.

Examples:
  briefly research "artificial intelligence trends" --depth 2
  briefly research "sustainable energy" --max-results 10 --provider google
  briefly research "cloud computing" --provider serpapi`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		topic := args[0]
		depth, _ := cmd.Flags().GetInt("depth")
		maxResults, _ := cmd.Flags().GetInt("max-results")
		outputFile, _ := cmd.Flags().GetString("output")
		provider, _ := cmd.Flags().GetString("provider")

		if err := runResearch(topic, depth, maxResults, outputFile, provider); err != nil {
			logger.Error("Failed to perform research", err)
			os.Exit(1)
		}
	},
}

// Messaging commands for v1.0
var sendSlackCmd = &cobra.Command{
	Use:   "send-slack [input-file]",
	Short: "Send digest to Slack webhook",
	Long: `Send a formatted digest to Slack using webhook integration.

Examples:
  briefly send-slack input/2025-06-03.md --webhook https://hooks.slack.com/...
  briefly send-slack input/2025-06-03.md --message-format highlights
  SLACK_WEBHOOK_URL=https://hooks.slack.com/... briefly send-slack input/2025-06-03.md`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]
		webhookURL, _ := cmd.Flags().GetString("webhook")
		messageFormat, _ := cmd.Flags().GetString("message-format")
		includeSentiment, _ := cmd.Flags().GetBool("include-sentiment")

		if err := runSendMessage(messaging.PlatformSlack, inputFile, webhookURL, messageFormat, includeSentiment); err != nil {
			logger.Error("Failed to send Slack message", err)
			os.Exit(1)
		}
	},
}

var sendDiscordCmd = &cobra.Command{
	Use:   "send-discord [input-file]",
	Short: "Send digest to Discord webhook",
	Long: `Send a formatted digest to Discord using webhook integration.

Examples:
  briefly send-discord input/2025-06-03.md --webhook https://discord.com/api/webhooks/...
  briefly send-discord input/2025-06-03.md --message-format summary
  DISCORD_WEBHOOK_URL=https://discord.com/api/webhooks/... briefly send-discord input/2025-06-03.md`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]
		webhookURL, _ := cmd.Flags().GetString("webhook")
		messageFormat, _ := cmd.Flags().GetString("message-format")
		includeSentiment, _ := cmd.Flags().GetBool("include-sentiment")

		if err := runSendMessage(messaging.PlatformDiscord, inputFile, webhookURL, messageFormat, includeSentiment); err != nil {
			logger.Error("Failed to send Discord message", err)
			os.Exit(1)
		}
	},
}

var generateTTSCmd = &cobra.Command{
	Use:   "generate-tts [input-file]",
	Short: "Generate TTS audio from digest",
	Long: `Generate Text-to-Speech audio file from digest content.

Examples:
  briefly generate-tts input/2025-06-03.md --provider openai --voice alloy
  briefly generate-tts input/2025-06-03.md --provider elevenlabs --voice Rachel
  OPENAI_API_KEY=sk-... briefly generate-tts input/2025-06-03.md`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]
		provider, _ := cmd.Flags().GetString("provider")
		voiceID, _ := cmd.Flags().GetString("voice")
		apiKey, _ := cmd.Flags().GetString("api-key")
		speed, _ := cmd.Flags().GetFloat64("speed")
		maxArticles, _ := cmd.Flags().GetInt("max-articles")
		includeSummaries, _ := cmd.Flags().GetBool("include-summaries")
		outputDir, _ := cmd.Flags().GetString("output")

		if err := runGenerateTTS(inputFile, provider, voiceID, apiKey, speed, maxArticles, includeSummaries, outputDir); err != nil {
			logger.Error("Failed to generate TTS audio", err)
			os.Exit(1)
		}
	},
}

// Deep Research command
var deepResearchCmd = &cobra.Command{
	Use:   "deep-research [topic]",
	Short: "Perform comprehensive deep research on a topic",
	Long: `Generate a comprehensive research brief by automatically decomposing a topic into sub-questions,
searching for relevant sources across the web, and synthesizing findings into a well-cited research brief.

Examples:
  briefly deep-research "open-source agent frameworks"
  briefly deep-research "sustainable energy trends" --since 7d --max-sources 30
  briefly deep-research "AI safety research" --html --model gemini-2.5-flash-preview-05-20
  briefly deep-research "cryptocurrency regulation" --search-provider google --refresh
  briefly deep-research "machine learning papers" --search-provider serpapi`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		topic := args[0]
		since, _ := cmd.Flags().GetString("since")
		maxSources, _ := cmd.Flags().GetInt("max-sources")
		outputHTML, _ := cmd.Flags().GetBool("html")
		model, _ := cmd.Flags().GetString("model")
		refresh, _ := cmd.Flags().GetBool("refresh")
		useJS, _ := cmd.Flags().GetBool("javascript")
		searchProvider, _ := cmd.Flags().GetString("search-provider")

		if err := runDeepResearch(topic, since, maxSources, outputHTML, model, refresh, useJS, searchProvider); err != nil {
			logger.Error("Failed to perform deep research", err)
			os.Exit(1)
		}
	},
}

// Implementation functions for the new commands

func runDeepResearch(topic, since string, maxSources int, outputHTML bool, model string, refresh, useJS bool, searchProvider string) error {
	fmt.Printf("🔍 Starting deep research on: %s\n", topic)
	fmt.Printf("📊 Max sources: %d, Time filter: %s, Search provider: %s\n", maxSources, since, searchProvider)
	if useJS {
		fmt.Printf("🌐 JavaScript execution enabled for content fetching\n")
	}
	fmt.Println()

	// Parse time filter
	sinceTime, err := parseDuration(since)
	if err != nil {
		return fmt.Errorf("invalid time filter '%s': %w", since, err)
	}

	// Initialize LLM client
	llmClient, err := llm.NewClient(model)
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	defer llmClient.Close()

	// Initialize cache store
	cacheStore, err := store.NewStore(".briefly-cache")
	if err != nil {
		fmt.Printf("⚠️  Cache disabled due to error: %s\n", err)
		cacheStore = nil
	} else {
		defer func() { _ = cacheStore.Close() }()
		fmt.Printf("📦 Cache store initialized\n")
	}

	// Initialize search provider using shared search module
	factory := search.NewProviderFactory()
	var searcher search.Provider

	switch searchProvider {
	case "duckduckgo":
		searcher, err = factory.CreateProvider(search.ProviderTypeDuckDuckGo, nil)
		fmt.Printf("🔌 Using DuckDuckGo search\n")
	case "serpapi":
		apiKey := os.Getenv("SERPAPI_KEY")
		if apiKey == "" {
			return fmt.Errorf("SERPAPI_KEY environment variable required for SerpAPI")
		}
		searcher, err = factory.CreateProvider(search.ProviderTypeSerpAPI, map[string]string{"api_key": apiKey})
		fmt.Printf("🔌 Using SerpAPI search\n")
	case "google":
		apiKey := os.Getenv("GOOGLE_CSE_API_KEY")
		searchID := os.Getenv("GOOGLE_CSE_ID")
		if apiKey == "" || searchID == "" {
			return fmt.Errorf("GOOGLE_CSE_API_KEY and GOOGLE_CSE_ID environment variables required for Google Custom Search")
		}
		searcher, err = factory.CreateProvider(search.ProviderTypeGoogle, map[string]string{
			"api_key":   apiKey,
			"search_id": searchID,
		})
		fmt.Printf("🔌 Using Google Custom Search\n")
	case "mock":
		searcher, err = factory.CreateProvider(search.ProviderTypeMock, nil)
		fmt.Printf("🔌 Using Mock search (testing)\n")
	default:
		return fmt.Errorf("unsupported search provider: %s", searchProvider)
	}

	if err != nil {
		return fmt.Errorf("failed to create search provider: %w", err)
	}

	// Initialize components
	planner := deepresearch.NewLLMPlanner(llmClient)
	fetcher := deepresearch.NewResearchContentFetcher(deepresearch.NewFetcher(), cacheStore)
	ranker := deepresearch.NewEmbeddingRanker()
	synthesizer := deepresearch.NewLLMSynthesizer(llmClient)

	// Create research engine
	engine := deepresearch.NewResearchEngine(planner, searcher, fetcher, ranker, synthesizer)

	// Configure research
	config := deepresearch.ResearchConfig{
		MaxSources:     maxSources,
		SinceTime:      sinceTime,
		Model:          model,
		SearchProvider: searchProvider,
		UseJavaScript:  useJS,
		RefreshCache:   refresh,
		OutputHTML:     outputHTML,
	}

	// Perform research
	ctx := context.Background()
	fmt.Printf("🚀 Starting research process...\n")
	brief, err := engine.Research(ctx, topic, config)
	if err != nil {
		return fmt.Errorf("research failed: %w", err)
	}

	// Generate outputs
	fmt.Printf("\n📝 Generating outputs...\n")

	// Create research directory if it doesn't exist
	if err := os.MkdirAll("research", 0755); err != nil {
		return fmt.Errorf("failed to create research directory: %w", err)
	}

	// Generate slug for filenames
	slug := generateSlug(topic)

	// Output Markdown
	markdownPath := fmt.Sprintf("research/%s.md", slug)
	markdownContent := formatBriefAsMarkdown(brief)
	if err := os.WriteFile(markdownPath, []byte(markdownContent), 0644); err != nil {
		return fmt.Errorf("failed to write markdown file: %w", err)
	}
	fmt.Printf("📄 Markdown brief: %s\n", markdownPath)

	// Output JSON artifact
	jsonPath := fmt.Sprintf("research/%s.json", slug)
	jsonData, err := json.MarshalIndent(brief, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write JSON file: %w", err)
	}
	fmt.Printf("📊 Raw data: %s\n", jsonPath)

	// Output HTML if requested
	if outputHTML {
		htmlPath := fmt.Sprintf("research/%s.html", slug)
		htmlContent := formatBriefAsHTML(brief)
		if err := os.WriteFile(htmlPath, []byte(htmlContent), 0644); err != nil {
			return fmt.Errorf("failed to write HTML file: %w", err)
		}
		fmt.Printf("🌐 HTML brief: %s\n", htmlPath)
	}

	// Display results summary
	fmt.Printf("\n✅ Research completed successfully!\n")
	fmt.Printf("📖 Topic: %s\n", brief.Topic)
	fmt.Printf("📝 Sub-queries generated: %d\n", len(brief.SubQueries))
	fmt.Printf("📚 Sources found: %d\n", len(brief.Sources))
	fmt.Printf("📊 Detailed findings: %d sections\n", len(brief.DetailedFindings))
	fmt.Printf("❓ Open questions: %d\n", len(brief.OpenQuestions))
	fmt.Printf("⏱️  Generated at: %s\n", brief.GeneratedAt.Format("2006-01-02 15:04:05"))

	// Output to stdout for immediate viewing
	fmt.Printf("\n" + strings.Repeat("=", 80) + "\n")
	fmt.Print(markdownContent)
	fmt.Printf(strings.Repeat("=", 80) + "\n")

	// Suggest next steps
	fmt.Printf("\n💡 Next steps:\n")
	fmt.Printf("   • Review the research brief above\n")
	fmt.Printf("   • Use 'briefly chat %s' to ask follow-up questions\n", slug)
	fmt.Printf("   • Add to digest: briefly digest research/%s.md\n", slug)

	return nil
}

func runTrends(period string) error {
	cacheStore, err := store.NewStore(".briefly-cache")
	if err != nil {
		return fmt.Errorf("failed to open cache: %w", err)
	}
	defer func() { _ = cacheStore.Close() }()

	analyzer := trends.NewTrendAnalyzer()

	// Get recent articles for current period
	currentArticles, err := cacheStore.GetRecentArticles(30) // Get articles from last 30 days
	if err != nil {
		return fmt.Errorf("failed to get current articles: %w", err)
	}

	// Get articles for previous period
	var previousArticles []core.Article
	switch period {
	case "weekly":
		previousArticles, err = cacheStore.GetArticlesByDateRange(
			time.Now().AddDate(0, 0, -14), // Two weeks ago
			time.Now().AddDate(0, 0, -7),  // One week ago
		)
	case "monthly":
		previousArticles, err = cacheStore.GetArticlesByDateRange(
			time.Now().AddDate(0, -2, 0), // Two months ago
			time.Now().AddDate(0, -1, 0), // One month ago
		)
	}
	if err != nil {
		return fmt.Errorf("failed to get previous articles: %w", err)
	}

	// Generate trend report using AnalyzeArticleTrends which works with articles
	report, err := analyzer.AnalyzeArticleTrends(currentArticles, previousArticles)
	if err != nil {
		return fmt.Errorf("failed to analyze trends: %w", err)
	}

	// Display the report
	fmt.Println(analyzer.FormatReport(report))

	return nil
}

func runListAlerts() error {
	alertManager := alerts.NewAlertManager()

	// Get default conditions (in real implementation, would load from config/database)
	conditions := alertManager.GetDefaultConditions()

	fmt.Println("Configured Alert Conditions:")
	fmt.Println("============================")
	fmt.Println()

	for _, condition := range conditions {
		status := "✅ Enabled"
		if !condition.Enabled {
			status = "❌ Disabled"
		}

		var levelIcon string
		switch condition.Level {
		case alerts.AlertLevelCritical:
			levelIcon = "🚨"
		case alerts.AlertLevelWarning:
			levelIcon = "⚠️"
		default:
			levelIcon = "ℹ️"
		}

		fmt.Printf("%s %s %s\n", status, levelIcon, condition.Name)
		fmt.Printf("   Type: %s\n", condition.Type)
		fmt.Printf("   Description: %s\n", condition.Description)

		// Show relevant config
		if keywords, ok := condition.Config["keywords"].([]interface{}); ok {
			keywordStrs := make([]string, len(keywords))
			for i, k := range keywords {
				keywordStrs[i] = k.(string)
			}
			fmt.Printf("   Keywords: %s\n", strings.Join(keywordStrs, ", "))
		}
		if threshold, ok := condition.Config["threshold"].(float64); ok {
			fmt.Printf("   Threshold: %.1f%%\n", threshold)
		}

		fmt.Printf("   ID: %s\n", condition.ID[:8])
		fmt.Println()
	}

	return nil
}

func runTestAlerts(inputFile string) error {
	// Read links from input file
	links, err := readLinksFromFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read links: %w", err)
	}

	cacheStore, err := store.NewStore(".briefly-cache")
	if err != nil {
		return fmt.Errorf("failed to open cache: %w", err)
	}
	defer func() { _ = cacheStore.Close() }()

	// Fetch articles
	var articles []core.Article
	for _, link := range links {
		article, err := cacheStore.GetCachedArticle(link.URL, 24*time.Hour)
		if err != nil {
			// If not in cache, would need to fetch - for now skip
			fmt.Printf("⚠️  Article not in cache: %s\n", link.URL)
			continue
		}
		articles = append(articles, *article)
	}

	alertManager := alerts.NewAlertManager()

	// Evaluate alerts against articles
	triggeredAlerts, err := alertManager.EvaluateAlerts(articles)
	if err != nil {
		return fmt.Errorf("failed to evaluate alerts: %w", err)
	}

	// Display results
	fmt.Printf("Alert Test Results for %s\n", inputFile)
	fmt.Println("================================")
	fmt.Printf("Articles analyzed: %d\n", len(articles))
	fmt.Printf("Alerts triggered: %d\n", len(triggeredAlerts))
	fmt.Println()

	if len(triggeredAlerts) == 0 {
		fmt.Println("✅ No alerts triggered")
		return nil
	}

	for _, alert := range triggeredAlerts {
		var levelIcon string
		switch alert.Level {
		case alerts.AlertLevelCritical:
			levelIcon = "🚨"
		case alerts.AlertLevelWarning:
			levelIcon = "⚠️"
		default:
			levelIcon = "ℹ️"
		}

		fmt.Printf("%s %s\n", levelIcon, alert.Title)
		fmt.Printf("   %s\n", alert.Message)
		if alert.Context != nil {
			if url, ok := alert.Context["url"]; ok {
				fmt.Printf("   Article: %s\n", url)
			}
		}
		fmt.Printf("   Triggered: %s\n", alert.TriggeredAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	return nil
}

func runSentiment(inputFile string) error {
	// Read links from input file
	links, err := readLinksFromFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read links: %w", err)
	}

	cacheStore, err := store.NewStore(".briefly-cache")
	if err != nil {
		return fmt.Errorf("failed to open cache: %w", err)
	}
	defer func() { _ = cacheStore.Close() }()

	// Fetch articles
	var articles []core.Article
	for _, link := range links {
		article, err := cacheStore.GetCachedArticle(link.URL, 24*time.Hour)
		if err != nil {
			fmt.Printf("⚠️  Article not in cache: %s\n", link.URL)
			continue
		}
		articles = append(articles, *article)
	}

	analyzer := sentiment.NewSentimentAnalyzer()

	fmt.Printf("Sentiment Analysis for %s\n", inputFile)
	fmt.Println("================================")
	fmt.Printf("Articles analyzed: %d\n", len(articles))
	fmt.Println()

	var totalSentiment float64
	sentimentCounts := make(map[sentiment.SentimentClassification]int)

	for _, article := range articles {
		articleSentiment, err := analyzer.AnalyzeArticle(article)
		if err != nil {
			fmt.Printf("⚠️  Failed to analyze sentiment for: %s\n", article.Title)
			continue
		}

		fmt.Printf("%s **%s**\n", articleSentiment.Emoji, article.Title)
		fmt.Printf("   Sentiment: %s (%.2f)\n", articleSentiment.Classification, articleSentiment.Score.Overall)
		fmt.Printf("   Confidence: %.1f%%\n", articleSentiment.Score.Confidence*100)
		if len(articleSentiment.KeyPhrases) > 0 {
			fmt.Printf("   Key phrases: %s\n", strings.Join(articleSentiment.KeyPhrases[:min(3, len(articleSentiment.KeyPhrases))], ", "))
		}
		fmt.Printf("   URL: %s\n", article.CleanedText[:min(100, len(article.CleanedText))])
		fmt.Println()

		totalSentiment += articleSentiment.Score.Overall
		sentimentCounts[articleSentiment.Classification]++
	}

	// Summary
	if len(articles) > 0 {
		avgSentiment := totalSentiment / float64(len(articles))
		fmt.Println("Summary:")
		fmt.Println("--------")
		fmt.Printf("Average sentiment: %.2f\n", avgSentiment)
		fmt.Println("Distribution:")
		for classification, count := range sentimentCounts {
			emoji := sentiment.SentimentEmoji[classification]
			fmt.Printf("  %s %s: %d articles\n", emoji, classification, count)
		}
	}

	return nil
}

func runResearch(topic string, depth, maxResults int, outputFile, provider string) error {
	fmt.Printf("🔍 Starting deep research on: %s\n", topic)
	fmt.Printf("📊 Depth: %d iterations, Max results per query: %d\n", depth, maxResults)
	fmt.Println()

	// Create LLM client for query generation (optional)
	llmClient, err := llm.NewClient("")
	if err != nil {
		fmt.Printf("⚠️  LLM client unavailable: %s\n", err)
		fmt.Printf("🔄 Research will use template-based queries instead of LLM-generated ones\n")
		llmClient = nil // Set to nil so research can handle fallback
	}
	if llmClient != nil {
		defer llmClient.Close()
	}

	// Initialize search provider based on user preference or auto-detection
	var searchProvider research.SearchProvider

	googleAPIKey := os.Getenv("GOOGLE_CSE_API_KEY")
	googleSearchID := os.Getenv("GOOGLE_CSE_ID")
	serpAPIKey := os.Getenv("SERPAPI_KEY")

	switch provider {
	case "google":
		if googleAPIKey != "" && googleSearchID != "" {
			searchProvider = research.NewGoogleCustomSearchProvider(googleAPIKey, googleSearchID)
			fmt.Printf("🔌 Using Google Custom Search (forced)\n")
		} else {
			return fmt.Errorf("google Custom Search requires both GOOGLE_CSE_API_KEY and GOOGLE_CSE_ID environment variables")
		}
	case "serpapi":
		if serpAPIKey != "" {
			searchProvider = research.NewSerpAPISearchProvider(serpAPIKey)
			fmt.Printf("🔌 Using SerpAPI (forced)\n")
		} else {
			return fmt.Errorf("serpAPI requires SERPAPI_KEY environment variable")
		}
	case "mock":
		searchProvider = research.NewMockSearchProvider()
		fmt.Printf("🔌 Using mock search provider (forced)\n")
	case "auto":
		fallthrough
	default:
		// Auto-detect based on available environment variables (priority: Google > SerpAPI > Mock)
		if googleAPIKey != "" && googleSearchID != "" {
			searchProvider = research.NewGoogleCustomSearchProvider(googleAPIKey, googleSearchID)
			fmt.Printf("🔌 Using Google Custom Search for real search results\n")
		} else if serpAPIKey != "" {
			searchProvider = research.NewSerpAPISearchProvider(serpAPIKey)
			fmt.Printf("🔌 Using SerpAPI for real search results\n")
		} else {
			searchProvider = research.NewMockSearchProvider()
			fmt.Printf("⚠️  Using mock search provider\n")
			fmt.Printf("💡 For real results, set either:\n")
			fmt.Printf("   • GOOGLE_CSE_API_KEY and GOOGLE_CSE_ID for Google Custom Search\n")
			fmt.Printf("   • SERPAPI_KEY for SerpAPI\n")
		}
	}

	researcher := research.NewDeepResearcher(llmClient, searchProvider)

	// Start research session
	session, err := researcher.StartResearch(topic, depth)
	if err != nil {
		return fmt.Errorf("failed to start research session: %w", err)
	}

	fmt.Printf("🚀 Research session started: %s\n", session.ID[:8])

	// Perform research
	err = researcher.ExecuteResearch(session)
	if err != nil {
		return fmt.Errorf("failed to perform research: %w", err)
	}

	// Display results
	fmt.Printf("\n📈 Research completed!\n")
	fmt.Printf("🔍 Queries generated: %d\n", len(session.Queries))
	fmt.Printf("📄 Results found: %d\n", len(session.Results))
	fmt.Printf("🔗 URLs discovered: %d\n", len(session.DiscoveredURLs))
	fmt.Println()

	if len(session.Queries) > 0 {
		fmt.Println("Generated Queries:")
		for i, query := range session.Queries {
			fmt.Printf("  %d. %s\n", i+1, query)
		}
		fmt.Println()
	}

	if len(session.Results) > 0 {
		fmt.Println("Top Results:")
		for i, result := range session.Results {
			if i >= 10 { // Show top 10
				break
			}
			fmt.Printf("  %d. %s\n", i+1, result.Title)
			fmt.Printf("     %s\n", result.URL)
			if result.Snippet != "" {
				fmt.Printf("     %s\n", result.Snippet[:min(100, len(result.Snippet))])
			}
			fmt.Println()
		}
	}

	// Generate links file if outputFile specified
	if outputFile != "" {
		err = researcher.GenerateLinksFile(session, outputFile)
		if err != nil {
			return fmt.Errorf("failed to generate links file: %w", err)
		}
		fmt.Printf("📝 Links file generated: %s\n", outputFile)
	}

	return nil
}

// Helper function to read links from markdown file (reused from existing code)
func readLinksFromFile(inputFile string) ([]core.Link, error) {
	// This would be implemented similar to existing link reading logic
	// For now, return empty slice as placeholder
	return []core.Link{}, fmt.Errorf("link reading not implemented in this example")
}

func runSendMessage(platform messaging.MessagePlatform, inputFile string, webhookURL string, messageFormat string, includeSentiment bool) error {
	// Get webhook URL from environment if not provided
	if webhookURL == "" {
		switch platform {
		case messaging.PlatformSlack:
			webhookURL = os.Getenv("SLACK_WEBHOOK_URL")
		case messaging.PlatformDiscord:
			webhookURL = os.Getenv("DISCORD_WEBHOOK_URL")
		}
	}

	// Validate webhook URL
	if err := messaging.ValidateWebhookURL(platform, webhookURL); err != nil {
		return fmt.Errorf("invalid webhook URL: %w", err)
	}

	// Validate message format
	validFormats := messaging.GetAvailableFormats()
	formatValid := false
	for _, format := range validFormats {
		if format == messageFormat {
			formatValid = true
			break
		}
	}
	if !formatValid {
		return fmt.Errorf("invalid message format: %s (available: %s)", messageFormat, strings.Join(validFormats, ", "))
	}

	// Read links from input file
	links, err := fetch.ReadLinksFromFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read links: %w", err)
	}

	if len(links) == 0 {
		return fmt.Errorf("no valid links found in input file")
	}

	// Initialize cache store
	cacheStore, err := store.NewStore(".briefly-cache")
	if err != nil {
		fmt.Printf("⚠️  Cache disabled due to error: %s\n", err)
		cacheStore = nil
	} else {
		defer func() { _ = cacheStore.Close() }()
	}

	// Fetch articles from cache or generate quick summaries
	var digestItems []render.DigestData
	fmt.Printf("Processing %d articles for %s message...\n", len(links), platform)

	for i, link := range links {
		fmt.Printf("Processing %d/%d: %s\n", i+1, len(links), link.URL)

		var article core.Article
		var summary string

		// Try to get from cache first
		if cacheStore != nil {
			cachedArticle, err := cacheStore.GetCachedArticle(link.URL, 24*time.Hour)
			if err == nil && cachedArticle != nil {
				article = *cachedArticle

				// Try to get cached summary
				contentHash := fmt.Sprintf("%d-%s-messaging", len(article.CleanedText), link.URL)
				cachedSummary, err := cacheStore.GetCachedSummary(link.URL, contentHash, 7*24*time.Hour)
				if err == nil && cachedSummary != nil {
					summary = cachedSummary.SummaryText
				}
			}
		}

		// If not in cache, fetch and summarize
		if article.Title == "" {
			fetchedArticle, err := fetch.FetchArticle(link)
			if err != nil {
				fmt.Printf("❌ Failed to fetch: %s\n", link.URL)
				continue
			}
			article = fetchedArticle

			if err := fetch.CleanArticleHTML(&article); err != nil {
				fmt.Printf("❌ Failed to parse: %s\n", link.URL)
				continue
			}
		}

		// Generate summary if not cached
		if summary == "" {
			// Create a quick summary for messaging (shorter than full digest)
			if len(article.CleanedText) > 500 {
				summary = article.CleanedText[:497] + "..."
			} else {
				summary = article.CleanedText
			}

			// Try to improve with LLM if available
			llmClient, err := llm.NewClient("")
			if err == nil {
				defer llmClient.Close()
				quickSummary, err := llmClient.SummarizeArticleTextWithFormat(article, "brief")
				if err == nil {
					summary = quickSummary.SummaryText
				}
			}
		}

		digestItem := render.DigestData{
			Title:          article.Title,
			URL:            link.URL,
			SummaryText:    summary,
			SentimentEmoji: article.SentimentEmoji,
			SentimentLabel: article.SentimentLabel,
			SentimentScore: article.SentimentScore,
		}

		digestItems = append(digestItems, digestItem)
		fmt.Printf("✅ %s\n", article.Title)
	}

	if len(digestItems) == 0 {
		return fmt.Errorf("no articles were successfully processed")
	}

	// Create messaging client
	slackURL := ""
	discordURL := ""
	if platform == messaging.PlatformSlack {
		slackURL = webhookURL
	} else {
		discordURL = webhookURL
	}

	client := messaging.NewMessagingClient(slackURL, discordURL)

	// Generate title
	title := fmt.Sprintf("Briefly Digest - %s", time.Now().Format("Jan 2, 2006"))

	// Convert message format
	msgFormat := messaging.MessageFormat(messageFormat)

	// Send message
	fmt.Printf("Sending %s message to %s...\n", messageFormat, platform)
	err = client.SendMessage(platform, digestItems, title, msgFormat, includeSentiment)
	if err != nil {
		return fmt.Errorf("failed to send %s message: %w", platform, err)
	}

	fmt.Printf("✅ Successfully sent %s message with %d articles\n", platform, len(digestItems))
	return nil
}

func runGenerateTTS(inputFile string, provider string, voiceID string, apiKey string, speed float64, maxArticles int, includeSummaries bool, outputDir string) error {
	// Get API key from environment if not provided
	if apiKey == "" {
		switch provider {
		case "openai":
			apiKey = os.Getenv("OPENAI_API_KEY")
		case "elevenlabs":
			apiKey = os.Getenv("ELEVENLABS_API_KEY")
		}
	}

	// Create TTS configuration
	config := &tts.TTSConfig{
		Provider:  tts.TTSProvider(provider),
		APIKey:    apiKey,
		Speed:     speed,
		OutputDir: outputDir,
	}

	// Set voice if provided
	if voiceID != "" {
		config.Voice = tts.TTSVoice{
			ID:   voiceID,
			Name: voiceID,
		}
	} else {
		// Use default voice for provider
		defaultVoices := tts.GetDefaultVoices()
		if voices, ok := defaultVoices[config.Provider]; ok && len(voices) > 0 {
			config.Voice = voices[0] // Use first default voice
		}
	}

	// Validate configuration
	if err := tts.ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid TTS configuration: %w", err)
	}

	// Read links from input file
	links, err := fetch.ReadLinksFromFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read links: %w", err)
	}

	if len(links) == 0 {
		return fmt.Errorf("no valid links found in input file")
	}

	// Initialize cache store
	cacheStore, err := store.NewStore(".briefly-cache")
	if err != nil {
		fmt.Printf("⚠️  Cache disabled due to error: %s\n", err)
		cacheStore = nil
	} else {
		defer func() { _ = cacheStore.Close() }()
	}

	// Fetch articles from cache or generate summaries
	var digestItems []render.DigestData
	fmt.Printf("Processing %d articles for TTS generation...\n", len(links))

	for i, link := range links {
		fmt.Printf("Processing %d/%d: %s\n", i+1, len(links), link.URL)

		var article core.Article
		var summary string

		// Try to get from cache first
		if cacheStore != nil {
			cachedArticle, err := cacheStore.GetCachedArticle(link.URL, 24*time.Hour)
			if err == nil && cachedArticle != nil {
				article = *cachedArticle

				// Try to get cached summary
				contentHash := fmt.Sprintf("%d-%s-tts", len(article.CleanedText), link.URL)
				cachedSummary, err := cacheStore.GetCachedSummary(link.URL, contentHash, 7*24*time.Hour)
				if err == nil && cachedSummary != nil {
					summary = cachedSummary.SummaryText
				}
			}
		}

		// If not in cache, fetch and summarize
		if article.Title == "" {
			fetchedArticle, err := fetch.FetchArticle(link)
			if err != nil {
				fmt.Printf("❌ Failed to fetch: %s\n", link.URL)
				continue
			}
			article = fetchedArticle

			if err := fetch.CleanArticleHTML(&article); err != nil {
				fmt.Printf("❌ Failed to parse: %s\n", link.URL)
				continue
			}
		}

		// Generate summary if not cached
		if summary == "" {
			// Create a brief summary for TTS
			if len(article.CleanedText) > 300 {
				summary = article.CleanedText[:297] + "..."
			} else {
				summary = article.CleanedText
			}

			// Try to improve with LLM if available
			llmClient, err := llm.NewClient("")
			if err == nil {
				defer llmClient.Close()
				briefSummary, err := llmClient.SummarizeArticleTextWithFormat(article, "brief")
				if err == nil {
					summary = briefSummary.SummaryText
				}
			}
		}

		digestItem := render.DigestData{
			Title:       article.Title,
			URL:         link.URL,
			SummaryText: summary,
		}

		digestItems = append(digestItems, digestItem)
		fmt.Printf("✅ %s\n", article.Title)
	}

	if len(digestItems) == 0 {
		return fmt.Errorf("no articles were successfully processed")
	}

	// Create TTS client
	client := tts.NewTTSClient(config)

	// Generate title
	title := fmt.Sprintf("Briefly Digest for %s", time.Now().Format("January 2, 2006"))

	// Prepare text for TTS
	ttsText := tts.PrepareTTSText(digestItems, title, includeSummaries, maxArticles)

	// Estimate audio length
	estimatedMinutes := tts.EstimateAudioLength(ttsText, speed)
	fmt.Printf("📝 Prepared %d characters of text\n", len(ttsText))
	fmt.Printf("⏱️  Estimated audio length: %.1f minutes\n", estimatedMinutes)

	// Generate filename
	dateStr := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("briefly_digest_%s.mp3", dateStr)

	// Generate audio
	fmt.Printf("🎵 Generating audio with %s using voice %s...\n", provider, config.Voice.Name)
	outputPath, err := client.GenerateAudio(ttsText, filename)
	if err != nil {
		return fmt.Errorf("failed to generate TTS audio: %w", err)
	}

	fmt.Printf("✅ TTS audio generated successfully!\n")
	fmt.Printf("📁 Output file: %s\n", outputPath)
	fmt.Printf("🎧 Articles included: %d\n", len(digestItems))

	if maxArticles > 0 && len(digestItems) > maxArticles {
		fmt.Printf("📝 Note: Limited to first %d articles (total available: %d)\n", maxArticles, len(digestItems))
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Helper functions for deep research

func parseDuration(s string) (time.Duration, error) {
	// Parse duration strings like "7d", "24h", "30m"
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	numStr := s[:len(s)-1]
	unit := s[len(s)-1:]

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid number in duration: %w", err)
	}

	switch unit {
	case "d":
		return time.Duration(num) * 24 * time.Hour, nil
	case "h":
		return time.Duration(num) * time.Hour, nil
	case "m":
		return time.Duration(num) * time.Minute, nil
	default:
		return 0, fmt.Errorf("unsupported time unit: %s", unit)
	}
}

func generateSlug(topic string) string {
	// Convert topic to filesystem-safe slug
	slug := strings.ToLower(topic)
	slug = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(slug, "-")
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 50 {
		slug = slug[:50]
	}
	return slug
}

func formatBriefAsMarkdown(brief *deepresearch.ResearchBrief) string {
	var md strings.Builder

	// Title and metadata
	md.WriteString(fmt.Sprintf("# Research Brief: %s\n\n", brief.Topic))
	md.WriteString(fmt.Sprintf("**Generated:** %s  \n", brief.GeneratedAt.Format("January 2, 2006 at 3:04 PM")))
	md.WriteString(fmt.Sprintf("**Model:** %s  \n", brief.Config.Model))
	md.WriteString(fmt.Sprintf("**Sources:** %d  \n", len(brief.Sources)))
	md.WriteString(fmt.Sprintf("**Sub-queries:** %d  \n\n", len(brief.SubQueries)))

	// Executive Summary
	md.WriteString("## Executive Summary\n\n")
	md.WriteString(brief.ExecutiveSummary)
	md.WriteString("\n\n")

	// Detailed Findings
	if len(brief.DetailedFindings) > 0 {
		md.WriteString("## Detailed Findings\n\n")
		for _, finding := range brief.DetailedFindings {
			md.WriteString(fmt.Sprintf("### %s\n\n", finding.Topic))
			md.WriteString(finding.Content)
			md.WriteString("\n\n")
		}
	}

	// Open Questions
	if len(brief.OpenQuestions) > 0 {
		md.WriteString("## Open Questions\n\n")
		for _, question := range brief.OpenQuestions {
			md.WriteString(fmt.Sprintf("- %s\n", question))
		}
		md.WriteString("\n")
	}

	// Sources
	md.WriteString("## Sources\n\n")
	for i, source := range brief.Sources {
		md.WriteString(fmt.Sprintf("[%d] **%s** - %s  \n", i+1, source.Title, source.Domain))
		md.WriteString(fmt.Sprintf("    %s  \n", source.URL))
		if source.Type != "" {
			md.WriteString(fmt.Sprintf("    *Type:* %s  \n", source.Type))
		}
		md.WriteString("\n")
	}

	// Sub-queries
	if len(brief.SubQueries) > 0 {
		md.WriteString("## Research Queries Used\n\n")
		for i, query := range brief.SubQueries {
			md.WriteString(fmt.Sprintf("%d. %s\n", i+1, query))
		}
		md.WriteString("\n")
	}

	return md.String()
}

func formatBriefAsHTML(brief *deepresearch.ResearchBrief) string {
	var html strings.Builder

	html.WriteString("<!DOCTYPE html>\n<html lang=\"en\">\n<head>\n")
	html.WriteString("    <meta charset=\"UTF-8\">\n")
	html.WriteString("    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	html.WriteString(fmt.Sprintf("    <title>Research Brief: %s</title>\n", brief.Topic))
	html.WriteString("    <style>\n")
	html.WriteString("        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }\n")
	html.WriteString("        .container { max-width: 800px; margin: 0 auto; padding: 20px; }\n")
	html.WriteString("        .metadata { background: #f5f5f5; padding: 15px; border-radius: 5px; margin-bottom: 20px; }\n")
	html.WriteString("        .source { margin-bottom: 10px; padding: 10px; border-left: 3px solid #007acc; }\n")
	html.WriteString("        h1 { color: #333; border-bottom: 2px solid #007acc; padding-bottom: 10px; }\n")
	html.WriteString("        h2 { color: #555; margin-top: 30px; }\n")
	html.WriteString("        h3 { color: #666; }\n")
	html.WriteString("    </style>\n")
	html.WriteString("</head>\n<body>\n")
	html.WriteString("    <div class=\"container\">\n")

	// Title and metadata
	html.WriteString(fmt.Sprintf("        <h1>Research Brief: %s</h1>\n", brief.Topic))
	html.WriteString("        <div class=\"metadata\">\n")
	html.WriteString(fmt.Sprintf("            <strong>Generated:</strong> %s<br>\n", brief.GeneratedAt.Format("January 2, 2006 at 3:04 PM")))
	html.WriteString(fmt.Sprintf("            <strong>Model:</strong> %s<br>\n", brief.Config.Model))
	html.WriteString(fmt.Sprintf("            <strong>Sources:</strong> %d<br>\n", len(brief.Sources)))
	html.WriteString(fmt.Sprintf("            <strong>Sub-queries:</strong> %d\n", len(brief.SubQueries)))
	html.WriteString("        </div>\n")

	// Executive Summary
	html.WriteString("        <h2>Executive Summary</h2>\n")
	html.WriteString(fmt.Sprintf("        <p>%s</p>\n", strings.ReplaceAll(brief.ExecutiveSummary, "\n", "<br>")))

	// Detailed Findings
	if len(brief.DetailedFindings) > 0 {
		html.WriteString("        <h2>Detailed Findings</h2>\n")
		for _, finding := range brief.DetailedFindings {
			html.WriteString(fmt.Sprintf("        <h3>%s</h3>\n", finding.Topic))
			html.WriteString(fmt.Sprintf("        <p>%s</p>\n", strings.ReplaceAll(finding.Content, "\n", "<br>")))
		}
	}

	// Open Questions
	if len(brief.OpenQuestions) > 0 {
		html.WriteString("        <h2>Open Questions</h2>\n")
		html.WriteString("        <ul>\n")
		for _, question := range brief.OpenQuestions {
			html.WriteString(fmt.Sprintf("            <li>%s</li>\n", question))
		}
		html.WriteString("        </ul>\n")
	}

	// Sources
	html.WriteString("        <h2>Sources</h2>\n")
	for i, source := range brief.Sources {
		html.WriteString("        <div class=\"source\">\n")
		html.WriteString(fmt.Sprintf("            <strong>[%d] %s</strong> - %s<br>\n", i+1, source.Title, source.Domain))
		html.WriteString(fmt.Sprintf("            <a href=\"%s\" target=\"_blank\">%s</a><br>\n", source.URL, source.URL))
		if source.Type != "" {
			html.WriteString(fmt.Sprintf("            <em>Type: %s</em>\n", source.Type))
		}
		html.WriteString("        </div>\n")
	}

	html.WriteString("    </div>\n")
	html.WriteString("</body>\n</html>\n")

	return html.String()
}
