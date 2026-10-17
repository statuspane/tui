package ui

import (
	"context"
	"errors"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

// AppDeleter is the one thing the delete dialog needs from the API client; tests pass a fake.
type AppDeleter interface {
	DeleteApp(ctx context.Context, appSlug string) error
}

type appDeleteFailedMsg struct{ err error }

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

func newDeleteAppDialog(appDeleter AppDeleter, request openDeleteAppDialogMsg) (deleteAppDialog, tea.Cmd) {
	input := textinput.New()
	input.Prompt = ""
	input.Placeholder = request.name
	input.SetWidth(inputWidth)
	focus := input.Focus()
	return deleteAppDialog{
		appDeleter:     appDeleter,
		slug:           request.slug,
		name:           request.name,
		componentCount: request.componentCount,
		input:          input,
	}, focus
}

func (d deleteAppDialog) update(msg tea.Msg) (dialog, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return d.handleKey(msg)
	case appDeleteFailedMsg:
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
		errorStyle.Bold(true).Render("Delete " + d.name + "?"),
		"",
		"It stops watching " + componentsText(d.componentCount) + ". This can't be undone.",
		"",
		mutedStyle.Render("Type the app's name to confirm:"),
		d.input.View(),
	}
	switch {
	case d.isSubmitting:
		lines = append(lines, "", "Deleting…")
	case d.err != nil:
		lines = append(lines, "", errorStyle.Render(errorMessage(d.err)))
	}
	lines = append(lines, "", mutedStyle.Render("⏎ delete (once the name matches) · esc cancel"))
	return renderDialog(colorRed, lipgloss.JoinVertical(lipgloss.Left, lines...))
}

// deleteApp treats a 404 as success: the app is already gone, which is what was asked for.
func deleteApp(appDeleter AppDeleter, appSlug string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		if err := appDeleter.DeleteApp(ctx, appSlug); err != nil && !isNotFound(err) {
			return appDeleteFailedMsg{err: err}
		}
		return appDeletedMsg{slug: appSlug}
	}
}

func isNotFound(err error) bool {
	var apiErr *api.APIError
	return errors.As(err, &apiErr) && apiErr.Code == api.CodeNotFound
}
