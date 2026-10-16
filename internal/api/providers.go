package api

import (
	"context"
	"net/http"
	"net/url"
)

const (
	providersPath    = "/providers"
	searchQueryParam = "q"
)

// Provider is a service StatusPane can watch (GitHub, Cloudflare, ...); the catalog is the
// same for every organization.
type Provider struct {
	Name            string `json:"name"`
	Slug            string `json:"slug"`
	Category        string `json:"category"`
	HomepageURL     string `json:"homepage_url"`
	StatusPageURL   string `json:"status_page_url"`
	ComponentsCount int    `json:"components_count"`
}

func (c *Client) ListProviders(ctx context.Context) ([]Provider, error) {
	var providers []Provider
	if err := c.do(ctx, http.MethodGet, providersPath, nil, &providers); err != nil {
		return nil, err
	}
	return providers, nil
}

// ListProviderComponents lists a provider's components; a non-empty query keeps only those
// whose name or group name contains it, ignoring case. The API cannot search across providers.
func (c *Client) ListProviderComponents(ctx context.Context, providerSlug, query string) ([]Component, error) {
	var components []Component
	if err := c.do(ctx, http.MethodGet, providerComponentsPath(providerSlug, query), nil, &components); err != nil {
		return nil, err
	}
	return components, nil
}

func providerComponentsPath(providerSlug, query string) string {
	path := providersPath + "/" + url.PathEscape(providerSlug) + "/components"
	if query == "" {
		return path
	}
	return path + "?" + url.Values{searchQueryParam: {query}}.Encode()
}
