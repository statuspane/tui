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

	// A banner is its two lines in a box.
	bannerLines  = 2
	bannerHeight = bannerLines + panelChromeHeight

	statusDot     = "●"
	selectedMark  = "▎"
	noSelectMark  = " "
	appsPanelName = "Apps"
)

// body is the sidebar of apps and, beside it, what the selected app is doing.
func (m AppsModel) body(width, height int) string {
	if !m.hasLoaded || len(m.apps) == 0 {
		contentWidth, contentHeight := panelContentSize(width, height)
		return panel(appsPanelName, m.sidebarBody(contentWidth, contentHeight), width, height)
	}
	sidebarWidth := min(max(width/sidebarShareOfWidth, sidebarMinWidth), sidebarMaxWidth)
	centreWidth := width - sidebarWidth - panelGapWidth
	return lipgloss.JoinHorizontal(lipgloss.Top,
		m.sidebar(sidebarWidth, height), strings.Repeat(" ", panelGapWidth), m.centre(centreWidth, height))
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
		noticeLines = strings.Split(lipgloss.NewStyle().Width(contentWidth).Render(notice), "\n")
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
		return noSelectMark + dotStyle.Render(statusDot) + " " + clipTo(app.Name, contentWidth-lipgloss.Width(noSelectMark+statusDot+" "))
	}

	tint := rowTintColor(m.hasDarkBackground)
	tinted := lipgloss.NewStyle().Background(tint)
	row := accentStyle.Background(tint).Render(selectedMark) + dotStyle.Background(tint).Render(statusDot) + tinted.Render(" ") +
		tinted.Bold(true).Render(clipTo(app.Name, contentWidth-lipgloss.Width(selectedMark+statusDot+" ")))
	return row + tinted.Render(strings.Repeat(" ", max(contentWidth-lipgloss.Width(row), 0)))
}

func (m AppsModel) centre(width, height int) string {
	app, ok := m.selectedApp()
	if !ok {
		return panel("", "", width, height)
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.banner(app, width))
}

// banner says in one headline how the app is doing, tinted by its status.
func (m AppsModel) banner(app api.App, width int) string {
	appearance := appearanceOf(app.Status)
	look := untintedPanelLook(false)
	if app.Status != nil && *app.Status != "operational" {
		look = panelLook{border: appearance.style.Faint(true), title: appearance.style.Bold(true)}
	}
	body := appearance.style.Bold(true).Render(statusDot+"  "+appearance.headline) + "\n" +
		mutedStyle.Render(m.bannerDetail(app))
	return panelWithLook(look, app.Name, body, width, bannerHeight)
}

// bannerDetail is how many of the app's components are affected and when it was checked, once the
// app's detail is in; until then the list's own count of components.
func (m AppsModel) bannerDetail(app api.App) string {
	appDetail, hasDetail := m.details[app.Slug]
	if !hasDetail {
		return fmt.Sprintf("%d components", app.WatchedComponentsCount)
	}
	affected := 0
	for _, component := range appDetail.WatchedComponents {
		if component.Status != nil && *component.Status != "operational" {
			affected++
		}
	}
	return fmt.Sprintf("%d of %d components affected · %s", affected, len(appDetail.WatchedComponents), checkedText(appDetail.LastCheckedAt, m.now()))
}
