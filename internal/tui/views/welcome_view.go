package views

import (
	"fmt"
	"strings"

	"github.com/KidiXDev/civ-cli/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type WelcomeView struct {
	router AppRouter
	input  string
	err    string
}

func NewWelcomeView(router AppRouter) *WelcomeView {
	return &WelcomeView{
		router: router,
	}
}

func (m *WelcomeView) Init() tea.Cmd {
	return nil
}

func (m *WelcomeView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.router.Quit()
			return m, tea.Quit
		case tea.KeyEnter:
			apiKey := strings.TrimSpace(m.input)
			if apiKey == "" {
				m.err = "API Key cannot be empty."
				return m, nil
			}
			err := m.router.SaveConfigAndProceed(apiKey)
			if err != nil {
				m.err = "Failed to save: " + err.Error()
			}
			// Don't return Quit cmd when pushing new view
			return m, nil
		case tea.KeyBackspace, tea.KeyDelete:
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		case tea.KeyRunes:
			m.input += msg.String()
		case tea.KeySpace:
			m.input += " "
		}
	}
	return m, nil
}

func (m *WelcomeView) View() string {
	var b strings.Builder

	b.WriteString(ui.Title("Welcome to Civitool!"))
	b.WriteString("\n\n")
	b.WriteString("It looks like this is your first time using Civitool.\n")
	b.WriteString("To get started, please enter your Civitai API Key.\n")
	b.WriteString("You can generate one from your User Account Settings on civitai.com.\n\n")

	b.WriteString("API Key: ")
	if m.input == "" {
		b.WriteString(ui.Sub("(type here...)"))
	} else {
		// Mask the input visually for security, or show it? We'll show partially padded or unmasked for simplicity.
		// A real implementation might mask it. We'll leave it visible for now.
		b.WriteString(m.input + "█")
	}

	b.WriteString("\n\n")
	if m.err != "" {
		b.WriteString(ui.Error(m.err) + "\n\n")
	}

	b.WriteString(ui.Sub("[Enter] Submit  [Esc] Quit"))

	// Padding
	return fmt.Sprintf("\n  %s\n", strings.ReplaceAll(b.String(), "\n", "\n  "))
}
