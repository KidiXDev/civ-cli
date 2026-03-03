package views

import (
	"fmt"
	"strings"

	"github.com/KidiXDev/civ-cli/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type HomeView struct {
	router AppRouter
	cursor int
	menu   []string
}

func NewHomeView(router AppRouter) *HomeView {
	return &HomeView{
		router: router,
		menu: []string{
			"Search Models",
			"Settings",
			"Exit",
		},
	}
}

func (m *HomeView) Init() tea.Cmd {
	return nil
}

func (m *HomeView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.router.Quit()
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.menu)-1 {
				m.cursor++
			}
		case "enter", " ":
			return m.handleSelection()
		}
	}
	return m, nil
}

func (m *HomeView) handleSelection() (tea.Model, tea.Cmd) {
	selected := m.menu[m.cursor]
	switch selected {
	case "Search Models":
		m.router.Push(NewSearchView(m.router))
	case "Settings":
		m.router.Push(NewSettingsView(m.router))
	case "Exit":
		m.router.Quit()
		return m, tea.Quit
	}
	return m, nil
}

func (m *HomeView) View() string {
	b := strings.Builder{}

	b.WriteString(ui.Title("Civitool Main Menu"))
	b.WriteString("\n\n")

	for i, item := range m.menu {
		cursor := "  " // no cursor
		if m.cursor == i {
			cursor = ui.Info("> ")
			item = ui.Info(item)
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, item))
	}

	b.WriteString("\n")
	b.WriteString(ui.Sub("Use arrow keys to navigate, [Enter] to select. [q] to quit."))

	return fmt.Sprintf("\n  %s\n", strings.ReplaceAll(b.String(), "\n", "\n  "))
}
