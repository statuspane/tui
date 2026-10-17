package ui

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

type fakeProviderCatalog struct {
	providers     []api.Provider
	components    map[string][]api.Component
	providersErr  error
	componentsErr error
	listings      *[]componentListing
}

func (f fakeProviderCatalog) ListProviders(context.Context) ([]api.Provider, error) {
	return f.providers, f.providersErr
}

// ListProviderComponents answers like the API: a query keeps the components whose name or group
// contains it, ignoring case.
func (f fakeProviderCatalog) ListProviderComponents(_ context.Context, providerSlug, query string) ([]api.Component, error) {
	if f.listings != nil {
		*f.listings = append(*f.listings, componentListing{providerSlug: providerSlug, query: query})
	}
	if f.componentsErr != nil {
		return nil, f.componentsErr
	}
	var matches []api.Component
	for _, candidate := range f.components[providerSlug] {
		group := ""
		if candidate.GroupName != nil {
			group = *candidate.GroupName
		}
		if strings.Contains(strings.ToLower(candidate.Name+" "+group), strings.ToLower(query)) {
			matches = append(matches, candidate)
		}
	}
	return matches, nil
}

type fakeWatchedComponentsSetter struct {
	appDetail api.AppDetail
	err       error
	saved     *[][]api.ComponentRef
}

func (f fakeWatchedComponentsSetter) SetWatchedComponents(_ context.Context, _ string, refs []api.ComponentRef) (api.AppDetail, error) {
	if f.saved != nil {
		*f.saved = append(*f.saved, refs)
	}
	return f.appDetail, f.err
}

type fakePickerBackend struct {
	fakeProviderCatalog
	fakeWatchedComponentsSetter
}

func testCatalog() fakeProviderCatalog {
	core := stringPtr("Core")
	return fakeProviderCatalog{
		providers: []api.Provider{{Name: "GitHub", Slug: "github"}, {Name: "Cloudflare", Slug: "cloudflare"}},
		components: map[string][]api.Component{
			"github":     {namedComponent("github", "api", "API Requests", core), namedComponent("github", "pages", "Pages", nil)},
			"cloudflare": {namedComponent("cloudflare", "cdn", "CDN", nil)},
		},
	}
}

var (
	keyTab    = tea.KeyPressMsg{Code: tea.KeyTab}
	keySpace  = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	keyEscape = tea.KeyPressMsg{Code: tea.KeyEscape}
	keyEnter  = tea.KeyPressMsg{Code: tea.KeyEnter}
	keyJ      = tea.KeyPressMsg{Code: 'j', Text: "j"}
	keyK      = tea.KeyPressMsg{Code: 'k', Text: "k"}
	keyS      = tea.KeyPressMsg{Code: 's', Text: "s"}
	keySlash  = tea.KeyPressMsg{Code: '/', Text: "/"}
	keyCtrlC  = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
)

func newTestPicker(backend fakePickerBackend, watched ...api.Component) WatchPickerModel {
	return NewWatchPickerModel(backend, openWatchPickerMsg{appSlug: "web-app", appName: "Web App", watchedComponents: watched}, 100, 30)
}

// settle runs the commands that answer at once and feeds their messages back until none are left.
func settle(m WatchPickerModel, cmd tea.Cmd) WatchPickerModel {
	for _, msg := range collectMessages(cmd) {
		var next tea.Cmd
		m, next = m.Update(msg)
		m = settle(m, next)
	}
	return m
}

func openedPicker(backend fakePickerBackend, watched ...api.Component) WatchPickerModel {
	m := newTestPicker(backend, watched...)
	return settle(m, m.Init())
}

// pickerOnComponents is an opened picker with the focus on the first provider's components.
func pickerOnComponents(backend fakePickerBackend, watched ...api.Component) WatchPickerModel {
	m, _ := openedPicker(backend, watched...).Update(keyTab)
	return m
}

// changedPicker has one watched component (API Requests) toggled off, so it has an unsaved change.
func changedPicker(backend fakePickerBackend) WatchPickerModel {
	m, _ := pickerOnComponents(backend, component("github", "api")).Update(keySpace)
	return m
}

func typeIntoPicker(m WatchPickerModel, text string) WatchPickerModel {
	for _, r := range text {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestThePickerLoadsProvidersAndTheFirstOnesComponents(t *testing.T) {
	var listings []componentListing
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog()}
	backend.listings = &listings

	view := openedPicker(backend).View().Content

	assertViewContains(t, view, "GitHub", "Cloudflare", "GitHub components", "API Requests", "Pages", "Core")
	if len(listings) != 1 || listings[0] != (componentListing{providerSlug: "github"}) {
		t.Fatalf("listings = %+v", listings)
	}
}

func TestProvidersShowHowManyOfTheAppsComponentsTheyContribute(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()},
		component("github", "api"), component("github", "pages"), component("cloudflare", "cdn"))

	view := plain(t, m.View().Content)

	if !strings.Contains(view, "GitHub     2 watched") && !strings.Contains(view, "GitHub  2 watched") {
		// The gap depends on the widest provider name, so match the pair loosely.
		assertViewContains(t, view, "2 watched", "1 watched")
	}
	if strings.Count(view, "watched") != 2 {
		t.Fatalf("want one count per provider that has watched components:\n%s", view)
	}
}

