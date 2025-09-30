package handlers

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"briefly/internal/core"
	"briefly/internal/services"
	"briefly/internal/store"
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
	
	// Initialize research store
	cacheStore, err := store.NewStore(".briefly-cache")
	if err != nil {
		fmt.Printf("⚠️ Failed to initialize cache store: %v\n", err)
		return nil
	}
	
	researchStore, err := services.NewSQLiteResearchStore(cacheStore.DB())
	if err != nil {
		fmt.Printf("⚠️ Failed to initialize research store: %v\n", err)
		return nil
	}

	// Initialize intelligence service with AI router and research store
	intelligenceService := services.NewIntelligenceService(
		nil, // articleProcessor - will use existing
		nil, // llmService - will use existing  
		nil, // cacheService - will use existing
		aiRouter,
		researchStore,
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
	fmt.Printf("🔄 Processing file with enhanced format...\n")
	
	// Phase 4: Use the new intelligence service for Signal+Sources processing
	input := services.ContentInput{
		FilePath: cmd.Input,
		Options: services.ProcessingOptions{
			ForceCloudAI:     cmd.Options.ForceCloudAI,
			MaxWordCount:     cmd.Options.MaxWordCount,
			QualityThreshold: cmd.Options.QualityThreshold,
			UserContext:      cmd.Options.UserContext,
			OutputFormat:     cmd.Options.OutputFormat,
		},
	}
	
	fmt.Printf("📊 Word limit: %d | Quality threshold: %.1f\n", 
		cmd.Options.MaxWordCount, cmd.Options.QualityThreshold)
	
	// Check if file has URLs that need to be extracted first
	if cmd.Input != "" {
		// For now, delegate to existing digest command for file processing
		// since we haven't implemented file URL extraction in intelligence service yet
		digestCmd := NewDigestCmd()
		args := []string{cmd.Input, "--format", cmd.Options.OutputFormat}
		
		if cmd.Options.MaxWordCount > 0 {
			args = append(args, "--max-words", fmt.Sprintf("%d", cmd.Options.MaxWordCount))
		}
		
		digestCmd.SetArgs(args)
		return digestCmd.Execute()
	}
	
	digest, err := h.intelligenceService.ProcessContent(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to process content: %w", err)
	}
	
	// Output the digest (Phase 4 will use proper template rendering)
	fmt.Printf("\n✅ Generated digest: %s\n", digest.Title)
	return nil
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
	fmt.Printf("🔍 Starting research session: %s\n", cmd.Query)
	
	// Start a new research session
	session, err := h.intelligenceService.StartResearchSession(ctx, cmd.Query)
	if err != nil {
		return fmt.Errorf("failed to start research session: %w", err)
	}
	
	fmt.Printf("✅ Research session started: %s\n", session.ID)
	fmt.Printf("📋 Current phase: %s\n", session.CurrentState.Phase)
	
	// Show initial conversation
	if len(session.ConversationLog) > 0 {
		lastTurn := session.ConversationLog[len(session.ConversationLog)-1]
		fmt.Printf("\n🤖 System: %s\n", lastTurn.Response)
	}
	
	// Start interactive conversation loop
	return h.startInteractiveResearch(ctx, session)
}

