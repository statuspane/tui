package api

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

const savedAppJSON = `{"name":"Web App","slug":"web-app","status":null,"watched_components_count":0,
	"last_checked_at":null,"watched_components":[],"active_incidents":[]}`

var githubComponent = ComponentRef{Provider: "github", ID: "abc"}

const githubComponentJSON = `{"provider":"github","id":"abc"}`

func TestCreateAppSendsOnlyTheNameWhenThereAreNoComponents(t *testing.T) {
	c, recorded := newRecordingClient(t, http.StatusCreated, savedAppJSON)

	appDetail, err := c.CreateApp(context.Background(), "Web App", nil)
	if err != nil {
		t.Fatal(err)
	}
	assertRequest(t, recorded, recordedRequest{
		method: http.MethodPost, path: "/api/v1/apps", contentType: mediaTypeJSON, requestBody: `{"name":"Web App"}`,
	})
	if appDetail.Slug != "web-app" {
		t.Fatalf("appDetail = %+v", appDetail)
	}
}

func TestCreateAppSendsTheComponentsItIsGiven(t *testing.T) {
	c, recorded := newRecordingClient(t, http.StatusCreated, savedAppJSON)

	if _, err := c.CreateApp(context.Background(), "Web App", []ComponentRef{githubComponent}); err != nil {
		t.Fatal(err)
	}
	assertRequest(t, recorded, recordedRequest{
		method: http.MethodPost, path: "/api/v1/apps", contentType: mediaTypeJSON,
		requestBody: `{"name":"Web App","watched_components":[` + githubComponentJSON + `]}`,
	})
}

func TestRenameAppSendsOnlyTheName(t *testing.T) {
	c, recorded := newRecordingClient(t, http.StatusOK, savedAppJSON)

	if _, err := c.RenameApp(context.Background(), "web-app", "Storefront"); err != nil {
		t.Fatal(err)
	}
	assertRequest(t, recorded, recordedRequest{
		method: http.MethodPatch, path: "/api/v1/apps/web-app", contentType: mediaTypeJSON, requestBody: `{"name":"Storefront"}`,
	})
}

func TestSetWatchedComponentsSendsOnlyTheSet(t *testing.T) {
	c, recorded := newRecordingClient(t, http.StatusOK, savedAppJSON)

	if _, err := c.SetWatchedComponents(context.Background(), "web-app", []ComponentRef{githubComponent}); err != nil {
		t.Fatal(err)
	}
	assertRequest(t, recorded, recordedRequest{
		method: http.MethodPatch, path: "/api/v1/apps/web-app", contentType: mediaTypeJSON,
		requestBody: `{"watched_components":[` + githubComponentJSON + `]}`,
	})
}

func TestClearingTheWatchedSetSendsAnEmptyListNeverNull(t *testing.T) {
	cases := []struct {
		name string
		refs []ComponentRef
	}{
		{"nil list", nil},
		{"empty list", []ComponentRef{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, recorded := newRecordingClient(t, http.StatusOK, savedAppJSON)

			if _, err := c.SetWatchedComponents(context.Background(), "web-app", tc.refs); err != nil {
				t.Fatal(err)
			}
			assertRequest(t, recorded, recordedRequest{
				method: http.MethodPatch, path: "/api/v1/apps/web-app", contentType: mediaTypeJSON,
				requestBody: `{"watched_components":[]}`,
			})
		})
	}
}

func TestWritesEscapeTheAppSlug(t *testing.T) {
	c, recorded := newRecordingClient(t, http.StatusOK, savedAppJSON)

	if _, err := c.RenameApp(context.Background(), "web app", "Storefront"); err != nil {
		t.Fatal(err)
	}
	assertRequest(t, recorded, recordedRequest{
		method: http.MethodPatch, path: "/api/v1/apps/web%20app", contentType: mediaTypeJSON, requestBody: `{"name":"Storefront"}`,
	})
}

func TestDeleteAppAcceptsAnEmptyNoContentResponse(t *testing.T) {
	c, recorded := newRecordingClient(t, http.StatusNoContent, "")

	if err := c.DeleteApp(context.Background(), "web-app"); err != nil {
		t.Fatal(err)
	}
	assertRequest(t, recorded, recordedRequest{method: http.MethodDelete, path: "/api/v1/apps/web-app"})
}

func TestDeletingAnAppThatIsGoneIsNotFound(t *testing.T) {
	c, _ := newRecordingClient(t, http.StatusNotFound, `{"error":{"code":"not_found","message":"No such app."}}`)

	err := c.DeleteApp(context.Background(), "web-app")

	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != CodeNotFound {
		t.Fatalf("got %v", err)
	}
}
