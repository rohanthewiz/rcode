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

	// Tool permissions endpoints
	s.Get("/api/session/:id/tools", getSessionToolsHandler)
	s.Put("/api/session/:id/tools/:tool", updateToolPermissionHandler)

	// Permission response endpoint
	s.Post("/api/permission-response", handlePermissionResponseHandler)

	// Context management endpoints
	s.Get("/api/context", getProjectContextHandler)
	s.Post("/api/context/initialize", initializeProjectContextHandler)
	s.Post("/api/context/relevant-files", getRelevantFilesHandler)
	s.Get("/api/context/changes", getChangeTrackingHandler)
	s.Get("/api/context/stats", getContextStatsHandler)
	s.Post("/api/context/suggest-tools", suggestToolsHandler)

	// Task planning endpoints
	s.Post("/api/session/:id/plan", createPlanHandler)
	s.Get("/api/session/:id/plans", listPlansHandler)
	s.Post("/api/plan/:id/execute", executePlanHandler)
	s.Get("/api/plan/:id/status", getPlanStatusHandler)
	s.Post("/api/plan/:id/rollback", rollbackPlanHandler)
	s.Get("/api/plan/:id/checkpoints", listCheckpointsHandler)
	s.Get("/api/plan/:id/analyze", analyzePlanHandler)
	s.Get("/api/plan/:id/git-operations", getGitOperationsHandler)

	// Plan history endpoints
	s.Get("/api/session/:id/plans/history", listPlanHistoryHandler)
	s.Get("/api/plan/:id/full", getPlanFullDetailsHandler)
	s.Post("/api/plan/:id/clone", clonePlanHandler)
	s.Delete("/api/plan/:id", deletePlanHandler)

	// SSE endpoint for streaming events
	s.Get("/events",
		func(c rweb.Context) error {

			// Create client channel
			clientChan := make(chan any, 10)
			sseHub.Register(clientChan)

			// We cannot unregister here become the conn is long-lived
			// // Ensure cleanup on disconnect
			// defer func() {
			// 	sseHub.Unregister(clientChan)
			// }()

			s.SetupSSE(c, clientChan, "")

			return nil
		},
	)

	// Prompt Manager UI
	s.Get("/prompts", PromptManagerHandler)

	// File Explorer endpoints
	s.Get("/api/files/tree", getFileTreeHandler)
	s.Get("/api/files/content/:path", getFileContentHandler)
	s.Post("/api/files/search", searchFilesHandler)
	s.Post("/api/session/:id/files/open", openFileHandler)
	s.Post("/api/session/:id/files/close", closeFileInSessionHandler)
	s.Get("/api/session/:id/files/recent", getRecentFilesHandler)
	s.Get("/api/session/:id/files/open", getSessionOpenFilesHandler)

	// Diff visualization endpoints
	s.Get("/api/diff/:sessionId/:path", getDiffHandler)
	s.Post("/api/diff/snapshot", createSnapshotHandler)
	s.Post("/api/diff/generate", generateDiffHandler)
	s.Get("/api/session/:id/diffs", listSessionDiffsHandler)
	s.Get("/api/diff/:id", getDiffByIdHandler)
	s.Post("/api/diff/:id/viewed", markDiffViewedHandler)
	s.Get("/api/diff/preferences", getDiffPreferencesHandler)
	s.Post("/api/diff/preferences", saveDiffPreferencesHandler)
	s.Post("/api/diff/apply", applyDiffHandler)
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
