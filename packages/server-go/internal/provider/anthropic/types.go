package anthropic

// AnthropicTool represents a tool definition in Anthropic's API format
type AnthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// Request represents an Anthropic API request
type Request struct {
	Model      string           `json:"model"`
	Messages   []Message        `json:"messages"`
	Tools      []AnthropicTool `json:"tools,omitempty"`
	MaxTokens  int              `json:"max_tokens"`
	Stream     bool             `json:"stream,omitempty"`
	System     string           `json:"system,omitempty"`
	StopTokens []string         `json:"stop_sequences,omitempty"`
}

// Message represents a message in the conversation
type Message struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

// TextContent represents text content in a message
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ToolUseContent represents a tool use request
type ToolUseContent struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ToolResultContent represents a tool execution result
type ToolResultContent struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
}

// StreamEvent represents a streaming response event
type StreamEvent struct {
	Type         string      `json:"type"`
	Index        int         `json:"index,omitempty"`
	Delta        interface{} `json:"delta,omitempty"`
	ContentBlock interface{} `json:"content_block,omitempty"`
	Message      interface{} `json:"message,omitempty"`
}

// ContentBlockDelta represents content changes in streaming
type ContentBlockDelta struct {
	Type         string `json:"type"`
	Text         string `json:"text,omitempty"`
	PartialJSON  string `json:"partial_json,omitempty"`
}