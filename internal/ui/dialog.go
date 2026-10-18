package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	inputWidth       = 40
	maxAppNameLength = 100
	nameField        = "name"
)

// dialog is a modal the root model draws over the current screen. It handles every key press
// while it is open, so typing into a field never triggers a screen's shortcuts.
type dialog interface {
	update(msg tea.Msg) (dialog, tea.Cmd)
	view() string
}

type closeDialogMsg struct{}

// dialogWidth fits the longest line a dialog says on one line, and the name field with its count.
const (
	dialogWidth        = 64
	dialogContentWidth = dialogWidth - panelChromeWidth
)

var (
	accentDialogLook = panelLook{border: lipgloss.NewStyle().Foreground(colorBlue), title: accentStyle}
	// dangerDialogLook is for the dialog that deletes something.
	dangerDialogLook = panelLook{border: errorStyle, title: errorStyle.Bold(true)}
)

func closeDialog() tea.Msg { return closeDialogMsg{} }

// dialogBox is a dialog: a panel with title in its top edge, lines inside it, and a fixed width.
func dialogBox(look panelLook, title string, lines ...string) string {
	body := strings.Join(lines, "\n")
	return panelWithLook(look, title, body, dialogWidth, strings.Count(body, "\n")+1+panelChromeHeight)
}

// nameBox is the name field with how many characters it has, out of how many it can take.
func nameBox(input textinput.Model) string {
	return inputBox(input.View(), mutedStyle.Render(nameLengthText(input.Value())), defaultPanelLook, dialogContentWidth)
}

// wrappedText breaks text into lines that fit inside a dialog.
func wrappedText(text string) string {
	return lipgloss.NewStyle().Width(dialogContentWidth).Render(text)
}

// dialogHints is the keys of a dialog on one line.
func dialogHints(hints ...string) string {
	return strings.Join(hints, hintGap)
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
