package ui

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

type fakeProviderCatalog struct {
	providers     []api.Provider
	components    map[string][]api.Component
	providersErr  error
	componentsErr error
	listings      *[]string
}

func (f fakeProviderCatalog) ListProviders(context.Context) ([]api.Provider, error) {
	return f.providers, f.providersErr
}

// ListProviderComponents answers like the API: a query keeps the components whose name or group
// contains it, ignoring case.
func (f fakeProviderCatalog) ListProviderComponents(_ context.Context, providerSlug, query string) ([]api.Component, error) {
	if query != "" {
		panic("the picker filters a provider's components in memory and must never send a query")
	}
	if f.listings != nil {
		*f.listings = append(*f.listings, providerSlug)
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
		providers: []api.Provider{
			{Name: "GitHub", Slug: "github", Category: "Developer Tools"},
			{Name: "Cloudflare", Slug: "cloudflare", Category: "Cloud Infrastructure"},
		},
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
	keyR      = tea.KeyPressMsg{Code: 'r', Text: "r"}
	keyS      = tea.KeyPressMsg{Code: 's', Text: "s"}
	keyG      = tea.KeyPressMsg{Code: 'g', Text: "g"}
	keySlash  = tea.KeyPressMsg{Code: '/', Text: "/"}
	keyCtrlC  = tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl}
)

func newTestCache() *componentCache {
	return newComponentCache(time.Now)
}

func newTestPicker(backend fakePickerBackend, watched ...api.Component) WatchPickerModel {
	return newTestPickerWithCache(backend, newTestCache(), watched...)
}

func newTestPickerWithCache(backend fakePickerBackend, cache *componentCache, watched ...api.Component) WatchPickerModel {
	return NewWatchPickerModel(backend, cache, openWatchPickerMsg{appSlug: "web-app", appName: "Web App", watchedComponents: watched}, 100, 30)
}

// settleCursor lets the cursor rest where it is, which is what makes the picker ask the API about
// the provider under it.
func settleCursor(m WatchPickerModel) (WatchPickerModel, tea.Cmd) {
	return m.Update(providerSettledMsg{sequence: m.settleSequence})
}

// restAndLoad lets the cursor rest where it is and lets the answer for that provider arrive.
func restAndLoad(m WatchPickerModel) WatchPickerModel {
	m, fetch := settleCursor(m)
	return settle(m, fetch)
}

// moveToProvider presses a key on the providers pane, waits for the cursor to rest and lets the
// answer arrive.
func moveToProvider(m WatchPickerModel, key tea.KeyPressMsg) WatchPickerModel {
	m, _ = m.Update(key)
	return restAndLoad(m)
}

// assertNoSideEffects runs the commands and fails if any of them saves, quits, closes the
// picker or asks the API for anything.
func assertNoSideEffects(t *testing.T, cmds ...tea.Cmd) {
	t.Helper()
	for _, cmd := range cmds {
		for _, msg := range collectMessages(cmd) {
			switch msg.(type) {
			case watchedComponentsSavedMsg, saveWatchedComponentsFailedMsg, tea.QuitMsg, closeWatchPickerMsg,
				providerComponentsLoadedMsg, providerComponentsFailedMsg, providersLoadedMsg, providersFailedMsg:
				t.Fatalf("a command had a side effect: %#v", msg)
			}
		}
	}
}

// typeRunes types text into whichever box has the focus and returns the commands the keys gave.
func typeRunes(m WatchPickerModel, text string) (WatchPickerModel, []tea.Cmd) {
	var cmds []tea.Cmd
	for _, r := range text {
		var cmd tea.Cmd
		m, cmd = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		cmds = append(cmds, cmd)
	}
	return m, cmds
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
	var listings []string
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog()}
	backend.listings = &listings

	view := openedPicker(backend).View().Content

	assertViewContains(t, view, "GitHub", "Cloudflare", "GitHub components", "API Requests", "Pages", "Core")
	if len(listings) != 1 || listings[0] != "github" {
		t.Fatalf("listings = %+v", listings)
	}
}

