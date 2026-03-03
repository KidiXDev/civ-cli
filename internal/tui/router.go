package tui

import (
	"github.com/KidiXDev/civ-cli/internal/civitai"
	"github.com/KidiXDev/civ-cli/internal/config"
	"github.com/KidiXDev/civ-cli/internal/downloader"
	"github.com/KidiXDev/civ-cli/internal/tui/views"
	tea "github.com/charmbracelet/bubbletea"
)

// Router manages a stack of Views (tea.Model)
type Router struct {
	stack      []tea.Model
	ShouldQuit bool

	cm     *config.Manager
	cfg    *config.Config
	client *civitai.Client
	dl     *downloader.Downloader
}

func NewRouter(cm *config.Manager, cfg *config.Config, client *civitai.Client, dl *downloader.Downloader) *Router {
	return &Router{
		stack:  make([]tea.Model, 0),
		cm:     cm,
		cfg:    cfg,
		client: client,
		dl:     dl,
	}
}

func (r *Router) Push(v tea.Model) {
	r.stack = append(r.stack, v)
}

func (r *Router) Pop() {
	if len(r.stack) > 1 {
		r.stack = r.stack[:len(r.stack)-1]
	}
}

func (r *Router) SetRoot(v tea.Model) {
	r.stack = []tea.Model{v}
}

func (r *Router) Quit() {
	r.ShouldQuit = true
}

func (r *Router) SaveConfigAndProceed(apiKey string) error {
	r.cfg.APIKey = apiKey
	if err := r.cm.Save(r.cfg); err != nil {
		return err
	}
	r.client.SetAuthToken(apiKey)

	// Switch to home view
	r.SetRoot(views.NewHomeView(r))
	return nil
}

func (r *Router) GetConfig() *config.Config {
	return r.cfg
}
func (r *Router) GetConfigManager() *config.Manager {
	return r.cm
}
func (r *Router) GetClient() *civitai.Client {
	return r.client
}
func (r *Router) GetDownloader() *downloader.Downloader {
	return r.dl
}

func (r *Router) Current() tea.Model {
	if len(r.stack) == 0 {
		return nil
	}
	return r.stack[len(r.stack)-1]
}

func (r *Router) Init() tea.Cmd {
	if c := r.Current(); c != nil {
		return c.Init()
	}
	return nil
}

func (r *Router) Update(msg tea.Msg) tea.Cmd {
	if len(r.stack) == 0 {
		return nil
	}

	idx := len(r.stack) - 1
	c := r.stack[idx]

	newModel, cmd := c.Update(msg)

	// Only update if the stack hasn't structurally replaced the current frame
	if idx < len(r.stack) {
		// Comparing interface types for pointer equality
		if r.stack[idx] == c {
			r.stack[idx] = newModel
		}
	}

	return cmd
}

func (r *Router) View() string {
	if c := r.Current(); c != nil {
		return c.View()
	}
	return "No view to display"
}
