package providers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
	"rcode/auth"
	"rcode/config"
)

const (
	anthropicBeta       = "oauth-2025-04-20"
	anthropicVersion    = "2023-06-01"
	claudeCodeUserAgent = "claude.ai/code"
)

// AnthropicClient handles communication with Claude API
type AnthropicClient struct {
	httpClient *http.Client
}

// NewAnthropicClient creates a new Anthropic API client
func NewAnthropicClient() *AnthropicClient {
	return &AnthropicClient{
		httpClient: &http.Client{},
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
		return nil, serr.New(fmt.Sprintf("API error: %s - %s", resp.Status, string(body)))
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
		return serr.New(fmt.Sprintf("API error: %s - %s", resp.Status, string(body)))
	}

	// Read SSE stream
	reader := io.Reader(resp.Body)
	buf := make([]byte, 4096)

	for {
		n, err := reader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return serr.Wrap(err, "failed to read stream")
		}

		// Parse SSE events
		lines := strings.Split(string(buf[:n]), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "data: ") {
				data := strings.TrimPrefix(line, "data: ")
				if data == "[DONE]" {
					return nil
				}

				var event StreamEvent
				if err := json.Unmarshal([]byte(data), &event); err != nil {
					logger.LogErr(err, "failed to parse event")
					continue
				}

				if err := onEvent(event); err != nil {
					return serr.Wrap(err, "error in event handler")
				}
			}
		}
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
