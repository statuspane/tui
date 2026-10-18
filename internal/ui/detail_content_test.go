package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/statuspane/tui/internal/api"
)

// testPaneWidth is a pane wide enough for a component and its status.
const testPaneWidth = 40

func TestComponentStatusIsAtTheRightEdgeOfThePane(t *testing.T) {
	components := []api.Component{namedComponent("github", "a", "API Requests", nil)}

	rows := strings.Split(plain(t, renderComponents(components, testPaneWidth)), "\n")

	row := rows[len(rows)-1]
	if lipgloss.Width(row) != testPaneWidth {
		t.Errorf("row is %d wide, want %d: %q", lipgloss.Width(row), testPaneWidth, row)
	}
	if !strings.HasSuffix(row, "Operational") {
		t.Errorf("row does not end with the status: %q", row)
	}
}

func TestComponentsAreListedUnderTheirProviderAndGroup(t *testing.T) {
	core := "Core"
	components := []api.Component{
		namedComponent("github", "a", "API Requests", &core),
		namedComponent("github", "b", "Pages", nil),
	}

	rows := strings.Split(plain(t, renderComponents(components, testPaneWidth)), "\n")

	withStatus := func(left string) string {
		return left + strings.Repeat(" ", testPaneWidth-lipgloss.Width(left)-len("Operational")) + "Operational"
	}
	want := []string{"github", "  Core", withStatus("    ● API Requests"), withStatus("  ● Pages")}
	if strings.Join(rows, "\n") != strings.Join(want, "\n") {
		t.Fatalf("rows = %q, want %q", rows, want)
	}
}

func TestNoComponentsAndNoIncidentsSayTheirOwnThing(t *testing.T) {
	if got := renderComponents(nil, 40); got != noComponentsLabel {
		t.Errorf("components = %q", got)
	}
	if got := renderIncidents(nil, detailNow); got != noIncidentsLabel {
		t.Errorf("incidents = %q", got)
	}
}

func incidentWithImpact(impact string) api.Incident {
	return api.Incident{Title: impact, Impact: &impact}
}

func TestIncidentsPanelTakesTheColourOfTheWorstIncident(t *testing.T) {
	incidents := []api.Incident{incidentWithImpact("minor"), incidentWithImpact("critical"), incidentWithImpact("major")}

	look := incidentsPanelLook(incidents, false)

	if look.border.Render("x") != impactStyles["critical"].Render("x") {
		t.Fatal("the border should use the critical colour")
	}
}

// sameLook compares two looks by what they draw, because a style can't be compared.
func sameLook(first, second panelLook) bool {
	return first.border.Render("x") == second.border.Render("x") && first.title.Render("x") == second.title.Render("x")
}

func TestIncidentsPanelWithoutIncidentsLooksLikeAnyOtherPane(t *testing.T) {
	if !sameLook(incidentsPanelLook(nil, false), untintedPanelLook(false)) || !sameLook(incidentsPanelLook(nil, true), untintedPanelLook(true)) {
		t.Fatal("no incidents, no tint")
	}
}

func TestFocusOnATintedIncidentsPanelShowsAsABoldTitle(t *testing.T) {
	incidents := []api.Incident{incidentWithImpact("major")}

	if !incidentsPanelLook(incidents, true).title.GetBold() || incidentsPanelLook(incidents, false).title.GetBold() {
		t.Fatal("the title is bold exactly while the pane has the focus")
	}
}
