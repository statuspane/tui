package ui

import "github.com/statuspane/tui/internal/api"

// filterComponents keeps the components whose name or group name contains the query, ignoring
// case and surrounding spaces: the rule the API applies to its own search, so a search can be
// answered from a list already in memory. An empty query keeps them all, and then the result is
// the input slice itself: do not modify it.
func filterComponents(components []api.Component, query string) []api.Component {
	query = normalizeQuery(query)
	if query == "" {
		return components
	}
	var matches []api.Component
	for _, component := range components {
		group := ""
		if component.GroupName != nil {
			group = *component.GroupName
		}
		if anyFieldContains(query, component.Name, group) {
			matches = append(matches, component)
		}
	}
	return matches
}
