package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

const (
	// The sidebar takes this share of the width, within these limits.
	sidebarShareOfWidth = 4
	sidebarMinWidth     = 28
	sidebarMaxWidth     = 40

	statusDot             = "●"
	sidebarSelectedMark   = "▎"
	sidebarUnselectedMark = " "
	appsPanelName         = "Apps"
)

// hasRoomForHistory is whether the status history fits in the room left under the incident without
// taking what the affected components (when there are any) need: History is the first thing to
// give way on a short window.
func hasRoomForHistory(roomAfterIncident int, hasAffected bool) bool {
	affectedNeed := 0
	if hasAffected {
		affectedNeed = minAffectedPanelHeight
	}
	return roomAfterIncident-historyPanelHeight >= affectedNeed
}

// bannerHeadlines are what a banner says for each status. A status this client doesn't know is
// said as it is, in capitals.
var bannerHeadlines = map[string]string{
	"operational":          "ALL SYSTEMS OPERATIONAL",
	"under_maintenance":    "UNDER MAINTENANCE",
	"degraded_performance": "DEGRADED PERFORMANCE",
	"partial_outage":       "PARTIAL OUTAGE",
	"major_outage":         "MAJOR OUTAGE",
}

// pendingHeadline is what a banner says before any check has run.
const pendingHeadline = "NOT CHECKED YET"

func headlineOf(status *string) string {
	if status == nil {
		return pendingHeadline
	}
	if headline, ok := bannerHeadlines[*status]; ok {
		return headline
	}
	return strings.ToUpper(*status)
}

// missionControl is the sidebar of apps and, beside it, what the selected app is doing.
func (m AppsModel) missionControl(width, height int) string {
	if !m.hasLoaded || len(m.apps) == 0 {
		return m.sidebar(width, height)
	}
	sidebarWidth := min(max(width/sidebarShareOfWidth, sidebarMinWidth), sidebarMaxWidth)
	centreWidth := width - sidebarWidth - panelGapWidth
	return lipgloss.JoinHorizontal(lipgloss.Top,
		m.sidebar(sidebarWidth, height), strings.Repeat(" ", panelGapWidth), m.selectedAppPanels(centreWidth, height))
}

func (m AppsModel) sidebar(width, height int) string {
	contentWidth, contentHeight := panelContentSize(width, height)
	return panel(appsPanelName, m.sidebarBody(contentWidth, contentHeight), width, height)
}

// sidebarBody is a line per app, scrolled to keep the selected one in view, or why there is none,
// and under it the notice of a refresh when apps are already on screen.
func (m AppsModel) sidebarBody(contentWidth, lineCount int) string {
	switch {
	case !m.hasLoaded && m.err != nil:
		return errorStyle.Render(errorMessage(m.err))
	case !m.hasLoaded:
		return m.spinner.View() + " Loading apps…"
	case len(m.apps) == 0:
		return "No apps yet. Press n to create one."
	}

	var noticeLines []string
	if notice := m.refreshNotice(); notice != "" {
		noticeLines = strings.Split(wrapText(notice, contentWidth), "\n")
	}
	start, end := visibleRange(len(m.apps), m.selectedIndex, max(lineCount-len(noticeLines), 0))
	lines := make([]string, 0, lineCount)
	for appIndex := start; appIndex < end; appIndex++ {
		lines = append(lines, m.sidebarRow(appIndex, contentWidth))
	}
	return strings.Join(append(lines, noticeLines...), "\n")
}

// sidebarRow is the app's status dot and name. The selected row is tinted and bold, the tint set
// on every piece because a style nested inside another one ends the outer one early.
func (m AppsModel) sidebarRow(appIndex, contentWidth int) string {
	app := m.apps[appIndex]
	dotStyle := appearanceOf(app.Status).style
	if appIndex != m.selectedIndex {
		return sidebarUnselectedMark + dotStyle.Render(statusDot) + " " + clipTo(app.Name, contentWidth-lipgloss.Width(sidebarUnselectedMark+statusDot+" "))
	}

	tint := rowTintColor(m.hasDarkBackground)
	selectedRowStyle := lipgloss.NewStyle().Background(tint)
	row := accentStyle.Background(tint).Render(sidebarSelectedMark) + dotStyle.Background(tint).Render(statusDot) + selectedRowStyle.Render(" ") +
		selectedRowStyle.Bold(true).Render(clipTo(app.Name, contentWidth-lipgloss.Width(sidebarSelectedMark+statusDot+" ")))
	return row + selectedRowStyle.Render(strings.Repeat(" ", max(contentWidth-lipgloss.Width(row), 0)))
}

// selectedAppPanels are the selected app's banner, status history, incident and affected components,
// stacked.
func (m AppsModel) selectedAppPanels(width, height int) string {
	app, ok := m.selectedApp()
	if !ok {
		return panel("", "", width, height)
	}
	banner := m.banner(app, width)
	room := height - lipgloss.Height(banner)
	incident := m.incidentPanel(app, width, room)

	panels := []string{banner}
	if hasRoomForHistory(room-lipgloss.Height(incident), m.hasAffectedComponents(app)) {
		strip := m.historyPanel(app, width)
		panels = append(panels, strip)
		room -= lipgloss.Height(strip)
	}
	panels = append(panels, incident)
	room -= lipgloss.Height(incident)
	if affected := m.affectedPanel(app, width, room); affected != "" {
		panels = append(panels, affected)
	}
	return lipgloss.JoinVertical(lipgloss.Left, panels...)
}

// banner says in one headline how the app is doing, tinted by its status.
func (m AppsModel) banner(app api.App, width int) string {
	appearance := appearanceOf(app.Status)
	look := untintedPanelLook(false)
	if isAffected(app.Status) {
		look = panelLook{border: appearance.style.Faint(true), title: appearance.style.Bold(true)}
	}
	contentWidth, _ := panelContentSize(width, 0)
	body := appearance.style.Bold(true).Render(statusDot+"  "+headlineOf(app.Status)) + "\n" +
		mutedStyle.Render(factsLine(m.bannerFacts(app), contentWidth))
	return panelWithLook(look, app.Name, body, width, lipgloss.Height(body)+panelChromeHeight)
}

// factsLine puts the facts on one line, and each on a line of its own when they do not fit: the
// facts are short phrases, which read badly when a wrap splits one.
func factsLine(facts []string, width int) string {
	if line := strings.Join(facts, " · "); lipgloss.Width(line) <= width {
		return line
	}
	return strings.Join(facts, "\n")
}

// bannerFacts are how many of the app's components are affected and when it was checked, once the
// app's detail is in; until then the list's own count of components.
func (m AppsModel) bannerFacts(app api.App) []string {
	appDetail, hasDetail := m.selectedDetail.of(app)
	if !hasDetail {
		return []string{fmt.Sprintf("%d components", app.WatchedComponentsCount)}
	}
	affected, _ := affectedComponents(appDetail.WatchedComponents)
	return []string{fmt.Sprintf("%d of %d components affected", len(affected), len(appDetail.WatchedComponents)), checkedText(appDetail.LastCheckedAt, m.now())}
}