func TestProvidersShowHowManyOfTheAppsComponentsTheyContribute(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()},
		component("github", "api"), component("github", "pages"), component("cloudflare", "cdn"))

	view := plain(t, m.View().Content)

	for _, want := range []string{`GitHub\s+2 watched`, `Cloudflare\s+1 watched`} {
		if !regexp.MustCompile(want).MatchString(view) {
			t.Errorf("view missing %q:\n%s", want, view)
		}
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

func TestLettersTypedIntoTheComponentFilterAreTextNotShortcuts(t *testing.T) {
	var saved [][]api.ComponentRef
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog(), fakeWatchedComponentsSetter: fakeWatchedComponentsSetter{saved: &saved}}
	m := changedPicker(backend)
	m, _ = m.Update(keySlash)

	m, cmds := typeRunes(m, "sqrj")

	assertNoSideEffects(t, cmds...)
	assertViewContains(t, m.View().Content, "/ sqrj")
	if m.isSaving || len(saved) != 0 {
		t.Fatal("s must not save while typing")
	}
}

func TestAnAnswerForAProviderTheUserLeftIsKeptButNotShown(t *testing.T) {
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog()}
	m := newTestPicker(backend)
	m, _ = m.Update(providersLoadedMsg{providers: backend.providers})
	m, _ = m.Update(keyJ)

	m, _ = m.Update(providerComponentsLoadedMsg{
		providerSlug: "github",
		components:   backend.components["github"],
	})

	assertViewContains(t, m.View().Content, "Cloudflare components", "Loading…")
	assertViewLacks(t, m.View().Content, "API Requests")
	m, _ = m.Update(keyK)
	m, backToGithub := settleCursor(m)
	if backToGithub != nil {
		t.Fatal("the late answer must have been cached")
	}
	assertViewContains(t, m.View().Content, "API Requests")
}

func TestAProviderAlreadyLoadedIsNotFetchedAgain(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m = moveToProvider(m, keyJ)

	m, _ = m.Update(keyK)
	_, backToGithub := settleCursor(m)

	if backToGithub != nil {
		t.Fatal("github is cached")
	}
}

func TestAFailedProviderFetchShowsTheErrorAndIsNotRetriedUntilRIsPressed(t *testing.T) {
	var listings []string
	catalog := testCatalog()
	catalog.componentsErr = errors.New("connection refused")
	catalog.listings = &listings
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: catalog})
	assertViewContains(t, m.View().Content, "connection refused", "Press r to try again")
	m = moveToProvider(m, keyJ)

	m, _ = m.Update(keyK)
	m, backToGithub := settleCursor(m)
	if backToGithub != nil || len(listings) != 2 {
		t.Fatalf("a refused request must not be repeated by visiting again: listings = %+v", listings)
	}
	_, retry := m.Update(keyR)

	if retry == nil {
		t.Fatal("r must ask again")
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
		providerSlug: "github",
		components:   testCatalog().components["github"][:1],
	})

	assertViewContains(t, m.View().Content, "▸ [ ] API Requests")
}

func TestALongListScrollsWithTheCursorInAShortWindow(t *testing.T) {
	catalog := testCatalog()
	catalog.components["github"] = nil
	for index := range 20 {
		catalog.components["github"] = append(catalog.components["github"], namedComponent("github", fmt.Sprintf("c%02d", index), fmt.Sprintf("Component %02d", index), nil))
	}
	m := NewWatchPickerModel(fakePickerBackend{fakeProviderCatalog: catalog}, newTestCache(), openWatchPickerMsg{appSlug: "web-app", appName: "Web App"}, 100, 12)
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
	m := NewWatchPickerModel(fakePickerBackend{fakeProviderCatalog: testCatalog()}, newTestCache(), openWatchPickerMsg{appSlug: "web-app", appName: "Web App"}, 100, 4)
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

func catalogOf(providerCount int) fakeProviderCatalog {
	catalog := fakeProviderCatalog{components: map[string][]api.Component{}}
	for index := range providerCount {
		slug := fmt.Sprintf("p%02d", index)
		catalog.providers = append(catalog.providers, api.Provider{Name: slug, Slug: slug})
		catalog.components[slug] = []api.Component{namedComponent(slug, "a", "A", nil)}
	}
	return catalog
}

func TestScrollingThroughTheCatalogAsksTheAPIOnlyWhereTheCursorRests(t *testing.T) {
	var listings []string
	catalog := catalogOf(87)
	catalog.listings = &listings
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: catalog})
	asked := len(listings)

	var staleSettles []providerSettledMsg
	for range 86 {
		m, _ = m.Update(keyJ)
		staleSettles = append(staleSettles, providerSettledMsg{sequence: m.settleSequence})
	}
	for _, settled := range staleSettles[:len(staleSettles)-1] {
		var fetch tea.Cmd
		m, fetch = m.Update(settled)
		if fetch != nil {
			t.Fatal("a settle from a position the cursor already left must not fetch")
		}
	}
	m, fetch := settleCursor(m)
	m = settle(m, fetch)

	if got := len(listings) - asked; got != 1 {
		t.Fatalf("scrolling through 87 providers asked the API %d times, want 1", got)
	}
	assertViewContains(t, m.View().Content, "p86 components")
}

