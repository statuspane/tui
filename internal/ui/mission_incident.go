package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

// incidentSteps are the stages of an incident's life, in order. The API's active statuses are the
// first three; an incident that is resolved is no longer active, so that step is always still ahead.
var incidentSteps = []struct{ status, label string }{
	{"investigating", "Investigating"},
	{"identified", "Identified"},
	{"monitoring", "Monitoring"},
	{"resolved", "Resolved"},
}

const (
	doneMark, currentMark, aheadMark = "✓", "●", "○"
	doneConnector, aheadConnector    = "─", "┄"
	// A connector between two steps is at least this wide, and has a space on each side.
	minConnectorWidth = 1
	connectorPadding  = 2

	incidentPanelTitle = "Active incident"
	loadingIncident    = "Loading…"
)

// incidentPanel is the app's worst active incident in full, as tall as it needs up to maxHeight.
func (m AppsModel) incidentPanel(app api.App, width, maxHeight int) string {
	contentWidth, _ := panelContentSize(width, maxHeight)
	appDetail, hasDetail := m.details[app.Slug]
	if !hasDetail {
		return panel(incidentPanelTitle, mutedStyle.Render(loadingIncident), width, min(oneLineBoxHeight, maxHeight))
	}
	if len(appDetail.ActiveIncidents) == 0 {
		return panel(incidentPanelTitle, mutedStyle.Render(noIncidentsLabel), width, min(oneLineBoxHeight, maxHeight))
	}

	incidents := appDetail.ActiveIncidents
	title := incidentPanelTitle
	body := incidentPanelBody(incidents[0], m.now(), contentWidth)
	if len(incidents) > 1 {
		title = fmt.Sprintf("%ss (%d)", incidentPanelTitle, len(incidents))
		body += "\n\n" + otherIncidentLines(incidents[1:], contentWidth)
	}
	return panelWithLook(incidentsPanelLook(incidents, false), title, body, width, min(lipgloss.Height(body)+panelChromeHeight, maxHeight))
}

// otherIncidentLines is a line for each incident past the first: its title in its impact's colour
// and, at the right edge, the provider and the stage it is at.
func otherIncidentLines(incidents []api.Incident, contentWidth int) string {
	lines := make([]string, len(incidents))
	for index, incident := range incidents {
		impactStyle := impactStyleOf(incident)
		where := incident.Provider
		if incident.Status != nil {
			where += " · " + *incident.Status
		}
		lines[index] = headerBar(impactStyle.Render(statusDot)+" "+incident.Title, mutedStyle.Render(where), contentWidth)
	}
	return strings.Join(lines, "\n")
}

// incidentPanelBody is the incident's title with where and when beside it, what the provider last
// said about it, and the components it touches.
func incidentPanelBody(incident api.Incident, now time.Time, contentWidth int) string {
	impactStyle := impactStyleOf(incident)
	where := incident.Provider
	if incident.StartedAt != nil {
		where += " · started " + ageText(now, *incident.StartedAt)
	}
	sections := []string{headerBar(impactStyle.Render(statusDot)+" "+titleStyle.Render(incident.Title), mutedStyle.Render(where), contentWidth)}
	if progress := progressLine(incident.Status, impactStyle, contentWidth); progress != "" {
		sections = append(sections, progress)
	}

	if incident.LatestUpdate != nil && incident.LatestUpdate.Body != "" {
		sections = append(sections, wrappedTo("“"+incident.LatestUpdate.Body+"”", contentWidth))
	}
	if len(incident.AffectedComponents) > 0 {
		names := make([]string, len(incident.AffectedComponents))
		for index, component := range incident.AffectedComponents {
			names[index] = component.Name
		}
		sections = append(sections, mutedStyle.Render(wrappedTo(fmt.Sprintf("Affects ▸ %s", strings.Join(names, " · ")), contentWidth)))
	}
	return strings.Join(sections, "\n\n")
}

// wrappedTo breaks text into lines of at most width cells.
func wrappedTo(text string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(text)
}

// progressLine is the incident's steps with a connector between each: a tick and a solid line for
// the steps done, a filled mark in the incident's colour for the current one, a hollow mark and a
// dotted line for those ahead. An incident with no status has no line, and when the steps do
// not fit only the current one is named.
func progressLine(status *string, currentStyle lipgloss.Style, contentWidth int) string {
	if status == nil {
		return ""
	}
	current := -1
	for index, step := range incidentSteps {
		if step.status == *status {
			current = index
		}
	}
	if current < 0 {
		return ""
	}

	labels := make([]string, len(incidentSteps))
	labelsWidth := 0
	for index, step := range incidentSteps {
		switch {
		case index < current:
			labels[index] = watchedStyle.Render(doneMark + " " + step.label)
		case index == current:
			labels[index] = currentStyle.Bold(true).Render(currentMark + " " + step.label)
		default:
			labels[index] = mutedStyle.Render(aheadMark + " " + step.label)
		}
		labelsWidth += lipgloss.Width(labels[index])
	}

	connectorCount := len(incidentSteps) - 1
	if labelsWidth+connectorCount*(connectorPadding+minConnectorWidth) > contentWidth {
		return currentStyle.Bold(true).Render(fmt.Sprintf("%s %s · step %d of %d", currentMark, incidentSteps[current].label, current+1, len(incidentSteps)))
	}
	connectorWidth := max((contentWidth-labelsWidth-connectorCount*connectorPadding)/connectorCount, minConnectorWidth)
	var line strings.Builder
	for index, label := range labels {
		line.WriteString(label)
		if index == connectorCount {
			break
		}
		if index < current {
			line.WriteString(" " + watchedStyle.Render(strings.Repeat(doneConnector, connectorWidth)) + " ")
		} else {
			line.WriteString(" " + mutedStyle.Render(strings.Repeat(aheadConnector, connectorWidth)) + " ")
		}
	}
	return line.String()
}
