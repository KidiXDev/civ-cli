package ui

import "github.com/fatih/color"

var (
	Success = color.New(color.FgGreen).SprintFunc()
	Info    = color.New(color.FgCyan).SprintFunc()
	Warning = color.New(color.FgYellow).SprintFunc()
	Error   = color.New(color.FgRed, color.Bold).SprintFunc()
	Title   = color.New(color.FgMagenta, color.Bold).SprintFunc()
	Sub     = color.New(color.FgHiBlack).SprintFunc()
)
