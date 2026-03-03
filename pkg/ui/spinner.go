package ui

import (
	"fmt"
	"time"
)

// Spinner represents a simple CLI spinner for headless mode
type Spinner struct {
	done    chan struct{}
	message string
}

func NewSpinner(message string) *Spinner {
	return &Spinner{
		message: message,
		done:    make(chan struct{}),
	}
}

func (s *Spinner) Start() {
	frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	go func() {
		i := 0
		for {
			select {
			case <-s.done:
				fmt.Printf("\r%s\n", Success("✔ "+s.message+" complete"))
				return
			default:
				fmt.Printf("\r%s %s", Info(frames[i]), s.message)
				i = (i + 1) % len(frames)
				time.Sleep(100 * time.Millisecond)
			}
		}
	}()
}

func (s *Spinner) Stop() {
	close(s.done)
	// Add a small delay so output is cleanly cleared
	time.Sleep(50 * time.Millisecond)
}
