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

type statusTally struct {
	appearance statusAppearance
	rank       int
	appCount   int
}

// statusTallies says how many apps are in each status, in summaryOrder.
func statusTallies(apps []api.App) []statusTally {
	var tallies []statusTally
	for _, app := range apps {
		appearance := appearanceOf(app.Status)
		index := slices.IndexFunc(tallies, func(existing statusTally) bool { return existing.appearance.label == appearance.label })
		if index < 0 {
			tallies = append(tallies, statusTally{appearance: appearance, rank: summaryRank(app.Status)})
			index = len(tallies) - 1
		}
		tallies[index].appCount++
	}
	slices.SortStableFunc(tallies, func(left, right statusTally) int { return left.rank - right.rank })
	return tallies
}

// summaryRank puts the statuses in summaryOrder first, then pending, then any this client doesn't know.
func summaryRank(status *string) int {
	pendingRank := len(summaryOrder)
	unknownRank := pendingRank + 1
	if status == nil {
		return pendingRank
	}
	if index := slices.Index(summaryOrder, *status); index >= 0 {
		return index
	}
	return unknownRank
}

// statusSummary is "3 operational   1 degraded", each count in its status colour.
func statusSummary(apps []api.App) string {
	var parts []string
	for _, tally := range statusTallies(apps) {
		parts = append(parts, tally.appearance.style.Render(fmt.Sprintf("%d %s", tally.appCount, strings.ToLower(tally.appearance.label))))
	}
	return strings.Join(parts, summaryGap)
}
