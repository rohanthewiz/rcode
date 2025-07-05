package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
	"github.com/sst/opencode/server-go/internal/provider/anthropic"
	"github.com/sst/opencode/server-go/internal/tool"
	"github.com/sst/opencode/server-go/internal/tools"
)

// Server represents the OpenCode Go server.
// It provides HTTP endpoints compatible with the TypeScript server API.
type Server struct {
	srv              *rweb.Server
	toolRegistry     *tool.Registry
	toolExecutor     *tool.Executor
	sessions         sync.Map // sessionID -> *Session
	authSessions     sync.Map // verifier -> expiry time
	eventBus         *EventBus
	anthropicProvider *anthropic.Provider
}

// Session represents an active chat session
type Session struct {
	ID       string
	Messages []Message
	mu       sync.RWMutex
}

// Message represents a chat message
type Message struct {
	ID       string                 `json:"id"`
	Role     string                 `json:"role"`
	Content  string                 `json:"content"`
	Parts    []MessagePart          `json:"parts,omitempty"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// MessagePart represents a part of a message (text, tool use, etc)
type MessagePart struct {
	Type       string                 `json:"type"`
	Text       string                 `json:"text,omitempty"`
	ToolUse    *ToolUse               `json:"toolUse,omitempty"`
	ToolResult *ToolResult            `json:"toolResult,omitempty"`
}

// ToolUse represents a tool invocation request
type ToolUse struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Parameters map[string]interface{} `json:"parameters"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

// NewServer creates a new server instance
func NewServer(port int) *Server {
	// Initialize tool registry
	registry := tool.NewRegistry()
	
	// Register built-in tools
	registry.Register(tools.NewReadTool())
	registry.Register(tools.NewBashTool())
	// TODO: Add more tools (write, edit, glob, grep, etc.)
	
	// Create tool executor
	executor := tool.NewExecutor(registry)
	
	// Create event bus for SSE
	eventBus := NewEventBus()
	
	// Create rweb server
	srv := rweb.NewServer(rweb.ServerOptions{
		Address: fmt.Sprintf(":%d", port),
		Verbose: true,
	})
	
	// Create Anthropic provider if API key is available
	var anthropicProvider *anthropic.Provider
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		anthropicProvider = anthropic.NewProvider(apiKey)
	}
	
	return &Server{
		srv:              srv,
		toolRegistry:     registry,
		toolExecutor:     executor,
		eventBus:         eventBus,
		anthropicProvider: anthropicProvider,
	}
}

// Run starts the server
func (s *Server) Run() error {
	// Setup middleware
	s.srv.Use(rweb.RequestInfo)
	s.srv.Use(corsMiddleware)
	
	// Setup routes
	s.setupRoutes()
	
	// Start the server
	logger.Info("Starting OpenCode Go server", "port", s.srv.Address)
	return s.srv.Run()
}

// corsMiddleware adds CORS headers for compatibility with the TUI
func corsMiddleware(next rweb.HandlerFunc) rweb.HandlerFunc {
	return func(c rweb.Context) error {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request().Method == "OPTIONS" {
			return c.NoContent(http.StatusOK)
		}
		
		return next(c)
	}
}

// setupRoutes configures all HTTP endpoints
func (s *Server) setupRoutes() {
	// Health check
	s.srv.Get("/health", s.healthHandler)
	
	// Session management
	s.srv.Post("/session", s.createSessionHandler)
	s.srv.Get("/session/:id", s.getSessionHandler)
	s.srv.Post("/session/:id/message", s.ChatHandler)  // Use new ChatHandler
	
	// Tool information
	s.srv.Get("/tools", s.listToolsHandler)
	
	// Server-Sent Events for real-time updates
	s.srv.Get("/event", s.eventStreamHandler)
	
	// Configuration endpoints
	s.srv.Get("/config", s.configHandler)
	s.srv.Get("/config/providers", s.providersHandler)
	s.srv.Get("/provider/:provider/models", s.modelsHandler)
	
	// Authentication endpoints
	s.setupAuthRoutes()
}

// healthHandler returns server health status
func (s *Server) healthHandler(c rweb.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "ok",
		"server": "opencode-go",
	})
}

// createSessionHandler creates a new chat session
func (s *Server) createSessionHandler(c rweb.Context) error {
	// Generate session ID
	sessionID := generateID()
	
	// Create new session
	session := &Session{
		ID:       sessionID,
		Messages: []Message{},
	}
	
	// Store session
	s.sessions.Store(sessionID, session)
	
	// Publish session created event
	s.eventBus.Publish(Event{
		Type: "session.created",
		Data: map[string]interface{}{
			"sessionID": sessionID,
		},
	})
	
	return c.JSON(http.StatusOK, map[string]string{
		"id": sessionID,
	})
}

// getSessionHandler retrieves session information
func (s *Server) getSessionHandler(c rweb.Context) error {
	sessionID := c.Param("id")
	
	sessionVal, exists := s.sessions.Load(sessionID)
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "session not found",
		})
	}
	
	session := sessionVal.(*Session)
	session.mu.RLock()
	defer session.mu.RUnlock()
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id":       session.ID,
		"messages": session.Messages,
	})
}

