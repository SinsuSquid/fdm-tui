package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// 1. Define the application state (Model)
type model struct {
	wikiName string
	loading  bool
	choice   int
}

func initialModel() model {
	return model{
		wikiName: "Genshin Impact / Elden Ring / Anime",
		loading:  false,
		choice:   0,
	}
}

// 2. Initialize the model (Cmds to run on start)
func (m model) Init() tea.Cmd {
	return nil // No side-effects right now!
}

// 3. Handle inputs and updates (Update)
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		// Panic buttons / Quit keys! 🤫
		case "ctrl+c", "q", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

// 4. Render the UI to the terminal (View)
func (m model) View() string {
	// Style our title beautifully using lipgloss!
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#A855F7")). // Sleek Purple
		Background(lipgloss.Color("#1E1E2E")).
		Padding(0, 1)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6C7086")).
		Italic(true)

	s := titleStyle.Render("⚡ fdm-tui v0.1.0 (Stealth Mode Active)") + "\n\n"
	s += fmt.Sprintf("Target Wiki: %s\n\n", m.wikiName)
	s += statusStyle.Render("[Press 'q' or 'Esc' to close undetected]")

	return s
}

func main() {
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Alas, an error occurred: %v", err)
		os.Exit(1)
	}
}
