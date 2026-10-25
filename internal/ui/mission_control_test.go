package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

func missionControlModel(apps ...api.App) AppsModel {
	m := newTestAppsModel()
	m, _ = send(m, tea.WindowSizeMsg{Width: testWidth, Height: testHeight})
	m, _ = send(m, appsLoadedMsg{apps: apps})
	return m
}

func TestTheSidebarListsTheAppsAndTheCentreHeadlinesTheSelectedOne(t *testing.T) {
	m := missionControlModel(
		api.App{Name: "Web App", Slug: "web-app", Status: stringPtr("degraded_performance"), WatchedComponentsCount: 12},
		api.App{Name: "Payments", Slug: "payments", Status: stringPtr("operational"), WatchedComponentsCount: 4},
	)

	view := m.View().Content

	assertViewContains(t, view, "Web App", "Payments", "DEGRADED PERFORMANCE")
	if got := plain(t, view); strings.Contains(got, "ALL SYSTEMS OPERATIONAL") {
		t.Errorf("only the selected app is headlined:\n%s", got)
	}

	m = pressRune(m, 'j')

	assertViewContains(t, m.View().Content, "ALL SYSTEMS OPERATIONAL")
}

// settledApps feeds a model the answers to its commands. The spinner's ticks are left out (each
// starts the next), and so are the timers, which collectMessages does not wait for.
func settledApps(m AppsModel, cmd tea.Cmd) AppsModel {
	for _, msg := range collectMessages(cmd) {
		if _, isSpinnerTick := msg.(spinner.TickMsg); isSpinnerTick {
			continue
		}
		var next tea.Cmd
		m, next = send(m, msg)
		m = settledApps(m, next)
	}
	return m
}

func TestTheBannerCountsTheAffectedComponentsOfTheSelectedAppAndSaysWhenItWasChecked(t *testing.T) {
	backend := fakeAppsBackend{
		fakeAppLister: fakeAppLister{apps: []api.App{{Name: "Web App", Slug: "web-app", Status: stringPtr("degraded_performance"), WatchedComponentsCount: 2}}},
		fakeAppGetter: fakeAppGetter{appDetail: degradedAppDetail()},
	}
	m := NewAppsModel(backend)
	m.now = func() time.Time { return detailNow }
	m, _ = send(m, tea.WindowSizeMsg{Width: testWidth, Height: testHeight})

	m = settledApps(m, m.Init())

	assertViewContains(t, m.View().Content, "1 of 2 components affected", "checked 3m ago")
}

func selectionBackend(asked *string, apps ...api.App) fakeAppsBackend {
	return fakeAppsBackend{
		fakeAppLister: fakeAppLister{apps: apps},
		fakeAppGetter: fakeAppGetter{appDetail: degradedAppDetail(), gotSlug: asked},
	}
}

func threeApps() []api.App {
	return []api.App{{Name: "Web App", Slug: "web-app"}, {Name: "Payments", Slug: "payments"}, {Name: "Mobile", Slug: "mobile"}}
}

func TestMovingTheSelectionLoadsTheNewAppOnlyOnceItHasRested(t *testing.T) {
	var asked string
	m := NewAppsModel(selectionBackend(&asked, threeApps()...))
	m = settledApps(m, m.Init())

	m, cmd := send(m, keyJ)
	m = settledApps(m, cmd)
	if asked != "web-app" {
		t.Fatalf("moving should not ask at once, but %q was asked for", asked)
	}
	m, cmd = send(m, keyJ)
	m = settledApps(m, cmd)
	m, cmd = send(m, detailSettledMsg{sequence: m.detailSequence - 1})
	m = settledApps(m, cmd)
	if asked != "web-app" {
		t.Fatalf("a pause that was interrupted must not ask, but %q was asked for", asked)
	}

	m, cmd = send(m, detailSettledMsg{sequence: m.detailSequence})
	settledApps(m, cmd)

	if asked != "mobile" {
		t.Errorf("once the selection rests, its app is asked for; got %q", asked)
	}
}

func TestAnAppsDetailShowsOnlyOnItsOwnBannerAndIsKeptWhenYouComeBack(t *testing.T) {
	var asked string
	apps := []api.App{
		{Name: "Web App", Slug: "web-app", Status: stringPtr("degraded_performance"), WatchedComponentsCount: 2},
		{Name: "Payments", Slug: "payments", Status: stringPtr("operational"), WatchedComponentsCount: 7},
	}
	m := NewAppsModel(selectionBackend(&asked, apps...))
	m.now = func() time.Time { return detailNow }
	m, _ = send(m, tea.WindowSizeMsg{Width: testWidth, Height: testHeight})
	m = settledApps(m, m.Init())

	m, _ = send(m, keyJ)

	view := plain(t, m.View().Content)
	if !strings.Contains(view, "7 components") || strings.Contains(view, "affected") {
		t.Errorf("Payments has no detail yet, so the banner shows the list's count:\n%s", view)
	}

	m, _ = send(m, keyK)

	assertViewContains(t, m.View().Content, "1 of 2 components affected")
}

func TestAFailedDetailRequestLeavesTheBannerOnTheListsCount(t *testing.T) {
	backend := fakeAppsBackend{
		fakeAppLister: fakeAppLister{apps: []api.App{{Name: "Web App", Slug: "web-app", Status: stringPtr("operational"), WatchedComponentsCount: 5}}},
		fakeAppGetter: fakeAppGetter{err: errors.New("boom")},
	}
	m := NewAppsModel(backend)
	m, _ = send(m, tea.WindowSizeMsg{Width: testWidth, Height: testHeight})

	m = settledApps(m, m.Init())

	view := plain(t, m.View().Content)
	assertViewContains(t, view, "5 components", "Web App")
	if strings.Contains(view, "boom") || strings.Contains(view, "failed") {
		t.Errorf("the banner can do without the detail, so a failure is not shown:\n%s", view)
	}
}
