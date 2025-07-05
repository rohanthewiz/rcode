package auth

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/url"

	"github.com/rohanthewiz/rweb"
)

// GetOAuthURLHandler returns the OAuth authorization URL as JSON
func GetOAuthURLHandler(c rweb.Context) error {
	// Generate PKCE challenge
	pkce, err := generatePKCE()
	if err != nil {
		return c.WriteJSON(map[string]string{"error": "failed to generate PKCE"})
	}

	// Generate state parameter for CSRF protection
	stateBytes := make([]byte, 16)
	if _, err := rand.Read(stateBytes); err != nil {
		return c.WriteJSON(map[string]string{"error": "failed to generate state"})
	}
	state := base64.RawURLEncoding.EncodeToString(stateBytes)

	// Store PKCE verifier for later use
	pkceStore.Store(state, pkce)

	// Build authorization URL
	params := url.Values{
		"code":                  {"true"},
		"client_id":             {clientID},
		"response_type":         {"code"},
		"redirect_uri":          {redirectURI},
		"scope":                 {scopes},
		"code_challenge":        {pkce.Challenge},
		"code_challenge_method": {pkce.Method},
		"state":                 {pkce.Verifier},
	}

	authURL := fmt.Sprintf("%s?%s", authorizeURL, params.Encode())

	// Return URL as JSON
	return c.WriteJSON(map[string]string{
		"url": authURL,
	})
}
