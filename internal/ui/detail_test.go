package ui

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

var detailNow = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

type fakeAppGetter struct {
	appDetail api.AppDetail
	err       error
	gotSlug   *string
}

func (f fakeAppGetter) GetApp(_ context.Context, slug string) (api.AppDetail, error) {
	if f.gotSlug != nil {
		*f.gotSlug = slug
	}
	return f.appDetail, f.err
}

func newTestDetailModel() AppDetailModel {
	m := NewAppDetailModel(fakeAppGetter{}, "web-app", "Web App", 140, 40)
	m.now = func() time.Time { return detailNow }
	return m
}

func sendDetail(m AppDetailModel, msg tea.Msg) (AppDetailModel, tea.Cmd) {
	return m.Update(msg)
}

func timePtr(t time.Time) *time.Time { return &t }

func degradedAppDetail() api.AppDetail {
	return api.AppDetail{
		App:           api.App{Name: "Web App", Slug: "web-app", Status: stringPtr("degraded_performance"), WatchedComponentsCount: 2},
		LastCheckedAt: timePtr(detailNow.Add(-3 * time.Minute)),
		WatchedComponents: []api.Component{
			{ComponentReference: api.ComponentReference{Provider: "github", ID: "a", Name: "API Requests"}, GroupName: stringPtr("Core"), Status: stringPtr("degraded_performance")},
			{ComponentReference: api.ComponentReference{Provider: "github", ID: "b", Name: "Pages"}, Status: stringPtr("operational")},
		},
		ActiveIncidents: []api.Incident{{
			ID: "kz9", Provider: "github", Title: "Git operations degraded",
			Impact: stringPtr("major"), Status: stringPtr("identified"), StartedAt: timePtr(detailNow.Add(-2 * time.Hour)),
			URL:                "https://example.test/incidents/kz9",
			LatestUpdate:       &api.IncidentUpdate{Body: "We are investigating elevated errors."},
			AffectedComponents: []api.ComponentReference{{Provider: "github", ID: "a", Name: "API Requests"}},
		}},
	}
}

func TestDetailShowsLoadingFirst(t *testing.T) {
	m := newTestDetailModel()

	view := m.View().Content
	if !strings.Contains(view, "Loading") || !strings.Contains(view, "Web App") {
		t.Fatalf("view = %q", view)
	}
}

func TestDetailShowsIncidentsAndGroupedComponents(t *testing.T) {
	m := newTestDetailModel()
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: degradedAppDetail()})

	view := m.View().Content
	for _, want := range []string{
		"Degraded", "checked 3m ago",
		"Active incidents (1)", "Git operations degraded", "major", "identified", "started 2h ago",
		"We are investigating elevated errors.", "Affects: API Requests", "https://example.test/incidents/kz9",
		"Watched components (2)", "github", "Core", "API Requests", "Pages", "Operational",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
}

func TestDetailHandlesNullsAndEmptyLists(t *testing.T) {
	m := newTestDetailModel()
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: api.AppDetail{
		App: api.App{Name: "Fresh", Slug: "fresh"},
	}})

	view := m.View().Content
	for _, want := range []string{"Pending", "not checked yet", "No active incidents.", "doesn't watch any components"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
}

func TestDetailIncidentWithoutUpdateOrImpactStillRenders(t *testing.T) {
	appDetail := degradedAppDetail()
	appDetail.ActiveIncidents[0].Impact = nil
	appDetail.ActiveIncidents[0].LatestUpdate = nil
	appDetail.ActiveIncidents[0].StartedAt = nil

	m := newTestDetailModel()
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: appDetail})

	if view := m.View().Content; !strings.Contains(view, "unrated") {
		t.Fatalf("view = %q", view)
	}
}

func TestDetailNotFoundExplainsAndAllowsGoingBack(t *testing.T) {
	m := newTestDetailModel()
	m, _ = sendDetail(m, appDetailFailedMsg{err: &api.APIError{Status: 404, Code: api.CodeNotFound}})

	if view := m.View().Content; !strings.Contains(view, "Not found") {
		t.Fatalf("view = %q", view)
	}
}

func TestDetailFailedRefreshKeepsContent(t *testing.T) {
	m := newTestDetailModel()
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: degradedAppDetail()})
	m, _ = sendDetail(m, appDetailFailedMsg{err: errors.New("connection refused")})

	view := m.View().Content
	if !strings.Contains(view, "Git operations degraded") || !strings.Contains(view, "Refresh failed: connection refused") {
		t.Fatalf("view = %q", view)
	}
}

