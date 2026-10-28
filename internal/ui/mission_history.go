package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

const (
	// A day is one block and a space before the next one. The no-data dot takes the block's width.
	dayCell         = "▇"
	noDataCell      = "·"
	dayCellGap      = " "
	dayCellWidth    = 1
	columnsPerDay   = dayCellWidth + len(dayCellGap)
	todayLabel      = "today"
	noHistoryText   = "No history yet"
	dateLabelLayout = "Jan 2"

	historyPanelTitle = "Status history"
	// The panel's body is the strip, the dates under it and when tracking began.
	historyPanelLines  = 3
	historyPanelHeight = historyPanelLines + panelChromeHeight

	loadingHistory     = "Loading…"
	historyUnavailable = "History unavailable"
)

// historyPanel is the app's last days as a strip, or why there is none yet. It is as tall as it is
// for every state, so the column does not jump when the history arrives.
func (m AppsModel) historyPanel(app api.App, width int) string {
	contentWidth, _ := panelContentSize(width, historyPanelHeight)
	appHistory, state := m.selectedHistory.of(app)
	switch state {
	case historyLoaded:
		shownDays := mostRecentDays(appHistory.Days, contentWidth)
		return panel(fmt.Sprintf("Last %d days", len(shownDays)), historyPanelBody(shownDays, appHistory.TrackingSince), width, historyPanelHeight)
	case historyFailed:
		return panel(historyPanelTitle, mutedStyle.Render(historyUnavailable), width, historyPanelHeight)
	}
	return panel(historyPanelTitle, mutedStyle.Render(loadingHistory), width, historyPanelHeight)
}

// mostRecentDays is the days that fit contentWidth, the newest ones when not all do. The last day
// needs no gap after it.
func mostRecentDays(days []api.HistoryDay, contentWidth int) []api.HistoryDay {
	fittingDayCount := min(len(days), (contentWidth+len(dayCellGap))/columnsPerDay)
	return days[len(days)-fittingDayCount:]
}

// historyPanelBody is the strip, the dates it covers under it and when tracking began.
func historyPanelBody(shownDays []api.HistoryDay, trackingSince *time.Time) string {
	return strings.Join([]string{
		historyCells(shownDays),
		mutedStyle.Render(historyRangeLabels(shownDays)),
		mutedStyle.Render(historyCaption(trackingSince)),
	}, "\n")
}

// historyCells is a cell per day, oldest first, coloured by the day's status. A day with no data
// is a dot in the muted colour, so it never looks like an operational day and reads without colour.
func historyCells(shownDays []api.HistoryDay) string {
	renderedCells := make([]string, len(shownDays))
	for index, day := range shownDays {
		if day.Status == nil {
			renderedCells[index] = mutedStyle.Render(noDataCell)
			continue
		}
		renderedCells[index] = appearanceOf(day.Status).style.Render(dayCell)
	}
	return strings.Join(renderedCells, dayCellGap)
}

// historyRangeLabels puts the date the strip starts on under its left end and "today" under its
// right. A date that cannot be read is said as how many days back the strip goes.
func historyRangeLabels(shownDays []api.HistoryDay) string {
	if len(shownDays) == 0 {
		return ""
	}
	left := fmt.Sprintf("%d days ago", len(shownDays))
	if firstDay, err := time.Parse(time.DateOnly, shownDays[0].Date); err == nil {
		left = firstDay.Format(dateLabelLayout)
	}
	stripWidth := len(shownDays)*columnsPerDay - len(dayCellGap)
	return left + strings.Repeat(" ", max(stripWidth-lipgloss.Width(left)-len(todayLabel), 1)) + todayLabel
}

// historyCaption says when StatusPane began recording, in UTC like the days themselves.
func historyCaption(trackingSince *time.Time) string {
	if trackingSince == nil {
		return noHistoryText
	}
	return "Tracking since " + trackingSince.UTC().Format(dateLabelLayout)
}
