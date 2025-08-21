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
	// Phase 3: Initialize hybrid AI services
	
	// Initialize local model service (Ollama)
	localModel := services.NewOllamaService("http://localhost:11434", "llama3.2:3b")
	
	// Initialize cost controller
	costController := services.NewCostController(".briefly-cache", 1.0) // $1/day budget
	
	// Initialize AI router (cloud LLM will be nil for now)
	aiRouter := services.NewAIRouter(localModel, nil, costController)
	
	// Initialize intelligence service with AI router
	intelligenceService := services.NewIntelligenceService(
		nil, // articleProcessor - will use existing
		nil, // llmService - will use existing  
		nil, // cacheService - will use existing
		aiRouter,
	)

	return &UnifiedHandler{
		intelligenceService: intelligenceService,
		cacheService:        nil, // Will initialize when needed
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
		
	// Phase 3: AI management commands
	case "ai-status", "ai-capabilities", "ollama-status":
		return &Command{
			Mode:    ModeCache, // Reuse cache mode for AI commands
			Input:   strings.Join(args, " "), // Full command
			Options: options,
		}, nil
		
	case "cost-analytics", "cost-report":
		return &Command{
			Mode:    ModeCache, // Reuse cache mode for cost commands
			Input:   strings.Join(args, " "), // Full command
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
	fmt.Printf("📄 Processing digest with Signal+Sources format...\n")
	
	// Phase 2: Use existing digest command but with new output format
	// This provides immediate value while we build the full pipeline
	
	if cmd.Input != "" {
		fmt.Printf("Input file: %s\n", cmd.Input)
		return h.processDigestFile(ctx, cmd)
	}
	
	if len(cmd.URLs) > 0 {
		fmt.Printf("URLs to process: %v\n", cmd.URLs)
		return h.processDigestURLs(ctx, cmd)
	}
	
	return fmt.Errorf("no input file or URLs provided")
}

// processDigestFile handles file-based digest generation
func (h *UnifiedHandler) processDigestFile(ctx context.Context, cmd *Command) error {
	// Delegate to existing digest command for Phase 2
	// But modify output format
	
	fmt.Printf("🔄 Processing file with enhanced format...\n")
	
	// Use existing digest command with scannable format
	digestCmd := NewDigestCmd()
	args := []string{cmd.Input, "--format", "scannable"}
	
	// Apply word limit if specified (default is 300)
	if cmd.Options.MaxWordCount > 0 {
		args = append(args, "--max-words", fmt.Sprintf("%d", cmd.Options.MaxWordCount))
	}
	
	digestCmd.SetArgs(args)
	
	fmt.Printf("📊 Word limit: %d | Quality threshold: %.1f\n", 
		cmd.Options.MaxWordCount, cmd.Options.QualityThreshold)
	
	return digestCmd.Execute()
}

// processDigestURLs handles URL-based digest generation  
func (h *UnifiedHandler) processDigestURLs(ctx context.Context, cmd *Command) error {
	// For Phase 2, create a temporary file with URLs and process it
	
	tempFile := "/tmp/briefly_urls.md"
	
	// Write URLs to temp file
	var content strings.Builder
	content.WriteString("# Temporary URL List\n\n")
	for _, url := range cmd.URLs {
		content.WriteString(fmt.Sprintf("- %s\n", url))
	}
	
	if err := os.WriteFile(tempFile, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	
	// Process the temp file
	cmd.Input = tempFile
	return h.processDigestFile(ctx, cmd)
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
	// Check for special Phase 3 AI management commands
	args := strings.Fields(cmd.Input)
	if len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "ollama-status", "ai-status":
			return h.checkOllamaStatus(ctx)
		case "cost-analytics", "cost-report":
			return h.showCostAnalytics(ctx)
		case "ai-capabilities":
			return h.showAICapabilities(ctx)
		}
	}
	
	// For traditional cache commands, show what we're doing
	if len(args) > 0 {
		fmt.Printf("💾 Cache operation: %s\n", args[0])
	}
	
	// Delegate to existing cache command for other operations
	cacheCmd := NewCacheCmd()
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
	// Phase 3: Enhanced user profile with cost awareness
	return &core.UserProfile{
		PreferLocal:      true,
		MaxCloudCost:     0.5, // $0.50 per operation to encourage local usage
		QualityThreshold: 0.6,
	}
}

// Phase 3 helper methods for AI system management

