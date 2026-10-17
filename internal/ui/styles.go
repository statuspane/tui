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

	// cursorRowStyle is the bar across the row under the cursor; reverse video works on any theme.
	cursorRowStyle = lipgloss.NewStyle().Bold(true).Reverse(true)
)

// Keyed by the API's incident impact levels.
var impactStyles = map[string]lipgloss.Style{
	"none":     mutedStyle,
	"minor":    lipgloss.NewStyle().Foreground(colorYellow),
	"major":    lipgloss.NewStyle().Foreground(colorOrange),
	"critical": lipgloss.NewStyle().Foreground(colorRed),
}
