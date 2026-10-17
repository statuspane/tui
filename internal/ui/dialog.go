package ui

import (
	"image/color"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const inputWidth = 44

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
