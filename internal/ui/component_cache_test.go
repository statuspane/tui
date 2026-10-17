package ui

import (
	"errors"
	"testing"
	"time"

	"github.com/statuspane/tui/internal/api"
)

type fakeClock struct{ current time.Time }

func (c *fakeClock) now() time.Time { return c.current }

func (c *fakeClock) advance(duration time.Duration) { c.current = c.current.Add(duration) }

var github = componentListing{providerSlug: "github"}

func TestACachedAnswerIsUsedUntilItIsOlderThanTheCacheLifetime(t *testing.T) {
	clock := &fakeClock{current: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)}
	cache := newComponentCache(clock.now)
	cache.store(github, []api.Component{component("github", "api")})

	clock.advance(cacheLifetime - time.Second)
	if _, ok := cache.lookup(github); !ok {
		t.Fatal("still fresh")
	}
	clock.advance(2 * time.Second)

	if _, ok := cache.lookup(github); ok {
		t.Fatal("an answer older than the cache lifetime must be asked for again")
	}
}

func TestCachedProvidersExpireLikeAnswersDo(t *testing.T) {
	clock := &fakeClock{current: time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)}
	cache := newComponentCache(clock.now)
	if _, ok := cache.cachedProviders(); ok {
		t.Fatal("nothing stored yet")
	}
	cache.storeProviders([]api.Provider{{Slug: "github"}})

	if providers, ok := cache.cachedProviders(); !ok || len(providers) != 1 {
		t.Fatal("want the stored providers")
	}
	clock.advance(cacheLifetime + time.Second)

	if _, ok := cache.cachedProviders(); ok {
		t.Fatal("the providers must expire")
	}
}

func TestAnAnswerOrAFailureEndsPending(t *testing.T) {
	cache := newTestCache()
	other := componentListing{providerSlug: "cloudflare"}
	cache.markPending(github)
	cache.markPending(other)

	cache.store(github, nil)
	cache.fail(other, errors.New("refused"))

	if cache.isPending(github) || cache.isPending(other) {
		t.Fatal("a listing that was answered is no longer on its way")
	}
}

func TestAnAnswerClearsAnEarlierFailure(t *testing.T) {
	cache := newTestCache()
	cache.fail(github, errors.New("refused"))

	cache.store(github, []api.Component{component("github", "api")})

	if cache.failure(github) != nil {
		t.Fatal("the failure is out of date")
	}
}

func TestPendingIsPerProvider(t *testing.T) {
	cache := newTestCache()

	cache.markPending(github)

	if cache.isPending(componentListing{providerSlug: "cloudflare"}) {
		t.Fatal("another provider is another listing")
	}
}
