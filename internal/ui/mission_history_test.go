package ui

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/statuspane/tui/internal/api"
)

func TestTheStripShowsADayEachWithItsRangeAndWhenTrackingBegan(t *testing.T) {
	m := modelWithDetail(t, degradedAppDetail(), testWidth, testHeight)

	view := plain(t, m.View().Content)

	assertViewContains(t, view, "Last 30 days", "Sep 15", "today", "Tracking since Oct 3")
	if cells := strings.Count(lineContaining(t, view, dayCell+dayCellGap+dayCell), dayCell); cells != historyDayCount {
		t.Errorf("the strip has %d day cells, want %d", cells, historyDayCount)
	}
}

func TestADayWithNoDataIsDotsAndNeverLooksOperational(t *testing.T) {
	trackingSince := time.Date(2026, time.October, 3, 0, 0, 0, 0, time.UTC)
	history := thirtyOperationalDays(&trackingSince)
	for index := range 5 {
		history.Days[index].Status = nil
	}
	m := modelWithHistory(t, degradedAppDetail(), history, nil, testWidth, testHeight)

	strip := lineContaining(t, m.View().Content, noDataCell+dayCellGap+noDataCell)

	if !strings.Contains(plain(t, strip), strings.Repeat(noDataCell+dayCellGap, 5)+dayCell) {
		t.Errorf("the first five days have no data and the sixth does:\n%s", plain(t, strip))
	}
}

func TestInANarrowWindowTheStripShowsTheMostRecentDaysThatFitAndSaysHowMany(t *testing.T) {
	m := modelWithDetail(t, degradedAppDetail(), fallbackWidth, 40)

	view := plain(t, m.View().Content)

	// The centre column's content is 40 cells at this width, and the last day needs no gap after it.
	fittingDays := (40 + len(dayCellGap)) / cellsPerDay
	firstShownDay := detailNow.AddDate(0, 0, 1-fittingDays)
	assertViewContains(t, view, fmt.Sprintf("Last %d days", fittingDays), firstShownDay.Format("Jan 2"))
	assertViewLacks(t, view, "days ago")
	assertEveryLineHasWidth(t, m.View().Content, fallbackWidth)
}

func TestAFailedHistoryRequestSaysSoAndLeavesTheRestOfTheScreenAlone(t *testing.T) {
	m := modelWithHistory(t, degradedAppDetail(), api.AppHistory{}, errors.New("boom"), testWidth, testHeight)

	view := plain(t, m.View().Content)

	assertViewContains(t, view, "History unavailable", "DEGRADED PERFORMANCE", "Git operations degraded", "github / API Requests")
	assertViewLacks(t, view, "boom")
}

func TestUntilTheHistoryArrivesTheStripSaysItIsLoading(t *testing.T) {
	m := missionControlModel(api.App{Name: "Web App", Slug: "web-app", Status: stringPtr("operational")})

	assertViewContains(t, m.View().Content, "Status history", "Loading…")
}

func TestWhenTheWindowIsTooShortTheStripGivesWayToTheIncidentAndTheComponents(t *testing.T) {
	m := modelWithDetail(t, degradedAppDetail(), testWidth, 30)

	view := plain(t, m.View().Content)

	assertViewLacks(t, view, "Last 30 days")
	assertViewContains(t, view, "Git operations degraded", "Affects ▸ API Requests", "github / API Requests")
}

func TestTheSelectedAppsHistoryIsAskedFor(t *testing.T) {
	var askedFor string
	backend := fakeAppsBackend{
		fakeAppLister: fakeAppLister{apps: []api.App{{Name: "Web App", Slug: "web-app"}}},
		fakeAppGetter: fakeAppGetter{gotHistorySlug: &askedFor},
	}
	m := NewAppsModel(backend)

	drainCommands(m, m.Init())

	if askedFor != "web-app" {
		t.Errorf("history was asked for %q", askedFor)
	}
}

func TestOneAppsHistoryIsNeverShownForAnotherAndAFailureKeepsWhatArrived(t *testing.T) {
	webApp, payments := api.App{Slug: "web-app"}, api.App{Slug: "payments"}
	history := thirtyOperationalDays(nil)

	detail := newSelectedDetail().withHistoryLoaded(selectedHistoryLoadedMsg{slug: "web-app", history: history})
	if _, state := detail.historyOf(payments); state != historyLoading {
		t.Errorf("payments has no history yet, but is %v", state)
	}

	detail = detail.withHistoryFailed(selectedHistoryFailedMsg{slug: "web-app"}).withHistoryFailed(selectedHistoryFailedMsg{slug: "payments"})

	if shown, state := detail.historyOf(webApp); state != historyLoaded || len(shown.Days) != historyDayCount {
		t.Errorf("a later failure must not take away the history web-app has: %v", state)
	}
	if _, state := detail.historyOf(payments); state != historyFailed {
		t.Errorf("payments never loaded and failed, but is %v", state)
	}

	detail = detail.withHistoryLoaded(selectedHistoryLoadedMsg{slug: "payments", history: history})
	if _, state := detail.historyOf(payments); state != historyLoaded {
		t.Errorf("a later success replaces the failure, but payments is %v", state)
	}
}

func TestTheStripKeepsTheNewestDaysWhenOnlySomeFit(t *testing.T) {
	trackingSince := time.Date(2026, time.October, 3, 0, 0, 0, 0, time.UTC)
	history := thirtyOperationalDays(&trackingSince)
	history.Days[0].Status = stringPtr("major_outage")
	history.Days[len(history.Days)-1].Status = stringPtr("degraded_performance")
	m := modelWithHistory(t, degradedAppDetail(), history, nil, fallbackWidth, 40)

	var strip string
	for _, line := range strings.Split(m.View().Content, "\n") {
		if strings.Contains(plain(t, line), dayCell+dayCellGap+dayCell) {
			strip = line
		}
	}

	if !strings.Contains(strip, appearanceOf(stringPtr("degraded_performance")).style.Render(dayCell)) {
		t.Errorf("today's degraded day should be the last cell:\n%s", strip)
	}
	if strings.Contains(strip, appearanceOf(stringPtr("major_outage")).style.Render(dayCell)) {
		t.Errorf("the outage was 30 days ago and does not fit:\n%s", strip)
	}
}

func TestADateThatCannotBeReadFallsBackToHowManyDaysBack(t *testing.T) {
	days := []api.HistoryDay{{Date: "not a date"}, {Date: "2026-10-14"}}

	if got := historyRangeLabels(days, 2); !strings.HasPrefix(got, "2 days ago") {
		t.Errorf("labels = %q", got)
	}
	if got := historyRangeLabels(nil, 0); got != "" {
		t.Errorf("no days, no labels, but got %q", got)
	}
}
