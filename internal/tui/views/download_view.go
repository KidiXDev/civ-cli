package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KidiXDev/civ-cli/internal/civitai"
	"github.com/KidiXDev/civ-cli/internal/downloader"
	"github.com/KidiXDev/civ-cli/pkg/ui"
	tea "github.com/charmbracelet/bubbletea"
)

type ConfirmView struct {
	router  AppRouter
	prompt  string
	onYes   func() tea.Cmd
	onNo    func() tea.Cmd
	confirm bool
}

func NewConfirmView(router AppRouter, prompt string, onYes, onNo func() tea.Cmd) *ConfirmView {
	return &ConfirmView{
		router:  router,
		prompt:  prompt,
		onYes:   onYes,
		onNo:    onNo,
		confirm: true, // default Yes
	}
}

func (m *ConfirmView) Init() tea.Cmd {
	return nil
}

func (m *ConfirmView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.router.Pop()
			if m.onNo != nil {
				return m, m.onNo()
			}
		case "left", "h":
			m.confirm = true
		case "right", "l":
			m.confirm = false
		case "enter", " ":
			m.router.Pop()
			if m.confirm && m.onYes != nil {
				return m, m.onYes()
			}
			if !m.confirm && m.onNo != nil {
				return m, m.onNo()
			}
		}
	}
	return m, nil
}

func (m *ConfirmView) View() string {
	b := strings.Builder{}

	b.WriteString(ui.Title("Confirmation"))
	b.WriteString("\n\n")

	b.WriteString(m.prompt)
	b.WriteString("\n\n")

	yes := "  Yes  "
	no := "  No   "

	if m.confirm {
		yes = ui.Info("> Yes <")
	} else {
		no = ui.Info("> No  <")
	}

	b.WriteString(fmt.Sprintf("%s    %s\n", yes, no))
	b.WriteString("\n")
	b.WriteString(ui.Sub("[Left/Right] Select  [Enter] Confirm  [Esc] Cancel"))

	return fmt.Sprintf("\n  %s\n", strings.ReplaceAll(b.String(), "\n", "\n  "))
}

// download_view.go helper definitions
type DownloadViewState int

const (
	DLStateConfirm     DownloadViewState = iota
	DLStateDownloading                   // actively downloading
	DLStateCancelling                    // user requested cancel, waiting for cleanup
	DLStateSuccess
	DLStateError
)

type DownloadView struct {
	router  AppRouter
	model   civitai.Model
	version civitai.ModelVersion
	state   DownloadViewState
	err     string

	progress   downloader.Progress        // latest progress snapshot
	result     *downloader.DownloadResult // populated on success
	outputFile string
	totalBytes int64

	progressChan chan downloader.Progress
	doneChan     chan dlCompleteMsg
	cancel       context.CancelFunc
}

// --- Bubbletea messages ---

type downloadProgressMsg struct {
	progress downloader.Progress
}

type dlCompleteMsg struct {
	result *downloader.DownloadResult
	err    error
}

func NewDownloadView(router AppRouter, model civitai.Model, version civitai.ModelVersion) *DownloadView {
	var totalBytes int64
	if len(version.Files) > 0 {
		for _, f := range version.Files {
			if f.Primary {
				totalBytes = int64(f.SizeKB * 1024)
			}
		}
		if totalBytes == 0 {
			totalBytes = int64(version.Files[0].SizeKB * 1024)
		}
	}

	return &DownloadView{
		router:       router,
		model:        model,
		version:      version,
		state:        DLStateConfirm,
		totalBytes:   totalBytes,
		progressChan: make(chan downloader.Progress, 50),
		doneChan:     make(chan dlCompleteMsg, 1),
	}
}

func (m *DownloadView) Init() tea.Cmd {
	return nil
}

// waitForDownloadEvent blocks until a progress update or completion arrives.
func waitForDownloadEvent(progressCh chan downloader.Progress, doneCh chan dlCompleteMsg) tea.Cmd {
	return func() tea.Msg {
		select {
		case p := <-progressCh:
			return downloadProgressMsg{progress: p}
		case d := <-doneCh:
			return d
		}
	}
}

