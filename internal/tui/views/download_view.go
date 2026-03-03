package views

import (
	"context"
	"fmt"
	"strings"

	"github.com/KidiXDev/civ-cli/internal/civitai"
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
	DLStateConfirm DownloadViewState = iota
	DLStateDownloading
	DLStateSuccess
	DLStateError
)

type DownloadView struct {
	router  AppRouter
	model   civitai.Model
	version civitai.ModelVersion
	state   DownloadViewState
	err     string

	bytesDld   int64
	totalBytes int64
	outputFile string

	progressChan chan int
}

type progressMsg struct {
	bytes int
}

type dlCompleteMsg struct {
	path string
	err  error
}

// customWriter bridges io.Writer to tea.Msg
type customWriter struct {
	ch chan int
}

func (cw *customWriter) Write(p []byte) (n int, err error) {
	cw.ch <- len(p)
	return len(p), nil
}
func (cw *customWriter) ChangeMax64(max int64) {} // ignore, we have it in totalBytes

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
		progressChan: make(chan int, 100),
	}
}

func (m *DownloadView) Init() tea.Cmd {
	return nil
}

func waitForProgress(ch chan int) tea.Cmd {
	return func() tea.Msg {
		bytes := <-ch
		return progressMsg{bytes}
	}
}

func (m *DownloadView) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			if m.state != DLStateDownloading {
				m.router.Pop()
			}
		case "enter", "y":
			if m.state == DLStateConfirm {
				m.state = DLStateDownloading

				return m, tea.Batch(
					m.startDownloadCmd(),
					waitForProgress(m.progressChan),
				)
			}
		}

	case progressMsg:
		m.bytesDld += int64(msg.bytes)
		if m.state == DLStateDownloading {
			return m, waitForProgress(m.progressChan)
		}

	case dlCompleteMsg:
		if msg.err != nil {
			m.err = msg.err.Error()
			m.state = DLStateError
		} else {
			m.outputFile = msg.path
			// artificially max out
			m.bytesDld = m.totalBytes
			m.state = DLStateSuccess
		}
		return m, nil
	}
	return m, nil
}

func (m *DownloadView) startDownloadCmd() tea.Cmd {
	return func() tea.Msg {
		cw := &customWriter{ch: m.progressChan}
		path, err := m.router.GetDownloader().DownloadToWriter(
			context.Background(),
			m.version.ID,
			m.router.GetConfig().DefaultDownloadDir,
			cw,
		)
		return dlCompleteMsg{path: path, err: err}
	}
}

func (m *DownloadView) View() string {
	b := strings.Builder{}

	b.WriteString(ui.Title("Download Model"))
	b.WriteString("\n\n")

	switch m.state {
	case DLStateConfirm:
		b.WriteString(fmt.Sprintf("Ready to download '%s' - %s?\n", m.model.Name, m.version.Name))
		b.WriteString(fmt.Sprintf("Size: ~%.2f MB\n", float64(m.totalBytes)/(1024*1024)))
		b.WriteString("\n")
		b.WriteString(ui.Sub("[Enter] or [y] Start Download   [Esc] Cancel"))

	case DLStateDownloading:
		pct := 0.0
		if m.totalBytes > 0 {
			pct = float64(m.bytesDld) / float64(m.totalBytes)
		}

		barWidth := 40
		filled := int(float64(barWidth) * pct)
		if filled > barWidth {
			filled = barWidth
		}

		bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

		b.WriteString(fmt.Sprintf("Downloading %s...\n", m.version.Name))
		b.WriteString(fmt.Sprintf("[%s] %.1f%%\n", ui.Info(bar), pct*100))
		b.WriteString(fmt.Sprintf("%d / %d KB\n", m.bytesDld/1024, m.totalBytes/1024))
		b.WriteString("\n")
		b.WriteString(ui.Warning("Please wait..."))

	case DLStateSuccess:
		b.WriteString(ui.Success(fmt.Sprintf("Download complete!\nSaved to: %s\n", m.outputFile)))
		b.WriteString("\n")
		b.WriteString(ui.Sub("[Esc] Back to Model"))

	case DLStateError:
		b.WriteString(ui.Error(fmt.Sprintf("Download failed:\n%s\n", m.err)))
		b.WriteString("\n")
		b.WriteString(ui.Sub("[Esc] Back"))
	}

	return fmt.Sprintf("\n  %s\n", strings.ReplaceAll(b.String(), "\n", "\n  "))
}
