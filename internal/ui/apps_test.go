package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

type fakeAppLister struct {
	apps []api.App
	err  error
}

func (f fakeAppLister) ListApps(context.Context) ([]api.App, error) { return f.apps, f.err }

func stringPtr(s string) *string { return &s }

func newTestAppsModel(apps ...api.App) AppsModel {
	return NewAppsModel(fakeAppLister{apps: apps})
}

func send(m AppsModel, msg tea.Msg) (AppsModel, tea.Cmd) {
	updated, cmd := m.Update(msg)
	return updated.(AppsModel), cmd
}

func pressRune(m AppsModel, r rune) AppsModel {
	m, _ = send(m, tea.KeyPressMsg{Code: r, Text: string(r)})
	return m
}

func pressSpecial(m AppsModel, code rune) AppsModel {
	m, _ = send(m, tea.KeyPressMsg{Code: code})
	return m
}

func TestShowsSpinnerBeforeFirstLoad(t *testing.T) {
	m := newTestAppsModel()

	if view := m.View().Content; !strings.Contains(view, "Loading apps") {
		t.Fatalf("view = %q", view)
	}
}

func TestLoadedAppsShowNamesStatusesAndNullAsPending(t *testing.T) {
	m := newTestAppsModel()
	m, _ = send(m, appsLoadedMsg{apps: []api.App{
		{Name: "Web App", Slug: "web-app", Status: stringPtr("degraded_performance"), WatchedComponentsCount: 3},
		{Name: "Fresh", Slug: "fresh", Status: nil},
	}})

	assertViewContains(t, m.View().Content, "Web App", "Fresh", "DEGRADED PERFORMANCE", "3 components")

	m = pressRune(m, 'j')

	assertViewContains(t, m.View().Content, "NOT CHECKED YET")
}

func TestEmptyOrganizationSaysSo(t *testing.T) {
	m := newTestAppsModel()
	m, _ = send(m, appsLoadedMsg{})

	if view := m.View().Content; !strings.Contains(view, "No apps yet") {
		t.Fatalf("view = %q", view)
	}
}

func TestRejectedKeyTellsUserWhichSettingToCheck(t *testing.T) {
	m := newTestAppsModel()
	m, _ = send(m, appsFailedMsg{err: &api.APIError{Status: 401, Code: api.CodeUnauthorized}})

	if view := m.View().Content; !strings.Contains(view, "STATUSPANE_API_KEY") {
		t.Fatalf("view = %q", view)
	}
}

func TestRateLimitShowsRetryAfterSeconds(t *testing.T) {
	m := newTestAppsModel()
	err := &api.APIError{Status: 429, Code: api.CodeRateLimited, RetryAfter: 42 * time.Second}
	m, _ = send(m, appsFailedMsg{err: err})

	if view := m.View().Content; !strings.Contains(view, "42 seconds") {
		t.Fatalf("view = %q", view)
	}
}

func TestFailedRefreshKeepsPreviousApps(t *testing.T) {
	m := newTestAppsModel()
	m, _ = send(m, appsLoadedMsg{apps: []api.App{{Name: "Web App", Status: stringPtr("operational")}}})
	m, _ = send(m, appsFailedMsg{err: errors.New("connection refused")})

	view := m.View().Content
	assertViewContains(t, view, "Web App", "Refresh failed:", "connection", "refused")
}

func TestCursorMovesAndStaysWithinList(t *testing.T) {
	m := newTestAppsModel()
	m, _ = send(m, appsLoadedMsg{apps: []api.App{{Name: "First"}, {Name: "Second"}}})

	m = pressSpecial(pressSpecial(pressRune(m, 'j'), tea.KeyDown), tea.KeyDown)
	if m.selectedIndex != 1 {
		t.Fatalf("selectedIndex = %d, want 1", m.selectedIndex)
	}
	m = pressSpecial(pressSpecial(pressRune(m, 'k'), tea.KeyUp), tea.KeyUp)
	if m.selectedIndex != 0 {
		t.Fatalf("selectedIndex = %d, want 0", m.selectedIndex)
	}
}

