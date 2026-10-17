package ui

import (
	"context"

	tea "charm.land/bubbletea/v2"

	"github.com/statuspane/tui/internal/api"
)

// ProviderCatalog is what the picker reads from the API client; tests pass a fake.
type ProviderCatalog interface {
	ListProviders(ctx context.Context) ([]api.Provider, error)
	ListProviderComponents(ctx context.Context, providerSlug, query string) ([]api.Component, error)
}

// WatchedComponentsSetter saves the picker's result; tests pass a fake.
type WatchedComponentsSetter interface {
	SetWatchedComponents(ctx context.Context, appSlug string, refs []api.ComponentRef) (api.AppDetail, error)
}

// WatchPickerBackend is everything the picker needs from the API client.
type WatchPickerBackend interface {
	ProviderCatalog
	WatchedComponentsSetter
}

func closeWatchPicker() tea.Msg { return closeWatchPickerMsg{} }

func fetchProviders(catalog ProviderCatalog) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		providers, err := catalog.ListProviders(ctx)
		if err != nil {
			return providersFailedMsg{err: err}
		}
		return providersLoadedMsg{providers: providers}
	}
}

func fetchProviderComponents(catalog ProviderCatalog, listing componentListing) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		components, err := catalog.ListProviderComponents(ctx, listing.providerSlug, listing.query)
		if err != nil {
			return providerComponentsFailedMsg{listing: listing, err: err}
		}
		return providerComponentsLoadedMsg{listing: listing, components: components}
	}
}

func saveWatchedComponents(setter WatchedComponentsSetter, appSlug string, refs []api.ComponentRef) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		defer cancel()

		appDetail, err := setter.SetWatchedComponents(ctx, appSlug, refs)
		if err != nil {
			return saveWatchedComponentsFailedMsg{err: err}
		}
		return watchedComponentsSavedMsg{appDetail: appDetail}
	}
}
