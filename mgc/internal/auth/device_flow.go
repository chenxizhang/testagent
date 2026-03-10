package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// deviceCodeResponse is the response from the device code endpoint.
type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
	Message         string `json:"message"`
}

// tokenResponse is the response from the token endpoint.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// httpClient is used for auth requests (can be overridden in tests).
var httpClient = &http.Client{Timeout: 30 * time.Second}

// requestDeviceCode initiates the device code flow.
func requestDeviceCode(tenantID, clientID string) (*deviceCodeResponse, error) {
	endpoint := fmt.Sprintf("%s/%s/oauth2/v2.0/devicecode", AuthorityBase, tenantID)

	data := url.Values{
		"client_id": {clientID},
		"scope":     {GraphScope},
	}

	resp, err := httpClient.PostForm(endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("POST %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("device code request failed (%d): %s", resp.StatusCode, body)
	}

	var code deviceCodeResponse
	if err := json.Unmarshal(body, &code); err != nil {
		return nil, fmt.Errorf("parsing device code response: %w", err)
	}

	if code.Interval == 0 {
		code.Interval = 5
	}

	return &code, nil
}

// pollForToken polls the token endpoint until the user authenticates or timeout.
func pollForToken(tenantID, clientID string, code *deviceCodeResponse) (*Token, error) {
	endpoint := fmt.Sprintf("%s/%s/oauth2/v2.0/token", AuthorityBase, tenantID)

	timeout := time.Now().Add(time.Duration(code.ExpiresIn) * time.Second)

	for time.Now().Before(timeout) {
		time.Sleep(time.Duration(code.Interval) * time.Second)

		data := url.Values{
			"client_id":   {clientID},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
			"device_code": {code.DeviceCode},
		}

		resp, err := httpClient.PostForm(endpoint, data)
		if err != nil {
			return nil, fmt.Errorf("token request: %w", err)
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var tokenResp tokenResponse
		if err := json.Unmarshal(body, &tokenResp); err != nil {
			continue
		}

		switch tokenResp.Error {
		case "":
			// Success
			return &Token{
				AccessToken:  tokenResp.AccessToken,
				RefreshToken: tokenResp.RefreshToken,
				ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
			}, nil
		case "authorization_pending":
			// User hasn't authenticated yet — keep polling
			continue
		case "slow_down":
			// Server asked us to slow down
			time.Sleep(5 * time.Second)
			continue
		case "expired_token":
			return nil, fmt.Errorf("device code expired — run 'mgc auth login' again")
		default:
			return nil, fmt.Errorf("authentication error %q: %s", tokenResp.Error, tokenResp.ErrorDesc)
		}
	}

	return nil, fmt.Errorf("authentication timed out — the code may have expired")
}

// refreshToken exchanges a refresh token for a new access token.
func refreshToken(tenantID, clientID, refreshTok string) (*Token, error) {
	endpoint := fmt.Sprintf("%s/%s/oauth2/v2.0/token", AuthorityBase, tenantID)

	data := url.Values{
		"client_id":     {clientID},
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshTok},
		"scope":         {GraphScope},
	}

	resp, err := httpClient.PostForm(endpoint, data)
	if err != nil {
		return nil, fmt.Errorf("token refresh request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading refresh response: %w", err)
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parsing refresh response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("token refresh failed %q: %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	return &Token{
		AccessToken:  tokenResp.AccessToken,
		RefreshToken: tokenResp.RefreshToken,
		ExpiresAt:    time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second),
	}, nil
}


