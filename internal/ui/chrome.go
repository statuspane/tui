package ui

import (
	"cmp"
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	// The screen sits inside a margin, and a blank line separates header, panel and footer.
	screenMarginX = 2
	screenMarginY = 1
	sectionGap    = 1

	headerHeight = 1
	// oneLineBoxHeight is a box around a single line: the line and its two borders.
	oneLineBoxHeight = 1 + borderCells
	footerHeight     = oneLineBoxHeight

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

	// The size of a classic terminal, used until the window size is known.
	fallbackWidth  = 80
	fallbackHeight = 24

	// The gap between two panels side by side.
	panelGapWidth = 2

	hintGap   = "  "
	brandText = "◉ STATUSPANE"
)

// keyHint is one entry of the footer: the key and what it does.
type keyHint struct{ key, label string }

func renderHint(hint keyHint) string {
	return keyStyle.Render(hint.key) + " " + mutedStyle.Render(hint.label)
}

// panelLook is how a panel's edges and title are coloured.
type panelLook struct{ border, title lipgloss.Style }

var (
	defaultPanelLook = panelLook{border: panelBorderStyle, title: accentStyle}
	accentPanelLook  = panelLook{border: lipgloss.NewStyle().Foreground(colorBlue), title: accentStyle}
	// dangerPanelLook is for what deletes something.
	dangerPanelLook = panelLook{border: errorStyle, title: errorStyle.Bold(true)}
)

// untintedPanelLook is the look of a pane that is not tinted by anything: the accent colour while
// it has the focus, quiet otherwise.
func untintedPanelLook(isFocused bool) panelLook {
	if isFocused {
		return accentPanelLook
	}
	return defaultPanelLook
}

func panel(title, body string, width, height int) string {
	return panelWithLook(defaultPanelLook, title, body, width, height)
}

// panelWithLook draws body in a rounded box of exactly width x height cells, with title set into
// the top edge. Lines longer than the box are cut, shorter ones padded; the box never wraps.
func panelWithLook(look panelLook, title, body string, width, height int) string {
	width, height = max(width, minChromeWidth), max(height, minPanelHeight)
	contentWidth, contentHeight := panelContentSize(width, height)

	lines := []string{panelTop(look, title, width)}
	bodyLines := strings.Split(body, "\n")
	for row := range contentHeight {
		bodyLine := ""
		if row < len(bodyLines) {
			bodyLine = clipTo(bodyLines[row], contentWidth)
		}
		lines = append(lines, panelRow(look, bodyLine, contentWidth))
	}
	lines = append(lines, panelBottom(look, width))
	return strings.Join(lines, "\n")
}

// panelContentSize is the room a panel of width x height leaves for its content.
func panelContentSize(width, height int) (contentWidth, contentHeight int) {
	return width - panelChromeWidth, height - panelChromeHeight
}

func panelTop(look panelLook, title string, width int) string {
	if title == "" {
		return look.border.Render("╭" + strings.Repeat("─", width-borderCells) + "╮")
	}
	title = clipTo(title, width-panelTitleFrameWidth)
	fill := width - panelTitleFrameWidth - lipgloss.Width(title)
	return look.border.Render("╭─ ") + look.title.Render(title) + look.border.Render(" "+strings.Repeat("─", fill)+"╮")
}

func panelRow(look panelLook, bodyLine string, contentWidth int) string {
	side := look.border.Render("│")
	margin := strings.Repeat(" ", panelPadding)
	return side + margin + padRight(bodyLine, contentWidth) + margin + side
}

func panelBottom(look panelLook, width int) string {
	return look.border.Render("╰" + strings.Repeat("─", width-borderCells) + "╯")
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

// alertBar is a footer-shaped box with a red edge that says something the footer's keys must not
// drown out, such as a question the user has to answer.
func alertBar(message string, width int) string {
	look := panelLook{border: errorStyle}
	return panelWithLook(look, "", errorStyle.Render(message), max(width, minChromeWidth), footerHeight)
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

// withFallbackSize replaces a size that is not known yet (zero) with a classic terminal's.
func withFallbackSize(width, height int) (int, int) {
	return cmp.Or(width, fallbackWidth), cmp.Or(height, fallbackHeight)
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

// centeredOver draws overlay in the middle of a window of windowWidth x windowHeight cells, over
// base: the cells of the overlay replace the base's, and the rest of the base stays visible around
// it. An overlay larger than the window sits at its top left corner.
func centeredOver(base, overlay string, windowWidth, windowHeight int) string {
	overlayWidth, overlayHeight := lipgloss.Size(overlay)
	overlayLayer := lipgloss.NewLayer(overlay).X(max((windowWidth-overlayWidth)/2, 0)).Y(max((windowHeight-overlayHeight)/2, 0)).Z(1)
	return lipgloss.NewCompositor(lipgloss.NewLayer(base, overlayLayer)).Render()
}

// inputBox is one line in a box width cells wide: left at the start and right at the end, as in a
// text field with its character count. The look says whether the field has the focus.
func inputBox(left, right string, look panelLook, width int) string {
	lineWidth, _ := panelContentSize(width, oneLineBoxHeight)
	return panelWithLook(look, "", headerBar(left, right, lineWidth), width, oneLineBoxHeight)
}

// clipTo cuts a possibly styled line to width cells, without a trailing ellipsis.
func clipTo(line string, width int) string {
	return lipgloss.NewStyle().MaxWidth(width).Render(line)
}

// wrapText breaks text into lines of at most width cells.
func wrapText(text string, width int) string {
	return lipgloss.NewStyle().Width(width).Render(text)
}

func padRight(text string, width int) string {
	return text + strings.Repeat(" ", max(width-lipgloss.Width(text), 0))
}
