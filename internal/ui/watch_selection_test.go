package ui

import (
	"reflect"
	"testing"

	"github.com/statuspane/tui/internal/api"
)

var (
	githubAPI   = componentKey{"github", "api"}
	githubPages = componentKey{"github", "pages"}
)

func TestAFreshSelectionHasNoChanges(t *testing.T) {
	selection := newWatchSelection([]api.Component{component("github", "api")})

	if selection.changeCount() != 0 || selection.stateOf(githubAPI) != stateWatched || selection.stateOf(githubPages) != stateUnwatched {
		t.Fatalf("selection = %+v", selection)
	}
}

func TestTogglingMarksAdditionsAndRemovals(t *testing.T) {
	selection := newWatchSelection([]api.Component{component("github", "api")})

	selection = selection.toggled(githubPages).toggled(githubAPI)

	if selection.stateOf(githubPages) != stateAdded || selection.stateOf(githubAPI) != stateRemoved || selection.changeCount() != 2 {
		t.Fatalf("selection = %+v", selection)
	}
}

func TestTogglingTwiceReturnsToUnchanged(t *testing.T) {
	selection := newWatchSelection([]api.Component{component("github", "api")})

	selection = selection.toggled(githubAPI).toggled(githubAPI).toggled(githubPages).toggled(githubPages)

	if selection.changeCount() != 0 || selection.stateOf(githubAPI) != stateWatched || selection.stateOf(githubPages) != stateUnwatched {
		t.Fatalf("selection = %+v", selection)
	}
}

func TestTogglingLeavesTheOriginalSelectionAlone(t *testing.T) {
	original := newWatchSelection([]api.Component{component("github", "api")})

	original.toggled(githubPages)

	if original.changeCount() != 0 {
		t.Fatal("toggled must return a new selection")
	}
}

func TestComponentRefsHoldEveryChosenComponentOrderedByProviderThenID(t *testing.T) {
	selection := newWatchSelection([]api.Component{component("github", "pages"), component("cloudflare", "cdn")})

	selection = selection.toggled(githubAPI).toggled(componentKey{"github", "pages"})

	want := []api.ComponentRef{{Provider: "cloudflare", ID: "cdn"}, {Provider: "github", ID: "api"}}
	if got := selection.componentRefs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("refs = %+v, want %+v", got, want)
	}
}

func TestComponentRefsOfAnEmptySelectionAreAnEmptyListNotNil(t *testing.T) {
	selection := newWatchSelection(nil)

	if refs := selection.componentRefs(); refs == nil || len(refs) != 0 {
		t.Fatalf("refs = %#v", refs)
	}
}

func TestChosenCountIsPerProvider(t *testing.T) {
	selection := newWatchSelection([]api.Component{component("github", "api"), component("github", "pages"), component("cloudflare", "cdn")})

	selection = selection.toggled(githubPages)

	if selection.chosenCountFor("github") != 1 || selection.chosenCountFor("cloudflare") != 1 || selection.chosenCountFor("openai") != 0 {
		t.Fatalf("selection = %+v", selection)
	}
}

func TestWithChosenChoosesOrUnchoosesSeveralAtOnce(t *testing.T) {
	selection := newWatchSelection([]api.Component{component("github", "api")})

	selection = selection.withChosen([]componentKey{githubPages, {"github", "git"}}, true)
	if selection.stateOf(githubPages) != stateAdded || selection.stateOf(componentKey{"github", "git"}) != stateAdded || selection.changeCount() != 2 {
		t.Fatalf("selection = %+v", selection)
	}
	selection = selection.withChosen([]componentKey{githubAPI, githubPages}, false)

	if selection.stateOf(githubAPI) != stateRemoved || selection.stateOf(githubPages) != stateUnwatched || !selection.isChosen(componentKey{"github", "git"}) {
		t.Fatalf("selection = %+v", selection)
	}
}

func TestWithChosenLeavesTheOriginalSelectionAlone(t *testing.T) {
	original := newWatchSelection([]api.Component{component("github", "api")})

	original.withChosen([]componentKey{githubAPI}, false)
	original.withChosen([]componentKey{githubPages}, true)

	if original.changeCount() != 0 {
		t.Fatal("withChosen must return a new selection")
	}
}

func TestToggledAllChoosesTheRestAndThenUnchoosesEverything(t *testing.T) {
	api1, git := component("github", "api"), component("github", "git")
	selection := newWatchSelection([]api.Component{api1})

	all := selection.toggledAll([]api.Component{api1, git})
	if all.chosenAmong([]api.Component{api1, git}) != 2 || all.stateOf(keyOf(git)) != stateAdded {
		t.Fatalf("not every component was chosen: %+v", all.chosen)
	}
	none := all.toggledAll([]api.Component{api1, git})

	if none.chosenAmong([]api.Component{api1, git}) != 0 || none.stateOf(keyOf(api1)) != stateRemoved {
		t.Fatalf("not every component was unchosen: %+v", none.chosen)
	}
	if selection.isChosen(keyOf(git)) {
		t.Fatal("toggledAll must return a new selection")
	}
}
