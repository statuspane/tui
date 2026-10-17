package ui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

const (
	// Rows around the lists: title, status line, blank, pane title, search line, blank, help, and
	// the newline that ends the screen.
	pickerChromeHeight = 8
	minPickerRows      = 5 // room for a heading and a few rows under it

	paneGap     = "    "
	loadingText = "Loading…"

	// Wide enough for a short error while the providers are not there to size the pane.
	minProvidersWidth = 30
)

func (m WatchPickerModel) View() tea.View {
	title := titleStyle.Render("StatusPane") + mutedStyle.Render(" › ") + m.appName + mutedStyle.Render(" › ") + titleStyle.Render("Watch components")
	panes := lipgloss.JoinHorizontal(lipgloss.Top, m.providersView(), paneGap, m.componentsView())
	return tea.NewView(title + "\n" + m.statusLine() + "\n\n" + panes + "\n\n" + mutedStyle.Render(m.helpText()) + "\n")
}

func (m WatchPickerModel) statusLine() string {
	switch {
	case m.isSaving:
		return "Saving…"
	case m.saveErr != nil:
		return errorStyle.Render("Could not save: " + validationMessage(m.saveErr))
	case m.confirmingDiscard:
		return errorStyle.Render(fmt.Sprintf("Discard %s? Press esc again to discard, any other key to keep editing.", changesText(m.selection.changeCount())))
	case m.selection.changeCount() > 0:
		return changesText(m.selection.changeCount()) + " not saved yet"
	}
	return ""
}

func changesText(count int) string {
	if count == 1 {
		return "1 change"
	}
	return strconv.Itoa(count) + " changes"
}

func (m WatchPickerModel) helpText() string {
	switch {
	case m.providersFilter.active || m.componentsFilter.active:
		return filteringHelpText
	case m.focusedPane == providersPane:
		return "↑/↓ move · tab switch pane · / filter providers · r refresh · s save · esc cancel"
	}
	return "↑/↓ move · tab switch pane · space watch or unwatch · g toggle group · / filter components · r refresh · s save · esc cancel"
}

func (m WatchPickerModel) visibleRowCount() int {
	return max(m.height-pickerChromeHeight, minPickerRows)
}

func (m WatchPickerModel) providersView() string {
	lines := []string{paneTitle("Providers", m.focusedPane == providersPane), m.providersFilter.line("/ to filter providers")}
	switch {
	case m.providersErr != nil:
		lines = append(lines, errorStyle.Render(errorMessage(m.providersErr)))
	case !m.providersLoaded:
		lines = append(lines, loadingText)
	case len(m.visibleProviders()) == 0:
		lines = append(lines, "No providers match.")
	default:
		lines = append(lines, m.providerRows()...)
	}
	return lipgloss.NewStyle().Width(m.providersWidth()).Render(strings.Join(lines, "\n"))
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

func (m WatchPickerModel) componentsView() string {
	provider, ok := m.providerAtCursor()
	if !ok {
		return paneTitle("Components", m.focusedPane == componentsPane)
	}
	lines := []string{
		paneTitle(provider.Name+" components", m.focusedPane == componentsPane),
		m.componentsFilter.line("/ to filter by name or group"),
	}
	return strings.Join(append(lines, m.componentLines()...), "\n")
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
	lines := groupLines(ordered, func(component api.Component) bool { return m.selection.isChosen(keyOf(component)) })
	nameWidth, statusWidth := columnWidths(ordered)
	start, end := visibleRange(len(lines), lineOfComponent(lines, m.componentIndex), m.visibleRowCount())
	rows := make([]string, 0, end-start)
	for _, line := range lines[start:end] {
		if line.isHeading {
			rows = append(rows, headingRow(line))
			continue
		}
		rows = append(rows, m.componentRow(ordered[line.componentIndex], line.componentIndex == m.componentIndex, nameWidth, statusWidth))
	}
	return rows
}

// lineOfComponent is the position among the lines of the component at index.
func lineOfComponent(lines []componentLine, index int) int {
	for position, line := range lines {
		if !line.isHeading && line.componentIndex == index {
			return position
		}
	}
	return 0
}

func headingRow(line componentLine) string {
	return noSelectionMarker + titleStyle.Render(line.groupName) + columnGap +
		mutedStyle.Render(fmt.Sprintf("%d/%d watched", line.chosen, line.total))
}

func (m WatchPickerModel) componentRow(component api.Component, isCursor bool, nameWidth, statusWidth int) string {
	marker, name := noSelectionMarker, fmt.Sprintf("%-*s", nameWidth, component.Name)
	if isCursor && m.focusedPane == componentsPane {
		marker, name = selectionMarker, selectedStyle.Render(name)
	}
	return marker + watchMarker(m.selection.stateOf(keyOf(component))) + " " + name + columnGap + renderStatus(component.Status, statusWidth)
}

// watchMarker shows whether a component is watched, will be, will stop being, or is not.
func watchMarker(state watchState) string {
	switch state {
	case stateWatched:
		return lipgloss.NewStyle().Foreground(colorGreen).Render("[x]")
	case stateAdded:
		return lipgloss.NewStyle().Foreground(colorBlue).Render("[+]")
	case stateRemoved:
		return lipgloss.NewStyle().Foreground(colorRed).Render("[-]")
	}
	return mutedStyle.Render("[ ]")
}

func columnWidths(components []api.Component) (nameWidth, statusWidth int) {
	for _, component := range components {
		nameWidth = max(nameWidth, lipgloss.Width(component.Name))
		statusWidth = max(statusWidth, lipgloss.Width(statusLabel(component.Status)))
	}
	return nameWidth, statusWidth
}

func paneTitle(title string, isFocused bool) string {
	if isFocused {
		return titleStyle.Foreground(colorBlue).Render(title)
	}
	return mutedStyle.Render(title)
}

// visibleRange is the slice of total rows to draw so the cursor stays in view.
func visibleRange(total, cursor, rows int) (start, end int) {
	if total <= rows {
		return 0, total
	}
	start = min(max(cursor-rows/2, 0), total-rows)
	return start, start + rows
}
