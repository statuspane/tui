package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

type pickerPane int

const (
	providersPane pickerPane = iota
	componentsPane
)

// providerSettleDelay is how long the cursor has to rest on a provider before its components are
// fetched. Scrolling through the catalog, for example by holding a key, must not ask the API
// about every provider passed on the way.
const providerSettleDelay = 300 * time.Millisecond

type WatchPickerModel struct {
	backend WatchPickerBackend

	appSlug string
	appName string

	selection watchSelection

	providers       []api.Provider
	providersLoaded bool
	providersErr    error
	providerIndex   int
	providersFilter paneFilter

	cache            *componentCache
	settleSequence   int
	componentIndex   int
	componentsFilter paneFilter

	focusedPane       pickerPane
	isSaving          bool
	saveErr           error
	confirmingDiscard bool

	width  int
	height int
}

// NewWatchPickerModel starts from the providers in the cache, if it has them; the cache outlives
// the picker, so reopening it asks the API for nothing it already knows.
func NewWatchPickerModel(backend WatchPickerBackend, cache *componentCache, open openWatchPickerMsg, width, height int) WatchPickerModel {
	providers, hasProviders := cache.cachedProviders()
	return WatchPickerModel{
		backend:          backend,
		appSlug:          open.appSlug,
		appName:          open.appName,
		selection:        newWatchSelection(open.watchedComponents),
		providers:        providers,
		providersLoaded:  hasProviders,
		providersFilter:  newPaneFilter(),
		cache:            cache,
		componentsFilter: newPaneFilter(),
		width:            width,
		height:           height,
	}
}

func (m WatchPickerModel) Init() tea.Cmd {
	if m.providersLoaded {
		return m.fetchComponentsIfMissing()
	}
	return fetchProviders(m.backend)
}

func (m WatchPickerModel) Update(msg tea.Msg) (WatchPickerModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case providersLoadedMsg:
		m.cache.storeProviders(msg.providers)
		m.providers, m.providersLoaded, m.providersErr = msg.providers, true, nil
		return m, m.fetchComponentsIfMissing()

	case providersFailedMsg:
		m.providersErr = msg.err
		return m, nil

	case providerSettledMsg:
		if msg.sequence != m.settleSequence {
			return m, nil
		}
		return m, m.fetchComponentsIfMissing()

	case providerComponentsLoadedMsg:
		m.cache.store(msg.providerSlug, msg.components)
		m.componentIndex = clampIndex(m.componentIndex, 0, len(m.visibleComponents()))
		return m, nil

	case providerComponentsFailedMsg:
		m.cache.fail(msg.providerSlug, msg.err)
		return m, nil

	case saveWatchedComponentsFailedMsg:
		m.isSaving, m.saveErr = false, msg.err
		return m, nil
	}

	// Messages the filter box in use is waiting for, such as its cursor blinking.
	var cmd tea.Cmd
	switch {
	case m.providersFilter.active:
		m.providersFilter, cmd = m.providersFilter.update(msg)
	case m.componentsFilter.active:
		m.componentsFilter, cmd = m.componentsFilter.update(msg)
	}
	return m, cmd
}

func (m WatchPickerModel) handleKey(keyPress tea.KeyPressMsg) (WatchPickerModel, tea.Cmd) {
	if keyPress.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.isSaving {
		return m, nil
	}
	if m.providersFilter.active {
		return m.handleProvidersFilterKey(keyPress)
	}
	if m.componentsFilter.active {
		return m.handleComponentsFilterKey(keyPress)
	}

	wasConfirmingDiscard := m.confirmingDiscard
	m.confirmingDiscard = false

	switch keyPress.String() {
	case "esc":
		if m.focusedFilter().isNarrowing() {
			return m.clearFocusedFilter(keyPress)
		}
		return m.cancelOrConfirmDiscard(wasConfirmingDiscard)
	case "tab":
		m.focusedPane = otherPane(m.focusedPane)
	case "enter":
		if m.focusedPane == providersPane {
			m.focusedPane = componentsPane
		}
	case "up", "k":
		return m.moveCursor(-1)
	case "down", "j":
		return m.moveCursor(1)
	case "space":
		return m.toggleComponentAtCursor(), nil
	case "g":
		return m.toggleGroupAtCursor(), nil
	case "/":
		return m.startFilter()
	case "r":
		return m.refresh()
	case "s":
		return m.startSave()
	}
	return m, nil
}

// cancelOrConfirmDiscard closes the picker, or, when closing would lose changes, asks first: the
// second esc in a row discards.
func (m WatchPickerModel) cancelOrConfirmDiscard(wasConfirmingDiscard bool) (WatchPickerModel, tea.Cmd) {
	if m.selection.changeCount() == 0 || wasConfirmingDiscard {
		return m, closeWatchPicker
	}
	m.confirmingDiscard = true
	return m, nil
}

