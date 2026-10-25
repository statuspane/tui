package ui

import (
	"slices"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

const (
	bullet            = "● "
	indent            = "  "
	noImpactLabel     = "unrated"
	noCheckYetLabel   = "not checked yet"
	noIncidentsLabel  = "No active incidents."
	noComponentsLabel = "This app doesn't watch any components yet."

	// The least space between a component's name and its status.
	minComponentGap = 2
)

// componentGroup is the components of one provider that share a group name ("" for none).
type componentGroup struct {
	provider  string
	groupName string
	members   []api.Component
}

// groupComponents keeps the order the API sent them in, which is by provider then name.
func groupComponents(components []api.Component) []componentGroup {
	var groups []componentGroup
	for _, component := range components {
		groupName := ""
		if component.GroupName != nil {
			groupName = *component.GroupName
		}
		found := false
		for index := range groups {
			if groups[index].provider == component.Provider && groups[index].groupName == groupName {
				groups[index].members = append(groups[index].members, component)
				found = true
				break
			}
		}
		if !found {
			groups = append(groups, componentGroup{component.Provider, groupName, []api.Component{component}})
		}
	}
	return groups
}

func checkedText(lastCheckedAt *time.Time, now time.Time) string {
	if lastCheckedAt == nil {
		return noCheckYetLabel
	}
	return "checked " + ageText(now, *lastCheckedAt)
}

// impactStyleOf is the colour of an incident's impact, muted when it has none or one we don't know.
func impactStyleOf(incident api.Incident) lipgloss.Style {
	if incident.Impact != nil {
		if style, ok := impactStyles[*incident.Impact]; ok {
			return style
		}
	}
	return mutedStyle
}

func renderIncident(incident api.Incident, now time.Time) string {
	impact := noImpactLabel
	if incident.Impact != nil {
		impact = *incident.Impact
	}
	impactStyle := impactStyleOf(incident)

	details := []string{impact}
	if incident.Status != nil {
		details = append(details, strings.ReplaceAll(*incident.Status, "_", " "))
	}
	if incident.StartedAt != nil {
		details = append(details, "started "+ageText(now, *incident.StartedAt))
	}

	lines := []string{
		impactStyle.Render(bullet) + titleStyle.Render(incident.Title),
		indent + mutedStyle.Render(incident.Provider+" · ") + impactStyle.Render(strings.Join(details, " · ")),
	}
	if incident.LatestUpdate != nil && incident.LatestUpdate.Body != "" {
		lines = append(lines, indent+incident.LatestUpdate.Body)
	}
	if len(incident.AffectedComponents) > 0 {
		names := make([]string, len(incident.AffectedComponents))
		for index, component := range incident.AffectedComponents {
			names[index] = component.Name
		}
		lines = append(lines, indent+mutedStyle.Render("Affects: "+strings.Join(names, ", ")))
	}
	return strings.Join(append(lines, indent+mutedStyle.Render(incident.URL)), "\n")
}

// renderComponents lists the components under their provider and group, each with its status
// label at the right edge of a pane width cells wide.
func renderComponents(components []api.Component, width int) string {
	if len(components) == 0 {
		return noComponentsLabel
	}

	var lines []string
	previousProvider := ""
	for _, group := range groupComponents(components) {
		if group.provider != previousProvider {
			lines = append(lines, titleStyle.Render(group.provider))
			previousProvider = group.provider
		}
		memberIndent := indent
		if group.groupName != "" {
			lines = append(lines, indent+mutedStyle.Render(group.groupName))
			memberIndent = indent + indent
		}
		for _, component := range group.members {
			lines = append(lines, componentRow(component, memberIndent, width))
		}
	}
	return strings.Join(lines, "\n")
}

// componentRow is "● Name" and the status label pushed to the right edge, in the status colour.
func componentRow(component api.Component, memberIndent string, width int) string {
	appearance := appearanceOf(component.Status)
	gap := max(width-lipgloss.Width(memberIndent+bullet+component.Name)-lipgloss.Width(appearance.label), minComponentGap)
	return memberIndent + appearance.style.Render(bullet) + component.Name + strings.Repeat(" ", gap) + appearance.style.Render(appearance.label)
}

// renderIncidents lists the active incidents, one after the other.
func renderIncidents(incidents []api.Incident, now time.Time) string {
	if len(incidents) == 0 {
		return noIncidentsLabel
	}
	var rendered []string
	for _, incident := range incidents {
		rendered = append(rendered, renderIncident(incident, now))
	}
	return strings.Join(rendered, "\n\n")
}

// impactRank orders the incident impact levels from mild to severe.
var impactRank = []string{"none", "minor", "major", "critical"}

// incidentsPanelLook tints the incidents panel by the worst active incident: its border and
// title take that impact's colour. Without incidents it looks like any other pane. Focus shows
// as a bold title, so the tint is never replaced by the focus colour.
func incidentsPanelLook(incidents []api.Incident, isFocused bool) panelLook {
	worstRank := -1
	for _, incident := range incidents {
		if incident.Impact != nil {
			worstRank = max(worstRank, slices.Index(impactRank, *incident.Impact))
		}
	}
	if worstRank < 0 {
		return untintedPanelLook(isFocused)
	}
	tint := impactStyles[impactRank[worstRank]]
	return panelLook{border: tint, title: tint.Bold(isFocused)}
}
