package ui

import (
	"context"
	"errors"
	"fmt"
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
	m := NewAppDetailModel(fakeAppGetter{}, "web-app", "Web App", 100, 40)
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

func TestDetailScrollsWhenContentIsTallerThanTheWindow(t *testing.T) {
	appDetail := degradedAppDetail()
	for index := range 30 {
		appDetail.WatchedComponents = append(appDetail.WatchedComponents, api.Component{
			ComponentReference: api.ComponentReference{Provider: "github", ID: fmt.Sprint(index), Name: fmt.Sprintf("Component %02d", index)},
		})
	}
	m := NewAppDetailModel(fakeAppGetter{}, "web-app", "Web App", 100, 12)
	m, _ = sendDetail(m, appDetailLoadedMsg{appDetail: appDetail})

	m, _ = sendDetail(m, tea.KeyPressMsg{Code: 'j', Text: "j"})

	if m.viewport.YOffset() != 1 {
		t.Fatalf("YOffset = %d, want 1", m.viewport.YOffset())
	}
}
