package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func assertEveryLineIs(t *testing.T, view string, width int) {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got != width {
			t.Errorf("line %q is %d wide, want %d", plain(t, line), got, width)
		}
	}
}

func TestPanelSetsTheTitleIntoTheTopEdge(t *testing.T) {
	view := panel("Apps", "hello", 30, 5)

	top := strings.Split(plain(t, view), "\n")[0]
	if !strings.HasPrefix(top, "╭─ Apps ─") || !strings.HasSuffix(top, "╮") {
		t.Fatalf("top edge = %q", top)
	}
	assertViewContains(t, view, "│ hello")
}

func TestPanelIsExactlyTheSizeAsked(t *testing.T) {
	view := panel("Apps", "short\na much longer line than the panel has room for\nthird", 30, 8)

	assertEveryLineIs(t, view, 30)
	if got := len(strings.Split(view, "\n")); got != 8 {
		t.Fatalf("height = %d, want 8", got)
	}
}

func TestPanelCutsLinesAndRowsThatDoNotFit(t *testing.T) {
	view := panel("Apps", "one\ntwo\nthree\nfour", 24, 4)

	assertViewContains(t, view, "one", "two")
	assertViewLacks(t, view, "three", "four")
}

func TestPanelWithoutATitleIsAPlainBox(t *testing.T) {
	top := strings.Split(plain(t, panel("", "x", 20, 3)), "\n")[0]

	if top != "╭──────────────────╮" {
		t.Fatalf("top edge = %q", top)
	}
}

func TestPanelCutsALongTitleInsteadOfBreakingTheEdge(t *testing.T) {
	view := panel(strings.Repeat("T", 50), "x", 24, 3)

	assertEveryLineIs(t, view, 24)
}

func TestHeaderBarPutsRightAtTheFarEnd(t *testing.T) {
	view := plain(t, headerBar("left", "right", 40))

	if lipgloss.Width(view) != 40 || !strings.HasPrefix(view, "left") || !strings.HasSuffix(view, "right") {
		t.Fatalf("header = %q", view)
	}
}

func TestHeaderBarDropsTheRightWhenItDoesNotFit(t *testing.T) {
	view := plain(t, headerBar("left side", strings.Repeat("r", 40), 30))

	if strings.Contains(view, "r") || lipgloss.Width(view) > 30 {
		t.Fatalf("header = %q", view)
	}
}

func TestFooterBarShowsHintsAndTrailing(t *testing.T) {
	hints := []keyHint{{"↑↓", "move"}, {"enter", "open"}}
	view := footerBar(hints, "q quit", 60)

	assertViewContains(t, view, "↑↓ move", "enter open", "q quit")
	assertEveryLineIs(t, view, 60)
	if got := len(strings.Split(view, "\n")); got != footerHeight {
		t.Fatalf("height = %d, want %d", got, footerHeight)
	}
}

func TestFooterBarDropsHintsFromTheRightWhenTheyDoNotFit(t *testing.T) {
	hints := []keyHint{{"↑↓", "move"}, {"enter", "open"}, {"n", "new app"}, {"d", "delete"}}
	view := footerBar(hints, "q quit", 36)

	assertViewContains(t, view, "↑↓ move", "q quit")
	assertViewLacks(t, view, "delete")
	assertEveryLineIs(t, view, 36)
}

func TestFooterBarDropsTrailingWhenNothingElseFits(t *testing.T) {
	view := footerBar([]keyHint{{"enter", "open"}}, strings.Repeat("t", 40), 24)

	assertViewLacks(t, view, "ttt")
	assertEveryLineIs(t, view, 24)
}

func TestBodyHeightLeavesRoomForHeaderAndFooter(t *testing.T) {
	if got := bodyHeight(30); got != 30-headerHeight-footerHeight {
		t.Fatalf("bodyHeight = %d", got)
	}
	if got := bodyHeight(2); got < minPanelHeight {
		t.Fatalf("bodyHeight = %d, want at least %d", got, minPanelHeight)
	}
}
