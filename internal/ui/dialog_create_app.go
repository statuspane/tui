package ui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

// AppCreator is the one thing the create dialog needs from the API client; tests pass a fake.
type AppCreator interface {
	CreateApp(ctx context.Context, name string, refs []api.ComponentRef) (api.AppDetail, error)
}

// openCreateAppDialogMsg asks the root model to show the dialog that creates an app.
type openCreateAppDialogMsg struct{}

// appCreatedMsg tells the root model the app exists; it closes the dialog and opens the app.
type appCreatedMsg struct{ appDetail api.AppDetail }

type createAppFailedMsg struct{ err error }

type createAppDialog struct {
	appCreator   AppCreator
	input        textinput.Model
	isSubmitting bool
	err          error
}

func newCreateAppDialog(appCreator AppCreator) (createAppDialog, tea.Cmd) {
	input, focusCmd := newNameInput("")
	input.Placeholder = "Storefront"
	return createAppDialog{appCreator: appCreator, input: input}, focusCmd
}

func (d createAppDialog) update(msg tea.Msg) (dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return d.handleKey(msg)
	case createAppFailedMsg:
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
	lines := []string{mutedStyle.Render("Name"), nameBox(d.input), mutedStyle.Render("After creating it you can choose the components to watch.")}
	switch {
	case d.isSubmitting:
		lines = append(lines, "", "Creating…")
	case d.err != nil:
		lines = append(lines, "", errorStyle.Render(wrappedText(fieldErrorMessage(d.err, nameField))))
	}
	lines = append(lines, "", dialogHints(renderHint(keyHint{"⏎", "create"}), renderHint(keyHint{"esc", "cancel"})))
	return dialogBox(accentDialogLook, "New app", lines...)
}

func createApp(appCreator AppCreator, name string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		appDetail, err := appCreator.CreateApp(ctx, name, nil)
		if err != nil {
			return createAppFailedMsg{err: err}
		}
		return appCreatedMsg{appDetail: appDetail}
	}
}
