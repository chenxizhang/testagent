package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// GraphError represents a Microsoft Graph API error response.
type GraphError struct {
	Code    string
	Message string
	Status  int
}

func (e *GraphError) Error() string {
	return fmt.Sprintf("graph API error %d (%s): %s", e.Status, e.Code, e.Message)
}

// parseGraphError parses a Graph API error response body into a GraphError.
func parseGraphError(statusCode int, body []byte) error {
	var errResp struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &errResp); err != nil || errResp.Error.Code == "" {
		return &GraphError{
			Code:    fmt.Sprintf("HTTP%d", statusCode),
			Message: string(body),
			Status:  statusCode,
		}
	}

	return &GraphError{
		Code:    errResp.Error.Code,
		Message: errResp.Error.Message,
		Status:  statusCode,
	}
}

// IsNotFound returns true if the error is a 404 Not Found GraphError.
func IsNotFound(err error) bool {
	var ge *GraphError
	if errors.As(err, &ge) {
		return ge.Status == http.StatusNotFound
	}
	return false
}

// IsUnauthorized returns true if the error is a 401 Unauthorized GraphError.
func IsUnauthorized(err error) bool {
	var ge *GraphError
	if errors.As(err, &ge) {
		return ge.Status == http.StatusUnauthorized
	}
	return false
}

// IsForbidden returns true if the error is a 403 Forbidden GraphError.
func IsForbidden(err error) bool {
	var ge *GraphError
	if errors.As(err, &ge) {
		return ge.Status == http.StatusForbidden
	}
	return false
}
