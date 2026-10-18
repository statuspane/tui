package ui

import (
	"errors"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

func linesOf(t *testing.T, view string) []string {
	t.Helper()
	return strings.Split(plain(t, view), "\n")
}

func TestADialogIsInTheMiddleOfTheWindowWithTheScreenAroundIt(t *testing.T) {
	m := loadedRoot(api.App{Name: "First", Slug: "first"})
	m, _ = sendRoot(m, openCreateAppDialogMsg{})

	lines := linesOf(t, m.View().Content)

	boxHeight := len(strings.Split(m.dialog.view(), "\n"))
	top := (testRootHeight - boxHeight) / 2
	column := lipgloss.Width(lines[top][:strings.Index(lines[top], "╭─ New app")])
	if column != (testRootWidth-dialogWidth)/2 {
		t.Fatalf("the box should start on line %d at column %d:\n%s", top, (testRootWidth-dialogWidth)/2, strings.Join(lines, "\n"))
	}
	if len(lines) != testRootHeight {
		t.Fatalf("the view is %d lines tall, want %d", len(lines), testRootHeight)
	}
}

func TestTheDeleteDialogLeavesTheAppScreenVisibleAroundIt(t *testing.T) {
	m := rootOnLoadedDetail()

	m, _ = sendRoot(m, openDeleteAppDialogMsg{slug: "web-app", name: "Web App", componentCount: 2})

	assertViewContains(t, m.View().Content, "Delete Web App?", "◉ STATUSPANE › Web App", "Watched components")
}

func TestTheCharacterCountIsInsideTheNameBox(t *testing.T) {
	d := typeText(newTestCreateDialog(fakeAppCreator{}), "abc")

	var nameLine string
	for _, line := range linesOf(t, d.view()) {
		if strings.Contains(line, "/ 100") {
			nameLine = line
		}
	}
	if !strings.HasPrefix(nameLine, "│") || !strings.Contains(nameLine, "abc") || !strings.Contains(nameLine, "3 / 100") {
		t.Fatalf("the name and its count should share a boxed line, got %q", nameLine)
	}
}

func TestTheDeleteHintIsDisabledUntilTheNameMatches(t *testing.T) {
	d := newTestDeleteDialog(fakeAppDeleter{})
	assertViewContains(t, d.view(), "⏎ delete (disabled until the name matches)")

	d = typeText(d, "Web App")

	assertViewContains(t, d.view(), "⏎ delete")
	assertViewLacks(t, d.view(), "disabled")
}

func TestTheDeleteDialogHasTheDangerLook(t *testing.T) {
	view := newTestDeleteDialog(fakeAppDeleter{}).view()

	if !strings.Contains(view, escapeBefore(errorStyle.Render("x"))+"╭") {
		t.Fatalf("the edge should be red:\n%q", view)
	}
}

func TestALongErrorWrapsInsideTheDialog(t *testing.T) {
	d := newTestCreateDialog(fakeAppCreator{})
	d, _ = d.update(createAppFailedMsg{err: errors.New(strings.Repeat("something went wrong ", 8))})

	view := d.view()

	assertEveryLineHasWidth(t, view, dialogWidth)
	if got := strings.Count(plain(t, view), "wrong"); got != 8 {
		t.Fatalf("the whole message should be there, found its last word %d times:\n%s", got, plain(t, view))
	}
}

func TestAVeryLongAppNameDoesNotBreakTheDeleteDialog(t *testing.T) {
	d, _ := newDeleteAppDialog(fakeAppDeleter{}, openDeleteAppDialogMsg{slug: "x", name: strings.Repeat("N", maxAppNameLength), componentCount: 1})

	assertEveryLineHasWidth(t, d.view(), dialogWidth)
}

func TestTheRenameDialogLooksLikeTheNewAppDialog(t *testing.T) {
	view := newTestRenameDialog(fakeAppRenamer{}).view()

	assertViewContains(t, view, "╭─ Rename app", "Web App", "/ 100", "⏎ rename", "esc cancel")
	if !strings.Contains(view, escapeBefore(accentDialogLook.border.Render("x"))+"╭") {
		t.Fatalf("the edge should be the accent colour:\n%q", view)
	}
}
