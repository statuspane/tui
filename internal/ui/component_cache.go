package ui

import (
	"time"

	"github.com/statuspane/tui/internal/api"
)

// cacheLifetime is how long an answer is trusted. Provider catalogs and component lists change
// rarely, but the statuses shown next to components do, so nothing is kept forever.
const cacheLifetime = 5 * time.Minute

type cachedComponents struct {
	components []api.Component
	storedAt   time.Time
}

// componentCache remembers what the API answered, by provider, so looking at a provider again
// costs no request.
// It also remembers which requests are still in flight and which failed, so the picker neither
// repeats a request that is already on its way nor hammers one that was refused. The root model
// owns it and every picker opened during the run shares it, which is why it is a pointer with
// named methods instead of maps edited in place.
type componentCache struct {
	now      func() time.Time
	answers  map[string]cachedComponents
	failures map[string]error
	pending  map[string]bool

	providers         []api.Provider
	providersStoredAt time.Time
	hasProviders      bool
}

func newComponentCache(now func() time.Time) *componentCache {
	return &componentCache{
		now:      now,
		answers:  map[string]cachedComponents{},
		failures: map[string]error{},
		pending:  map[string]bool{},
	}
}

func (c *componentCache) isFresh(storedAt time.Time) bool {
	return c.now().Sub(storedAt) < cacheLifetime
}

// markPending notes that a request for the provider's components is on its way.
func (c *componentCache) markPending(providerSlug string) {
	c.pending[providerSlug] = true
}

func (c *componentCache) isPending(providerSlug string) bool {
	return c.pending[providerSlug]
}

func (c *componentCache) store(providerSlug string, components []api.Component) {
	c.answers[providerSlug] = cachedComponents{components: components, storedAt: c.now()}
	delete(c.failures, providerSlug)
	delete(c.pending, providerSlug)
}

func (c *componentCache) fail(providerSlug string, err error) {
	c.failures[providerSlug] = err
	delete(c.pending, providerSlug)
}

// forgetFailure lets a provider whose request failed be asked for again.
func (c *componentCache) forgetFailure(providerSlug string) {
	delete(c.failures, providerSlug)
}

// lookup returns the provider's components unless they are missing or older than cacheLifetime.
func (c *componentCache) lookup(providerSlug string) ([]api.Component, bool) {
	cached, ok := c.answers[providerSlug]
	if !ok || !c.isFresh(cached.storedAt) {
		return nil, false
	}
	return cached.components, true
}

// failure is the error the last request for the provider's components ended with, or nil.
func (c *componentCache) failure(providerSlug string) error {
	return c.failures[providerSlug]
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
