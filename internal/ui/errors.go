package ui

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/statuspane/tui/internal/api"
	"github.com/statuspane/tui/internal/config"
)

func errorMessage(err error) string {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case api.CodeUnauthorized:
			return "The API key was rejected. Check " + config.EnvAPIKey + "."
		case api.CodeRateLimited:
			return "Rate limited. Try again in " + secondsText(apiErr.RetryAfter) + "."
		case api.CodeNotFound:
			return "Not found. It may have been deleted or renamed."
		}
	}
	return err.Error()
}

// fieldErrorMessage prefers the API's message about one request field, then the general text.
func fieldErrorMessage(err error, field string) string {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) {
		for _, detail := range apiErr.Details {
			if detail.Field == field {
				return detail.Message
			}
		}
	}
	return errorMessage(err)
}

// validationMessage joins the API's messages about each rejected field, or gives the general text.
func validationMessage(err error) string {
	var apiErr *api.APIError
	if errors.As(err, &apiErr) && len(apiErr.Details) > 0 {
		messages := make([]string, len(apiErr.Details))
		for index, detail := range apiErr.Details {
			messages[index] = detail.Message
		}
		return strings.Join(messages, "; ")
	}
	return errorMessage(err)
}

func isNotFound(err error) bool {
	var apiErr *api.APIError
	return errors.As(err, &apiErr) && apiErr.Code == api.CodeNotFound
}

// secondsText rounds up, so a sub-second wait never reads as "0 seconds".
func secondsText(duration time.Duration) string {
	seconds := int(math.Ceil(duration.Seconds()))
	if seconds == 1 {
		return "1 second"
	}
	return fmt.Sprintf("%d seconds", seconds)
}
