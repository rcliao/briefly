package handlers

import (
	"briefly/internal/config"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var cfgFile string // Configuration file path

// NewSimplifiedRootCmd creates the root command
func NewSimplifiedRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "briefly",
		Short: "Weekly digest generator for curated links",
		Long: `Briefly — turn a curated list of URLs into a scannable weekly digest.

Workflow:
  1. Collect links during the week in a markdown file (input/YYYY-MM-DD.md)
  2. Generate the digest: briefly digest from-file input/YYYY-MM-DD.md
  3. Paste the result into LinkedIn/Slack/email

Commands:
  digest from-file  Generate a digest from a curated markdown file
  read              Quick summary of a single article
  cache             Manage the local article/summary cache

Requires GEMINI_API_KEY (env or .env file).

Examples:
  # Generate weekly digest
  briefly digest from-file input/weekly.md

  # Quick read a single article
  briefly read https://example.com/article

  # Check cache statistics
  briefly cache stats`,
		Version: "4.0.0-editorial",
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is .briefly.yaml)")

	// Add subcommands
	rootCmd.AddCommand(NewDigestCmd())
	rootCmd.AddCommand(NewReadSimplifiedCmd())
	rootCmd.AddCommand(NewCacheCmd())

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
