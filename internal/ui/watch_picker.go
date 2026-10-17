package ui

import (
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
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
	providerQuery   string

	cache          *componentCache
	settleSequence int
	componentIndex int
	componentQuery string
	search         textinput.Model
	isSearching    bool

	providerFilter       textinput.Model
	isFilteringProviders bool

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
	search := textinput.New()
	search.Prompt = "/ "
	providerFilter := textinput.New()
	providerFilter.Prompt = "/ "
	providers, hasProviders := cache.cachedProviders()
	return WatchPickerModel{
		backend:         backend,
		appSlug:         open.appSlug,
		appName:         open.appName,
		selection:       newWatchSelection(open.watchedComponents),
		providers:       providers,
		providersLoaded: hasProviders,
		cache:           cache,
		search:          search,
		providerFilter:  providerFilter,
		width:           width,
		height:          height,
	}
}

func (m WatchPickerModel) Init() tea.Cmd {
	if m.providersLoaded {
		return m.fetchListingIfMissing()
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
		return m, m.fetchListingIfMissing()

	case providersFailedMsg:
		m.providersErr = msg.err
		return m, nil

	case providerSettledMsg:
		if msg.sequence != m.settleSequence {
			return m, nil
		}
		return m, m.fetchListingIfMissing()

	case providerComponentsLoadedMsg:
		m.cache.store(msg.listing, msg.components)
		m.componentIndex = clampIndex(m.componentIndex, 0, len(m.currentComponents()))
		return m, nil

	case providerComponentsFailedMsg:
		m.cache.fail(msg.listing, msg.err)
		return m, nil

	case saveWatchedComponentsFailedMsg:
		m.isSaving, m.saveErr = false, msg.err
		return m, nil
	}

	// Messages the text box in use is waiting for, such as the cursor blinking.
	var cmd tea.Cmd
	switch {
	case m.isSearching:
		m.search, cmd = m.search.Update(msg)
	case m.isFilteringProviders:
		m.providerFilter, cmd = m.providerFilter.Update(msg)
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
	if m.isSearching {
		return m.handleSearchKey(keyPress)
	}
	if m.isFilteringProviders {
		return m.handleProviderFilterKey(keyPress)
	}

	wasConfirmingDiscard := m.confirmingDiscard
	m.confirmingDiscard = false

	switch keyPress.String() {
	case "esc":
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
	case "/":
		// The key searches the pane that has the focus: providers are filtered where they are,
		// components are searched on the server.
		if m.focusedPane == providersPane {
			return m.startProviderFilter()
		}
		return m.startSearch()
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

// handleSearchKey filters the provider's components live, as the user types. The list is already
// in memory, so no request is made.
func (m WatchPickerModel) handleSearchKey(keyPress tea.KeyPressMsg) (WatchPickerModel, tea.Cmd) {
	switch keyPress.String() {
	case "enter":
		m.isSearching = false
		m.search.Blur()
		return m, nil
	case "esc":
		m.isSearching = false
		m.search.SetValue("")
		m.search.Blur()
		return m.withComponentQuery(""), nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(keyPress)
	return m.withComponentQuery(strings.TrimSpace(m.search.Value())), cmd
}

// withComponentQuery filters the components on screen and puts the cursor on the first match.
func (m WatchPickerModel) withComponentQuery(query string) WatchPickerModel {
	m.componentQuery, m.componentIndex = query, 0
	return m
}

func (m WatchPickerModel) startSearch() (WatchPickerModel, tea.Cmd) {
	m.focusedPane, m.isSearching = componentsPane, true
	m.search.SetValue(m.componentQuery)
	return m, m.search.Focus()
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
		m.componentIndex = clampIndex(m.componentIndex, step, len(m.currentComponents()))
		return m, nil
	}
	previous := m.providerIndex
	m.providerIndex = clampIndex(m.providerIndex, step, len(m.visibleProviders()))
	m.componentIndex = 0
	if m.providerIndex != previous {
		// A search belongs to the provider it was typed for.
		m = m.withComponentQuery("")
	}
	return m.scheduleSettle()
}

// refresh asks again for what is on screen: the provider list if that failed, else the selected
// provider's components, which stay visible until the new answer arrives.
func (m WatchPickerModel) refresh() (WatchPickerModel, tea.Cmd) {
	if m.providersErr != nil {
		m.providersErr = nil
		return m, fetchProviders(m.backend)
	}
	listing, ok := m.currentListing()
	if !ok || m.cache.isPending(listing) {
		return m, nil
	}
	m.cache.forgetFailure(listing)
	m.cache.markPending(listing)
	return m, fetchProviderComponents(m.backend, listing)
}

func (m WatchPickerModel) toggleComponentAtCursor() WatchPickerModel {
	if m.focusedPane != componentsPane {
		return m
	}
	components := m.currentComponents()
	if m.componentIndex >= len(components) {
		return m
	}
	m.selection = m.selection.toggled(keyOf(components[m.componentIndex]))
	return m
}

func (m WatchPickerModel) providerAtCursor() (api.Provider, bool) {
	providers := m.visibleProviders()
	if m.providerIndex >= len(providers) {
		return api.Provider{}, false
	}
	return providers[m.providerIndex], true
}

func (m WatchPickerModel) currentListing() (componentListing, bool) {
	provider, ok := m.providerAtCursor()
	return componentListing{providerSlug: provider.Slug}, ok
}

// currentComponents are the selected provider's cached components that match the search on
// screen; nil while they are still loading.
func (m WatchPickerModel) currentComponents() []api.Component {
	listing, ok := m.currentListing()
	if !ok {
		return nil
	}
	components, _ := m.cache.lookup(listing)
	return filterComponents(components, m.componentQuery)
}

// fetchListingIfMissing fetches what the screen shows unless it is cached, already on its way, or
// failed: a failure is asked for again only when the user presses r, so a refused request, such
// as a rate limit, is not repeated on every visit.
func (m WatchPickerModel) fetchListingIfMissing() tea.Cmd {
	listing, ok := m.currentListing()
	if !ok {
		return nil
	}
	if _, cached := m.cache.lookup(listing); cached || m.cache.isPending(listing) || m.cache.failure(listing) != nil {
		return nil
	}
	m.cache.markPending(listing)
	return fetchProviderComponents(m.backend, listing)
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
