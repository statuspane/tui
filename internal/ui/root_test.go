package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/matiasleidemer/statuspane-tui/internal/api"
)

type fakeBackend struct {
	fakeAppLister
	fakeAppGetter
}

func sendRoot(m RootModel, msg tea.Msg) (RootModel, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(RootModel), cmd
}

func loadedRoot(apps ...api.App) RootModel {
	m := NewRootModel(fakeBackend{})
	m, _ = sendRoot(m, tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = sendRoot(m, appsLoadedMsg{apps: apps})
	return m
}

func TestEnterOpensTheSelectedAppsDetail(t *testing.T) {
	m := loadedRoot(api.App{Name: "First", Slug: "first"}, api.App{Name: "Second", Slug: "second"})
	m, _ = sendRoot(m, tea.KeyPressMsg{Code: 'j', Text: "j"})

	m, cmd := sendRoot(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	opened, ok := cmd().(openAppMsg)
	if !ok || opened.slug != "second" {
		t.Fatalf("got %#v", opened)
	}
	m, _ = sendRoot(m, opened)

	if view := m.View().Content; !strings.Contains(view, "Second") || !strings.Contains(view, "Loading") {
		t.Fatalf("view = %q", view)
	}
}

func TestEnterOnEmptyListDoesNothing(t *testing.T) {
	m := loadedRoot()

	_, cmd := sendRoot(m, tea.KeyPressMsg{Code: tea.KeyEnter})

	if cmd != nil {
		t.Fatal("want no command")
	}
}

func TestEscReturnsToListWithSelectionKept(t *testing.T) {
	m := loadedRoot(api.App{Name: "First", Slug: "first"}, api.App{Name: "Second", Slug: "second"})
	m, _ = sendRoot(m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	m, _ = sendRoot(m, openAppMsg{slug: "second", name: "Second"})

	m, _ = sendRoot(m, closeDetailMsg{})

	if m.detail != nil || m.apps.selectedIndex != 1 {
		t.Fatalf("detail = %v, selectedIndex = %d", m.detail, m.apps.selectedIndex)
	}
	if view := m.View().Content; !strings.Contains(view, "First") {
		t.Fatalf("view = %q", view)
	}
}

func TestKeysGoOnlyToTheDetailWhileItIsOpen(t *testing.T) {
	m := loadedRoot(api.App{Name: "First", Slug: "first"}, api.App{Name: "Second", Slug: "second"})
	m, _ = sendRoot(m, openAppMsg{slug: "first", name: "First"})

	m, _ = sendRoot(m, tea.KeyPressMsg{Code: 'j', Text: "j"})

	if m.apps.selectedIndex != 0 {
		t.Fatalf("list selection moved to %d", m.apps.selectedIndex)
	}
}

func TestDetailLoadsThroughTheRoot(t *testing.T) {
	m := loadedRoot(api.App{Name: "Web App", Slug: "web-app"})
	m, _ = sendRoot(m, openAppMsg{slug: "web-app", name: "Web App"})

	m, _ = sendRoot(m, appDetailLoadedMsg{appDetail: degradedAppDetail()})

	if view := m.View().Content; !strings.Contains(view, "Git operations degraded") {
		t.Fatalf("view = %q", view)
	}
}