func TestDetailEscAsksToGoBack(t *testing.T) {
	_, cmd := sendDetail(newTestDetailModel(), tea.KeyPressMsg{Code: tea.KeyEscape})

	if cmd == nil {
		t.Fatal("want a command")
	}
	if _, ok := cmd().(closeDetailMsg); !ok {
		t.Fatal("want closeDetailMsg")
	}
}

func TestDetailFetchesBySlugAndFollowsRename(t *testing.T) {
	var gotSlug string
	cmd := fetchAppDetail(fakeAppGetter{appDetail: degradedAppDetail(), gotSlug: &gotSlug}, "old-slug")
	if _, ok := cmd().(appDetailLoadedMsg); !ok || gotSlug != "old-slug" {
		t.Fatalf("slug = %q", gotSlug)
	}

	m := newTestDetailModel()
	renamed := degradedAppDetail()
	renamed.Slug = "new-slug"
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: renamed})
	if m.slug != "new-slug" {
		t.Fatalf("slug = %q, want new-slug", m.slug)
	}
}

func TestDetailStaleTickIsIgnored(t *testing.T) {
	m := newTestDetailModel()
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: degradedAppDetail()})
	stale := m.refreshSequence

	m, _ = sendDetail(m, tea.KeyPressMsg{Code: 'r', Text: "r"})
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: degradedAppDetail()})

	m, cmd := sendDetail(m, detailRefreshTickMsg{sequence: stale})
	if cmd != nil || m.isLoading {
		t.Fatal("stale tick must not start a fetch")
	}
	m, cmd = sendDetail(m, detailRefreshTickMsg{sequence: m.refreshSequence})
	if cmd == nil || !m.isLoading {
		t.Fatal("current tick must start a fetch")
	}
}

func TestDetailScrollsTheComponentsWhenTheyAreTallerThanThePane(t *testing.T) {
	appDetail := degradedAppDetail()
	for index := range 30 {
		appDetail.WatchedComponents = append(appDetail.WatchedComponents, api.Component{
			ComponentReference: api.ComponentReference{Provider: "github", ID: fmt.Sprint(index), Name: fmt.Sprintf("Component %02d", index)},
		})
	}
	m := NewAppDetailModel(fakeAppGetter{}, "web-app", "Web App", 100, 12)
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: appDetail})

	m, _ = sendDetail(m, tea.KeyPressMsg{Code: 'j', Text: "j"})

	if m.componentsViewport.YOffset() != 1 {
		t.Fatalf("YOffset = %d, want 1", m.componentsViewport.YOffset())
	}
}

func TestDetailDAsksToDeleteTheApp(t *testing.T) {
	m := newTestDetailModel()
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: degradedAppDetail()})

	_, cmd := sendDetail(m, tea.KeyPressMsg{Code: 'd', Text: "d"})

	request, ok := cmd().(openDeleteAppDialogMsg)
	if !ok || request.slug != "web-app" || request.name != "Web App" || request.componentCount != 2 {
		t.Fatalf("got %#v", request)
	}
}

func TestDetailEAsksToRenameTheApp(t *testing.T) {
	m := newTestDetailModel()
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: degradedAppDetail()})

	_, cmd := sendDetail(m, tea.KeyPressMsg{Code: 'e', Text: "e"})

	request, ok := cmd().(openRenameAppDialogMsg)
	if !ok || request.slug != "web-app" || request.name != "Web App" {
		t.Fatalf("got %#v", request)
	}
}

func TestDetailEDoesNothingBeforeTheAppHasLoaded(t *testing.T) {
	_, cmd := sendDetail(newTestDetailModel(), tea.KeyPressMsg{Code: 'e', Text: "e"})

	if cmd != nil {
		t.Fatal("want no command")
	}
}

func TestDetailDDoesNothingBeforeTheAppHasLoaded(t *testing.T) {
	_, cmd := sendDetail(newTestDetailModel(), tea.KeyPressMsg{Code: 'd', Text: "d"})

	if cmd != nil {
		t.Fatal("want no command")
	}
}

func TestDetailWAsksForThePickerWithTheCurrentComponents(t *testing.T) {
	m := newTestDetailModel()
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: degradedAppDetail()})

	_, cmd := sendDetail(m, tea.KeyPressMsg{Code: 'w', Text: "w"})

	request, ok := cmd().(openWatchPickerMsg)
	if !ok || request.appSlug != "web-app" || request.appName != "Web App" || len(request.watchedComponents) != 2 {
		t.Fatalf("got %#v", request)
	}
}