func (m *DownloadView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			switch m.state {
			case DLStateDownloading:
				m.state = DLStateCancelling
				if m.cancel != nil {
					m.cancel()
				}
				// keep listening for the completion message
			case DLStateCancelling:
				// already cancelling – ignore
			default:
				m.router.Pop()
			}
		case "enter", "y":
			if m.state == DLStateConfirm {
				m.state = DLStateDownloading
				return m, m.startDownloadCmd()
			}
		}

	case downloadProgressMsg:
		m.progress = msg.progress
		if m.state == DLStateDownloading || m.state == DLStateCancelling {
			return m, waitForDownloadEvent(m.progressChan, m.doneChan)
		}

	case dlCompleteMsg:
		if msg.err != nil {
			if m.state == DLStateCancelling {
				m.err = "Download cancelled by user"
			} else {
				m.err = msg.err.Error()
			}
			m.state = DLStateError
		} else {
			m.result = msg.result
			m.outputFile = msg.result.FilePath
			m.progress.DownloadedBytes = m.totalBytes // ensure bar shows 100%
			m.state = DLStateSuccess
		}
		return m, nil
	}
	return m, nil
}

func (m *DownloadView) startDownloadCmd() tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	return func() tea.Msg {
		go func() {
			result, err := m.router.GetDownloader().Download(
				ctx,
				m.version.ID,
				downloader.DownloadOptions{
					OutputDir: m.router.GetConfig().DefaultDownloadDir,
					ProgressCb: func(p downloader.Progress) {
						select {
						case m.progressChan <- p:
						default: // don't block if TUI is slow
						}
					},
				},
			)
			m.doneChan <- dlCompleteMsg{result: result, err: err}
		}()

		// Block until the first event arrives
		select {
		case p := <-m.progressChan:
			return downloadProgressMsg{progress: p}
		case d := <-m.doneChan:
			return d
		}
	}
}

func (m *DownloadView) View() string {
	b := strings.Builder{}

	b.WriteString(ui.Title("Download Model"))
	b.WriteString("\n\n")

	switch m.state {
	case DLStateConfirm:
		b.WriteString(fmt.Sprintf("Ready to download '%s' - %s?\n", m.model.Name, m.version.Name))
		b.WriteString(fmt.Sprintf("Size: ~%s\n", downloader.FormatBytes(m.totalBytes)))
		b.WriteString("\n")
		b.WriteString(ui.Sub("[Enter] or [y] Start Download   [Esc] Cancel"))

	case DLStateDownloading, DLStateCancelling:
		p := m.progress
		pct := p.Percentage / 100.0
		if pct > 1 {
			pct = 1
		}

		barWidth := 40
		filled := int(float64(barWidth) * pct)
		if filled > barWidth {
			filled = barWidth
		}

		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		b.WriteString(fmt.Sprintf("Downloading %s...\n", m.version.Name))
		b.WriteString(fmt.Sprintf("[%s] %.1f%%\n", ui.Info(bar), p.Percentage))
		b.WriteString(fmt.Sprintf("%s / %s\n",
			downloader.FormatBytes(p.DownloadedBytes),
			downloader.FormatBytes(p.TotalBytes)))
		b.WriteString("\n")

		// Speed + ETA
		b.WriteString(fmt.Sprintf("Speed: %s  |  ETA: %s\n",
			ui.Info(downloader.FormatSpeed(p.Speed)),
			downloader.FormatETA(p.ETA)))

		// Chunk indicator
		if p.ActiveChunks > 0 {
			b.WriteString(fmt.Sprintf("Chunks: %s active  |  Elapsed: %s\n",
				ui.Info(fmt.Sprintf("%d", p.ActiveChunks)),
				p.Elapsed.Truncate(time.Second).String()))
		}
		b.WriteString("\n")

		if m.state == DLStateCancelling {
			b.WriteString(ui.Warning("Cancelling download..."))
		} else {
			b.WriteString(ui.Warning("Please wait...  [Esc] Cancel"))
		}

	case DLStateSuccess:
		var summary string
		if m.result != nil {
			summary = fmt.Sprintf("Download complete!\nSaved to: %s\nAvg speed: %s  |  Time: %s  |  Chunks: %d\n",
				m.outputFile,
				downloader.FormatSpeed(m.result.AvgSpeed),
				m.result.Duration.Truncate(time.Second).String(),
				m.result.ChunksUsed)
		} else {
			summary = fmt.Sprintf("Download complete!\nSaved to: %s\n", m.outputFile)
		}
		b.WriteString(ui.Success(summary))
		b.WriteString("\n")
		b.WriteString(ui.Sub("[Esc] Back to Model"))

	case DLStateError:
		b.WriteString(ui.Error(fmt.Sprintf("Download failed:\n%s\n", m.err)))
		b.WriteString("\n")
		b.WriteString(ui.Sub("[Esc] Back"))
	}

	return fmt.Sprintf("\n  %s\n", strings.ReplaceAll(b.String(), "\n", "\n  "))
}
