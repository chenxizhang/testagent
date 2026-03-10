// Package client provides the Microsoft Graph API HTTP client.
package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	// BaseURL is the Microsoft Graph v1.0 API base URL.
	BaseURL = "https://graph.microsoft.com/v1.0"
	// DefaultTimeout is the HTTP client timeout.
	DefaultTimeout = 30 * time.Second
	// MaxRetries is the maximum number of retries for transient errors.
	MaxRetries = 3
)

// TokenProvider is the interface for obtaining an access token.
type TokenProvider interface {
	GetToken() (string, error)
}

// Client is an HTTP client for the Microsoft Graph API.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Auth       TokenProvider
	Debug      bool
}

// New creates a new Graph API client.
func New(auth TokenProvider) *Client {
	return &Client{
		BaseURL: BaseURL,
		HTTPClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		Auth: auth,
	}
}

// Get sends a GET request to the given Graph API path with optional query parameters.
func (c *Client) Get(path string, params url.Values) (*Response, error) {
	u := c.BaseURL + path
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	return c.doRequest(http.MethodGet, u, nil)
}

// Post sends a POST request with a JSON body.
func (c *Client) Post(path string, body interface{}) (*Response, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}
	return c.doRequest(http.MethodPost, c.BaseURL+path, bodyBytes)
}

// Patch sends a PATCH request with a JSON body.
func (c *Client) Patch(path string, body interface{}) (*Response, error) {
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request body: %w", err)
	}
	return c.doRequest(http.MethodPatch, c.BaseURL+path, bodyBytes)
}

// Delete sends a DELETE request.
func (c *Client) Delete(path string) error {
	_, err := c.doRequest(http.MethodDelete, c.BaseURL+path, nil)
	return err
}

// GetAll follows OData @odata.nextLink pagination and returns all items.
func (c *Client) GetAll(path string, params url.Values) ([]json.RawMessage, error) {
	var all []json.RawMessage
	nextURL := c.BaseURL + path
	if len(params) > 0 {
		nextURL += "?" + params.Encode()
	}

	for nextURL != "" {
		resp, err := c.doRequest(http.MethodGet, nextURL, nil)
		if err != nil {
			return all, err
		}

		all = append(all, resp.Value...)
		nextURL = resp.NextLink
	}

	return all, nil
}

// doRequest performs an HTTP request with authentication and retry logic.
func (c *Client) doRequest(method, url string, body []byte) (*Response, error) {
	token, err := c.Auth.GetToken()
	if err != nil {
		return nil, fmt.Errorf("getting token: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s
			time.Sleep(time.Duration(1<<uint(attempt-1)) * time.Second)
		}

		var reqBody io.Reader
		if body != nil {
			reqBody = bytes.NewReader(body)
		}

		req, err := http.NewRequest(method, url, reqBody)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		if c.Debug {
			fmt.Printf("[DEBUG] %s %s\n", method, url)
		}

		httpResp, err := c.HTTPClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request: %w", err)
			continue
		}

		respBody, err := io.ReadAll(httpResp.Body)
		httpResp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("reading response body: %w", err)
			continue
		}

		if c.Debug {
			fmt.Printf("[DEBUG] Response: %d\n", httpResp.StatusCode)
		}

		// Handle retryable errors
		if httpResp.StatusCode == http.StatusTooManyRequests || httpResp.StatusCode == http.StatusServiceUnavailable {
			lastErr = &GraphError{
				Code:    "TooManyRequests",
				Message: "rate limit exceeded",
				Status:  httpResp.StatusCode,
			}
			if attempt < MaxRetries {
				continue
			}
			return nil, lastErr
		}

		// Handle error responses
		if httpResp.StatusCode >= 400 {
			return nil, parseGraphError(httpResp.StatusCode, respBody)
		}

		// Success — parse response
		return parseResponse(httpResp.StatusCode, respBody)
	}

	return nil, lastErr
}

// parseResponse parses a successful Graph API response.
func parseResponse(statusCode int, body []byte) (*Response, error) {
	resp := &Response{
		StatusCode: statusCode,
		Body:       body,
	}

	if len(body) == 0 {
		return resp, nil
	}

	// Try to parse as OData collection
	var collection struct {
		Value    []json.RawMessage `json:"value"`
		NextLink string            `json:"@odata.nextLink"`
	}
	if err := json.Unmarshal(body, &collection); err == nil && collection.Value != nil {
		resp.Value = collection.Value
		resp.NextLink = collection.NextLink
	}

	return resp, nil
}
