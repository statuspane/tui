package ui

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

// AppRenamer is the one thing the rename dialog needs from the API client; tests pass a fake.
type AppRenamer interface {
	RenameApp(ctx context.Context, appSlug, newName string) (api.AppDetail, error)
}

type openRenameAppDialogMsg struct {
	slug string
	name string
}

// appRenamedMsg tells the root model the app has its new name; it closes the dialog and, if that
// app is open, gives its screen the app as the API now has it. The slug may have changed too.
type appRenamedMsg struct {
	previousSlug string
	appDetail    api.AppDetail
}

type renameAppFailedMsg struct{ err error }

type renameAppDialog struct {
	appRenamer   AppRenamer
	slug         string
	currentName  string
	input        textinput.Model
	isSubmitting bool
	err          error
}

func newRenameAppDialog(appRenamer AppRenamer, msg openRenameAppDialogMsg) (renameAppDialog, tea.Cmd) {
	input, focusCmd := newNameInput(msg.name)
	return renameAppDialog{appRenamer: appRenamer, slug: msg.slug, currentName: msg.name, input: input}, focusCmd
}

func (d renameAppDialog) update(msg tea.Msg) (dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return d.handleKey(msg)
	case renameAppFailedMsg:
		d.isSubmitting, d.err = false, msg.err
		return d, nil
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return d, cmd
}

func (d renameAppDialog) handleKey(keyPress tea.KeyPressMsg) (dialog, tea.Cmd) {
	switch keyPress.String() {
	case "ctrl+c":
		return d, tea.Quit
	case "esc":
		if d.isSubmitting {
			return d, nil
		}
		return d, closeDialog
	case "enter":
		if d.isSubmitting {
			return d, nil
		}
		newName := strings.TrimSpace(d.input.Value())
		switch newName {
		case "":
			return d, nil
		case d.currentName:
			return d, closeDialog
		}
		d.isSubmitting, d.err = true, nil
		return d, renameApp(d.appRenamer, d.slug, newName)
	}
	if d.isSubmitting {
		return d, nil
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(keyPress)
	return d, cmd
}

func (d renameAppDialog) view() string {
	lines := []string{mutedStyle.Render("Name"), nameBox(d.input)}
	lines = append(lines, statusLines(d.isSubmitting, "Renaming…", d.err, func(err error) string { return fieldErrorMessage(err, nameField) })...)
	lines = append(lines, hintLines(renderHint(keyHint{"⏎", "rename"}), renderHint(keyHint{"esc", "cancel"}))...)
	return dialogBox(accentPanelLook, "Rename app", lines)
}

func renameApp(appRenamer AppRenamer, appSlug, newName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		appDetail, err := appRenamer.RenameApp(ctx, appSlug, newName)
		if err != nil {
			return renameAppFailedMsg{err: err}
		}
		return appRenamedMsg{previousSlug: appSlug, appDetail: appDetail}
	}
}
