package ui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

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
