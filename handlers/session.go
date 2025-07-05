package handlers

import (
	"encoding/json"
	"strings"
	"time"

	"rcode/db"
	"rcode/providers"
	"rcode/tools"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

// Session represents a chat session (alias for db.Session for backward compatibility)
type Session = db.Session

// createSession creates a new chat session in the database
func createSession() (*Session, error) {
	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return nil, serr.Wrap(err, "failed to get database")
	}

	// Create session with default initial prompt
	session, err := database.CreateSession(db.SessionOptions{
		Title: "New Chat",
		InitialPrompts: []string{
			"Always ask before creating or writing files or using any tools",
		},
	})
	if err != nil {
		return nil, err
	}

	// Add the initial prompt as the first message
	initialPrompt := strings.Join(session.InitialPrompts, "\n")
	err = database.AddMessage(session.ID, providers.ChatMessage{
		Role:    "user",
		Content: initialPrompt,
	}, "", nil)
	if err != nil {
		logger.LogErr(err, "failed to add initial message")
	}

	return session, nil
}

// MessageRequest represents a request to send a message
type MessageRequest struct {
	Content string `json:"content"`
	Model   string `json:"model,omitempty"`
}

// Updated handlers with proper implementation

func listSessionsHandler(c rweb.Context) error {
	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// List sessions from database
	sessions, err := database.ListSessions()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to list sessions"), 500)
	}

	return c.WriteJSON(sessions)
}

func createSessionHandler(c rweb.Context) error {
	session, err := createSession()
	if err != nil {
		return c.WriteError(err, 500)
	}

	logger.F("Created new session: %s", session.ID)

	// Broadcast session list update
	BroadcastSessionList()

	return c.WriteJSON(session)
}

func deleteSessionHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Delete session from database
	err = database.DeleteSession(sessionID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to delete session"), 500)
	}

	// Broadcast session list update
	BroadcastSessionList()

	return c.WriteJSON(map[string]bool{"success": true})
}

func sendMessageHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")
	logger.Info("Sending message to session: " + sessionID)

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Get session from database
	session, err := database.GetSession(sessionID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get session"), 500)
	}
	if session == nil {
		logger.Info("Session not found for message: " + sessionID)
		return c.WriteError(serr.New("session not found"), 404)
	}

	// Parse request body
	body := c.Request().Body()
	var msgReq MessageRequest
	if err := json.Unmarshal(body, &msgReq); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}

	// Add user message to database
	userMsg := providers.ChatMessage{
		Role:    "user",
		Content: msgReq.Content,
	}
	err = database.AddMessage(sessionID, userMsg, "", nil)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to add user message"), 500)
	}

	// Get all messages for context
	messages, err := database.GetMessages(sessionID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get messages"), 500)
	}

	// Create Anthropic client and tool registry
	client := providers.NewAnthropicClient()
	toolRegistry := tools.DefaultRegistry()

	// Use the model from the request, or default to Claude Sonnet 4
	model := msgReq.Model
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	logger.Info("Using model", "model", model)

	// Get available tools
	availableTools := toolRegistry.GetTools()

	// Prepare request with tools
	systemPrompt := "You are Claude Code, Anthropic's official CLI for Claude."

	request := providers.CreateMessageRequest{
		Model:     model,
		Messages:  providers.ConvertToAPIMessages(messages),
		MaxTokens: 4096,
		Stream:    false,
		System:    systemPrompt,
		Tools:     availableTools,
	}

	// Keep trying until we get a final response (not a tool use)
	for {
		// Send message to Claude
		response, err := client.SendMessage(request)
		if err != nil {
			logger.LogErr(err, "failed to send message to Claude")
			return c.WriteError(err, 500)
		}

		// Check if response contains tool uses
		var hasToolUse bool
		var toolResults []interface{}

		for _, content := range response.Content {
			if content.Type == "tool_use" {
				hasToolUse = true

				// Parse the tool use
				toolUseData, _ := json.Marshal(content)
				var toolUse tools.ToolUse
				json.Unmarshal(toolUseData, &toolUse)

				logger.Info("Executing tool", "name", toolUse.Name)

				// Log tool usage (measure execution time)
				startTime := time.Now()

				// Execute the tool
				result, err := toolRegistry.Execute(toolUse)
				durationMs := int(time.Since(startTime).Milliseconds())
				
				// Log tool usage to database
				if logErr := database.LogToolUsage(sessionID, toolUse.Name, toolUse.Input, result.Content, durationMs, err); logErr != nil {
					logger.LogErr(logErr, "failed to log tool usage")
				}
				
				if err != nil {
					logger.LogErr(err, "tool execution failed")
				}

				// Add tool result to results
				toolResults = append(toolResults, result)
			}
		}

		// If there were tool uses, add assistant message and tool results, then continue
		if hasToolUse {
			// Add the assistant's message with tool uses to database
			assistantMsg := providers.ChatMessage{
				Role:    "assistant",
				Content: response.Content,
			}
			err = database.AddMessage(sessionID, assistantMsg, response.Model, &response.Usage)
			if err != nil {
				logger.LogErr(err, "failed to add assistant message with tool use")
			}

			// Add tool results as user message
			toolResultMsg := providers.ChatMessage{
				Role:    "user",
				Content: toolResults,
			}
			err = database.AddMessage(sessionID, toolResultMsg, "", nil)
			if err != nil {
				logger.LogErr(err, "failed to add tool result message")
			}

			// Get updated messages
			messages, err = database.GetMessages(sessionID)
			if err != nil {
				return c.WriteError(serr.Wrap(err, "failed to get updated messages"), 500)
			}

			// Update request with new messages and continue
			request.Messages = providers.ConvertToAPIMessages(messages)
			continue
		}

		// No tool use, extract text content from response
		var responseText string
		for _, content := range response.Content {
			if content.Type == "text" {
				responseText += content.Text
			}
		}

		// Add assistant message to database
		assistantMsg := providers.ChatMessage{
			Role:    "assistant",
			Content: responseText,
		}
		err = database.AddMessage(sessionID, assistantMsg, response.Model, &response.Usage)
		if err != nil {
			logger.LogErr(err, "failed to add assistant message")
		}

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
}

// Add a handler to get messages for a session
func getSessionMessagesHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")

	logger.Info("Getting messages for session: " + sessionID)

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Get messages from database
	messages, err := database.GetMessages(sessionID)
	if err != nil {
		logger.LogErr(err, "failed to get messages")
		// Return empty array instead of error to avoid breaking the UI
		return c.WriteJSON([]providers.ChatMessage{})
	}

	logger.F("Found session with messages: %d", len(messages))
	return c.WriteJSON(messages)
}
