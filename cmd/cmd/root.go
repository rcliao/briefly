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
	"briefly/internal/core"
	"briefly/internal/cost"
	"briefly/internal/fetch"
	"briefly/internal/llm"
	"briefly/internal/logger"
	"briefly/internal/render"
	"briefly/internal/store"
	"briefly/internal/templates"
	"briefly/internal/tui"
	"briefly/llmclient"
	"fmt"
	"os"
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
		viper.AddConfigPath(".")         // Current directory
		viper.AddConfigPath(home)        // Home directory
		viper.SetConfigType("yaml")
		viper.SetConfigName(".briefly")
	}

	// Automatically bind environment variables
	viper.AutomaticEnv()
	
	// Set defaults for configuration
	viper.SetDefault("gemini.api_key", "")
	viper.SetDefault("gemini.model", "gemini-1.5-flash-latest")
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
	Short: "Generate a digest from URLs in a markdown file",
	Long: `Process URLs from a markdown file, fetch articles, summarize them using Gemini,
and generate a markdown digest file.

Available formats: brief, standard, detailed, newsletter

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
			model = "gemini-1.5-flash-latest"
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
		defer cacheStore.Close()
		logger.Info("Cache store initialized")
	}

	// Initialize LLM client
	llmClient, err := llm.NewClient("")
	if err != nil {
		return fmt.Errorf("failed to initialize LLM client: %w", err)
	}
	defer llmClient.Close()

	var digestItems []render.DigestData
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
		contentHash := fmt.Sprintf("%d-%s", len(article.CleanedText), link.URL)
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
			generatedSummary, err := llmClient.SummarizeArticleText(article)
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

		// Create digest item
		digestItem := render.DigestData{
			Title:       article.Title,
			URL:         link.URL,
			SummaryText: summary.SummaryText,
			MyTake:      article.MyTake,
		}
		
		digestItems = append(digestItems, digestItem)
		logger.Info("Successfully processed article", "title", article.Title)
		fmt.Printf("✅ %s\n", article.Title)
	}

	// Display cache statistics
	if cacheStore != nil {
		fmt.Printf("\n📊 Cache Statistics: %d hits, %d misses (%.1f%% hit rate)\n", 
			cacheHits, cacheMisses, float64(cacheHits)/float64(cacheHits+cacheMisses)*100)
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
			combinedSummaries.WriteString(fmt.Sprintf("   Source: %s\n", item.URL))
			combinedSummaries.WriteString(fmt.Sprintf("   Summary: %s\n\n", item.SummaryText))
		}
		
		// Get API key from environment or config (prefer environment)
		apiKey := os.Getenv("GEMINI_API_KEY")
		if apiKey == "" {
			apiKey = viper.GetString("gemini.api_key")
		}
		model := viper.GetString("gemini.model")
		if model == "" {
			model = "gemini-1.5-flash-latest"
		}
		
		// Generate final digest using llmclient
		finalDigest, err := llmclient.GenerateFinalDigest(apiKey, model, combinedSummaries.String())
		if err != nil {
			logger.Error("Failed to generate final digest", err)
			fmt.Printf("⚠️  Failed to generate final digest summary, using individual summaries: %s\n", err)
			// Fall back to template rendering without final digest
			digestPath, err := templates.RenderWithTemplate(digestItems, outputDir, "", template)
			if err != nil {
				return fmt.Errorf("failed to render digest with template: %w", err)
			}
			logger.Info("Digest generated successfully with template (no final summary)", "path", digestPath, "format", format)
			fmt.Printf("✅ %s digest generated: %s\n", format, digestPath)
		} else {
			// Cache the digest if store is available
			if cacheStore != nil {
				digestID := uuid.NewString()
				var articleURLs []string
				for _, item := range digestItems {
					articleURLs = append(articleURLs, item.URL)
				}
				if err := cacheStore.CacheDigest(digestID, template.Title, string(digestFormat), finalDigest, articleURLs, model); err != nil {
					logger.Error("Failed to cache digest", err)
				}
			}

			// Render digest with template and final summary
			digestPath, err := templates.RenderWithTemplate(digestItems, outputDir, finalDigest, template)
			if err != nil {
				return fmt.Errorf("failed to render digest with template: %w", err)
			}
			logger.Info("Digest generated successfully with template and final summary", "path", digestPath, "format", format)
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
		defer cacheStore.Close()
		
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
		defer cacheStore.Close()
		
		if err := cacheStore.ClearCache(); err != nil {
			fmt.Printf("Error clearing cache: %s\n", err)
			return
		}
		
		fmt.Println("✅ Cache cleared successfully")
	},
}

func init() {
	// Add cache commands
	rootCmd.AddCommand(cacheCmd)
	rootCmd.AddCommand(listFormatsCmd)
	
	cacheCmd.AddCommand(cacheStatsCmd)
	cacheCmd.AddCommand(cacheClearCmd)
	
	cacheClearCmd.Flags().Bool("confirm", false, "Confirm cache deletion")
}


