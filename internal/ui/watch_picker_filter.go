package ui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

// visibleProviders are the providers the filter lets through; the cursor moves over these.
func (m WatchPickerModel) visibleProviders() []api.Provider {
	return filterProviders(m.providers, m.providerQuery)
}

func (m WatchPickerModel) startProviderFilter() (WatchPickerModel, tea.Cmd) {
	m.isFilteringProviders = true
	m.providerFilter.SetValue(m.providerQuery)
	return m, m.providerFilter.Focus()
}

// handleProviderFilterKey filters live, as the user types: the catalog is already in memory, so
// no request is made. Only where the cursor ends up is fetched, after the usual settle delay.
func (m WatchPickerModel) handleProviderFilterKey(keyPress tea.KeyPressMsg) (WatchPickerModel, tea.Cmd) {
	switch keyPress.String() {
	case "enter":
		m.isFilteringProviders = false
		m.providerFilter.Blur()
		return m, nil
	case "esc":
		m.isFilteringProviders = false
		m.providerFilter.SetValue("")
		m.providerFilter.Blur()
		return m.withProviderQuery("")
	}
	var inputCmd tea.Cmd
	m.providerFilter, inputCmd = m.providerFilter.Update(keyPress)
	m, settleCmd := m.withProviderQuery(strings.TrimSpace(m.providerFilter.Value()))
	return m, tea.Batch(inputCmd, settleCmd)
}

// withProviderQuery filters the providers and puts the cursor on the first match.
func (m WatchPickerModel) withProviderQuery(query string) (WatchPickerModel, tea.Cmd) {
	m.providerQuery, m.providerIndex = query, 0
	m = m.withComponentQuery("")
	return m.scheduleSettle()
}

// scheduleSettle starts the timer after which the provider under the cursor is fetched; a timer
// from an earlier cursor position is ignored when it fires.
func (m WatchPickerModel) scheduleSettle() (WatchPickerModel, tea.Cmd) {
	m.settleSequence++
	sequence := m.settleSequence
	return m, tea.Tick(providerSettleDelay, func(time.Time) tea.Msg {
		return providerSettledMsg{sequence: sequence}
	})
}
