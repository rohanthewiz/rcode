package auth

import (
	"encoding/json"
	"strings"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
)

// ExchangeRequest represents the request to exchange code for tokens
type ExchangeRequest struct {
	Code string `json:"code"`
}

// AnthropicExchangeHandler handles the code exchange
func AnthropicExchangeHandler(c rweb.Context) error {
	// Parse request body
	body := c.Request().Body()
	var req ExchangeRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return c.WriteJSON(map[string]string{"error": "invalid request"})
	}

	// The code from Anthropic contains both code and state separated by #
	parts := strings.Split(req.Code, "#")
	if len(parts) != 2 {
		return c.WriteJSON(map[string]string{"error": "invalid code format"})
	}

	code := parts[0]
	state := parts[1]

	// Retrieve PKCE verifier using state
	verifierValue, ok := pkceStore.LoadAndDelete(state)
	if !ok {
		// The state is the verifier in the TypeScript implementation
		// So let's use the state as the verifier directly
		verifierValue = state
	}

	var verifier string
	switch v := verifierValue.(type) {
	case string:
		verifier = v
	case *PKCEChallenge:
		verifier = v.Verifier
	default:
		verifier = state
	}

	// Exchange code for tokens
	tokens, err := exchangeCodeForTokens(code, verifier)
	if err != nil {
		logger.LogErr(err, "failed to exchange code for tokens")
		return c.WriteJSON(map[string]string{"error": "failed to exchange code"})
	}

	// Store tokens persistently
	if err := storage.SaveAnthropicTokens(tokens); err != nil {
		logger.LogErr(err, "failed to save tokens")
		return c.WriteJSON(map[string]string{"error": "failed to save tokens"})
	}

	return c.WriteJSON(map[string]string{"status": "success"})
}
