package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
	"rcode/auth"
	"rcode/config"
	contextpkg "rcode/context"
	"rcode/tools"
)

const (
	anthropicBeta       = "oauth-2025-04-20"
	anthropicVersion    = "2023-06-01"
	claudeCodeUserAgent = "claude.ai/code"
)

// AnthropicClient handles communication with Claude API
type AnthropicClient struct {
	httpClient     *http.Client
	contextManager *contextpkg.Manager
}

// NewAnthropicClient creates a new Anthropic API client
func NewAnthropicClient() *AnthropicClient {
	return &AnthropicClient{
		httpClient:     &http.Client{},
		contextManager: contextpkg.NewManager(),
	}
}

// Message represents a chat message
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// TextContent represents text content in a message
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolUse represents a tool use in the assistant's response
type ToolUse struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

// CreateMessageRequest represents the request to create a message
type CreateMessageRequest struct {
	Model     string      `json:"model"`
	Messages  []Message   `json:"messages"`
	MaxTokens int         `json:"max_tokens"`
	Stream    bool        `json:"stream"`
	System    string      `json:"system,omitempty"`
	Tools     interface{} `json:"tools,omitempty"`
}

// CreateMessageResponse represents the response from creating a message
type CreateMessageResponse struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Role    string    `json:"role"`
	Content []Content `json:"content"`
	Model   string    `json:"model"`
	Usage   Usage     `json:"usage"`
}

