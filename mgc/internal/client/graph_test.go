package client_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/chenxizhang/testagent/mgc/internal/client"
)

// mockAuth is a test TokenProvider that returns a fixed token.
type mockAuth struct {
	token string
}

func (m *mockAuth) GetToken() (string, error) {
	return m.token, nil
}

// newTestClient creates a client pointing to the given server URL.
func newTestClient(serverURL string) *client.Client {
	c := client.New(&mockAuth{token: "test-token"})
	c.BaseURL = serverURL
	return c
}

func TestGetSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Error("expected Authorization: Bearer test-token")
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"value": []map[string]interface{}{
				{"id": "1", "displayName": "Alice"},
				{"id": "2", "displayName": "Bob"},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	resp, err := c.Get("/users", nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if len(resp.Value) != 2 {
		t.Errorf("expected 2 items, got %d", len(resp.Value))
	}
}

func TestGetWithQueryParams(t *testing.T) {
	var receivedURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"value": []interface{}{}})
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	params := url.Values{
		"$filter": {"startsWith(displayName,'A')"},
		"$top":    {"10"},
	}
	_, err := c.Get("/users", params)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if receivedURL == "" || receivedURL == "/users" {
		t.Errorf("expected URL with query params, got %q", receivedURL)
	}
}

func TestGetUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"code":    "InvalidAuthenticationToken",
				"message": "Access token has expired.",
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.Get("/users", nil)
	if err == nil {
		t.Fatal("expected error for 401 response")
	}

	if !client.IsUnauthorized(err) {
		t.Errorf("expected IsUnauthorized to be true, got error: %v", err)
	}
}

func TestGetNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"code":    "Request_ResourceNotFound",
				"message": "Resource not found.",
			},
		})
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	_, err := c.Get("/users/unknown-id", nil)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}

	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound to be true, got error: %v", err)
	}
}

func TestGetPagination(t *testing.T) {
	page := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		page++
		if page == 1 {
			// Return first page with nextLink
			json.NewEncoder(w).Encode(map[string]interface{}{
				"value": []map[string]interface{}{{"id": "1"}},
				"@odata.nextLink": "http://" + r.Host + "/users?page=2",
			})
		} else {
			// Return second page with no nextLink
			json.NewEncoder(w).Encode(map[string]interface{}{
				"value": []map[string]interface{}{{"id": "2"}, {"id": "3"}},
			})
		}
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	all, err := c.GetAll("/users", nil)
	if err != nil {
		t.Fatalf("GetAll failed: %v", err)
	}

	if len(all) != 3 {
		t.Errorf("expected 3 total items from pagination, got %d", len(all))
	}
}

func TestPostSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "new-user-id"})
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	resp, err := c.Post("/users", map[string]string{"displayName": "New User"})
	if err != nil {
		t.Fatalf("Post failed: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", resp.StatusCode)
	}
}

func TestDeleteSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newTestClient(server.URL)
	err := c.Delete("/users/some-id")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestIsNotFoundHelper(t *testing.T) {
	notFoundErr := &client.GraphError{Status: 404, Code: "NotFound", Message: "Not found"}
	if !client.IsNotFound(notFoundErr) {
		t.Error("expected IsNotFound=true for 404 error")
	}

	otherErr := &client.GraphError{Status: 500, Code: "ServiceError", Message: "Server error"}
	if client.IsNotFound(otherErr) {
		t.Error("expected IsNotFound=false for 500 error")
	}
}

func TestGraphErrorMessage(t *testing.T) {
	err := &client.GraphError{
		Status:  403,
		Code:    "Forbidden",
		Message: "Insufficient permissions",
	}
	got := err.Error()
	if got == "" {
		t.Error("expected non-empty error message")
	}
	t.Logf("Error message: %s", got)
}
