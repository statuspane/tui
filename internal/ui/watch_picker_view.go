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
	minPickerRows      = 3

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
	case m.isSearching:
		return "type to filter · ⏎ keep · esc clear"
	case m.isFilteringProviders:
		return "type to filter · ⏎ keep · esc clear"
	case m.focusedPane == providersPane:
		return "↑/↓ move · tab switch pane · / filter providers · r refresh · s save · esc cancel"
	}
	return "↑/↓ move · tab switch pane · space watch or unwatch · / search components · r refresh · s save · esc cancel"
}

func (m WatchPickerModel) visibleRowCount() int {
	return max(m.height-pickerChromeHeight, minPickerRows)
}

func (m WatchPickerModel) providersView() string {
	lines := []string{paneTitle("Providers", m.focusedPane == providersPane), m.providerFilterLine()}
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

func (m WatchPickerModel) providerFilterLine() string {
	switch {
	case m.isFilteringProviders:
		return m.providerFilter.View()
	case m.providerQuery != "":
		return mutedStyle.Render("Filter: " + m.providerQuery + " (esc clears)")
	}
	return mutedStyle.Render("/ to filter providers")
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
	lines := []string{paneTitle(provider.Name+" components", m.focusedPane == componentsPane), m.searchLine()}
	if placeholder, isPlaceholder := m.componentsPlaceholder(); isPlaceholder {
		lines = append(lines, placeholder...)
	} else {
		lines = append(lines, m.componentRows()...)
	}
	return strings.Join(lines, "\n")
}

func (m WatchPickerModel) searchLine() string {
	switch {
	case m.isSearching:
		return m.search.View()
	case m.componentQuery != "":
		return mutedStyle.Render("Search: " + m.componentQuery + "  (/ to change, esc inside to clear)")
	}
	return mutedStyle.Render("/ to search by name or group")
}

// componentsPlaceholder says why there are no component rows to draw, if that is so.
func (m WatchPickerModel) componentsPlaceholder() ([]string, bool) {
	listing, ok := m.currentListing()
	if !ok {
		return nil, true
	}
	if err := m.cache.failure(listing); err != nil {
		return []string{errorStyle.Render(errorMessage(err)), mutedStyle.Render("Press r to try again.")}, true
	}
	components, loaded := m.cache.lookup(listing)
	switch {
	case !loaded:
		return []string{loadingText}, true
	case len(components) == 0:
		return []string{"This provider has no components."}, true
	case len(filterComponents(components, m.componentQuery)) == 0:
		return []string{"No components match."}, true
	}
	return nil, false
}

func (m WatchPickerModel) componentRows() []string {
	components := m.currentComponents()
	nameWidth, groupWidth, statusWidth := columnWidths(components)
	start, end := visibleRange(len(components), m.componentIndex, m.visibleRowCount())
	rows := make([]string, 0, end-start)
	for index := start; index < end; index++ {
		rows = append(rows, m.componentRow(components[index], index == m.componentIndex, nameWidth, groupWidth, statusWidth))
	}
	return rows
}

func (m WatchPickerModel) componentRow(component api.Component, isCursor bool, nameWidth, groupWidth, statusWidth int) string {
	marker, name := noSelectionMarker, fmt.Sprintf("%-*s", nameWidth, component.Name)
	if isCursor && m.focusedPane == componentsPane {
		marker, name = selectionMarker, selectedStyle.Render(name)
	}
	group := ""
	if component.GroupName != nil {
		group = *component.GroupName
	}
	return marker + watchMarker(m.selection.stateOf(keyOf(component))) + " " + name + columnGap +
		mutedStyle.Render(fmt.Sprintf("%-*s", groupWidth, group)) + columnGap + renderStatus(component.Status, statusWidth)
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

func columnWidths(components []api.Component) (nameWidth, groupWidth, statusWidth int) {
	for _, component := range components {
		nameWidth = max(nameWidth, lipgloss.Width(component.Name))
		if component.GroupName != nil {
			groupWidth = max(groupWidth, lipgloss.Width(*component.GroupName))
		}
		statusWidth = max(statusWidth, lipgloss.Width(statusLabel(component.Status)))
	}
	return nameWidth, groupWidth, statusWidth
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