func TestGoingBackAndForthBeforeAnAnswerArrivesAsksOnce(t *testing.T) {
	var listings []string
	catalog := testCatalog()
	catalog.listings = &listings
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: catalog})
	asked := len(listings)

	m, _ = m.Update(keyJ)
	m, firstVisit := settleCursor(m) // the answer for Cloudflare has not arrived yet
	m, _ = m.Update(keyK)
	m, _ = settleCursor(m)
	m, _ = m.Update(keyJ)
	_, secondVisit := settleCursor(m)

	if firstVisit == nil || secondVisit != nil {
		t.Fatalf("first visit asked = %v, second visit asked = %v", firstVisit != nil, secondVisit != nil)
	}
	settle(m, firstVisit)
	if got := len(listings) - asked; got != 1 {
		t.Fatalf("asked the API %d times for Cloudflare, want 1", got)
	}
}

func TestRRefreshesTheProviderOnScreenAndKeepsItsRowsUntilTheAnswer(t *testing.T) {
	var listings []string
	catalog := testCatalog()
	catalog.listings = &listings
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: catalog})

	m, refresh := m.Update(keyR)
	if refresh == nil {
		t.Fatal("r must ask again")
	}
	assertViewContains(t, m.View().Content, "API Requests")
	_, again := m.Update(keyR)
	if again != nil {
		t.Fatal("a second r while the first is on its way must not ask twice")
	}
	settle(m, refresh)

	if len(listings) != 2 {
		t.Fatalf("listings = %+v", listings)
	}
}

func TestRTriesAFailedProviderListAgain(t *testing.T) {
	catalog := testCatalog()
	catalog.providersErr = errors.New("connection refused")
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: catalog})

	_, retry := m.Update(keyR)

	if retry == nil {
		t.Fatal("r must fetch the providers again")
	}
}

func TestReopeningThePickerReusesWhatTheCacheAlreadyHolds(t *testing.T) {
	var listings []string
	catalog := testCatalog()
	catalog.listings = &listings
	cache := newTestCache()
	first := newTestPickerWithCache(fakePickerBackend{fakeProviderCatalog: catalog}, cache)
	settle(first, first.Init())
	asked := len(listings)

	second := newTestPickerWithCache(fakePickerBackend{fakeProviderCatalog: catalog}, cache)
	cmd := second.Init()

	if cmd != nil || len(listings) != asked {
		t.Fatal("everything the second picker shows is already cached")
	}
	assertViewContains(t, second.View().Content, "GitHub", "Cloudflare", "API Requests")
}

func TestSlashOnTheProvidersFiltersThemLiveWithoutAskingTheAPI(t *testing.T) {
	var listings []string
	catalog := testCatalog()
	catalog.listings = &listings
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: catalog})
	asked := len(listings)

	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "cloud")

	view := m.View().Content
	assertViewContains(t, view, "Cloudflare components", "▸ Cloudflare")
	assertViewLacks(t, view, "GitHub")
	if len(listings) != asked {
		t.Fatalf("filtering asked the API: %+v", listings[asked:])
	}
}

func TestTheProviderFilterMatchesSlugAndCategoryToo(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)

	m = typeIntoPicker(m, "dev")
	assertViewContains(t, m.View().Content, "GitHub")
	assertViewLacks(t, m.View().Content, "Cloudflare")
	m = typeIntoPicker(m, "zzz")

	assertViewContains(t, m.View().Content, "No providers match.")
}

