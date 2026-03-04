package views

import (
	"fmt"
	"strings"

	"github.com/KidiXDev/civ-cli/internal/civitai"
	"github.com/KidiXDev/civ-cli/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
)

// Lines reserved for chrome (title, header info, footer hints, padding).
const modelViewChrome = 10

type ModelView struct {
	router       AppRouter
	model        civitai.Model
	cursor       int
	scrollOffset int
	height       int // terminal height from WindowSizeMsg
}

func NewModelView(router AppRouter, model civitai.Model) *ModelView {
	return &ModelView{
		router: router,
		model:  model,
		height: 24, // sensible default before first WindowSizeMsg
	}
}

func (m *ModelView) Init() tea.Cmd {
	return nil
}

// maxVisible returns how many version items fit in the current terminal.
// Each version occupies 2 lines (name + file details).
func (m *ModelView) maxVisible() int {
	usable := m.height - modelViewChrome
	if usable < 2 {
		usable = 2
	}
	return usable / 2 // 2 lines per version entry
}

// ensureCursorVisible adjusts the scroll offset so the cursor is in view.
func (m *ModelView) ensureCursorVisible() {
	visible := m.maxVisible()
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+visible {
		m.scrollOffset = m.cursor - visible + 1
	}
	// Clamp
	maxOffset := len(m.model.ModelVersions) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m *ModelView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.ensureCursorVisible()

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.router.Pop()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.ensureCursorVisible()
			}
		case "down", "j":
			if m.cursor < len(m.model.ModelVersions)-1 {
				m.cursor++
				m.ensureCursorVisible()
			}
		case "home", "g":
			m.cursor = 0
			m.ensureCursorVisible()
		case "end", "G":
			m.cursor = len(m.model.ModelVersions) - 1
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureCursorVisible()
		case "pgup":
			m.cursor -= m.maxVisible()
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureCursorVisible()
		case "pgdown":
			m.cursor += m.maxVisible()
			if m.cursor >= len(m.model.ModelVersions) {
				m.cursor = len(m.model.ModelVersions) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureCursorVisible()
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

	total := len(m.model.ModelVersions)
	b.WriteString(ui.Info(fmt.Sprintf("Versions available for download: (%d total)\n", total)))

	if total == 0 {
		b.WriteString(ui.Warning("  No versions available.\n"))
	} else {
		visible := m.maxVisible()
		endIdx := m.scrollOffset + visible
		if endIdx > total {
			endIdx = total
		}

		// Scroll-up indicator
		if m.scrollOffset > 0 {
			b.WriteString(ui.Sub(fmt.Sprintf("    ▲ %d more above\n", m.scrollOffset)))
		}

		for i := m.scrollOffset; i < endIdx; i++ {
			ver := m.model.ModelVersions[i]
			cursor := "  "
			if m.cursor == i {
				cursor = ui.Info("> ")
				b.WriteString(fmt.Sprintf("%s%s\n", cursor, ui.Info(ver.Name)))
			} else {
				b.WriteString(fmt.Sprintf("%s%s\n", cursor, ver.Name))
			}

			// File details
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

		// Scroll-down indicator
		remaining := total - endIdx
		if remaining > 0 {
			b.WriteString(ui.Sub(fmt.Sprintf("    ▼ %d more below\n", remaining)))
		}
	}

	b.WriteString("\n")
	b.WriteString(ui.Sub("[↑/↓] Select  [PgUp/PgDn] Page  [Home/End] Jump  [Enter] Download  [Esc] Back"))

	return fmt.Sprintf("\n  %s\n", strings.ReplaceAll(b.String(), "\n", "\n  "))
}
