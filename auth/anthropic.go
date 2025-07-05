package auth

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

const (
	// OAuth client ID for OpenCode
	clientID = "9d1c250a-e61b-44d9-88ed-5944d1962f5e"
	// OAuth endpoints
	authorizeURL = "https://claude.ai/oauth/authorize"
	tokenURL     = "https://console.anthropic.com/v1/oauth/token"
	// OAuth scopes
	scopes = "org:create_api_key user:profile user:inference"
	// Redirect URI - must match what's registered with Anthropic
	redirectURI = "https://console.anthropic.com/oauth/code/callback"
)

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	Scope        string    `json:"scope"`
	ExpiresAt    time.Time `json:"expires_at,omitempty"` // Added for tracking expiration
}

// PKCEChallenge holds PKCE parameters
type PKCEChallenge struct {
	Verifier  string
	Challenge string
	Method    string
}

// Storage for PKCE challenges (in-memory) and tokens (persistent)
var (
	pkceStore = sync.Map{}
	storage   *TokenStorage
)

func init() {
	var err error
	storage, err = NewTokenStorage()
	if err != nil {
		logger.LogErr(err, "failed to initialize token storage")
	}
}

// generatePKCE creates a new PKCE challenge
func generatePKCE() (*PKCEChallenge, error) {
	// Generate code verifier (43-128 characters)
	verifierBytes := make([]byte, 32)
	if _, err := rand.Read(verifierBytes); err != nil {
		return nil, serr.Wrap(err, "failed to generate random bytes")
	}

	// Base64 URL encode without padding
	verifier := base64.RawURLEncoding.EncodeToString(verifierBytes)

	// Generate code challenge using SHA256
	h := sha256.New()
	h.Write([]byte(verifier))
	challengeBytes := h.Sum(nil)
	challenge := base64.RawURLEncoding.EncodeToString(challengeBytes)

	return &PKCEChallenge{
		Verifier:  verifier,
		Challenge: challenge,
		Method:    "S256",
	}, nil
}

// AnthropicAuthorizeHandler initiates the OAuth flow
func AnthropicAuthorizeHandler(c rweb.Context) error {
	// Generate PKCE challenge
	pkce, err := generatePKCE()
	if err != nil {
		logger.LogErr(err, "failed to generate PKCE")
		return c.WriteJSON(map[string]string{"error": "failed to generate PKCE"})
	}

	// Generate state parameter for CSRF protection
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return serr.Wrap(err, "failed to generate state")
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// Store PKCE verifier for later use
	pkceStore.Store(state, pkce)

	// Build authorization URL
	params := url.Values{
		"response_type":         {"code"},
		"client_id":             {clientID},
		"redirect_uri":          {redirectURI},
		"scope":                 {scopes},
		"state":                 {state},
		"code_challenge":        {pkce.Challenge},
		"code_challenge_method": {pkce.Method},
	}

	authURL := fmt.Sprintf("%s?%s", authorizeURL, params.Encode())

	// Redirect to Anthropic's OAuth page
	return c.Redirect(302, authURL)
}

// AnthropicCallbackHandler handles the OAuth callback
func AnthropicCallbackHandler(c rweb.Context) error {
	// Get code and state from query parameters
	req := c.Request()
	queryString := req.Query()
	params, _ := url.ParseQuery(queryString)
	code := params.Get("code")
	state := params.Get("state")

	if code == "" {
		return c.WriteJSON(map[string]string{"error": "no authorization code received"})
	}

	// Retrieve PKCE verifier
	pkceValue, ok := pkceStore.LoadAndDelete(state)
	if !ok {
		return c.WriteJSON(map[string]string{"error": "invalid state parameter"})
	}

	pkce := pkceValue.(*PKCEChallenge)

	// Exchange code for tokens
	tokens, err := exchangeCodeForTokens(code, pkce.Verifier)
	if err != nil {
		logger.LogErr(err, "failed to exchange code for tokens")
		return c.WriteJSON(map[string]string{"error": "failed to exchange code for tokens"})
	}

	// Store tokens persistently
	if err := storage.SaveAnthropicTokens(tokens); err != nil {
		logger.LogErr(err, "failed to save tokens")
		return c.WriteJSON(map[string]string{"error": "failed to save tokens"})
	}

	// Redirect to success page or return success response
	successHTML := `<!DOCTYPE html>
<html>
<head>
    <title>Authentication Successful</title>
    <style>
        body { font-family: system-ui; margin: 40px auto; max-width: 650px; }
        .success { padding: 20px; background: #d4edda; color: #155724; border-radius: 8px; }
    </style>
</head>
<body>
    <h1>Authentication Successful!</h1>
    <div class="success">
        <p>You have successfully authenticated with Claude Pro/Max.</p>
        <p>You can now close this window and return to OpenCode.</p>
        <p><a href="/">Return to OpenCode</a></p>
    </div>
</body>
</html>`

	return c.WriteHTML(successHTML)
}

