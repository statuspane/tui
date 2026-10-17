package ui

import "github.com/statuspane/tui/internal/api"

// openWatchPickerMsg asks the root model to show the picker for one app, which currently watches
// the given components.
type openWatchPickerMsg struct {
	appSlug           string
	appName           string
	watchedComponents []api.Component
}

type closeWatchPickerMsg struct{}

// watchedComponentsSavedMsg carries the app as the API now has it; the root model closes the
// picker and gives the app screen the new state.
type watchedComponentsSavedMsg struct{ appDetail api.AppDetail }

type providersLoadedMsg struct{ providers []api.Provider }

type providersFailedMsg struct{ err error }

// providerSettledMsg says the cursor stayed on a provider for providerSettleDelay. The sequence
// tells a settle from an older cursor position apart from the current one.
type providerSettledMsg struct{ sequence int }

// The responses name their provider, so one that arrives after the user moved on lands in the
// cache without disturbing what is on screen.
type providerComponentsLoadedMsg struct {
	providerSlug string
	components   []api.Component
}

type providerComponentsFailedMsg struct {
	providerSlug string
	err          error
}

type saveWatchedComponentsFailedMsg struct{ err error }
