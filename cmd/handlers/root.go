/*
Copyright © 2025 Your Name

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package handlers

import (
	"fmt"
	"os"

	"briefly/internal/config"
	"github.com/spf13/cobra"
)

var cfgFile string

// CommandOptions holds common command options (shared with unified.go)
type CommandOptions struct {
	ForceCloudAI     bool    // Override local processing
	MaxWordCount     int     // Hard limit (default 300)
	QualityThreshold float64 // Minimum article quality
	UserContext      string  // Additional context
	Interactive      bool    // Force interactive mode
	OutputFormat     string  // Output format (for backward compatibility)
}

// NewRootCmd creates the root command with all subcommands attached
func NewRootCmd() *cobra.Command {
	var options CommandOptions
	
	rootCmd := &cobra.Command{
		Use:   "briefly",
		Short: "Intelligent content assistant with unified interface",
		Long: `Briefly v3.0 - Unified content intelligence

Briefly transforms articles into actionable insights with AI-powered processing.
It automatically detects your intent and routes to the appropriate processor.

Examples:
  briefly                           # Start interactive mode
  briefly weekly-links.md           # Generate digest from file  
  briefly https://example.com       # Process single article
  briefly explore "AI trends"       # Research mode
  
The tool uses hybrid AI (local + cloud) to provide fast, cost-effective processing
while maintaining high-quality output.`,
		// Use Args to handle unknown commands gracefully
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Use unified handler directly
			handler := NewUnifiedHandler()
			return handler.Execute(cmd.Context(), args, options)
		},
	}

	// Initialize configuration
	cobra.OnInitialize(initConfig)

	// Add persistent flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.briefly.yaml)")
	
	// Add unified command options
	rootCmd.Flags().BoolVar(&options.ForceCloudAI, "cloud", false, "Force cloud AI processing (override local models)")
	rootCmd.Flags().IntVar(&options.MaxWordCount, "max-words", 300, "Maximum word count for output")
	rootCmd.Flags().Float64Var(&options.QualityThreshold, "quality", 0.6, "Minimum quality threshold (0.0-1.0)")
	rootCmd.Flags().StringVar(&options.UserContext, "context", "", "Additional context for processing")
	rootCmd.Flags().BoolVarP(&options.Interactive, "interactive", "i", false, "Force interactive mode")
	rootCmd.Flags().StringVarP(&options.OutputFormat, "format", "f", "scannable", "Output format (backward compatibility)")

	// Custom command handling for unified interface
	originalRunE := rootCmd.RunE
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// If the first argument matches a known subcommand, let Cobra handle it
		if len(args) > 0 {
			for _, subCmd := range cmd.Commands() {
				if subCmd.Name() == args[0] || subCmd.HasAlias(args[0]) {
					// This is a real subcommand, not unified args
					return cmd.Help()
				}
			}
		}
		// Not a subcommand, handle with unified interface
		return originalRunE(cmd, args)
	}

	// Add subcommands
	
	// Legacy commands (backward compatibility)
	digestCmd := NewDigestCmd()
	digestCmd.Hidden = true // Hide from help but keep functional
	rootCmd.AddCommand(digestCmd)
	
	summarizeCmd := NewSummarizeCmd()
	summarizeCmd.Hidden = true
	rootCmd.AddCommand(summarizeCmd)
	
	researchCmd := NewResearchCmd()
	researchCmd.Hidden = true
	rootCmd.AddCommand(researchCmd)
	
	// Utility commands (remain visible)
	rootCmd.AddCommand(NewCacheCmd())
	rootCmd.AddCommand(NewTUICmd())

	return rootCmd
}

// Execute runs the root command
func Execute() {
	rootCmd := NewRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Load configuration using the centralized config module
	_, err := config.Load(cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Show which config file is being used (if any)
	if config.Get().App.ConfigFile != "" {
		fmt.Fprintf(os.Stderr, "Using config file: %s\n", config.Get().App.ConfigFile)
	}
}
