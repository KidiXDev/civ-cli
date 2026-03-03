package views

import (
	"github.com/KidiXDev/civ-cli/internal/civitai"
	"github.com/KidiXDev/civ-cli/internal/config"
	"github.com/KidiXDev/civ-cli/internal/downloader"
	tea "github.com/charmbracelet/bubbletea"
)

// AppRouter defines the capabilities an underlying router must provide to views.
type AppRouter interface {
	Push(v tea.Model)
	Pop()
	SetRoot(v tea.Model)
	Quit()
	SaveConfigAndProceed(apiKey string) error
	GetConfig() *config.Config
	GetConfigManager() *config.Manager
	GetClient() *civitai.Client
	GetDownloader() *downloader.Downloader
}
