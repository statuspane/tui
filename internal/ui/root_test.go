package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

type fakeBackend struct {
	fakeAppLister
	fakeAppGetter
	fakeAppCreator
	fakeAppDeleter
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

// collectMessages runs a command that returns immediately, flattening batches. Don't use it on
// a command that includes a tea.Tick: that one sleeps for its whole duration.
func collectMessages(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	batch, isBatch := msg.(tea.BatchMsg)
	if !isBatch {
		return []tea.Msg{msg}
	}
	var messages []tea.Msg
	for _, inner := range batch {
		messages = append(messages, collectMessages(inner)...)
	}
	return messages
}

func containsAppsChanged(messages []tea.Msg) bool {
	for _, msg := range messages {
		if _, ok := msg.(appsChangedMsg); ok {
			return true
		}
	}
	return false
}

func TestNOpensTheCreateDialogInPlaceOfTheList(t *testing.T) {
	m := loadedRoot(api.App{Name: "First", Slug: "first"})

	m, _ = sendRoot(m, openCreateAppDialogMsg{})

	if view := m.View().Content; !strings.Contains(view, "New app") || strings.Contains(view, "First") {
		t.Fatalf("view = %q", view)
	}
}

func TestKeysGoToTheOpenDialogNotTheScreenBehindIt(t *testing.T) {
	m := loadedRoot(api.App{Name: "First", Slug: "first"}, api.App{Name: "Second", Slug: "second"})
	m, _ = sendRoot(m, openCreateAppDialogMsg{})

	m, _ = sendRoot(m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	m, _ = sendRoot(m, tea.KeyPressMsg{Code: 'q', Text: "q"})

	if m.apps.selectedIndex != 0 || m.dialog == nil {
		t.Fatalf("selectedIndex = %d, dialog open = %v", m.apps.selectedIndex, m.dialog != nil)
	}
	if view := m.View().Content; !strings.Contains(view, "jq") {
		t.Fatalf("typed text missing from %q", view)
	}
}

func TestEscClosesTheDialogAndShowsTheListAgain(t *testing.T) {
	m := loadedRoot(api.App{Name: "First", Slug: "first"})
	m, _ = sendRoot(m, openCreateAppDialogMsg{})

	m, _ = sendRoot(m, closeDialogMsg{})

	if m.dialog != nil || !strings.Contains(m.View().Content, "First") {
		t.Fatal("want the list back")
	}
}

func TestACreatedAppOpensItsDetailAndRefreshesTheList(t *testing.T) {
	m := loadedRoot()
	m, _ = sendRoot(m, openCreateAppDialogMsg{})
	created := api.AppDetail{App: api.App{Name: "Storefront", Slug: "storefront"}}

	m, cmd := sendRoot(m, appCreatedMsg{appDetail: created})

	if m.dialog != nil || m.detail == nil || m.detail.slug != "storefront" {
		t.Fatalf("dialog = %v, detail = %v", m.dialog, m.detail)
	}
	if !containsAppsChanged(collectMessages(cmd)) {
		t.Fatal("the list must be told to reload")
	}
}

func TestDeletingTheOpenAppClosesItsDetailAndRefreshesTheList(t *testing.T) {
	m := loadedRoot(api.App{Name: "Web App", Slug: "web-app"})
	m, _ = sendRoot(m, openAppMsg{slug: "web-app", name: "Web App"})
	m, _ = sendRoot(m, openDeleteAppDialogMsg{slug: "web-app", name: "Web App", componentCount: 2})

	m, cmd := sendRoot(m, appDeletedMsg{slug: "web-app"})

	if m.dialog != nil || m.detail != nil {
		t.Fatalf("dialog = %v, detail = %v", m.dialog, m.detail)
	}
	if !containsAppsChanged(collectMessages(cmd)) {
		t.Fatal("the list must be told to reload")
	}
}

func TestDeletingAnotherAppLeavesTheOpenDetailAlone(t *testing.T) {
	m := loadedRoot(api.App{Name: "Web App", Slug: "web-app"})
	m, _ = sendRoot(m, openAppMsg{slug: "web-app", name: "Web App"})

	m, _ = sendRoot(m, appDeletedMsg{slug: "other"})

	if m.detail == nil {
		t.Fatal("the detail of a different app must stay open")
	}
}

func TestTheDeleteDialogOpensOverTheDetailToo(t *testing.T) {
	m := loadedRoot(api.App{Name: "Web App", Slug: "web-app"})
	m, _ = sendRoot(m, openAppMsg{slug: "web-app", name: "Web App"})

	m, _ = sendRoot(m, openDeleteAppDialogMsg{slug: "web-app", name: "Web App", componentCount: 2})

	if view := m.View().Content; !strings.Contains(view, "Delete Web App?") {
		t.Fatalf("view = %q", view)
	}
}

func TestCtrlCQuitsFromADialog(t *testing.T) {
	m := loadedRoot()
	m, _ = sendRoot(m, openCreateAppDialogMsg{})

	_, cmd := sendRoot(m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})

	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("want tea.QuitMsg")
	}
}
