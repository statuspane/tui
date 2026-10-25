package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

// statusAppearance is how a status reads: a short label for rows, a headline for banners, and
// the colour of both.
type statusAppearance struct {
	label    string
	headline string
	style    lipgloss.Style
}

// A nil status means no check has run yet.
var pendingAppearance = statusAppearance{"Pending", "NOT CHECKED YET", mutedStyle}

var statusAppearances = map[string]statusAppearance{
	"operational":          {"Operational", "ALL SYSTEMS OPERATIONAL", lipgloss.NewStyle().Foreground(colorGreen)},
	"under_maintenance":    {"Maintenance", "UNDER MAINTENANCE", lipgloss.NewStyle().Foreground(colorBlue)},
	"degraded_performance": {"Degraded", "DEGRADED PERFORMANCE", lipgloss.NewStyle().Foreground(colorYellow)},
	"partial_outage":       {"Partial outage", "PARTIAL OUTAGE", lipgloss.NewStyle().Foreground(colorOrange)},
	"major_outage":         {"Major outage", "MAJOR OUTAGE", lipgloss.NewStyle().Foreground(colorRed)},
}

// appearanceOf shows a status this client doesn't know as it is, unstyled.
func appearanceOf(status *string) statusAppearance {
	if status == nil {
		return pendingAppearance
	}
	if appearance, ok := statusAppearances[*status]; ok {
		return appearance
	}
	return statusAppearance{label: *status, headline: strings.ToUpper(*status), style: lipgloss.NewStyle()}
}

func statusLabel(status *string) string {
	return appearanceOf(status).label
}

// renderStatus pads the plain label before styling: fmt would count the colour escape codes.
func renderStatus(status *string, width int) string {
	appearance := appearanceOf(status)
	return appearance.style.Render(fmt.Sprintf("%-*s", width, appearance.label))
}
