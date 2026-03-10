// Package auth provides OAuth2 device flow authentication for Microsoft Graph.
package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const (
	// DefaultClientID is the Microsoft Graph Explorer public client ID.
	DefaultClientID = "14d82eec-204b-4c2f-b7e8-296a70dab67e"
	// GraphScope is the default scope for Microsoft Graph access.
	GraphScope = "https://graph.microsoft.com/.default offline_access"
	// AuthorityBase is the base URL for Microsoft identity platform.
	AuthorityBase = "https://login.microsoftonline.com"
)

// Token holds OAuth2 token data for one tenant.
type Token struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	UserID       string    `json:"user_id,omitempty"`
	TenantID     string    `json:"tenant_id,omitempty"`
}

// IsExpired returns true if the access token has expired (with 60s buffer).
func (t *Token) IsExpired() bool {
	return time.Now().After(t.ExpiresAt.Add(-60 * time.Second))
}

// credentialsFile holds tokens for all tenants.
type credentialsFile struct {
	Tenants map[string]*Token `json:"tenants"`
}

// Manager manages authentication state.
type Manager struct {
	ClientID  string
	TenantID  string
	configDir string
}

// New creates an AuthManager with the given tenant and client ID.
func New(tenantID, clientID string) *Manager {
	if clientID == "" {
		clientID = DefaultClientID
	}
	return &Manager{
		ClientID:  clientID,
		TenantID:  tenantID,
		configDir: configDirectory(),
	}
}

// configDirectory returns the platform-appropriate config directory.
func configDirectory() string {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			appData = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Roaming")
		}
		return filepath.Join(appData, "mgc")
	default:
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "mgc")
	}
}

// credentialsPath returns the path to the credentials file.
func (m *Manager) credentialsPath() string {
	return filepath.Join(m.configDir, "credentials.json")
}

// loadCredentials reads the credentials file from disk.
func (m *Manager) loadCredentials() (*credentialsFile, error) {
	creds := &credentialsFile{Tenants: make(map[string]*Token)}
	path := m.credentialsPath()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return creds, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading credentials: %w", err)
	}

	decoded, err := decrypt(data)
	if err != nil {
		// Fall back to plain JSON for backward compatibility
		decoded = data
	}

	if err := json.Unmarshal(decoded, creds); err != nil {
		return nil, fmt.Errorf("parsing credentials: %w", err)
	}
	return creds, nil
}

// saveCredentials writes credentials to disk (encrypted).
func (m *Manager) saveCredentials(creds *credentialsFile) error {
	if err := os.MkdirAll(m.configDir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(creds, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling credentials: %w", err)
	}

	encrypted := encrypt(data)

	return os.WriteFile(m.credentialsPath(), encrypted, 0600)
}

// GetToken returns a valid access token, refreshing if necessary.
func (m *Manager) GetToken() (string, error) {
	creds, err := m.loadCredentials()
	if err != nil {
		return "", err
	}

	token, ok := creds.Tenants[m.TenantID]
	if !ok || token == nil {
		return "", fmt.Errorf("not authenticated for tenant %q — run 'mgc auth login'", m.TenantID)
	}

	if !token.IsExpired() {
		return token.AccessToken, nil
	}

	// Token expired — refresh it
	if token.RefreshToken == "" {
		return "", fmt.Errorf("token expired and no refresh token — run 'mgc auth login'")
	}

	newToken, err := refreshToken(m.TenantID, m.ClientID, token.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("refreshing token: %w", err)
	}

	// Preserve refresh token if new one is not provided
	if newToken.RefreshToken == "" {
		newToken.RefreshToken = token.RefreshToken
	}
	newToken.TenantID = m.TenantID
	newToken.UserID = token.UserID

	creds.Tenants[m.TenantID] = newToken
	if err := m.saveCredentials(creds); err != nil {
		// Log but don't fail — we have a valid token
		fmt.Fprintf(os.Stderr, "warning: could not save refreshed token: %v\n", err)
	}

	return newToken.AccessToken, nil
}

// Login initiates the OAuth2 device flow for the given tenant.
func (m *Manager) Login() error {
	deviceCode, err := requestDeviceCode(m.TenantID, m.ClientID)
	if err != nil {
		return fmt.Errorf("device code request: %w", err)
	}

	fmt.Fprintf(os.Stderr, "\nTo sign in, use a web browser to open:\n  %s\n", deviceCode.VerificationURI)
	fmt.Fprintf(os.Stderr, "And enter the code: %s\n\n", deviceCode.UserCode)
	fmt.Fprintf(os.Stderr, "Waiting for authentication")

	token, err := pollForToken(m.TenantID, m.ClientID, deviceCode)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	fmt.Fprintln(os.Stderr, " done!")

	creds, err := m.loadCredentials()
	if err != nil {
		creds = &credentialsFile{Tenants: make(map[string]*Token)}
	}

	token.TenantID = m.TenantID
	creds.Tenants[m.TenantID] = token

	if err := m.saveCredentials(creds); err != nil {
		return fmt.Errorf("saving credentials: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Authentication successful! Logged in as %s\n", token.UserID)
	return nil
}

// Logout removes cached credentials for the current tenant.
func (m *Manager) Logout() error {
	creds, err := m.loadCredentials()
	if err != nil {
		return err
	}

	delete(creds.Tenants, m.TenantID)

	if len(creds.Tenants) == 0 {
		// Remove the file entirely if no tenants remain
		return os.Remove(m.credentialsPath())
	}

	return m.saveCredentials(creds)
}

// Status returns the current authenticated user ID for this tenant, or empty string if not authenticated.
func (m *Manager) Status() string {
	creds, err := m.loadCredentials()
	if err != nil {
		return ""
	}
	token, ok := creds.Tenants[m.TenantID]
	if !ok || token == nil {
		return ""
	}
	return token.UserID
}
