package ui

import (
	"reflect"
	"testing"

	"github.com/statuspane/tui/internal/api"
)

func inGroup(id, group string) api.Component {
	if group == "" {
		return namedComponent("github", id, id, nil)
	}
	return namedComponent("github", id, id, stringPtr(group))
}

func ids(components []api.Component) []string {
	var got []string
	for _, component := range components {
		got = append(got, component.ID)
	}
	return got
}

func TestOrderedByGroupSortsGroupsIgnoringCaseWithUngroupedLastAndKeepsTheOrderInsideAGroup(t *testing.T) {
	components := []api.Component{
		inGroup("a1", "zebra"), inGroup("loose1", ""), inGroup("b1", "Apple"), inGroup("a2", "zebra"), inGroup("b2", "Apple"), inGroup("loose2", ""),
	}

	got := ids(orderedByGroup(components))

	want := []string{"b1", "b2", "a1", "a2", "loose1", "loose2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestOrderedByGroupKeepsGroupsThatDifferOnlyByCaseApartAndStable(t *testing.T) {
	components := []api.Component{inGroup("lower1", "core"), inGroup("upper1", "Core"), inGroup("lower2", "core")}

	got := ids(orderedByGroup(components))

	if !reflect.DeepEqual(got, []string{"upper1", "lower1", "lower2"}) {
		t.Fatalf("got %v", got)
	}
}

func TestOrderedByGroupTreatsAnEmptyGroupNameAsNoGroup(t *testing.T) {
	components := []api.Component{inGroup("empty", ""), namedComponent("github", "blank", "blank", stringPtr("")), inGroup("grouped", "Core")}

	got := ids(orderedByGroup(components))

	if !reflect.DeepEqual(got, []string{"grouped", "empty", "blank"}) {
		t.Fatalf("got %v", got)
	}
}

func TestOrderedByGroupDoesNotChangeItsInput(t *testing.T) {
	components := []api.Component{inGroup("b", "B"), inGroup("a", "A")}

	orderedByGroup(components)

	if !reflect.DeepEqual(ids(components), []string{"b", "a"}) {
		t.Fatal("the cached list must keep its order")
	}
}

func TestGroupBoundsIsTheRunOfTheSameGroup(t *testing.T) {
	ordered := orderedByGroup([]api.Component{inGroup("a1", "A"), inGroup("a2", "A"), inGroup("b1", "B"), inGroup("loose", "")})

	cases := []struct {
		index, start, end int
	}{{0, 0, 2}, {1, 0, 2}, {2, 2, 3}, {3, 3, 4}}
	for _, tc := range cases {
		if start, end := groupBounds(ordered, tc.index); start != tc.start || end != tc.end {
			t.Errorf("index %d: got %d-%d, want %d-%d", tc.index, start, end, tc.start, tc.end)
		}
	}
}

func TestLinesWithHeadingsPutAHeadingWithProgressBeforeEachGroup(t *testing.T) {
	ordered := orderedByGroup([]api.Component{inGroup("a1", "A"), inGroup("a2", "A"), inGroup("loose", "")})
	selection := newWatchSelection([]api.Component{inGroup("a2", "A")})

	lines := linesWithHeadings(ordered, selection)

	want := []paneLine{
		{isHeading: true, groupName: "A", watched: 1, total: 2},
		{index: 0}, {index: 1},
		{isHeading: true, groupName: ungroupedLabel, watched: 0, total: 1},
		{index: 2},
	}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("got %+v\nwant %+v", lines, want)
	}
}

func TestLinesWithHeadingsStartWithAnUngroupedHeadingWhenOnlySomeHaveAGroup(t *testing.T) {
	ordered := []api.Component{inGroup("loose", ""), inGroup("a1", "A")}

	lines := linesWithHeadings(ordered, newWatchSelection(nil))

	if len(lines) != 4 || !lines[0].isHeading || lines[0].groupName != ungroupedLabel || !lines[2].isHeading || lines[2].groupName != "A" {
		t.Fatalf("got %+v", lines)
	}
}

func TestLinesWithHeadingsHaveNoHeadingsWhenNothingHasAGroup(t *testing.T) {
	ordered := []api.Component{inGroup("x", ""), inGroup("y", "")}

	lines := linesWithHeadings(ordered, newWatchSelection(nil))

	if !reflect.DeepEqual(lines, []paneLine{{index: 0}, {index: 1}}) {
		t.Fatalf("got %+v", lines)
	}
}

func TestLinesWithHeadingsOfNothingAreEmpty(t *testing.T) {
	if lines := linesWithHeadings(nil, newWatchSelection(nil)); len(lines) != 0 {
		t.Fatalf("got %+v", lines)
	}
}

func TestLineOfComponentSkipsHeadingsAndSaysWhenThereIsNoSuchComponent(t *testing.T) {
	lines := linesWithHeadings(orderedByGroup([]api.Component{inGroup("a1", "A"), inGroup("b1", "B")}), newWatchSelection(nil))

	if position, found := lineOfComponent(lines, 1); !found || position != 3 {
		t.Errorf("got %d, %v, want 3, true", position, found)
	}
	if _, found := lineOfComponent(lines, 5); found {
		t.Error("a component that does not exist was found")
	}
}
