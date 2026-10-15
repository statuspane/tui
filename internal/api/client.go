// Package api is a client for the StatusPane public API.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	apiPath        = "/api/v1"
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
	if err := c.get(ctx, "/apps", &apps); err != nil {
		return nil, err
	}
	return apps, nil
}

func (c *Client) GetApp(ctx context.Context, slug string) (AppDetail, error) {
	var appDetail AppDetail
	if err := c.get(ctx, "/apps/"+url.PathEscape(slug), &appDetail); err != nil {
		return AppDetail{}, err
	}
	return appDetail, nil
}

func (c *Client) newRequest(ctx context.Context, method, path string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+apiPath+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", c.userAgent)
	return req, nil
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := c.newRequest(ctx, http.MethodGet, path)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return decodeError(resp)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding %s response: %w", path, err)
	}
	return nil
}
