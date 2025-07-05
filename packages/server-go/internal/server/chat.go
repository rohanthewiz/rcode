package server

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
	"github.com/sst/opencode/server-go/internal/provider/anthropic"
	"github.com/sst/opencode/server-go/internal/tool"
)

// ChatRequest represents a chat API request
type ChatRequest struct {
	Parts      []MessagePartParam `json:"parts"`
	ProviderID string             `json:"providerID"`
	ModelID    string             `json:"modelID"`
}

// MessagePartParam represents input message parts
type MessagePartParam struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ChatHandler handles the main chat endpoint with Anthropic integration
func (s *Server) ChatHandler(c rweb.Context) error {
	sessionID := c.Param("id")

	// Parse request
	var req ChatRequest
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "invalid request"})
	}

	// Get session
	sessionVal, exists := s.sessions.Load(sessionID)
	if !exists {
		return c.JSON(404, map[string]string{"error": "session not found"})
	}
	session := sessionVal.(*Session)

	// Build user message from parts
	var textParts []string
	for _, part := range req.Parts {
		if part.Type == "text" && part.Text != "" {
			textParts = append(textParts, part.Text)
		}
	}
	userContent := strings.Join(textParts, " ")

	// Add user message to session
	userMessage := Message{
		ID:      generateID(),
		Role:    "user",
		Content: userContent,
	}

	session.mu.Lock()
	session.Messages = append(session.Messages, userMessage)
	messages := append([]Message{}, session.Messages...) // Copy for thread safety
	session.mu.Unlock()

	// Publish user message event
	s.eventBus.Publish(Event{
		Type: "message.created",
		Data: map[string]interface{}{
			"sessionID": sessionID,
			"message":   userMessage,
		},
	})

	// Process with Anthropic provider
	go s.processWithAnthropic(session, messages, req.ProviderID, req.ModelID)

	return c.JSON(200, map[string]string{
		"status":    "processing",
		"messageID": userMessage.ID,
	})
}

// processWithAnthropic handles the actual Anthropic API call and streaming
func (s *Server) processWithAnthropic(session *Session, messages []Message, providerID, modelID string) {
	// Get API key
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		logger.Error("ANTHROPIC_API_KEY not set")
		s.publishError(session.ID, "API key not configured")
		return
	}

	// Create Anthropic provider
	provider := anthropic.NewProvider(apiKey)

	// Convert messages to Anthropic format
	anthropicMessages := make([]anthropic.Message, 0, len(messages))
	for _, msg := range messages {
		anthropicMessages = append(anthropicMessages, anthropic.Message{
			Role: msg.Role,
			Content: anthropic.TextContent{
				Type: "text",
				Text: msg.Content,
			},
		})
	}

	// Create assistant message to track the response
	assistantMessage := Message{
		ID:       generateID(),
		Role:     "assistant",
		Content:  "",
		Parts:    []MessagePart{},
		Metadata: map[string]interface{}{"tools": map[string]interface{}{}},
	}

	// Add to session
	session.mu.Lock()
	session.Messages = append(session.Messages, assistantMessage)
	msgIndex := len(session.Messages) - 1
	session.mu.Unlock()

	// Publish assistant message creation
	s.eventBus.Publish(Event{
		Type: "message.created",
		Data: map[string]interface{}{
			"sessionID": session.ID,
			"message":   assistantMessage,
		},
	})

	// Map model names
	model := modelID
	if model == "" {
		model = "claude-3-opus-20240229"
	}

	// Make streaming request
	response, err := provider.Chat(anthropicMessages, anthropic.ChatOptions{
		Model:     model,
		MaxTokens: 4096,
		Stream:    true,
		System:    "You are a helpful AI assistant with access to tools for reading files, executing commands, and more.",
	})

	if err != nil {
		logger.LogErr(err, "failed to start chat")
		s.publishError(session.ID, fmt.Sprintf("Chat error: %v", err))
		return
	}

	// Process streaming response
	processor := &anthropic.StreamProcessor{
		Provider: provider,
		OnText: func(text string) {
			// Accumulate text
			assistantMessage.Content += text

			// Add or update text part
			if len(assistantMessage.Parts) == 0 || assistantMessage.Parts[len(assistantMessage.Parts)-1].Type != "text" {
				assistantMessage.Parts = append(assistantMessage.Parts, MessagePart{
					Type: "text",
					Text: text,
				})
			} else {
				assistantMessage.Parts[len(assistantMessage.Parts)-1].Text += text
			}

			// Update session
			session.mu.Lock()
			session.Messages[msgIndex] = assistantMessage
			session.mu.Unlock()

			// Publish update
			s.eventBus.Publish(Event{
				Type: "message.chunk",
				Data: map[string]interface{}{
					"sessionID": session.ID,
					"messageID": assistantMessage.ID,
					"content":   text,
				},
			})
		},
		OnToolUse: func(toolUse anthropic.ToolUseContent) {
			// Add tool use part
			assistantMessage.Parts = append(assistantMessage.Parts, MessagePart{
				Type: "toolUse",
				ToolUse: &ToolUse{
					ID:         toolUse.ID,
					Name:       toolUse.Name,
					Parameters: toolUse.Input,
				},
			})

			// Publish tool use event
			s.eventBus.Publish(Event{
				Type: "tool.start",
				Data: map[string]interface{}{
					"sessionID": session.ID,
					"messageID": assistantMessage.ID,
					"toolID":    toolUse.ID,
					"toolName":  toolUse.Name,
					"input":     toolUse.Input,
				},
			})

			// Execute tool
			go s.executeToolForMessage(session, &assistantMessage, toolUse)
		},
		OnToolResult: func(result anthropic.ToolResultContent) {
			// Add tool result part
			assistantMessage.Parts = append(assistantMessage.Parts, MessagePart{
				Type: "toolResult",
				ToolResult: &ToolResult{
					ID:      result.ToolUseID,
					Content: result.Content,
				},
			})

			// Update metadata
			if assistantMessage.Metadata["tools"] == nil {
				assistantMessage.Metadata["tools"] = map[string]interface{}{}
			}
			assistantMessage.Metadata["tools"].(map[string]interface{})[result.ToolUseID] = map[string]interface{}{
				"status": "completed",
				"output": result.Content,
			}

			// Update session
			session.mu.Lock()
			session.Messages[msgIndex] = assistantMessage
			session.mu.Unlock()

			// Publish tool result event
			s.eventBus.Publish(Event{
				Type: "tool.result",
				Data: map[string]interface{}{
					"sessionID": session.ID,
					"messageID": assistantMessage.ID,
					"toolID":    result.ToolUseID,
					"result":    result.Content,
				},
			})
		},
		OnError: func(err error) {
			logger.LogErr(err, "stream processing error")
			s.publishError(session.ID, fmt.Sprintf("Stream error: %v", err))
		},
	}

	// Process the stream
	if err := processor.ProcessStream(response.Stream); err != nil {
		logger.LogErr(err, "failed to process stream")
		s.publishError(session.ID, fmt.Sprintf("Processing error: %v", err))
		return
	}

	// Final update
	session.mu.Lock()
	session.Messages[msgIndex] = assistantMessage
	session.mu.Unlock()

	// Publish completion
	s.eventBus.Publish(Event{
		Type: "message.completed",
		Data: map[string]interface{}{
			"sessionID": session.ID,
			"message":   assistantMessage,
		},
	})
}

