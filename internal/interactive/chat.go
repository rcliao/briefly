package interactive

import (
	"briefly/internal/core"
	"briefly/internal/llm"
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// ChatHandler manages the interactive chat session with the LLM
type ChatHandler struct {
	llmClient   *llm.Client
	scanner     *bufio.Scanner
	article     *core.Article
	articleURL  string
	summary     *core.Summary
	session     *llm.ChatSession
	history     []ChatMessage
}

// ChatMessage represents a single message in the chat history
type ChatMessage struct {
	Role      string    // "user" or "assistant"
	Content   string
	Timestamp time.Time
}

// NewChatHandler creates a new chat handler
func NewChatHandler(llmClient *llm.Client) *ChatHandler {
	return &ChatHandler{
		llmClient: llmClient,
		scanner:   bufio.NewScanner(os.Stdin),
		history:   make([]ChatMessage, 0),
	}
}

// StartChatSession initializes a chat session with article context
func (h *ChatHandler) StartChatSession(article *core.Article, articleURL string, summary *core.Summary) error {
	h.article = article
	h.articleURL = articleURL
	h.summary = summary

	// Create initial context for the chat
	initialContext := fmt.Sprintf(`You are an AI assistant helping the user explore and understand an article they just read.

Article Title: %s
Article URL: %s

Summary:
%s

Article Content (truncated to 3000 chars for context):
%s

You have access to the full article content and can answer questions about it, explain concepts, provide additional insights, or discuss related topics. Be helpful, accurate, and conversational.`,
		article.Title,
		articleURL,
		summary.SummaryText,
		truncateText(article.CleanedText, 3000))

	// Initialize the chat session with the LLM
	ctx := context.Background()
	session, err := h.llmClient.StartChatSession(ctx, initialContext)
	if err != nil {
		return fmt.Errorf("failed to start chat session: %w", err)
	}
	h.session = session

	// Display chat introduction
	fmt.Printf("\n💬 Interactive Chat Session Started\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("Chat about the article: %s\n", article.Title)
	fmt.Printf("\nCommands:\n")
	fmt.Printf("  /help    - Show available commands\n")
	fmt.Printf("  /save    - Save conversation to file\n")
	fmt.Printf("  /context - Show article context\n")
	fmt.Printf("  /exit    - End chat session\n")
	fmt.Printf("  quit     - End chat session\n")
	fmt.Printf("\nType your question or 'quit' to exit.\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	return nil
}

// RunChatLoop runs the main interactive chat loop
func (h *ChatHandler) RunChatLoop() error {
	for {
		fmt.Print("You: ")
		if !h.scanner.Scan() {
			break
		}

		input := strings.TrimSpace(h.scanner.Text())
		if input == "" {
			continue
		}

		// Check for commands
		if strings.HasPrefix(input, "/") {
			if err := h.handleCommand(input); err != nil {
				fmt.Printf("Error: %v\n", err)
			}
			continue
		}

		// Check for quit
		if strings.ToLower(input) == "quit" || strings.ToLower(input) == "exit" {
			fmt.Println("\n👋 Chat session ended. Goodbye!")
			break
		}

		// Process user input
		if err := h.processUserInput(input); err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}

	return nil
}

// processUserInput sends the user's message to the LLM and displays the response
func (h *ChatHandler) processUserInput(input string) error {
	// Add user message to history
	h.history = append(h.history, ChatMessage{
		Role:      "user",
		Content:   input,
		Timestamp: time.Now(),
	})

	// Send message to LLM
	ctx := context.Background()
	response, err := h.llmClient.SendChatMessage(ctx, h.session, input)
	if err != nil {
		return fmt.Errorf("failed to get response: %w", err)
	}

	// Add assistant response to history
	h.history = append(h.history, ChatMessage{
		Role:      "assistant",
		Content:   response,
		Timestamp: time.Now(),
	})

	// Display response
	fmt.Printf("\nAssistant: %s\n\n", response)
	return nil
}

// handleCommand processes chat commands
func (h *ChatHandler) handleCommand(command string) error {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "/help":
		h.showHelp()
	case "/save":
		filename := "chat-log.md"
		if len(parts) > 1 {
			filename = strings.Join(parts[1:], " ")
		}
		return h.saveConversation(filename)
	case "/context":
		h.showContext()
	case "/exit":
		fmt.Println("\n👋 Chat session ended. Goodbye!")
		os.Exit(0)
	default:
		fmt.Printf("Unknown command: %s. Type /help for available commands.\n", parts[0])
	}

	return nil
}

// showHelp displays available commands
func (h *ChatHandler) showHelp() {
	fmt.Println("\n📚 Available Commands:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  /help          - Show this help message")
	fmt.Println("  /save [file]   - Save conversation to file (default: chat-log.md)")
	fmt.Println("  /context       - Show article context being used")
	fmt.Println("  /exit          - End chat session")
	fmt.Println("  quit           - End chat session")
	fmt.Println("\nYou can ask questions about:")
	fmt.Println("  - The article's main points and arguments")
	fmt.Println("  - Technical details or concepts mentioned")
	fmt.Println("  - Related topics or implications")
	fmt.Println("  - Comparisons with other ideas or technologies")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

// showContext displays the article context
func (h *ChatHandler) showContext() {
	fmt.Println("\n📋 Article Context:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Title: %s\n", h.article.Title)
	fmt.Printf("URL: %s\n", h.articleURL)
	fmt.Printf("Content Length: %d characters\n", len(h.article.CleanedText))
	fmt.Printf("Summary Length: %d characters\n", len(h.summary.SummaryText))
	fmt.Printf("Chat Messages: %d\n", len(h.history))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
}

// saveConversation saves the chat history to a file
func (h *ChatHandler) saveConversation(filename string) error {
	var content strings.Builder

	// Add header
	content.WriteString(fmt.Sprintf("# Chat Log - %s\n\n", h.article.Title))
	content.WriteString(fmt.Sprintf("**Article URL:** %s\n", h.articleURL))
	content.WriteString(fmt.Sprintf("**Date:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))
	
	// Add summary
	content.WriteString("## Article Summary\n\n")
	content.WriteString(fmt.Sprintf("%s\n\n", h.summary.SummaryText))
	
	// Add conversation
	content.WriteString("## Conversation\n\n")
	for _, msg := range h.history {
		if msg.Role == "user" {
			content.WriteString(fmt.Sprintf("**You:** %s\n\n", msg.Content))
		} else {
			content.WriteString(fmt.Sprintf("**Assistant:** %s\n\n", msg.Content))
		}
	}

	// Write to file
	if err := os.WriteFile(filename, []byte(content.String()), 0644); err != nil {
		return fmt.Errorf("failed to save conversation: %w", err)
	}

	fmt.Printf("💾 Conversation saved to: %s\n", filename)
	return nil
}

// truncateText truncates text to specified length
func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
}