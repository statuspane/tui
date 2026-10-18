package ui

import (
	"context"
	"fmt"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

// AppDeleter is the one thing the delete dialog needs from the API client; tests pass a fake.
type AppDeleter interface {
	DeleteApp(ctx context.Context, appSlug string) error
}

type openDeleteAppDialogMsg struct {
	slug           string
	name           string
	componentCount int
}

// appDeletedMsg tells the root model the app is gone; it closes the dialog and, if that app is
// open, its detail.
type appDeletedMsg struct{ slug string }

type deleteAppFailedMsg struct{ err error }

// deleteAppDialog asks for the app's name to be typed, because deleting can't be undone.
type deleteAppDialog struct {
	appDeleter     AppDeleter
	slug           string
	name           string
	componentCount int
	input          textinput.Model
	isSubmitting   bool
	err            error
}

func newDeleteAppDialog(appDeleter AppDeleter, msg openDeleteAppDialogMsg) (deleteAppDialog, tea.Cmd) {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = msg.name
	input.SetWidth(inputWidth)
	focusCmd := input.Focus()
	return deleteAppDialog{
		appDeleter:     appDeleter,
		slug:           msg.slug,
		name:           msg.name,
		componentCount: msg.componentCount,
		input:          input,
	}, focusCmd
}

func (d deleteAppDialog) update(msg tea.Msg) (dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return d.handleKey(msg)
	case deleteAppFailedMsg:
		d.isSubmitting, d.err = false, msg.err
		return d, nil
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return d, cmd
}

func (d deleteAppDialog) handleKey(keyPress tea.KeyPressMsg) (dialog, tea.Cmd) {
	switch keyPress.String() {
	case "ctrl+c":
		return d, tea.Quit
	case "esc":
		if d.isSubmitting {
			return d, nil
		}
		return d, closeDialog
	case "enter":
		if !d.nameMatches() || d.isSubmitting {
			return d, nil
		}
		d.isSubmitting, d.err = true, nil
		return d, deleteApp(d.appDeleter, d.slug)
	}
	if d.isSubmitting {
		return d, nil
	}
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(keyPress)
	return d, cmd
}

func (d deleteAppDialog) nameMatches() bool {
	return d.input.Value() == d.name
}

func (d deleteAppDialog) view() string {
	lines := []string{
		wrappedText("It stops watching " + componentsText(d.componentCount) + ". This can't be undone."),
		"",
		mutedStyle.Render("Type the app's name to confirm:"),
		inputBox(d.input.View(), "", defaultPanelLook, dialogContentWidth),
	}
	switch {
	case d.isSubmitting:
		lines = append(lines, "", "Deleting…")
	case d.err != nil:
		lines = append(lines, "", errorStyle.Render(wrappedText(errorMessage(d.err))))
	}
	lines = append(lines, "", dialogHints(d.deleteHint(), renderHint(keyHint{"esc", "cancel"})))
	return dialogBox(dangerDialogLook, "Delete "+d.name+"?", lines...)
}

// deleteHint reads as a live key once the typed name matches, and as a disabled one until then.
func (d deleteAppDialog) deleteHint() string {
	if d.nameMatches() {
		return errorStyle.Bold(true).Render("⏎") + " " + errorStyle.Render("delete")
	}
	return mutedStyle.Render("⏎ delete (disabled until the name matches)")
}

// deleteApp treats a 404 as success: the app is already gone, which is what was asked for.
func deleteApp(appDeleter AppDeleter, appSlug string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		if err := appDeleter.DeleteApp(ctx, appSlug); err != nil && !isNotFound(err) {
			return deleteAppFailedMsg{err: err}
		}
		return appDeletedMsg{slug: appSlug}
	}
}

func componentsText(count int) string {
	if count == 1 {
		return "1 component"
	}
	return fmt.Sprintf("%d components", count)
}
