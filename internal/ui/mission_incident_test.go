package ui

import (
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

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
	appDetail := degradedAppDetail()
	appDetail.ActiveIncidents = append(appDetail.ActiveIncidents, api.Incident{
		Provider: "cloudflare", Title: "DNS resolution slow", Impact: stringPtr("minor"), Status: stringPtr("monitoring"),
	})
	m := modelWithDetail(t, appDetail, testWidth, testHeight)

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
	appDetail := degradedAppDetail()
	appDetail.ActiveIncidents[0].Status = nil
	m := modelWithDetail(t, appDetail, testWidth, testHeight)

	view := plain(t, m.View().Content)

	assertViewContains(t, view, "Git operations degraded")
	assertViewLacks(t, view, "Investigating", "Identified", "step")
}

func TestAStatusThatIsNotAStepHasNoProgressLine(t *testing.T) {
	for _, status := range []*string{nil, stringPtr("something_new")} {
		if got := progressLine(status, mutedStyle, 80); got != "" {
			t.Errorf("progressLine(%v) = %q, want none", status, got)
		}
	}
}
