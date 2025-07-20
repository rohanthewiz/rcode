package web

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// getContextPrompt returns context information as an initial prompt
func getContextPrompt() string {
	cm := GetContextManager()
	if cm == nil || !cm.IsInitialized() {
		return ""
	}
	
	ctx := cm.GetContext()
	if ctx == nil {
		return ""
	}
	
	// Build context information as a user prompt
	var contextInfo strings.Builder
	contextInfo.WriteString("Project Context Information:\n")
	contextInfo.WriteString(fmt.Sprintf("- Working in a %s project", ctx.Language))
	if ctx.Framework != "" {
		contextInfo.WriteString(fmt.Sprintf(" using %s framework", ctx.Framework))
	}
	contextInfo.WriteString(fmt.Sprintf("\n- Project root: %s", ctx.RootPath))
	
	if ctx.Statistics.TotalFiles > 0 {
		contextInfo.WriteString(fmt.Sprintf("\n- Total files: %d (%d lines of code)", 
			ctx.Statistics.TotalFiles, ctx.Statistics.TotalLines))
	}
	
	// Add file type breakdown if available
	if len(ctx.Statistics.FilesByLanguage) > 0 {
		contextInfo.WriteString("\n- File types:")
		for lang, count := range ctx.Statistics.FilesByLanguage {
			if count > 0 {
				contextInfo.WriteString(fmt.Sprintf(" %s(%d)", lang, count))
			}
		}
	}
	
	return contextInfo.String()
}

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
		
		// Add context information if available
		contextInfo := getContextPrompt()
		if contextInfo != "" {
			initialPrompt = initialPrompt + "\n\n" + contextInfo
		}
		
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
	
	// Track tool usage for this request
	var toolSummaries []string

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

	// Create Anthropic client
	client := providers.NewAnthropicClient()
	
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
	
	// Create tool registry with custom tools support
	workDir, err := os.Getwd()
	if err != nil {
		logger.LogErr(err, "failed to get working directory for tools")
		workDir = "."
	}
	toolRegistry, err := tools.DefaultRegistryWithPlugins(workDir)
	if err != nil {
		logger.LogErr(err, "failed to create tool registry with plugins")
		// Fall back to default registry
		toolRegistry = tools.DefaultRegistry()
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
		// Send message to Claude with retry for transient errors
		response, err := client.SendMessageWithRetry(request)
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

				// Add session ID to tool input for diff tracking
				toolUse.Input["_sessionId"] = sessionID

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

				// Create and broadcast tool usage summary
				summary := createToolSummary(toolUse.Name, toolUse.Input, result.Content, err)
				logger.Info("Broadcasting tool usage", "tool", toolUse.Name, "summary", summary)
				BroadcastToolUsage(sessionID, toolUse.Name, summary)
				
				// Collect tool summaries for response
				toolSummaries = append(toolSummaries, summary)

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

		// Return the assistant's response with model info and tool summaries
		return c.WriteJSON(map[string]interface{}{
			"role":          "assistant",
			"content":       responseText,
			"usage":         response.Usage,
			"model":         response.Model,
			"toolSummaries": toolSummaries,
		})
	}
}

// createToolSummary creates a concise summary of tool usage
func createToolSummary(toolName string, input map[string]interface{}, result string, err error) string {
	if err != nil {
		return fmt.Sprintf("❌ Failed: %s", err.Error())
	}

	switch toolName {
	case "write_file":
		if path, ok := tools.GetString(input, "path"); ok {
			// Count bytes written
			if content, ok := tools.GetString(input, "content"); ok {
				bytes := len([]byte(content))
				return fmt.Sprintf("✓ Wrote %s (%d bytes)", filepath.Base(path), bytes)
			}
			return fmt.Sprintf("✓ Wrote %s", filepath.Base(path))
		}

	case "edit_file":
		if path, ok := tools.GetString(input, "path"); ok {
			if startLine, ok := tools.GetInt(input, "start_line"); ok {
				if endLine, ok := tools.GetInt(input, "end_line"); ok && endLine > startLine {
					return fmt.Sprintf("✓ Edited %s (lines %d-%d)", filepath.Base(path), startLine, endLine)
				}
				return fmt.Sprintf("✓ Edited %s (line %d)", filepath.Base(path), startLine)
			}
			return fmt.Sprintf("✓ Edited %s", filepath.Base(path))
		}

	case "read_file":
		if path, ok := tools.GetString(input, "path"); ok {
			// Extract line count from result if available
			lines := strings.Count(result, "\n")
			if lines > 0 {
				return fmt.Sprintf("✓ Read %s (%d lines)", filepath.Base(path), lines)
			}
			return fmt.Sprintf("✓ Read %s", filepath.Base(path))
		}

	case "bash":
		if cmd, ok := tools.GetString(input, "command"); ok {
			// Truncate long commands
			if len(cmd) > 50 {
				cmd = cmd[:47] + "..."
			}
			return fmt.Sprintf("✓ Ran: %s", cmd)
		}

	case "search":
		if pattern, ok := tools.GetString(input, "pattern"); ok {
			// Count matches in result
			matches := strings.Count(result, "Match")
			if matches > 0 {
				return fmt.Sprintf("✓ Found %d matches for '%s'", matches, pattern)
			}
			return fmt.Sprintf("✓ Searched for '%s'", pattern)
		}

	case "list_dir":
		if path, ok := tools.GetString(input, "path"); ok {
			// Count items in result
			lines := strings.Split(result, "\n")
			count := 0
			for _, line := range lines {
				if strings.TrimSpace(line) != "" {
					count++
				}
			}
			return fmt.Sprintf("✓ Listed %s (%d items)", filepath.Base(path), count)
		}

	case "make_dir":
		if path, ok := tools.GetString(input, "path"); ok {
			return fmt.Sprintf("✓ Created directory %s", filepath.Base(path))
		}

	case "remove":
		if path, ok := tools.GetString(input, "path"); ok {
			return fmt.Sprintf("✓ Removed %s", filepath.Base(path))
		}

	case "move":
		if src, ok := tools.GetString(input, "source"); ok {
			if dst, ok := tools.GetString(input, "destination"); ok {
				return fmt.Sprintf("✓ Moved %s → %s", filepath.Base(src), filepath.Base(dst))
			}
		}

	case "tree":
		// Count lines in tree output
		lines := strings.Count(result, "\n")
		return fmt.Sprintf("✓ Generated tree (%d lines)", lines)

	case "git_status":
		// Check for clean/dirty status
		if strings.Contains(result, "nothing to commit") {
			return "✓ Git status: clean"
		}
		return "✓ Git status: changes detected"

	case "git_diff":
		// Count changed files
		changes := strings.Count(result, "+++")
		if changes > 0 {
			return fmt.Sprintf("✓ Git diff: %d files changed", changes)
		}
		return "✓ Git diff: no changes"

	case "git_log":
		// Count commits shown
		commits := strings.Count(result, "commit ")
		return fmt.Sprintf("✓ Git log: %d commits", commits)

	case "git_branch":
		// Count branches
		branches := strings.Count(result, "\n") + 1
		return fmt.Sprintf("✓ Git branches: %d total", branches)
	}

	// Default summary
	return fmt.Sprintf("✓ Executed %s", toolName)
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
