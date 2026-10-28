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

// selectedDetailLoadedMsg is the detail of the app the sidebar had selected. It is not
// appDetailLoadedMsg, which the app screen would take for its own.
type selectedDetailLoadedMsg struct{ appDetail api.AppDetail }

// selectedDetail is what the apps screen knows about the apps its selection has been on: their
// details, by slug, and how many times the selection has moved. A detail that has arrived is kept,
// so going back to an app shows it at once.
type selectedDetail struct {
	appDetailBySlug map[string]api.AppDetail
	sequence        int
}

func newSelectedDetail() selectedDetail {
	return selectedDetail{appDetailBySlug: map[string]api.AppDetail{}}
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
