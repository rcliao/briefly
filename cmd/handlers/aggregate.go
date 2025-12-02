package handlers

import (
	"briefly/internal/config"
	"briefly/internal/core"
	"briefly/internal/fetch"
	"briefly/internal/llm"
	"briefly/internal/logger"
	"briefly/internal/observability"
	"briefly/internal/persistence"
	"briefly/internal/sources"
	"briefly/internal/themes"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// classifierAdapter adapts themes.Classifier to sources.ThemeClassifier
type classifierAdapter struct {
	classifier *themes.Classifier
}

// GetBestMatch implements sources.ThemeClassifier interface
func (c *classifierAdapter) GetBestMatch(ctx context.Context, article core.Article, themes []core.Theme, minRelevance float64) (sources.ThemeClassificationResult, error) {
	result, err := c.classifier.GetBestMatch(ctx, article, themes, minRelevance)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	// *themes.ClassificationResult implements sources.ThemeClassificationResult interface
	return result, nil
}

// NewAggregateCmd creates the aggregate command for news aggregation with inline classification
func NewAggregateCmd() *cobra.Command {
	var (
		maxArticles  int
		concurrency  int
		sinceHours   int
		dryRun       bool
		minRelevance float64
		themeFilter  string
		withLangfuse bool
	)

	cmd := &cobra.Command{
		Use:   "aggregate",
		Short: "Aggregate and classify news from RSS feeds",
		Long: `Aggregate fetches articles from RSS/Atom feeds and classifies them by theme.

This command (Phase 1 - Enhanced RSS Aggregation):
  • Fetches new items from all active feeds
  • Fetches full article content
  • Classifies articles by theme using LLM
  • Filters articles below relevance threshold
  • Stores classified articles in database
  • Respects feed update frequencies (uses conditional GET)
  • Runs concurrently for better performance

Typical usage:
  • Run daily via cron: briefly aggregate --since 24
  • Filter by relevance: briefly aggregate --min-relevance 0.6
  • Single theme: briefly aggregate --theme "AI & Machine Learning"
  • Test new feeds: briefly aggregate --dry-run

Examples:
  # Aggregate last 24 hours with default relevance threshold (0.4)
  briefly aggregate --since 24

  # Only high-relevance articles
  briefly aggregate --min-relevance 0.7

  # Only articles matching specific theme
  briefly aggregate --theme "AI & Machine Learning" --min-relevance 0.6

  # Limit processing and enable observability
  briefly aggregate --max-articles 20 --with-langfuse

  # Dry run to see what would be fetched
  briefly aggregate --dry-run`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAggregateWithClassification(cmd.Context(), maxArticles, concurrency, sinceHours, minRelevance, themeFilter, dryRun, withLangfuse)
		},
	}

	cmd.Flags().IntVar(&maxArticles, "max-articles", 50, "Maximum articles to fetch per feed")
	cmd.Flags().IntVar(&concurrency, "concurrency", 5, "Number of articles to process concurrently")
	cmd.Flags().IntVar(&sinceHours, "since", 24, "Only fetch articles published in the last N hours")
	cmd.Flags().Float64Var(&minRelevance, "min-relevance", 0.4, "Minimum relevance score to keep article (0.0-1.0)")
	cmd.Flags().StringVar(&themeFilter, "theme", "", "Only process articles matching this theme name")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be fetched without storing")
	cmd.Flags().BoolVar(&withLangfuse, "with-langfuse", false, "Enable LangFuse observability tracking")

	return cmd
}

