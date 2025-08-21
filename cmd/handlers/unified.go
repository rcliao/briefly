package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"briefly/internal/core"
	"briefly/internal/services"
	"github.com/spf13/cobra"
)

// HandlerMode represents different operation modes
type HandlerMode string

const (
	ModeInteractive HandlerMode = "interactive" // No args - start chat
	ModeDigest      HandlerMode = "digest"      // File/URLs provided
	ModeResearch    HandlerMode = "research"    // Query provided
	ModeExplore     HandlerMode = "explore"     // Exploration command
	ModeCache       HandlerMode = "cache"       // Cache operations
	ModeTUI         HandlerMode = "tui"         // Terminal UI
)

// Command represents a parsed user command
type Command struct {
	Mode    HandlerMode
	Input   string   // File path or URL
	Query   string   // Search/research query
	URLs    []string // Multiple URLs
	Options CommandOptions
}

// CommandOptions is now defined in root.go to avoid duplication

// UnifiedHandler handles all command types through a single interface
type UnifiedHandler struct {
	intelligenceService services.IntelligenceService
	cacheService        services.CacheService
}

// NewUnifiedHandler creates a new unified command handler
func NewUnifiedHandler() *UnifiedHandler {
	// TODO: Initialize services properly in Phase 2
	return &UnifiedHandler{
		// intelligenceService: services.NewIntelligenceService(),
		// cacheService:        services.NewCacheService(),
	}
}

