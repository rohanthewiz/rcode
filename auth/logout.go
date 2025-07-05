package auth

import (
	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
)

// LogoutHandler handles user logout by removing stored tokens
func LogoutHandler(c rweb.Context) error {
	// Remove Anthropic tokens from storage
	if err := storage.RemoveAnthropicTokens(); err != nil {
		logger.LogErr(err, "failed to remove tokens during logout")
		// Continue with logout even if token removal fails
	}

	// Return success response
	return c.WriteJSON(map[string]string{
		"status":  "success",
		"message": "Logged out successfully",
	})
}
