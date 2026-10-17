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

func TestGroupLinesPutAHeadingWithProgressBeforeEachGroup(t *testing.T) {
	ordered := orderedByGroup([]api.Component{inGroup("a1", "A"), inGroup("a2", "A"), inGroup("loose", "")})
	chosen := func(component api.Component) bool { return component.ID == "a2" }

	lines := groupLines(ordered, chosen)

	want := []componentLine{
		{isHeading: true, groupName: "A", chosen: 1, total: 2},
		{componentIndex: 0}, {componentIndex: 1},
		{isHeading: true, groupName: ungroupedLabel, chosen: 0, total: 1},
		{componentIndex: 2},
	}
	if !reflect.DeepEqual(lines, want) {
		t.Fatalf("got %+v\nwant %+v", lines, want)
	}
}

func TestGroupLinesHaveNoHeadingsWhenNothingHasAGroup(t *testing.T) {
	ordered := []api.Component{inGroup("x", ""), inGroup("y", "")}

	lines := groupLines(ordered, func(api.Component) bool { return false })

	if !reflect.DeepEqual(lines, []componentLine{{componentIndex: 0}, {componentIndex: 1}}) {
		t.Fatalf("got %+v", lines)
	}
}

func TestGroupLinesOfNothingAreEmpty(t *testing.T) {
	if lines := groupLines(nil, func(api.Component) bool { return false }); len(lines) != 0 {
		t.Fatalf("got %+v", lines)
	}
}
