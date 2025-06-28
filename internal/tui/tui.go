package tui

import (
	"briefly/internal/config"
	"briefly/internal/core"
	"briefly/internal/llm"
	"briefly/internal/render"
	"briefly/internal/store"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewMode int

const (
	viewMainMenu viewMode = iota
	viewTeamContextSetup
	viewDigestPipeline
	viewArticleReview
	viewRelevanceTuning
	viewDigestHistory
	viewEditMyTake
)

type pipelineStep int

const (
	stepFetching pipelineStep = iota
	stepSummarizing
	stepClustering
	stepGeneratingInsights
	stepFilteringRelevance
	stepGeneratingDigest
	stepComplete
)

// Model represents the state of the TUI application.
type model struct {
	// Core state
	store    *store.Store
	llmClient *llm.Client
	width    int
	height   int
	mode     viewMode
	quitting bool

	// Navigation
	selectedIdx    int
	
	// Menu state
	menuItems []string
	
	// Team context setup
	teamContext     config.Team
	editingField    string
	editingValue    string
	
	// Pipeline state
	currentStep     pipelineStep
	pipelineStatus  string
	
	// Article processing
	articles         []core.Article  
	digestItems      []render.DigestData
	insights         map[string]string
	relevanceScores  map[string]float64
	
	// Digest history
	digests         []core.Digest
	
	// My-take editing
	editingMyTake   string
	editingDigestID string
	
	// UI state
	showHelp        bool
	errorMessage    string
	statusMessage   string
}

// InitialModel returns the initial state of the TUI model.
func InitialModel() model {
	cacheStore, err := store.NewStore(".briefly-cache")
	if err != nil {
		// If we can't open the store, continue without it
		cacheStore = nil
	}

	// Initialize LLM client
	llmClient, err := llm.NewClient("")
	if err != nil {
		llmClient = nil
	}

	// Load current team context from config
	teamContext := config.GetTeamContext()

	return model{
		store:       cacheStore,
		llmClient:   llmClient,
		mode:        viewMainMenu,
		selectedIdx: 0,
		
		// Initialize menu items
		menuItems: []string{
			"🎯 Configure Team Context",
			"📝 Generate Digest (Interactive)",
			"📊 Review Articles",
			"🔍 Tune Relevance Filtering", 
			"📚 View Digest History",
			"❌ Exit",
		},
		
		// Load team context
		teamContext:     teamContext,
		insights:        make(map[string]string),
		relevanceScores: make(map[string]float64),
		
		// UI state
		showHelp: true,
	}
}

// Init is the first command that will be run.
func (m model) Init() tea.Cmd {
	return nil
}

// Message types for our enhanced TUI
type digestsLoadedMsg struct {
	digests []core.Digest
}

type pipelineStepMsg struct {
	step   pipelineStep
	status string
	data   interface{}
}

type articlesProcessedMsg struct {
	digestItems []render.DigestData
	articles    []core.Article
}

type insightsGeneratedMsg struct {
	insights map[string]string
}

type relevanceScoresMsg struct {
	scores map[string]float64
}

type errorMsg struct {
	err error
}

type statusMsg struct {
	message string
}

// Commands for async operations
func loadDigests(store *store.Store) tea.Cmd {
	return func() tea.Msg {
		if store == nil {
			return digestsLoadedMsg{digests: []core.Digest{}}
		}

		digests, err := store.GetLatestDigests(10)
		if err != nil {
			return errorMsg{err: err}
		}

		return digestsLoadedMsg{digests: digests}
	}
}

// Update handles messages and updates the model accordingly.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case digestsLoadedMsg:
		m.digests = msg.digests

	case pipelineStepMsg:
		m.currentStep = msg.step
		m.pipelineStatus = msg.status
		if msg.data != nil {
			switch data := msg.data.(type) {
			case []render.DigestData:
				m.digestItems = data
			case []core.Article:
				m.articles = data
			}
		}

	case articlesProcessedMsg:
		m.digestItems = msg.digestItems
		m.articles = msg.articles

	case insightsGeneratedMsg:
		m.insights = msg.insights

	case relevanceScoresMsg:
		m.relevanceScores = msg.scores

	case errorMsg:
		m.errorMessage = msg.err.Error()

	case statusMsg:
		m.statusMessage = msg.message

	case tea.KeyMsg:
		// Global key bindings
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
		}

		// Mode-specific key bindings
		switch m.mode {
		case viewMainMenu:
			cmd = m.updateMainMenu(msg)
		case viewTeamContextSetup:
			cmd = m.updateTeamContextSetup(msg)
		case viewDigestPipeline:
			cmd = m.updateDigestPipeline(msg)
		case viewArticleReview:
			cmd = m.updateArticleReview(msg)
		case viewRelevanceTuning:
			cmd = m.updateRelevanceTuning(msg)
		case viewDigestHistory:
			cmd = m.updateDigestHistory(msg)
		case viewEditMyTake:
			cmd = m.updateEditMyTake(msg)
		}
	}

	return m, cmd
}

