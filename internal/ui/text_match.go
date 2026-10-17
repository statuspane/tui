package ui

import "strings"

// normalizeQuery trims and lower-cases what the user typed, so " Git " finds "GitHub".
func normalizeQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

// anyFieldContains reports whether one of the fields contains the normalized query. It looks at
// each field on its own, so a query never matches across the end of one field and the start of
// the next.
func anyFieldContains(query string, fields ...string) bool {
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}
