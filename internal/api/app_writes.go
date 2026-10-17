package api

import (
	"context"
	"net/http"
)

const (
	nameField              = "name"
	watchedComponentsField = "watched_components"
)

// ComponentRef names a component in a write: the provider's slug and the component's own id.
type ComponentRef struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
}

// CreateApp creates an app. A nil refs list sends no watched_components, so the app
// watches nothing yet; creating is not idempotent, so callers must not retry it blindly.
func (c *Client) CreateApp(ctx context.Context, name string, refs []ComponentRef) (AppDetail, error) {
	fields := map[string]any{nameField: name}
	if refs != nil {
		fields[watchedComponentsField] = refs
	}
	return c.sendAppWrite(ctx, http.MethodPost, appsPath, fields)
}

func (c *Client) RenameApp(ctx context.Context, appSlug, name string) (AppDetail, error) {
	return c.sendAppWrite(ctx, http.MethodPatch, appPath(appSlug), map[string]any{nameField: name})
}

// SetWatchedComponents replaces the whole set the app watches; an empty or nil list clears it.
func (c *Client) SetWatchedComponents(ctx context.Context, appSlug string, refs []ComponentRef) (AppDetail, error) {
	// The API rejects null, and a nil slice would marshal to null.
	if refs == nil {
		refs = []ComponentRef{}
	}
	return c.sendAppWrite(ctx, http.MethodPatch, appPath(appSlug), map[string]any{watchedComponentsField: refs})
}

func (c *Client) DeleteApp(ctx context.Context, appSlug string) error {
	return c.do(ctx, http.MethodDelete, appPath(appSlug), nil, nil)
}

// sendAppWrite sends the changed fields and returns the app as the API now has it.
func (c *Client) sendAppWrite(ctx context.Context, method, path string, fields map[string]any) (AppDetail, error) {
	var appDetail AppDetail
	if err := c.do(ctx, method, path, fields, &appDetail); err != nil {
		return AppDetail{}, err
	}
	return appDetail, nil
}
