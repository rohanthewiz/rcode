package web

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
	s.Get("/api/session/:id/prompts", getSessionPromptsHandler)

	// Prompt management endpoints
	s.Get("/api/prompts", listPromptsHandler)
	s.Get("/api/prompts/:id", getPromptHandler)
	s.Post("/api/prompts", createPromptHandler)
	s.Put("/api/prompts/:id", updatePromptHandler)
	s.Delete("/api/prompts/:id", deletePromptHandler)

	// Context management endpoints
	s.Get("/api/context", getProjectContextHandler)
	s.Post("/api/context/initialize", initializeProjectContextHandler)
	s.Post("/api/context/relevant-files", getRelevantFilesHandler)
	s.Get("/api/context/changes", getChangeTrackingHandler)
	s.Get("/api/context/stats", getContextStatsHandler)
	s.Post("/api/context/suggest-tools", suggestToolsHandler)

	// SSE endpoint for streaming events
	s.Get("/events", eventsHandler)

	// Prompt Manager UI
	s.Get("/prompts", PromptManagerHandler)
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
