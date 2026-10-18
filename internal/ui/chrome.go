package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	// The screen sits inside a margin, and a blank line separates header, panel and footer.
	screenMarginX = 2
	screenMarginY = 1
	sectionGap    = 1

	headerHeight = 1
	// footerHeight is one line of key hints inside a border.
	footerHeight = 1 + borderCells

	// Below this the frame would have no room for content; the terminal wraps what is left.
	minChromeWidth = 20
	minPanelHeight = 3

	// A border takes two cells along each axis, one on each side.
	borderCells  = 2
	panelPadding = 2
	// What a panel's border and padding take from the content.
	panelChromeWidth  = borderCells + 2*panelPadding
	panelChromeHeight = borderCells
	// A panel's top edge around its title: "╭─ " before it, " " and "╮" after.
	panelTitleFrameWidth = 5

	hintGap   = "  "
	brandText = "◉ STATUSPANE"
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
	contentWidth, contentHeight := panelContentSize(width, height)

	lines := []string{panelTop(title, width)}
	bodyLines := strings.Split(body, "\n")
	for row := range contentHeight {
		bodyLine := ""
		if row < len(bodyLines) {
			bodyLine = clipTo(bodyLines[row], contentWidth)
		}
		lines = append(lines, panelRow(bodyLine, contentWidth))
	}
	lines = append(lines, panelBottom(width))
	return strings.Join(lines, "\n")
}

// panelContentSize is the room a panel of width x height leaves for its content.
func panelContentSize(width, height int) (contentWidth, contentHeight int) {
	return width - panelChromeWidth, height - panelChromeHeight
}

func panelTop(title string, width int) string {
	if title == "" {
		return panelBorderStyle.Render("╭" + strings.Repeat("─", width-borderCells) + "╮")
	}
	fill := width - panelTitleFrameWidth - lipgloss.Width(title)
	return panelBorderStyle.Render("╭─ ") + accentStyle.Render(title) + panelBorderStyle.Render(" "+strings.Repeat("─", fill)+"╮")
}

func panelRow(bodyLine string, contentWidth int) string {
	side := panelBorderStyle.Render("│")
	margin := strings.Repeat(" ", panelPadding)
	return side + margin + padRight(bodyLine, contentWidth) + margin + side
}

func panelBottom(width int) string {
	return panelBorderStyle.Render("╰" + strings.Repeat("─", width-borderCells) + "╯")
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

// footerBar is the key hints in a box of their own, with trailing at the far end.
func footerBar(hints []keyHint, trailing string, width int) string {
	width = max(width, minChromeWidth)
	renderedHints := make([]string, len(hints))
	for index, hint := range hints {
		renderedHints[index] = renderHint(hint)
	}
	contentWidth, _ := panelContentSize(width, footerHeight)
	return panel("", footerLine(renderedHints, trailing, contentWidth), width, footerHeight)
}

// footerLine fits the most hints beside trailing: hints are dropped from the right first, and
// trailing goes last.
func footerLine(renderedHints []string, trailing string, contentWidth int) string {
	for hintCount := len(renderedHints); hintCount >= 0; hintCount-- {
		hintsText := strings.Join(renderedHints[:hintCount], hintGap)
		if gap := contentWidth - lipgloss.Width(hintsText) - lipgloss.Width(trailing); gap >= 1 {
			return hintsText + strings.Repeat(" ", gap) + trailing
		}
	}
	return ""
}

// layoutWidth is the width of the screen's content, inside the margin.
func layoutWidth(screenWidth int) int {
	return screenWidth - 2*screenMarginX
}

// bodyHeight is what is left of a screen height for the content between header and footer.
func bodyHeight(screenHeight int) int {
	taken := 2*screenMarginY + headerHeight + 2*sectionGap + footerHeight
	return max(screenHeight-taken, minPanelHeight)
}

// stackedScreen stacks the header, the body and the footer with a blank line between them,
// inside the screen margin.
func stackedScreen(header, body, footer string) string {
	stacked := lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", footer)
	return lipgloss.NewStyle().Padding(screenMarginY, screenMarginX).Render(stacked)
}

// clipTo cuts a possibly styled line to width cells, without a trailing ellipsis.
func clipTo(line string, width int) string {
	return lipgloss.NewStyle().MaxWidth(width).Render(line)
}

func padRight(text string, width int) string {
	return text + strings.Repeat(" ", max(width-lipgloss.Width(text), 0))
}
