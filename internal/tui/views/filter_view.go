package views

import (
	"fmt"
	"strings"

	"github.com/KidiXDev/civ-cli/internal/civitai"
	"github.com/KidiXDev/civ-cli/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type FilterViewState struct {
	Options civitai.SearchModelsOptions
	OnApply func(civitai.SearchModelsOptions) tea.Cmd
}

type FilterView struct {
	router AppRouter
	state  FilterViewState
	cursor int

	types  []string
	sorts  []string
	period []string

	// current index selections
	typeIdx   int
	sortIdx   int
	periodIdx int
}

func NewFilterView(router AppRouter, initialOptions civitai.SearchModelsOptions, onApply func(civitai.SearchModelsOptions) tea.Cmd) *FilterView {
	types := []string{"", "Checkpoint", "TextualInversion", "Hypernetwork", "AestheticGradient", "LORA", "Controlnet", "Poses"}
	sorts := []string{"", "Highest Rated", "Most Downloaded", "Newest"}
	period := []string{"", "AllTime", "Year", "Month", "Week", "Day"}

	typeIdx := 0
	sortIdx := 0
	periodIdx := 0

	// Set initial indices based on current options
	if len(initialOptions.Types) > 0 {
		for i, t := range types {
			if t == initialOptions.Types[0] {
				typeIdx = i
				break
			}
		}
	}
	for i, s := range sorts {
		if s == initialOptions.Sort {
			sortIdx = i
			break
		}
	}
	for i, p := range period {
		if p == initialOptions.Period {
			periodIdx = i
			break
		}
	}

	return &FilterView{
		router: router,
		state: FilterViewState{
			Options: initialOptions,
			OnApply: onApply,
		},
		types:     types,
		sorts:     sorts,
		period:    period,
		typeIdx:   typeIdx,
		sortIdx:   sortIdx,
		periodIdx: periodIdx,
		cursor:    0,
	}
}

func (m *FilterView) Init() tea.Cmd {
	return nil
}

func (m *FilterView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.router.Quit()
			return m, tea.Quit
		case "esc", "q":
			m.router.Pop()
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < 4 { // 5 options total (0-4)
				m.cursor++
			}
		case "left", "h":
			m.adjustOption(-1)
		case "right", "l":
			m.adjustOption(1)
		case "enter", " ":
			if m.cursor == 4 {
				// Toggle NSFW
				m.state.Options.NSFW = !m.state.Options.NSFW
			} else {
				// Apply filters
				m.applyFilters()
				m.router.Pop()
				var applyCmd tea.Cmd
				if m.state.OnApply != nil {
					applyCmd = m.state.OnApply(m.state.Options)
				}
				return m, applyCmd
			}
		}
	}
	return m, nil
}

func (m *FilterView) adjustOption(dir int) {
	switch m.cursor {
	case 0:
		m.typeIdx = m.cycle(m.typeIdx, dir, len(m.types))
	case 1:
		m.sortIdx = m.cycle(m.sortIdx, dir, len(m.sorts))
	case 2:
		m.periodIdx = m.cycle(m.periodIdx, dir, len(m.period))
	case 3:
		// Rating (not currently bounded strictly, but we can cycle 0-5)
		m.state.Options.Rating = m.cycle(m.state.Options.Rating, dir, 6)
	case 4:
		// NSFW is a boolean, handled by Enter
	}
}

func (m *FilterView) cycle(val, dir, max int) int {
	val += dir
	if val < 0 {
		return max - 1
	}
	if val >= max {
		return 0
	}
	return val
}

func (m *FilterView) applyFilters() {
	if m.typeIdx > 0 {
		m.state.Options.Types = []string{m.types[m.typeIdx]}
	} else {
		m.state.Options.Types = nil
	}
	m.state.Options.Sort = m.sorts[m.sortIdx]
	m.state.Options.Period = m.period[m.periodIdx]
	m.state.Options.Page = 1 // reset page on filter change
}

func (m *FilterView) renderOption(idx int, label string, value string) string {
	cursor := "  "
	if m.cursor == idx {
		cursor = ui.Info("> ")
	}
	return fmt.Sprintf("%s%-10s : %s\n", cursor, label, value)
}

func (m *FilterView) View() string {
	b := strings.Builder{}

	b.WriteString(ui.Title("Filter Models"))
	b.WriteString("\n\n")

	tVal := m.types[m.typeIdx]
	if tVal == "" {
		tVal = "Any"
	}
	b.WriteString(m.renderOption(0, "Type", tVal))

	sVal := m.sorts[m.sortIdx]
	if sVal == "" {
		sVal = "Default"
	}
	b.WriteString(m.renderOption(1, "Sort", sVal))

	pVal := m.period[m.periodIdx]
	if pVal == "" {
		pVal = "Default"
	}
	b.WriteString(m.renderOption(2, "Period", pVal))

	rVal := "Any"
	if m.state.Options.Rating > 0 {
		rVal = fmt.Sprintf("%d Stars+", m.state.Options.Rating)
	}
	b.WriteString(m.renderOption(3, "Rating", rVal))

	nsfwVal := "False"
	if m.state.Options.NSFW {
		nsfwVal = "True"
	}
	b.WriteString(m.renderOption(4, "NSFW", nsfwVal))

	b.WriteString("\n")
	b.WriteString(ui.Sub("[Up/Down] Navigate  [Left/Right] Change Value  [Enter] Apply (Toggle NSFW)  [Esc] Back"))

	return fmt.Sprintf("\n  %s\n", strings.ReplaceAll(b.String(), "\n", "\n  "))
}