func (h *UnifiedHandler) processExplore(ctx context.Context, cmd *Command) error {
	fmt.Printf("🚀 Exploring topic: %s\n", cmd.Query)
	
	// Start exploration session - similar to research but with exploration context
	session, err := h.intelligenceService.ExploreTopicFurther(ctx, cmd.Query, nil)
	if err != nil {
		return fmt.Errorf("failed to start exploration session: %w", err)
	}
	
	fmt.Printf("✅ Exploration session started: %s\n", session.ID)
	fmt.Printf("📋 Phase: %s | Topic: %s\n", session.CurrentState.Phase, session.CurrentState.CurrentTopic)
	
	// Start interactive conversation loop
	return h.startInteractiveResearch(ctx, session)
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

// startInteractiveResearch handles the interactive conversation loop
func (h *UnifiedHandler) startInteractiveResearch(ctx context.Context, session *core.ResearchSession) error {
	fmt.Printf("\n💬 Interactive Research Session")
	fmt.Printf("\n📍 Available actions: %s", strings.Join(session.CurrentState.AvailableActions, ", "))
	fmt.Printf("\n📊 Progress: %.1f%%", session.CurrentState.Progress*100)
	fmt.Printf("\n\n💡 Type your questions, requests, or commands. Use 'quit' to exit.\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  - 'search for latest developments'\n")
	fmt.Printf("  - 'explore this topic deeper'\n")
	fmt.Printf("  - 'refine to focus on security aspects'\n")
	fmt.Printf("  - 'summarize what we've found'\n\n")

	// Create buffered reader for better input handling
	reader := bufio.NewReader(os.Stdin)

	// Interactive loop
	for {
		fmt.Print("🔍 You: ")
		
		// Read user input using buffered reader
		userInput, err := reader.ReadString('\n')
		
		// Handle EOF or error gracefully
		if err != nil {
			fmt.Printf("\n✅ Research session saved: %s\n", session.ID)
			fmt.Printf("💾 Resume later with: briefly continue %s\n", session.ID)
			break
		}
		
		// Handle special commands
		userInput = strings.TrimSpace(userInput)
		if userInput == "" {
			continue
		}
		
		if strings.ToLower(userInput) == "quit" || strings.ToLower(userInput) == "exit" {
			fmt.Printf("\n✅ Research session saved: %s\n", session.ID)
			fmt.Printf("💾 You can resume later with: briefly continue %s\n", session.ID)
			break
		}
		
		if strings.ToLower(userInput) == "help" {
			h.showResearchHelp()
			continue
		}
		
		if strings.ToLower(userInput) == "status" {
			h.showSessionStatus(session)
			continue
		}
		
		// Phase 5.4: Queue management commands
		if strings.HasPrefix(strings.ToLower(userInput), "queue") {
			if err := h.handleQueueCommand(ctx, session.ID, userInput); err != nil {
				fmt.Printf("❌ Queue error: %v\n\n", err)
			}
			continue
		}
		
		if strings.HasPrefix(strings.ToLower(userInput), "digest") {
			if err := h.handleDigestFromQueue(ctx, session.ID, userInput); err != nil {
				fmt.Printf("❌ Digest error: %v\n\n", err)
			}
			continue
		}
		
		// Process user input through intelligence service
		fmt.Printf("\n🤖 Processing...\n")
		
		response, err := h.intelligenceService.ContinueResearch(ctx, session.ID, userInput)
		if err != nil {
			fmt.Printf("❌ Error: %v\n\n", err)
			continue
		}
		
		// Display response
		fmt.Printf("🤖 System: %s\n", response.Message)
		
		// Show discovered items if any
		if len(response.DiscoveredItems) > 0 {
			fmt.Printf("\n📚 New discoveries:\n")
			for i, item := range response.DiscoveredItems {
				fmt.Printf("  %d. %s (relevance: %.2f)\n", i+1, item.Title, item.Relevance)
				fmt.Printf("     %s\n", item.URL)
			}
		}
		
		// Show available actions
		if len(response.Actions) > 0 {
			fmt.Printf("\n💡 Available actions:\n")
			for _, action := range response.Actions {
				fmt.Printf("  • %s: %s\n", action.ID, action.Description)
			}
		}
		
		// Show cost information
		if response.ProcessingCost.EstimatedUSD > 0 {
			fmt.Printf("\n💰 Cost: $%.4f\n", response.ProcessingCost.EstimatedUSD)
		}
		
		// Check if session should continue
		if !response.ContinueSession {
			fmt.Printf("\n✅ Research session completed!\n")
			fmt.Printf("📋 Session ID: %s\n", session.ID)
			break
		}
		
		fmt.Printf("\n")
	}
	
	return nil
}

// showResearchHelp displays help for research commands
func (h *UnifiedHandler) showResearchHelp() {
	fmt.Printf(`
📖 Research Session Help

Commands:
  search [query]    - Search for specific information
  explore [topic]   - Deep dive into a topic
  refine [query]    - Refine your research focus
  queue [items]     - Add items to digest queue
  summarize         - Summarize research findings
  status           - Show current session status
  help             - Show this help
  quit/exit        - Exit session (saves progress)

Examples:
  search for security vulnerabilities
  explore machine learning applications
  refine to focus on enterprise solutions
  queue the last 3 items
  summarize our findings so far

`)
}

// showSessionStatus displays current session information
func (h *UnifiedHandler) showSessionStatus(session *core.ResearchSession) {
	fmt.Printf(`
📊 Research Session Status

Session ID: %s
Query: %s
Phase: %s
Current Topic: %s
Progress: %.1f%%
Conversations: %d turns
Discovered Items: %d
Queued for Digest: %d items

Available Actions: %s
`, 
		session.ID,
		session.InitialQuery,
		session.CurrentState.Phase,
		session.CurrentState.CurrentTopic,
		session.CurrentState.Progress*100,
		len(session.ConversationLog),
		len(session.DiscoveredItems),
		len(session.QueuedForDigest),
		strings.Join(session.CurrentState.AvailableActions, ", "),
	)
}

// Phase 5.4: Queue Management Methods

// handleQueueCommand processes queue-related commands
func (h *UnifiedHandler) handleQueueCommand(ctx context.Context, sessionID string, userInput string) error {
	parts := strings.Fields(userInput)
	if len(parts) < 2 {
		return h.showQueueHelp(ctx, sessionID)
	}

	subcommand := strings.ToLower(parts[1])
	
	switch subcommand {
	case "list", "show", "status":
		return h.showQueueStatus(ctx, sessionID)
	case "process", "review":
		return h.processQueueInteractively(ctx, sessionID)
	case "clear", "empty":
		return h.clearQueue(ctx, sessionID)
	case "summary":
		return h.showQueueSummary(ctx, sessionID)
	default:
		return h.showQueueHelp(ctx, sessionID)
	}
}

// handleDigestFromQueue generates digest from queue items
func (h *UnifiedHandler) handleDigestFromQueue(ctx context.Context, sessionID string, userInput string) error {
	parts := strings.Fields(userInput)
	
	// Default options
	options := services.ProcessingOptions{
		OutputFormat:     "signal",
		MaxWordCount:     400,
		QualityThreshold: 0.6,
	}
	
	// Parse additional options
	for i, part := range parts {
		switch strings.ToLower(part) {
		case "--format":
			if i+1 < len(parts) {
				options.OutputFormat = parts[i+1]
			}
		case "--words":
			if i+1 < len(parts) {
				if wordCount, err := strconv.Atoi(parts[i+1]); err == nil {
					options.MaxWordCount = wordCount
				}
			}
		}
	}

	fmt.Printf("🔄 Generating digest from research queue...\n")
	
	digest, err := h.intelligenceService.GenerateDigestFromQueue(ctx, sessionID, options)
	if err != nil {
		return fmt.Errorf("failed to generate digest from queue: %w", err)
	}

	fmt.Printf("\n✅ Digest generated successfully!\n")
	fmt.Printf("📊 Signal: %s\n", digest.Signal.Content)
	fmt.Printf("📰 Sources: %d article groups\n", len(digest.ArticleGroups))
	fmt.Printf("💰 Cost: $%.4f\n", digest.Metadata.ProcessingCost.EstimatedUSD)
	
	return nil
}

// showQueueHelp displays queue command help
func (h *UnifiedHandler) showQueueHelp(ctx context.Context, sessionID string) error {
	fmt.Printf(`
📋 Queue Management Commands

queue list/show/status    - Show current queue items
queue process/review      - Interactively process queue items  
queue clear/empty         - Clear all queue items
queue summary            - Show queue statistics

digest                   - Generate digest from completed queue items
digest --format signal   - Generate in Signal+Sources format
digest --words 200       - Limit to 200 words

Examples:
  queue list
  queue process
  digest --format scannable

`)
	return h.showQueueSummary(ctx, sessionID)
}

// showQueueStatus displays current queue items
func (h *UnifiedHandler) showQueueStatus(ctx context.Context, sessionID string) error {
	// Get queue items by priority  
	queueItems, err := h.intelligenceService.GetQueueByPriority(ctx, sessionID, 10)
	if err != nil {
		return fmt.Errorf("failed to get queue items: %w", err)
	}

	if len(queueItems) == 0 {
		fmt.Printf("📝 Research queue is empty\n")
		return nil
	}

	fmt.Printf("\n📋 Research Queue (%d items)\n", len(queueItems))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	for i, item := range queueItems {
		statusIcon := "⏳"
		switch item.Status {
		case "completed":
			statusIcon = "✅"
		case "processing":
			statusIcon = "🔄"
		case "failed":
			statusIcon = "❌"
		case "skipped":
			statusIcon = "⏭️"
		}

		fmt.Printf("%d. %s Priority %d: %s\n", i+1, statusIcon, item.Priority, item.Title)
		fmt.Printf("   📍 %s\n", item.URL)
		fmt.Printf("   📊 Status: %s | Category: %s\n", item.Status, item.Category)
		if len(item.Tags) > 0 {
			fmt.Printf("   🏷️ Tags: %s\n", strings.Join(item.Tags, ", "))
		}
		if item.Notes != "" {
			fmt.Printf("   📝 Notes: %s\n", item.Notes)
		}
		fmt.Println()
	}

	return nil
}

// processQueueInteractively starts interactive queue processing
func (h *UnifiedHandler) processQueueInteractively(ctx context.Context, sessionID string) error {
	return h.intelligenceService.ProcessQueueItemsInteractively(ctx, sessionID)
}

// clearQueue removes all items from the queue
func (h *UnifiedHandler) clearQueue(ctx context.Context, sessionID string) error {
	fmt.Print("⚠️ Are you sure you want to clear the entire queue? (y/N): ")
	var response string
	fmt.Scanln(&response)
	
	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Printf("✅ Queue clear cancelled\n")
		return nil
	}

	if err := h.intelligenceService.ClearQueue(ctx, sessionID); err != nil {
		return fmt.Errorf("failed to clear queue: %w", err)
	}

	fmt.Printf("🗑️ Queue cleared successfully\n")
	return nil
}

// showQueueSummary displays queue statistics
func (h *UnifiedHandler) showQueueSummary(ctx context.Context, sessionID string) error {
	summary, err := h.intelligenceService.GetQueueSummary(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("failed to get queue summary: %w", err)
	}

	fmt.Printf(`
📊 Queue Summary for Session: %s

Total Items: %d
├── ⏳ Pending: %d
├── 🔄 Processing: %d  
├── ✅ Completed: %d
└── ⏭️ Skipped: %d

Ready for Digest: %s

`, 
		summary.SessionID,
		summary.TotalItems,
		summary.PendingCount,
		summary.ProcessingCount,
		summary.CompletedCount,
		summary.SkippedCount,
		func() string {
			if summary.ReadyForDigest {
				return "✅ Yes"
			}
			return "❌ No (no completed items)"
		}(),
	)

	return nil
}