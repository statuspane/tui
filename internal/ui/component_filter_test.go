package ui

import (
	"reflect"
	"testing"

	"github.com/statuspane/tui/internal/api"
)

func TestFilterComponentsMatchesNameAndGroupIgnoringCase(t *testing.T) {
	core := stringPtr("Core Services")
	components := []api.Component{
		namedComponent("github", "api", "API Requests", core),
		namedComponent("github", "pages", "Pages", nil),
		namedComponent("github", "git", "Git Operations", core),
	}
	cases := []struct {
		name  string
		query string
		want  []string
	}{
		{"empty query keeps all", "", []string{"API Requests", "Pages", "Git Operations"}},
		{"by name, any case", "PAGES", []string{"Pages"}},
		{"by group", "core serv", []string{"API Requests", "Git Operations"}},
		{"inside a word", "equest", []string{"API Requests"}},
		{"ignores surrounding spaces", " pages ", []string{"Pages"}},
		{"no match", "zzz", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, component := range filterComponents(components, tc.query) {
				got = append(got, component.Name)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFilterComponentsDoesNotMatchAcrossTheNameAndGroup(t *testing.T) {
	components := []api.Component{namedComponent("github", "a", "Alpha", stringPtr("Beta"))}

	if got := filterComponents(components, "alphabeta"); len(got) != 0 {
		t.Fatalf("got %v: a query must match inside one field", got)
	}
}
