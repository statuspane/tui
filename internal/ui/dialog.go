package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	maxAppNameLength = 100
	nameField        = "name"

	// nameCountReserve is the room the name box keeps for the count ("100 / 100"), a gap and the
	// cursor; the text field takes the rest of its line.
	nameCountReserve = 12
	inputWidth       = dialogContentWidth - panelChromeWidth - nameCountReserve
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

func closeDialog() tea.Msg { return closeDialogMsg{} }

// dialogBox is a dialog: a panel with title in its top edge, lines inside it, and a fixed width.
func dialogBox(look panelLook, title string, lines []string) string {
	bodyText := strings.Join(lines, "\n")
	return panelWithLook(look, title, bodyText, dialogWidth, lipgloss.Height(bodyText)+panelChromeHeight)
}

// nameBox is the name field with how many characters it has, out of how many it can take.
func nameBox(nameInput textinput.Model) string {
	return inputBox(nameInput.View(), mutedStyle.Render(nameLengthText(nameInput.Value())), defaultPanelLook, dialogContentWidth)
}

// statusLines is what a dialog says while its request is running or after it failed, under a
// blank line; nothing otherwise.
func statusLines(isSubmitting bool, pending string, err error, errorText func(error) string) []string {
	switch {
	case isSubmitting:
		return []string{"", pending}
	case err != nil:
		return []string{"", errorStyle.Render(wrappedText(errorText(err)))}
	}
	return nil
}

// hintLines is the keys of a dialog on one line, under a blank line.
func hintLines(hints ...string) []string {
	return []string{"", strings.Join(hints, hintGap)}
}

// wrappedText breaks text into lines that fit inside a dialog.
func wrappedText(text string) string {
	return lipgloss.NewStyle().Width(dialogContentWidth).Render(text)
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
