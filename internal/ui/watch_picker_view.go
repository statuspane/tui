package ui

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

const (
	loadingText = "Loading…"

	// Wide enough for a short error while the providers are not there to size the pane.
	minProvidersWidth = 30

	minPickerRows = 5 // room for a heading and a few rows under it

	// Under a panel's rows come a rule and a footer line, and above them the filter's box.
	panelRuleAndFooterHeight  = 2
	pickerPanelOverheadHeight = panelChromeHeight + oneLineBoxHeight + panelRuleAndFooterHeight

	providersFilterPlaceholder  = "/ to filter providers"
	componentsFilterPlaceholder = "/ to filter by name or group"
	providersMatchScope         = "name, slug or category"
	componentsMatchScope        = "name or group"
)

func (m WatchPickerModel) View() tea.View {
	layout := m.layout()
	header := headerBar(m.headerLeft(), m.headerNotice(), layout.contentWidth)
	panels := lipgloss.JoinHorizontal(lipgloss.Top,
		m.providersPanel(layout), strings.Repeat(" ", panelGapWidth), m.componentsPanel(layout))
	return tea.NewView(stackedScreen(header, panels, m.footer(layout.contentWidth)))
}

// pickerLayout is how the window is shared out: the width inside the margin, each panel's width,
// the panels' height and how many rows of a list fit in one.
type pickerLayout struct{ contentWidth, providersWidth, componentsWidth, panelHeight, rowCount int }

// layout gives the providers panel what its content needs, but no more than half the width, and
// the components panel the rest. The panels are never shorter than room for minPickerRows, so a
// short window still shows the cursor row.
func (m WatchPickerModel) layout() pickerLayout {
	windowWidth, windowHeight := withFallbackSize(m.width, m.height)
	contentWidth := layoutWidth(windowWidth)
	panelSpace := contentWidth - panelGapWidth
	providersWidth := min(m.providersWidth()+panelChromeWidth, panelSpace/2)
	panelHeight := max(bodyHeight(windowHeight), pickerPanelOverheadHeight+minPickerRows)
	return pickerLayout{
		contentWidth:    contentWidth,
		providersWidth:  providersWidth,
		componentsWidth: panelSpace - providersWidth,
		panelHeight:     panelHeight,
		rowCount:        panelHeight - pickerPanelOverheadHeight,
	}
}

func (m WatchPickerModel) visibleRowCount() int {
	return m.layout().rowCount
}

func (m WatchPickerModel) headerLeft() string {
	return accentStyle.Render(brandText) + mutedStyle.Render(" › ") + m.appName + mutedStyle.Render(" › ") + titleStyle.Render("Watch components")
}

// headerNotice is what the screen is doing about the user's changes: saving them, or holding
// some that are not saved yet.
func (m WatchPickerModel) headerNotice() string {
	switch {
	case m.isSaving:
		return "Saving…"
	case m.selection.changeCount() > 0:
		return warningStyle.Render("● " + unsavedChangesText(m.selection.changeCount()))
	}
	return ""
}

// alertText is a question or an error the footer's keys must not drown out; empty for none.
func (m WatchPickerModel) alertText() string {
	switch {
	case m.saveErr != nil:
		return "Could not save: " + validationMessage(m.saveErr)
	case m.confirmingDiscard:
		return fmt.Sprintf("Discard %s? Press esc again to discard, any other key to keep editing.", changesText(m.selection.changeCount()))
	}
	return ""
}

func changesText(count int) string {
	if count == 1 {
		return "1 change"
	}
	return strconv.Itoa(count) + " changes"
}

func unsavedChangesText(count int) string {
	if count == 1 {
		return "1 unsaved change"
	}
	return strconv.Itoa(count) + " unsaved changes"
}

func (m WatchPickerModel) footer(contentWidth int) string {
	if alert := m.alertText(); alert != "" {
		return alertBar(alert, contentWidth)
	}
	if m.providersFilter.active || m.componentsFilter.active {
		return footerBar(filteringHints, "", contentWidth)
	}
	return footerBar(m.keyHints(), renderHint(m.escapeHint()), contentWidth)
}

// escapeHint says what esc does now: clear the focused pane's kept filter, or cancel the picker.
func (m WatchPickerModel) escapeHint() keyHint {
	if m.focusedFilter().isNarrowing() {
		return keyHint{"esc", "clear filter"}
	}
	return keyHint{"esc", "cancel"}
}