func TestToggleMarksAnAdditionAndCountsTheChange(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()}, component("github", "api"))

	m, _ = m.Update(keyJ)
	m, _ = m.Update(keySpace)

	assertViewContains(t, m.View().Content, "[x] API Requests", "[+] Pages", "1 change not saved yet")
}

func TestToggleMarksARemovalAndTogglingBackClearsIt(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()}, component("github", "api"))

	m, _ = m.Update(keySpace)
	assertViewContains(t, m.View().Content, "[-] API Requests")
	m, _ = m.Update(keySpace)

	assertViewContains(t, m.View().Content, "[x] API Requests")
	assertViewLacks(t, m.View().Content, "not saved yet")
}

func TestSpaceDoesNothingWhileTheProvidersHaveTheFocus(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()}, component("github", "api"))

	m, _ = m.Update(keySpace)

	assertViewLacks(t, m.View().Content, "not saved yet", "[-]")
}

func TestEnterOnTheProvidersMovesTheFocusToTheComponents(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()}, component("github", "api"))

	m, _ = m.Update(keyEnter)
	m, _ = m.Update(keySpace)

	assertViewContains(t, m.View().Content, "[-] API Requests")
}

func TestSavingSendsTheWholeSetIncludingProvidersNeverOpened(t *testing.T) {
	var saved [][]api.ComponentRef
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog(), fakeWatchedComponentsSetter: fakeWatchedComponentsSetter{saved: &saved}}
	m := pickerOnComponents(backend, component("github", "api"), component("cloudflare", "cdn"))
	m, _ = m.Update(keyJ)
	m, _ = m.Update(keySpace)

	_, cmd := m.Update(keyS)
	msg := cmd()

	want := []api.ComponentRef{{Provider: "cloudflare", ID: "cdn"}, {Provider: "github", ID: "api"}, {Provider: "github", ID: "pages"}}
	if _, ok := msg.(watchedComponentsSavedMsg); !ok || len(saved) != 1 || !reflect.DeepEqual(saved[0], want) {
		t.Fatalf("msg = %#v, saved = %+v", msg, saved)
	}
}

func TestSavingAnEmptiedSelectionSendsAnEmptyListNotNull(t *testing.T) {
	var saved [][]api.ComponentRef
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog(), fakeWatchedComponentsSetter: fakeWatchedComponentsSetter{saved: &saved}}
	m := changedPicker(backend)

	_, cmd := m.Update(keyS)
	cmd()

	if len(saved) != 1 || saved[0] == nil || len(saved[0]) != 0 {
		t.Fatalf("saved = %#v", saved)
	}
}

func TestSavingWithoutChangesDoesNothing(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()}, component("github", "api"))

	_, cmd := m.Update(keyS)

	if cmd != nil {
		t.Fatal("nothing to save")
	}
}

func TestPressingSaveTwiceSendsOneRequestAndEscapeWaitsForIt(t *testing.T) {
	m := changedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	m, first := m.Update(keyS)
	m, second := m.Update(keyS)
	_, escape := m.Update(keyEscape)

	if first == nil || second != nil || escape != nil {
		t.Fatalf("first = %v, second = %v, escape = %v", first != nil, second != nil, escape != nil)
	}
	assertViewContains(t, m.View().Content, "Saving…")
}

func TestWhileSavingTheCursorTheSearchAndTheTabKeyAreIgnored(t *testing.T) {
	m := changedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keyS)
	before := m.View().Content

	for _, key := range []tea.KeyPressMsg{keyJ, keyK, keyTab, keySlash, keySpace} {
		var cmd tea.Cmd
		m, cmd = m.Update(key)
		if cmd != nil {
			t.Fatalf("%q returned a command while saving", key.String())
		}
	}

	if m.View().Content != before {
		t.Fatal("keys changed the picker while a save was in flight")
	}
}

func TestCtrlCStillQuitsWhileSaving(t *testing.T) {
	m := changedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keyS)

	_, cmd := m.Update(keyCtrlC)

	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("want tea.QuitMsg")
	}
}

