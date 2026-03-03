package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger sets up zerolog for the application.
func InitLogger(debug bool) {
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// We'll write logs to a file in the executable's directory to avoid clashing with TUI and CLI output
	logFile, err := os.OpenFile("civitool.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.Logger = zerolog.New(logFile).With().Timestamp().Logger()
	} else {
		// Fallback to console with console writer (will disrupt TUI, but good for headless debugging)
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
		log.Warn().Err(err).Msg("Failed to open log file, falling back to stderr")
	}
}
