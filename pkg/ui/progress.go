package ui

import (
	"fmt"
	"time"

	"github.com/schollz/progressbar/v3"
)

// NewDownloadProgressBar returns a new progress bar configured for file downloading.
func NewDownloadProgressBar(maxBytes int64, description string) *progressbar.ProgressBar {
	return progressbar.NewOptions64(
		maxBytes,
		progressbar.OptionSetDescription(Info(description)),
		progressbar.OptionSetWriter(nil),
		progressbar.OptionShowBytes(true),
		progressbar.OptionSetWidth(30),
		progressbar.OptionThrottle(65*time.Millisecond),
		progressbar.OptionShowCount(),
		progressbar.OptionOnCompletion(func() {
			fmt.Printf("\n%s\n", Success("Download complete!"))
		}),
		progressbar.OptionSpinnerType(14),
		progressbar.OptionFullWidth(),
		progressbar.OptionSetRenderBlankState(true),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
}
