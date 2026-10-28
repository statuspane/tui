package api

import (
	"context"
	"net/http"
	"testing"
)

func TestGetAppHistoryRequestsTheWindowAndDecodesNullDays(t *testing.T) {
	c, recorded := newRecordingClient(t, http.StatusOK, `{
		"tracking_since":"2026-10-03T16:39:03Z",
		"days":[{"date":"2026-10-02","status":null},{"date":"2026-10-03","status":"degraded_performance"}],
		"components":[{"provider":"github","id":"br0l2tvcx85d","name":"API Requests","tracking_since":null,"days":[]}]
	}`)

	appHistory, err := c.GetAppHistory(context.Background(), "web-app", 30)
	if err != nil {
		t.Fatal(err)
	}

	assertRequest(t, recorded, recordedRequest{method: http.MethodGet, path: "/api/v1/apps/web-app/history", query: "days=30"})
	if appHistory.TrackingSince == nil || appHistory.TrackingSince.Day() != 3 {
		t.Fatalf("tracking_since = %v", appHistory.TrackingSince)
	}
	if len(appHistory.Days) != 2 || appHistory.Days[0].Status != nil || *appHistory.Days[1].Status != "degraded_performance" {
		t.Fatalf("days = %+v", appHistory.Days)
	}
}
