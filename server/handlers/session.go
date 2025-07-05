package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/opencodesdev/opencode/server/providers"
	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

// Session represents a chat session
type Session struct {
	ID        string                  `json:"id"`
	Title     string                  `json:"title"`
	Messages  []providers.ChatMessage `json:"messages"`
	CreatedAt time.Time               `json:"created_at"`
	UpdatedAt time.Time               `json:"updated_at"`
}

// In-memory session storage (for now)
var sessions = make(map[string]*Session)

// createSession creates a new chat session
func createSession() *Session {
	session := &Session{
		ID:        fmt.Sprintf("session-%d", time.Now().Unix()),
		Title:     "New Chat",
		Messages:  []providers.ChatMessage{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	sessions[session.ID] = session
	return session
}

// MessageRequest represents a request to send a message
type MessageRequest struct {
	Content string `json:"content"`
	Model   string `json:"model,omitempty"`
}

// Updated handlers with proper implementation

func listSessionsHandler(c rweb.Context) error {
	sessionList := make([]*Session, 0, len(sessions))
	for _, session := range sessions {
		sessionList = append(sessionList, session)
	}
	return c.WriteJSON(sessionList)
}

func createSessionHandler(c rweb.Context) error {
	session := createSession()
	logger.F("Created new session: %s", session.ID)

	// Broadcast session list update
	BroadcastSessionList()

	return c.WriteJSON(session)
}

func deleteSessionHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")
	delete(sessions, sessionID)

	// Broadcast session list update
	BroadcastSessionList()

	return c.WriteJSON(map[string]bool{"success": true})
}

func sendMessageHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")
	logger.Info("Sending message to session: " + sessionID)

	// Get session
	session, exists := sessions[sessionID]
	if !exists {
		logger.Info("Session not found for message: " + sessionID)
		return c.WriteError(serr.New("session not found"), 404)
	}

	// Parse request body
	body := c.Request().Body()
	var msgReq MessageRequest
	if err := json.Unmarshal(body, &msgReq); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}

	// Add user message to session
	userMsg := providers.ChatMessage{
		Role:    "user",
		Content: msgReq.Content,
	}
	session.Messages = append(session.Messages, userMsg)
	session.UpdatedAt = time.Now()

	// Create Anthropic client
	client := providers.NewAnthropicClient()

	// Use the model from the request, or default to Claude Sonnet 4
	model := msgReq.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	logger.Info("Using model", "model", model)

	// Prepare request - Anthropic API expects system in a separate field
	// IMPORTANT: Must use exact system prompt for OAuth authentication
	systemPrompt := "You are Claude Code, Anthropic's official CLI for Claude."

	request := providers.CreateMessageRequest{
		Model:     model,
		Messages:  providers.ConvertToAPIMessages(session.Messages),
		MaxTokens: 4096,
		Stream:    false,
		System:    systemPrompt,
	}

	// Send message to Claude
	response, err := client.SendMessage(request)
	if err != nil {
		logger.LogErr(err, "failed to send message to Claude")
		return c.WriteError(err, 500)
	}

	// Extract text content from response
	var responseText string
	for _, content := range response.Content {
		if content.Type == "text" {
			responseText += content.Text
		}
	}

	// Add assistant message to session
	assistantMsg := providers.ChatMessage{
		Role:    "assistant",
		Content: responseText,
	}
	session.Messages = append(session.Messages, assistantMsg)

	// Broadcast the assistant's message
	BroadcastMessage(sessionID, map[string]interface{}{
		"role":    "assistant",
		"content": responseText,
		"model":   response.Model,
	})

	// Return the assistant's response with model info
	return c.WriteJSON(map[string]interface{}{
		"role":    "assistant",
		"content": responseText,
		"usage":   response.Usage,
		"model":   response.Model,
	})
}

// Add a handler to get messages for a session
func getSessionMessagesHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")

	logger.Info("Getting messages for session: " + sessionID)
	logger.F("Number of active sessions: %d", len(sessions))

	session, exists := sessions[sessionID]
	if !exists {
		logger.Info("Session not found: " + sessionID)
		// Return empty array instead of 404 to avoid breaking the UI
		return c.WriteJSON([]providers.ChatMessage{})
	}

	logger.F("Found session with messages: %d", len(session.Messages))
	return c.WriteJSON(session.Messages)
}
