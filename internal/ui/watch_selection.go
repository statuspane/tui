package ui

import (
	"sort"

	"github.com/statuspane/tui/internal/api"
)

type componentKey struct {
	provider string
	id       string
}

func keyOf(component api.Component) componentKey {
	return componentKey{provider: component.Provider, id: component.ID}
}

// watchState says how a component stands against what the app watches now.
type watchState int

const (
	stateUnwatched watchState = iota
	stateWatched
	stateAdded
	stateRemoved
)

// watchSelection is the set of components an app watches now (baseline) next to the set the user
// has chosen so far. The API replaces the whole set on save, so chosen has to hold every watched
// component, including those of providers whose components were never shown.
type watchSelection struct {
	baseline map[componentKey]bool
	chosen   map[componentKey]bool
}

func newWatchSelection(watched []api.Component) watchSelection {
	baseline := make(map[componentKey]bool, len(watched))
	chosen := make(map[componentKey]bool, len(watched))
	for _, component := range watched {
		baseline[keyOf(component)] = true
		chosen[keyOf(component)] = true
	}
	return watchSelection{baseline: baseline, chosen: chosen}
}

// toggled returns the selection with the component's choice flipped; s is left as it was.
func (s watchSelection) toggled(key componentKey) watchSelection {
	chosen := make(map[componentKey]bool, len(s.chosen)+1)
	for chosenKey := range s.chosen {
		chosen[chosenKey] = true
	}
	if chosen[key] {
		delete(chosen, key)
	} else {
		chosen[key] = true
	}
	return watchSelection{baseline: s.baseline, chosen: chosen}
}

func (s watchSelection) isChosen(key componentKey) bool {
	return s.chosen[key]
}

// withChosen returns the selection with every one of the components chosen or not chosen; s is
// left as it was.
func (s watchSelection) withChosen(keys []componentKey, chosen bool) watchSelection {
	next := make(map[componentKey]bool, len(s.chosen)+len(keys))
	for key := range s.chosen {
		next[key] = true
	}
	for _, key := range keys {
		if chosen {
			next[key] = true
		} else {
			delete(next, key)
		}
	}
	return watchSelection{baseline: s.baseline, chosen: next}
}

// chosenAmong counts the components that are chosen.
func (s watchSelection) chosenAmong(components []api.Component) int {
	count := 0
	for _, component := range components {
		if s.chosen[keyOf(component)] {
			count++
		}
	}
	return count
}

// toggledAll chooses the components that are not chosen yet or, when they all are, unchooses them
// all; s is left as it was.
func (s watchSelection) toggledAll(components []api.Component) watchSelection {
	keys := make([]componentKey, 0, len(components))
	for _, component := range components {
		keys = append(keys, keyOf(component))
	}
	return s.withChosen(keys, s.chosenAmong(components) < len(components))
}

func (s watchSelection) stateOf(key componentKey) watchState {
	wasWatched, isChosen := s.baseline[key], s.chosen[key]
	switch {
	case wasWatched && isChosen:
		return stateWatched
	case isChosen:
		return stateAdded
	case wasWatched:
		return stateRemoved
	}
	return stateUnwatched
}

func (s watchSelection) changeCount() int {
	changes := 0
	for key := range s.chosen {
		if !s.baseline[key] {
			changes++
		}
	}
	for key := range s.baseline {
		if !s.chosen[key] {
			changes++
		}
	}
	return changes
}

// chosenCount is how many components are chosen, across every provider.
func (s watchSelection) chosenCount() int {
	return len(s.chosen)
}

func (s watchSelection) chosenCountFor(providerSlug string) int {
	count := 0
	for key := range s.chosen {
		if key.provider == providerSlug {
			count++
		}
	}
	return count
}

// componentRefs lists every chosen component, ordered by provider then id so a save is deterministic.
// It is never nil: an empty selection must reach the API as [], not null.
func (s watchSelection) componentRefs() []api.ComponentRef {
	refs := make([]api.ComponentRef, 0, len(s.chosen))
	for key := range s.chosen {
		refs = append(refs, api.ComponentRef{Provider: key.provider, ID: key.id})
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Provider != refs[j].Provider {
			return refs[i].Provider < refs[j].Provider
		}
		return refs[i].ID < refs[j].ID
	})
	return refs
}
