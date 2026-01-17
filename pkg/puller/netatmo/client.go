package netatmo

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client handles Netatmo API communication
type Client struct {
	httpClient     *http.Client
	clientID       string
	clientSecret   string
	redirectURI    string
	accessToken    string
	refreshToken   string
	tokenExpiry    time.Time
	state          string
	onTokenRefresh func(accessToken, refreshToken string, expiry time.Time) error
	onTokenInvalid func(state string) error
}

// NewClient creates a new Netatmo API client
func NewClient(clientID, clientSecret, redirectURI string) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURI:  redirectURI,
	}
}

// tokenResponse represents the Netatmo OAuth2 token response
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

// SetAccessToken sets the access token directly (e.g., from database)
func (c *Client) SetAccessToken(token string) {
	c.accessToken = token
}

// GetAccessToken returns the current access token
func (c *Client) GetAccessToken() string {
	return c.accessToken
}

// SetRefreshToken sets the refresh token directly (e.g., from database)
func (c *Client) SetRefreshToken(token string) {
	c.refreshToken = token
}

// GetRefreshToken returns the current refresh token
func (c *Client) GetRefreshToken() string {
	return c.refreshToken
}

// SetTokenExpiry sets the token expiry time directly (e.g., from database)
func (c *Client) SetTokenExpiry(tokenExpiry time.Time) {
	c.tokenExpiry = tokenExpiry
}

// GetTokenExpiry returns the current token expiry time
func (c *Client) GetTokenExpiry() time.Time {
	return c.tokenExpiry
}

// IsTokenValid checks if the current token is still valid
func (c *Client) IsTokenValid() bool {
	return c.accessToken != "" && time.Now().Before(c.tokenExpiry)
}

func (c *Client) SetState(state string) {
	c.state = state
}

// SetTokenRefreshCallback sets a callback that's called when tokens are refreshed
func (c *Client) SetTokenRefreshCallback(callback func(accessToken, refreshToken string, expiry time.Time) error) {
	c.onTokenRefresh = callback
}

// SetTokenInvalidCallback sets a callback that's called when tokens are invalid
func (c *Client) SetTokenInvalidCallback(callback func(state string) error) {
	c.onTokenInvalid = callback
}

// GetAuthorizationURL returns the URL where the user should authenticate
func (c *Client) GetAuthorizationURL() (string, string) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		fmt.Printf("failed to generate state: %v\n", err)
		return "", ""
	}
	c.state = base64.URLEncoding.EncodeToString(b)

	params := url.Values{}
	params.Set("client_id", c.clientID)
	params.Set("redirect_uri", c.redirectURI)
	params.Set("scope", "read_station")
	params.Set("state", c.state)

	return "https://api.netatmo.com/oauth2/authorize?" + params.Encode(), c.state
}

// GetAccessTokenFromCode exchanges an authorization code for an access token
func (c *Client) GetAccessTokenFromCode(ctx context.Context, code string, state string) error {
	if state != c.state {
		return fmt.Errorf("state mismatch: expected %s, got %s", c.state, state)
	}

	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", c.redirectURI)
	data.Set("scope", "read_station")

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.netatmo.com/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return fmt.Errorf("failed to parse token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.refreshToken = tokenResp.RefreshToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	return nil
}

// RefreshAccessToken uses the refresh token to get a new access token
func (c *Client) RefreshAccessToken(ctx context.Context) error {
	if c.refreshToken == "" {
		return fmt.Errorf("no refresh token available")
	}

	data := url.Values{}
	data.Set("grant_type", "refresh_token")
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)
	data.Set("refresh_token", c.refreshToken)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.netatmo.com/oauth2/token", strings.NewReader(data.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create refresh token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to refresh access token: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]interface{}
		if err := json.Unmarshal(body, &errResp); err == nil {
			if errMsg, ok := errResp["error"].(string); ok {
				if errMsg == "invalid_grant" {
					authUrl, state := c.GetAuthorizationURL()
					if c.onTokenInvalid != nil {
						if err := c.onTokenInvalid(state); err != nil {
							return fmt.Errorf("token is invalid or expired but failed to execute token invalid callback : %w", err)
						}
					}

					return fmt.Errorf("refresh token is invalid or expired - you need to re-authorize by visiting the authorization URL: %s", authUrl)
				}
			}
		}
		return fmt.Errorf("refresh token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp tokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return fmt.Errorf("failed to parse refresh token response: %w", err)
	}

	c.accessToken = tokenResp.AccessToken
	c.refreshToken = tokenResp.RefreshToken
	c.tokenExpiry = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)

	// Persist the new tokens if callback is set
	if c.onTokenRefresh != nil {
		if err := c.onTokenRefresh(c.accessToken, c.refreshToken, c.tokenExpiry); err != nil {
			return fmt.Errorf("failed to execute token refresh callback: %w", err)
		}
	}

	return nil
}

// ensureValidToken checks if token is still valid, refreshes if needed
func (c *Client) ensureValidToken(ctx context.Context) error {
	if time.Now().Before(c.tokenExpiry.Add(-15 * time.Minute)) {
		// Token is still valid (with 5 minute buffer)
		return nil
	}

	// Token is expired or about to expire, refresh it
	return c.RefreshAccessToken(ctx)
}
