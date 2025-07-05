package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rohanthewiz/serr"
)

const (
	AnthropicClientID    = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	AnthropicAuthURL     = "https://claude.ai/oauth/authorize"
	AnthropicTokenURL    = "https://console.anthropic.com/v1/oauth/token"
	AnthropicRedirectURI = "https://console.anthropic.com/oauth/code/callback"
	AnthropicScopes      = "org:create_api_key user:profile user:inference"
)

// AnthropicAuth handles OAuth authentication for Anthropic
type AnthropicAuth struct {
	storage *Storage
	client  *http.Client
}

// NewAnthropicAuth creates a new Anthropic auth handler
func NewAnthropicAuth() *AnthropicAuth {
	return &AnthropicAuth{
		storage: NewStorage(),
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// AuthorizeURL generates the OAuth authorization URL with PKCE
func (a *AnthropicAuth) AuthorizeURL() (*AuthorizeResult, error) {
	// Generate PKCE parameters
	pkce, err := GeneratePKCE()
	if err != nil {
		return nil, serr.Wrap(err, "failed to generate PKCE")
	}

	// Build authorization URL
	u, err := url.Parse(AnthropicAuthURL)
	if err != nil {
		return nil, serr.Wrap(err, "failed to parse auth URL")
	}

	q := u.Query()
	q.Set("code", "true")
	q.Set("client_id", AnthropicClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", AnthropicRedirectURI)
	q.Set("scope", AnthropicScopes)
	q.Set("code_challenge", pkce.Challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("state", pkce.Verifier)
	u.RawQuery = q.Encode()

	return &AuthorizeResult{
		URL:      u.String(),
		Verifier: pkce.Verifier,
	}, nil
}

// Exchange exchanges an authorization code for tokens
func (a *AnthropicAuth) Exchange(code, verifier string) error {
	// Parse code (format: "code#state")
	splits := strings.Split(code, "#")
	if len(splits) != 2 {
		return serr.New("invalid code format")
	}

	// Prepare token request
	reqBody := map[string]string{
		"code":          splits[0],
		"state":         splits[1],
		"grant_type":    "authorization_code",
		"client_id":     AnthropicClientID,
		"redirect_uri":  AnthropicRedirectURI,
		"code_verifier": verifier,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return serr.Wrap(err, "failed to marshal request")
	}

	// Make token request
	req, err := http.NewRequest("POST", AnthropicTokenURL, bytes.NewReader(body))
	if err != nil {
		return serr.Wrap(err, "failed to create request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return serr.Wrap(err, "failed to exchange code")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return serr.New("token exchange failed with status %d", resp.StatusCode)
	}

	// Parse response
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return serr.Wrap(err, "failed to decode token response")
	}

	// Store credentials
	creds := &Credentials{
		Type:    "oauth",
		Refresh: tokenResp.RefreshToken,
		Access:  tokenResp.AccessToken,
		Expires: time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Unix(),
	}

	if err := a.storage.Set("anthropic", creds); err != nil {
		return serr.Wrap(err, "failed to store credentials")
	}

	return nil
}

// Access returns a valid access token, refreshing if necessary
func (a *AnthropicAuth) Access() (string, error) {
	// Get stored credentials
	creds, err := a.storage.Get("anthropic")
	if err != nil {
		return "", serr.Wrap(err, "failed to get credentials")
	}

	if creds == nil || creds.Type != "oauth" {
		return "", serr.New("no OAuth credentials found")
	}

	// Check if access token is still valid
	if creds.Access != "" && time.Now().Unix() < creds.Expires {
		return creds.Access, nil
	}

	// Refresh the token
	if creds.Refresh == "" {
		return "", serr.New("no refresh token available")
	}

	// Prepare refresh request
	reqBody := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": creds.Refresh,
		"client_id":     AnthropicClientID,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", serr.Wrap(err, "failed to marshal refresh request")
	}

	// Make refresh request
	req, err := http.NewRequest("POST", AnthropicTokenURL, bytes.NewReader(body))
	if err != nil {
		return "", serr.Wrap(err, "failed to create refresh request")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return "", serr.Wrap(err, "failed to refresh token")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", serr.New("token refresh failed with status %d", resp.StatusCode)
	}

	// Parse response
	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", serr.Wrap(err, "failed to decode refresh response")
	}

	// Update stored credentials
	creds.Access = tokenResp.AccessToken
	creds.Refresh = tokenResp.RefreshToken
	creds.Expires = time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second).Unix()

	if err := a.storage.Set("anthropic", creds); err != nil {
		return "", serr.Wrap(err, "failed to update credentials")
	}

	return tokenResp.AccessToken, nil
}

// Logout removes stored credentials
func (a *AnthropicAuth) Logout() error {
	return a.storage.Delete("anthropic")
}

// AuthorizeResult contains the authorization URL and PKCE verifier
type AuthorizeResult struct {
	URL      string `json:"url"`
	Verifier string `json:"verifier"`
}

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	TokenType    string `json:"token_type"`
}