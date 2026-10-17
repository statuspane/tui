package ui

import (
	"reflect"
	"testing"

	"github.com/statuspane/tui/internal/api"
)

func TestFilterProvidersMatchesNameSlugAndCategoryIgnoringCase(t *testing.T) {
	providers := []api.Provider{
		{Name: "GitHub", Slug: "github", Category: "Developer Tools"},
		{Name: "Cloudflare", Slug: "cloudflare", Category: "Cloud Infrastructure"},
		{Name: "Google Cloud", Slug: "gcp", Category: "Cloud Infrastructure"},
	}
	cases := []struct {
		name  string
		query string
		want  []string
	}{
		{"empty query keeps all", "", []string{"GitHub", "Cloudflare", "Google Cloud"}},
		{"by name, any case", "CLOUDF", []string{"Cloudflare"}},
		{"by slug", "gcp", []string{"Google Cloud"}},
		{"by category", "infrastructure", []string{"Cloudflare", "Google Cloud"}},
		{"matches inside a word", "hub", []string{"GitHub"}},
		{"no match", "zzz", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, provider := range filterProviders(providers, tc.query) {
				got = append(got, provider.Name)
			}

			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestFilterProvidersDoesNotMatchAcrossFields(t *testing.T) {
	providers := []api.Provider{{Name: "Alpha", Slug: "alpha", Category: "Beta"}}

	if got := filterProviders(providers, "alphabeta"); len(got) != 0 {
		t.Fatalf("got %v: a query must match inside one field, not across the join", got)
	}
}
