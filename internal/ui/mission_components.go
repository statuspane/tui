package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/statuspane/tui/internal/api"
)

const affectedPanelTitle = "Affected components"

// affectedPanel lists the app's components that are not operational, the worst first, and counts
// the operational ones under them. There is none when nothing is affected or the detail is not in.
func (m AppsModel) affectedPanel(app api.App, width, maxHeight int) string {
	appDetail, hasDetail := m.selectedDetail.of(app)
	if !hasDetail {
		return ""
	}
	affected, operationalCount := affectedComponents(appDetail.WatchedComponents)
	if len(affected) == 0 {
		return ""
	}

	contentWidth, contentHeight := panelContentSize(width, maxHeight)
	rowRoom := contentHeight - 1 // the line that counts the operational ones
	if rowRoom < 1 {
		return ""
	}

	lines := make([]string, 0, contentHeight)
	for _, component := range affected[:min(len(affected), rowRoom)] {
		lines = append(lines, affectedRow(component, contentWidth))
	}
	if hidden := len(affected) - rowRoom; hidden > 0 {
		lines[len(lines)-1] = mutedStyle.Render(fmt.Sprintf("… and %d more", hidden+1))
	}
	lines = append(lines, mutedStyle.Render(fmt.Sprintf("%d operational", operationalCount)))
	return panel(affectedPanelTitle, strings.Join(lines, "\n"), width, len(lines)+panelChromeHeight)
}

// affectedComponents are the components whose status is known and not operational, worst first and
// otherwise in the order given, and how many are operational.
func affectedComponents(components []api.Component) (affected []api.Component, operationalCount int) {
	for _, component := range components {
		switch {
		case isAffected(component.Status):
			affected = append(affected, component)
		case component.Status != nil:
			operationalCount++
		}
	}
	slices.SortStableFunc(affected, func(left, right api.Component) int {
		return summaryRank(right.Status) - summaryRank(left.Status)
	})
	return affected, operationalCount
}

// affectedRow is "● provider / Name" and, at the right edge, its status.
func affectedRow(component api.Component, contentWidth int) string {
	appearance := appearanceOf(component.Status)
	left := appearance.style.Render(statusDot) + " " + mutedStyle.Render(component.Provider+" / ") + component.Name
	return headerBar(left, appearance.style.Render(appearance.label), contentWidth)
}