// sendMessageHandler handles sending a message in a session
func (s *Server) sendMessageHandler(c rweb.Context) error {
	sessionID := c.Param("id")
	
	// Parse request body
	var req struct {
		Content    string `json:"content"`
		ProviderID string `json:"providerID"`
		Model      string `json:"model"`
	}
	
	if err := c.BindJSON(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request",
		})
	}
	
	// Get session
	sessionVal, exists := s.sessions.Load(sessionID)
	if !exists {
		return c.JSON(http.StatusNotFound, map[string]string{
			"error": "session not found",
		})
	}
	
	session := sessionVal.(*Session)
	
	// Add user message
	userMessage := Message{
		ID:      generateID(),
		Role:    "user",
		Content: req.Content,
	}
	
	session.mu.Lock()
	session.Messages = append(session.Messages, userMessage)
	session.mu.Unlock()
	
	// For now, simulate a simple response with tool use
	// In a real implementation, this would call the AI provider
	go s.processMessage(session, req.ProviderID, req.Model)
	
	return c.JSON(http.StatusOK, map[string]string{
		"status": "processing",
	})
}

// processMessage simulates AI message processing with tool execution
func (s *Server) processMessage(session *Session, providerID, model string) {
	// Create assistant message
	assistantMessage := Message{
		ID:   generateID(),
		Role: "assistant",
		Parts: []MessagePart{
			{
				Type: "text",
				Text: "I'll help you with that. Let me read the file first.",
			},
		},
		Metadata: map[string]interface{}{
			"tool": map[string]interface{}{},
		},
	}
	
	// Add to session
	session.mu.Lock()
	session.Messages = append(session.Messages, assistantMessage)
	session.mu.Unlock()
	
	// Publish message event
	s.eventBus.Publish(Event{
		Type: "message.created",
		Data: map[string]interface{}{
			"sessionID": session.ID,
			"message":   assistantMessage,
		},
	})
	
	// Simulate tool execution (in real implementation, this would come from AI)
	if strings.Contains(session.Messages[len(session.Messages)-2].Content, "read") {
		// Execute read tool
		toolCallID := generateID()
		ctx := tool.Context{
			SessionID: session.ID,
			MessageID: assistantMessage.ID,
			Abort:     context.Background(),
			Metadata: func(meta map[string]any) {
				// Update metadata and publish event
				assistantMessage.Metadata["tool"].(map[string]interface{})[toolCallID] = meta
				
				s.eventBus.Publish(Event{
					Type: "message.updated",
					Data: map[string]interface{}{
						"sessionID": session.ID,
						"message":   assistantMessage,
					},
				})
			},
		}
		
		// Example: read a file
		result, err := s.toolExecutor.Execute("read", map[string]any{
			"filePath": "/tmp/test.txt",
		}, ctx)
		
		// Add tool result to message
		toolResult := ""
		if err != nil {
			toolResult = fmt.Sprintf("Error: %v", err)
		} else {
			toolResult = result.Output
		}
		
		assistantMessage.Parts = append(assistantMessage.Parts, MessagePart{
			Type: "toolResult",
			ToolResult: &ToolResult{
				ID:      toolCallID,
				Content: toolResult,
			},
		})
		
		// Final response
		assistantMessage.Parts = append(assistantMessage.Parts, MessagePart{
			Type: "text",
			Text: "I've read the file. The content is shown above.",
		})
		
		// Update session
		session.mu.Lock()
		session.Messages[len(session.Messages)-1] = assistantMessage
		session.mu.Unlock()
		
		// Publish final update
		s.eventBus.Publish(Event{
			Type: "message.updated",
			Data: map[string]interface{}{
				"sessionID": session.ID,
				"message":   assistantMessage,
			},
		})
	}
}

// listToolsHandler returns available tools
func (s *Server) listToolsHandler(c rweb.Context) error {
	tools := s.toolRegistry.GetAll()
	
	// Convert to API format
	toolList := make([]map[string]interface{}, len(tools))
	for i, t := range tools {
		toolList[i] = map[string]interface{}{
			"id":          t.ID(),
			"description": t.Description(),
			// TODO: Add schema conversion to JSON
			"parameters":  map[string]interface{}{"type": "object"},
		}
	}
	
	return c.JSON(http.StatusOK, toolList)
}

// listProvidersHandler returns available providers (stub)
func (s *Server) listProvidersHandler(c rweb.Context) error {
	// For now, return a stub response
	// In real implementation, this would list actual providers
	return c.JSON(http.StatusOK, []map[string]interface{}{
		{
			"id":   "anthropic",
			"name": "Anthropic",
			"models": []map[string]interface{}{
				{
					"id":   "claude-3-opus",
					"name": "Claude 3 Opus",
				},
			},
		},
	})
}

// eventStreamHandler handles Server-Sent Events
func (s *Server) eventStreamHandler(c rweb.Context) error {
	// Set SSE headers
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	
	// Create event subscription
	sub := s.eventBus.Subscribe()
	defer s.eventBus.Unsubscribe(sub)
	
	// Send events
	for {
		select {
		case event := <-sub:
			data, _ := json.Marshal(event)
			fmt.Fprintf(c.Response(), "data: %s\n\n", data)
			c.Response().Flush()
			
		case <-c.Request().Context().Done():
			return nil
		}
	}
}

// generateID generates a unique ID (simplified version)
func generateID() string {
	// In production, use a proper UUID generator
	return fmt.Sprintf("%d", time.Now().UnixNano())
}