func TestOnlyTheProviderWhereTheCursorLandsIsFetchedAfterTheSettleDelay(t *testing.T) {
	var listings []string
	catalog := catalogOf(87)
	catalog.listings = &listings
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: catalog})
	asked := len(listings)
	m, _ = m.Update(keySlash)

	m, firstKeyCmds := typeRunes(m, "p")
	firstKeySequence := m.settleSequence
	m, _ = typeRunes(m, "8")

	if firstKeyCmds[0] == nil || m.settleSequence == firstKeySequence {
		t.Fatal("each change of the filter must start a new settle timer")
	}
	m, staleFetch := m.Update(providerSettledMsg{sequence: firstKeySequence})
	if staleFetch != nil {
		t.Fatal("a timer from before the last keystroke must not fetch")
	}
	m = restAndLoad(m)

	if got := listings[asked:]; len(got) != 1 || got[0] != "p80" {
		t.Fatalf("fetched %v after typing p8", got)
	}
	assertViewContains(t, m.View().Content, "p80 components")
}

func TestEnterKeepsTheProviderFilterAndTheKeysWorkAgain(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "c")

	m, _ = m.Update(keyEnter)

	assertViewContains(t, m.View().Content, "Filter: c (esc clears)", "Cloudflare")
	assertViewLacks(t, m.View().Content, "GitHub")
	m, fetch := settleCursor(m)
	m = settle(m, fetch)
	m, _ = m.Update(keyTab)
	m, _ = m.Update(keySpace)
	assertViewContains(t, m.View().Content, "[+] CDN", "1 change not saved yet")
}

func TestEscapeInTheProviderFilterClearsItAndPutsTheCursorOnTheFirstProvider(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "cloud")
	m, _ = m.Update(keyEnter)

	m, _ = m.Update(keySlash)
	m, _ = m.Update(keyEscape)

	assertViewContains(t, m.View().Content, "GitHub", "Cloudflare", "▸ GitHub")
	assertViewLacks(t, m.View().Content, "Filter:")
}

func TestLettersTypedIntoTheProviderFilterAreTextNotShortcuts(t *testing.T) {
	var saved [][]api.ComponentRef
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog(), fakeWatchedComponentsSetter: fakeWatchedComponentsSetter{saved: &saved}}
	m := openedPicker(backend, component("github", "api"))
	m, _ = m.Update(keySlash)

	m, cmds := typeRunes(m, "sqrj")

	assertNoSideEffects(t, cmds...)
	assertViewContains(t, m.View().Content, "/ sqrj", "No providers match.")
	if m.isSaving || len(saved) != 0 {
		t.Fatal("s must not save while typing")
	}
}

func TestSlashOnTheComponentsFiltersComponentsNotProviders(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag")

	view := m.View().Content
	assertViewContains(t, view, "GitHub", "Cloudflare", "/ pag", "Pages")
	assertViewLacks(t, view, "API Requests")
}

func TestKeysAreHarmlessWhenTheFilterLeavesNoProviders(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()}, component("github", "api"))
	m, _ = m.Update(keySlash)
	m, _ = typeRunes(m, "zzz")
	m, _ = m.Update(keyEnter)

	var cmds []tea.Cmd
	for _, key := range []tea.KeyPressMsg{keyJ, keyK, keyTab, keySpace, keyR, keyEnter} {
		var cmd tea.Cmd
		m, cmd = m.Update(key)
		cmds = append(cmds, cmd)
	}

	assertNoSideEffects(t, cmds...)
	assertViewContains(t, m.View().Content, "No providers match.")
}

func TestTheProviderPaneStaysTheSameWidthWhileFiltering(t *testing.T) {
	// Names longer than the pane's minimum width decide its width; filtering them away must not
	// make the pane jump while the user types.
	catalog := fakeProviderCatalog{
		providers:  []api.Provider{{Name: strings.Repeat("L", 40), Slug: "long"}, {Name: "Short", Slug: "short"}},
		components: map[string][]api.Component{},
	}
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: catalog})
	wide := m.providersWidth()

	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "short")

	if wide <= minProvidersWidth || m.providersWidth() != wide {
		t.Fatalf("width went from %d to %d (minimum %d): the pane would jump while the user types", wide, m.providersWidth(), minProvidersWidth)
	}
}