// Content represents content in the response
type Content struct {
	Type  string      `json:"type"`
	Text  string      `json:"text,omitempty"`
	ID    string      `json:"id,omitempty"`
	Name  string      `json:"name,omitempty"`
	Input interface{} `json:"input,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// StreamEvent represents a server-sent event in the stream
type StreamEvent struct {
	Type    string          `json:"type"`
	Message json.RawMessage `json:"message,omitempty"`
	Delta   json.RawMessage `json:"delta,omitempty"`
	Index   int             `json:"index,omitempty"`
}

// SendMessage sends a message to Claude and returns the response
func (c *AnthropicClient) SendMessage(request CreateMessageRequest) (*CreateMessageResponse, error) {
	// Get access token
	accessToken, err := auth.GetAccessToken()
	if err != nil {
		return nil, serr.Wrap(err, "failed to get access token")
	}

	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return nil, serr.Wrap(err, "failed to marshal request")
	}

	// Get API URL from config
	apiURL := config.Get().AnthropicAPIURL
	
	// Log the request for debugging
	logger.Info("Anthropic API Request ->" + string(requestBody))
	logger.Info("Using model: " + request.Model)
	logger.Info("API URL", "url", apiURL)
	
	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(requestBody))
	if err != nil {
		return nil, serr.Wrap(err, "failed to create request")
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("anthropic-beta", anthropicBeta)
	req.Header.Set("anthropic-version", anthropicVersion)

	// Log headers and model for debugging
	logger.Info("Request details",
		"model", request.Model,
		"anthropic-beta", anthropicBeta,
		"anthropic-version", anthropicVersion)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, serr.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, serr.Wrap(err, "failed to read response")
	}

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		apiErr := serr.New(fmt.Sprintf("API error: %d - %s", resp.StatusCode, string(body)))
		
		// Classify API errors for retry handling
		switch resp.StatusCode {
		case 429: // Rate limit
			// Extract retry-after if available
			retryAfter := 60 // default
			if retryHeader := resp.Header.Get("Retry-After"); retryHeader != "" {
				// Parse retry-after header
				if seconds, err := time.ParseDuration(retryHeader + "s"); err == nil {
					retryAfter = int(seconds.Seconds())
				}
			}
			return nil, tools.NewRateLimitError(apiErr, retryAfter)
		case 500, 502, 503, 504, 529: // Server errors including overloaded
			return nil, tools.NewRetryableError(apiErr, "server error")
		case 400, 401, 403, 404: // Client errors
			return nil, tools.NewPermanentError(apiErr, "client error")
		default:
			if resp.StatusCode >= 500 {
				return nil, tools.NewRetryableError(apiErr, "server error")
			}
			return nil, tools.NewPermanentError(apiErr, "client error")
		}
	}

	// Parse response
	var response CreateMessageResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, serr.Wrap(err, "failed to parse response")
	}

	// Log the model from the response
	logger.Info("API Response model", "model", response.Model)

	return &response, nil
}

// SendMessageWithRetry sends a message to Claude with automatic retry for transient errors
func (c *AnthropicClient) SendMessageWithRetry(request CreateMessageRequest) (*CreateMessageResponse, error) {
	// Define retry policy for API calls
	retryPolicy := tools.RetryPolicy{
		MaxAttempts:     5,
		InitialDelay:    1 * time.Second,
		MaxDelay:        60 * time.Second,
		Multiplier:      2.0,
		Jitter:          true,
		RetryableErrors: tools.IsRetryableError,
	}
	
	var response *CreateMessageResponse
	
	operation := func(ctx context.Context) error {
		resp, err := c.SendMessage(request)
		if err != nil {
			return err
		}
		response = resp
		return nil
	}
	
	result := tools.Retry(context.Background(), retryPolicy, operation)
	if result.LastError != nil {
		// Log retry details if we had retries
		if result.Attempts > 1 {
			logger.LogErr(result.LastError, 
				fmt.Sprintf("Failed to send message after %d attempts", result.Attempts))
		}
		return nil, result.LastError
	}
	
	// Log successful retry if needed
	if result.Attempts > 1 {
		logger.Info("Message sent successfully after retries",
			"attempts", result.Attempts,
			"duration", result.TotalDuration)
	}
	
	return response, nil
}

// StreamMessage sends a message to Claude and streams the response
func (c *AnthropicClient) StreamMessage(request CreateMessageRequest, onEvent func(StreamEvent) error) error {
	// Ensure streaming is enabled
	request.Stream = true

	// Get access token
	accessToken, err := auth.GetAccessToken()
	if err != nil {
		return serr.Wrap(err, "failed to get access token")
	}

	// Marshal request
	requestBody, err := json.Marshal(request)
	if err != nil {
		return serr.Wrap(err, "failed to marshal request")
	}

	// Get API URL from config
	apiURL := config.Get().AnthropicAPIURL
	
	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(requestBody))
	if err != nil {
		return serr.Wrap(err, "failed to create request")
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("anthropic-beta", anthropicBeta)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("Accept", "text/event-stream")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return serr.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	// Check for errors
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		apiErr := serr.New(fmt.Sprintf("API error: %d - %s", resp.StatusCode, string(body)))
		
		// Classify API errors for retry handling
		switch resp.StatusCode {
		case 429: // Rate limit
			retryAfter := 60 // default
			if retryHeader := resp.Header.Get("Retry-After"); retryHeader != "" {
				if seconds, err := time.ParseDuration(retryHeader + "s"); err == nil {
					retryAfter = int(seconds.Seconds())
				}
			}
			return tools.NewRateLimitError(apiErr, retryAfter)
		case 500, 502, 503, 504, 529: // Server errors including overloaded
			return tools.NewRetryableError(apiErr, "server error")
		case 400, 401, 403, 404: // Client errors
			return tools.NewPermanentError(apiErr, "client error")
		default:
			if resp.StatusCode >= 500 {
				return tools.NewRetryableError(apiErr, "server error")
			}
			return tools.NewPermanentError(apiErr, "client error")
		}
	}

	// Read SSE stream with proper buffering
	scanner := bufio.NewScanner(resp.Body)
	var currentEvent strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		
		// Handle event data
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return nil
			}
			currentEvent.WriteString(data)
		} else if line == "" && currentEvent.Len() > 0 {
			// Empty line indicates end of event
			eventData := currentEvent.String()
			currentEvent.Reset()
			
			// First try to get just the type
			var typeCheck struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal([]byte(eventData), &typeCheck); err != nil {
				logger.LogErr(err, "failed to unmarshal SSE event type: " + eventData)
				continue
			}
			
			// For content_block_start, the structure might be different
			if typeCheck.Type == "content_block_start" {
				// Try parsing as a content block start event with content_block at top level
				var blockStart struct {
					Type         string `json:"type"`
					Index        int    `json:"index"`
					ContentBlock struct {
						Type string `json:"type"`
						ID   string `json:"id"`
						Name string `json:"name,omitempty"`
					} `json:"content_block"`
				}
				if err := json.Unmarshal([]byte(eventData), &blockStart); err == nil && blockStart.ContentBlock.Type != "" {
					// Convert to StreamEvent format
					event := StreamEvent{
						Type:  blockStart.Type,
						Index: blockStart.Index,
					}
					// Marshal the content block as the message
					if blockMsg, err := json.Marshal(blockStart.ContentBlock); err == nil {
						event.Message = blockMsg
					}
					if err := onEvent(event); err != nil {
						return serr.Wrap(err, "error in event handler")
					}
					continue
				}
			}
			
			// Parse as regular StreamEvent
			var event StreamEvent
			if err := json.Unmarshal([]byte(eventData), &event); err != nil {
				logger.LogErr(err, "failed to parse event: " + eventData)
				continue
			}

			if err := onEvent(event); err != nil {
				return serr.Wrap(err, "error in event handler")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return serr.Wrap(err, "failed to read stream")
	}

	return nil
}

// StreamMessageWithRetry sends a message to Claude and streams the response with retry
func (c *AnthropicClient) StreamMessageWithRetry(request CreateMessageRequest, onEvent func(StreamEvent) error) error {
	// Define retry policy for streaming API calls
	retryPolicy := tools.RetryPolicy{
		MaxAttempts:     5,
		InitialDelay:    1 * time.Second,
		MaxDelay:        60 * time.Second,
		Multiplier:      2.0,
		Jitter:          true,
		RetryableErrors: tools.IsRetryableError,
	}
	
	operation := func(ctx context.Context) error {
		return c.StreamMessage(request, onEvent)
	}
	
	result := tools.Retry(context.Background(), retryPolicy, operation)
	if result.LastError != nil {
		// Log retry details if we had retries
		if result.Attempts > 1 {
			logger.LogErr(result.LastError, 
				fmt.Sprintf("Failed to stream message after %d attempts", result.Attempts))
		}
		return result.LastError
	}
	
	// Log successful retry if needed
	if result.Attempts > 1 {
		logger.Info("Message streamed successfully after retries",
			"attempts", result.Attempts,
			"duration", result.TotalDuration)
	}
	
	return nil
}

// ConvertToAPIMessages converts internal messages to API format
func ConvertToAPIMessages(messages []ChatMessage) []Message {
	apiMessages := make([]Message, len(messages))
	for i, msg := range messages {
		// Handle different content types
		switch content := msg.Content.(type) {
		case string:
			apiMessages[i] = Message{
				Role: msg.Role,
				Content: []TextContent{{
					Type: "text",
					Text: content,
				}},
			}
		default:
			// For complex content, pass through as-is
			apiMessages[i] = Message{
				Role:    msg.Role,
				Content: content,
			}
		}
	}
	return apiMessages
}

// ChatMessage represents an internal chat message
type ChatMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// SetContextManager sets the context manager for the client
func (c *AnthropicClient) SetContextManager(cm *contextpkg.Manager) {
	c.contextManager = cm
}

// GetContextManager returns the context manager
func (c *AnthropicClient) GetContextManager() *contextpkg.Manager {
	return c.contextManager
}

// InitializeContext initializes the project context
func (c *AnthropicClient) InitializeContext(projectPath string) error {
	if c.contextManager == nil {
		return serr.New("context manager not initialized")
	}
	
	_, err := c.contextManager.ScanProject(projectPath)
	if err != nil {
		return serr.Wrap(err, "failed to scan project")
	}
	
	return nil
}


// GetRelevantFiles returns files relevant to the current task
func (c *AnthropicClient) GetRelevantFiles(task string, maxFiles int) ([]string, error) {
	if c.contextManager == nil || !c.contextManager.IsInitialized() {
		return nil, nil
	}
	
	files, err := c.contextManager.PrioritizeFiles(task)
	if err != nil {
		return nil, serr.Wrap(err, "failed to prioritize files")
	}
	
	// Limit to maxFiles
	if len(files) > maxFiles {
		files = files[:maxFiles]
	}
	
	return files, nil
}

// TrackFileChange tracks a file change in the context
func (c *AnthropicClient) TrackFileChange(filepath string, changeType contextpkg.ChangeType) {
	if c.contextManager != nil {
		c.contextManager.TrackChange(filepath, changeType)
	}
}
