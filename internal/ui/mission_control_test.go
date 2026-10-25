package ui

import (
	"errors"
	"fmt"
	"slices"
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
	m, cmd = send(m, detailSettledMsg{sequence: m.selectedDetail.sequence - 1})
	m = settledApps(m, cmd)
	if asked != "web-app" {
		t.Fatalf("a pause that was interrupted must not ask, but %q was asked for", asked)
	}

	m, cmd = send(m, detailSettledMsg{sequence: m.selectedDetail.sequence})
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

// modelWithDetail is the screen with one app, "web-app", whose detail has loaded.
func modelWithDetail(t *testing.T, appDetail api.AppDetail, width, height int) AppsModel {
	t.Helper()
	backend := fakeAppsBackend{
		fakeAppLister: fakeAppLister{apps: []api.App{{Name: "Web App", Slug: "web-app", Status: stringPtr("degraded_performance"), WatchedComponentsCount: 2}}},
		fakeAppGetter: fakeAppGetter{appDetail: appDetail},
	}
	m := NewAppsModel(backend)
	m.now = func() time.Time { return detailNow }
	m, _ = send(m, tea.WindowSizeMsg{Width: width, Height: height})
	return settledApps(m, m.Init())
}

func TestThePanelShowsTheSelectedAppsIncident(t *testing.T) {
	m := modelWithDetail(t, degradedAppDetail(), testWidth, testHeight)

	assertViewContains(t, m.View().Content,
		"Active incident", "Git operations degraded", "github", "started 2h ago",
		"We are investigating elevated errors.", "Affects ▸ API Requests")
}

func TestTheProgressLineTicksTheStepsDoneAndMarksTheCurrentOne(t *testing.T) {
	m := modelWithDetail(t, degradedAppDetail(), testWidth, testHeight)

	line := lineContaining(t, m.View().Content, "Investigating")

	var at []int
	for _, step := range []string{"✓ Investigating", "● Identified", "○ Monitoring", "○ Resolved"} {
		index := strings.Index(line, step)
		if index < 0 {
			t.Fatalf("the progress line lacks %q: %q", step, line)
		}
		at = append(at, index)
	}
	if !slices.IsSorted(at) {
		t.Errorf("the steps are out of order: %q", line)
	}
}

func TestWhenTheFourStepsDoNotFitTheProgressLineShowsOnlyTheCurrentOne(t *testing.T) {
	m := modelWithDetail(t, degradedAppDetail(), fallbackWidth, testHeight)

	view := plain(t, m.View().Content)

	assertEveryLineHasWidth(t, m.View().Content, fallbackWidth)
	if strings.Contains(view, "Resolved") {
		t.Fatalf("the steps should not be cut at the panel's edge, but they are shown:\n%s", view)
	}
	assertViewContains(t, view, "● Identified · step 2 of 4")
}

func TestSeveralIncidentsShowTheFirstInFullAndTheRestOnALineEach(t *testing.T) {
	detail := degradedAppDetail()
	detail.ActiveIncidents = append(detail.ActiveIncidents, api.Incident{
		Provider: "cloudflare", Title: "DNS resolution slow", Impact: stringPtr("minor"), Status: stringPtr("monitoring"),
	})
	m := modelWithDetail(t, detail, testWidth, testHeight)

	view := plain(t, m.View().Content)

	assertViewContains(t, view, "Active incidents (2)", "Git operations degraded", "We are investigating elevated errors.")
	line := lineContaining(t, view, "DNS resolution slow")
	assertViewContains(t, line, "cloudflare", "monitoring")
}

func TestWithoutIncidentsOrBeforeTheDetailTheIncidentPanelSaysSo(t *testing.T) {
	quiet := degradedAppDetail()
	quiet.ActiveIncidents = nil

	assertViewContains(t, modelWithDetail(t, quiet, testWidth, testHeight).View().Content, "No active incidents.")

	loading := NewAppsModel(fakeAppsBackend{fakeAppLister: fakeAppLister{apps: []api.App{{Name: "Web App", Slug: "web-app"}}}})
	loading, _ = send(loading, tea.WindowSizeMsg{Width: testWidth, Height: testHeight})
	loading, _ = send(loading, appsLoadedMsg{apps: []api.App{{Name: "Web App", Slug: "web-app"}}})

	assertViewContains(t, loading.View().Content, "Loading…")
}

func TestAnIncidentWithoutAStatusHasNoProgressLine(t *testing.T) {
	detail := degradedAppDetail()
	detail.ActiveIncidents[0].Status = nil
	m := modelWithDetail(t, detail, testWidth, testHeight)

	view := plain(t, m.View().Content)

	assertViewContains(t, view, "Git operations degraded")
	assertViewLacks(t, view, "Investigating", "Identified", "step")
}

func TestInANarrowWindowTheBannerWrapsInsteadOfCuttingItsSecondLine(t *testing.T) {
	m := modelWithDetail(t, degradedAppDetail(), fallbackWidth, testHeight)

	view := plain(t, m.View().Content)

	assertViewContains(t, view, "1 of 2 components affected", "checked 3m ago")
	assertEveryLineHasWidth(t, m.View().Content, fallbackWidth)
}

func TestTheComponentsPanelListsWhatIsNotOperationalAndCountsTheRest(t *testing.T) {
	m := modelWithDetail(t, degradedAppDetail(), testWidth, testHeight)

	view := plain(t, m.View().Content)

	assertViewContains(t, view, "Affected components", "github / API Requests", "Degraded", "1 operational")
	if strings.Contains(view, "github / Pages") {
		t.Errorf("an operational component is counted, not listed:\n%s", view)
	}
}

func componentWithStatus(name, status string) api.Component {
	return api.Component{
		ComponentReference: api.ComponentReference{Provider: "github", ID: name, Name: name},
		Status:             stringPtr(status),
	}
}

func TestTheWorstComponentsComeFirst(t *testing.T) {
	detail := degradedAppDetail()
	detail.WatchedComponents = []api.Component{
		componentWithStatus("Webhooks", "degraded_performance"),
		componentWithStatus("Git Operations", "major_outage"),
		componentWithStatus("Pages", "partial_outage"),
	}
	m := modelWithDetail(t, detail, testWidth, testHeight)

	view := plain(t, m.View().Content)

	major, partial, degraded := strings.Index(view, "Git Operations"), strings.Index(view, "Pages"), strings.Index(view, "Webhooks")
	if major < 0 || major > partial || partial > degraded {
		t.Errorf("want major, partial, then degraded outage:\n%s", view)
	}
}

func TestWhenNothingIsAffectedThereIsNoComponentsPanel(t *testing.T) {
	detail := degradedAppDetail()
	detail.WatchedComponents = []api.Component{componentWithStatus("Pages", "operational")}
	m := modelWithDetail(t, detail, testWidth, testHeight)

	assertViewLacks(t, plain(t, m.View().Content), "Affected components")
}

func TestTooManyAffectedComponentsAreCutWithACountOfWhatIsLeftOut(t *testing.T) {
	detail := degradedAppDetail()
	detail.WatchedComponents = nil
	for index := range 12 {
		detail.WatchedComponents = append(detail.WatchedComponents, componentWithStatus(fmt.Sprintf("Part %02d", index), "degraded_performance"))
	}
	m := modelWithDetail(t, detail, testWidth, 30)

	view := m.View().Content

	assertViewContains(t, view, "Part 00", "… and ", " more", "0 operational")
	assertHeightIs(t, view, 30)
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

func TestAStatusThatIsNotAStepHasNoProgressLine(t *testing.T) {
	for _, status := range []*string{nil, stringPtr("something_new")} {
		if got := progressLine(status, mutedStyle, 80); got != "" {
			t.Errorf("progressLine(%v) = %q, want none", status, got)
		}
	}
}
