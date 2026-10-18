package ui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/statuspane/tui/internal/api"
)

func TestADialogOpensInTheMiddleOfTheWindow(t *testing.T) {
	m := loadedRoot(api.App{Name: "First", Slug: "first"})

	m, _ = sendRoot(m, openCreateAppDialogMsg{})

	line := lineContaining(t, m.View().Content, "╭─ New app")
	if got, want := columnOf(t, line, "╭─ New app"), (testRootWidth-dialogWidth)/2; got != want {
		t.Fatalf("the dialog starts at column %d, want %d:\n%s", got, want, line)
	}
}

func TestADialogDoesNotChangeTheHeightOfTheView(t *testing.T) {
	m := loadedRoot(api.App{Name: "First", Slug: "first"})

	m, _ = sendRoot(m, openCreateAppDialogMsg{})

	assertHeightIs(t, m.View().Content, testRootHeight)
}

func TestTheDeleteDialogLeavesTheAppScreenVisibleAroundIt(t *testing.T) {
	m := rootOnLoadedDetail()

	m, _ = sendRoot(m, openDeleteAppDialogMsg{slug: "web-app", name: "Web App", componentCount: 2})

	assertViewContains(t, m.View().Content, "Delete Web App?", "◉ STATUSPANE › Web App", "Watched components")
}

func TestTheNameBoxShowsTheNameAndItsCountOnOneBoxedLine(t *testing.T) {
	d := typeText(newTestCreateDialog(fakeAppCreator{}), "abc")

	line := lineContaining(t, d.view(), fmt.Sprintf("3 / %d", maxAppNameLength))

	if !strings.Contains(line, "│  abc") || !strings.Contains(line, fmt.Sprintf("3 / %d  │", maxAppNameLength)) {
		t.Fatalf("the name and its count should share a boxed line, got %q", line)
	}
}

func TestTheDeleteHintIsDisabledUntilTheNameMatches(t *testing.T) {
	d := newTestDeleteDialog(fakeAppDeleter{})

	assertViewContains(t, d.view(), "⏎ delete (disabled until the name matches)")
}

func TestTheDeleteHintIsLiveOnceTheNameMatches(t *testing.T) {
	d := typeText(newTestDeleteDialog(fakeAppDeleter{}), "Web App")

	assertViewContains(t, d.view(), "⏎ delete")
	assertViewLacks(t, d.view(), "disabled")
}

func TestTheDeleteDialogHasTheDangerLook(t *testing.T) {
	assertEdgeIs(t, newTestDeleteDialog(fakeAppDeleter{}).view(), dangerPanelLook)
}

func TestTheRenameDialogHasTheAccentLook(t *testing.T) {
	assertEdgeIs(t, newTestRenameDialog(fakeAppRenamer{}).view(), accentPanelLook)
}

func TestTheRenameDialogShowsTheCurrentNameCountAndKeys(t *testing.T) {
	assertViewContains(t, newTestRenameDialog(fakeAppRenamer{}).view(), "╭─ Rename app", "Web App", "/ 100", "⏎ rename", "esc cancel")
}

// longErrorWords are distinct, so a wrapped message that loses one is noticed.
var longErrorWords = []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india", "juliet", "kilo", "lima"}

func TestALongErrorKeepsAllItsWordsInsideTheDialog(t *testing.T) {
	d := newTestCreateDialog(fakeAppCreator{})
	d, _ = d.update(createAppFailedMsg{err: errors.New(strings.Repeat(strings.Join(longErrorWords, " ")+" ", 2))})

	assertViewContains(t, d.view(), longErrorWords...)
}

func TestALongErrorDoesNotWidenTheDialog(t *testing.T) {
	d := newTestCreateDialog(fakeAppCreator{})
	d, _ = d.update(createAppFailedMsg{err: errors.New(strings.Repeat(strings.Join(longErrorWords, " ")+" ", 2))})

	assertEveryLineHasWidth(t, d.view(), dialogWidth)
}

func TestAVeryLongAppNameDoesNotBreakTheDeleteDialog(t *testing.T) {
	d, _ := newDeleteAppDialog(fakeAppDeleter{}, openDeleteAppDialogMsg{slug: "x", name: strings.Repeat("N", maxAppNameLength), componentCount: 1})

	assertEveryLineHasWidth(t, d.view(), dialogWidth)
}
