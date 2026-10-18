package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestThePickerIsTwoTitledPanelsThatFillTheWindow(t *testing.T) {
	view := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()}).View().Content

	assertViewContains(t, view, "◉ STATUSPANE › Web App › Watch components", "╭─ Providers", "╭─ GitHub components · 2")
	assertEveryLineHasWidth(t, view, testPickerWidth)
	assertHeightIs(t, view, testPickerHeight)
}

func TestThePickerRedrawsAtTheNewWindowSize(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(tea.WindowSizeMsg{Width: fallbackWidth, Height: fallbackHeight})

	view := m.View().Content
	assertEveryLineHasWidth(t, view, fallbackWidth)
	assertHeightIs(t, view, fallbackHeight)
}

func TestTheHeaderCountsTheUnsavedChanges(t *testing.T) {
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog()}

	assertViewLacks(t, openedPicker(backend).View().Content, "unsaved")
	assertViewContains(t, changedPicker(backend).View().Content, "● 1 unsaved change")
}

func TestTheProvidersPanelSaysHowManyComponentsAreWatchedInTotal(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()}, component("github", "api"), component("cloudflare", "cdn"))

	assertViewContains(t, m.View().Content, "2 watched in total")
}

func TestTheWatchedTotalFollowsTheSelection(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()}, component("github", "api"), component("cloudflare", "cdn"))

	m, _ = m.Update(keySpace)

	assertViewContains(t, m.View().Content, "1 watched in total")
}

func TestTheComponentsPanelHasTheLegend(t *testing.T) {
	view := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()}).View().Content

	assertViewContains(t, view, "[+] to watch", "[-] to stop", "saved with s")
}

func TestAnIdleFilterBoxShowsNoMatchCount(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	assertViewLacks(t, m.View().Content, "match")
}

func TestAFilterBoxCountsItsMatchesWhileTyping(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag")

	assertViewContains(t, m.View().Content, "1 match · name or group")
}

func TestAKeptFilterStillShowsItsQueryAndMatchCount(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag")

	m, _ = m.Update(keyEnter)

	assertViewContains(t, m.View().Content, "Filter: pag", "1 match")
}

func TestTheProviderFilterBoxCountsItsMatchesToo(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "c")

	assertViewContains(t, m.View().Content, "/ c", "1 match")
}

// providersPanelEdge is what is drawn on the top line of the screen to the left of the providers'
// title: the corner of the providers panel, with its colour.
func providersPanelEdge(t *testing.T, m WatchPickerModel) string {
	t.Helper()
	for _, line := range strings.Split(m.View().Content, "\n") {
		if index := strings.Index(line, "Providers"); index >= 0 {
			return line[:index]
		}
	}
	t.Fatal("no providers panel")
	return ""
}

func TestTheFocusedPanelHasTheAccentEdge(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	if !strings.Contains(providersPanelEdge(t, m), escapeBefore(untintedPanelLook(true).border.Render("x"))) {
		t.Fatal("the providers panel has the focus and should have the accent edge")
	}
}

func TestAPanelThatLostTheFocusIsQuiet(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	m, _ = m.Update(keyTab)

	if !strings.Contains(providersPanelEdge(t, m), escapeBefore(untintedPanelLook(false).border.Render("x"))) {
		t.Fatal("the providers panel lost the focus and should be quiet")
	}
}

func TestTheProvidersFocusHasItsOwnKeyHints(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	assertViewContains(t, m.View().Content, "/ filter", "esc cancel")
	assertViewLacks(t, m.View().Content, "g group")
}

func TestTheComponentsFocusHasItsOwnKeyHints(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	assertViewContains(t, m.View().Content, "space toggle", "g group", "/ filter")
}

func TestWhileFilteringTheKeyBarSaysHowToLeaveTheBox(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	m, _ = m.Update(keySlash)

	assertViewContains(t, m.View().Content, "type to filter", "⏎ keep", "esc clear")
	assertViewLacks(t, m.View().Content, "esc cancel")
}

func TestTheDiscardQuestionReplacesTheKeyHintsUntilAnswered(t *testing.T) {
	m := changedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	m, _ = m.Update(keyEscape)
	assertViewContains(t, m.View().Content, "Discard 1 change? Press esc again to discard")
	assertViewLacks(t, m.View().Content, "switch pane")

	m, _ = m.Update(keyJ)
	assertViewContains(t, m.View().Content, "switch pane")
	assertViewLacks(t, m.View().Content, "Discard")
}

// shortWindowHeight is below what the panels need, so the picker has to make room for its rows.
const (
	shortWindowHeight = 10
	longListSize      = 20
	rowsScrolled      = 12
)

func catalogWithManyComponents() fakeProviderCatalog {
	catalog := testCatalog()
	catalog.components["github"] = nil
	for index := range longListSize {
		catalog.components["github"] = append(catalog.components["github"], namedComponent("github", fmt.Sprint(index), fmt.Sprintf("Component %d", index), nil))
	}
	return catalog
}

func TestAShortWindowStillShowsTheCursorRowAndTheFooter(t *testing.T) {
	m := NewWatchPickerModel(fakePickerBackend{fakeProviderCatalog: catalogWithManyComponents()}, newTestCache(), openWatchPickerMsg{appSlug: "web-app", appName: "Web App"}, testPickerWidth, shortWindowHeight)
	m = settle(m, m.Init())
	m, _ = m.Update(keyTab)
	for range rowsScrolled {
		m, _ = m.Update(keyJ)
	}

	assertViewContains(t, m.View().Content, fmt.Sprintf("▸ [ ] Component %d", rowsScrolled), "watched in total", "switch pane")
}
