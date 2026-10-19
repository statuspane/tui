package ui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// paneFilter is the text box that narrows one pane's list as the user types.
type paneFilter struct {
	input  textinput.Model
	query  string // what the list is narrowed by; empty means not narrowed
	active bool   // the box has the focus and takes every key
}

func newPaneFilter() paneFilter {
	input := textinput.New()
	input.Prompt = "/ "
	return paneFilter{input: input}
}

// start gives the box the focus with the kept query in it.
func (f paneFilter) start() (paneFilter, tea.Cmd) {
	f.active = true
	f.input.SetValue(f.query)
	return f, f.input.Focus()
}

// handleKey applies a key while the box has the focus: enter keeps the query and leaves the box,
// esc clears it and leaves, anything else is text. changed says the query is now different.
func (f paneFilter) handleKey(keyPress tea.KeyPressMsg) (filter paneFilter, changed bool, cmd tea.Cmd) {
	switch keyPress.String() {
	case "enter":
		f.active = false
		f.input.Blur()
		return f, false, nil
	case "esc":
		changed = f.query != ""
		return f.cleared(), changed, nil
	}
	f.input, cmd = f.input.Update(keyPress)
	changed = f.input.Value() != f.query
	f.query = f.input.Value()
	return f, changed, cmd
}

// update passes on what the box is waiting for besides keys, such as its cursor blinking.
func (f paneFilter) update(msg tea.Msg) (paneFilter, tea.Cmd) {
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return f, cmd
}

// cleared is the filter with no query that has let go of the focus.
func (f paneFilter) cleared() paneFilter {
	f.active, f.query = false, ""
	f.input.SetValue("")
	f.input.Blur()
	return f
}

func (f paneFilter) isNarrowing() bool {
	return strings.TrimSpace(f.query) != ""
}

// line is what the pane shows for the filter: the box while typing, the kept query, or a hint.
func (f paneFilter) line(hint string) string {
	switch {
	case f.active:
		return f.input.View()
	case f.isNarrowing():
		return mutedStyle.Render("Filter: " + f.query)
	}
	return mutedStyle.Render(hint)
}

// filteringHints are the key hints while a filter box has the focus.
var filteringHints = []keyHint{{"type", "to filter"}, {"⏎", "keep"}, {"esc", "clear"}}

// minFilterBoxGap is the least space between the filter and the match count in its box.
const minFilterBoxGap = 1

// filterBox is the filter in a box of its own, width cells wide, with how many items match at the
// right while a query is being typed or kept, and what they match on when there is room. The box
// takes the accent colour while it has the focus.
func filterBox(filter paneFilter, placeholder string, matchCount int, scope string, width int) string {
	lineWidth, _ := panelContentSize(width, oneLineBoxHeight)
	left := filter.line(placeholder)
	right := ""
	if filter.active || filter.isNarrowing() {
		right = matchCountText(matchCount, scope)
		if lipgloss.Width(left)+minFilterBoxGap+lipgloss.Width(right) > lineWidth {
			right = matchCountText(matchCount, "")
		}
	}
	return inputBox(left, mutedStyle.Render(right), untintedPanelLook(filter.active), width)
}

// matchCountText is "3 matches · name or group"; without a scope, just the count.
func matchCountText(count int, scope string) string {
	text := fmt.Sprintf("%d matches", count)
	if count == 1 {
		text = "1 match"
	}
	if scope == "" {
		return text
	}
	return text + " · " + scope
}
