package app

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/KidiXDev/civ-cli/internal/civitai"
	"github.com/KidiXDev/civ-cli/internal/tui"
	"github.com/KidiXDev/civ-cli/pkg/output"
	"github.com/KidiXDev/civ-cli/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	headlessFlag bool
	debugFlag    bool
	jsonFlag     bool
)

func Execute() {
	rootCmd := &cobra.Command{
		Use:   "civitool",
		Short: "Civitool - The Ultimate Civitai CLI",
		Long:  `A hybrid CLI and TUI application for interacting with the Civitai Public API.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := Bootstrap(debugFlag)
			if err != nil {
				return err
			}
			a.IsHeadless = headlessFlag

			if a.IsHeadless {
				return fmt.Errorf("headless mode requires a subcommand (e.g., search, config). Use --help for more info")
			}

			// Start TUI
			p := tea.NewProgram(tui.InitialAppModel(a.ConfigManager, a.Config, a.CivitaiClient, a.Downloader), tea.WithAltScreen())
			_, err = p.Run()
			return err
		},
	}

	rootCmd.PersistentFlags().BoolVar(&headlessFlag, "headless", false, "Run in headless CLI mode instead of TUI")
	rootCmd.PersistentFlags().BoolVar(&debugFlag, "debug", false, "Enable debug logging")
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output results in JSON format (headless only)")

	// --- Headless Subcommands ---

	// Search
	searchCmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search for models",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := Bootstrap(debugFlag)
			if err != nil {
				return err
			}

			query := ""
			if len(args) > 0 {
				query = args[0]
			}
			limit, _ := cmd.Flags().GetInt("limit")

			s := ui.NewSpinner("Searching models...")
			s.Start()
			opts := civitai.SearchModelsOptions{
				Query: query,
				Limit: limit,
				Page:  1,
			}
			res, err := a.CivitaiClient.SearchModels(context.Background(), opts)
			s.Stop()

			if err != nil {
				fmt.Println(ui.Error("Search failed:", err))
				return err
			}

			if jsonFlag || a.Config.OutputFormat == "json" {
				formatter := &output.JSONFormatter{}
				return formatter.Print(res.Items)
			}

			// Table Output
			formatter := &output.TableFormatter{
				Headers: []string{"ID", "Name", "Type", "NSFW", "Downloads"},
				RowFunc: func(item interface{}) []string {
					m := item.(civitai.Model)
					nsfwStr := "No"
					if m.NSFW {
						nsfwStr = "Yes"
					}
					return []string{
						strconv.Itoa(m.ID),
						m.Name,
						m.Type,
						nsfwStr,
						strconv.Itoa(m.Stats.DownloadCount),
					}
				},
			}
			var items []interface{}
			for _, m := range res.Items {
				items = append(items, m)
			}
			return formatter.Print(items)
		},
	}
	searchCmd.Flags().IntP("limit", "l", 20, "Number of results to fetch")

	// Config
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	configShowCmd := &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := Bootstrap(debugFlag)
			if err != nil {
				return err
			}
			f := &output.JSONFormatter{}
			return f.Print(a.Config)
		},
	}

	configSetCmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := Bootstrap(debugFlag)
			if err != nil {
				return err
			}
			key, val := args[0], args[1]

			switch key {
			case "api_key":
				a.Config.APIKey = val
			case "default_search_limit":
				l, _ := strconv.Atoi(val)
				a.Config.DefaultSearchLimit = l
			case "default_download_path":
				a.Config.DefaultDownloadDir = val
			case "output_format":
				a.Config.OutputFormat = val
			case "theme":
				a.Config.Theme = val
			case "timeout":
				t, _ := strconv.Atoi(val)
				a.Config.TimeoutSeconds = t
			case "retry_count":
				r, _ := strconv.Atoi(val)
				a.Config.RetryCount = r
			default:
				return fmt.Errorf("unknown config key: %s", key)
			}

			if err := a.ConfigManager.Save(a.Config); err != nil {
				return err
			}
			fmt.Println(ui.Success(fmt.Sprintf("Saved %s = %s", key, val)))
			return nil
		},
	}

	configCmd.AddCommand(configShowCmd, configSetCmd)
	rootCmd.AddCommand(searchCmd, configCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
