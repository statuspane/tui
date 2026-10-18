package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

func TestThePickerIsTwoTitledPanelsThatFillTheWindow(t *testing.T) {
	view := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()}).View().Content

	assertViewContains(t, view, "◉ STATUSPANE › Web App › Watch components", "╭─ Providers", "╭─ GitHub components · 2")
	assertEveryLineHasWidth(t, view, 100)
	assertHeightIs(t, view, 30)
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
	backend := fakePickerBackend{fakeProviderCatalog: testCatalog()}
	m := pickerOnComponents(backend, component("github", "api"), component("cloudflare", "cdn"))

	assertViewContains(t, m.View().Content, "2 watched in total")
	m, _ = m.Update(keySpace)
	assertViewContains(t, m.View().Content, "1 watched in total")
}

func TestTheComponentsPanelHasTheLegend(t *testing.T) {
	view := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()}).View().Content

	assertViewContains(t, view, "[+] to watch", "[-] to stop", "saved with s")
}

func TestAFilterBoxCountsItsMatchesWhileItIsInUse(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	assertViewLacks(t, m.View().Content, "match")

	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag")
	assertViewContains(t, m.View().Content, "1 match · name or group")

	m, _ = m.Update(keyEnter)
	assertViewContains(t, m.View().Content, "Filter: pag", "1 match")
}

func TestTheProviderFilterBoxCountsItsMatchesToo(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "c")

	assertViewContains(t, m.View().Content, "/ c", "1 match")
}

func TestTheFocusedPanelHasTheAccentEdge(t *testing.T) {
	accent := escapeBefore(untintedPanelLook(true).border.Render("x")) + "╭─ "
	quiet := escapeBefore(untintedPanelLook(false).border.Render("x")) + "╭─ "
	// panelTops is the top edge of the providers panel, the one on the left of the line.
	panelTops := func(m WatchPickerModel) string {
		for _, line := range strings.Split(m.View().Content, "\n") {
			if index := strings.Index(line, "Providers"); index >= 0 {
				return line[:index]
			}
		}
		return ""
	}
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})

	if !strings.Contains(panelTops(m), accent) {
		t.Fatal("the providers panel has the focus and should have the accent edge")
	}
	m, _ = m.Update(keyTab)
	if strings.Contains(panelTops(m), accent) || !strings.Contains(panelTops(m), quiet) {
		t.Fatal("the providers panel lost the focus and should be quiet")
	}
}

func TestTheKeyHintsFollowTheFocus(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	assertViewContains(t, m.View().Content, "/ filter", "esc cancel")
	assertViewLacks(t, m.View().Content, "g group")

	m, _ = m.Update(keyTab)
	assertViewContains(t, m.View().Content, "space toggle", "g group", "/ filter")

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

func TestAShortWindowStillShowsTheCursorRowAndKeepsTheFooter(t *testing.T) {
	catalog := testCatalog()
	catalog.components["github"] = nil
	for index := range 20 {
		catalog.components["github"] = append(catalog.components["github"], namedComponent("github", "c"+string(rune('a'+index)), "Component "+string(rune('A'+index)), nil))
	}
	m := NewWatchPickerModel(fakePickerBackend{fakeProviderCatalog: catalog}, newTestCache(), openWatchPickerMsg{appSlug: "web-app", appName: "Web App", watchedComponents: []api.Component{}}, 100, 10)
	m = settle(m, m.Init())
	m, _ = m.Update(keyTab)
	for range 12 {
		m, _ = m.Update(keyJ)
	}

	assertViewContains(t, m.View().Content, "▸ [ ] Component M")
}

func TestEscapeClearsAKeptComponentFilterInsteadOfClosingThePicker(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag")
	m, _ = m.Update(keyEnter)

	m, cmd := m.Update(keyEscape)

	assertNoSideEffects(t, cmd)
	assertViewLacks(t, m.View().Content, "Filter:")
	assertViewContains(t, m.View().Content, "API Requests", "Pages")
}

func TestEscapeClearsAKeptProviderFilterInsteadOfClosingThePicker(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "cloud")
	m, _ = m.Update(keyEnter)

	m, cmd := m.Update(keyEscape)

	assertNoSideEffects(t, cmd)
	assertViewLacks(t, m.View().Content, "Filter:")
	assertViewContains(t, m.View().Content, "GitHub", "Cloudflare")
}

func TestEscapeClosesThePickerOnceNoFilterIsKept(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag")
	m, _ = m.Update(keyEnter)
	m, _ = m.Update(keyEscape)

	_, cmd := m.Update(keyEscape)

	if _, closed := cmd().(closeWatchPickerMsg); !closed {
		t.Fatal("the second esc should close the picker")
	}
}

func TestOnlyTheFocusedPanesKeptFilterIsClearedByEscape(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "cloud")
	m, _ = m.Update(keyEnter)
	m, _ = m.Update(keyTab)

	_, cmd := m.Update(keyEscape)

	if _, closed := cmd().(closeWatchPickerMsg); !closed {
		t.Fatal("esc on the components pane must not touch the providers filter")
	}
}

func TestTheKeyBarSaysWhatEscapeDoes(t *testing.T) {
	m := pickerOnComponents(fakePickerBackend{fakeProviderCatalog: testCatalog()})
	assertViewContains(t, m.View().Content, "esc cancel")

	m, _ = m.Update(keySlash)
	m = typeIntoPicker(m, "pag")
	m, _ = m.Update(keyEnter)
	assertViewContains(t, m.View().Content, "esc clear filter")
	assertViewLacks(t, m.View().Content, "esc cancel")
}
