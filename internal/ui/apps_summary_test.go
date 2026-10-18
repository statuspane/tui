package ui

import (
	"strings"
	"testing"

	"github.com/statuspane/tui/internal/api"
)

func TestStatusTalliesCountTheAppsInEachStatus(t *testing.T) {
	tallies := statusTallies([]api.App{
		{Status: stringPtr("operational")}, {Status: stringPtr("operational")}, {Status: stringPtr("major_outage")},
	})

	if len(tallies) != 2 || tallies[0].appCount != 2 || tallies[1].appCount != 1 {
		t.Fatalf("tallies = %+v", tallies)
	}
}

func TestStatusTalliesListTheCalmestStatusFirstAndTheUnknownLast(t *testing.T) {
	tallies := statusTallies([]api.App{
		{Status: stringPtr("major_outage")},
		{Status: stringPtr("from_the_future")},
		{Status: nil},
		{Status: stringPtr("partial_outage")},
		{Status: stringPtr("degraded_performance")},
		{Status: stringPtr("under_maintenance")},
		{Status: stringPtr("operational")},
	})

	var labels []string
	for _, tally := range tallies {
		labels = append(labels, tally.appearance.label)
	}
	want := []string{"Operational", "Maintenance", "Degraded", "Partial outage", "Major outage", "Pending", "from_the_future"}
	if strings.Join(labels, ",") != strings.Join(want, ",") {
		t.Fatalf("order = %v, want %v", labels, want)
	}
}

func TestStatusSummaryReadsCountThenLowercaseStatus(t *testing.T) {
	summary := statusSummary([]api.App{
		{Status: stringPtr("operational")}, {Status: stringPtr("operational")}, {Status: stringPtr("degraded_performance")},
	})

	assertViewContains(t, summary, "2 operational", "1 degraded")
}
