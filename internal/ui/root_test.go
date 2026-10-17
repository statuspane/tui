package ui

import (
	"errors"
	"fmt"
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
	fakeProviderCatalog
	fakeWatchedComponentsSetter
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

	m, cmd := sendRoot(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	m, _ = sendRoot(m, cmd())

	if m.dialog != nil || !strings.Contains(m.View().Content, "First") {
		t.Fatal("want the list back")
	}
}

func typeIntoRoot(m RootModel, text string) RootModel {
	for _, r := range text {
		m, _ = sendRoot(m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestAFailedCreateShowsTheErrorAndKeepsTheDialogOpen(t *testing.T) {
	m := NewRootModel(fakeBackend{fakeAppCreator: fakeAppCreator{err: errors.New("connection refused")}})
	m, _ = sendRoot(m, openCreateAppDialogMsg{})
	m, cmd := sendRoot(typeIntoRoot(m, "Storefront"), tea.KeyPressMsg{Code: tea.KeyEnter})

	m, _ = sendRoot(m, cmd())

	view := m.View().Content
	if m.dialog == nil || !strings.Contains(view, "connection refused") || !strings.Contains(view, "Storefront") {
		t.Fatalf("view = %q", view)
	}
}

func TestAFailedDeleteShowsTheErrorAndKeepsTheDialogOpen(t *testing.T) {
	m := NewRootModel(fakeBackend{fakeAppDeleter: fakeAppDeleter{err: errors.New("connection refused")}})
	m, _ = sendRoot(m, openDeleteAppDialogMsg{slug: "web-app", name: "Web App", componentCount: 2})
	m, cmd := sendRoot(typeIntoRoot(m, "Web App"), tea.KeyPressMsg{Code: tea.KeyEnter})

	m, _ = sendRoot(m, cmd())

	if view := m.View().Content; m.dialog == nil || !strings.Contains(view, "connection refused") {
		t.Fatalf("view = %q", view)
	}
}

func TestADialogSurvivesTheListRefreshingBehindIt(t *testing.T) {
	m := loadedRoot(api.App{Name: "First", Slug: "first"})
	m, _ = sendRoot(m, openCreateAppDialogMsg{})
	m = typeIntoRoot(m, "Store")

	m, _ = sendRoot(m, refreshTickMsg{sequence: m.apps.refreshSequence})
	m, _ = sendRoot(m, appsLoadedMsg{apps: []api.App{{Name: "Other", Slug: "other"}}})

	if view := m.View().Content; m.dialog == nil || !strings.Contains(view, "Store") {
		t.Fatalf("view = %q", view)
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

func rootOnLoadedDetail() RootModel {
	m := loadedRoot(api.App{Name: "Web App", Slug: "web-app"})
	m, _ = sendRoot(m, openAppMsg{slug: "web-app", name: "Web App"})
	m, _ = sendRoot(m, appDetailLoadedMsg{appDetail: degradedAppDetail()})
	return m
}

func TestWOpensThePickerForTheAppOnScreen(t *testing.T) {
	m := rootOnLoadedDetail()

	m, cmd := sendRoot(m, tea.KeyPressMsg{Code: 'w', Text: "w"})
	opened, ok := cmd().(openWatchPickerMsg)
	if !ok || opened.appSlug != "web-app" || len(opened.watchedComponents) != 2 {
		t.Fatalf("got %#v", opened)
	}
	m, _ = sendRoot(m, opened)

	if m.picker == nil || !strings.Contains(m.View().Content, "Watch components") {
		t.Fatalf("view = %q", m.View().Content)
	}
}

func TestThePickerLoadsItsProvidersThroughTheRoot(t *testing.T) {
	m := rootOnLoadedDetail()
	m, _ = sendRoot(m, openWatchPickerMsg{appSlug: "web-app", appName: "Web App"})

	m, _ = sendRoot(m, providersLoadedMsg{providers: testCatalog().providers})

	if view := m.View().Content; !strings.Contains(view, "GitHub") || !strings.Contains(view, "Cloudflare") {
		t.Fatalf("view = %q", view)
	}
}

func TestKeysGoOnlyToThePickerWhileItIsOpen(t *testing.T) {
	m := loadedRoot(api.App{Name: "Web App", Slug: "web-app"}, api.App{Name: "Other", Slug: "other"})
	m, _ = sendRoot(m, tea.WindowSizeMsg{Width: 100, Height: 12})
	m, _ = sendRoot(m, openAppMsg{slug: "web-app", name: "Web App"})
	tall := degradedAppDetail()
	for index := range 30 {
		tall.WatchedComponents = append(tall.WatchedComponents, component("github", fmt.Sprintf("extra-%02d", index)))
	}
	m, _ = sendRoot(m, appDetailLoadedMsg{appDetail: tall})
	detailBefore := m.View().Content
	m, _ = sendRoot(m, openWatchPickerMsg{appSlug: "web-app", appName: "Web App"})
	m, _ = sendRoot(m, providersLoadedMsg{providers: testCatalog().providers})

	m, _ = sendRoot(m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	m, _ = sendRoot(m, tea.KeyPressMsg{Code: 'j', Text: "j"})
	m, _ = sendRoot(m, tea.KeyPressMsg{Code: 'd', Text: "d"})
	assertViewContains(t, m.View().Content, "Cloudflare components")

	m, _ = sendRoot(m, closeWatchPickerMsg{})
	if m.dialog != nil {
		t.Fatal("d must not open the delete dialog from the picker")
	}
	if m.View().Content != detailBefore {
		t.Fatalf("keys leaked to the screen behind the picker:\nbefore: %q\nafter:  %q", detailBefore, m.View().Content)
	}
}

func TestEscThroughTheRootClosesThePickerAndShowsTheDetailAgain(t *testing.T) {
	m := rootOnLoadedDetail()
	m, _ = sendRoot(m, openWatchPickerMsg{appSlug: "web-app", appName: "Web App"})

	m, cmd := sendRoot(m, tea.KeyPressMsg{Code: tea.KeyEscape})
	m, _ = sendRoot(m, cmd())

	if m.picker != nil || !strings.Contains(m.View().Content, "Git operations degraded") {
		t.Fatalf("view = %q", m.View().Content)
	}
}

func TestASavedSetClosesThePickerUpdatesTheDetailAndReloadsTheList(t *testing.T) {
	m := rootOnLoadedDetail()
	m, _ = sendRoot(m, openWatchPickerMsg{appSlug: "web-app", appName: "Web App"})
	saved := degradedAppDetail()
	saved.WatchedComponents = append(saved.WatchedComponents, namedComponent("github", "actions", "Actions Runner", nil))

	m, cmd := sendRoot(m, watchedComponentsSavedMsg{appDetail: saved})

	if m.picker != nil || !strings.Contains(m.detail.viewport.GetContent(), "Actions Runner") {
		t.Fatalf("picker open = %v", m.picker != nil)
	}
	if !containsAppsChanged(collectMessages(cmd)) {
		t.Fatal("the list must be told to reload")
	}
}
