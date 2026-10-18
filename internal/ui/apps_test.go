package ui

import (
	"context"
	"errors"
	"fmt"
	"slices"
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

	view := m.View().Content
	for _, want := range []string{"Web App", "Degraded", "3 components", "Fresh", "Pending"} {
		if !strings.Contains(view, want) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
	}
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
	if !strings.Contains(view, "Web App") || !strings.Contains(view, "Refresh failed: connection refused") {
		t.Fatalf("view = %q", view)
	}
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

func TestStatusColumnsAlignWhenStyled(t *testing.T) {
	m := newTestAppsModel()
	m, _ = send(m, appsLoadedMsg{apps: []api.App{
		{Name: "Alpha", Status: stringPtr("operational"), WatchedComponentsCount: 1},
		{Name: "Bravo", Status: stringPtr("partial_outage"), WatchedComponentsCount: 1},
		{Name: "Delta", Status: nil, WatchedComponentsCount: 1},
	}})

	var columns []int
	for _, line := range m.listLines(76, 10) {
		text := plain(t, line)
		if index := strings.Index(text, " component"); index >= 0 {
			columns = append(columns, lipgloss.Width(text[:index]))
		}
	}
	if len(columns) != 3 {
		t.Fatalf("want 3 rows with a component count, got %v", columns)
	}
	for _, column := range columns {
		if column != columns[0] {
			t.Fatalf("component columns differ: %v", columns)
		}
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
	assertViewContains(t, view, "◉ STATUSPANE", "╭─ Apps ─", "NAME", "STATUS", "WATCHING", "refreshes every 30s", "q quit")
	assertEveryLineIs(t, view, 100)
	if got := len(strings.Split(view, "\n")); got != 30 {
		t.Fatalf("height = %d, want 30", got)
	}
}

func TestAppsScreenFallsBackToAClassicTerminalBeforeTheSizeIsKnown(t *testing.T) {
	m := newTestAppsModel()

	assertEveryLineIs(t, m.View().Content, fallbackWidth)
}

func TestAppsHeaderSummarisesHowManyAppsAreInEachStatus(t *testing.T) {
	m := sizedAppsModel(100, 30,
		api.App{Name: "A", Status: stringPtr("degraded_performance")},
		api.App{Name: "B", Status: stringPtr("operational")},
		api.App{Name: "C", Status: stringPtr("operational")},
		api.App{Name: "D", Status: stringPtr("under_maintenance")},
		api.App{Name: "E"},
	)

	var header string
	for _, line := range strings.Split(plain(t, m.View().Content), "\n") {
		if strings.Contains(line, "STATUSPANE") {
			header = line
		}
	}
	for _, want := range []string{"2 operational", "1 maintenance", "1 degraded", "1 pending"} {
		if !strings.Contains(header, want) {
			t.Errorf("header missing %q: %q", want, header)
		}
	}
	if strings.Index(header, "operational") > strings.Index(header, "degraded") {
		t.Errorf("want calmer statuses first: %q", header)
	}
}

func TestAppsFooterOffersEveryAppAction(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "Web App"})

	assertViewContains(t, m.View().Content, "enter open", "n new app", "e rename", "d delete", "r refresh")
}

func TestSelectedAppIsMarkedAndTheOthersAreNot(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "First"}, api.App{Name: "Second"})
	m = pressRune(m, 'j')

	view := plain(t, m.View().Content)
	if !strings.Contains(view, "▸ Second") || strings.Contains(view, "▸ First") {
		t.Fatalf("view = %s", view)
	}
}

func TestStatesBeforeTheFirstLoadAreInsideTheFrame(t *testing.T) {
	loading := newTestAppsModel()
	failed, _ := send(newTestAppsModel(), appsFailedMsg{err: errors.New("boom")})
	empty, _ := send(newTestAppsModel(), appsLoadedMsg{})

	for _, m := range []AppsModel{loading, failed, empty} {
		view := m.View().Content
		assertViewContains(t, view, "╭─ Apps ─", "q quit")
		assertEveryLineIs(t, view, fallbackWidth)
	}
}

func TestRefreshStatusSitsInsideThePanelUnderTheList(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "Web App"})
	m, _ = send(m, appsFailedMsg{err: errors.New("connection refused")})

	view := m.View().Content
	assertViewContains(t, view, "Web App", "Refresh failed: connection refused")
	assertEveryLineIs(t, view, 100)
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
	assertViewContains(t, view, "▸ App 30")
	assertViewLacks(t, view, "App 00")
	if got := len(strings.Split(view, "\n")); got != 16 {
		t.Fatalf("height = %d, want 16", got)
	}
}

