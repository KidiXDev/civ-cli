package views

import (
	"fmt"
	"strings"

	"github.com/KidiXDev/civ-cli/internal/civitai"
	"github.com/KidiXDev/civ-cli/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type ModelView struct {
	router AppRouter
	model  civitai.Model
	cursor int
}

func NewModelView(router AppRouter, model civitai.Model) *ModelView {
	return &ModelView{
		router: router,
		model:  model,
	}
}

func (m *ModelView) Init() tea.Cmd {
	return nil
}

func (m *ModelView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.router.Pop()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.model.ModelVersions)-1 {
				m.cursor++
			}
		case "enter", " ":
			if len(m.model.ModelVersions) > 0 {
				m.router.Push(NewDownloadView(m.router, m.model, m.model.ModelVersions[m.cursor]))
			}
		}
	}
	return m, nil
}

func (m *ModelView) View() string {
	b := strings.Builder{}

	b.WriteString(ui.Title(m.model.Name))
	b.WriteString("\n")
	b.WriteString(ui.Sub(fmt.Sprintf("Type: %s | Creator: %s | Rating: %.1f", m.model.Type, m.model.Creator.Username, m.model.Stats.Rating)))
	b.WriteString("\n\n")

	b.WriteString(ui.Info("Versions available for download:\n"))

	if len(m.model.ModelVersions) == 0 {
		b.WriteString(ui.Warning("  No versions available.\n"))
	} else {
		for i, ver := range m.model.ModelVersions {
			cursor := "  "
			if m.cursor == i {
				cursor = ui.Info("> ")
				b.WriteString(fmt.Sprintf("%s%s\n", cursor, ui.Info(ver.Name)))
			} else {
				b.WriteString(fmt.Sprintf("%s%s\n", cursor, ver.Name))
			}

			// Simple details
			if len(ver.Files) > 0 {
				primary := ver.Files[0]
				for _, f := range ver.Files {
					if f.Primary {
						primary = f
						break
					}
				}
				b.WriteString(fmt.Sprintf("    File: %s (%.2f MB)\n", primary.Name, primary.SizeKB/1024))
			}
		}
	}

	b.WriteString("\n")
	b.WriteString(ui.Sub("[Up/Down] Select Version  [Enter] Start Download  [Esc] Back"))

	return fmt.Sprintf("\n  %s\n", strings.ReplaceAll(b.String(), "\n", "\n  "))
}
