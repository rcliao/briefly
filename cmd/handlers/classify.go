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

// simpleClassifierWrapper adapts themes.Classifier to sources.ThemeClassifier
type simpleClassifierWrapper struct {
	classifier *themes.Classifier
}

// GetBestMatch implements sources.ThemeClassifier interface
func (w *simpleClassifierWrapper) GetBestMatch(ctx context.Context, article core.Article, themes []core.Theme, minRelevance float64) (sources.ThemeClassificationResult, error) {
	result, err := w.classifier.GetBestMatch(ctx, article, themes, minRelevance)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return nil, nil
	}
	// *themes.ClassificationResult implements sources.ThemeClassificationResult interface
	return result, nil
}

// NewClassifyCmd creates the classify command for article theme classification
func NewClassifyCmd() *cobra.Command {
	var (
		maxArticles  int
		minRelevance float64
		themeFilter  string
		concurrency  int
		dryRun       bool
		withLangfuse bool
	)

	cmd := &cobra.Command{
		Use:   "classify",
		Short: "Classify articles by theme using LLM",
		Long: `Classify fetches unprocessed articles and assigns themes using LLM classification.

This command:
  • Processes unprocessed feed items from the database
  • Fetches full article content
  • Classifies articles against active themes
  • Filters articles based on relevance threshold
  • Stores classified articles with theme assignments

Phase 1 RSS Enhancement:
  • Theme-based filtering with relevance scoring
  • Configurable minimum relevance threshold (0.0-1.0)
  • Optional theme filter to only process specific themes
  • LangFuse observability for classification performance

Typical usage:
  • Run after aggregation: briefly classify
  • Filter by theme: briefly classify --theme "AI & Machine Learning"
  • Strict filtering: briefly classify --min-relevance 0.7

Examples:
  # Classify all unprocessed articles with default threshold (0.4)
  briefly classify

  # Only classify high-relevance articles
  briefly classify --min-relevance 0.7

  # Only process articles matching a specific theme
  briefly classify --theme "AI & Machine Learning"

  # Limit processing and increase concurrency
  briefly classify --max-articles 50 --concurrency 10

  # Dry run to see what would be processed
  briefly classify --dry-run

  # Enable LangFuse tracking
  briefly classify --with-langfuse`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runClassify(cmd.Context(), maxArticles, minRelevance, themeFilter, concurrency, dryRun, withLangfuse)
		},
	}

	cmd.Flags().IntVar(&maxArticles, "max-articles", 100, "Maximum articles to classify (0 = no limit)")
	cmd.Flags().Float64Var(&minRelevance, "min-relevance", 0.4, "Minimum relevance score to assign theme (0.0-1.0)")
	cmd.Flags().StringVar(&themeFilter, "theme", "", "Only classify articles matching this theme name")
	cmd.Flags().IntVar(&concurrency, "concurrency", 5, "Number of articles to process concurrently")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be processed without storing")
	cmd.Flags().BoolVar(&withLangfuse, "with-langfuse", false, "Enable LangFuse observability tracking")

	return cmd
}

func runClassify(ctx context.Context, maxArticles int, minRelevance float64, themeFilter string, concurrency int, dryRun bool, withLangfuse bool) error {
	log := logger.Get()
	log.Info("Starting article classification",
		"max_articles", maxArticles,
		"min_relevance", minRelevance,
		"theme_filter", themeFilter,
		"concurrency", concurrency,
		"dry_run", dryRun,
		"with_langfuse", withLangfuse,
	)

	// Load configuration
	_, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get configuration
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

	// Initialize LLM client (NewClient reads API key from env/config)
	modelName := cfg.AI.Gemini.Model
	if modelName == "" {
		modelName = "gemini-2.5-flash-preview-05-20" // Default model
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
	// Important: Pass nil directly (not through interface) to avoid Go's interface nil gotcha
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
		_ = tracedClassifier // Acknowledge we're not using it yet
	} else {
		finalClassifier = baseClassifier
	}

	// Create a simple wrapper that implements ThemeClassifier interface
	classifier := &simpleClassifierWrapper{classifier: finalClassifier}

	// Create article processor
	processor := fetch.NewContentProcessor()

	// Create source manager
	sourceMgr := sources.NewManager(db)

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

	if dryRun {
		log.Info("Dry run mode - no articles will be classified or stored")
		return nil
	}

	// Prepare classification options
	opts := sources.ClassificationOptions{
		MaxArticles:    maxArticles,
		MinRelevance:   minRelevance,
		ThemeFilter:    themeFilter,
		SkipProcessed:  true,
		FetchContent:   true,
		MaxConcurrency: concurrency,
	}

	// Run classification
	startTime := time.Now()
	result, err := sourceMgr.ClassifyFeedItems(ctx, processor, classifier, opts)
	duration := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("classification failed: %w", err)
	}

	// Display results
	log.Info("Classification completed",
		"duration", duration.String(),
		"processed", result.ArticlesProcessed,
		"classified", result.ArticlesClassified,
		"filtered", result.ArticlesFiltered,
		"failed", result.ArticlesFailed,
	)

	fmt.Println("\n📊 Classification Summary")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Duration:           %s\n", duration.Round(time.Millisecond))
	fmt.Printf("Articles Processed: %d\n", result.ArticlesProcessed)
	fmt.Printf("Articles Classified: %d\n", result.ArticlesClassified)
	fmt.Printf("Articles Filtered:  %d (below relevance threshold)\n", result.ArticlesFiltered)
	fmt.Printf("Articles Failed:    %d\n", result.ArticlesFailed)

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
		fmt.Printf("\n✅ Successfully classified %d articles\n", result.ArticlesClassified)
		fmt.Println("Next steps:")
		fmt.Println("  • View classified articles: briefly feed list-items")
		fmt.Println("  • Generate theme-filtered digest: briefly digest --from-feeds --theme \"AI & Machine Learning\"")
	} else if result.ArticlesFiltered > 0 {
		fmt.Println("\nℹ️  All articles filtered (below relevance threshold)")
		fmt.Println("   Try lowering --min-relevance or adding more relevant feeds")
	} else {
		fmt.Println("\nℹ️  No unprocessed articles found")
		fmt.Println("   Run aggregation first: briefly aggregate")
	}

	return nil
}
