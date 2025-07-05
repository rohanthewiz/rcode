# Anthropic Tools Integration Guide

This document describes how to integrate tools with Anthropic's Claude API in the Go server implementation.

## Overview

Tools allow Claude to perform actions like reading files, executing commands, and manipulating data. The Go server needs to format tools according to Anthropic's API specification and handle tool execution during conversations.

## Tool Schema Format for Anthropic

### 1. Anthropic Tool Structure

When sending requests to Claude via the Messages API, tools must be formatted as:

```go
type AnthropicTool struct {
    Name         string                 `json:"name"`
    Description  string                 `json:"description"`
    InputSchema  map[string]interface{} `json:"input_schema"`
}
```

### 2. Example Tool Schema

Here's how the read tool is formatted for Anthropic:

```go
readToolSchema := AnthropicTool{
    Name:        "read",
    Description: "Reads a file from the local filesystem with line numbers.",
    InputSchema: map[string]interface{}{
        "type": "object",
        "properties": map[string]interface{}{
            "filePath": map[string]interface{}{
                "type":        "string",
                "description": "The path to the file to read",
            },
            "offset": map[string]interface{}{
                "type":        "number",
                "description": "The line number to start reading from (1-based)",
            },
            "limit": map[string]interface{}{
                "type":        "number",
                "description": "The maximum number of lines to read",
            },
        },
        "required": []string{"filePath"},
    },
}
```

## Including Tools in API Requests

### 1. Request Structure

```go
type AnthropicRequest struct {
    Model     string            `json:"model"`
    Messages  []Message         `json:"messages"`
    Tools     []AnthropicTool   `json:"tools,omitempty"`
    MaxTokens int               `json:"max_tokens"`
    Stream    bool              `json:"stream,omitempty"`
}
```

### 2. Making Requests with Tools

```go
request := AnthropicRequest{
    Model: "claude-3-opus-20240229",
    Messages: messages,
    Tools: []AnthropicTool{
        readToolSchema,
        bashToolSchema,
        writeToolSchema,
        // ... other tools
    },
    MaxTokens: 4096,
    Stream: true,
}
```

## Converting Go Tool Schema to JSON Schema

### 1. Tool Interface Extension

Add a method to convert tools to Anthropic's format:

```go
type AnthropicTool interface {
    tool.Tool
    ToAnthropicSchema() map[string]interface{}
}
```

### 2. Implementation Example

```go
func (t *ReadTool) ToAnthropicSchema() map[string]interface{} {
    return map[string]interface{}{
        "name":        t.ID(),
        "description": t.Description(),
        "input_schema": map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "filePath": map[string]interface{}{
                    "type":        "string",
                    "description": "The path to the file to read",
                },
                "offset": map[string]interface{}{
                    "type":        "number",
                    "description": "The line number to start reading from (1-based)",
                },
                "limit": map[string]interface{}{
                    "type":        "number",
                    "description": "The maximum number of lines to read",
                },
            },
            "required": []string{"filePath"},
        },
    }
}
```

### 3. Generic Schema Converter

For a more automated approach:

```go
func ConvertSchemaToJSON(s tool.Schema) map[string]interface{} {
    // Convert internal schema representation to JSON Schema
    switch schema := s.(type) {
    case *schema.ObjectSchema:
        props := make(map[string]interface{})
        for name, field := range schema.Properties {
            props[name] = ConvertSchemaToJSON(field)
        }
        return map[string]interface{}{
            "type":       "object",
            "properties": props,
            "required":   schema.Required,
        }
    case *schema.StringSchema:
        return map[string]interface{}{
            "type":        "string",
            "description": schema.Description,
        }
    case *schema.NumberSchema:
        return map[string]interface{}{
            "type":        "number",
            "description": schema.Description,
        }
    // ... handle other types
    }
}
```

## Tool Registration and Filtering

### 1. Provider Implementation

```go
type AnthropicProvider struct {
    apiKey   string
    registry *tool.Registry
}

func (p *AnthropicProvider) GetAvailableTools() []tool.Tool {
    // Get all registered tools
    allTools := p.registry.GetAll()
    
    // Filter out tools not supported by Anthropic
    var supportedTools []tool.Tool
    for _, t := range allTools {
        // Anthropic doesn't support the patch tool
        if t.ID() != "patch" {
            supportedTools = append(supportedTools, t)
        }
    }
    
    return supportedTools
}
```

### 2. Tool Registry

