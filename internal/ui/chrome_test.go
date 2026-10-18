package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

func TestPanelSetsTheTitleIntoTheTopEdge(t *testing.T) {
	view := panel("Apps", "hello", 30, 5)

	top := strings.Split(plain(t, view), "\n")[0]
	if !strings.HasPrefix(top, "╭─ Apps ─") || !strings.HasSuffix(top, "╮") {
		t.Fatalf("top edge = %q", top)
	}
	assertViewContains(t, view, "│  hello")
}

func TestPanelIsExactlyTheSizeAsked(t *testing.T) {
	view := panel("Apps", "short\na much longer line than the panel has room for\nthird", 30, 8)

	assertEveryLineHasWidth(t, view, 30)
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
	assertEveryLineHasWidth(t, view, 60)
	if got := len(strings.Split(view, "\n")); got != footerHeight {
		t.Fatalf("height = %d, want %d", got, footerHeight)
	}
}

func TestFooterBarDropsHintsFromTheRightWhenTheyDoNotFit(t *testing.T) {
	hints := []keyHint{{"↑↓", "move"}, {"enter", "open"}, {"n", "new app"}, {"d", "delete"}}
	view := footerBar(hints, "q quit", 36)

	assertViewContains(t, view, "↑↓ move", "q quit")
	assertViewLacks(t, view, "delete")
	assertEveryLineHasWidth(t, view, 36)
}

func TestFooterBarDropsTrailingWhenNothingElseFits(t *testing.T) {
	view := footerBar([]keyHint{{"enter", "open"}}, strings.Repeat("t", 40), 24)

	assertViewLacks(t, view, "ttt")
	assertEveryLineHasWidth(t, view, 24)
}

func TestBodyHeightLeavesRoomForMarginHeaderAndFooter(t *testing.T) {
	// 30 lines less two margin lines, the header, two blank lines and the three-line footer.
	if got := bodyHeight(30); got != 22 {
		t.Fatalf("bodyHeight = %d, want 22", got)
	}
}

func TestBodyHeightNeverDropsBelowTheMinimumPanel(t *testing.T) {
	if got := bodyHeight(2); got != minPanelHeight {
		t.Fatalf("bodyHeight = %d, want %d", got, minPanelHeight)
	}
}

func TestPanelWithLookColoursTheEdgeAndTheTitle(t *testing.T) {
	look := panelLook{border: errorStyle, title: titleStyle}

	view := panelWithLook(look, "Apps", "x", 30, 4)

	if !strings.Contains(view, escapeBefore(errorStyle.Render("x"))+"╭") {
		t.Errorf("the edge does not use the border style:\n%q", view)
	}
	if !strings.Contains(view, escapeBefore(titleStyle.Render("x"))+"Apps") {
		t.Errorf("the title does not use the title style:\n%q", view)
	}
}

func TestPanelKeepsTheDefaultLook(t *testing.T) {
	if panel("Apps", "x", 30, 4) != panelWithLook(defaultPanelLook, "Apps", "x", 30, 4) {
		t.Fatal("panel should draw with the default look")
	}
}

func TestWithFallbackSizeOnlyReplacesWhatIsNotKnown(t *testing.T) {
	width, height := withFallbackSize(0, 50)

	if width != fallbackWidth || height != 50 {
		t.Fatalf("size = %dx%d", width, height)
	}
}
