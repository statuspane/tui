package ui

import (
	"strings"

	"github.com/statuspane/tui/internal/api"
)

// filterComponents keeps the components whose name or group name contains the query, ignoring
// case: the rule the API applies to its own search, so a search needs no request once the
// provider's list is in memory. An empty query keeps them all.
func filterComponents(components []api.Component, query string) []api.Component {
	query = strings.ToLower(query)
	if query == "" {
		return components
	}
	var matches []api.Component
	for _, component := range components {
		haystack := component.Name
		if component.GroupName != nil {
			haystack += "\n" + *component.GroupName
		}
		if strings.Contains(strings.ToLower(haystack), query) {
			matches = append(matches, component)
		}
	}
	return matches
}
