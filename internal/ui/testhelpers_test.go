package ui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

func component(provider, id string) api.Component {
	return api.Component{ComponentReference: api.ComponentReference{Provider: provider, ID: id, Name: id}}
}

func namedComponent(provider, id, name string, group *string) api.Component {
	return api.Component{
		ComponentReference: api.ComponentReference{Provider: provider, ID: id, Name: name},
		GroupName:          group,
		Status:             stringPtr("operational"),
	}
}

func apiError(status int, code string, details ...api.FieldError) *api.APIError {
	return &api.APIError{Status: status, Code: code, Details: details}
}

// commandWait is how long collectMessages gives a command to answer. A tea.Tick command sleeps
// for its whole interval, so it counts as "no immediate message" instead of hanging the test.
const commandWait = 200 * time.Millisecond

// collectMessages runs a command and flattens batches into the messages that arrive at once.
func collectMessages(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	answer := make(chan tea.Msg, 1)
	go func() { answer <- cmd() }()

	var msg tea.Msg
	select {
	case msg = <-answer:
	case <-time.After(commandWait):
		return nil
	}
	batch, isBatch := msg.(tea.BatchMsg)
	if !isBatch {
		return []tea.Msg{msg}
	}
	var messages []tea.Msg
	for _, inner := range batch {
		messages = append(messages, collectMessages(inner)...)
	}
	return messages
}

func containsAppsChanged(messages []tea.Msg) bool {
	for _, msg := range messages {
		if _, ok := msg.(appsChangedMsg); ok {
			return true
		}
	}
	return false
}

var colorCodes = regexp.MustCompile("\x1b\\[[0-9;]*m")

// plain is a view without its colour codes, so a test can read it the way a person does.
func plain(t *testing.T, view string) string {
	t.Helper()
	return colorCodes.ReplaceAllString(view, "")
}

func assertViewContains(t *testing.T, view string, wants ...string) {
	t.Helper()
	text := plain(t, view)
	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Errorf("view missing %q:\n%s", want, text)
		}
	}
}

func assertViewLacks(t *testing.T, view string, unwanted ...string) {
	t.Helper()
	text := plain(t, view)
	for _, bad := range unwanted {
		if strings.Contains(text, bad) {
			t.Errorf("view should not contain %q:\n%s", bad, text)
		}
	}
}

func assertEveryLineHasWidth(t *testing.T, view string, width int) {
	t.Helper()
	for _, line := range strings.Split(view, "\n") {
		if got := lipgloss.Width(line); got != width {
			t.Errorf("line %q is %d wide, want %d", plain(t, line), got, width)
		}
	}
}

func assertHeightIs(t *testing.T, view string, height int) {
	t.Helper()
	if got := len(strings.Split(view, "\n")); got != height {
		t.Errorf("view is %d lines tall, want %d", got, height)
	}
}

// escapeBefore is the colour codes a style puts before its text, so a test can look for a style
// without writing the codes out.
func escapeBefore(styled string) string {
	return styled[:strings.Index(styled, "x")]
}

// plainLines is a view without its colour codes, one string per line.
func plainLines(t *testing.T, view string) []string {
	t.Helper()
	return strings.Split(plain(t, view), "\n")
}

// lineContaining is the first line of the view, without colour codes, that has text in it.
func lineContaining(t *testing.T, view, text string) string {
	t.Helper()
	for _, line := range plainLines(t, view) {
		if strings.Contains(line, text) {
			return line
		}
	}
	t.Fatalf("no line contains %q:\n%s", text, plain(t, view))
	return ""
}

// columnOf is the cell, not the byte, at which text starts in a line.
func columnOf(t *testing.T, line, text string) int {
	t.Helper()
	index := strings.Index(line, text)
	if index < 0 {
		t.Fatalf("%q is not in %q", text, line)
	}
	return lipgloss.Width(line[:index])
}

// assertEdgeIs fails unless the view has a panel corner drawn in the border of look.
func assertEdgeIs(t *testing.T, view string, look panelLook) {
	t.Helper()
	if !strings.Contains(view, escapeBefore(look.border.Render("x"))+"╭") {
		t.Errorf("no panel corner in the colour of that look:\n%q", view)
	}
}

// The window the root tests open in.
const (
	testRootWidth  = 100
	testRootHeight = 30
)

// drainCommands feeds a model the answers to its commands. The spinner's ticks are left out (each
// starts the next), and so are the timers, which collectMessages does not wait for.
func drainCommands(m AppsModel, cmd tea.Cmd) AppsModel {
	for _, msg := range collectMessages(cmd) {
		if _, isSpinnerTick := msg.(spinner.TickMsg); isSpinnerTick {
			continue
		}
		var next tea.Cmd
		m, next = send(m, msg)
		m = drainCommands(m, next)
	}
	return m
}

// modelWithDetail is the screen with one app, "web-app", whose detail has loaded, with thirty days
// of history behind it.
func modelWithDetail(t *testing.T, appDetail api.AppDetail, width, height int) AppsModel {
	t.Helper()
	trackingSince := time.Date(2026, time.October, 3, 0, 0, 0, 0, time.UTC)
	return modelWithHistory(t, appDetail, thirtyOperationalDays(&trackingSince), nil, width, height)
}

// modelWithHistory is the same with the history it is given, or the error its request fails with.
func modelWithHistory(t *testing.T, appDetail api.AppDetail, history api.AppHistory, historyErr error, width, height int) AppsModel {
	t.Helper()
	backend := fakeAppsBackend{
		fakeAppLister: fakeAppLister{apps: []api.App{{Name: "Web App", Slug: "web-app", Status: stringPtr("degraded_performance"), WatchedComponentsCount: 2}}},
		fakeAppGetter: fakeAppGetter{appDetail: appDetail, history: history, historyErr: historyErr},
	}
	m := NewAppsModel(backend)
	m.now = func() time.Time { return detailNow }
	m, _ = send(m, tea.WindowSizeMsg{Width: width, Height: height})
	return drainCommands(m, m.Init())
}

func thirtyOperationalDays(trackingSince *time.Time) api.AppHistory {
	history := api.AppHistory{TrackingSince: trackingSince}
	firstDay := detailNow.AddDate(0, 0, 1-historyDayCount)
	for offset := range historyDayCount {
		history.Days = append(history.Days, api.HistoryDay{Date: firstDay.AddDate(0, 0, offset).Format(time.DateOnly), Status: stringPtr("operational")})
	}
	return history
}
