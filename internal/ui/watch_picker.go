package ui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

type pickerPane int

const (
	providersPane pickerPane = iota
	componentsPane
)

type WatchPickerModel struct {
	backend WatchPickerBackend

	appSlug string
	appName string

	selection watchSelection

	providers       []api.Provider
	providersLoaded bool
	providersErr    error
	providerIndex   int

	cache          *componentCache
	componentIndex int
	query          string
	search         textinput.Model
	isSearching    bool

	focusedPane       pickerPane
	isSaving          bool
	saveErr           error
	confirmingDiscard bool

	width  int
	height int
}

func NewWatchPickerModel(backend WatchPickerBackend, open openWatchPickerMsg, width, height int) WatchPickerModel {
	search := textinput.New()
	search.Prompt = "/ "
	return WatchPickerModel{
		backend:   backend,
		appSlug:   open.appSlug,
		appName:   open.appName,
		selection: newWatchSelection(open.watchedComponents),
		cache:     newComponentCache(),
		search:    search,
		width:     width,
		height:    height,
	}
}

func (m WatchPickerModel) Init() tea.Cmd {
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
		m.providers, m.providersLoaded, m.providersErr = msg.providers, true, nil
		return m, m.fetchListingIfMissing()

	case providersFailedMsg:
		m.providersErr = msg.err
		return m, nil

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

	if m.isSearching {
		var cmd tea.Cmd
		m.search, cmd = m.search.Update(msg)
		return m, cmd
	}
	return m, nil
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
		return m.startSearch()
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

func (m WatchPickerModel) handleSearchKey(keyPress tea.KeyPressMsg) (WatchPickerModel, tea.Cmd) {
	switch keyPress.String() {
	case "enter":
		return m.applySearch(strings.TrimSpace(m.search.Value()))
	case "esc":
		m.search.SetValue("")
		return m.applySearch("")
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(keyPress)
	return m, cmd
}

func (m WatchPickerModel) applySearch(query string) (WatchPickerModel, tea.Cmd) {
	m.query, m.isSearching, m.componentIndex = query, false, 0
	m.search.Blur()
	return m, m.fetchListingIfMissing()
}

func (m WatchPickerModel) startSearch() (WatchPickerModel, tea.Cmd) {
	m.focusedPane, m.isSearching = componentsPane, true
	m.search.SetValue(m.query)
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
	m.providerIndex = clampIndex(m.providerIndex, step, len(m.providers))
	m.componentIndex = 0
	return m, m.fetchListingIfMissing()
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
	if m.providerIndex >= len(m.providers) {
		return api.Provider{}, false
	}
	return m.providers[m.providerIndex], true
}

func (m WatchPickerModel) currentListing() (componentListing, bool) {
	provider, ok := m.providerAtCursor()
	return componentListing{providerSlug: provider.Slug, query: m.query}, ok
}

// currentComponents are the cached components for the provider and search on screen; nil while
// they are still loading.
func (m WatchPickerModel) currentComponents() []api.Component {
	listing, ok := m.currentListing()
	if !ok {
		return nil
	}
	components, _ := m.cache.lookup(listing)
	return components
}

// fetchListingIfMissing fetches what the screen shows unless it is already cached. A listing that
// failed is fetched again when the user comes back to it.
func (m WatchPickerModel) fetchListingIfMissing() tea.Cmd {
	listing, ok := m.currentListing()
	if !ok {
		return nil
	}
	if _, cached := m.cache.lookup(listing); cached {
		return nil
	}
	m.cache.forgetFailure(listing)
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