// Menu navigation helpers
func (m *model) updateMainMenu(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case "down", "j":
		if m.selectedIdx < len(m.menuItems)-1 {
			m.selectedIdx++
		}
	case "enter", " ":
		return m.handleMenuSelection()
	case "q", "esc":
		m.quitting = true
		return tea.Quit
	}
	return nil
}

func (m *model) handleMenuSelection() tea.Cmd {
	switch m.selectedIdx {
	case 0: // Configure Team Context
		m.mode = viewTeamContextSetup
		m.selectedIdx = 0
	case 1: // Generate Digest
		m.mode = viewDigestPipeline
		m.currentStep = stepFetching
		m.selectedIdx = 0
		// TODO: Prompt for input file
	case 2: // Review Articles
		m.mode = viewArticleReview
		m.selectedIdx = 0
		return loadDigests(m.store)
	case 3: // Tune Relevance
		m.mode = viewRelevanceTuning
		m.selectedIdx = 0
	case 4: // View History
		m.mode = viewDigestHistory
		m.selectedIdx = 0
		return loadDigests(m.store)
	case 5: // Exit
		m.quitting = true
		return tea.Quit
	}
	return nil
}

// Additional update functions for each view mode
func (m *model) updateTeamContextSetup(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = viewMainMenu
		m.selectedIdx = 0
	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case "down", "j":
		// Navigate through team context fields
		maxFields := 6 // Tech stack, challenges, interests, product type, priority, save
		if m.selectedIdx < maxFields {
			m.selectedIdx++
		}
	case "enter", " ":
		// Edit field or save
		return m.handleTeamContextEdit()
	}
	return nil
}

func (m *model) updateDigestPipeline(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = viewMainMenu
		m.selectedIdx = 0
	case "enter", " ":
		// Progress pipeline or complete
		return m.progressPipeline()
	}
	return nil
}

func (m *model) updateArticleReview(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = viewMainMenu
		m.selectedIdx = 0
	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case "down", "j":
		if m.selectedIdx < len(m.digestItems)-1 {
			m.selectedIdx++
		}
	}
	return nil
}

func (m *model) updateRelevanceTuning(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = viewMainMenu
		m.selectedIdx = 0
	}
	return nil
}

func (m *model) updateDigestHistory(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = viewMainMenu
		m.selectedIdx = 0
	case "up", "k":
		if m.selectedIdx > 0 {
			m.selectedIdx--
		}
	case "down", "j":
		if m.selectedIdx < len(m.digests)-1 {
			m.selectedIdx++
		}
	case "enter", "e":
		// Edit my-take for selected digest
		if len(m.digests) > 0 && m.selectedIdx < len(m.digests) {
			m.mode = viewEditMyTake
			m.editingDigestID = m.digests[m.selectedIdx].ID
			m.editingMyTake = m.digests[m.selectedIdx].MyTake
		}
	}
	return nil
}

func (m *model) updateEditMyTake(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		// Cancel editing
		m.mode = viewDigestHistory
		m.editingMyTake = ""
		m.editingDigestID = ""
	case "ctrl+s":
		// Save my-take and regenerate with LLM
		return m.saveAndRegenerateMyTake()
	case "backspace":
		if len(m.editingMyTake) > 0 {
			m.editingMyTake = m.editingMyTake[:len(m.editingMyTake)-1]
		}
	default:
		// Add character to my-take
		if len(msg.String()) == 1 {
			m.editingMyTake += msg.String()
		}
	}
	return nil
}

// Helper functions for complex operations
func (m *model) handleTeamContextEdit() tea.Cmd {
	switch m.selectedIdx {
	case 0: // Tech Stack
		m.editingField = "tech_stack"
		m.editingValue = strings.Join(m.teamContext.TechStack, ", ")
		return m.startFieldEdit()
	case 1: // Current Challenges  
		m.editingField = "current_challenges"
		m.editingValue = strings.Join(m.teamContext.CurrentChallenges, ", ")
		return m.startFieldEdit()
	case 2: // Interests
		m.editingField = "interests"
		m.editingValue = strings.Join(m.teamContext.Interests, ", ")
		return m.startFieldEdit()
	case 3: // Product Type
		m.editingField = "product_type"
		m.editingValue = m.teamContext.ProductType
		return m.startFieldEdit()
	case 4: // Save Configuration
		return m.saveTeamContext()
	}
	return nil
}

