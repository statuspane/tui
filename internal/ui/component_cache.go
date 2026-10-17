package ui

import "github.com/statuspane/tui/internal/api"

// componentListing names one list of components: a provider's, optionally narrowed by a search.
type componentListing struct {
	providerSlug string
	query        string
}

// componentCache remembers what the API answered for each listing while the picker is open, so
// moving back to a provider costs no request. Every copy of the picker model shares one cache,
// which is why it is a pointer with named methods instead of maps edited in place.
type componentCache struct {
	answers  map[componentListing][]api.Component
	failures map[componentListing]error
}

func newComponentCache() *componentCache {
	return &componentCache{
		answers:  map[componentListing][]api.Component{},
		failures: map[componentListing]error{},
	}
}

func (c *componentCache) store(listing componentListing, components []api.Component) {
	c.answers[listing] = components
	delete(c.failures, listing)
}

func (c *componentCache) fail(listing componentListing, err error) {
	c.failures[listing] = err
}

// forgetFailure lets a listing that failed be fetched again.
func (c *componentCache) forgetFailure(listing componentListing) {
	delete(c.failures, listing)
}

func (c *componentCache) lookup(listing componentListing) ([]api.Component, bool) {
	components, ok := c.answers[listing]
	return components, ok
}

// failure is the error the last fetch of the listing ended with, or nil.
func (c *componentCache) failure(listing componentListing) error {
	return c.failures[listing]
}
