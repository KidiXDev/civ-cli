package tui

import (
	"github.com/KidiXDev/civ-cli/internal/civitai"
	"github.com/KidiXDev/civ-cli/internal/config"
	"github.com/KidiXDev/civ-cli/internal/downloader"
	"github.com/KidiXDev/civ-cli/internal/tui/views"
	tea "github.com/charmbracelet/bubbletea"
)

// MainModel is the top-level BubbleTea model.
type MainModel struct {
	// Which sub-model is currently active
	router *Router

	width  int
	height int
}

func InitialAppModel(cm *config.Manager, cfg *config.Config, client *civitai.Client, dl *downloader.Downloader) *MainModel {
	r := NewRouter(cm, cfg, client, dl)

	m := &MainModel{
		router: r,
	}

	// Decision logic for initial view as required by project specs:
	// If the config file does not exist or API Key is empty, go to Onboarding (Welcome)
	// Otherwise, go to Home
	if !cm.FileExists() || cfg.APIKey == "" {
		r.Push(views.NewWelcomeView(r))
	} else {
		r.Push(views.NewHomeView(r))
	}

	return m
}

func (m *MainModel) Init() tea.Cmd {
	return m.router.Init()
}

func (m *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle global messages like window resize
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
	}

	// Handle exit
	if msg, ok := msg.(tea.KeyMsg); ok {
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	// Route the message to the current active view via the router
	cmd := m.router.Update(msg)

	// If router says quit, we quit
	if m.router.ShouldQuit {
		return m, tea.Quit
	}

	return m, cmd
}

func (m *MainModel) View() string {
	return m.router.View()
}