func (m *model) startFieldEdit() tea.Cmd {
	// In a real implementation, this would switch to an input mode
	// For now, we'll simulate the edit by toggling some state
	m.statusMessage = fmt.Sprintf("Editing %s: %s", m.editingField, m.editingValue)
	return nil
}

func (m *model) saveTeamContext() tea.Cmd {
	// Save the team context to config override
	err := config.SaveTeamContextOverride(m.teamContext)
	if err != nil {
		m.errorMessage = fmt.Sprintf("Failed to save team context: %v", err)
	} else {
		m.statusMessage = "Team context saved successfully!"
	}
	return nil
}

func (m *model) progressPipeline() tea.Cmd {
	// TODO: Implement pipeline progression
	return nil
}

func (m *model) saveAndRegenerateMyTake() tea.Cmd {
	// TODO: Implement LLM-based my-take regeneration
	if m.store != nil {
		err := m.store.UpdateDigestMyTake(m.editingDigestID, m.editingMyTake)
		if err == nil {
			// Update the digest in our local list
			for i := range m.digests {
				if m.digests[i].ID == m.editingDigestID {
					m.digests[i].MyTake = m.editingMyTake
					break
				}
			}
		}
	}
	m.mode = viewDigestHistory
	m.editingMyTake = ""
	m.editingDigestID = ""
	return nil
}

// View renders the TUI.
func (m model) View() string {
	if m.quitting {
		return "Thanks for using Briefly! 👋\n"
	}

	// Define styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("105")).
		Padding(0, 1)

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("99")).
		BorderStyle(lipgloss.NormalBorder()).
		BorderBottom(true).
		Padding(0, 1)

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		Background(lipgloss.Color("57"))

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("71")).
		Italic(true)

	var content strings.Builder
	
	// Header
	content.WriteString(titleStyle.Render("📚 Briefly - Interactive Digest Generator"))
	content.WriteString("\n\n")

	// Error messages
	if m.errorMessage != "" {
		content.WriteString(errorStyle.Render("❌ Error: " + m.errorMessage))
		content.WriteString("\n\n")
	}

	// Status messages
	if m.statusMessage != "" {
		content.WriteString(statusStyle.Render("ℹ️  " + m.statusMessage))
		content.WriteString("\n\n")
	}

	// Mode-specific content
	switch m.mode {
	case viewMainMenu:
		content.WriteString(m.renderMainMenu(headerStyle, selectedStyle, normalStyle))
	case viewTeamContextSetup:
		content.WriteString(m.renderTeamContextSetup(headerStyle, selectedStyle, normalStyle))
	case viewDigestPipeline:
		content.WriteString(m.renderDigestPipeline(headerStyle, selectedStyle, normalStyle))
	case viewArticleReview:
		content.WriteString(m.renderArticleReview(headerStyle, selectedStyle, normalStyle))
	case viewRelevanceTuning:
		content.WriteString(m.renderRelevanceTuning(headerStyle, selectedStyle, normalStyle))
	case viewDigestHistory:
		content.WriteString(m.renderDigestHistory(headerStyle, selectedStyle, normalStyle))
	case viewEditMyTake:
		content.WriteString(m.renderEditMyTake(headerStyle, selectedStyle, normalStyle))
	}

	// Help section
	if m.showHelp {
		content.WriteString("\n")
		content.WriteString(normalStyle.Render("💡 Press [?] to toggle help • [Ctrl+C] to quit • [Esc] to go back"))
	}

	return content.String()
}

// Render methods for each view mode
func (m model) renderMainMenu(headerStyle, selectedStyle, normalStyle lipgloss.Style) string {
	var content strings.Builder
	
	content.WriteString(headerStyle.Render("📋 Main Menu"))
	content.WriteString("\n\n")
	
	for i, item := range m.menuItems {
		if i == m.selectedIdx {
			content.WriteString(selectedStyle.Render("  " + item))
		} else {
			content.WriteString(normalStyle.Render("  " + item))
		}
		content.WriteString("\n")
	}
	
	return content.String()
}

func (m model) renderTeamContextSetup(headerStyle, selectedStyle, normalStyle lipgloss.Style) string {
	var content strings.Builder
	
	content.WriteString(headerStyle.Render("🎯 Team Context Setup"))
	content.WriteString("\n\n")
	
	content.WriteString(normalStyle.Render("Configure your team context for personalized digests:\n\n"))
	
	fields := []string{
		fmt.Sprintf("Tech Stack: %v", m.teamContext.TechStack),
		fmt.Sprintf("Current Challenges: %v", m.teamContext.CurrentChallenges),
		fmt.Sprintf("Interests: %v", m.teamContext.Interests),
		fmt.Sprintf("Product Type: %s", m.teamContext.ProductType),
		"Save Configuration",
	}
	
	for i, field := range fields {
		if i == m.selectedIdx {
			content.WriteString(selectedStyle.Render("  " + field))
		} else {
			content.WriteString(normalStyle.Render("  " + field))
		}
		content.WriteString("\n")
	}
	
	return content.String()
}

