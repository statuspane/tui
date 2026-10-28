package ui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

// historyDayCount is how many days of history the screen asks for: the strip shows as many of them
// as fit.
const historyDayCount = 30

// AppHistoryGetter is what the apps screen needs to draw the selected app's history.
type AppHistoryGetter interface {
	GetAppHistory(ctx context.Context, slug string, days int) (api.AppHistory, error)
}

// selectedHistoryLoadedMsg and selectedHistoryFailedMsg carry the slug they were asked for: the
// history's own answer does not name the app.
type selectedHistoryLoadedMsg struct {
	slug       string
	appHistory api.AppHistory
}

type selectedHistoryFailedMsg struct{ slug string }

// historyState is whether an app's history has arrived, failed to, or is still on its way.
type historyState int

const (
	historyLoading historyState = iota
	historyLoaded
	historyFailed
)

// selectedHistory is what the apps screen knows about the history of the apps its selection has
// been on, by slug. A history that has arrived is kept, so going back to an app shows it at once.
type selectedHistory struct {
	appHistoryBySlug map[string]api.AppHistory
	// failedSlugs holds the apps whose history could not be fetched and has none to show.
	failedSlugs map[string]bool
}

func newSelectedHistory() selectedHistory {
	return selectedHistory{appHistoryBySlug: map[string]api.AppHistory{}, failedSlugs: map[string]bool{}}
}

// of is the app's history, and whether it has arrived, failed or is still on its way. An app keeps
// the history it had when a later request fails, so a refresh cannot take it away.
func (h selectedHistory) of(app api.App) (api.AppHistory, historyState) {
	if appHistory, hasHistory := h.appHistoryBySlug[app.Slug]; hasHistory {
		return appHistory, historyLoaded
	}
	if h.failedSlugs[app.Slug] {
		return api.AppHistory{}, historyFailed
	}
	return api.AppHistory{}, historyLoading
}

func (h selectedHistory) withLoaded(msg selectedHistoryLoadedMsg) selectedHistory {
	h.appHistoryBySlug[msg.slug] = msg.appHistory
	delete(h.failedSlugs, msg.slug)
	return h
}

func (h selectedHistory) withFailed(msg selectedHistoryFailedMsg) selectedHistory {
	if _, hasHistory := h.appHistoryBySlug[msg.slug]; !hasHistory {
		h.failedSlugs[msg.slug] = true
	}
	return h
}

// fetchSelectedAppHistory asks for the app's last historyDayCount days. It is a request of its own,
// so a slow or failed one leaves the banner, the incident and the components as they are.
func fetchSelectedAppHistory(historyGetter AppHistoryGetter, app api.App) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		appHistory, err := historyGetter.GetAppHistory(ctx, app.Slug, historyDayCount)
		if err != nil {
			return selectedHistoryFailedMsg{slug: app.Slug}
		}
		return selectedHistoryLoadedMsg{slug: app.Slug, appHistory: appHistory}
	}
}