func TestAFailedSaveKeepsTheChangesAndExplainsWhy(t *testing.T) {
	failure := apiError(400, api.CodeValidationFailed, api.FieldError{Field: "watched_components[0]", Code: "unknown_component", Message: "No component github/api"})
	m := changedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog(), fakeWatchedComponentsSetter: fakeWatchedComponentsSetter{err: failure}})
	m, cmd := m.Update(keyS)

	m, _ = m.Update(cmd())

	assertViewContains(t, m.View().Content, "No component github/api", "[-] API Requests")
	if _, retry := m.Update(keyS); retry == nil {
		t.Fatal("must be able to try again")
	}
}

func TestEscapeClosesAtOnceWhenNothingChanged(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	_, cmd := m.Update(keyEscape)

	if _, ok := cmd().(closeWatchPickerMsg); !ok {
		t.Fatal("want closeWatchPickerMsg")
	}
}

func TestEscapeAsksBeforeDiscardingChangesAndAnyOtherKeyCancelsTheQuestion(t *testing.T) {
	m := changedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	m, first := m.Update(keyEscape)
	if first != nil {
		t.Fatal("the first esc must only ask")
	}
	assertViewContains(t, m.View().Content, "Press esc again to discard")
	m, _ = m.Update(keyJ)
	m, afterOtherKey := m.Update(keyEscape)
	if afterOtherKey != nil {
		t.Fatal("the question must start over after another key")
	}
	_, confirmed := m.Update(keyEscape)

	if _, ok := confirmed().(closeWatchPickerMsg); !ok {
		t.Fatal("the second esc in a row discards")
	}
}

func TestSearchSendsTheQueryOnEnter(t *testing.T) {
	var listings []componentListing
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog()}
	backend.listings = &listings
	m := openedPicker(backend)

	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag")
	m, cmd := m.Update(keyEnter)
	m = settle(m, cmd)

	view := m.View().Content
	assertViewContains(t, view, "Search: pag", "Pages")
	assertViewLacks(t, view, "API Requests")
	want := []componentListing{{providerSlug: "github"}, {providerSlug: "github", query: "pag"}}
	if !reflect.DeepEqual(listings, want) {
		t.Fatalf("listings = %+v, want %+v", listings, want)
	}
}

func TestEscapeInTheSearchClearsItWithoutAskingTheAPIAgain(t *testing.T) {
	var listings []componentListing
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog()}
	backend.listings = &listings
	m := openedPicker(backend)
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag")
	m, cmd := m.Update(keyEnter)
	m = settle(m, cmd)

	m, _ = m.Update(keySlash)
	m, cmd = m.Update(keyEscape)

	if cmd != nil {
		t.Fatal("the unfiltered list is cached")
	}
	assertViewContains(t, m.View().Content, "API Requests")
	assertViewLacks(t, m.View().Content, "Search: pag")
}

func TestASearchOfOnlySpacesIsNoSearch(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "   ")

	m, cmd := m.Update(keyEnter)

	if cmd != nil {
		t.Fatal("an empty search is the cached full list")
	}
	assertViewLacks(t, m.View().Content, "Search:")
}

func TestLettersTypedIntoTheSearchBoxAreTextNotShortcuts(t *testing.T) {
	var saved [][]api.ComponentRef
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog(), fakeWatchedComponentsSetter: fakeWatchedComponentsSetter{saved: &saved}}
	m := changedPicker(backend)
	m, _ = m.Update(keySlash)

	m, saveCmd := m.Update(keyS)
	m, quitCmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})

	// The text box may answer with a cursor command; what must not happen is a save or a quit.
	for _, msg := range append(collectMessages(saveCmd), collectMessages(quitCmd)...) {
		switch msg.(type) {
		case watchedComponentsSavedMsg, saveWatchedComponentsFailedMsg, tea.QuitMsg:
			t.Fatalf("typing triggered %#v", msg)
		}
	}
	if m.isSaving || len(saved) != 0 {
		t.Fatal("s must not save while typing")
	}
	if m.search.Value() != "sq" {
		t.Fatalf("search = %q", m.search.Value())
	}
}

func TestAnAnswerForAProviderTheUserLeftIsKeptButNotShown(t *testing.T) {
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog()}
	m := newTestPicker(backend)
	m, _ = m.Update(providersLoadedMsg{providers: backend.providers})
	m, _ = m.Update(keyJ)

	m, _ = m.Update(providerComponentsLoadedMsg{
		listing:    componentListing{providerSlug: "github"},
		components: backend.components["github"],
	})

	assertViewContains(t, m.View().Content, "Cloudflare components", "Loading…")
	assertViewLacks(t, m.View().Content, "API Requests")
	m, backToGithub := m.Update(keyK)
	if backToGithub != nil {
		t.Fatal("the late answer must have been cached")
	}
	assertViewContains(t, m.View().Content, "API Requests")
}