func TestCursorClampsWhenListShrinks(t *testing.T) {
	m := newTestAppsModel()
	m, _ = send(m, appsLoadedMsg{apps: []api.App{{Name: "First"}, {Name: "Second"}}})
	m = pressRune(m, 'j')
	m, _ = send(m, appsLoadedMsg{apps: []api.App{{Name: "First"}}})

	if m.selectedIndex != 0 {
		t.Fatalf("selectedIndex = %d, want 0", m.selectedIndex)
	}
}

func TestQuitKey(t *testing.T) {
	_, cmd := send(newTestAppsModel(), tea.KeyPressMsg{Code: 'q', Text: "q"})

	if cmd == nil {
		t.Fatal("want a quit command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("want tea.QuitMsg")
	}
}

func TestInitFetchesApps(t *testing.T) {
	want := []api.App{{Name: "Web App"}}
	cmd := fetchApps(fakeAppLister{apps: want})

	loaded, ok := cmd().(appsLoadedMsg)
	if !ok || len(loaded.apps) != 1 || loaded.apps[0].Name != "Web App" {
		t.Fatalf("got %#v", cmd())
	}
}

func TestFetchFailureBecomesFailedMessage(t *testing.T) {
	cmd := fetchApps(fakeAppLister{err: errors.New("boom")})

	if _, ok := cmd().(appsFailedMsg); !ok {
		t.Fatalf("got %#v", cmd())
	}
}

func TestRefreshTickStartsOneFetchAndStaleTicksAreIgnored(t *testing.T) {
	m := newTestAppsModel()
	m, _ = send(m, appsLoadedMsg{})
	staleSequence := m.refreshSequence

	m = pressRune(m, 'r')
	m, _ = send(m, appsLoadedMsg{})

	m, cmd := send(m, refreshTickMsg{sequence: staleSequence})
	if cmd != nil || m.isLoading {
		t.Fatal("stale tick must not start a fetch")
	}

	m, cmd = send(m, refreshTickMsg{sequence: m.refreshSequence})
	if cmd == nil || !m.isLoading {
		t.Fatal("current tick must start a fetch")
	}
}

func TestRefreshKeyIgnoredWhileLoading(t *testing.T) {
	m := newTestAppsModel()

	_, cmd := send(m, tea.KeyPressMsg{Code: 'r', Text: "r"})

	if cmd != nil {
		t.Fatal("r while loading must not start another fetch")
	}
}

func TestRateLimitUnderOneSecondReadsAsOneSecond(t *testing.T) {
	m := newTestAppsModel()
	err := &api.APIError{Status: 429, Code: api.CodeRateLimited, RetryAfter: 500 * time.Millisecond}
	m, _ = send(m, appsFailedMsg{err: err})

	if view := m.View().Content; !strings.Contains(view, "in 1 second.") {
		t.Fatalf("view = %q", view)
	}
}

func sizedAppsModel(width, height int, apps ...api.App) AppsModel {
	m := newTestAppsModel()
	m, _ = send(m, tea.WindowSizeMsg{Width: width, Height: height})
	m, _ = send(m, appsLoadedMsg{apps: apps})
	return m
}

func TestAppsScreenIsFramedAndFillsTheWindow(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "Web App", Status: stringPtr("operational")})

	view := m.View().Content
	assertViewContains(t, view, "◉ STATUSPANE", "╭─ Apps ─", "refreshes every 30s", "q quit")
	assertEveryLineHasWidth(t, view, 100)
	assertHeightIs(t, view, 30)
}

func TestAppsScreenFallsBackToAClassicTerminalBeforeTheSizeIsKnown(t *testing.T) {
	m := newTestAppsModel()

	assertEveryLineHasWidth(t, m.View().Content, fallbackWidth)
}

func TestAppsHeaderShowsTheStatusSummary(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "A", Status: stringPtr("operational")}, api.App{Name: "B", Status: stringPtr("operational")})

	assertViewContains(t, m.View().Content, "2 operational")
}