func TestTheComponentFilterNarrowsAsTheUserTypesWithoutAskingTheAPI(t *testing.T) {
	var listings []string
	catalog := testCatalog()
	catalog.listings = &listings
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: catalog})
	asked := len(listings)

	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "p")
	assertViewContains(t, m.View().Content, "API Requests", "Pages")
	m = typeIntoPicker(m, "ag")

	view := m.View().Content
	assertViewContains(t, view, "/ pag", "Pages")
	assertViewLacks(t, view, "API Requests")
	if len(listings) != asked {
		t.Fatalf("filtering asked the API: %v", listings[asked:])
	}
}

func TestTheComponentFilterMatchesTheGroupNameIgnoringCase(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)

	m = typeIntoPicker(m, "CORE")

	assertViewContains(t, m.View().Content, "API Requests")
	assertViewLacks(t, m.View().Content, "Pages")
}

func TestEnterKeepsTheComponentFilterAndSpaceActsOnTheFilteredRow(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag")

	m, _ = m.Update(keyEnter)
	m, _ = m.Update(keySpace)

	assertViewContains(t, m.View().Content, "Filter: pag", "[+] Pages", "1 change not saved yet")
	assertViewLacks(t, m.View().Content, "API Requests")
}

func TestEscapeInTheComponentFilterBringsAllTheComponentsBackWithoutAskingTheAPI(t *testing.T) {
	var listings []string
	catalog := testCatalog()
	catalog.listings = &listings
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: catalog})
	asked := len(listings)
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag")

	m, cmd := m.Update(keyEscape)

	if cmd != nil || len(listings) != asked {
		t.Fatal("clearing a filter must not ask the API")
	}
	assertViewContains(t, m.View().Content, "API Requests", "Pages")
	assertViewLacks(t, m.View().Content, "Filter:")
}

func TestAFilterOfOnlySpacesFiltersNothing(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)

	m = typeIntoPicker(m, "   ")
	m, _ = m.Update(keyEnter)

	assertViewContains(t, m.View().Content, "API Requests", "Pages")
	assertViewLacks(t, m.View().Content, "Filter:")
}

func TestAComponentFilterWithNoMatchSaysSo(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)

	m = typeIntoPicker(m, "zzz")

	assertViewContains(t, m.View().Content, "No components match.")
}

func TestTheCursorGoesToTheFirstMatchWhileTyping(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keyJ)
	m, _ = m.Update(keySlash)

	m = typeIntoPicker(m, "a")

	assertViewContains(t, m.View().Content, "▸ [ ] API Requests")
}

func TestAComponentFilterBelongsToTheProviderItWasTypedFor(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag")
	m, _ = m.Update(keyEnter)
	m, _ = m.Update(keyTab)

	m = moveToProvider(m, keyJ)

	assertViewContains(t, m.View().Content, "Cloudflare components", "CDN")
	assertViewLacks(t, m.View().Content, "Filter:", "No components match.")
}

func TestFilteringBeforeTheComponentsHaveLoadedNarrowsThemWhenTheyArrive(t *testing.T) {
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog()}
	m := newTestPicker(backend)
	m, _ = m.Update(providersLoadedMsg{providers: backend.providers})
	m, _ = m.Update(keyTab)
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag")
	assertViewContains(t, m.View().Content, "Loading…")

	m, _ = m.Update(providerComponentsLoadedMsg{providerSlug: "github", components: backend.components["github"]})

	assertViewContains(t, m.View().Content, "Pages")
	assertViewLacks(t, m.View().Content, "API Requests")
}

func TestSavingKeepsTheWatchedComponentsAFilterIsHiding(t *testing.T) {
	var saved [][]api.ComponentRef
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog(), fakeWatchedComponentsSetter: fakeWatchedComponentsSetter{saved: &saved}}
	m := pickerOnComponents(backend, component("github", "api"), component("github", "pages"))
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag") // API Requests is hidden, but still watched
	m, _ = m.Update(keyEnter)
	m, _ = m.Update(keySpace) // unwatch Pages

	_, cmd := m.Update(keyS)
	cmd()

	want := []api.ComponentRef{{Provider: "github", ID: "api"}}
	if len(saved) != 1 || !reflect.DeepEqual(saved[0], want) {
		t.Fatalf("saved = %+v, want %+v: the hidden component must stay in the set", saved, want)
	}
}