func TestResizingRedrawsAtTheNewWidth(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "Web App"})
	m, _ = send(m, tea.WindowSizeMsg{Width: 60, Height: 20})

	assertEveryLineIs(t, m.View().Content, 60)
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

func TestColumnHeaderHasARuleUnderIt(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "Web App"})

	assertViewContains(t, m.View().Content, "│  "+strings.Repeat("─", 90)+"  │")
}

func TestStatusAndWatchingColumnsSitAtTheRightOfTheRow(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "Web App", Status: stringPtr("operational"), WatchedComponentsCount: 3})

	var columns []int
	for _, line := range strings.Split(plain(t, m.View().Content), "\n") {
		for _, marker := range []string{"WATCHING", "3 components"} {
			if index := strings.Index(line, marker); index >= 0 {
				columns = append(columns, lipgloss.Width(line[:index]))
			}
		}
	}
	if len(columns) != 2 || columns[0] != columns[1] || columns[0] < 70 {
		t.Fatalf("watching columns = %v, want both at the right edge", columns)
	}
}

func selectedRowOf(t *testing.T, m AppsModel) string {
	t.Helper()
	for _, line := range strings.Split(m.View().Content, "\n") {
		if strings.Contains(plain(t, line), "▸") {
			return line
		}
	}
	t.Fatal("no selected row")
	return ""
}

func TestSelectedRowIsTintedAndKeepsItsStatusColour(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "Web App", Status: stringPtr("operational")})

	row := selectedRowOf(t, m)
	if !strings.Contains(row, "48;5;236") {
		t.Errorf("selected row has no dark tint: %q", row)
	}
	if !strings.Contains(row, "32;48;5;236") {
		t.Errorf("selected row lost the status colour: %q", row)
	}
}

func TestSelectedRowTintFollowsALightBackground(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "Web App"})
	m, _ = send(m, tea.BackgroundColorMsg{Color: lipgloss.Color("#ffffff")})

	if row := selectedRowOf(t, m); !strings.Contains(row, "48;5;254") {
		t.Errorf("selected row has no light tint: %q", row)
	}
}

func TestOnlyTheSelectedAppsOwnLineIsTinted(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "First"}, api.App{Name: "Second"}, api.App{Name: "Third"})
	m = pressRune(m, 'j')

	var tinted int
	for _, line := range strings.Split(m.View().Content, "\n") {
		if strings.Contains(line, "48;5;236") {
			tinted++
			assertViewContains(t, line, "▸ Second")
		}
	}
	if tinted != 1 {
		t.Fatalf("%d lines are tinted, want 1", tinted)
	}
}

func TestAppsAreSeparatedByABlankLine(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "First"}, api.App{Name: "Second"}, api.App{Name: "Third"})
	m = pressRune(m, 'j')
	m = pressRune(m, 'j')

	lines := strings.Split(plain(t, m.View().Content), "\n")
	first := slices.IndexFunc(lines, func(line string) bool { return strings.Contains(line, "First") })
	blank := strings.Trim(strings.Join(strings.Fields(lines[first+1]), ""), "│")
	if blank != "" || !strings.Contains(lines[first+2], "Second") {
		t.Fatalf("want First, a blank line, then Second:\n%s", strings.Join(lines[first:first+3], "\n"))
	}
}

func TestSelectedRowIsPaddedByHalfALineAboveAndBelow(t *testing.T) {
	m := sizedAppsModel(100, 30, api.App{Name: "First"}, api.App{Name: "Second"}, api.App{Name: "Third"})
	m = pressRune(m, 'j')

	lines := strings.Split(plain(t, m.View().Content), "\n")
	selected := slices.IndexFunc(lines, func(line string) bool { return strings.Contains(line, "▸") })
	if !strings.Contains(lines[selected-1], "▄▄▄▄") || !strings.Contains(lines[selected+1], "▀▀▀▀") {
		t.Fatalf("want a half block above and below the selected row:\n%s", strings.Join(lines[selected-1:selected+2], "\n"))
	}
	if strings.ContainsAny(lines[selected-3], "▄▀") || strings.ContainsAny(lines[selected+3], "▄▀") {
		t.Fatal("only the selected row is padded")
	}
}
