package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(srv.URL, "sp_test", "test")
}

func writeJSON(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func jsonHandler(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) { writeJSON(w, status, body) }
}

func TestListAppsDecodesNullableStatus(t *testing.T) {
	c := newTestClient(t, jsonHandler(http.StatusOK, `[
		{"name":"Web App","slug":"web-app","status":"operational","watched_components_count":12},
		{"name":"New","slug":"new","status":null,"watched_components_count":0}
	]`))

	apps, err := c.ListApps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(apps) != 2 || apps[0].Slug != "web-app" || apps[0].WatchedComponentsCount != 12 {
		t.Fatalf("got %+v", apps)
	}
	if apps[0].Status == nil || *apps[0].Status != "operational" {
		t.Fatalf("status = %v", apps[0].Status)
	}
	if apps[1].Status != nil {
		t.Fatalf("null status should be nil, got %v", *apps[1].Status)
	}
}

func TestGetAppDecodesComponentsAndIncidents(t *testing.T) {
	var gotPath string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		writeJSON(w, http.StatusOK, `{
			"name":"Web App","slug":"web-app","status":"degraded","watched_components_count":1,
			"last_checked_at":"2026-10-11T17:30:00Z",
			"watched_components":[{"provider":"github","id":"abc","name":"API Requests","group_name":null,"status":"degraded"}],
			"active_incidents":[{
				"id":"kz9","provider":"github","title":"Git operations degraded","impact":"major",
				"status":"identified","started_at":"2026-10-11T13:00:00Z","url":"https://example.test/i/kz9",
				"latest_update":null,
				"affected_components":[{"provider":"github","id":"abc","name":"API Requests"}]
			}]
		}`)
	})

	appDetail, err := c.GetApp(context.Background(), "web-app")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/api/v1/apps/web-app" {
		t.Fatalf("path = %q", gotPath)
	}
	if appDetail.LastCheckedAt == nil || !appDetail.LastCheckedAt.Equal(time.Date(2026, 10, 11, 17, 30, 0, 0, time.UTC)) {
		t.Fatalf("last_checked_at = %v", appDetail.LastCheckedAt)
	}
	if len(appDetail.WatchedComponents) != 1 || appDetail.WatchedComponents[0].GroupName != nil {
		t.Fatalf("components = %+v", appDetail.WatchedComponents)
	}
	incident := appDetail.ActiveIncidents[0]
	if incident.Title != "Git operations degraded" || incident.LatestUpdate != nil || len(incident.AffectedComponents) != 1 {
		t.Fatalf("incident = %+v", incident)
	}
}

func TestRequestSendsBearerKey(t *testing.T) {
	var got string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Authorization")
		writeJSON(w, http.StatusOK, `[]`)
	})

	if _, err := c.ListApps(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got != "Bearer sp_test" {
		t.Fatalf("Authorization = %q", got)
	}
}

func TestUnauthorizedBecomesAPIError(t *testing.T) {
	c := newTestClient(t, jsonHandler(http.StatusUnauthorized, `{"error":{"code":"unauthorized","message":"Invalid API key."}}`))

	_, err := c.ListApps(context.Background())

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want *APIError, got %v", err)
	}
	if apiErr.Status != http.StatusUnauthorized || apiErr.Code != CodeUnauthorized || apiErr.Message != "Invalid API key." {
		t.Fatalf("got %+v", apiErr)
	}
}

func TestRateLimitedExposesRetryAfter(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "42")
		writeJSON(w, http.StatusTooManyRequests, `{"error":{"code":"rate_limited","message":"Too many requests."}}`)
	})

	_, err := c.ListApps(context.Background())

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != CodeRateLimited || apiErr.RetryAfter != 42*time.Second {
		t.Fatalf("got %v", err)
	}
}

func TestNonAPIErrorBodyStillReturnsAPIError(t *testing.T) {
	c := newTestClient(t, jsonHandler(http.StatusBadGateway, `<html>Bad gateway</html>`))

	_, err := c.ListApps(context.Background())

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusBadGateway || apiErr.Code != "" {
		t.Fatalf("got %v", err)
	}
}

func TestCancelledContextStopsRequest(t *testing.T) {
	c := newTestClient(t, jsonHandler(http.StatusOK, `[]`))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := c.ListApps(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}
