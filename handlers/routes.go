package handlers

import (
	"rcode/auth"

	"github.com/rohanthewiz/rweb"
)

// SetupRoutes configures all HTTP routes for the server
func SetupRoutes(s *rweb.Server) {
	// Root endpoint - serves the main web UI
	s.Get("/", rootHandler)

	// Auth endpoints
	s.Get("/auth/anthropic/authorize", auth.AnthropicAuthorizeHandler)
	s.Get("/auth/anthropic/oauth-url", auth.GetOAuthURLHandler)
	s.Get("/auth/anthropic/callback", auth.AnthropicCallbackHandler)
	s.Post("/auth/anthropic/exchange", auth.AnthropicExchangeHandler)
	s.Post("/auth/anthropic/refresh", auth.AnthropicRefreshHandler)
	s.Get("/auth/callback", AuthCallbackHandler)

	// Logout endpoint
	s.Post("/api/auth/logout", auth.LogoutHandler)

	// API endpoints
	s.Get("/api/app", appInfoHandler)
	s.Get("/api/session", listSessionsHandler)
	s.Post("/api/session", createSessionHandler)
	s.Delete("/api/session/:id", deleteSessionHandler)
	s.Post("/api/session/:id/message", sendMessageHandler)
	s.Get("/api/session/:id/messages", getSessionMessagesHandler)

	// SSE endpoint for streaming events
	s.Get("/events", eventsHandler)
}

// rootHandler serves the main web UI
func rootHandler(c rweb.Context) error {
	return UIHandler(c)
}

// appInfoHandler returns application information
func appInfoHandler(c rweb.Context) error {
	return c.WriteJSON(map[string]interface{}{
		"version":  "0.1.0",
		"status":   "ok",
		"provider": "anthropic",
		"model":    "claude-3-5-sonnet-20241022",
	})
}