// NewUnifiedCmd creates the unified command that replaces all others
func NewUnifiedCmd() *cobra.Command {
	var options CommandOptions
	
	cmd := &cobra.Command{
		Use:   "briefly [input] [query...]",
		Short: "Intelligent content assistant with unified interface",
		Long: `Briefly v3.0 - Unified content intelligence
		
Examples:
  briefly                           # Start interactive mode
  briefly weekly-links.md           # Generate digest from file
  briefly https://example.com       # Process single article
  briefly explore "AI trends"       # Research mode
  briefly cache stats               # Cache operations
  
The command automatically detects your intent and routes to the appropriate processor.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			handler := NewUnifiedHandler()
			return handler.Execute(cmd.Context(), args, options)
		},
	}

	// Add command options
	cmd.Flags().BoolVar(&options.ForceCloudAI, "cloud", false, "Force cloud AI processing (override local models)")
	cmd.Flags().IntVar(&options.MaxWordCount, "max-words", 300, "Maximum word count for output")
	cmd.Flags().Float64Var(&options.QualityThreshold, "quality", 0.6, "Minimum quality threshold (0.0-1.0)")
	cmd.Flags().StringVar(&options.UserContext, "context", "", "Additional context for processing")
	cmd.Flags().BoolVarP(&options.Interactive, "interactive", "i", false, "Force interactive mode")
	cmd.Flags().StringVarP(&options.OutputFormat, "format", "f", "scannable", "Output format (backward compatibility)")

	return cmd
}

// Execute processes the unified command
func (h *UnifiedHandler) Execute(ctx context.Context, args []string, options CommandOptions) error {
	// Parse command and determine mode
	command, err := h.ParseCommand(args, options)
	if err != nil {
		return fmt.Errorf("command parsing failed: %w", err)
	}

	// Route to appropriate processor
	switch command.Mode {
	case ModeInteractive:
		return h.processInteractive(ctx, command)
	case ModeDigest:
		return h.processDigest(ctx, command)
	case ModeResearch:
		return h.processResearch(ctx, command)
	case ModeExplore:
		return h.processExplore(ctx, command)
	case ModeCache:
		return h.processCache(ctx, command)
	case ModeTUI:
		return h.processTUI(ctx, command)
	default:
		return fmt.Errorf("unsupported mode: %s", command.Mode)
	}
}

// ParseCommand analyzes the arguments and determines the appropriate mode
func (h *UnifiedHandler) ParseCommand(args []string, options CommandOptions) (*Command, error) {
	// Force interactive mode if requested
	if options.Interactive {
		return &Command{
			Mode:    ModeInteractive,
			Options: options,
		}, nil
	}

	// No arguments - start interactive mode
	if len(args) == 0 {
		return &Command{
			Mode:    ModeInteractive,
			Options: options,
		}, nil
	}

	// Check for specific command keywords
	firstArg := strings.ToLower(args[0])
	
	switch firstArg {
	case "explore":
		if len(args) < 2 {
			return nil, fmt.Errorf("explore command requires a query")
		}
		return &Command{
			Mode:    ModeExplore,
			Query:   strings.Join(args[1:], " "),
			Options: options,
		}, nil
		
	case "cache":
		return &Command{
			Mode:    ModeCache,
			Input:   strings.Join(args[1:], " "), // cache subcommand
			Options: options,
		}, nil
		
	case "tui":
		return &Command{
			Mode:    ModeTUI,
			Options: options,
		}, nil
	}

	// Check if first argument is a file or URL
	if h.isFileOrURL(args[0]) {
		command := &Command{
			Mode:    ModeDigest,
			Options: options,
		}
		
		// Determine if it's a file or URL
		if h.isFile(args[0]) {
			command.Input = args[0]
		} else if h.isURL(args[0]) {
			// Single URL or multiple URLs
			command.URLs = args
		}
		
		return command, nil
	}

	// Default: treat as research query
	return &Command{
		Mode:    ModeResearch,
		Query:   strings.Join(args, " "),
		Options: options,
	}, nil
}

// Helper methods for command detection
func (h *UnifiedHandler) isFileOrURL(input string) bool {
	return h.isFile(input) || h.isURL(input)
}

func (h *UnifiedHandler) isFile(input string) bool {
	// Check if it exists as a file
	if _, err := os.Stat(input); err == nil {
		return true
	}
	
	// Check for common file extensions
	ext := strings.ToLower(filepath.Ext(input))
	commonExtensions := []string{".md", ".txt", ".html", ".pdf", ".json"}
	for _, validExt := range commonExtensions {
		if ext == validExt {
			return true
		}
	}
	
	return false
}

func (h *UnifiedHandler) isURL(input string) bool {
	return strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://")
}

// Mode processors (temporary implementations for Phase 1)
func (h *UnifiedHandler) processInteractive(ctx context.Context, cmd *Command) error {
	fmt.Println("🤖 Briefly v3.0 - Interactive Mode")
	fmt.Println("Type 'help' for commands or 'exit' to quit")
	fmt.Println()
	
	// TODO: Implement interactive session in Phase 5
	fmt.Println("Interactive mode will be implemented in Phase 5")
	fmt.Println("For now, use: briefly <file> or briefly explore <query>")
	
	return nil
}

func (h *UnifiedHandler) processDigest(ctx context.Context, cmd *Command) error {
	fmt.Printf("📄 Processing digest request...\n")
	
	if cmd.Input != "" {
		fmt.Printf("Input file: %s\n", cmd.Input)
	}
	
	if len(cmd.URLs) > 0 {
		fmt.Printf("URLs to process: %v\n", cmd.URLs)
	}
	
	// TODO: Implement unified digest processing in Phase 2-4
	fmt.Println("Unified digest processing will be implemented in Phase 2-4")
	fmt.Println("For now, use the existing 'digest' command")
	
	return nil
}

func (h *UnifiedHandler) processResearch(ctx context.Context, cmd *Command) error {
	fmt.Printf("🔍 Research query: %s\n", cmd.Query)
	
	// TODO: Implement research processing in Phase 5
	fmt.Println("Research processing will be implemented in Phase 5")
	fmt.Println("For now, use the existing 'research' command")
	
	return nil
}

func (h *UnifiedHandler) processExplore(ctx context.Context, cmd *Command) error {
	fmt.Printf("🚀 Exploring topic: %s\n", cmd.Query)
	
	// TODO: Implement exploration mode in Phase 5
	fmt.Println("Exploration mode will be implemented in Phase 5")
	
	return nil
}

func (h *UnifiedHandler) processCache(ctx context.Context, cmd *Command) error {
	fmt.Printf("💾 Cache operation: %s\n", cmd.Input)
	
	// Delegate to existing cache command for now
	cacheCmd := NewCacheCmd()
	args := strings.Fields(cmd.Input)
	cacheCmd.SetArgs(args)
	return cacheCmd.Execute()
}

func (h *UnifiedHandler) processTUI(ctx context.Context, cmd *Command) error {
	fmt.Println("📺 Starting Terminal UI...")
	
	// Delegate to existing TUI command for now
	tuiCmd := NewTUICmd()
	return tuiCmd.Execute()
}

// getUserProfile loads user preferences and context
func (h *UnifiedHandler) getUserProfile(ctx context.Context) *core.UserProfile {
	// TODO: Implement user profile loading in Phase 2
	return &core.UserProfile{
		PreferLocal:      true,
		MaxCloudCost:     1.0, // $1 per operation
		QualityThreshold: 0.6,
	}
}