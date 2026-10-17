package ui

import (
	"errors"
	"fmt"
	"image/color"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

const (
	maxAppNameLength = 100
	inputWidth       = 44
	nameField        = "name"
)

// dialog is a modal the root model shows instead of the current screen. It handles every key
// press while it is open, so typing into a field never triggers a screen's shortcuts.
type dialog interface {
	update(msg tea.Msg) (dialog, tea.Cmd)
	view() string
}

type openDeleteAppDialogMsg struct {
	slug           string
	name           string
	componentCount int
}

type closeDialogMsg struct{}

// appCreatedMsg and appDeletedMsg tell the root model a write succeeded; it closes the dialog.
type appCreatedMsg struct{ appDetail api.AppDetail }

type appDeletedMsg struct{ slug string }

var dialogBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)

func closeDialog() tea.Msg { return closeDialogMsg{} }

func lengthText(length int) string {
	return fmt.Sprintf("%d / %d", length, maxAppNameLength)
}

func renderDialog(borderColor color.Color, content string) string {
	return dialogBoxStyle.BorderForeground(borderColor).Render(content)
}

func componentsText(count int) string {
	if count == 1 {
		return "1 component"
	}
	return fmt.Sprintf("%d components", count)
}

// fieldErrorMessage prefers the API's message about one request field, then the general text.
func fieldErrorMessage(err error, field string) string {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		for _, detail := range apiErr.Details {
			if detail.Field == field {
				return detail.Message
			}
		}
	}
	return errorMessage(err)
}
