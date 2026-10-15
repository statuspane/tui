package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Error codes the API answers with.
const (
	CodeUnauthorized = "unauthorized"
	CodeRateLimited  = "rate_limited"
)

// APIError is a non-2xx response. Code and Message come from the API's error body
// and are empty when the response wasn't in that shape.
type APIError struct {
	Status     int
	Code       string
	Message    string
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("API returned HTTP %d", e.Status)
	}
	return fmt.Sprintf("%s (HTTP %d): %s", e.Code, e.Status, e.Message)
}

func decodeError(resp *http.Response) *APIError {
	apiErr := &APIError{Status: resp.StatusCode}
	if secs, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil {
		apiErr.RetryAfter = time.Duration(secs) * time.Second
	}

	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err == nil {
		apiErr.Code = body.Error.Code
		apiErr.Message = body.Error.Message
	}
	return apiErr
}
