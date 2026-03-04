package views

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/KidiXDev/civ-cli/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type SettingsView struct {
	router AppRouter
	cursor int
	items  []string

	editing bool
	input   string
}

func NewSettingsView(router AppRouter) *SettingsView {
	return &SettingsView{
		router: router,
		items: []string{
			"API Key",
			"Default Search Limit",
			"Default Download Directory",
			"JSON Format",
			"Timeout (seconds)",
			"Retry Count",
			"Back",
		},
	}
}

func (m *SettingsView) Init() tea.Cmd {
	return nil
}

func (m *SettingsView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.editing {
			switch msg.Type {
			case tea.KeyEsc:
				m.editing = false
				m.input = ""
			case tea.KeyEnter:
				m.saveSetting(m.items[m.cursor], m.input)
				m.editing = false
				m.input = ""
			case tea.KeyBackspace, tea.KeyDelete:
				if len(m.input) > 0 {
					m.input = m.input[:len(m.input)-1]
				}
			case tea.KeyRunes:
				m.input += msg.String()
			case tea.KeySpace:
				m.input += " "
			}
			return m, nil
		}

		// Navigation
		switch msg.String() {
		case "esc", "q":
			m.router.Pop()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", " ":
			selected := m.items[m.cursor]
			if selected == "Back" {
				m.router.Pop()
			} else {
				m.editing = true
				m.input = ""
			}
		}
	}
	return m, nil
}

func (m *SettingsView) saveSetting(key, val string) {
	cfg := m.router.GetConfig()
	val = strings.ReplaceAll(val, "\x00", "")

	switch key {
	case "API Key":
		cfg.APIKey = strings.TrimSpace(val)
	case "Default Search Limit":
		if v, err := strconv.Atoi(val); err == nil {
			cfg.DefaultSearchLimit = v
		}
	case "Default Download Directory":
		cfg.DefaultDownloadDir = val
	case "JSON Format":
		cfg.OutputFormat = val
	case "Timeout (seconds)":
		if v, err := strconv.Atoi(val); err == nil {
			cfg.TimeoutSeconds = v
		}
	case "Retry Count":
		if v, err := strconv.Atoi(val); err == nil {
			cfg.RetryCount = v
		}
	}
	// Try save
	_ = m.router.GetConfigManager().Save(cfg)
}

func (m *SettingsView) View() string {
	b := strings.Builder{}
	b.WriteString(ui.Title("Settings"))
	b.WriteString("\n\n")

	cfg := m.router.GetConfig()

	for i, item := range m.items {
		cursor := "  "
		if m.cursor == i {
			cursor = ui.Info("> ")
		}

		val := ""
		switch item {
		case "API Key":
			val = cfg.APIKey
		case "Default Search Limit":
			val = strconv.Itoa(cfg.DefaultSearchLimit)
		case "Default Download Directory":
			val = cfg.DefaultDownloadDir
		case "JSON Format":
			val = cfg.OutputFormat
		case "Timeout (seconds)":
			val = strconv.Itoa(cfg.TimeoutSeconds)
		case "Retry Count":
			val = strconv.Itoa(cfg.RetryCount)
		}

		if m.editing && m.cursor == i {
			b.WriteString(fmt.Sprintf("%s%s: [%s█]\n", cursor, ui.Warning(item), m.input))
		} else {
			if item != "Back" {
				b.WriteString(fmt.Sprintf("%s%s = %s\n", cursor, item, ui.Sub(val)))
			} else {
				b.WriteString(fmt.Sprintf("%s%s\n", cursor, item))
			}
		}
	}

	b.WriteString("\n")
	if m.editing {
		b.WriteString(ui.Sub("Type new value and press [Enter] to save, [Esc] to cancel."))
	} else {
		b.WriteString(ui.Sub("[Enter] Edit  [Esc] Back"))
	}

	return fmt.Sprintf("\n  %s\n", strings.ReplaceAll(b.String(), "\n", "\n  "))
}
