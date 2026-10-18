package ui

import (
	"cmp"
	"slices"
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
	ordered := slices.Clone(components)
	slices.SortStableFunc(ordered, func(a, b api.Component) int {
		groupA, groupB := groupNameOf(a), groupNameOf(b)
		return cmp.Or(
			cmp.Compare(ungroupedRank(groupA), ungroupedRank(groupB)),
			strings.Compare(strings.ToLower(groupA), strings.ToLower(groupB)),
			strings.Compare(groupA, groupB),
		)
	})
	return ordered
}

// ungroupedRank sorts components without a group after the ones with.
func ungroupedRank(group string) int {
	if group == "" {
		return 1
	}
	return 0
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

// paneLine is one line of the components pane: a group heading with its progress, or a
// component, by its index in the ordered list.
type paneLine struct {
	isHeading bool
	groupName string
	watched   int // components of the group that are watched or about to be
	total     int
	index     int
}

// linesWithHeadings lays ordered components out under group headings. A provider none of whose
// components has a group gets no headings at all.
func linesWithHeadings(ordered []api.Component, selection watchSelection) []paneLine {
	hasGroups := slices.ContainsFunc(ordered, func(component api.Component) bool { return groupNameOf(component) != "" })

	lines := make([]paneLine, 0, len(ordered))
	for start := 0; start < len(ordered); {
		_, end := groupBounds(ordered, start)
		if hasGroups {
			lines = append(lines, headingFor(ordered[start:end], selection))
		}
		for index := start; index < end; index++ {
			lines = append(lines, paneLine{index: index})
		}
		start = end
	}
	return lines
}

// headingFor is the heading of a group, given all of its members.
func headingFor(members []api.Component, selection watchSelection) paneLine {
	name := groupNameOf(members[0])
	if name == "" {
		name = ungroupedLabel
	}
	return paneLine{isHeading: true, groupName: name, watched: selection.chosenAmong(members), total: len(members)}
}

// lineOfComponent is the position among the lines of the component at index; found is false when
// there is no such component.
func lineOfComponent(lines []paneLine, index int) (position int, found bool) {
	for position, line := range lines {
		if !line.isHeading && line.index == index {
			return position, true
		}
	}
	return 0, false
}
