// Package api is a client for the StatusPane public API.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	apiPath        = "/api/v1"
	appsPath       = "/apps"
	mediaTypeJSON  = "application/json"
	requestTimeout = 10 * time.Second
)

type Client struct {
	baseURL    string
	apiKey     string
	userAgent  string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey, version string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		userAgent:  "statuspane-tui/" + version,
		httpClient: &http.Client{Timeout: requestTimeout},
	}
}

func (c *Client) ListApps(ctx context.Context) ([]App, error) {
	var apps []App
	if err := c.do(ctx, http.MethodGet, appsPath, nil, &apps); err != nil {
		return nil, err
	}
	return apps, nil
}

func (c *Client) GetApp(ctx context.Context, appSlug string) (AppDetail, error) {
	var appDetail AppDetail
	if err := c.do(ctx, http.MethodGet, appPath(appSlug), nil, &appDetail); err != nil {
		return AppDetail{}, err
	}
	return appDetail, nil
}

func appPath(appSlug string) string {
	return appsPath + "/" + url.PathEscape(appSlug)
}

func (c *Client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+apiPath+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", mediaTypeJSON)
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", mediaTypeJSON)
	}
	return req, nil
}

// do sends requestBody as JSON when it is not nil. A nil out means the caller expects no
// response body (a 204 from a delete), so the body is not read; otherwise it is decoded into out.
func (c *Client) do(ctx context.Context, method, path string, requestBody, out any) error {
	var body io.Reader
	if requestBody != nil {
		encoded, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("encoding %s request: %w", path, err)
		}
		body = bytes.NewReader(encoded)
	}
	req, err := c.newRequest(ctx, method, path, body)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return decodeError(resp)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding %s response: %w", path, err)
	}
	return nil
}