func (m WatchPickerModel) keyHints() []keyHint {
	if m.focusedPane == providersPane {
		return []keyHint{{"↑↓", "move"}, {"/", "filter"}, {"tab", "switch pane"}, {"s", "save"}, {"r", "refresh"}}
	}
	return []keyHint{{"↑↓", "move"}, {"space", "toggle"}, {"g", "group"}, {"/", "filter"}, {"tab", "switch pane"}, {"s", "save"}, {"r", "refresh"}}
}

func (m WatchPickerModel) providersPanel(layout pickerLayout) string {
	panelContentWidth, _ := panelContentSize(layout.providersWidth, layout.panelHeight)
	filterBoxView := filterBox(m.providersFilter, providersFilterPlaceholder, len(m.visibleProviders()), providersMatchScope, panelContentWidth)
	footer := mutedStyle.Render(fmt.Sprintf("%d watched in total", m.selection.chosenCount()))
	body := stackedPanelBody(filterBoxView, m.providerLines(), layout.rowCount, footer, panelContentWidth)
	return panelWithLook(untintedPanelLook(m.focusedPane == providersPane), "Providers", body, layout.providersWidth, layout.panelHeight)
}

func (m WatchPickerModel) componentsPanel(layout pickerLayout) string {
	panelContentWidth, _ := panelContentSize(layout.componentsWidth, layout.panelHeight)
	filterBoxView := filterBox(m.componentsFilter, componentsFilterPlaceholder, len(m.visibleComponents()), componentsMatchScope, panelContentWidth)
	body := stackedPanelBody(filterBoxView, m.componentLines(), layout.rowCount, componentsLegend(), panelContentWidth)
	return panelWithLook(untintedPanelLook(m.focusedPane == componentsPane), m.componentsTitle(), body, layout.componentsWidth, layout.panelHeight)
}

// componentsTitle names the provider under the cursor and how many components it has, once known.
func (m WatchPickerModel) componentsTitle() string {
	provider, ok := m.providerAtCursor()
	if !ok {
		return "Components"
	}
	title := provider.Name + " components"
	if listed, loaded := m.providerComponents(); loaded {
		title += fmt.Sprintf(" · %d", len(listed))
	}
	return title
}

func componentsLegend() string {
	return addedStyle.Render("[+]") + mutedStyle.Render(" to watch · ") + removedStyle.Render("[-]") +
		mutedStyle.Render(" to stop · saved with ") + keyStyle.Render("s")
}

// stackedPanelBody stacks a panel's filter box, its rows padded to rowCount lines, a rule and its
// footer, so the footer stays at the bottom however few rows there are.
func stackedPanelBody(filterBoxView string, rows []string, rowCount int, panelFooter string, panelContentWidth int) string {
	paddedRows := append(slices.Clone(rows), make([]string, max(rowCount-len(rows), 0))...)
	rule := panelBorderStyle.Render(strings.Repeat("┄", panelContentWidth))
	return strings.Join([]string{filterBoxView, strings.Join(paddedRows, "\n"), rule, panelFooter}, "\n")
}

// providerLines are the provider rows to draw, or, when there are none, why.
func (m WatchPickerModel) providerLines() []string {
	switch {
	case m.providersErr != nil:
		return []string{errorStyle.Render(errorMessage(m.providersErr))}
	case !m.providersLoaded:
		return []string{loadingText}
	case len(m.visibleProviders()) == 0:
		return []string{"No providers match."}
	}
	return m.providerRows()
}

func (m WatchPickerModel) providerNameWidth() int {
	width := 0
	for _, provider := range m.providers {
		width = max(width, lipgloss.Width(provider.Name))
	}
	return width
}

func (m WatchPickerModel) providersWidth() int {
	watchedWidth := 0
	for _, provider := range m.providers {
		watchedWidth = max(watchedWidth, lipgloss.Width(m.watchedText(provider)))
	}
	width := max(m.providerNameWidth(), lipgloss.Width("Providers"))
	return max(width+lipgloss.Width(selectionMarker)+lipgloss.Width(columnGap)+watchedWidth, minProvidersWidth)
}

// watchedText says how many of the app's chosen components belong to the provider; empty for none.
func (m WatchPickerModel) watchedText(provider api.Provider) string {
	count := m.selection.chosenCountFor(provider.Slug)
	if count == 0 {
		return ""
	}
	return strconv.Itoa(count) + " watched"
}

