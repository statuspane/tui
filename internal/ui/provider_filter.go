package ui

import "github.com/statuspane/tui/internal/api"

// filterProviders keeps the providers whose name, slug or category contains the query, ignoring
// case and surrounding spaces. An empty query keeps them all, and then the result is the input
// slice itself: do not modify it.
func filterProviders(providers []api.Provider, query string) []api.Provider {
	query = normalizeQuery(query)
	if query == "" {
		return providers
	}
	var matches []api.Provider
	for _, provider := range providers {
		if anyFieldContains(query, provider.Name, provider.Slug, provider.Category) {
			matches = append(matches, provider)
		}
	}
	return matches
}
