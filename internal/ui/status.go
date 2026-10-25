package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"
)

type statusAppearance struct {
	label string
	style lipgloss.Style
}

// A nil status means no check has run yet.
var pendingAppearance = statusAppearance{"Pending", mutedStyle}

var statusAppearances = map[string]statusAppearance{
	"operational":          {"Operational", lipgloss.NewStyle().Foreground(colorGreen)},
	"under_maintenance":    {"Maintenance", lipgloss.NewStyle().Foreground(colorBlue)},
	"degraded_performance": {"Degraded", lipgloss.NewStyle().Foreground(colorYellow)},
	"partial_outage":       {"Partial outage", lipgloss.NewStyle().Foreground(colorOrange)},
	"major_outage":         {"Major outage", lipgloss.NewStyle().Foreground(colorRed)},
}

// statusOperational is the status of something with nothing wrong.
const statusOperational = "operational"

// isAffected is whether something has a known status that is not operational. A status that is not
// there yet means no check has run, which is neither.
func isAffected(status *string) bool {
	return status != nil && *status != statusOperational
}

// appearanceOf shows a status this client doesn't know as it is, unstyled.
func appearanceOf(status *string) statusAppearance {
	if status == nil {
		return pendingAppearance
	}
	if appearance, ok := statusAppearances[*status]; ok {
		return appearance
	}
	return statusAppearance{label: *status, style: lipgloss.NewStyle()}
}

func statusLabel(status *string) string {
	return appearanceOf(status).label
}

// renderStatus pads the plain label before styling: fmt would count the colour escape codes.
func renderStatus(status *string, width int) string {
	appearance := appearanceOf(status)
	return appearance.style.Render(fmt.Sprintf("%-*s", width, appearance.label))
}
