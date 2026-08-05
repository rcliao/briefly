package handlers

import (
	"github.com/spf13/cobra"
)

// NewDigestCmd creates the parent digest command with subcommands
func NewDigestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "digest",
		Short: "Generate digests from curated links",
		Long: `Generate a digest from a curated markdown file of URLs.

Subcommands:
  from-file - Generate digest from curated markdown file

Examples:
  # Generate from curated markdown file
  briefly digest from-file input/weekly.md

  # Slack-optimized format
  briefly digest from-file input/weekly.md --format slack`,
	}

	cmd.AddCommand(NewDigestFromFileCmd())

	return cmd
}