```go
func NewToolRegistry() *ToolRegistry {
    registry := &ToolRegistry{
        tools: make(map[string]tool.Tool),
    }
    
    // Register built-in tools
    registry.Register(tools.NewReadTool())
    registry.Register(tools.NewBashTool())
    registry.Register(tools.NewWriteTool())
    registry.Register(tools.NewEditTool())
    registry.Register(tools.NewMultiEditTool())
    registry.Register(tools.NewGlobTool())
    registry.Register(tools.NewGrepTool())
    registry.Register(tools.NewLSTool())
    registry.Register(tools.NewTodoReadTool())
    registry.Register(tools.NewTodoWriteTool())
    registry.Register(tools.NewWebFetchTool())
    
    return registry
}
```

## Handling Tool Calls in Streaming Response

### 1. Tool Use Events

Anthropic sends tool use requests in the streaming response:

```go
type ToolUse struct {
    Type  string                 `json:"type"`  // "tool_use"
    ID    string                 `json:"id"`
    Name  string                 `json:"name"`
    Input map[string]interface{} `json:"input"`
}

type ToolResult struct {
    Type      string `json:"type"`  // "tool_result"
    ToolUseID string `json:"tool_use_id"`
    Content   string `json:"content"`
}
```

### 2. Processing Tool Calls

```go
func (p *AnthropicProvider) handleStreamingResponse(stream io.Reader) error {
    decoder := json.NewDecoder(stream)
    
    for {
        var event StreamEvent
        if err := decoder.Decode(&event); err != nil {
            if err == io.EOF {
                break
            }
            return err
        }
        
        switch event.Type {
        case "content_block_start":
            if block, ok := event.ContentBlock.(map[string]interface{}); ok {
                if block["type"] == "tool_use" {
                    // Tool use started
                    toolUse := ToolUse{
                        ID:   block["id"].(string),
                        Name: block["name"].(string),
                    }
                    // Store for accumulating input
                }
            }
            
        case "content_block_delta":
            // Accumulate tool input
            
        case "content_block_stop":
            // Execute tool and send result
            result, err := p.executeTool(toolUse)
            if err != nil {
                // Handle error
            }
            
            // Continue conversation with tool result
            p.sendToolResult(toolUse.ID, result)
        }
    }
}
```

### 3. Tool Execution

```go
func (p *AnthropicProvider) executeTool(toolUse ToolUse) (tool.Result, error) {
    t := p.registry.Get(toolUse.Name)
    if t == nil {
        return tool.Result{}, fmt.Errorf("unknown tool: %s", toolUse.Name)
    }
    
    // Create execution context
    ctx := tool.NewContext(tool.ContextOptions{
        CWD:      p.workingDir,
        Metadata: func(data map[string]any) {
            // Send metadata updates via SSE
            p.sendMetadata(toolUse.ID, data)
        },
    })
    
    // Execute tool
    return t.Execute(ctx, toolUse.Input)
}
```

## Complete Example

Here's a minimal example of making a request to Claude with tools:

```go
func main() {
    // Initialize provider
    provider := NewAnthropicProvider(apiKey)
    
    // Get available tools
    tools := provider.GetAvailableTools()
    
    // Convert to Anthropic format
    var anthropicTools []AnthropicTool
    for _, t := range tools {
        anthropicTools = append(anthropicTools, AnthropicTool{
            Name:        t.ID(),
            Description: t.Description(),
            InputSchema: ConvertSchemaToJSON(t.Parameters()),
        })
    }
    
    // Make request
    request := AnthropicRequest{
        Model: "claude-3-opus-20240229",
        Messages: []Message{
            {
                Role:    "user",
                Content: "Read the contents of main.go",
            },
        },
        Tools:     anthropicTools,
        MaxTokens: 4096,
        Stream:    true,
    }
    
    // Send request and handle response
    if err := provider.Chat(request); err != nil {
        log.Fatal(err)
    }
}
```

## Key Differences from TypeScript Implementation

1. **No patch tool**: Anthropic doesn't support the patch tool, so it's filtered out
2. **No parameter transformation**: Unlike OpenAI/Azure, Anthropic doesn't need optional-to-nullable conversion
3. **Tool ID normalization**: Replace dots with underscores (e.g., "todo.read" → "todo_read")
4. **Streaming format**: Anthropic uses a different streaming event structure than OpenAI

## OAuth Authentication

When using Claude Pro/Max OAuth authentication:
- Add `Authorization: Bearer {access_token}` header
- Include `anthropic-beta: oauth-2025-04-20` header
- Remove `x-api-key` header

## References

- [Anthropic API Documentation](https://docs.anthropic.com/claude/reference/messages)
- [Tool Use Guide](https://docs.anthropic.com/claude/docs/tool-use)
- TypeScript implementation: `/packages/opencode/src/provider/provider.ts`