func TestAppsFooterOffersEveryAppAction(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "Web App"})

	assertViewContains(t, m.View().Content, "enter open", "n new app", "e rename", "d delete", "r refresh")
}

func TestSelectedAppIsMarkedAndTheOthersAreNot(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "First"}, api.App{Name: "Second"})
	m = pressRune(m, 'j')

	view := plain(t, m.View().Content)
	if !strings.Contains(view, "▎● Second") || strings.Contains(view, "▎● First") {
		t.Fatalf("view = %s", view)
	}
}

func TestStatesBeforeTheFirstLoadAreInsideTheFrame(t *testing.T) {
	failed, _ := send(newTestAppsModel(), appsFailedMsg{err: errors.New("boom")})
	empty, _ := send(newTestAppsModel(), appsLoadedMsg{})
	states := []struct {
		name     string
		model    AppsModel
		wantText string
	}{
		{"loading", newTestAppsModel(), "Loading apps"},
		{"failed", failed, "boom"},
		{"empty", empty, "No apps yet"},
	}

	for _, state := range states {
		t.Run(state.name, func(t *testing.T) {
			view := state.model.View().Content
			assertViewContains(t, view, "╭─ Apps ─", state.wantText, "q quit")
			assertEveryLineHasWidth(t, view, fallbackWidth)
		})
	}
}

func TestRefreshStatusSitsInsideThePanelUnderTheList(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "Web App"})
	m, _ = send(m, appsFailedMsg{err: errors.New("connection refused")})

	view := m.View().Content
	// The notice wraps to the sidebar's width, so it is read word by word.
	assertViewContains(t, view, "Web App", "Refresh failed:", "connection", "refused")
	assertEveryLineHasWidth(t, view, 100)
}

func TestLongListScrollsToKeepTheSelectedAppInView(t *testing.T) {
	var apps []api.App
	for index := range 40 {
		apps = append(apps, api.App{Name: fmt.Sprintf("App %02d", index)})
	}
	m := sizedAppsModel(100, 16, apps...)
	for range 30 {
		m = pressRune(m, 'j')
	}

	view := m.View().Content
	assertViewContains(t, view, "▎● App 30")
	assertViewLacks(t, view, "App 00")
	assertHeightIs(t, view, 16)
}

func TestResizingRedrawsAtTheNewWidth(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "Web App"})
	m, _ = send(m, tea.WindowSizeMsg{Width: 60, Height: 20})

	assertEveryLineHasWidth(t, m.View().Content, 60)
}

func TestNAsksForTheCreateDialog(t *testing.T) {
	_, cmd := send(newTestAppsModel(), tea.KeyPressMsg{Code: 'n', Text: "n"})

	if _, ok := cmd().(openCreateAppDialogMsg); !ok {
		t.Fatal("want openCreateAppDialogMsg")
	}
}

func TestDAsksToDeleteTheSelectedApp(t *testing.T) {
	m := newTestAppsModel()
	m, _ = send(m, appsLoadedMsg{apps: []api.App{{Name: "First", Slug: "first"}, {Name: "Second", Slug: "second", WatchedComponentsCount: 3}}})
	m = pressRune(m, 'j')

	_, cmd := send(m, tea.KeyPressMsg{Code: 'd', Text: "d"})

	request, ok := cmd().(openDeleteAppDialogMsg)
	if !ok || request.slug != "second" || request.name != "Second" || request.componentCount != 3 {
		t.Fatalf("got %#v", request)
	}
}

func TestEAsksToRenameTheSelectedApp(t *testing.T) {
	m := newTestAppsModel()
	m, _ = send(m, appsLoadedMsg{apps: []api.App{{Name: "First", Slug: "first"}, {Name: "Second", Slug: "second"}}})
	m = pressRune(m, 'j')

	_, cmd := send(m, tea.KeyPressMsg{Code: 'e', Text: "e"})

	request, ok := cmd().(openRenameAppDialogMsg)
	if !ok || request.slug != "second" || request.name != "Second" {
		t.Fatalf("got %#v", request)
	}
}

