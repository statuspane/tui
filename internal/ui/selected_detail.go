package ui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

// detailSettleDelay is how long the selection has to rest on an app before its detail is asked
// for, so moving down a long list does not spend the API's request limit.
const detailSettleDelay = 300 * time.Millisecond

// detailSettledMsg says the selection stayed put for detailSettleDelay. Its sequence number tells a
// pause that was interrupted by another move from the last one.
type detailSettledMsg struct{ sequence int }

// historyDayCount is how many days of history the screen asks for: the strip shows as many of them
// as fit.
const historyDayCount = 30

// selectedHistoryLoadedMsg and selectedHistoryFailedMsg carry the slug they were asked for: the
// history's own answer does not name the app.
type selectedHistoryLoadedMsg struct {
	slug    string
	history api.AppHistory
}

type selectedHistoryFailedMsg struct{ slug string }

// historyState is whether an app's history has arrived, failed to, or is still on its way.
type historyState int

const (
	historyLoading historyState = iota
	historyLoaded
	historyFailed
)

// selectedDetailLoadedMsg is the detail of the app the sidebar had selected. It is not
// appDetailLoadedMsg, which the app screen would take for its own.
type selectedDetailLoadedMsg struct{ appDetail api.AppDetail }

// selectedDetail is what the apps screen knows about the apps its selection has been on: their
// details, by slug, and how many times the selection has moved. A detail that has arrived is kept,
// so going back to an app shows it at once.
type selectedDetail struct {
	appDetailBySlug map[string]api.AppDetail
	historyBySlug   map[string]api.AppHistory
	// historyFailedSlugs holds the apps whose history could not be fetched and has none to show.
	historyFailedSlugs map[string]bool
	sequence           int
}

func newSelectedDetail() selectedDetail {
	return selectedDetail{
		appDetailBySlug:    map[string]api.AppDetail{},
		historyBySlug:      map[string]api.AppHistory{},
		historyFailedSlugs: map[string]bool{},
	}
}

// historyOf is the app's history, and whether it has arrived, failed or is still on its way. An
// app keeps the history it had when a later request fails, so a refresh cannot take it away.
func (d selectedDetail) historyOf(app api.App) (api.AppHistory, historyState) {
	if history, hasHistory := d.historyBySlug[app.Slug]; hasHistory {
		return history, historyLoaded
	}
	if d.historyFailedSlugs[app.Slug] {
		return api.AppHistory{}, historyFailed
	}
	return api.AppHistory{}, historyLoading
}

func (d selectedDetail) withHistoryLoaded(msg selectedHistoryLoadedMsg) selectedDetail {
	d.historyBySlug[msg.slug] = msg.history
	delete(d.historyFailedSlugs, msg.slug)
	return d
}

func (d selectedDetail) withHistoryFailed(msg selectedHistoryFailedMsg) selectedDetail {
	if _, hasHistory := d.historyBySlug[msg.slug]; !hasHistory {
		d.historyFailedSlugs[msg.slug] = true
	}
	return d
}

// of is the app's detail, and whether it has arrived.
func (d selectedDetail) of(app api.App) (api.AppDetail, bool) {
	appDetail, hasDetail := d.appDetailBySlug[app.Slug]
	return appDetail, hasDetail
}

func (d selectedDetail) withLoaded(msg selectedDetailLoadedMsg) selectedDetail {
	d.appDetailBySlug[msg.appDetail.Slug] = msg.appDetail
	return d
}

// afterSelectionMoved counts the move and starts the wait for the selection to rest.
func (d selectedDetail) afterSelectionMoved() (selectedDetail, tea.Cmd) {
	d.sequence++
	sequence := d.sequence
	return d, tea.Tick(detailSettleDelay, func(time.Time) tea.Msg { return detailSettledMsg{sequence: sequence} })
}

// hasRested is whether no move has happened since the pause that sent msg began.
func (d selectedDetail) hasRested(msg detailSettledMsg) bool {
	return msg.sequence == d.sequence
}

// fetchSelectedAppDetail asks for the app's detail; a failure is left unsaid, because the banner
// reads fine from the list's own figures.
func fetchSelectedAppDetail(appGetter AppGetter, app api.App) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		appDetail, err := appGetter.GetApp(ctx, app.Slug)
		if err != nil {
			return nil
		}
		return selectedDetailLoadedMsg{appDetail: appDetail}
	}
}

// fetchSelectedAppHistory asks for the app's last historyDayCount days. It is a request of its own,
// so a slow or failed one leaves the banner, the incident and the components as they are.
func fetchSelectedAppHistory(historyGetter AppHistoryGetter, app api.App) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		history, err := historyGetter.GetAppHistory(ctx, app.Slug, historyDayCount)
		if err != nil {
			return selectedHistoryFailedMsg{slug: app.Slug}
		}
		return selectedHistoryLoadedMsg{slug: app.Slug, history: history}
	}
}
