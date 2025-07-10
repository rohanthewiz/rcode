package web

import (
	"encoding/json"
	"os"
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

// CreateSessionRequest represents a request to create a session
type CreateSessionRequest struct {
	Title            string `json:"title,omitempty"`
	InitialPromptIDs []int  `json:"initial_prompt_ids,omitempty"`
	ModelPreference  string `json:"model_preference,omitempty"`
}

// createSession creates a new chat session in the database
func createSession(req *CreateSessionRequest) (*Session, error) {
	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return nil, serr.Wrap(err, "failed to get database")
	}

	// Prepare session options
	opts := db.SessionOptions{
		Title:            req.Title,
		InitialPromptIDs: req.InitialPromptIDs,
		ModelPreference:  req.ModelPreference,
	}

	// If no title provided, it will default to "New Chat" in CreateSession
	// If no prompt IDs provided, it will use default prompts

	// Create session (this will handle loading prompts and permissions)
	session, err := database.CreateSession(opts)
	if err != nil {
		return nil, err
	}

	// Add the initial prompts as the first message if any exist
	if len(session.InitialPrompts) > 0 {
		initialPrompt := strings.Join(session.InitialPrompts, "\n")
		err = database.AddMessage(session.ID, providers.ChatMessage{
			Role:    "user",
			Content: initialPrompt,
		}, "", nil)
		if err != nil {
			logger.LogErr(err, "failed to add initial message")
		}
	}

	return session, nil
}

// MessageRequest represents a request to send a message
type MessageRequest struct {
	Content string `json:"content"`
	Model   string `json:"model,omitempty"`
}

// generateSessionTitle creates a friendly title from the first user message
func generateSessionTitle(content string) string {
	// Trim whitespace
	content = strings.TrimSpace(content)

	// If content is too short, use default
	if len(content) < 3 {
		return "New Chat"
	}

	// Take first line only (in case of multi-line messages)
	lines := strings.Split(content, "\n")
	title := lines[0]

	// Remove any leading command-like prefixes
	title = strings.TrimPrefix(title, "/")
	title = strings.TrimSpace(title)

	// Limit length and add ellipsis if needed
	maxLength := 50
	if len(title) > maxLength {
		// Try to cut at a word boundary
		if idx := strings.LastIndexAny(title[:maxLength-3], " .,!?"); idx > maxLength/2 {
			title = title[:idx] + "..."
		} else {
			title = title[:maxLength-3] + "..."
		}
	}

	return title
}

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
	// Parse request body if provided
	var req CreateSessionRequest
	body := c.Request().Body()
	if len(body) > 0 {
		if err := json.Unmarshal(body, &req); err != nil {
			// If parsing fails, just use defaults
			logger.LogErr(err, "failed to parse create session request")
		}
	}

	session, err := createSession(&req)
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

	// Check if this is the first user message (after initial prompt)
	// and update session title if needed
	messageCount, err := database.GetMessageCount(sessionID)
	if err != nil {
		logger.LogErr(err, "failed to get message count")
	} else if messageCount == 2 && session.Title == "New Chat" {
		// This is the first real user message, generate a title
		newTitle := generateSessionTitle(msgReq.Content)
		if err := database.UpdateSession(sessionID, newTitle, session.Metadata); err != nil {
			logger.LogErr(err, "failed to update session title")
		} else {
			logger.Info("Updated session title", "session_id", sessionID, "title", newTitle)
			// Broadcast session list update so UI refreshes
			BroadcastSessionList()
		}
	}

	// Get all messages for context
	messages, err := database.GetMessages(sessionID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get messages"), 500)
	}

	// Create Anthropic client and tool registry
	client := providers.NewAnthropicClient()
	toolRegistry := tools.DefaultRegistry()
	
	// Initialize context if not already done
	if !client.GetContextManager().IsInitialized() {
		workDir, err := os.Getwd()
		if err != nil {
			logger.LogErr(err, "failed to get working directory")
			workDir = "."
		}
		if err := client.InitializeContext(workDir); err != nil {
			logger.LogErr(err, "failed to initialize context")
		}
	}
	
	// Create context-aware tool executor
	contextExecutor := tools.NewContextAwareExecutor(toolRegistry, client.GetContextManager())

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
	
	// Enhance system prompt with context
	systemPrompt = client.EnhanceSystemPromptWithContext(systemPrompt)

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
				err = json.Unmarshal(toolUseData, &toolUse)
				if err != nil {
					logger.Warn("Error unmarshalling tool_use data", "error", err, "contentName", content.Name)
					continue
				}

				logger.Info("Executing tool", "name", toolUse.Name)

				// Log tool usage (measure execution time)
				startTime := time.Now()

				// Execute the tool with context awareness
				result, err := contextExecutor.Execute(toolUse)
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
