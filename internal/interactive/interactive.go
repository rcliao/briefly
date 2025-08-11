package interactive

import (
	"briefly/internal/render"
	"bufio"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Handler manages the interactive article selection workflow
type Handler struct {
	session *render.InteractiveSession
	scanner *bufio.Scanner
}

// NewHandler creates a new interactive handler
func NewHandler() *Handler {
	return &Handler{
		scanner: bufio.NewScanner(os.Stdin),
	}
}

// StartSession initializes an interactive session with processed articles
func (h *Handler) StartSession(articles []render.DigestData) *render.InteractiveSession {
	// Sort articles by priority score (highest first)
	sortedArticles := make([]render.DigestData, len(articles))
	copy(sortedArticles, articles)
	sort.Slice(sortedArticles, func(i, j int) bool {
		return sortedArticles[i].PriorityScore > sortedArticles[j].PriorityScore
	})

	session := &render.InteractiveSession{
		SessionID:   uuid.New().String(),
		ArticlePool: sortedArticles,
		CreatedAt:   time.Now(),
		Status:      "created",
	}

	h.session = session
	return session
}

// SelectGameChangerArticle presents articles for user selection
func (h *Handler) SelectGameChangerArticle() (*render.DigestData, error) {
	if h.session == nil {
		return nil, fmt.Errorf("no active session")
	}

	h.session.Status = "selecting"
	
	fmt.Printf("\n📖 Select Game-Changer Article (%d articles processed)\n", len(h.session.ArticlePool))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	
	// Display articles with priority scores
	for i, article := range h.session.ArticlePool {
		// Truncate long titles
		title := article.Title
		if len(title) > 60 {
			title = title[:57] + "..."
		}
		
		// Get category emoji from MyTake
		categoryEmoji := h.extractCategoryEmoji(article.MyTake)
		
		fmt.Printf("[%d] %s %s (Score: %.2f)\n", 
			i+1, categoryEmoji, title, article.PriorityScore)
	}
	
	fmt.Printf("\nEnter number (1-%d), or 'a' for auto-selection: ", len(h.session.ArticlePool))
	
	for {
		if !h.scanner.Scan() {
			return nil, fmt.Errorf("failed to read input")
		}
		
		input := strings.TrimSpace(h.scanner.Text())
		
		// Handle auto-selection
		if strings.ToLower(input) == "a" || strings.ToLower(input) == "auto" {
			selectedArticle := &h.session.ArticlePool[0] // Highest priority
			selectedArticle.UserSelected = false // Mark as auto-selected
			h.session.SelectedArticle = selectedArticle
			fmt.Printf("✅ Auto-selected: %s\n", selectedArticle.Title)
			return selectedArticle, nil
		}
		
		// Parse number selection
		num, err := strconv.Atoi(input)
		if err != nil || num < 1 || num > len(h.session.ArticlePool) {
			fmt.Printf("Invalid selection. Enter number (1-%d) or 'a' for auto: ", len(h.session.ArticlePool))
			continue
		}
		
		// Select the article
		selectedArticle := &h.session.ArticlePool[num-1]
		selectedArticle.UserSelected = true // Mark as user-selected
		h.session.SelectedArticle = selectedArticle
		fmt.Printf("✅ Selected: %s\n", selectedArticle.Title)
		return selectedArticle, nil
	}
}

// CaptureUserTake prompts for and captures user's personal take
func (h *Handler) CaptureUserTake() (string, error) {
	if h.session == nil || h.session.SelectedArticle == nil {
		return "", fmt.Errorf("no article selected")
	}
	
	h.session.Status = "inputting"
	
	fmt.Printf("\n📝 Add your personal take on: %s\n", h.session.SelectedArticle.Title)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("Enter your commentary (press Enter twice to finish, or just Enter to skip):")
	fmt.Print("\n> ")
	
	var lines []string
	emptyLineCount := 0
	
	for {
		if !h.scanner.Scan() {
			break
		}
		
		line := h.scanner.Text()
		
		// If empty line
		if strings.TrimSpace(line) == "" {
			emptyLineCount++
			// Two consecutive empty lines = done
			if emptyLineCount >= 2 {
				break
			}
			// First empty line = just skip if no content yet
			if len(lines) == 0 {
				break
			}
			lines = append(lines, line)
		} else {
			emptyLineCount = 0
			lines = append(lines, line)
			fmt.Print("> ")
		}
	}
	
	userTake := strings.TrimSpace(strings.Join(lines, "\n"))
	h.session.UserTake = userTake
	h.session.SelectedArticle.UserTakeText = userTake
	
	if userTake == "" {
		fmt.Println("No take provided - continuing with auto-generated content.")
	} else {
		fmt.Printf("✅ Take captured (%d characters)\n", len(userTake))
	}
	
	return userTake, nil
}

// CompleteSession finalizes the interactive session
func (h *Handler) CompleteSession() error {
	if h.session == nil {
		return fmt.Errorf("no active session")
	}
	
	now := time.Now()
	h.session.CompletedAt = &now
	h.session.Status = "completed"
	
	fmt.Printf("\n✅ Interactive session completed in %.1fs\n", now.Sub(h.session.CreatedAt).Seconds())
	return nil
}

// GetSession returns the current session
func (h *Handler) GetSession() *render.InteractiveSession {
	return h.session
}

// extractCategoryEmoji extracts emoji from category info in MyTake field
func (h *Handler) extractCategoryEmoji(myTake string) string {
	if myTake == "" {
		return "📄"
	}
	
	// Extract emoji from format like "🔥 Breaking & Hot | insight"
	parts := strings.Split(myTake, " ")
	if len(parts) > 0 {
		// Check if first part contains emoji
		emoji := parts[0]
		if len(emoji) > 0 {
			// Basic emoji detection (Unicode range for common emojis)
			for _, r := range emoji {
				if r >= 0x1F300 && r <= 0x1F9FF {
					return emoji
				}
			}
		}
	}
	
	return "📄" // Default document emoji
}