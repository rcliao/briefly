package handlers

import (
	"briefly/internal/config"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string // Configuration file path

// NewSimplifiedRootCmd creates the new simplified root command
// This replaces the complex root.go with a clean, focused interface
func NewSimplifiedRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "briefly",
		Short: "LLM-focused news aggregator and digest generator",
		Long: `Briefly - Content Digest & News Aggregation Tool

A focused tool for:
  • Automated news aggregation from RSS/Atom feeds
  • Quality weekly digests from classified articles
  • Quick summaries of individual articles

Core workflows:
  • News Aggregation: Fetch and store articles from feeds
  • Weekly Digest: Generate LinkedIn-ready digest from classified articles
  • Quick Read: Single URL → Fast summary with key points
  • Feed Management: Add/remove/manage RSS feed sources

Features:
  • RSS/Atom feed support with conditional GET
  • Smart caching (avoid redundant API calls)
  • Topic clustering (automatic article grouping)
  • Hierarchical summarization (ALL articles included)
  • Executive summaries (story-driven narratives)
  • LinkedIn-optimized output
  • PostgreSQL persistence for scalable storage

Examples:
  # Add RSS feeds
  briefly feed add https://hnrss.org/newest

  # Aggregate news (run daily)
  briefly aggregate --since 24

  # Generate weekly digest from database
  briefly digest generate --since 7

  # Quick read a single article
  briefly read https://example.com/article

  # Check cache statistics
  briefly cache stats`,
		Version: "3.1.0-hierarchical-summarization",
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .briefly.yaml)")

	// Add subcommands
	rootCmd.AddCommand(NewMigrateCmd())        // NEW: Database migrations
	rootCmd.AddCommand(NewAggregateCmd())      // NEW: News aggregation
	rootCmd.AddCommand(NewClassifyCmd())       // NEW: Article classification (Phase 1)
	rootCmd.AddCommand(NewFeedCmd())           // NEW: Feed management
	rootCmd.AddCommand(NewThemeCmd())          // NEW: Theme management (Phase 0)
	rootCmd.AddCommand(NewManualURLCmd())      // NEW: Manual URL management (Phase 0)
	rootCmd.AddCommand(NewServeCmd())          // NEW: HTTP server
	rootCmd.AddCommand(NewQualityCmd())        // NEW: Quality evaluation and metrics (Phase 1)
	rootCmd.AddCommand(NewDigestCmd())         // Digest commands (file-based and database-based)
	rootCmd.AddCommand(NewReadSimplifiedCmd()) // Existing: Quick read
	rootCmd.AddCommand(NewCacheCmd())          // Existing: Cache management
	rootCmd.AddCommand(NewSearchCmd())         // NEW: Semantic search (Phase 2)

	// Initialize config before running any command
	cobra.OnInitialize(initSimplifiedConfig)

	return rootCmd
}

// initSimplifiedConfig reads in config file and ENV variables
func initSimplifiedConfig() {
	_, err := config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
		// Don't exit - allow running with just environment variables
	}
}

// Execute runs the root command
func ExecuteSimplified() error {
	rootCmd := NewSimplifiedRootCmd()
	return rootCmd.Execute()
}
