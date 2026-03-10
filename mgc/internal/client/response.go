package client

import (
	"encoding/json"
	"fmt"
)

// Response wraps a Graph API HTTP response.
type Response struct {
	StatusCode int
	Body       []byte
	Value      []json.RawMessage // populated for OData collection responses
	NextLink   string            // populated when there are more pages
}

// Unmarshal deserializes the response body into v.
func (r *Response) Unmarshal(v interface{}) error {
	if len(r.Body) == 0 {
		return nil
	}
	if err := json.Unmarshal(r.Body, v); err != nil {
		return fmt.Errorf("unmarshaling response: %w", err)
	}
	return nil
}

// UnmarshalValue deserializes the OData value array into a slice.
// Use this for collection responses (GET /users, GET /groups, etc.).
func (r *Response) UnmarshalValue(v interface{}) error {
	if r.Value == nil {
		return r.Unmarshal(v)
	}
	valueBytes, err := json.Marshal(r.Value)
	if err != nil {
		return fmt.Errorf("re-marshaling value: %w", err)
	}
	if err := json.Unmarshal(valueBytes, v); err != nil {
		return fmt.Errorf("unmarshaling value: %w", err)
	}
	return nil
}
