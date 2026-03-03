package app

import (
	"fmt"

	"github.com/KidiXDev/civ-cli/internal/civitai"
	"github.com/KidiXDev/civ-cli/internal/config"
	"github.com/KidiXDev/civ-cli/internal/downloader"
	"github.com/KidiXDev/civ-cli/internal/logger"
)

// Bootstrap initializes all core services before the UI or CLI is executed.
func Bootstrap(debug bool) (*App, error) {
	logger.InitLogger(debug)

	cm := config.NewManager()
	cfg, err := cm.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	client := civitai.NewClient(cfg)
	dl := downloader.NewDownloader(cfg)

	return &App{
		ConfigManager: cm,
		Config:        cfg,
		CivitaiClient: client,
		Downloader:    dl,
	}, nil
}
