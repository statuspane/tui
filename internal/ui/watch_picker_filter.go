package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

// visibleProviders are the providers the filter lets through; the cursor moves over these.
func (m WatchPickerModel) visibleProviders() []api.Provider {
	return filterProviders(m.providers, m.providersFilter.query)
}

// startFilter gives the focus to the filter of the pane that has it. Both filter live and in
// memory, so typing makes no request.
func (m WatchPickerModel) startFilter() (WatchPickerModel, tea.Cmd) {
	var cmd tea.Cmd
	if m.focusedPane == providersPane {
		m.providersFilter, cmd = m.providersFilter.start()
	} else {
		m.componentsFilter, cmd = m.componentsFilter.start()
	}
	return m, cmd
}

// handleProvidersFilterKey narrows the providers as the user types and puts the cursor on the
// first match; where it lands is fetched after the usual settle delay.
func (m WatchPickerModel) handleProvidersFilterKey(keyPress tea.KeyPressMsg) (WatchPickerModel, tea.Cmd) {
	filter, changed, inputCmd := m.providersFilter.handleKey(keyPress)
	m.providersFilter = filter
	if !changed {
		return m, inputCmd
	}
	m, settleCmd := m.landOnProvider(0)
	return m, tea.Batch(inputCmd, settleCmd)
}

// handleComponentsFilterKey narrows the provider's components as the user types and puts the
// cursor on the first match.
func (m WatchPickerModel) handleComponentsFilterKey(keyPress tea.KeyPressMsg) (WatchPickerModel, tea.Cmd) {
	filter, changed, cmd := m.componentsFilter.handleKey(keyPress)
	m.componentsFilter = filter
	if changed {
		m.componentIndex = 0
	}
	return m, cmd
}
