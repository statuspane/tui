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

func TestCenteredOverPutsTheBoxInTheMiddleAndKeepsTheBaseAround(t *testing.T) {
	baseLine := "● " + strings.Repeat("abcdefghij", 4) + " ●"
	base := strings.TrimSuffix(strings.Repeat(baseLine+"\n", 9), "\n")
	box := panel("Box", "hello", 20, 5)

	lines := strings.Split(plain(t, centeredOver(base, box, lipgloss.Width(baseLine), 9)), "\n")

	if !strings.Contains(lines[2], "╭─ Box") || !strings.Contains(lines[6], "╰") {
		t.Fatalf("the box should be centred vertically, on lines 2 to 6:\n%s", strings.Join(lines, "\n"))
	}
	corner := strings.Index(lines[2], "╭")
	if want := (lipgloss.Width(baseLine) - 20) / 2; lipgloss.Width(lines[2][:corner]) != want {
		t.Fatalf("the box should start at column %d:\n%s", want, lines[2])
	}
	if lines[0] != baseLine || !strings.HasPrefix(lines[3], "● ") || !strings.HasSuffix(lines[3], " ●") {
		t.Fatalf("the base should stay visible around the box:\n%s", strings.Join(lines, "\n"))
	}
}

func TestCenteredOverCoversTheBaseUnderTheBox(t *testing.T) {
	base := strings.TrimSuffix(strings.Repeat(strings.Repeat("x", 40)+"\n", 9), "\n")

	view := plain(t, centeredOver(base, panel("Title", "hello", 20, 5), 40, 9))

	if strings.Count(view, "x") != 40*9-20*5 {
		t.Fatalf("the box should replace exactly its own cells:\n%s", view)
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
