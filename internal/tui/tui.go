package tui

import (
	"briefly/internal/core"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the state of the TUI application.
// For now, it's simple, but will grow to include lists of articles, summaries, etc.
type model struct {
	articles    []core.Article // Placeholder for articles
	summaries   []core.Summary   // Placeholder for summaries
	selectedIdx int            // Index of the selected article/summary
	width       int            // Terminal width
	height      int            // Terminal height
	quitting    bool
}

// InitialModel returns the initial state of the TUI model.
func InitialModel() model {
	return model{
		// Initialize with some dummy data or leave empty
		articles:    []core.Article{},
		summaries:   []core.Summary{},
		selectedIdx: 0,
	}
}

// Init is the first command that will be run. We don't need any for now.
func (m model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model accordingly.
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.selectedIdx > 0 {
				m.selectedIdx--
			}
		case "down", "j":
			// Assuming articles and summaries are linked 1-to-1 for now
			if m.selectedIdx < len(m.articles)-1 { // or len(m.summaries) -1
				m.selectedIdx++
			}
		}
	}

	return m, cmd
}

// View renders the TUI.
func (m model) View() string {
	if m.quitting {
		return "Quitting...\n"
	}

	// Basic styles
	docStyle := lipgloss.NewStyle().Margin(1, 2)
	listStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Padding(1).Width(m.width/2 - 5) // Dynamic width
	detailStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Padding(1).Width(m.width/2 - 5) // Dynamic width

	// Placeholder content
	articleListContent := "Article List (Placeholder)\n\n"
	if len(m.articles) == 0 {
		articleListContent += "No articles loaded."
	} else {
		for i, article := range m.articles {
			cursor := " "
			if i == m.selectedIdx {
				cursor = ">"
			}
			articleListContent += fmt.Sprintf("%s %s\n", cursor, article.Title)
		}
	}

	summaryViewContent := "Summary View (Placeholder)\n\n"
	if len(m.summaries) == 0 || m.selectedIdx >= len(m.summaries) {
		summaryViewContent += "No summary to display or selection out of bounds."
	} else {
		summaryViewContent += m.summaries[m.selectedIdx].SummaryText
	}

	// Layout
	leftPane := listStyle.Render(articleListContent)
	rightPane := detailStyle.Render(summaryViewContent)

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)

	help := "\n\n[↑/k] Up | [↓/j] Down | [q] Quit"

	return docStyle.Render(mainContent + help)
}

// StartTUI initializes and starts the Bubble Tea application.
func StartTUI() {
	p := tea.NewProgram(InitialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