func TestAnAnswerForAProviderTheFilterHidesIsKeptAndNotAskedForAgain(t *testing.T) {
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog()}
	m := newTestPicker(backend)
	m, _ = m.Update(providersLoadedMsg{providers: backend.providers}) // GitHub is on its way
	m, _ = m.Update(keySlash)
	m, _ = typeRunes(m, "cloud") // hides GitHub

	m, _ = m.Update(providerComponentsLoadedMsg{providerSlug: "github", components: backend.components["github"]})

	assertViewContains(t, m.View().Content, "Cloudflare components", "Loading…")
	assertViewLacks(t, m.View().Content, "API Requests")
	m, _ = m.Update(keyEscape) // clearing the filter lands on GitHub again
	m, fetch := settleCursor(m)
	if fetch != nil {
		t.Fatal("the answer that arrived while the provider was hidden must have been kept")
	}
	assertViewContains(t, m.View().Content, "API Requests")
}

func TestARequestStillOnItsWayIsNotRepeatedAfterTheFilterHidAndShowedItsProvider(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keyJ)
	m, onItsWay := settleCursor(m) // Cloudflare is asked for, the answer has not arrived
	m, _ = m.Update(keySlash)
	m, _ = typeRunes(m, "git")
	m, _ = m.Update(keyEnter) // hides Cloudflare
	m, _ = m.Update(keySlash)
	m, _ = m.Update(keyEscape) // shows it again, the cursor is back on GitHub

	m, _ = m.Update(keyJ)
	_, again := m.Update(providerSettledMsg{sequence: m.settleSequence})

	if onItsWay == nil || again != nil {
		t.Fatalf("first ask = %v, second ask = %v", onItsWay != nil, again != nil)
	}
}

func TestRefreshingKeepsTheComponentFilter(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag")
	m, _ = m.Update(keyEnter)

	m, refresh := m.Update(keyR)
	m = settle(m, refresh)

	assertViewContains(t, m.View().Content, "Filter: pag", "Pages")
	assertViewLacks(t, m.View().Content, "API Requests")
}

func TestSlashAgainShowsTheKeptProviderFilterAndEditingItRefilters(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "clx")
	m, _ = m.Update(keyEnter)
	assertViewContains(t, m.View().Content, "No providers match.")

	m, _ = m.Update(keySlash)
	assertViewContains(t, m.View().Content, "/ clx")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})

	assertViewContains(t, m.View().Content, "/ cl", "▸ Cloudflare")
	assertViewLacks(t, m.View().Content, "No providers match.")
}

func TestTheTwoFiltersAreIndependent(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "cloud")
	m, _ = m.Update(keyEnter)
	m = restAndLoad(m)
	m, _ = m.Update(keyTab)
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "cdn")
	m, _ = m.Update(keyEnter)
	assertViewContains(t, m.View().Content, "Filter: cloud", "Filter: cdn")

	m, _ = m.Update(keySlash)
	m, _ = m.Update(keyEscape) // clears the components filter only

	assertViewContains(t, m.View().Content, "Filter: cloud", "CDN")
	assertViewLacks(t, m.View().Content, "Filter: cdn")
}

func TestOpeningAFilterAndLeavingItWithoutTypingChangesNothing(t *testing.T) {
	for _, leave := range []tea.KeyPressMsg{keyEnter, keyEscape} {
		t.Run(leave.String(), func(t *testing.T) {
			m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})

			m, _ = m.Update(keySlash)
			m, _ = m.Update(leave)

			assertViewContains(t, m.View().Content, "/ to filter providers", "GitHub", "Cloudflare")
			assertViewLacks(t, m.View().Content, "Filter:")
		})
	}
}

func TestMovingPastTheEndsOfTheProvidersDoesNothing(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	_, atTheTop := m.Update(keyK)
	m = moveToProvider(m, keyJ)
	_, atTheBottom := m.Update(keyJ)

	if atTheTop != nil || atTheBottom != nil {
		t.Fatal("a cursor that did not move must not start a settle timer")
	}
}

