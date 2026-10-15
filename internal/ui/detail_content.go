package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/statuspane/tui/internal/api"
)

const (
	bullet           = "● "
	indent           = "  "
	noImpactLabel    = "unrated"
	noCheckYetLabel  = "not checked yet"
	noIncidentsLabel = "No active incidents."
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

func renderAppDetail(appDetail api.AppDetail, now time.Time) string {
	var content strings.Builder
	content.WriteString(titleStyle.Render(appDetail.Name) + "\n")
	content.WriteString(renderStatus(appDetail.Status, 0) + mutedStyle.Render(" · "+checkedText(appDetail.LastCheckedAt, now)) + "\n\n")

	content.WriteString(titleStyle.Render(fmt.Sprintf("Active incidents (%d)", len(appDetail.ActiveIncidents))) + "\n")
	if len(appDetail.ActiveIncidents) == 0 {
		content.WriteString(indent + noIncidentsLabel + "\n")
	}
	for _, incident := range appDetail.ActiveIncidents {
		content.WriteString("\n" + renderIncident(incident, now))
	}

	content.WriteString("\n" + titleStyle.Render(fmt.Sprintf("Watched components (%d)", len(appDetail.WatchedComponents))) + "\n")
	content.WriteString(renderComponents(appDetail.WatchedComponents))
	return content.String()
}

func checkedText(lastCheckedAt *time.Time, now time.Time) string {
	if lastCheckedAt == nil {
		return noCheckYetLabel
	}
	return "checked " + ageText(now, *lastCheckedAt)
}

func renderIncident(incident api.Incident, now time.Time) string {
	impact := noImpactLabel
	impactStyle := mutedStyle
	if incident.Impact != nil {
		impact = *incident.Impact
		if style, ok := impactStyles[impact]; ok {
			impactStyle = style
		}
	}

	details := []string{impact}
	if incident.Status != nil {
		details = append(details, strings.ReplaceAll(*incident.Status, "_", " "))
	}
	if incident.StartedAt != nil {
		details = append(details, "started "+ageText(now, *incident.StartedAt))
	}

	var lines strings.Builder
	lines.WriteString(impactStyle.Render(bullet) + titleStyle.Render(incident.Title) + "\n")
	lines.WriteString(indent + mutedStyle.Render(incident.Provider+" · ") + impactStyle.Render(strings.Join(details, " · ")) + "\n")
	if incident.LatestUpdate != nil && incident.LatestUpdate.Body != "" {
		lines.WriteString(indent + incident.LatestUpdate.Body + "\n")
	}
	if len(incident.AffectedComponents) > 0 {
		names := make([]string, len(incident.AffectedComponents))
		for index, component := range incident.AffectedComponents {
			names[index] = component.Name
		}
		lines.WriteString(indent + mutedStyle.Render("Affects: "+strings.Join(names, ", ")) + "\n")
	}
	lines.WriteString(indent + mutedStyle.Render(incident.URL) + "\n")
	return lines.String()
}

func renderComponents(components []api.Component) string {
	if len(components) == 0 {
		return indent + "This app doesn't watch any components yet.\n"
	}

	nameWidth, statusWidth := 0, 0
	for _, component := range components {
		nameWidth = max(nameWidth, len([]rune(component.Name)))
		statusWidth = max(statusWidth, len([]rune(statusLabel(component.Status))))
	}

	var lines strings.Builder
	previousProvider := ""
	for _, group := range groupComponents(components) {
		if group.provider != previousProvider {
			lines.WriteString(indent + titleStyle.Render(group.provider) + "\n")
			previousProvider = group.provider
		}
		if group.groupName != "" {
			lines.WriteString(indent + indent + mutedStyle.Render(group.groupName) + "\n")
		}
		for _, component := range group.members {
			name := fmt.Sprintf("%-*s", nameWidth, component.Name)
			lines.WriteString(indent + indent + indent + name + columnGap + renderStatus(component.Status, statusWidth) + "\n")
		}
	}
	return lines.String()
}