func TestDetailWDoesNothingBeforeTheAppHasLoaded(t *testing.T) {
	_, cmd := sendDetail(newTestDetailModel(), tea.KeyPressMsg{Code: 'w', Text: "w"})

	if cmd != nil {
		t.Fatal("want no command")
	}
}

func tallDetail() api.AppDetail {
	appDetail := degradedAppDetail()
	for index := range 30 {
		appDetail.WatchedComponents = append(appDetail.WatchedComponents, namedComponent("github", fmt.Sprint(index), fmt.Sprintf("Component %02d", index), nil))
	}
	return appDetail
}

func TestDetailIsTwoTitledPanesThatFillTheWindow(t *testing.T) {
	m := newTestDetailModel()
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: degradedAppDetail()})

	view := m.View().Content
	assertViewContains(t, view, "╭─ Active incidents (1)", "╭─ Watched components (2)", "◉ STATUSPANE › Web App", "checked 3m ago", "w watch components", "tab switch pane", "esc back")
	assertEveryLineHasWidth(t, view, 140)
	assertHeightIs(t, view, 40)
}

func TestDetailRedrawsAtTheNewWindowSize(t *testing.T) {
	m := newTestDetailModel()
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: degradedAppDetail()})
	m, _ = sendDetail(m, tea.WindowSizeMsg{Width: 80, Height: 24})

	view := m.View().Content
	assertEveryLineHasWidth(t, view, 80)
	assertHeightIs(t, view, 24)
}

func TestTabMovesTheScrollingToTheOtherPane(t *testing.T) {
	m := NewAppDetailModel(fakeAppGetter{}, "web-app", "Web App", 140, 16)
	appDetail := tallDetail()
	for range 20 {
		appDetail.ActiveIncidents = append(appDetail.ActiveIncidents, appDetail.ActiveIncidents[0])
	}
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: appDetail})

	m, _ = sendDetail(m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	if m.componentsViewport.YOffset() != 1 || m.incidentsViewport.YOffset() != 0 {
		t.Fatalf("offsets = %d, %d: only the components should scroll first", m.componentsViewport.YOffset(), m.incidentsViewport.YOffset())
	}
	m, _ = sendDetail(m, tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = sendDetail(m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	if m.componentsViewport.YOffset() != 1 || m.incidentsViewport.YOffset() != 1 {
		t.Fatalf("offsets = %d, %d: only the incidents should scroll after tab", m.componentsViewport.YOffset(), m.incidentsViewport.YOffset())
	}
}

var moreHintText = regexp.MustCompile(`↓ \d+ more`)

func TestAMoreHintShowsWhileThereIsMoreBelowAndNotAtTheBottom(t *testing.T) {
	m := NewAppDetailModel(fakeAppGetter{}, "web-app", "Web App", 140, 16)
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: tallDetail()})

	if !moreHintText.MatchString(plain(t, m.View().Content)) {
		t.Fatal("a tall list should say there is more below")
	}
	for range 40 {
		m, _ = sendDetail(m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	}
	if moreHintText.MatchString(plain(t, m.View().Content)) {
		t.Fatal("at the bottom there is nothing more to say")
	}
}

func TestNoMoreHintWhenEverythingFits(t *testing.T) {
	m := newTestDetailModel()
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: degradedAppDetail()})

	if moreHintText.MatchString(plain(t, m.View().Content)) {
		t.Fatal("nothing is hidden")
	}
}

func TestARefreshKeepsEachPanesScrollPosition(t *testing.T) {
	m := NewAppDetailModel(fakeAppGetter{}, "web-app", "Web App", 140, 16)
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: tallDetail()})
	for range 5 {
		m, _ = sendDetail(m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	}

	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: tallDetail()})

	if m.componentsViewport.YOffset() != 5 {
		t.Fatalf("YOffset = %d, want 5", m.componentsViewport.YOffset())
	}
}

func TestTheIncidentsPaneIsTintedByTheWorstIncident(t *testing.T) {
	m := newTestDetailModel()
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: degradedAppDetail()})

	want := escapeBefore(impactStyles["major"].Render("x")) + "╭"
	if !strings.Contains(m.View().Content, want) {
		t.Fatal("the incidents panel should use the major impact colour")
	}
}

func TestTheIncidentsPaneIsOnTheRight(t *testing.T) {
	m := newTestDetailModel()
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: degradedAppDetail()})

	panelTops := plain(t, strings.Split(m.View().Content, "\n")[3])
	if strings.Index(panelTops, "Watched components") > strings.Index(panelTops, "Active incidents") {
		t.Fatalf("want the components on the left and the incidents on the right: %q", panelTops)
	}
}
