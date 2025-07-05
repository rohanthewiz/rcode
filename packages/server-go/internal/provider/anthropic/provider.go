package anthropic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
	"github.com/sst/opencode/server-go/internal/auth"
	"github.com/sst/opencode/server-go/internal/tool"
	"github.com/sst/opencode/server-go/internal/tools"
)

const (
	apiURL        = "https://api.anthropic.com/v1/messages"
	anthropicBeta = "messages-2023-12-15"
)

// Provider implements the Anthropic provider
type Provider struct {
	apiKey   string
	auth     *auth.AnthropicAuth
	registry *tool.Registry
	client   *http.Client
}

// NewProvider creates a new Anthropic provider
func NewProvider(apiKey string) *Provider {
	p := &Provider{
		apiKey:   apiKey,
		auth:     auth.NewAnthropicAuth(),
		registry: tool.NewRegistry(),
		client:   &http.Client{},
	}

	// Register built-in tools
	p.registerTools()

	return p
}

// registerTools registers all built-in tools with the provider
func (p *Provider) registerTools() {
	// Register the read tool
	if err := p.registry.Register(tools.NewReadTool()); err != nil {
		logger.LogErr(err, "failed to register read tool")
	}

	// Register bash tool if implemented
	// if err := p.registry.Register(tools.NewBashTool()); err != nil {
	//     logger.LogErr(err, "failed to register bash tool")
	// }

	// Add other tools as they are implemented...
}

// GetAvailableTools returns tools available for Anthropic
func (p *Provider) GetAvailableTools() []tool.Tool {
	// Anthropic doesn't support the patch tool
	return p.registry.GetFiltered(func(t tool.Tool) bool {
		return t.ID() != "patch"
	})
}

// Chat sends a chat request to Anthropic with tool support
func (p *Provider) Chat(messages []Message, options ChatOptions) (*ChatResponse, error) {
	// Get available tools
	tools := p.GetAvailableTools()
	
	// Convert tools to Anthropic format
	anthropicTools := make([]AnthropicTool, 0, len(tools))
	for _, t := range tools {
		anthropicTools = append(anthropicTools, ConvertToolToAnthropicFormat(t))
	}

	// Build request
	request := Request{
		Model:     options.Model,
		Messages:  messages,
		Tools:     anthropicTools,
		MaxTokens: options.MaxTokens,
		Stream:    options.Stream,
		System:    options.System,
	}

	// Send request
	if options.Stream {
		return p.streamChat(request)
	}
	return p.normalChat(request)
}

// normalChat handles non-streaming chat requests
func (p *Provider) normalChat(request Request) (*ChatResponse, error) {
	// Marshal request
	body, err := json.Marshal(request)
	if err != nil {
		return nil, serr.Wrap(err, "failed to marshal request")
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, serr.Wrap(err, "failed to create request")
	}

	// Set headers
	if err := p.setHeaders(req); err != nil {
		return nil, serr.Wrap(err, "failed to set headers")
	}

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, serr.Wrap(err, "failed to send request")
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, serr.New("API error: %d - %s", resp.StatusCode, string(body))
	}

	// Parse response
	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, serr.Wrap(err, "failed to decode response")
	}

	return &chatResp, nil
}

// streamChat handles streaming chat requests
func (p *Provider) streamChat(request Request) (*ChatResponse, error) {
	// Marshal request
	body, err := json.Marshal(request)
	if err != nil {
		return nil, serr.Wrap(err, "failed to marshal request")
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	if err != nil {
		return nil, serr.Wrap(err, "failed to create request")
	}

	// Set headers
	if err := p.setHeaders(req); err != nil {
		return nil, serr.Wrap(err, "failed to set headers")
	}

	// Send request
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, serr.Wrap(err, "failed to send request")
	}

	// Check status
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, serr.New("API error: %d - %s", resp.StatusCode, string(body))
	}

	// Return response with body reader for streaming
	return &ChatResponse{
		Stream: resp.Body,
	}, nil
}

