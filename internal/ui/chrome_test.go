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

func TestWithFallbackSizeOnlyReplacesWhatIsNotKnown(t *testing.T) {
	for _, test := range []struct{ width, height, wantWidth, wantHeight int }{
		{0, 50, fallbackWidth, 50},
		{70, 0, 70, fallbackHeight},
		{70, 50, 70, 50},
	} {
		if width, height := withFallbackSize(test.width, test.height); width != test.wantWidth || height != test.wantHeight {
			t.Errorf("withFallbackSize(%d, %d) = %dx%d, want %dx%d", test.width, test.height, width, height, test.wantWidth, test.wantHeight)
		}
	}
}

func TestAlertBarIsAFooterShapedBoxWithTheMessageInIt(t *testing.T) {
	view := alertBar("Discard 2 changes?", 60)

	assertViewContains(t, view, "Discard 2 changes?")
	assertEveryLineHasWidth(t, view, 60)
	assertHeightIs(t, view, footerHeight)
	if !strings.Contains(view, escapeBefore(errorStyle.Render("x"))+"╭") {
		t.Errorf("the edge should be red:\n%q", view)
	}
}

// A base and an overlay for the centeredOver tests: the overlay's own text has no 'x' in it.
const (
	overlayBaseWidth     = 40
	overlayBaseHeight    = 9
	overlayWidth         = 20
	overlayHeight        = 5
	overlayBaseCellsText = "x"
)

func overlayBase() string {
	line := strings.Repeat(overlayBaseCellsText, overlayBaseWidth)
	return strings.TrimSuffix(strings.Repeat(line+"\n", overlayBaseHeight), "\n")
}

func overlayOnBase(t *testing.T) []string {
	t.Helper()
	view := centeredOver(overlayBase(), panel("Title", "hello", overlayWidth, overlayHeight), overlayBaseWidth, overlayBaseHeight)
	return plainLines(t, view)
}

func TestCenteredOverPutsTheOverlayInTheMiddleVertically(t *testing.T) {
	lines := overlayOnBase(t)

	top := (overlayBaseHeight - overlayHeight) / 2
	if !strings.Contains(lines[top], "╭─ Title") || !strings.Contains(lines[top+overlayHeight-1], "╰") {
		t.Fatalf("the overlay should be on lines %d to %d:\n%s", top, top+overlayHeight-1, strings.Join(lines, "\n"))
	}
}

func TestCenteredOverPutsTheOverlayInTheMiddleHorizontally(t *testing.T) {
	lines := overlayOnBase(t)

	line := lines[(overlayBaseHeight-overlayHeight)/2]
	if got, want := columnOf(t, line, "╭"), (overlayBaseWidth-overlayWidth)/2; got != want {
		t.Fatalf("the overlay starts at column %d, want %d:\n%s", got, want, line)
	}
}

func TestCenteredOverKeepsTheBaseAroundTheOverlay(t *testing.T) {
	lines := overlayOnBase(t)

	baseLine := strings.Repeat(overlayBaseCellsText, overlayBaseWidth)
	if lines[0] != baseLine || lines[overlayBaseHeight-1] != baseLine {
		t.Fatalf("the lines above and below the overlay should be the base's:\n%s", strings.Join(lines, "\n"))
	}
	margin := strings.Repeat(overlayBaseCellsText, (overlayBaseWidth-overlayWidth)/2)
	for _, line := range lines[(overlayBaseHeight-overlayHeight)/2:][:overlayHeight] {
		if !strings.HasPrefix(line, margin) || !strings.HasSuffix(line, margin) {
			t.Fatalf("the base should show on both sides of the overlay: %q", line)
		}
	}
}

func TestCenteredOverCoversTheBaseUnderTheOverlay(t *testing.T) {
	view := strings.Join(overlayOnBase(t), "\n")

	covered := overlayWidth * overlayHeight
	if got := strings.Count(view, overlayBaseCellsText); got != overlayBaseWidth*overlayBaseHeight-covered {
		t.Fatalf("the overlay should replace exactly its own cells, %d of the base are left:\n%s", got, view)
	}
}

func TestCenteredOverPutsAnOverlayLargerThanTheWindowAtTheTopLeft(t *testing.T) {
	view := centeredOver("ab\ncd", panel("Title", "hello", overlayWidth, overlayHeight), 2, 2)

	if !strings.HasPrefix(plain(t, view), "╭") {
		t.Fatalf("view = %q", view)
	}
}

func TestInputBoxHasItsTextAtBothEnds(t *testing.T) {
	view := plain(t, inputBox("Storefront", "10 / 100", defaultPanelLook, 40))

	assertEveryLineHasWidth(t, view, 40)
	assertHeightIs(t, view, oneLineBoxHeight)
	middle := strings.Split(view, "\n")[1]
	if !strings.Contains(middle, "│  Storefront") || !strings.Contains(middle, "10 / 100  │") {
		t.Fatalf("middle = %q", middle)
	}
}

func TestPanelCutsALongTitleInsteadOfBreakingTheEdge(t *testing.T) {
	const panelWidth = 24
	view := panel(strings.Repeat("T", 2*panelWidth), "x", panelWidth, 3)

	assertEveryLineHasWidth(t, view, panelWidth)
	assertViewContains(t, view, "╭─ TTTT")
}
