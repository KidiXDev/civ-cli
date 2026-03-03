package app

import (
	"github.com/KidiXDev/civ-cli/internal/civitai"
	"github.com/KidiXDev/civ-cli/internal/config"
	"github.com/KidiXDev/civ-cli/internal/downloader"
)

// App is the dependency container for the entire application.
// We avoid global variables by injecting this container where needed.
type App struct {
	ConfigManager *config.Manager
	Config        *config.Config
	CivitaiClient *civitai.Client
	Downloader    *downloader.Downloader
	IsHeadless    bool
}
