package api

import (
	"io"
	"net/http"
	"testing"
)

type recordedRequest struct {
	method      string
	path        string
	query       string
	contentType string
	requestBody string
}

// newRecordingClient answers every request with status and responseBody and remembers the last
// request it saw.
func newRecordingClient(t *testing.T, status int, responseBody string) (*Client, *recordedRequest) {
	t.Helper()
	recorded := &recordedRequest{}
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requestBody, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
		}
		*recorded = recordedRequest{
			method:      r.Method,
			path:        r.URL.EscapedPath(),
			query:       r.URL.RawQuery,
			contentType: r.Header.Get("Content-Type"),
			requestBody: string(requestBody),
		}
		writeJSON(w, status, responseBody)
	})
	return c, recorded
}

func assertRequest(t *testing.T, got *recordedRequest, want recordedRequest) {
	t.Helper()
	if *got != want {
		t.Errorf("request =\n  %+v\nwant\n  %+v", *got, want)
	}
}
