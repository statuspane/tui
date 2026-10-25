package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// clockInterval is how often the header's clock is redrawn.
const clockInterval = time.Second

// clockTickMsg redraws the header's clock; each one arms the next.
type clockTickMsg struct{}

func clockTick() tea.Cmd {
	return tea.Tick(clockInterval, func(time.Time) tea.Msg { return clockTickMsg{} })
}

// clockText is the time where the user is, with its zone, and whether the screen is keeping up:
// "live" while refreshes succeed, and a warning once one fails, since the figures on screen are
// then getting old.
func (m AppsModel) clockText() string {
	text := mutedStyle.Render(m.now().Format("15:04:05 MST"))
	switch {
	case m.err != nil && m.hasLoaded:
		return text + mutedStyle.Render(" · ") + errorStyle.Render("not refreshing")
	case m.hasLoaded:
		return text + mutedStyle.Render(" · ") + watchedStyle.Render("live")
	}
	return text
}
