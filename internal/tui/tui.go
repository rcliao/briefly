package tui

import (
	"briefly/internal/core"
	"briefly/internal/store"
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type viewMode int

const (
	viewDigests viewMode = iota
	viewEditMyTake
)

// Model represents the state of the TUI application.
type model struct {
	store           *store.Store
	digests         []core.Digest  // Recent digests
	articles        []core.Article // Placeholder for articles
	summaries       []core.Summary // Placeholder for summaries
	selectedIdx     int            // Index of the selected item
	width           int            // Terminal width
	height          int            // Terminal height
	mode            viewMode       // Current view mode
	editingMyTake   string         // Text being edited for my-take
	editingDigestID string         // ID of digest being edited
	quitting        bool
}

// InitialModel returns the initial state of the TUI model.
func InitialModel() model {
	cacheStore, err := store.NewStore(".briefly-cache")
	if err != nil {
		// If we can't open the store, continue without it
		cacheStore = nil
	}

	return model{
		store:       cacheStore,
		articles:    []core.Article{},
		summaries:   []core.Summary{},
		digests:     []core.Digest{},
		selectedIdx: 0,
		mode:        viewDigests,
	}
}

// Init is the first command that will be run. Load digests.
func (m model) Init() tea.Cmd {
	return loadDigests(m.store)
}

// loadDigests command to load recent digests
func loadDigests(store *store.Store) tea.Cmd {
	return func() tea.Msg {
		if store == nil {
			return digestsLoadedMsg{digests: []core.Digest{}}
		}

		digests, err := store.GetLatestDigests(10)
		if err != nil {
			return digestsLoadedMsg{digests: []core.Digest{}}
		}

		return digestsLoadedMsg{digests: digests}
	}
}

// Message types
type digestsLoadedMsg struct {
	digests []core.Digest
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

	case tea.KeyMsg:
		switch m.mode {
		case viewDigests:
			switch msg.String() {
			case "ctrl+c", "q":
				m.quitting = true
				return m, tea.Quit
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
			case "r":
				// Refresh digests
				return m, loadDigests(m.store)
			}

		case viewEditMyTake:
			switch msg.String() {
			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			case "esc":
				// Cancel editing
				m.mode = viewDigests
				m.editingMyTake = ""
				m.editingDigestID = ""
			case "ctrl+s":
				// Save my-take
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
				m.mode = viewDigests
				m.editingMyTake = ""
				m.editingDigestID = ""
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
	listStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Padding(1).Width(m.width/2 - 5)
	detailStyle := lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true).Padding(1).Width(m.width/2 - 5)

	switch m.mode {
	case viewDigests:
		// Digest list content
		digestListContent := "Recent Digests\n\n"
		if len(m.digests) == 0 {
			digestListContent += "No digests found.\nGenerate some digests first!"
		} else {
			for i, digest := range m.digests {
				cursor := " "
				if i == m.selectedIdx {
					cursor = ">"
				}

				myTakeStatus := "❌"
				if digest.MyTake != "" {
					myTakeStatus = "✅"
				}

				digestListContent += fmt.Sprintf("%s %s %s (%s)\n", cursor, myTakeStatus, digest.ID[:8], digest.Format)
				digestListContent += fmt.Sprintf("   %s\n", digest.DateGenerated.Format("2006-01-02 15:04"))
				if digest.Title != "" {
					digestListContent += fmt.Sprintf("   %s\n", digest.Title)
				}
				digestListContent += "\n"
			}
		}

		// Digest detail content
		digestDetailContent := "Digest Details\n\n"
		if len(m.digests) == 0 || m.selectedIdx >= len(m.digests) {
			digestDetailContent += "No digest selected."
		} else {
			selectedDigest := m.digests[m.selectedIdx]
			digestDetailContent += fmt.Sprintf("ID: %s\n", selectedDigest.ID)
			digestDetailContent += fmt.Sprintf("Format: %s\n", selectedDigest.Format)
			digestDetailContent += fmt.Sprintf("Generated: %s\n\n", selectedDigest.DateGenerated.Format("2006-01-02 15:04"))

			if selectedDigest.DigestSummary != "" {
				digestDetailContent += "Summary:\n"
				// Truncate long summaries
				summary := selectedDigest.DigestSummary
				if len(summary) > 300 {
					summary = summary[:300] + "..."
				}
				digestDetailContent += summary + "\n\n"
			}

			if selectedDigest.MyTake != "" {
				digestDetailContent += "My Take:\n"
				digestDetailContent += selectedDigest.MyTake + "\n"
			} else {
				digestDetailContent += "No personal take added yet.\nPress [Enter] to add one!"
			}
		}

		// Layout
		leftPane := listStyle.Render(digestListContent)
		rightPane := detailStyle.Render(digestDetailContent)
		mainContent := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
		help := "\n\n[↑/k] Up | [↓/j] Down | [Enter/e] Edit My-Take | [r] Refresh | [q] Quit"
		return docStyle.Render(mainContent + help)

	case viewEditMyTake:
		// Edit my-take view
		editContent := "Edit My Take\n\n"
		if m.editingDigestID != "" {
			// Find the digest being edited
			var digest *core.Digest
			for _, d := range m.digests {
				if d.ID == m.editingDigestID {
					digest = &d
					break
				}
			}

			if digest != nil {
				editContent += fmt.Sprintf("Digest: %s (%s)\n", digest.ID[:8], digest.Format)
				editContent += fmt.Sprintf("Generated: %s\n\n", digest.DateGenerated.Format("2006-01-02 15:04"))
			}
		}

		editContent += "Your Take:\n"
		editContent += strings.Repeat("-", 40) + "\n"
		editContent += m.editingMyTake
		editContent += "\n" + strings.Repeat("-", 40) + "\n"

		editBox := listStyle.Width(m.width - 10).Render(editContent)
		help := "\n\n[Ctrl+S] Save | [Esc] Cancel | [Ctrl+C] Quit"
		return docStyle.Render(editBox + help)

	default:
		return "Unknown view mode"
	}
}

// StartTUI initializes and starts the Bubble Tea application.
func StartTUI() {
	p := tea.NewProgram(InitialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
