package ui

import (
	"fmt"
	"image/color"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	inputWidth       = 44
	maxAppNameLength = 100
	nameField        = "name"
)

// dialog is a modal the root model shows instead of the current screen. It handles every key
// press while it is open, so typing into a field never triggers a screen's shortcuts.
type dialog interface {
	update(msg tea.Msg) (dialog, tea.Cmd)
	view() string
}

type closeDialogMsg struct{}

var dialogBoxStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)

func closeDialog() tea.Msg { return closeDialogMsg{} }

func renderDialog(borderColor color.Color, content string) string {
	return dialogBoxStyle.BorderForeground(borderColor).Render(content)
}

// newNameInput is the focused text box for an app's name, starting with value.
func newNameInput(value string) (textinput.Model, tea.Cmd) {
	input := textinput.New()
	input.Prompt = ""
	input.CharLimit = maxAppNameLength
	input.SetWidth(inputWidth)
	input.SetValue(value)
	return input, input.Focus()
}

func nameLengthText(name string) string {
	return fmt.Sprintf("%d / %d", len([]rune(name)), maxAppNameLength)
}