func TestEOnAnEmptyListDoesNothing(t *testing.T) {
	m := newTestAppsModel()
	m, _ = send(m, appsLoadedMsg{})

	_, cmd := send(m, tea.KeyPressMsg{Code: 'e', Text: "e"})

	if cmd != nil {
		t.Fatal("want no command")
	}
}

func TestDOnAnEmptyListDoesNothing(t *testing.T) {
	m := newTestAppsModel()
	m, _ = send(m, appsLoadedMsg{})

	_, cmd := send(m, tea.KeyPressMsg{Code: 'd', Text: "d"})

	if cmd != nil {
		t.Fatal("want no command")
	}
}

func TestAnEmptyListInvitesCreatingAnApp(t *testing.T) {
	m := newTestAppsModel()
	m, _ = send(m, appsLoadedMsg{})

	if view := m.View().Content; !strings.Contains(view, "Press n to create one") {
		t.Fatalf("view = %q", view)
	}
}

func TestAppsChangedReloadsAtOnceWhenNothingIsLoading(t *testing.T) {
	m := newTestAppsModel()
	m, _ = send(m, appsLoadedMsg{})

	m, cmd := send(m, appsChangedMsg{})

	if cmd == nil || !m.isLoading {
		t.Fatal("want a reload")
	}
}

func TestAppsChangedDuringALoadReloadsRightAfterIt(t *testing.T) {
	m := newTestAppsModel()

	m, cmd := send(m, appsChangedMsg{})
	if cmd != nil {
		t.Fatal("must not start a second fetch while one is running")
	}
	m, cmd = send(m, appsLoadedMsg{})

	if cmd == nil || !m.isLoading {
		t.Fatal("want a reload as soon as the running fetch lands")
	}
}

func selectedRowOf(t *testing.T, m AppsModel) string {
	t.Helper()
	for _, line := range strings.Split(m.View().Content, "\n") {
		if strings.Contains(plain(t, line), "▎") {
			return line
		}
	}
	t.Fatal("no selected row")
	return ""
}

func TestSelectedRowIsTinted(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "Web App", Status: stringPtr("operational")})

	tint := escapeBefore(lipgloss.NewStyle().Background(rowTintColor(true)).Render("x"))
	if row := selectedRowOf(t, m); !strings.Contains(row, tint) {
		t.Errorf("selected row has no dark tint: %q", row)
	}
}

func TestSelectedRowKeepsItsStatusColour(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "Web App", Status: stringPtr("operational")})

	statusStyle := appearanceOf(stringPtr("operational")).style.Background(rowTintColor(true))
	if row := selectedRowOf(t, m); !strings.Contains(row, escapeBefore(statusStyle.Render("x"))) {
		t.Errorf("selected row lost the status colour: %q", row)
	}
}

func TestSelectedRowTintFollowsALightBackground(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "Web App"})
	m, _ = send(m, tea.BackgroundColorMsg{Color: lipgloss.Color("#ffffff")})

	lightTint := escapeBefore(lipgloss.NewStyle().Background(rowTintColor(false)).Render("x"))
	if row := selectedRowOf(t, m); !strings.Contains(row, lightTint) {
		t.Errorf("selected row has no light tint: %q", row)
	}
}

func TestOnlyTheSelectedAppsOwnLineIsTinted(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "First"}, api.App{Name: "Second"}, api.App{Name: "Third"})
	m = pressRune(m, 'j')

	tint := escapeBefore(lipgloss.NewStyle().Background(rowTintColor(true)).Render("x"))
	var tinted []string
	for _, line := range strings.Split(m.View().Content, "\n") {
		if strings.Contains(line, tint) {
			tinted = append(tinted, plain(t, line))
		}
	}
	if len(tinted) != 1 {
		t.Fatalf("%d lines are tinted, want 1", len(tinted))
	}
	assertViewContains(t, tinted[0], "▎● Second")
}