func (m WatchPickerModel) providerRows() []string {
	nameWidth := m.providerNameWidth()
	providers := m.visibleProviders()
	start, end := visibleRange(len(providers), m.providerIndex, m.visibleRowCount())
	rows := make([]string, 0, end-start)
	for index := start; index < end; index++ {
		provider := providers[index]
		marker, name := noSelectionMarker, fmt.Sprintf("%-*s", nameWidth, provider.Name)
		if index == m.providerIndex {
			marker, name = selectionMarker, selectedStyle.Render(name)
		}
		rows = append(rows, marker+name+columnGap+mutedStyle.Render(m.watchedText(provider)))
	}
	return rows
}

// componentLines are the rows to draw under the filter line, or, when there are none, why.
func (m WatchPickerModel) componentLines() []string {
	providerSlug, _ := m.providerSlugAtCursor()
	if err := m.cache.failure(providerSlug); err != nil {
		return []string{errorStyle.Render(errorMessage(err)), mutedStyle.Render("Press r to try again.")}
	}
	all, loaded := m.providerComponents()
	visible := m.visibleComponents()
	switch {
	case !loaded:
		return []string{loadingText}
	case len(all) == 0:
		return []string{"This provider has no components."}
	case len(visible) == 0:
		return []string{"No components match."}
	}
	return m.componentRows(visible)
}

// componentRows draws the ordered components under their group headings, scrolled so the row
// under the cursor is in view; the headings take rows like any other line.
func (m WatchPickerModel) componentRows(ordered []api.Component) []string {
	lines := linesWithHeadings(ordered, m.selection)
	nameWidth, statusWidth := columnWidths(ordered)
	cursorLine, _ := lineOfComponent(lines, m.componentIndex) // no cursor line: show the top
	start, end := visibleRange(len(lines), cursorLine, m.visibleRowCount())
	rows := make([]string, 0, end-start)
	for _, line := range lines[start:end] {
		if line.isHeading {
			rows = append(rows, headingRow(line))
			continue
		}
		rows = append(rows, m.componentRow(ordered[line.index], line.index == m.componentIndex, nameWidth, statusWidth))
	}
	return rows
}

func headingRow(line paneLine) string {
	return noSelectionMarker + titleStyle.Render(line.groupName) + columnGap +
		mutedStyle.Render(fmt.Sprintf("%d/%d watched", line.watched, line.total))
}

func (m WatchPickerModel) componentRow(component api.Component, isCursor bool, nameWidth, statusWidth int) string {
	state := m.selection.stateOf(keyOf(component))
	padding := strings.Repeat(" ", max(nameWidth-lipgloss.Width(component.Name), 0))
	if isCursor && m.focusedPane == componentsPane {
		return cursorRow(component, state, padding, statusWidth)
	}
	return stateRow(component, state, padding, statusWidth)
}

// cursorRow is a bar across the whole row, drawn from plain text because styles nested inside a
// bar would end it early.
func cursorRow(component api.Component, state watchState, padding string, statusWidth int) string {
	status := fmt.Sprintf("%-*s", statusWidth, statusLabel(component.Status))
	return cursorRowStyle.Render(selectionMarker + markerText(state) + " " + component.Name + padding + columnGap + status)
}

// stateRow shows the state of the component in its marker and across its name. The padding stays
// outside the name's style: a strikethrough would run through it.
func stateRow(component api.Component, state watchState, padding string, statusWidth int) string {
	return noSelectionMarker + styledMarker(state) + " " + nameStyleFor(state).Render(component.Name) + padding + columnGap + renderStatus(component.Status, statusWidth)
}

// markerText is how a state reads without colour: watched, to be added, to be removed, not watched.
func markerText(state watchState) string {
	switch state {
	case stateWatched:
		return "[x]"
	case stateAdded:
		return "[+]"
	case stateRemoved:
		return "[-]"
	}
	return "[ ]"
}

func styledMarker(state watchState) string {
	return markerStyleFor(state).Render(markerText(state))
}

func columnWidths(components []api.Component) (nameWidth, statusWidth int) {
	for _, component := range components {
		nameWidth = max(nameWidth, lipgloss.Width(component.Name))
		statusWidth = max(statusWidth, lipgloss.Width(statusLabel(component.Status)))
	}
	return nameWidth, statusWidth
}

// visibleRange is the slice of total rows to draw so the cursor stays in view.
func visibleRange(total, cursor, rows int) (start, end int) {
	if total <= rows {
		return 0, total
	}
	start = min(max(cursor-rows/2, 0), total-rows)
	return start, start + rows
}
