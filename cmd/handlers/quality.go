package handlers

import (
	"github.com/spf13/cobra"
)

// NewQualityCmd creates the parent quality command with subcommands
func NewQualityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quality",
		Short: "Evaluate and track digest quality metrics",
		Long: `Evaluate digest quality, track metrics over time, and analyze improvements.

Subcommands:
  audit     - Audit quality of recent digests (coverage, vagueness, specificity)
  report    - Generate detailed quality report for a specific digest
  trends    - Analyze quality trends over time

Examples:
  # Audit last 10 digests
  briefly quality audit --limit 10

  # Audit digests from last 30 days
  briefly quality audit --since 30

  # Get detailed report for specific digest
  briefly quality report <digest-id>

  # Analyze quality trends
  briefly quality trends --since 90`,
	}

	// Add subcommands
	cmd.AddCommand(NewQualityAuditCmd())
	cmd.AddCommand(NewQualityReportCmd())
	cmd.AddCommand(NewQualityTrendsCmd())

	return cmd
}