// groupedCatalog has GitHub components in two groups plus one without a group, and a Cloudflare
// with no groups at all.
func groupedCatalog() fakeProviderCatalog {
	core, developer := stringPtr("Core"), stringPtr("Developer")
	return fakeProviderCatalog{
		providers: []api.Provider{{Name: "GitHub", Slug: "github"}, {Name: "Cloudflare", Slug: "cloudflare"}},
		components: map[string][]api.Component{
			// In the order the API sends them: by name.
			"github": {
				namedComponent("github", "api", "API Requests", core),
				namedComponent("github", "git", "Git Operations", core),
				namedComponent("github", "pages", "Pages", nil),
				namedComponent("github", "hooks", "Webhooks", developer),
			},
			"cloudflare": {namedComponent("cloudflare", "cdn", "CDN", nil)},
		},
	}
}

func groupedPicker(watched ...api.Component) WatchPickerModel {
	return pickerOnComponents(fakePickerBackend{fakeProviderCatalog: groupedCatalog()}, watched...)
}

func pressTimes(m WatchPickerModel, key tea.KeyPressMsg, times int) WatchPickerModel {
	for range times {
		m, _ = m.Update(key)
	}
	return m
}

func TestComponentsAreListedUnderGroupHeadingsWithTheUngroupedLast(t *testing.T) {
	view := plain(t, groupedPicker().View().Content)

	assertViewContains(t, view, "Core  0/2 watched", "Developer  0/1 watched", "Ungrouped  0/1 watched")
	core, developer, ungrouped := strings.Index(view, "Core  0/2"), strings.Index(view, "Developer  0/1"), strings.Index(view, "Ungrouped  0/1")
	if core >= developer || developer >= ungrouped {
		t.Fatalf("headings out of order (%d, %d, %d):\n%s", core, developer, ungrouped, view)
	}
	if strings.Index(view, "API Requests") > developer || strings.Index(view, "Pages") < ungrouped {
		t.Fatalf("components are not under their headings:\n%s", view)
	}
}

func TestAProviderWithoutGroupsHasNoHeadings(t *testing.T) {
	m := moveToProvider(openedPicker(fakePickerBackend{fakeProviderCatalog: groupedCatalog()}), keyJ)

	assertViewContains(t, m.View().Content, "Cloudflare components", "CDN")
	assertViewLacks(t, m.View().Content, "Ungrouped", "0/1 watched")
}

func TestARowDoesNotRepeatItsGroupNameTheHeadingSaysIt(t *testing.T) {
	for _, line := range strings.Split(plain(t, groupedPicker().View().Content), "\n") {
		if strings.Contains(line, "API Requests") && strings.Contains(line, "Core") {
			t.Fatalf("a component row carries its group: %q", line)
		}
	}
}

func TestTheCursorSkipsTheHeadings(t *testing.T) {
	m := groupedPicker()
	assertViewContains(t, m.View().Content, "▸ [ ] API Requests")

	m = pressTimes(m, keyJ, 2)
	assertViewContains(t, m.View().Content, "▸ [ ] Webhooks")
	m = pressTimes(m, keyJ, 1)
	assertViewContains(t, m.View().Content, "▸ [ ] Pages")
	m = pressTimes(m, keyJ, 5)
	assertViewContains(t, m.View().Content, "▸ [ ] Pages")
	m = pressTimes(m, keyK, 2)
	assertViewContains(t, m.View().Content, "▸ [ ] Git Operations")
}

func TestAHeadingCountsTheComponentsChosenInItsGroup(t *testing.T) {
	m := groupedPicker()

	m, _ = m.Update(keySpace)

	assertViewContains(t, m.View().Content, "Core  1/2 watched")
}

func TestGTogglesTheWholeGroupAndAgainBringsItBack(t *testing.T) {
	m := groupedPicker()

	m, _ = m.Update(keyG)
	assertViewContains(t, m.View().Content, "[+] API Requests", "[+] Git Operations", "Core  2/2 watched", "2 changes not saved yet")
	assertViewLacks(t, m.View().Content, "[+] Webhooks")
	m, _ = m.Update(keyG)

	assertViewContains(t, m.View().Content, "[ ] API Requests", "[ ] Git Operations", "Core  0/2 watched")
	assertViewLacks(t, m.View().Content, "not saved yet")
}

