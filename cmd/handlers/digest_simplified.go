package handlers

// ============================================================================
// ⚠️  DEPRECATED: This is the v1.0 file-based digest pipeline
// ============================================================================
//
// This file implements the legacy file-based digest generation workflow:
// - Input: Markdown file with URLs
// - Processing: Direct pipeline execution (parse → fetch → cluster → digest)
// - Output: Single consolidated digest markdown file
//
// STATUS: Deprecated, intended for removal in future version
//
// MIGRATION PATH:
// Use the v2.0 database-driven workflow instead:
//   1. briefly aggregate --since 24          (fetch from RSS + manual URLs)
//   2. briefly digest generate --since 7     (generate from classified articles)
//
// DO NOT MODIFY THIS FILE - It will be removed once v2.0 is fully validated.
// All new features should go into digest_generate.go (v2.0 database pipeline).
//
// ============================================================================

import (
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
// DEPRECATED: Use `briefly digest generate` instead (v2.0 database-driven)
func NewDigestSimplifiedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "digest [input-file.md]",
		Short: "[DEPRECATED] Generate a weekly digest from URLs in a markdown file",
		Long: `[DEPRECATED] Generate a LinkedIn-ready digest from a list of URLs.

⚠️  This command is deprecated in favor of the v2.0 database-driven pipeline.
   Prefer using: briefly aggregate + briefly digest generate

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

	// v2.0: Generate multiple digests (one per cluster)
	results, err := pipe.GenerateDigests(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\n❌ Digest generation failed: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Fprintf(os.Stderr, "\n❌ No digests were generated\n")
		os.Exit(1)
	}

	// Display results
	elapsed := time.Since(startTime)

	fmt.Println("\n" + "═══════════════════════════════════════")
	fmt.Printf("✅ Generated %d Digests Successfully!\n", len(results))
	fmt.Println("═══════════════════════════════════════")

	// Show each digest that was generated
	for i, result := range results {
		fmt.Printf("\n📄 Digest %d/%d: %s\n", i+1, len(results), result.Digest.Title)
		fmt.Printf("   • File: %s\n", result.MarkdownPath)
		fmt.Printf("   • Articles: %d\n", result.Digest.ArticleCount)
		if result.BannerPath != "" {
			fmt.Printf("   • Banner: %s\n", result.BannerPath)
		}
	}

	// Aggregate statistics from first result (they all share the same stats)
	if len(results) > 0 {
		stats := results[0].Stats
		fmt.Println("\n📊 Statistics:")
		fmt.Printf("   • Total URLs: %d\n", stats.TotalURLs)
		fmt.Printf("   • Successful: %d\n", stats.SuccessfulArticles)
		fmt.Printf("   • Failed: %d\n", stats.FailedArticles)
		fmt.Printf("   • Cache Hits: %d\n", stats.CacheHits)
		fmt.Printf("   • Cache Misses: %d\n", stats.CacheMisses)
		fmt.Printf("   • Topic Clusters: %d\n", len(results))
		fmt.Printf("   • Processing Time: %v\n", elapsed.Round(time.Second))

		if stats.CacheHits > 0 {
			cachePercent := float64(stats.CacheHits) / float64(stats.TotalURLs) * 100
			fmt.Printf("   • Cache Hit Rate: %.1f%%\n", cachePercent)
		}
	}

	fmt.Println("\n💡 Next steps:")
	fmt.Printf("   1. Review digests in: %s/\n", outputDir)
	fmt.Println("   2. Each digest focuses on a specific topic cluster")
	if !withBanner {
		fmt.Println("   3. Optional: Re-run with --with-banner to generate social images")
	}

	fmt.Println("\n" + "═══════════════════════════════════════")

	// Log completion
	logger.Info("Digest generation completed",
		"output_dir", outputDir,
		"digests_generated", len(results),
		"articles", results[0].Stats.SuccessfulArticles,
		"duration", elapsed)
}

// getOutputFilePath generates the output file path
// getOutputFilePath removed as unused

// validateConfiguration checks that required configuration is present
// validateConfiguration removed as unused

// printProgressBar prints a simple progress indicator
// printProgressBar removed as unused