func (m model) renderDigestPipeline(headerStyle, selectedStyle, normalStyle lipgloss.Style) string {
	var content strings.Builder
	
	content.WriteString(headerStyle.Render("📝 Digest Pipeline"))
	content.WriteString("\n\n")
	
	// Pipeline progress
	steps := []string{
		"📥 Fetching Articles",
		"📝 Summarizing Content", 
		"🧮 Clustering Topics",
		"🧠 Generating Insights",
		"🔍 Filtering Relevance",
		"📊 Creating Digest",
	}
	
	for i, step := range steps {
		if pipelineStep(i) == m.currentStep {
			content.WriteString(selectedStyle.Render("  ➤ " + step))
		} else if pipelineStep(i) < m.currentStep {
			content.WriteString(normalStyle.Render("  ✅ " + step))
		} else {
			content.WriteString(normalStyle.Render("  ⏳ " + step))
		}
		content.WriteString("\n")
	}
	
	if m.pipelineStatus != "" {
		content.WriteString("\n")
		content.WriteString(normalStyle.Render("Status: " + m.pipelineStatus))
	}
	
	return content.String()
}

func (m model) renderArticleReview(headerStyle, selectedStyle, normalStyle lipgloss.Style) string {
	var content strings.Builder
	
	content.WriteString(headerStyle.Render("📊 Article Review"))
	content.WriteString("\n\n")
	
	if len(m.digestItems) == 0 {
		content.WriteString(normalStyle.Render("No articles to review yet. Generate a digest first."))
		return content.String()
	}
	
	content.WriteString(normalStyle.Render(fmt.Sprintf("Reviewing %d articles:\n\n", len(m.digestItems))))
	
	for i, item := range m.digestItems {
		if i == m.selectedIdx {
			content.WriteString(selectedStyle.Render(fmt.Sprintf("  ➤ %s", item.Title)))
		} else {
			content.WriteString(normalStyle.Render(fmt.Sprintf("    %s", item.Title)))
		}
		content.WriteString("\n")
	}
	
	return content.String()
}

func (m model) renderRelevanceTuning(headerStyle, selectedStyle, normalStyle lipgloss.Style) string {
	var content strings.Builder
	
	content.WriteString(headerStyle.Render("🔍 Relevance Tuning"))
	content.WriteString("\n\n")
	
	content.WriteString(normalStyle.Render("Adjust relevance filtering settings:\n\n"))
	content.WriteString(normalStyle.Render("• Minimum relevance score\n"))
	content.WriteString(normalStyle.Render("• Team context weight\n"))
	content.WriteString(normalStyle.Render("• Content filtering rules\n"))
	
	return content.String()
}

func (m model) renderDigestHistory(headerStyle, selectedStyle, normalStyle lipgloss.Style) string {
	var content strings.Builder
	
	content.WriteString(headerStyle.Render("📚 Digest History"))
	content.WriteString("\n\n")
	
	if len(m.digests) == 0 {
		content.WriteString(normalStyle.Render("No previous digests found."))
		return content.String()
	}
	
	for i, digest := range m.digests {
		if i == m.selectedIdx {
			content.WriteString(selectedStyle.Render(fmt.Sprintf("  ➤ %s (%s)", digest.Title, digest.Format)))
		} else {
			content.WriteString(normalStyle.Render(fmt.Sprintf("    %s (%s)", digest.Title, digest.Format)))
		}
		content.WriteString("\n")
	}
	
	content.WriteString("\n")
	content.WriteString(normalStyle.Render("Press [e] to edit my-take for selected digest"))
	
	return content.String()
}

func (m model) renderEditMyTake(headerStyle, selectedStyle, normalStyle lipgloss.Style) string {
	var content strings.Builder
	
	content.WriteString(headerStyle.Render("✏️ Edit My Take"))
	content.WriteString("\n\n")
	
	content.WriteString(normalStyle.Render("Editing my-take for digest: " + m.editingDigestID))
	content.WriteString("\n\n")
	
	content.WriteString(normalStyle.Render("Current take:"))
	content.WriteString("\n")
	content.WriteString(selectedStyle.Render(m.editingMyTake))
	content.WriteString("\n\n")
	
	content.WriteString(normalStyle.Render("Press [Ctrl+S] to save and regenerate with LLM"))
	
	return content.String()
}

// StartTUI initializes and starts the Bubble Tea application.
func StartTUI() {
	p := tea.NewProgram(InitialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