// exchangeCodeForTokens exchanges the authorization code for access/refresh tokens
func exchangeCodeForTokens(code, verifier string) (*TokenResponse, error) {
	// Prepare request body as JSON (matching TypeScript implementation)
	requestData := map[string]string{
		"code":          code,
		"state":         verifier,
		"grant_type":    "authorization_code",
		"client_id":     clientID,
		"redirect_uri":  redirectURI,
		"code_verifier": verifier,
	}

	jsonData, err := json.Marshal(requestData)
	if err != nil {
		return nil, serr.Wrap(err, "failed to marshal request data")
	}

	// Make POST request to token endpoint
	req, err := http.NewRequest("POST", tokenURL, bytes.NewReader(jsonData))
	if err != nil {
		return nil, serr.Wrap(err, "failed to create request")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, serr.Wrap(err, "failed to make token request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, serr.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, serr.New(fmt.Sprintf("token exchange failed: %s", string(body)))
	}

	var tokens TokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, serr.Wrap(err, "failed to parse token response")
	}

	// Calculate expiration time
	tokens.ExpiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)

	return &tokens, nil
}

// AnthropicRefreshHandler refreshes the access token
func AnthropicRefreshHandler(c rweb.Context) error {
	// Get current tokens from storage
	storedTokens, err := storage.GetAnthropicTokens()
	if err != nil {
		logger.LogErr(err, "failed to get tokens")
		return c.WriteJSON(map[string]string{"error": "no tokens found"})
	}

	tokens := &storedTokens.TokenResponse

	// Refresh the token
	newTokens, err := refreshToken(tokens.RefreshToken)
	if err != nil {
		logger.LogErr(err, "failed to refresh token")
		return c.WriteJSON(map[string]string{"error": "failed to refresh token"})
	}

	// Update stored tokens
	if err := storage.SaveAnthropicTokens(newTokens); err != nil {
		logger.LogErr(err, "failed to save refreshed tokens")
		return c.WriteJSON(map[string]string{"error": "failed to save tokens"})
	}

	return c.WriteJSON(map[string]string{"status": "token refreshed"})
}

// refreshToken refreshes the access token using the refresh token
func refreshToken(refreshToken string) (*TokenResponse, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {clientID},
	}

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, serr.Wrap(err, "failed to create request")
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, serr.Wrap(err, "failed to make refresh request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, serr.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, serr.New(fmt.Sprintf("token refresh failed: %s", string(body)))
	}

	var tokens TokenResponse
	if err := json.Unmarshal(body, &tokens); err != nil {
		return nil, serr.Wrap(err, "failed to parse token response")
	}

	// Calculate expiration time
	tokens.ExpiresAt = time.Now().Add(time.Duration(tokens.ExpiresIn) * time.Second)

	return &tokens, nil
}

// GetAccessToken returns a valid access token, refreshing if necessary
func GetAccessToken() (string, error) {
	storedTokens, err := storage.GetAnthropicTokens()
	if err != nil {
		return "", err
	}

	tokens := &storedTokens.TokenResponse

	// Check if token is expired or about to expire (5 minutes buffer)
	if time.Now().Add(5 * time.Minute).After(tokens.ExpiresAt) {
		// Refresh the token
		newTokens, err := refreshToken(tokens.RefreshToken)
		if err != nil {
			return "", serr.Wrap(err, "failed to refresh token")
		}

		// Update stored tokens
		if err := storage.SaveAnthropicTokens(newTokens); err != nil {
			return "", serr.Wrap(err, "failed to save refreshed tokens")
		}
		tokens = newTokens
	}

	return tokens.AccessToken, nil
}
