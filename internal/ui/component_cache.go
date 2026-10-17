package ui

import (
	"time"

	"github.com/statuspane/tui/internal/api"
)

// cacheLifetime is how long an answer is trusted. Provider catalogs and component lists change
// rarely, but the statuses shown next to components do, so nothing is kept forever.
const cacheLifetime = 5 * time.Minute

// componentListing names one list of components: a provider's, optionally narrowed by a search.
type componentListing struct {
	providerSlug string
	query        string
}

type cachedComponents struct {
	components []api.Component
	storedAt   time.Time
}

// componentCache remembers what the API answered, so looking at a provider again costs no request.
// It also remembers which requests are still in flight and which failed, so the picker neither
// repeats a request that is already on its way nor hammers one that was refused. The root model
// owns it and every picker opened during the run shares it, which is why it is a pointer with
// named methods instead of maps edited in place.
type componentCache struct {
	now      func() time.Time
	answers  map[componentListing]cachedComponents
	failures map[componentListing]error
	pending  map[componentListing]bool

	providers         []api.Provider
	providersStoredAt time.Time
	hasProviders      bool
}

func newComponentCache(now func() time.Time) *componentCache {
	return &componentCache{
		now:      now,
		answers:  map[componentListing]cachedComponents{},
		failures: map[componentListing]error{},
		pending:  map[componentListing]bool{},
	}
}

func (c *componentCache) isFresh(storedAt time.Time) bool {
	return c.now().Sub(storedAt) < cacheLifetime
}

// markPending notes that a request for the listing is on its way.
func (c *componentCache) markPending(listing componentListing) {
	c.pending[listing] = true
}

func (c *componentCache) isPending(listing componentListing) bool {
	return c.pending[listing]
}

func (c *componentCache) store(listing componentListing, components []api.Component) {
	c.answers[listing] = cachedComponents{components: components, storedAt: c.now()}
	delete(c.failures, listing)
	delete(c.pending, listing)
}

func (c *componentCache) fail(listing componentListing, err error) {
	c.failures[listing] = err
	delete(c.pending, listing)
}

// forgetFailure lets a listing that failed be asked for again.
func (c *componentCache) forgetFailure(listing componentListing) {
	delete(c.failures, listing)
}

// lookup returns the answer for the listing unless it is missing or older than cacheLifetime.
func (c *componentCache) lookup(listing componentListing) ([]api.Component, bool) {
	cached, ok := c.answers[listing]
	if !ok || !c.isFresh(cached.storedAt) {
		return nil, false
	}
	return cached.components, true
}

// failure is the error the last request for the listing ended with, or nil.
func (c *componentCache) failure(listing componentListing) error {
	return c.failures[listing]
}

func (c *componentCache) storeProviders(providers []api.Provider) {
	c.providers, c.providersStoredAt, c.hasProviders = providers, c.now(), true
}

func (c *componentCache) cachedProviders() ([]api.Provider, bool) {
	if !c.hasProviders || !c.isFresh(c.providersStoredAt) {
		return nil, false
	}
	return c.providers, true
}