func runAggregateWithClassification(ctx context.Context, maxArticles, concurrency, sinceHours int, minRelevance float64, themeFilter string, dryRun bool, withLangfuse bool) error {
	log := logger.Get()
	log.Info("Starting news aggregation with classification",
		"max_articles", maxArticles,
		"concurrency", concurrency,
		"since_hours", sinceHours,
		"min_relevance", minRelevance,
		"theme_filter", themeFilter,
		"dry_run", dryRun,
		"with_langfuse", withLangfuse,
	)

	// Load configuration
	_, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	cfg := config.Get()

	// Get database connection string
	dbConnStr := cfg.Database.ConnectionString
	if dbConnStr == "" {
		dbConnStr = os.Getenv("DATABASE_URL")
		if dbConnStr == "" {
			return fmt.Errorf("database connection string not configured (set database.connection_string in config or DATABASE_URL env var)")
		}
	}

	// Connect to database
	db, err := persistence.NewPostgresDB(dbConnStr)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	// Test database connection
	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	log.Info("Connected to database")

	// Initialize LLM client for classification
	modelName := cfg.AI.Gemini.Model
	if modelName == "" {
		modelName = "gemini-flash-lite-latest"
	}

	llmClient, err := llm.NewClient(modelName)
	if err != nil {
		return fmt.Errorf("failed to create LLM client: %w", err)
	}

	// Initialize observability clients
	var posthogClient *observability.PostHogClient
	var langfuseClient *observability.LangFuseClient

	// Initialize PostHog if configured
	posthogAPIKey := cfg.Observability.PostHog.APIKey
	if posthogAPIKey == "" {
		posthogAPIKey = os.Getenv("POSTHOG_API_KEY")
	}
	if posthogAPIKey != "" {
		var err error
		posthogClient, err = observability.NewPostHogClient()
		if err != nil {
			log.Warn("Failed to initialize PostHog", "error", err)
			posthogClient = nil
		}
	}

	// Initialize LangFuse if requested
	if withLangfuse {
		var err error
		langfuseClient, err = observability.NewLangFuseClient()
		if err != nil {
			log.Warn("Failed to initialize LangFuse", "error", err)
			langfuseClient = nil
		} else {
			log.Info("LangFuse observability enabled for classification tracking")
		}
	}

	// Create base classifier with PostHog tracking
	var posthogTracker themes.PostHogTracker
	if posthogClient != nil {
		posthogTracker = posthogClient
	}
	baseClassifier := themes.NewClassifier(llmClient, posthogTracker)

	// Optionally wrap with LangFuse tracking
	var finalClassifier *themes.Classifier
	if langfuseClient != nil && langfuseClient.IsEnabled() {
		tracedClassifier := themes.NewTracedClassifier(baseClassifier, langfuseClient)
		// For now, use base classifier (TracedClassifier wrapping will be fixed later)
		finalClassifier = baseClassifier
		_ = tracedClassifier
	} else {
		finalClassifier = baseClassifier
	}

	// Create article processor
	processor := fetch.NewContentProcessor()

	// Create source manager
	sourceMgr := sources.NewManager(db)

	// Wrap classifier to match sources.ThemeClassifier interface
	classifierWrapper := &classifierAdapter{classifier: finalClassifier}

	// Check if there are any active themes
	themesList, err := db.Themes().List(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to list themes: %w", err)
	}

	if len(themesList) == 0 {
		log.Warn("No active themes found. Add themes first using: briefly theme add <name>")
		fmt.Println("⚠️  No active themes configured")
		fmt.Println("   Add themes using: briefly theme add \"Theme Name\"")
		return nil
	}

	log.Info("Found active themes", "count", len(themesList))
	for i, theme := range themesList {
		log.Info(fmt.Sprintf("  [%d] %s", i+1, theme.Name), "keywords", len(theme.Keywords))
	}

	// Check if there are any active feeds
	feeds, err := sourceMgr.ListFeeds(ctx, true)
	if err != nil {
		return fmt.Errorf("failed to list feeds: %w", err)
	}

	if len(feeds) == 0 {
		log.Warn("No active feeds found. Add feeds first using: briefly feed add <url>")
		fmt.Println("⚠️  No active feeds configured")
		fmt.Println("   Add feeds using: briefly feed add <url>")
		return nil
	}

	log.Info("Found active feeds", "count", len(feeds))
	for i, feed := range feeds {
		log.Info(fmt.Sprintf("  [%d] %s", i+1, feed.Title), "url", feed.URL)
	}

	if dryRun {
		log.Info("Dry run mode - no articles will be classified or stored")
		return nil
	}

	// Prepare aggregation with classification options
	opts := sources.AggregateWithClassificationOptions{
		MaxArticles:    maxArticles,
		MinRelevance:   minRelevance,
		ThemeFilter:    themeFilter,
		Since:          time.Now().Add(-time.Duration(sinceHours) * time.Hour),
		MaxConcurrency: concurrency,
	}

	// Run aggregation with inline classification
	startTime := time.Now()
	result, err := sourceMgr.AggregateWithClassification(ctx, processor, classifierWrapper, opts)
	duration := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("aggregation with classification failed: %w", err)
	}

	// Display results
	log.Info("Aggregation with classification completed",
		"duration", duration.String(),
		"feeds_fetched", result.FeedsFetched,
		"articles_fetched", result.ArticlesFetched,
		"articles_classified", result.ArticlesClassified,
		"articles_filtered", result.ArticlesFiltered,
		"articles_failed", result.ArticlesFailed,
	)

	fmt.Println("\n📊 Aggregation Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Duration:             %s\n", duration.Round(time.Millisecond))
	fmt.Printf("Feeds Fetched:        %d\n", result.FeedsFetched)
	fmt.Printf("Articles Fetched:     %d\n", result.ArticlesFetched)
	fmt.Printf("Articles Classified:  %d\n", result.ArticlesClassified)
	fmt.Printf("Articles Filtered:    %d (below relevance threshold)\n", result.ArticlesFiltered)
	fmt.Printf("Articles Failed:      %d\n", result.ArticlesFailed)

	if len(result.ThemeDistribution) > 0 {
		fmt.Println("\n🎨 Theme Distribution:")
		for themeName, count := range result.ThemeDistribution {
			fmt.Printf("  • %s: %d articles\n", themeName, count)
		}
	}

	if len(result.Errors) > 0 {
		fmt.Println("\n⚠️  Errors:")
		for i, err := range result.Errors {
			if i >= 5 {
				fmt.Printf("  ... and %d more errors\n", len(result.Errors)-5)
				break
			}
			fmt.Printf("  [%d] %v\n", i+1, err)
		}
	}

	if result.ArticlesClassified > 0 {
		fmt.Printf("\n✅ Successfully aggregated and classified %d articles\n", result.ArticlesClassified)
		fmt.Println("Next steps:")
		fmt.Println("  • View classified articles: briefly feed list-items")
		fmt.Println("  • Generate digest: briefly digest generate --since 7")
	} else if result.ArticlesFiltered > 0 {
		fmt.Println("\nℹ️  All articles filtered (below relevance threshold)")
		fmt.Println("   Try lowering --min-relevance or adding more relevant feeds")
	} else {
		fmt.Println("\nℹ️  No articles found")
		fmt.Println("   Try adjusting --since or --max-articles parameters")
	}

	return nil
}
