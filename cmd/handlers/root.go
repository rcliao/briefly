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

// NewRootCmd creates the root command with all subcommands attached
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "briefly",
		Short: "Briefly is a CLI tool for fetching, summarizing, and managing articles.",
		Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Do Stuff Here
		},
	}

	// Initialize configuration
	cobra.OnInitialize(initConfig)

	// Add persistent flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.briefly.yaml)")
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")

	// Add subcommands
	rootCmd.AddCommand(NewDigestCmd())
	rootCmd.AddCommand(NewResearchCmd())
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