func TestAProviderAlreadyLoadedIsNotFetchedAgain(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, cmd := m.Update(keyJ)
	m = settle(m, cmd)

	_, backToGithub := m.Update(keyK)

	if backToGithub != nil {
		t.Fatal("github is cached")
	}
}

func TestAFailedProviderFetchShowsTheErrorAndIsTriedAgainOnReturn(t *testing.T) {
	catalog := testCatalog()
	catalog.componentsErr = errors.New("connection refused")
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: catalog})
	assertViewContains(t, m.View().Content, "connection refused")

	m, away := m.Update(keyJ)
	m = settle(m, away)
	_, back := m.Update(keyK)

	if back == nil {
		t.Fatal("a failed request must be tried again")
	}
}

func TestAFailedProviderListIsShown(t *testing.T) {
	catalog := testCatalog()
	catalog.providersErr = errors.New("connection refused")

	view := openedPicker(fakePickerBackend{fakeProviderCatalog: catalog}).View().Content

	assertViewContains(t, view, "connection refused")
}

func TestAnEmptyProviderListDoesNotBreakTheKeys(t *testing.T) {
	m := openedPicker(fakePickerBackend{})

	for _, key := range []tea.KeyPressMsg{keyJ, keyK, keyTab, keySpace, keySlash, keyEscape, keyS} {
		m, _ = m.Update(key)
	}

	assertViewContains(t, m.View().Content, "Providers")
}

func TestTheCursorStaysOnARowWhenTheListShrinksUnderIt(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keyJ)

	m, _ = m.Update(providerComponentsLoadedMsg{
		listing:    componentListing{providerSlug: "github"},
		components: testCatalog().components["github"][:1],
	})

	assertViewContains(t, m.View().Content, "▸ [ ] API Requests")
}

func TestALongListScrollsWithTheCursorInAShortWindow(t *testing.T) {
	catalog := testCatalog()
	catalog.components["github"] = nil
	for index := range 20 {
		catalog.components["github"] = append(catalog.components["github"], namedComponent("github", fmt.Sprintf("c%02d", index), fmt.Sprintf("Component %02d", index), nil))
	}
	m := NewWatchPickerModel(fakePickerBackend{fakeProviderCatalog: catalog}, openWatchPickerMsg{appSlug: "web-app", appName: "Web App"}, 100, 12)
	m = settle(m, m.Init())
	m, _ = m.Update(keyTab)
	assertViewContains(t, m.View().Content, "Component 00")
	assertViewLacks(t, m.View().Content, "Component 19")

	for range 19 {
		m, _ = m.Update(keyJ)
	}

	assertViewContains(t, m.View().Content, "▸ [ ] Component 19")
	assertViewLacks(t, m.View().Content, "Component 00")
}

func TestAWindowShorterThanTheChromeStillShowsAFewRows(t *testing.T) {
	m := NewWatchPickerModel(fakePickerBackend{fakeProviderCatalog: testCatalog()}, openWatchPickerMsg{appSlug: "web-app", appName: "Web App"}, 100, 4)
	m = settle(m, m.Init())

	assertViewContains(t, m.View().Content, "API Requests", "Pages")
}

func TestCtrlCQuitsFromThePicker(t *testing.T) {
	_, cmd := newTestPicker(fakePickerBackend{}).Update(keyCtrlC)

	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("want tea.QuitMsg")
	}
}

func TestVisibleRangeKeepsTheCursorInView(t *testing.T) {
	cases := []struct {
		name                string
		total, cursor, rows int
		wantStart, wantEnd  int
	}{
		{"everything fits", 3, 2, 10, 0, 3},
		{"cursor at the top", 30, 0, 10, 0, 10},
		{"cursor in the middle", 30, 15, 10, 10, 20},
		{"cursor at the end", 30, 29, 10, 20, 30},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, end := visibleRange(tc.total, tc.cursor, tc.rows)

			if start != tc.wantStart || end != tc.wantEnd {
				t.Fatalf("got %d-%d, want %d-%d", start, end, tc.wantStart, tc.wantEnd)
			}
		})
	}
}

func TestClampIndexStaysInsideTheList(t *testing.T) {
	cases := []struct {
		name                      string
		index, step, length, want int
	}{
		{"moves down", 1, 1, 5, 2},
		{"stops at the end", 4, 1, 5, 4},
		{"stops at the start", 0, -1, 5, 0},
		{"empty list", 0, 1, 0, 0},
		{"pulled back when the list shrinks", 6, 0, 3, 2},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := clampIndex(tc.index, tc.step, tc.length); got != tc.want {
				t.Fatalf("got %d, want %d", got, tc.want)
			}
		})
	}
}
