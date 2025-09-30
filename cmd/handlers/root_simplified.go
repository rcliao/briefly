package handlers

import (
	"briefly/internal/config"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewSimplifiedRootCmd creates the new simplified root command
// This replaces the complex root.go with a clean, focused interface
func NewSimplifiedRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "briefly",
		Short: "Generate quality digests for your weekly LinkedIn posts",
		Long: `Briefly - Simplified Content Digest Tool

A focused tool for creating quality weekly digests from curated URLs.

Core workflows:
  • Weekly Digest: Process markdown file with URLs → LinkedIn-ready digest
  • Quick Read: Single URL → Fast summary with key points

Features:
  • Smart caching (avoid redundant API calls)
  • Topic clustering (automatic article grouping)
  • Executive summaries (story-driven narratives)
  • LinkedIn-optimized output

Examples:
  # Generate weekly digest
  briefly digest input/weekly-links.md

  # Quick read a single article
  briefly read https://example.com/article

  # Check cache statistics
  briefly cache stats`,
		Version: "3.0.0-simplified",
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .briefly.yaml)")

	// Add subcommands
	rootCmd.AddCommand(NewDigestSimplifiedCmd())
	rootCmd.AddCommand(NewReadSimplifiedCmd())
	rootCmd.AddCommand(NewCacheCmd()) // Keep existing cache command

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