package handlers

import (
	"briefly/internal/tui"
	"fmt"

	"github.com/spf13/cobra"
)

// NewTUICmd creates the TUI command
func NewTUICmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Launch the Briefly Terminal User Interface",
		Long:  `Launch the Briefly TUI to browse and manage articles and summaries.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Launching TUI...")
			tui.StartTUI()
		},
	}
}
