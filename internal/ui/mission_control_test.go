package ui

import (
	"strings"
	"testing"

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
