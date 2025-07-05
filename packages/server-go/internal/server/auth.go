package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/sst/opencode/server-go/internal/auth"
)

// authStartHandler initiates OAuth flow for a provider
func (s *Server) authStartHandler(c rweb.Context) error {
	provider := c.Param("provider")
	
	switch provider {
	case "anthropic":
		anthropicAuth := auth.NewAnthropicAuth()
		result, err := anthropicAuth.AuthorizeURL()
		if err != nil {
			logger.LogErr(err, "failed to generate auth URL")
			return c.JSON(500, map[string]string{"error": "Failed to start authentication"})
		}

		// Open browser
		if err := openBrowser(result.URL); err != nil {
			logger.LogErr(err, "failed to open browser")
		}

		// Store verifier in session for later use
		s.authSessions.Store(result.Verifier, time.Now().Add(10 * time.Minute))

		return c.JSON(200, map[string]interface{}{
			"url":      result.URL,
			"verifier": result.Verifier,
			"message":  "Please complete authentication in your browser",
		})

	default:
		return c.JSON(400, map[string]string{"error": "Unsupported provider"})
	}
}

// authCallbackHandler handles OAuth callback
func (s *Server) authCallbackHandler(c rweb.Context) error {
	provider := c.Param("provider")
	
	// Parse request body
	var req struct {
		Code     string `json:"code"`
		Verifier string `json:"verifier"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request"})
	}

	// Verify the verifier exists and hasn't expired
	if stored, ok := s.authSessions.Load(req.Verifier); ok {
		if expiry, ok := stored.(time.Time); ok && time.Now().Before(expiry) {
			s.authSessions.Delete(req.Verifier)
		} else {
			return c.JSON(400, map[string]string{"error": "Verifier expired"})
		}
	} else {
		return c.JSON(400, map[string]string{"error": "Invalid verifier"})
	}

	switch provider {
	case "anthropic":
		anthropicAuth := auth.NewAnthropicAuth()
		if err := anthropicAuth.Exchange(req.Code, req.Verifier); err != nil {
			logger.LogErr(err, "failed to exchange code")
			return c.JSON(500, map[string]string{"error": "Failed to complete authentication"})
		}

		// Update provider to use OAuth
		s.updateAnthropicProvider()

		return c.JSON(200, map[string]interface{}{
			"success": true,
			"message": "Successfully authenticated with Anthropic Pro/Max",
		})

	default:
		return c.JSON(400, map[string]string{"error": "Unsupported provider"})
	}
}

// authStatusHandler returns authentication status
func (s *Server) authStatusHandler(c rweb.Context) error {
	storage := auth.NewStorage()
	providers, err := storage.List()
	if err != nil {
		logger.LogErr(err, "failed to list auth providers")
		return c.JSON(500, map[string]string{"error": "Failed to get auth status"})
	}

	status := make(map[string]interface{})
	for _, provider := range providers {
		creds, err := storage.Get(provider)
		if err != nil {
			continue
		}

		providerStatus := map[string]interface{}{
			"authenticated": true,
			"type":          creds.Type,
		}

		// For OAuth, check if token is expired
		if creds.Type == "oauth" {
			providerStatus["expired"] = time.Now().Unix() > creds.Expires
		}

		status[provider] = providerStatus
	}

	return c.JSON(200, status)
}

// authLogoutHandler logs out from a provider
func (s *Server) authLogoutHandler(c rweb.Context) error {
	provider := c.Param("provider")
	
	switch provider {
	case "anthropic":
		anthropicAuth := auth.NewAnthropicAuth()
		if err := anthropicAuth.Logout(); err != nil {
			logger.LogErr(err, "failed to logout")
			return c.JSON(500, map[string]string{"error": "Failed to logout"})
		}

		// Revert to API key if available
		s.updateAnthropicProvider()

		return c.JSON(200, map[string]interface{}{
			"success": true,
			"message": fmt.Sprintf("Logged out from %s", provider),
		})

	default:
		return c.JSON(400, map[string]string{"error": "Unsupported provider"})
	}
}

// updateAnthropicProvider updates the Anthropic provider based on available auth
func (s *Server) updateAnthropicProvider() {
	// This will be called after auth changes to update the provider
	// The provider will check for OAuth tokens first, then fall back to API key
	logger.Info("Updated Anthropic provider authentication")
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "linux":
		cmd = "xdg-open"
		args = []string{url}
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", url}
	default:
		return fmt.Errorf("unsupported platform")
	}

	return exec.Command(cmd, args...).Start()
}

// Add auth routes to setupRoutes
func (s *Server) setupAuthRoutes() {
	// Authentication endpoints
	s.srv.Post("/auth/:provider/start", s.authStartHandler)
	s.srv.Post("/auth/:provider/callback", s.authCallbackHandler)
	s.srv.Get("/auth/status", s.authStatusHandler)
	s.srv.Post("/auth/:provider/logout", s.authLogoutHandler)
}