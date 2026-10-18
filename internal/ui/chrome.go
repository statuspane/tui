package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	headerHeight = 1
	// footerHeight is one line of key hints inside a border.
	footerHeight = 3

	// Below this the frame would have no room for content; the terminal wraps what is left.
	minChromeWidth = 20
	minPanelHeight = 3

	panelPadding = 1
	hintGap      = "  "
)

// keyHint is one entry of the footer: the key and what it does.
type keyHint struct{ key, label string }

func renderHint(hint keyHint) string {
	return keyStyle.Render(hint.key) + " " + mutedStyle.Render(hint.label)
}

// panel draws body in a rounded box of exactly width x height cells, with title set into the
// top edge. Lines longer than the box are cut, shorter ones padded; the box never wraps.
func panel(title, body string, width, height int) string {
	width, height = max(width, minChromeWidth), max(height, minPanelHeight)
	contentWidth := width - 2 - 2*panelPadding

	lines := make([]string, 0, height)
	lines = append(lines, panelTop(title, width))
	side := panelBorderStyle.Render("│")
	bodyLines := strings.Split(body, "\n")
	for row := range height - 2 {
		content := ""
		if row < len(bodyLines) {
			content = clipTo(bodyLines[row], contentWidth)
		}
		gap := strings.Repeat(" ", contentWidth-lipgloss.Width(content))
		lines = append(lines, side+strings.Repeat(" ", panelPadding)+content+gap+strings.Repeat(" ", panelPadding)+side)
	}
	lines = append(lines, panelBorderStyle.Render("╰"+strings.Repeat("─", width-2)+"╯"))
	return strings.Join(lines, "\n")
}

func panelTop(title string, width int) string {
	if title == "" {
		return panelBorderStyle.Render("╭" + strings.Repeat("─", width-2) + "╮")
	}
	// "╭─ " + title + " " + fill + "╮"
	title = clipTo(title, width-5)
	fill := width - 5 - lipgloss.Width(title)
	return panelBorderStyle.Render("╭─ ") + accentStyle.Render(title) + panelBorderStyle.Render(" "+strings.Repeat("─", fill)+"╮")
}

// headerBar is one line: left at the start, right at the end. Right is dropped when both
// don't fit.
func headerBar(left, right string, width int) string {
	width = max(width, minChromeWidth)
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return clipTo(left, width)
	}
	return left + strings.Repeat(" ", gap) + right
}

// footerBar is the key hints in a box of their own, with trailing at the far end. Hints that
// don't fit are dropped from the right, then trailing.
func footerBar(hints []keyHint, trailing string, width int) string {
	width = max(width, minChromeWidth)
	contentWidth := width - 2 - 2*panelPadding

	shown := make([]string, len(hints))
	for index, hint := range hints {
		shown[index] = renderHint(hint)
	}
	for count := len(shown); ; count-- {
		left := strings.Join(shown[:count], hintGap)
		gap := contentWidth - lipgloss.Width(left) - lipgloss.Width(trailing)
		if gap >= 1 {
			return panel("", left+strings.Repeat(" ", gap)+trailing, width, footerHeight)
		}
		if count == 0 {
			return panel("", left, width, footerHeight)
		}
	}
}

// bodyHeight is what is left of a screen height for the content between header and footer.
func bodyHeight(screenHeight int) int {
	return max(screenHeight-headerHeight-footerHeight, minPanelHeight)
}

// clipTo cuts a possibly styled line to width cells, without a trailing ellipsis.
func clipTo(line string, width int) string {
	return lipgloss.NewStyle().MaxWidth(max(width, 0)).Render(line)
}
