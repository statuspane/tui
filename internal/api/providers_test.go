package api

import (
	"context"
	"net/http"
	"testing"
)

func TestListProvidersDecodesTheCatalog(t *testing.T) {
	c, recorded := newRecordingClient(t, http.StatusOK, `[
		{"name":"GitHub","slug":"github","category":"Developer Tools","homepage_url":"https://github.com","status_page_url":"https://www.githubstatus.com","components_count":31}
	]`)

	providers, err := c.ListProviders(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	assertRequest(t, recorded, recordedRequest{method: http.MethodGet, path: "/api/v1/providers"})
	if len(providers) != 1 || providers[0].Slug != "github" || providers[0].ComponentsCount != 31 || providers[0].Category != "Developer Tools" {
		t.Fatalf("providers = %+v", providers)
	}
}

func TestListProviderComponentsDecodesGroupsAndStatuses(t *testing.T) {
	c, recorded := newRecordingClient(t, http.StatusOK, `[
		{"provider":"github","id":"a","name":"API Requests","group_name":"Core","status":"operational"},
		{"provider":"github","id":"b","name":"Pages","group_name":null,"status":null}
	]`)

	components, err := c.ListProviderComponents(context.Background(), "github", "")
	if err != nil {
		t.Fatal(err)
	}
	assertRequest(t, recorded, recordedRequest{method: http.MethodGet, path: "/api/v1/providers/github/components"})
	if len(components) != 2 || components[0].GroupName == nil || components[1].GroupName != nil || components[1].Status != nil {
		t.Fatalf("components = %+v", components)
	}
}

func TestListProviderComponentsSendsTheQueryEncoded(t *testing.T) {
	c, recorded := newRecordingClient(t, http.StatusOK, `[]`)

	if _, err := c.ListProviderComponents(context.Background(), "github", "a b&c"); err != nil {
		t.Fatal(err)
	}
	assertRequest(t, recorded, recordedRequest{method: http.MethodGet, path: "/api/v1/providers/github/components", query: "q=a+b%26c"})
}

func TestListProviderComponentsEscapesTheProviderSlug(t *testing.T) {
	c, recorded := newRecordingClient(t, http.StatusOK, `[]`)

	if _, err := c.ListProviderComponents(context.Background(), "git hub", ""); err != nil {
		t.Fatal(err)
	}
	assertRequest(t, recorded, recordedRequest{method: http.MethodGet, path: "/api/v1/providers/git%20hub/components"})
}
