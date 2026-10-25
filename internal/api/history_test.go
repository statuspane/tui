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
		"components":[{
			"provider":"github","id":"br0l2tvcx85d","name":"API Requests","tracking_since":null,
			"days":[{"date":"2026-10-02","status":null},{"date":"2026-10-03","status":"operational"}]
		}]
	}`)

	history, err := c.GetAppHistory(context.Background(), "web-app", 30)
	if err != nil {
		t.Fatal(err)
	}

	assertRequest(t, recorded, recordedRequest{method: http.MethodGet, path: "/api/v1/apps/web-app/history", query: "days=30"})
	if history.TrackingSince == nil || history.TrackingSince.Day() != 3 {
		t.Fatalf("tracking_since = %v", history.TrackingSince)
	}
	if len(history.Days) != 2 || history.Days[0].Status != nil || *history.Days[1].Status != "degraded_performance" {
		t.Fatalf("days = %+v", history.Days)
	}
	component := history.Components[0]
	if component.Provider != "github" || component.Name != "API Requests" || component.TrackingSince != nil {
		t.Fatalf("component = %+v", component)
	}
	if component.Days[0].Status != nil || *component.Days[1].Status != "operational" {
		t.Fatalf("component days = %+v", component.Days)
	}
}
