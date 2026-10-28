package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/statuspane/tui/internal/api"
)

const (
	affectedPanelTitle = "Affected components"

	// Under the status history the affected panel keeps at least its borders, two rows and the line
	// that counts the operational components.
	minAffectedRows        = 2
	operationalCountLines  = 1
	minAffectedPanelHeight = panelChromeHeight + minAffectedRows + operationalCountLines
)

// hasAffectedComponents is whether the app's detail is in and some component is not operational.
func (m AppsModel) hasAffectedComponents(app api.App) bool {
	appDetail, hasDetail := m.selectedDetail.of(app)
	if !hasDetail {
		return false
	}
	affected, _ := affectedComponents(appDetail.WatchedComponents)
	return len(affected) > 0
}

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
	rowRoom := contentHeight - operationalCountLines
	if rowRoom < 1 {
		return ""
	}

	shown, leftOut := rowsThatFit(affected, rowRoom)
	lines := make([]string, 0, contentHeight)
	for _, component := range shown {
		lines = append(lines, affectedRow(component, contentWidth))
	}
	if leftOut > 0 {
		lines = append(lines, mutedStyle.Render(fmt.Sprintf("… and %d more", leftOut)))
	}
	lines = append(lines, mutedStyle.Render(fmt.Sprintf("%d operational", operationalCount)))
	return panel(affectedPanelTitle, strings.Join(lines, "\n"), width, len(lines)+panelChromeHeight)
}

// rowsThatFit is the components that get a row when only rowRoom rows are free, and how many are
// left out. When some are, the last free row says so, so one fewer component is shown.
func rowsThatFit(affected []api.Component, rowRoom int) (shown []api.Component, leftOut int) {
	if len(affected) <= rowRoom {
		return affected, 0
	}
	shown = affected[:rowRoom-1]
	return shown, len(affected) - len(shown)
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
