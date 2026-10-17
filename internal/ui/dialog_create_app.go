package ui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

// AppCreator is the one thing the create dialog needs from the API client; tests pass a fake.
type AppCreator interface {
	CreateApp(ctx context.Context, name string, refs []api.ComponentRef) (api.AppDetail, error)
}

type appCreateFailedMsg struct{ err error }

type createAppDialog struct {
	appCreator   AppCreator
	input        textinput.Model
	isSubmitting bool
	err          error
}

func newCreateAppDialog(appCreator AppCreator) (createAppDialog, tea.Cmd) {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = "Storefront"
	input.CharLimit = maxAppNameLength
	input.SetWidth(inputWidth)
	focus := input.Focus()
	return createAppDialog{appCreator: appCreator, input: input}, focus
}

func (d createAppDialog) update(msg tea.Msg) (dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return d.handleKey(msg)
	case appCreateFailedMsg:
		d.isSubmitting, d.err = false, msg.err
		return d, nil
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return d, cmd
}

func (d createAppDialog) handleKey(keyPress tea.KeyPressMsg) (dialog, tea.Cmd) {
	switch keyPress.String() {
	case "ctrl+c":
		return d, tea.Quit
	case "esc":
		if d.isSubmitting {
			return d, nil
		}
		return d, closeDialog
	case "enter":
		name := strings.TrimSpace(d.input.Value())
		if name == "" || d.isSubmitting {
			return d, nil
		}
		d.isSubmitting, d.err = true, nil
		return d, createApp(d.appCreator, name)
	}
	if d.isSubmitting {
		return d, nil
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(keyPress)
	return d, cmd
}

func (d createAppDialog) view() string {
	length := len([]rune(d.input.Value()))
	lines := []string{
		titleStyle.Render("New app"),
		"",
		mutedStyle.Render("Name"),
		d.input.View(),
		mutedStyle.Render(lengthText(length)),
	}
	switch {
	case d.isSubmitting:
		lines = append(lines, "", "Creating…")
	case d.err != nil:
		lines = append(lines, "", errorStyle.Render(fieldErrorMessage(d.err, nameField)))
	}
	lines = append(lines, "", mutedStyle.Render("⏎ create · esc cancel"))
	return renderDialog(colorBlue, lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func createApp(appCreator AppCreator, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		appDetail, err := appCreator.CreateApp(ctx, name, nil)
		if err != nil {
			return appCreateFailedMsg{err: err}
		}
		return appCreatedMsg{appDetail: appDetail}
	}
}
