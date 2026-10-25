package ui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

// incidentSteps are the stages of an incident's life, in order. The API's active statuses are the
// first three; an incident that is resolved is no longer active, so that step is always still ahead.
var incidentSteps = [...]incidentStep{
	{"investigating", "Investigating"},
	{"identified", "Identified"},
	{"monitoring", "Monitoring"},
	{"resolved", "Resolved"},
}

type incidentStep struct{ status, label string }

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
	appDetail, hasDetail := m.selectedDetail.of(app)
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
		providerAndStage := incident.Provider
		if incident.Status != nil {
			providerAndStage += " · " + *incident.Status
		}
		lines[index] = headerBar(impactStyle.Render(statusDot)+" "+incident.Title, mutedStyle.Render(providerAndStage), contentWidth)
	}
	return strings.Join(lines, "\n")
}

// incidentPanelBody is the incident's title with where and when beside it, what the provider last
// said about it, and the components it touches.
func incidentPanelBody(incident api.Incident, now time.Time, contentWidth int) string {
	impactStyle := impactStyleOf(incident)
	providerAndAge := incident.Provider
	if incident.StartedAt != nil {
		providerAndAge += " · started " + ageText(now, *incident.StartedAt)
	}
	sections := []string{headerBar(impactStyle.Render(statusDot)+" "+titleStyle.Render(incident.Title), mutedStyle.Render(providerAndAge), contentWidth)}
	if progress := progressLine(incident.Status, impactStyle, contentWidth); progress != "" {
		sections = append(sections, progress)
	}

	if incident.LatestUpdate != nil && incident.LatestUpdate.Body != "" {
		sections = append(sections, wrapText("“"+incident.LatestUpdate.Body+"”", contentWidth))
	}
	if len(incident.AffectedComponents) > 0 {
		names := make([]string, len(incident.AffectedComponents))
		for index, component := range incident.AffectedComponents {
			names[index] = component.Name
		}
		sections = append(sections, mutedStyle.Render(wrapText(fmt.Sprintf("Affects ▸ %s", strings.Join(names, " · ")), contentWidth)))
	}
	return strings.Join(sections, "\n\n")
}

// progressLine is the incident's steps with a connector between each: a tick and a solid line for
// the steps done, a filled mark in the incident's colour for the current one, a hollow mark and a
// dotted line for those ahead. An incident with no status has no line, and when the steps do
// not fit only the current one is named.
func progressLine(status *string, currentStyle lipgloss.Style, contentWidth int) string {
	current, isKnown := currentStepIndex(status)
	if !isKnown {
		return ""
	}

	labels := stepLabels(current, currentStyle)
	connectorCount := len(labels) - 1
	labelsWidth := 0
	for _, label := range labels {
		labelsWidth += lipgloss.Width(label)
	}
	if labelsWidth+connectorCount*(connectorPadding+minConnectorWidth) > contentWidth {
		return compactProgress(current, currentStyle)
	}

	connectorWidth := max((contentWidth-labelsWidth-connectorCount*connectorPadding)/connectorCount, minConnectorWidth)
	return labelsJoinedByConnectors(labels, current, connectorWidth)
}

// currentStepIndex is the position of the incident's status among the steps, and whether it is one
// of them: an incident with no status, or one this client doesn't know, has no current step.
func currentStepIndex(status *string) (index int, isKnown bool) {
	if status == nil {
		return 0, false
	}
	index = slices.IndexFunc(incidentSteps[:], func(step incidentStep) bool { return step.status == *status })
	return index, index >= 0
}

// stepLabels are the steps with their marks: done, current (in the incident's colour) or ahead.
func stepLabels(current int, currentStyle lipgloss.Style) []string {
	labels := make([]string, len(incidentSteps))
	for index, step := range incidentSteps {
		switch {
		case index < current:
			labels[index] = watchedStyle.Render(doneMark + " " + step.label)
		case index == current:
			labels[index] = currentStyle.Bold(true).Render(currentMark + " " + step.label)
		default:
			labels[index] = mutedStyle.Render(aheadMark + " " + step.label)
		}
	}
	return labels
}

// compactProgress names only the current step and where it is among them.
func compactProgress(current int, currentStyle lipgloss.Style) string {
	return currentStyle.Bold(true).Render(fmt.Sprintf("%s %s · step %d of %d", currentMark, incidentSteps[current].label, current+1, len(incidentSteps)))
}

// labelsJoinedByConnectors puts a line of connectorWidth cells between the labels: solid up to the
// current step, dotted after it.
func labelsJoinedByConnectors(labels []string, current, connectorWidth int) string {
	var line strings.Builder
	for index, label := range labels {
		line.WriteString(label)
		if index == len(labels)-1 {
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
