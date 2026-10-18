package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

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

	warningStyle = lipgloss.NewStyle().Foreground(colorYellow)
	watchedStyle = lipgloss.NewStyle().Foreground(colorGreen)
	addedStyle   = lipgloss.NewStyle().Foreground(colorBlue)
	removedStyle = lipgloss.NewStyle().Foreground(colorRed)

	// Roles of the shared screen chrome: panel edges and titles, and the keys in the footer.
	accentStyle      = lipgloss.NewStyle().Bold(true).Foreground(colorBlue)
	panelBorderStyle = lipgloss.NewStyle().Foreground(colorMuted)
	keyStyle         = lipgloss.NewStyle().Bold(true)

	// cursorRowStyle is the bar across the row under the cursor; reverse video works on any theme.
	cursorRowStyle = lipgloss.NewStyle().Bold(true).Reverse(true)
)

// rowTintColor is the subtle background of the selected row: a neutral grey, darker or lighter to
// suit the terminal's background.
func rowTintColor(hasDarkBackground bool) color.Color {
	return lipgloss.LightDark(hasDarkBackground)(lipgloss.Color("254"), lipgloss.Color("236"))
}

// Keyed by the API's incident impact levels.
var impactStyles = map[string]lipgloss.Style{
	"none":     mutedStyle,
	"minor":    lipgloss.NewStyle().Foreground(colorYellow),
	"major":    lipgloss.NewStyle().Foreground(colorOrange),
	"critical": lipgloss.NewStyle().Foreground(colorRed),
}

func markerStyleFor(state watchState) lipgloss.Style {
	switch state {
	case stateWatched:
		return watchedStyle
	case stateAdded:
		return addedStyle
	case stateRemoved:
		return removedStyle
	}
	return mutedStyle
}

// nameStyleFor makes the chosen components easy to spot in a long list: watched in green, to be
// added in bold blue, to be removed in red and struck through; a component that is not watched
// stays plain.
func nameStyleFor(state watchState) lipgloss.Style {
	switch state {
	case stateWatched:
		return watchedStyle
	case stateAdded:
		return addedStyle.Bold(true)
	case stateRemoved:
		return removedStyle.Strikethrough(true)
	}
	return lipgloss.NewStyle()
}
