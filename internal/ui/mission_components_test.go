package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/statuspane/tui/internal/api"
)

func TestTheComponentsPanelListsWhatIsNotOperationalAndCountsTheRest(t *testing.T) {
	m := modelWithDetail(t, degradedAppDetail(), testWidth, testHeight)

	view := plain(t, m.View().Content)

	assertViewContains(t, view, "Affected components", "github / API Requests", "Degraded", "1 operational")
	if strings.Contains(view, "github / Pages") {
		t.Errorf("an operational component is counted, not listed:\n%s", view)
	}
}

func componentWithStatus(name, status string) api.Component {
	return api.Component{
		ComponentReference: api.ComponentReference{Provider: "github", ID: name, Name: name},
		Status:             stringPtr(status),
	}
}

func TestTheWorstComponentsComeFirst(t *testing.T) {
	appDetail := degradedAppDetail()
	appDetail.WatchedComponents = []api.Component{
		componentWithStatus("Webhooks", "degraded_performance"),
		componentWithStatus("Git Operations", "major_outage"),
		componentWithStatus("Pages", "partial_outage"),
	}
	m := modelWithDetail(t, appDetail, testWidth, testHeight)

	view := plain(t, m.View().Content)

	major, partial, degraded := strings.Index(view, "Git Operations"), strings.Index(view, "Pages"), strings.Index(view, "Webhooks")
	if major < 0 || major > partial || partial > degraded {
		t.Errorf("want major, partial, then degraded outage:\n%s", view)
	}
}

func TestWhenNothingIsAffectedThereIsNoComponentsPanel(t *testing.T) {
	appDetail := degradedAppDetail()
	appDetail.WatchedComponents = []api.Component{componentWithStatus("Pages", "operational")}
	m := modelWithDetail(t, appDetail, testWidth, testHeight)

	assertViewLacks(t, plain(t, m.View().Content), "Affected components")
}

func TestTooManyAffectedComponentsAreCutWithACountOfWhatIsLeftOut(t *testing.T) {
	appDetail := degradedAppDetail()
	appDetail.WatchedComponents = nil
	for index := range 12 {
		appDetail.WatchedComponents = append(appDetail.WatchedComponents, componentWithStatus(fmt.Sprintf("Part %02d", index), "degraded_performance"))
	}
	m := modelWithDetail(t, appDetail, testWidth, 30)

	view := m.View().Content

	assertViewContains(t, view, "Part 00", "… and ", " more", "0 operational")
	assertHeightIs(t, view, 30)
}

func TestWhenNotAllRowsFitTheLastRowIsGivenToTheCountOfWhatIsLeftOut(t *testing.T) {
	affected := []api.Component{componentWithStatus("A", "major_outage"), componentWithStatus("B", "partial_outage"), componentWithStatus("C", "degraded_performance")}
	cases := []struct {
		name        string
		rowRoom     int
		wantShown   int
		wantLeftOut int
	}{
		{"all fit", 3, 3, 0},
		{"more room than rows", 5, 3, 0},
		{"one too many", 2, 1, 2},
		{"room for the count only", 1, 0, 3},
	}

	for _, testCase := range cases {
		shown, leftOut := rowsThatFit(affected, testCase.rowRoom)

		if len(shown) != testCase.wantShown || leftOut != testCase.wantLeftOut {
			t.Errorf("%s: shown %d, left out %d; want %d and %d", testCase.name, len(shown), leftOut, testCase.wantShown, testCase.wantLeftOut)
		}
	}
}

func TestWithNoRoomForARowThereIsNoComponentsPanel(t *testing.T) {
	m := modelWithDetail(t, degradedAppDetail(), testWidth, testHeight)
	app, _ := m.selectedApp()

	for _, maxHeight := range []int{0, panelChromeHeight, panelChromeHeight + 1} {
		if panel := m.affectedPanel(app, 60, maxHeight); panel != "" {
			t.Errorf("a panel %d lines tall has no room for a row and its count, but is drawn:\n%s", maxHeight, panel)
		}
	}
}