// setHeaders sets the required headers for Anthropic API
func (p *Provider) setHeaders(req *http.Request) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("anthropic-beta", anthropicBeta)
	
	// Try OAuth first, fall back to API key
	accessToken, err := p.auth.Access()
	if err == nil && accessToken != "" {
		// Use OAuth token
		req.Header.Set("Authorization", "Bearer "+accessToken)
		req.Header.Set("anthropic-beta", anthropicBeta+",oauth-2025-04-20")
		logger.Debug("Using OAuth authentication for Anthropic")
	} else if p.apiKey != "" {
		// Fall back to API key
		req.Header.Set("x-api-key", p.apiKey)
		logger.Debug("Using API key authentication for Anthropic")
	} else {
		return serr.New("no authentication method available")
	}
	
	return nil
}

// ExecuteTool executes a tool by ID with the given parameters
func (p *Provider) ExecuteTool(toolID string, params map[string]interface{}, ctx tool.Context) (tool.Result, error) {
	t := p.registry.Get(toolID)
	if t == nil {
		return tool.Result{}, serr.New("unknown tool: %s", toolID)
	}

	return t.Execute(ctx, params)
}

// StreamProcessor handles streaming responses with tool execution
type StreamProcessor struct {
	Provider     *Provider
	OnText       func(text string)
	OnToolUse    func(toolUse ToolUseContent)
	OnToolResult func(result ToolResultContent)
	OnError      func(err error)
}

// ProcessStream processes a streaming response
func (sp *StreamProcessor) ProcessStream(reader io.ReadCloser) error {
	defer reader.Close()

	decoder := json.NewDecoder(reader)
	var currentToolUse *ToolUseContent
	var partialJSON strings.Builder

	for {
		line, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return serr.Wrap(err, "failed to read stream")
		}

		// Parse SSE data
		if str, ok := line.(string); ok && strings.HasPrefix(str, "data: ") {
			data := strings.TrimPrefix(str, "data: ")
			if data == "[DONE]" {
				break
			}

			var event StreamEvent
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				logger.LogErr(err, "failed to parse stream event")
				continue
			}

			// Handle different event types
			switch event.Type {
			case "content_block_start":
				if block, ok := event.ContentBlock.(map[string]interface{}); ok {
					if block["type"] == "tool_use" {
						currentToolUse = &ToolUseContent{
							Type: "tool_use",
							ID:   block["id"].(string),
							Name: block["name"].(string),
						}
						partialJSON.Reset()
					}
				}

			case "content_block_delta":
				if delta, ok := event.Delta.(map[string]interface{}); ok {
					if text, ok := delta["text"].(string); ok && sp.OnText != nil {
						sp.OnText(text)
					}
					if partial, ok := delta["partial_json"].(string); ok && currentToolUse != nil {
						partialJSON.WriteString(partial)
					}
				}

			case "content_block_stop":
				if currentToolUse != nil {
					// Parse accumulated JSON for tool input
					if partialJSON.Len() > 0 {
						var input map[string]interface{}
						if err := json.Unmarshal([]byte(partialJSON.String()), &input); err == nil {
							currentToolUse.Input = input
						}
					}

					// Notify about tool use
					if sp.OnToolUse != nil {
						sp.OnToolUse(*currentToolUse)
					}

					// Execute tool
					ctx := tool.NewContext(tool.ContextOptions{})
					result, err := sp.Provider.ExecuteTool(currentToolUse.Name, currentToolUse.Input, ctx)
					
					toolResult := ToolResultContent{
						Type:      "tool_result",
						ToolUseID: currentToolUse.ID,
					}

					if err != nil {
						toolResult.Content = fmt.Sprintf("Error: %v", err)
					} else {
						toolResult.Content = result.Output
					}

					// Notify about tool result
					if sp.OnToolResult != nil {
						sp.OnToolResult(toolResult)
					}

					currentToolUse = nil
					partialJSON.Reset()
				}
			}
		}
	}

	return nil
}

// ChatOptions represents options for chat requests
type ChatOptions struct {
	Model     string
	MaxTokens int
	Stream    bool
	System    string
}

// ChatResponse represents a chat response
type ChatResponse struct {
	ID      string    `json:"id,omitempty"`
	Type    string    `json:"type,omitempty"`
	Role    string    `json:"role,omitempty"`
	Content []Content `json:"content,omitempty"`
	Stream  io.ReadCloser
}

// Content represents content in a response
type Content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}