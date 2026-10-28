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
	dayCell       = "▇"
	noDataCell    = "·"
	dayCellGap    = " "
	dayCellWidth  = 1
	cellsPerDay   = dayCellWidth + len(dayCellGap)
	todayLabel    = "today"
	noHistoryText = "No history yet"

	historyPanelLines  = 3
	historyPanelHeight = historyPanelLines + panelChromeHeight

	loadingHistory     = "Loading…"
	historyUnavailable = "History unavailable"
)

// historyPanel is the app's last days as a strip, or why there is none yet. It is as tall as it is
// for every state, so the column does not jump when the history arrives.
func (m AppsModel) historyPanel(app api.App, width int) string {
	contentWidth, _ := panelContentSize(width, historyPanelHeight)
	history, state := m.selectedDetail.historyOf(app)
	switch state {
	case historyLoaded:
		body, shownDays := historyPanelBody(history, contentWidth)
		return panel(fmt.Sprintf("Last %d days", shownDays), body, width, historyPanelHeight)
	case historyFailed:
		return panel(historyPanelTitle, mutedStyle.Render(historyUnavailable), width, historyPanelHeight)
	}
	return panel(historyPanelTitle, mutedStyle.Render(loadingHistory), width, historyPanelHeight)
}

const historyPanelTitle = "Status history"

// historyPanelBody is the strip, the dates it covers under it and when tracking began, and how
// many days it shows: the most recent ones that fit contentWidth.
func historyPanelBody(history api.AppHistory, contentWidth int) (body string, shownDays int) {
	cells, shownDays := historyCells(history.Days, contentWidth)
	firstShownDay := history.Days[len(history.Days)-shownDays:]
	return strings.Join([]string{
		cells,
		mutedStyle.Render(historyRangeLabels(firstShownDay, shownDays)),
		mutedStyle.Render(historyCaption(history.TrackingSince)),
	}, "\n"), shownDays
}

// historyCells is a cell per day, oldest first, coloured by the day's status. A day with no data
// is dots in the muted colour, so it never looks like an operational day and reads without colour.
func historyCells(days []api.HistoryDay, contentWidth int) (cells string, shownDays int) {
	shownDays = min(len(days), (contentWidth+len(dayCellGap))/cellsPerDay)
	shown := days[len(days)-shownDays:]

	rendered := make([]string, len(shown))
	for index, day := range shown {
		if day.Status == nil {
			rendered[index] = mutedStyle.Render(noDataCell)
			continue
		}
		rendered[index] = appearanceOf(day.Status).style.Render(dayCell)
	}
	return strings.Join(rendered, dayCellGap), shownDays
}

// historyRangeLabels puts the date the strip starts on under its left end and "today" under its
// right. A date that cannot be read is said as how many days back the strip goes.
func historyRangeLabels(shownDays []api.HistoryDay, dayCount int) string {
	if dayCount == 0 {
		return ""
	}
	left := fmt.Sprintf("%d days ago", dayCount)
	if firstDay, err := time.Parse(time.DateOnly, shownDays[0].Date); err == nil {
		left = firstDay.Format("Jan 2")
	}
	stripWidth := dayCount*cellsPerDay - len(dayCellGap)
	return left + strings.Repeat(" ", max(stripWidth-lipgloss.Width(left)-len(todayLabel), 1)) + todayLabel
}

// historyCaption says when StatusPane began recording, in UTC like the days themselves.
func historyCaption(trackingSince *time.Time) string {
	if trackingSince == nil {
		return noHistoryText
	}
	return "Tracking since " + trackingSince.UTC().Format("Jan 2")
}
