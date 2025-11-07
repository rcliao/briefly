package handlers

import (
	"github.com/spf13/cobra"
)

// NewDigestCmd creates the parent digest command with subcommands
func NewDigestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "digest",
		Short: "Manage and generate digests",
		Long: `Generate and manage digests from classified articles in database.

Subcommands:
  generate  - Generate digest from classified articles in database
  list      - List recent digests from database
  show      - Display a specific digest

Examples:
  # Generate from database (last 7 days)
  briefly digest generate --since 7

  # List recent digests
  briefly digest list --limit 20

  # Show a specific digest
  briefly digest show abc123`,
	}

	// Add subcommands
	cmd.AddCommand(NewDigestGenerateCmd())      // Database-driven digest generation
	cmd.AddCommand(NewDigestListCmd())          // List recent digests
	cmd.AddCommand(NewDigestShowCmd())          // Show specific digest

	return cmd
}
