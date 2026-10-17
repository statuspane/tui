package ui

import (
	"strings"

	"github.com/statuspane/tui/internal/api"
)

// filterProviders keeps the providers whose name, slug or category contains the query, ignoring
// case. An empty query keeps them all.
func filterProviders(providers []api.Provider, query string) []api.Provider {
	query = strings.ToLower(query)
	if query == "" {
		return providers
	}
	var matches []api.Provider
	for _, provider := range providers {
		haystack := strings.ToLower(provider.Name + "\n" + provider.Slug + "\n" + provider.Category)
		if strings.Contains(haystack, query) {
			matches = append(matches, provider)
		}
	}
	return matches
}
