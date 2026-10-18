package ui

import (
	"fmt"
	"slices"
	"strings"

	"github.com/statuspane/tui/internal/api"
)

const summaryGap = "   "

// summaryOrder lists the statuses from calmest to worst; "pending" and any status this client
// doesn't know come after them.
var summaryOrder = []string{"operational", "under_maintenance", "degraded_performance", "partial_outage", "major_outage"}

type statusCount struct {
	appearance statusAppearance
	rank       int
	count      int
}

// statusCounts says how many apps are in each status, in summaryOrder.
func statusCounts(apps []api.App) []statusCount {
	var counts []statusCount
	for _, app := range apps {
		appearance := appearanceOf(app.Status)
		index := slices.IndexFunc(counts, func(c statusCount) bool { return c.appearance.label == appearance.label })
		if index < 0 {
			counts = append(counts, statusCount{appearance: appearance, rank: summaryRank(app.Status)})
			index = len(counts) - 1
		}
		counts[index].count++
	}
	slices.SortStableFunc(counts, func(a, b statusCount) int { return a.rank - b.rank })
	return counts
}

func summaryRank(status *string) int {
	if status == nil {
		return len(summaryOrder)
	}
	if index := slices.Index(summaryOrder, *status); index >= 0 {
		return index
	}
	return len(summaryOrder) + 1
}

// statusSummary is "3 operational   1 degraded", each count in its status colour.
func statusSummary(apps []api.App) string {
	counts := statusCounts(apps)
	parts := make([]string, len(counts))
	for index, count := range counts {
		parts[index] = count.appearance.style.Render(fmt.Sprintf("%d %s", count.count, strings.ToLower(count.appearance.label)))
	}
	return strings.Join(parts, summaryGap)
}