// executeToolForMessage executes a tool and updates the message
func (s *Server) executeToolForMessage(session *Session, message *Message, toolUse anthropic.ToolUseContent) {
	// Create tool context
	ctx := tool.NewContext(tool.ContextOptions{
		CWD: "/", // Use root as default, could be configured
		Metadata: func(data map[string]any) {
			// Update tool metadata
			if message.Metadata["tools"] == nil {
				message.Metadata["tools"] = map[string]interface{}{}
			}
			toolMeta := message.Metadata["tools"].(map[string]interface{})
			if toolMeta[toolUse.ID] == nil {
				toolMeta[toolUse.ID] = map[string]interface{}{}
			}
			
			// Merge metadata
			currentMeta := toolMeta[toolUse.ID].(map[string]interface{})
			for k, v := range data {
				currentMeta[k] = v
			}

			// Publish metadata update
			s.eventBus.Publish(Event{
				Type: "tool.metadata",
				Data: map[string]interface{}{
					"sessionID": session.ID,
					"messageID": message.ID,
					"toolID":    toolUse.ID,
					"metadata":  data,
				},
			})
		},
	})

	// Execute tool
	result, err := s.toolRegistry.Get(toolUse.Name).Execute(ctx, toolUse.Input)

	// Build result content
	var resultContent string
	if err != nil {
		resultContent = fmt.Sprintf("Error: %v", err)
	} else {
		resultContent = result.Output
	}

	// The StreamProcessor will handle adding the result to the message
	// This is just for immediate feedback
	s.eventBus.Publish(Event{
		Type: "tool.executed",
		Data: map[string]interface{}{
			"sessionID": session.ID,
			"messageID": message.ID,
			"toolID":    toolUse.ID,
			"success":   err == nil,
			"result":    resultContent,
		},
	})
}

// publishError publishes an error event
func (s *Server) publishError(sessionID string, errorMsg string) {
	s.eventBus.Publish(Event{
		Type: "error",
		Data: map[string]interface{}{
			"sessionID": sessionID,
			"error":     errorMsg,
		},
	})
}

// UpdatedStreamProcessor extends the Anthropic StreamProcessor
type UpdatedStreamProcessor struct {
	*anthropic.StreamProcessor
	server    *Server
	sessionID string
	messageID string
}

// ProcessStreamWithServer processes stream with server integration
func (s *Server) ProcessStreamWithServer(reader io.ReadCloser, sessionID, messageID string) error {
	processor := &UpdatedStreamProcessor{
		StreamProcessor: &anthropic.StreamProcessor{
			Provider: s.anthropicProvider,
		},
		server:    s,
		sessionID: sessionID,
		messageID: messageID,
	}

	// Set callbacks with server integration
	processor.OnText = func(text string) {
		s.eventBus.Publish(Event{
			Type: "message.chunk",
			Data: map[string]interface{}{
				"sessionID": sessionID,
				"messageID": messageID,
				"content":   text,
			},
		})
	}

	return processor.ProcessStream(reader)
}