// checkOllamaStatus checks if Ollama is running and available
func (h *UnifiedHandler) checkOllamaStatus(ctx context.Context) error {
	fmt.Println("🔍 Checking Ollama Status...")
	
	// Create local model service to check status
	localModel := services.NewOllamaService("http://localhost:11434", "llama3.2:3b")
	
	available, err := localModel.IsAvailable(ctx)
	if err != nil {
		fmt.Printf("❌ Ollama connection failed: %v\n", err)
		fmt.Println("\n💡 To install Ollama:")
		fmt.Println("   1. Visit: https://ollama.ai")
		fmt.Println("   2. Download and install")
		fmt.Println("   3. Run: ollama pull llama3.2:3b")
		return err
	}
	
	if !available {
		fmt.Println("❌ Ollama is not available")
		return fmt.Errorf("ollama not available")
	}
	
	fmt.Println("✅ Ollama is running and available")
	
	// Check model availability
	if err := localModel.Initialize(ctx); err != nil {
		fmt.Printf("⚠️ Model not available: %v\n", err)
		fmt.Println("\n💡 To install required model:")
		fmt.Println("   ollama pull llama3.2:3b")
		return err
	}
	
	fmt.Println("✅ Model llama3.2:3b is available")
	fmt.Println("\n🎯 Local AI is ready for cost-effective processing!")
	
	return nil
}

// showCostAnalytics displays current cost analytics
func (h *UnifiedHandler) showCostAnalytics(ctx context.Context) error {
	fmt.Println("💰 Cost Analytics (Last 7 Days)")
	fmt.Println("================================")
	
	costController := services.NewCostController(".briefly-cache", 1.0)
	
	analytics, err := costController.GetCostAnalytics(ctx, 7)
	if err != nil {
		return fmt.Errorf("failed to get cost analytics: %w", err)
	}
	
	fmt.Printf("📊 Total Spent: $%.4f\n", analytics.TotalSpent)
	fmt.Printf("🏠 Local Processing: $%.4f (%.1f%%)\n", 
		analytics.LocalVsCloud["local"],
		(analytics.LocalVsCloud["local"]/analytics.TotalSpent)*100)
	fmt.Printf("☁️ Cloud Processing: $%.4f (%.1f%%)\n", 
		analytics.LocalVsCloud["cloud"],
		(analytics.LocalVsCloud["cloud"]/analytics.TotalSpent)*100)
	
	fmt.Println("\n📈 Cost by Task:")
	for task, cost := range analytics.CostByTask {
		if cost > 0 {
			fmt.Printf("   %s: $%.4f\n", task, cost)
		}
	}
	
	if len(analytics.Recommendations) > 0 {
		fmt.Println("\n💡 Recommendations:")
		for _, rec := range analytics.Recommendations {
			fmt.Printf("   • %s\n", rec)
		}
	}
	
	// Show daily breakdown if available
	if len(analytics.DailyCosts) > 0 {
		fmt.Println("\n📅 Daily Breakdown:")
		for _, daily := range analytics.DailyCosts {
			total := daily.LocalCost + daily.CloudCost
			if total > 0 {
				fmt.Printf("   %s: $%.4f (%d tasks)\n", 
					daily.Date.Format("Jan 02"), total, daily.TaskCount)
			}
		}
	}
	
	return nil
}

// showAICapabilities displays current AI system capabilities
func (h *UnifiedHandler) showAICapabilities(ctx context.Context) error {
	fmt.Println("🤖 AI System Capabilities")
	fmt.Println("=========================")
	
	localModel := services.NewOllamaService("http://localhost:11434", "llama3.2:3b")
	costController := services.NewCostController(".briefly-cache", 1.0)
	aiRouter := services.NewAIRouter(localModel, nil, costController)
	
	capabilities, err := aiRouter.GetCapabilities(ctx)
	if err != nil {
		return fmt.Errorf("failed to get capabilities: %w", err)
	}
	
	fmt.Printf("🏠 Local Models: %s\n", formatAvailable(capabilities.LocalModelsAvailable))
	fmt.Printf("☁️ Cloud Models: %s\n", formatAvailable(capabilities.CloudModelsAvailable))
	fmt.Printf("🎯 Max Local Complexity: %.1f/1.0\n", capabilities.MaxLocalComplexity)
	fmt.Printf("💰 Cost Per Token: $%.8f\n", capabilities.CostPerToken)
	
	fmt.Println("\n🔧 Supported Tasks:")
	for _, task := range capabilities.SupportedTasks {
		fmt.Printf("   • %s\n", task)
	}
	
	// Show routing strategy
	fmt.Println("\n🔀 Routing Strategy:")
	fmt.Println("   • Simple tasks (categorize, analyze): Local preferred")
	fmt.Println("   • Medium tasks (summarize): Local for cost, Cloud for quality")
	fmt.Println("   • Complex tasks (synthesize, generate): Cloud required")
	
	// Show current budget status
	budget, _ := costController.GetDailyBudget(ctx)
	spent, _ := costController.GetSpentToday(ctx)
	remaining := budget - spent
	
	fmt.Printf("\n💳 Daily Budget: $%.2f\n", budget)
	fmt.Printf("💸 Spent Today: $%.4f\n", spent)
	fmt.Printf("💰 Remaining: $%.4f\n", remaining)
	
	return nil
}

// formatAvailable formats availability status
func formatAvailable(available bool) string {
	if available {
		return "✅ Available"
	}
	return "❌ Not Available"
}