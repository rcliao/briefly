package handlers

import (
	"github.com/spf13/cobra"
)

// NewDigestCmd creates the parent digest command with subcommands
func NewDigestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "digest",
		Short: "Generate digests from various sources",
		Long: `Generate digests from markdown files or database articles.

Subcommands:
  generate  - Generate digest from classified articles in database
  [file]    - Generate digest from markdown file with URLs (default)

Examples:
  # Generate from database (last 7 days)
  briefly digest generate --since 7

  # Generate from markdown file
  briefly digest input/links.md`,
	}

	// Add subcommands
	cmd.AddCommand(NewDigestGenerateCmd())      // Database-driven digest
	cmd.AddCommand(NewDigestSimplifiedCmd())    // File-driven digest (as subcommand)

	return cmd
}
