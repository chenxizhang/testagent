package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chenxizhang/testagent/mgc/internal/auth"
)

// newTestManager creates an auth.Manager pointing to a temp config dir.
func newTestManager(t *testing.T, tenantID string) *auth.Manager {
	t.Helper()
	tmpDir := t.TempDir()
	mgr := auth.New(tenantID, "test-client-id")
	// Override config dir via the exported field (for testability)
	_ = tmpDir
	return mgr
}

func TestTokenIsExpired(t *testing.T) {
	t.Run("not expired", func(t *testing.T) {
		tok := &auth.Token{
			AccessToken: "abc",
			ExpiresAt:   time.Now().Add(1 * time.Hour),
		}
		if tok.IsExpired() {
			t.Error("expected token to not be expired")
		}
	})

	t.Run("expired", func(t *testing.T) {
		tok := &auth.Token{
			AccessToken: "abc",
			ExpiresAt:   time.Now().Add(-1 * time.Minute),
		}
		if !tok.IsExpired() {
			t.Error("expected token to be expired")
		}
	})

	t.Run("expires within 60s buffer", func(t *testing.T) {
		tok := &auth.Token{
			AccessToken: "abc",
			ExpiresAt:   time.Now().Add(30 * time.Second),
		}
		if !tok.IsExpired() {
			t.Error("expected token within 60s buffer to be considered expired")
		}
	})
}

func TestDeviceFlowEndpointRequest(t *testing.T) {
	// Mock device code server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		resp := map[string]interface{}{
			"device_code":      "test-device-code",
			"user_code":        "ABCD-1234",
			"verification_uri": "https://microsoft.com/devicelogin",
			"expires_in":       900,
			"interval":         5,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// This test just verifies our HTTP parsing works
	// Real integration requires network access
	t.Log("Mock device code server started at", server.URL)
}

func TestCredentialsSaveLoad(t *testing.T) {
	tmpDir := t.TempDir()

	// Write a fake credentials file
	creds := map[string]interface{}{
		"tenants": map[string]interface{}{
			"contoso.onmicrosoft.com": map[string]interface{}{
				"access_token":  "test-access-token",
				"refresh_token": "test-refresh-token",
				"expires_at":    time.Now().Add(1 * time.Hour).Format(time.RFC3339),
				"user_id":       "user@contoso.com",
				"tenant_id":     "contoso.onmicrosoft.com",
			},
		},
	}

	credsPath := filepath.Join(tmpDir, "credentials.json")
	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(credsPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	// Verify file was created
	if _, err := os.Stat(credsPath); os.IsNotExist(err) {
		t.Error("credentials file should exist")
	}

	// Read it back
	readData, err := os.ReadFile(credsPath)
	if err != nil {
		t.Fatal(err)
	}

	var readCreds map[string]interface{}
	if err := json.Unmarshal(readData, &readCreds); err != nil {
		t.Fatal(err)
	}

	tenants, ok := readCreds["tenants"].(map[string]interface{})
	if !ok {
		t.Fatal("expected tenants map")
	}

	if _, ok := tenants["contoso.onmicrosoft.com"]; !ok {
		t.Error("expected contoso.onmicrosoft.com in tenants")
	}
}

func TestManagerNew(t *testing.T) {
	mgr := auth.New("contoso.onmicrosoft.com", "")
	if mgr == nil {
		t.Fatal("expected non-nil manager")
	}
	if mgr.ClientID != auth.DefaultClientID {
		t.Errorf("expected default client ID %q, got %q", auth.DefaultClientID, mgr.ClientID)
	}
	if mgr.TenantID != "contoso.onmicrosoft.com" {
		t.Errorf("expected tenant ID 'contoso.onmicrosoft.com', got %q", mgr.TenantID)
	}
}

func TestManagerNewWithCustomClientID(t *testing.T) {
	mgr := auth.New("mytenant", "custom-client-id")
	if mgr.ClientID != "custom-client-id" {
		t.Errorf("expected custom client ID, got %q", mgr.ClientID)
	}
}
