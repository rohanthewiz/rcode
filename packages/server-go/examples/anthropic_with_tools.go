package main

import (
	"fmt"
	"log"
	"os"

	"github.com/rohanthewiz/logger"
	"github.com/sst/opencode/server-go/internal/provider/anthropic"
	"github.com/sst/opencode/server-go/internal/tool"
)

func main() {
	// Get API key from environment
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable not set")
	}

	// Create provider with tools
	provider := anthropic.NewProvider(apiKey)

	// Example 1: Simple file reading request
	example1(provider)

	// Example 2: Streaming response with tool use
	example2(provider)
}

func example1(provider *anthropic.Provider) {
	fmt.Println("=== Example 1: Simple File Reading ===")

	// Create messages
	messages := []anthropic.Message{
		{
			Role: "user",
			Content: anthropic.TextContent{
				Type: "text",
				Text: "Please read the contents of the file /tmp/test.txt",
			},
		},
	}

	// Send chat request
	response, err := provider.Chat(messages, anthropic.ChatOptions{
		Model:     "claude-3-opus-20240229",
		MaxTokens: 1024,
		Stream:    false,
	})

	if err != nil {
		logger.LogErr(err, "chat request failed")
		return
	}

	fmt.Printf("Response: %+v\n", response)
}

func example2(provider *anthropic.Provider) {
	fmt.Println("\n=== Example 2: Streaming with Tools ===")

	// Create messages
	messages := []anthropic.Message{
		{
			Role: "user",
			Content: anthropic.TextContent{
				Type: "text",
				Text: "Can you read the Go files in the current directory and tell me what they do?",
			},
		},
	}

	// Send streaming chat request
	response, err := provider.Chat(messages, anthropic.ChatOptions{
		Model:     "claude-3-opus-20240229",
		MaxTokens: 4096,
		Stream:    true,
		System:    "You are a helpful assistant that can read files and analyze code.",
	})

	if err != nil {
		logger.LogErr(err, "chat request failed")
		return
	}

	// Process streaming response
	processor := &anthropic.StreamProcessor{
		provider: provider,
		onText: func(text string) {
			fmt.Print(text)
		},
		onToolUse: func(toolUse anthropic.ToolUseContent) {
			fmt.Printf("\n[Tool Use: %s with ID %s]\n", toolUse.Name, toolUse.ID)
			fmt.Printf("Input: %+v\n", toolUse.Input)
		},
		onToolResult: func(result anthropic.ToolResultContent) {
			fmt.Printf("\n[Tool Result for %s]:\n%s\n", result.ToolUseID, result.Content)
		},
		onError: func(err error) {
			logger.LogErr(err, "stream processing error")
		},
	}

	if err := processor.ProcessStream(response.Stream); err != nil {
		logger.LogErr(err, "failed to process stream")
	}
}

// Example showing how to use the provider in a server context
func serverExample() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	provider := anthropic.NewProvider(apiKey)

	// Get available tools for display
	tools := provider.GetAvailableTools()
	fmt.Println("Available tools:")
	for _, t := range tools {
		fmt.Printf("- %s: %s\n", t.ID(), t.Description())
	}

	// Convert tools to Anthropic format for API request
	anthropicTools := make([]anthropic.AnthropicTool, 0, len(tools))
	for _, t := range tools {
		anthropicTools = append(anthropicTools, anthropic.ConvertToolToAnthropicFormat(t))
	}

	fmt.Printf("\nConverted %d tools to Anthropic format\n", len(anthropicTools))

	// Example of manual tool execution
	ctx := tool.NewContext(tool.ContextOptions{
		CWD: "/tmp",
		Metadata: func(data map[string]any) {
			fmt.Printf("Tool metadata: %+v\n", data)
		},
	})

	result, err := provider.ExecuteTool("read", map[string]interface{}{
		"filePath": "/tmp/test.txt",
		"limit":    10,
	}, ctx)

	if err != nil {
		logger.LogErr(err, "tool execution failed")
		return
	}

	fmt.Printf("Tool output:\n%s\n", result.Output)
	fmt.Printf("Tool metadata: %+v\n", result.Metadata)
}