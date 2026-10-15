package ui

import "charm.land/lipgloss/v2"

// ANSI palette indexes, so the terminal's own theme decides the exact shades.
var (
	colorRed    = lipgloss.Color("1")
	colorGreen  = lipgloss.Color("2")
	colorYellow = lipgloss.Color("3")
	colorBlue   = lipgloss.Color("4")
	colorOrange = lipgloss.Color("208")
	colorMuted  = lipgloss.Color("8")
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Bold(true)
	mutedStyle    = lipgloss.NewStyle().Foreground(colorMuted)
	errorStyle    = lipgloss.NewStyle().Foreground(colorRed)
)
