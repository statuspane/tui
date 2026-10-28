package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

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

func TestTheBannerCountsTheAffectedComponentsOfTheSelectedAppAndSaysWhenItWasChecked(t *testing.T) {
	backend := fakeAppsBackend{
		fakeAppLister: fakeAppLister{apps: []api.App{{Name: "Web App", Slug: "web-app", Status: stringPtr("degraded_performance"), WatchedComponentsCount: 2}}},
		fakeAppGetter: fakeAppGetter{appDetail: degradedAppDetail()},
	}
	m := NewAppsModel(backend)
	m.now = func() time.Time { return detailNow }
	m, _ = send(m, tea.WindowSizeMsg{Width: testWidth, Height: testHeight})

	m = drainCommands(m, m.Init())

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
	m = drainCommands(m, m.Init())

	m, cmd := send(m, keyJ)
	m = drainCommands(m, cmd)
	if asked != "web-app" {
		t.Fatalf("moving should not ask at once, but %q was asked for", asked)
	}
	m, cmd = send(m, keyJ)
	m = drainCommands(m, cmd)
	m, cmd = send(m, detailSettledMsg{sequence: m.selectedDetail.sequence - 1})
	m = drainCommands(m, cmd)
	if asked != "web-app" {
		t.Fatalf("a pause that was interrupted must not ask, but %q was asked for", asked)
	}

	m, cmd = send(m, detailSettledMsg{sequence: m.selectedDetail.sequence})
	drainCommands(m, cmd)

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
	m = drainCommands(m, m.Init())

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

	m = drainCommands(m, m.Init())

	view := plain(t, m.View().Content)
	assertViewContains(t, view, "5 components", "Web App")
	if strings.Contains(view, "boom") || strings.Contains(view, "failed") {
		t.Errorf("the banner can do without the detail, so a failure is not shown:\n%s", view)
	}
}

func TestInANarrowWindowTheBannerWrapsInsteadOfCuttingItsSecondLine(t *testing.T) {
	m := modelWithDetail(t, degradedAppDetail(), fallbackWidth, testHeight)

	view := plain(t, m.View().Content)

	assertViewContains(t, view, "1 of 2 components affected", "checked 3m ago")
	assertEveryLineHasWidth(t, m.View().Content, fallbackWidth)
}

func TestTheHeaderShowsTheTimeInUTCAndThatTheScreenIsLive(t *testing.T) {
	m := modelWithDetail(t, degradedAppDetail(), testWidth, testHeight)

	assertViewContains(t, m.View().Content, "12:00:00 UTC", "live", "1 degraded")
}

func TestTheClockTicksOnAndSaysWhenRefreshesStopped(t *testing.T) {
	m := modelWithDetail(t, degradedAppDetail(), testWidth, testHeight)
	m.now = func() time.Time { return detailNow.Add(time.Second) }

	m, cmd := send(m, clockTickMsg{})

	if cmd == nil {
		t.Error("the clock should arm its next tick")
	}
	assertViewContains(t, m.View().Content, "12:00:01 UTC")

	m, _ = send(m, appsFailedMsg{err: errors.New("boom")})

	view := plain(t, m.View().Content)
	assertViewContains(t, view, "not refreshing")
	assertViewLacks(t, view, "live")
}

func TestTheClockShowsTheTimeInTheUsersOwnTimezone(t *testing.T) {
	m := modelWithDetail(t, degradedAppDetail(), testWidth, testHeight)
	brasilia := time.FixedZone("BRT", -3*60*60)
	m.now = func() time.Time { return detailNow.In(brasilia) }

	view := plain(t, m.View().Content)

	assertViewContains(t, view, "09:00:00 BRT")
	assertViewLacks(t, view, "12:00:00")
}

func TestAHeadlineSaysEachStatusInWordsAndAnUnknownOneAsItIs(t *testing.T) {
	cases := []struct {
		status *string
		want   string
	}{
		{nil, "NOT CHECKED YET"},
		{stringPtr("under_maintenance"), "UNDER MAINTENANCE"},
		{stringPtr("partial_outage"), "PARTIAL OUTAGE"},
		{stringPtr("something_new"), "SOMETHING_NEW"},
	}

	for _, testCase := range cases {
		if got := headlineOf(testCase.status); got != testCase.want {
			t.Errorf("headlineOf(%v) = %q, want %q", testCase.status, got, testCase.want)
		}
	}
}

func TestTheSidebarTakesAQuarterOfTheWidthWithinLimits(t *testing.T) {
	cases := []struct {
		name                          string
		windowWidth, wantSidebarWidth int
	}{
		{"a narrow window gets the least", 60, sidebarMinWidth},
		{"a wide window gets the most", 220, sidebarMaxWidth},
	}

	for _, testCase := range cases {
		m := modelWithDetail(t, degradedAppDetail(), testCase.windowWidth, testHeight)

		topEdge := lineContaining(t, m.View().Content, "╭─ Apps")
		sidebarEnd := lipgloss.Width(topEdge[:strings.Index(topEdge, "╮")+len("╮")])

		if sidebarEnd != screenMarginX+testCase.wantSidebarWidth {
			t.Errorf("%s: the sidebar ends at column %d, want %d", testCase.name, sidebarEnd, screenMarginX+testCase.wantSidebarWidth)
		}
	}
}

func TestHistoryGivesWayBeforeTheIncidentOrTheAffectedComponentsAreSqueezed(t *testing.T) {
	cases := []struct {
		name              string
		roomAfterIncident int
		hasAffected       bool
		want              bool
	}{
		{"exactly the strip", historyPanelHeight, false, true},
		{"one line short of the strip", historyPanelHeight - 1, false, false},
		{"the strip and the least the components need", historyPanelHeight + minAffectedPanelHeight, true, true},
		{"one line short of the components' least", historyPanelHeight + minAffectedPanelHeight - 1, true, false},
		{"room to spare but nothing affected", historyPanelHeight + minAffectedPanelHeight - 1, false, true},
	}

	for _, testCase := range cases {
		if got := hasRoomForHistory(testCase.roomAfterIncident, testCase.hasAffected); got != testCase.want {
			t.Errorf("%s: hasRoomForHistory(%d, %v) = %v, want %v", testCase.name, testCase.roomAfterIncident, testCase.hasAffected, got, testCase.want)
		}
	}
}
