package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/KidiXDev/civ-cli/internal/civitai"
	"github.com/KidiXDev/civ-cli/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type SearchViewState int

const (
	StateTyping SearchViewState = iota
	StateLoading
	StateResults
	StateError
)

type SearchView struct {
	router  AppRouter
	state   SearchViewState
	query   string
	models  []civitai.Model
	cursor  int
	err     string
	page    int
	options civitai.SearchModelsOptions
}

type modelsFoundMsg struct {
	models []civitai.Model
}
type errorMsg struct {
	err error
}

func NewSearchView(router AppRouter) *SearchView {
	opts := civitai.SearchModelsOptions{
		Limit: router.GetConfig().DefaultSearchLimit,
		Page:  1,
	}
	return &SearchView{
		router:  router,
		state:   StateTyping,
		page:    1,
		options: opts,
	}
}

func (m *SearchView) Init() tea.Cmd {
	return nil
}

func (m *SearchView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case modelsFoundMsg:
		m.models = msg.models
		m.state = StateResults
		m.cursor = 0
		return m, nil

	case errorMsg:
		m.err = msg.err.Error()
		m.state = StateError
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.router.Quit()
			return m, tea.Quit
		}

		if m.state == StateTyping {
			switch msg.Type {
			case tea.KeyEsc:
				m.router.Pop()
				return m, nil
			case tea.KeyEnter:
				m.state = StateLoading
				m.options.Query = m.query
				m.page = 1
				m.options.Page = 1
				return m, m.searchCmd()
			case tea.KeyBackspace, tea.KeyDelete:
				if len(m.query) > 0 {
					m.query = m.query[:len(m.query)-1]
				}
			case tea.KeyRunes:
				if msg.String() == "f" && m.query == "" { // Only open filter if no query typed or user explicitly just pressed f with intention. Adjust to ctrl+f for better UX. We can just use ctrl+f
					// Do nothing here, we'll bind ctrl+f outside
				}
				m.query += msg.String()
			case tea.KeySpace:
				m.query += " "
			}

			// Add a specific keybind for filter trigger from main search
			if msg.String() == "ctrl+f" {
				m.router.Push(NewFilterView(m.router, m.options, func(opts civitai.SearchModelsOptions) tea.Cmd {
					m.options = opts
					m.page = opts.Page
					m.query = opts.Query // Sync query
					m.state = StateLoading
					return m.searchCmd()
				}))
				return m, nil
			}
		} else if m.state == StateResults {
			switch msg.String() {
			case "esc", "q":
				m.state = StateTyping
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.models)-1 {
					m.cursor++
				}
			case "right", "n":
				m.page++
				m.options.Page = m.page
				m.state = StateLoading
				return m, m.searchCmd()
			case "left", "p":
				if m.page > 1 {
					m.page--
					m.options.Page = m.page
					m.state = StateLoading
					return m, m.searchCmd()
				}
			case "f":
				m.router.Push(NewFilterView(m.router, m.options, func(opts civitai.SearchModelsOptions) tea.Cmd {
					m.options = opts
					m.page = opts.Page
					m.state = StateLoading
					return m.searchCmd()
				}))
				return m, nil
			case "enter", " ":
				if len(m.models) > 0 {
					m.router.Push(NewModelView(m.router, m.models[m.cursor]))
				}
			}
		} else if m.state == StateError {
			if msg.Type == tea.KeyEsc || msg.String() == "q" {
				m.state = StateTyping
				m.err = ""
			}
		}
	}
	return m, nil
}

func (m *SearchView) searchCmd() tea.Cmd {
	return func() tea.Msg {
		opts := m.options
		if opts.Query == "" && m.query != "" {
			opts.Query = m.query
		}
		opts.Limit = m.router.GetConfig().DefaultSearchLimit
		opts.Page = m.page

		res, err := m.router.GetClient().SearchModels(
			context.Background(),
			opts,
		)
		if err != nil {
			return errorMsg{err}
		}
		return modelsFoundMsg{res.Items}
	}
}

func (m *SearchView) View() string {
	b := strings.Builder{}

	b.WriteString(ui.Title("Search Models"))
	b.WriteString("\n\n")

	switch m.state {
	case StateTyping:
		b.WriteString(fmt.Sprintf("Query: %s█\n\n", m.query))

		// Show active filters
		filters := []string{}
		if len(m.options.Types) > 0 {
			filters = append(filters, "Type="+m.options.Types[0])
		}
		if m.options.Sort != "" {
			filters = append(filters, "Sort="+m.options.Sort)
		}
		if m.options.Period != "" {
			filters = append(filters, "Period="+m.options.Period)
		}
		if m.options.Rating > 0 {
			filters = append(filters, fmt.Sprintf("Rating=%d+", m.options.Rating))
		}
		if m.options.NSFW {
			filters = append(filters, "NSFW=true")
		}
		if len(filters) > 0 {
			b.WriteString(ui.Info(fmt.Sprintf("Filters: %s\n\n", strings.Join(filters, ", "))))
		}

		b.WriteString(ui.Sub("[Enter] Search  [Ctrl+F] Filter  [Esc] Back"))
	case StateLoading:
		b.WriteString(ui.Info(fmt.Sprintf("Searching for '%s'...\n", m.query)))
	case StateError:
		b.WriteString(ui.Error(fmt.Sprintf("Error: %s\n\n", m.err)))
		b.WriteString(ui.Sub("[Esc] Back to search"))
	case StateResults:
		b.WriteString(fmt.Sprintf("Results for '%s':\n\n", m.query))

		if len(m.models) == 0 {
			b.WriteString(ui.Warning("No models found.\n"))
		} else {
			// Basic pagination view, scrolling
			start := 0
			end := len(m.models)
			// Simplistic viewport mapping (show 10 items)
			if m.cursor >= 10 {
				start = m.cursor - 9
			}
			if end-start > 10 {
				end = start + 10
			}

			for i := start; i < end; i++ {
				model := m.models[i]
				cursor := "  "
				if m.cursor == i {
					cursor = ui.Info("> ")
				}
				ratingStr := "N/A"
				if model.Stats.RatingCount > 0 {
					ratingStr = fmt.Sprintf("%.1f", model.Stats.Rating)
				}
				b.WriteString(fmt.Sprintf("%s%s (%s) - %d dl | ★ %s | NSFW:%t\n", cursor, model.Name, model.Type, model.Stats.DownloadCount, ratingStr, model.NSFW))
			}
			b.WriteString(fmt.Sprintf("\nPage %d (%d items shown)\n", m.page, end-start))
		}
		b.WriteString("\n")
		b.WriteString(ui.Sub("[Up/Down] Navigate  [Left/Right] Prev/Next Page  [F] Filter  [Enter] Select  [Esc] Back"))
	}

	return fmt.Sprintf("\n  %s\n", strings.ReplaceAll(b.String(), "\n", "\n  "))
}
