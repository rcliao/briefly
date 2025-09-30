package handlers

import (
	"briefly/internal/config"
	"briefly/internal/llm"
	"briefly/internal/logger"
	"briefly/internal/pipeline"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

// NewDigestSimplifiedCmd creates the new simplified digest command
func NewDigestSimplifiedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "digest [input-file.md]",
		Short: "Generate a weekly digest from URLs in a markdown file",
		Long: `Generate a LinkedIn-ready digest from a list of URLs.

This command orchestrates the full digest pipeline:
1. Parse URLs from markdown file
2. Fetch and summarize articles
3. Cluster articles by topic
4. Generate executive summary
5. Render LinkedIn-ready markdown

Examples:
  briefly digest input/links.md
  briefly digest --output digests input/weekly-links.md
  briefly digest --with-banner input/links.md
  briefly digest --dry-run input/links.md`,
		Args: cobra.ExactArgs(1),
		Run:  digestSimplifiedRun,
	}

	// Flags
	cmd.Flags().StringP("output", "o", "digests", "Output directory for digest file")
	cmd.Flags().Bool("with-banner", false, "Generate AI banner image for social sharing")
	cmd.Flags().Bool("dry-run", false, "Estimate costs without processing")
	cmd.Flags().Bool("no-cache", false, "Disable caching (fetch all articles fresh)")

	return cmd
}

func digestSimplifiedRun(cmd *cobra.Command, args []string) {
	startTime := time.Now()

	// Get flags
	inputFile := args[0]
	outputDir, _ := cmd.Flags().GetString("output")
	withBanner, _ := cmd.Flags().GetBool("with-banner")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	noCache, _ := cmd.Flags().GetBool("no-cache")

	logger.Info("Starting simplified digest generation",
		"input", inputFile,
		"output", outputDir,
		"banner", withBanner,
		"dry_run", dryRun)

	// Validate input file exists
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "❌ Input file not found: %s\n", inputFile)
		os.Exit(1)
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to create output directory: %v\n", err)
		os.Exit(1)
	}

	// Initialize LLM client
	fmt.Println("🔧 Initializing AI client...")
	llmClient, err := llm.NewClient("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to initialize AI client: %v\n", err)
		fmt.Fprintf(os.Stderr, "💡 Make sure GEMINI_API_KEY is set in your environment or .env file\n")
		os.Exit(1)
	}
	defer llmClient.Close()

	// Build pipeline
	fmt.Println("🔧 Building processing pipeline...")
	builder := pipeline.NewBuilder().
		WithLLMClient(llmClient).
		WithCacheDir(".briefly-cache")

	if noCache {
		builder = builder.WithoutCache()
	}

	if withBanner {
		builder = builder.WithBanner("tech")
	}

	pipe, err := builder.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to build pipeline: %v\n", err)
		os.Exit(1)
	}

	// Handle dry run (cost estimation)
	if dryRun {
		fmt.Println("💰 Dry run mode - estimating costs...")
		fmt.Println("📊 Cost estimation not yet implemented in simplified pipeline")
		fmt.Println("💡 Remove --dry-run flag to process articles")
		return
	}

	// Execute pipeline
	ctx := context.Background()
	opts := pipeline.DigestOptions{
		InputFile:      inputFile,
		OutputPath:     outputDir,
		GenerateBanner: withBanner,
	}

	fmt.Printf("\n📖 Processing digest from: %s\n\n", inputFile)

	result, err := pipe.GenerateDigest(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Digest generation failed: %v\n", err)
		os.Exit(1)
	}

	// Display results
	elapsed := time.Since(startTime)

	fmt.Println("\n" + "═══════════════════════════════════════")
	fmt.Println("✅ Digest Generated Successfully!")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("📄 Output: %s\n", result.MarkdownPath)

	if result.BannerPath != "" {
		fmt.Printf("🖼️  Banner: %s\n", result.BannerPath)
	}

	fmt.Println("\n📊 Statistics:")
	fmt.Printf("   • Total URLs: %d\n", result.Stats.TotalURLs)
	fmt.Printf("   • Successful: %d\n", result.Stats.SuccessfulArticles)
	fmt.Printf("   • Failed: %d\n", result.Stats.FailedArticles)
	fmt.Printf("   • Cache Hits: %d\n", result.Stats.CacheHits)
	fmt.Printf("   • Cache Misses: %d\n", result.Stats.CacheMisses)
	fmt.Printf("   • Clusters: %d\n", result.Stats.ClustersGenerated)
	fmt.Printf("   • Processing Time: %v\n", elapsed.Round(time.Second))

	if result.Stats.CacheHits > 0 {
		cachePercent := float64(result.Stats.CacheHits) / float64(result.Stats.TotalURLs) * 100
		fmt.Printf("   • Cache Hit Rate: %.1f%%\n", cachePercent)
	}

	fmt.Println("\n💡 Next steps:")
	fmt.Printf("   1. Review digest: cat %s\n", result.MarkdownPath)
	fmt.Println("   2. Copy content to LinkedIn")
	if !withBanner {
		fmt.Println("   3. Optional: Re-run with --with-banner to generate social image")
	}

	fmt.Println("\n" + "═══════════════════════════════════════\n")

	// Log completion
	logger.Info("Digest generation completed",
		"output", result.MarkdownPath,
		"articles", result.Stats.SuccessfulArticles,
		"duration", elapsed)
}

// getOutputFilePath generates the output file path
func getOutputFilePath(outputDir string) string {
	timestamp := time.Now().Format("2006-01-02")
	return fmt.Sprintf("%s/digest-%s.md", outputDir, timestamp)
}

// validateConfiguration checks that required configuration is present
func validateConfiguration() error {
	cfg := config.Get()

	// Check for API key
	if cfg.AI.Gemini.APIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY not configured")
	}

	return nil
}

// printProgressBar prints a simple progress indicator
func printProgressBar(current, total int, message string) {
	percent := float64(current) / float64(total) * 100
	fmt.Printf("\r⏳ %s [%d/%d] %.0f%%", message, current, total, percent)
	if current == total {
		fmt.Println() // New line when complete
	}
}