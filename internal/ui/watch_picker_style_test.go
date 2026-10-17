package ui

import (
	"regexp"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

// rawRow is the component row of a view, colour codes included, that shows the text. It reads the
// text without the codes, because strikethrough is applied to each character on its own.
func rawRow(t *testing.T, view, text string) string {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(plain(t, line), "] "+text) {
			return line
		}
	}
	t.Fatalf("no row shows %q:\n%s", text, plain(t, view))
	return ""
}

var (
	reverseVideo  = regexp.MustCompile(`\x1b\[(?:\d+;)*7(?:;\d+)*m`)
	strikethrough = regexp.MustCompile(`\x1b\[(?:\d+;)*9(?:;\d+)*m`)
	struckSpace   = regexp.MustCompile(`\x1b\[(?:\d+;)*9(?:;\d+)*m \x1b\[m`)
)

func TestEachWatchStateHasItsOwnNameStyle(t *testing.T) {
	watched, added, removed, unwatched := nameStyleFor(stateWatched), nameStyleFor(stateAdded), nameStyleFor(stateRemoved), nameStyleFor(stateUnwatched)

	if watched.GetForeground() != colorGreen || watched.GetBold() || watched.GetStrikethrough() {
		t.Errorf("watched should be plain green: %+v", watched)
	}
	if added.GetForeground() != colorBlue || !added.GetBold() {
		t.Errorf("added should be bold blue: %+v", added)
	}
	if removed.GetForeground() != colorRed || !removed.GetStrikethrough() {
		t.Errorf("removed should be red and struck through: %+v", removed)
	}
	if unwatched.GetBold() || unwatched.GetStrikethrough() || unwatched.GetReverse() {
		t.Errorf("a component that is not watched stays plain: %+v", unwatched)
	}
	if !cursorRowStyle.GetReverse() || !cursorRowStyle.GetBold() {
		t.Error("the cursor row is a bold reverse-video bar")
	}
}

func chosenRowsPicker() WatchPickerModel {
	// API Requests watched, Git Operations added, Webhooks removed, Pages not watched.
	m := groupedPicker(component("github", "api"), component("github", "hooks"))
	m, _ = m.Update(keyJ)
	m, _ = m.Update(keySpace)
	m, _ = m.Update(keyJ)
	m, _ = m.Update(keySpace)
	m, _ = m.Update(keyJ)
	return m // the cursor is on Pages
}

func TestChosenRowsAreStyledAcrossTheNameAndOthersAreNot(t *testing.T) {
	view := chosenRowsPicker().View().Content

	if !strings.Contains(rawRow(t, view, "API Requests"), "\x1b[32mAPI Requests") {
		t.Error("a watched component should be green across its name")
	}
	if !strings.Contains(rawRow(t, view, "Git Operations"), "\x1b[1;34mGit Operations") {
		t.Error("an added component should be bold blue across its name")
	}
	if !strikethrough.MatchString(rawRow(t, view, "Webhooks")) {
		t.Error("a component to be removed should be struck through")
	}
	assertViewContains(t, view, "[x] API Requests", "[+] Git Operations", "[-] Webhooks")
}

func TestTheStrikethroughStopsAtTheEndOfTheName(t *testing.T) {
	row := rawRow(t, chosenRowsPicker().View().Content, "Webhooks") // shorter than Git Operations, so it is padded

	if struckSpace.MatchString(row) {
		t.Fatalf("the strikethrough runs through the padding: %q", row)
	}
}

func TestTheRowUnderTheCursorIsABarAcrossTheWholeRow(t *testing.T) {
	m := chosenRowsPicker()

	row := rawRow(t, m.View().Content, "Pages")

	if !strings.Contains(row, "\x1b[1;7m▸ [ ] Pages") || !strings.Contains(row, "Operational\x1b[m") {
		t.Fatalf("the cursor row should be one reverse-video run from the marker to the status: %q", row)
	}
	if got := len(reverseVideo.FindAllString(m.View().Content, -1)); got != 1 {
		t.Fatalf("%d reverse-video runs, want exactly the cursor row's", got)
	}
}

func TestTheBarShowsTheStateOfTheRowInPlainText(t *testing.T) {
	// API Requests watched, Git Operations added, Webhooks removed, Pages not watched.
	cases := []struct {
		presses int
		row     string
	}{{0, "▸ [x] API Requests"}, {1, "▸ [+] Git Operations"}, {2, "▸ [-] Webhooks"}, {3, "▸ [ ] Pages"}}
	m := chosenRowsPicker()
	m = pressTimes(m, keyK, 3) // back on API Requests

	for _, tc := range cases {
		view := pressTimes(m, keyJ, tc.presses).View().Content
		if !strings.Contains(plain(t, view), tc.row) || len(reverseVideo.FindAllString(view, -1)) != 1 {
			t.Errorf("want one bar on %q:\n%q", tc.row, view)
		}
	}
}

func TestThereIsNoCursorBarWhileTheProvidersHaveTheFocus(t *testing.T) {
	m := openedPicker(fakePickerBackend{fakeProviderCatalog: groupedCatalog()}, component("github", "api"))

	view := m.View().Content

	if reverseVideo.MatchString(view) {
		t.Fatalf("a component row is highlighted although the providers have the focus:\n%q", view)
	}
	assertViewContains(t, view, "[x] API Requests")
}

func TestTheCursorBarMovesWithTheCursor(t *testing.T) {
	m := groupedPicker()
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})

	view := m.View().Content

	if !strings.Contains(rawRow(t, view, "Git Operations"), "\x1b[1;7m") || strings.Contains(rawRow(t, view, "API Requests"), "\x1b[1;7m") {
		t.Fatalf("the bar should be on Git Operations only:\n%q", view)
	}
}
