package ui

import (
	"sort"
	"strings"

	"github.com/statuspane/tui/internal/api"
)

// ungroupedLabel heads the components that belong to no group, in a provider that has groups.
const ungroupedLabel = "Ungrouped"

func groupNameOf(component api.Component) string {
	if component.GroupName == nil {
		return ""
	}
	return *component.GroupName
}

// orderedByGroup returns the components with each group together: groups in alphabetical order
// ignoring case, components without a group last, and the order inside a group as given. The
// input is not modified.
func orderedByGroup(components []api.Component) []api.Component {
	ordered := append([]api.Component(nil), components...)
	sort.SliceStable(ordered, func(i, j int) bool {
		first, second := groupNameOf(ordered[i]), groupNameOf(ordered[j])
		switch {
		case first == second:
			return false
		case first == "":
			return false
		case second == "":
			return true
		}
		if folded := strings.Compare(strings.ToLower(first), strings.ToLower(second)); folded != 0 {
			return folded < 0
		}
		return first < second
	})
	return ordered
}

// groupBounds is the range [start, end) of the ordered components that share the group of the one
// at index. Groups are contiguous in an ordered list.
func groupBounds(ordered []api.Component, index int) (start, end int) {
	name := groupNameOf(ordered[index])
	start, end = index, index+1
	for start > 0 && groupNameOf(ordered[start-1]) == name {
		start--
	}
	for end < len(ordered) && groupNameOf(ordered[end]) == name {
		end++
	}
	return start, end
}

// componentLine is one line of the components pane: a group heading with its progress, or a
// component, by its index in the ordered list.
type componentLine struct {
	isHeading      bool
	groupName      string
	chosen         int
	total          int
	componentIndex int
}

// groupLines lays ordered components out under group headings. A provider none of whose
// components has a group gets no headings at all. isChosen says whether a component is chosen,
// for the progress shown on each heading.
func groupLines(ordered []api.Component, isChosen func(api.Component) bool) []componentLine {
	hasGroups := false
	for _, component := range ordered {
		if groupNameOf(component) != "" {
			hasGroups = true
			break
		}
	}

	lines := make([]componentLine, 0, len(ordered))
	previousGroup := ""
	for index, component := range ordered {
		group := groupNameOf(component)
		if hasGroups && (index == 0 || group != previousGroup) {
			lines = append(lines, headingFor(ordered, index, isChosen))
		}
		lines = append(lines, componentLine{componentIndex: index})
		previousGroup = group
	}
	return lines
}

func headingFor(ordered []api.Component, index int, isChosen func(api.Component) bool) componentLine {
	start, end := groupBounds(ordered, index)
	chosen := 0
	for _, member := range ordered[start:end] {
		if isChosen(member) {
			chosen++
		}
	}
	name := groupNameOf(ordered[index])
	if name == "" {
		name = ungroupedLabel
	}
	return componentLine{isHeading: true, groupName: name, chosen: chosen, total: end - start}
}