func TestGChoosesTheRestOfAGroupThatIsPartlyChosen(t *testing.T) {
	m := groupedPicker(component("github", "api"))

	m, _ = m.Update(keyG)

	assertViewContains(t, m.View().Content, "[x] API Requests", "[+] Git Operations", "Core  2/2 watched", "1 change not saved yet")
}

func TestGUnchoosesAGroupThatIsFullyWatched(t *testing.T) {
	m := groupedPicker(component("github", "api"), component("github", "git"))

	m, _ = m.Update(keyG)

	assertViewContains(t, m.View().Content, "[-] API Requests", "[-] Git Operations", "Core  0/2 watched", "2 changes not saved yet")
}

func TestGActsOnlyOnTheComponentsAFilterLetsThrough(t *testing.T) {
	m := groupedPicker()
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "oper") // Git Operations only; API Requests is in the same group but hidden
	m, _ = m.Update(keyEnter)

	m, _ = m.Update(keyG)
	m, _ = m.Update(keySlash)
	m, _ = m.Update(keyEscape)

	assertViewContains(t, m.View().Content, "[+] Git Operations", "[ ] API Requests", "Core  1/2 watched")
}

func TestGDoesNothingOutsideTheComponentsPane(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: groupedCatalog()})

	m, _ = m.Update(keyG)

	assertViewLacks(t, m.View().Content, "not saved yet")
}

func TestGBeforeTheComponentsHaveLoadedDoesNothing(t *testing.T) {
	backend := fakePickerBackend{fakeProviderCatalog: groupedCatalog()}
	m := newTestPicker(backend)
	m, _ = m.Update(providersLoadedMsg{providers: backend.providers})
	m, _ = m.Update(keyTab)

	m, _ = m.Update(keyG)

	assertViewContains(t, m.View().Content, "Loading…")
	assertViewLacks(t, m.View().Content, "not saved yet")
}

func TestSavingAfterGSendsTheWholeSetIncludingOtherGroupsAndProviders(t *testing.T) {
	var saved [][]api.ComponentRef
	backend := fakePickerBackend{fakeProviderCatalog: groupedCatalog(), fakeWatchedComponentsSetter: fakeWatchedComponentsSetter{saved: &saved}}
	m := pickerOnComponents(backend, component("github", "hooks"), component("cloudflare", "cdn"))
	m, _ = m.Update(keyG)

	_, cmd := m.Update(keyS)
	cmd()

	want := []api.ComponentRef{{Provider: "cloudflare", ID: "cdn"}, {Provider: "github", ID: "api"}, {Provider: "github", ID: "git"}, {Provider: "github", ID: "hooks"}}
	if len(saved) != 1 || !reflect.DeepEqual(saved[0], want) {
		t.Fatalf("saved = %+v, want %+v", saved, want)
	}
}

func TestTheFilterRegroupsItsResults(t *testing.T) {
	m := groupedPicker()
	m, _ = m.Update(keySlash)

	m = typeIntoPicker(m, "core")

	assertViewContains(t, m.View().Content, "Core  0/2 watched", "API Requests", "Git Operations")
	assertViewLacks(t, m.View().Content, "Developer", "Ungrouped", "Pages")
}

func TestAGroupedListScrollsWithTheCursorAndTheHeadingsTakeRows(t *testing.T) {
	catalog := fakeProviderCatalog{
		providers:  []api.Provider{{Name: "Big", Slug: "big"}},
		components: map[string][]api.Component{"big": nil},
	}
	for _, group := range []string{"A", "B", "C"} {
		for index := range 6 {
			id := fmt.Sprintf("%s%d", strings.ToLower(group), index)
			catalog.components["big"] = append(catalog.components["big"], namedComponent("big", id, group+fmt.Sprint(index), stringPtr(group)))
		}
	}
	m := NewWatchPickerModel(fakePickerBackend{fakeProviderCatalog: catalog}, newTestCache(), openWatchPickerMsg{appSlug: "x", appName: "X"}, 100, 13)
	m = settle(m, m.Init())
	m, _ = m.Update(keyTab)
	assertViewContains(t, m.View().Content, "A  0/6 watched", "A0")
	assertViewLacks(t, m.View().Content, "C5")

	m = pressTimes(m, keyJ, 17)

	assertViewContains(t, m.View().Content, "▸ [ ] C5")
	assertViewLacks(t, m.View().Content, "A0")
}