func (m WatchPickerModel) startSave() (WatchPickerModel, tea.Cmd) {
	if m.selection.changeCount() == 0 {
		return m, nil
	}
	m.isSaving, m.saveErr = true, nil
	return m, saveWatchedComponents(m.backend, m.appSlug, m.selection.componentRefs())
}

func (m WatchPickerModel) moveCursor(step int) (WatchPickerModel, tea.Cmd) {
	if m.focusedPane == componentsPane {
		m.componentIndex = clampIndex(m.componentIndex, step, len(m.visibleComponents()))
		return m, nil
	}
	index := clampIndex(m.providerIndex, step, len(m.visibleProviders()))
	if index == m.providerIndex {
		return m, nil
	}
	return m.landOnProvider(index)
}

// landOnProvider puts the cursor on the visible provider at index and drops what belonged to the
// one it left, the component cursor and the component filter. The provider's components are
// fetched once the cursor rests there (see scheduleSettle).
func (m WatchPickerModel) landOnProvider(index int) (WatchPickerModel, tea.Cmd) {
	m.providerIndex, m.componentIndex = index, 0
	m.componentsFilter = m.componentsFilter.cleared()
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

// refresh asks again for what is on screen: the provider list if that failed, else the selected
// provider's components, which stay visible until the new answer arrives.
func (m WatchPickerModel) refresh() (WatchPickerModel, tea.Cmd) {
	if m.providersErr != nil {
		m.providersErr = nil
		return m, fetchProviders(m.backend)
	}
	providerSlug, ok := m.providerSlugAtCursor()
	if !ok || m.cache.isPending(providerSlug) {
		return m, nil
	}
	m.cache.forgetFailure(providerSlug)
	m.cache.markPending(providerSlug)
	return m, fetchProviderComponents(m.backend, providerSlug)
}

// componentsAtCursor are the visible components when the cursor is on one of them in the
// components pane; ok is false otherwise.
func (m WatchPickerModel) componentsAtCursor() (components []api.Component, ok bool) {
	if m.focusedPane != componentsPane {
		return nil, false
	}
	components = m.visibleComponents()
	return components, m.componentIndex < len(components)
}

func (m WatchPickerModel) toggleComponentAtCursor() WatchPickerModel {
	components, ok := m.componentsAtCursor()
	if !ok {
		return m
	}
	m.selection = m.selection.toggled(keyOf(components[m.componentIndex]))
	return m
}

// toggleGroupAtCursor watches the rest of the visible components of the group under the cursor
// or, when they are all chosen already, unwatches them all. Components a filter hides are left
// alone: the user never changes what cannot be seen.
func (m WatchPickerModel) toggleGroupAtCursor() WatchPickerModel {
	components, ok := m.componentsAtCursor()
	if !ok {
		return m
	}
	start, end := groupBounds(components, m.componentIndex)
	m.selection = m.selection.toggledAll(components[start:end])
	return m
}

func (m WatchPickerModel) providerAtCursor() (api.Provider, bool) {
	providers := m.visibleProviders()
	if m.providerIndex >= len(providers) {
		return api.Provider{}, false
	}
	return providers[m.providerIndex], true
}

func (m WatchPickerModel) providerSlugAtCursor() (string, bool) {
	provider, ok := m.providerAtCursor()
	return provider.Slug, ok
}

// providerComponents are all of the selected provider's cached components, whatever the filter
// says; loaded is false while they are on their way.
func (m WatchPickerModel) providerComponents() (components []api.Component, loaded bool) {
	providerSlug, ok := m.providerSlugAtCursor()
	if !ok {
		return nil, false
	}
	return m.cache.lookup(providerSlug)
}

// visibleComponents are the selected provider's components that the filter lets through, with
// each group together; the cursor moves over these, in this order, and group headings are only
// drawn between them. Nil while they are still loading.
func (m WatchPickerModel) visibleComponents() []api.Component {
	components, _ := m.providerComponents()
	return orderedByGroup(filterComponents(components, m.componentsFilter.query))
}

// fetchComponentsIfMissing fetches what the screen shows unless it is cached, already on its way,
// or failed: a failure is asked for again only when the user presses r, so a refused request, such
// as a rate limit, is not repeated on every visit.
func (m WatchPickerModel) fetchComponentsIfMissing() tea.Cmd {
	providerSlug, ok := m.providerSlugAtCursor()
	if !ok {
		return nil
	}
	if _, cached := m.cache.lookup(providerSlug); cached || m.cache.isPending(providerSlug) || m.cache.failure(providerSlug) != nil {
		return nil
	}
	m.cache.markPending(providerSlug)
	return fetchProviderComponents(m.backend, providerSlug)
}

func otherPane(pane pickerPane) pickerPane {
	if pane == providersPane {
		return componentsPane
	}
	return providersPane
}

// clampIndex moves an index by step and keeps it inside a list of the given length.
func clampIndex(index, step, length int) int {
	return min(max(index+step, 0), max(length-1, 0))
}
