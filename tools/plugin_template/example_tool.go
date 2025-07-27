//go:build ignore

package main

import (
	"context"
	"fmt"
	"rcode/tools"
)

// Tool is the exported symbol that RCode looks for
var Tool ExampleTool

// Metadata provides information about this plugin
var Metadata = &tools.PluginMetadata{
	Name:            "example_tool",
	Version:         "1.0.0",
	Author:          "Your Name",
	Description:     "An example custom tool",
	MinRCodeVersion: "0.1.0",
}

// ExampleTool implements the ToolPlugin interface
type ExampleTool struct {
	config map[string]interface{}
}

// GetDefinition returns the tool metadata
func (t ExampleTool) GetDefinition() tools.Tool {
	return tools.Tool{
		Name:        "example_tool",
		Description: "An example custom tool that demonstrates the plugin interface",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{
					"type":        "string",
					"description": "A message to process",
				},
				"uppercase": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to convert the message to uppercase",
					"default":     false,
				},
			},
			"required": []string{"message"},
		},
	}
}

// Execute runs the tool
func (t ExampleTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
	// Extract the message parameter
	message, ok := input["message"].(string)
	if !ok {
		return "", fmt.Errorf("message parameter is required and must be a string")
	}

	// Check if we should convert to uppercase
	uppercase := false
	if val, ok := input["uppercase"].(bool); ok {
		uppercase = val
	}

	// Check context for cancellation
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Process the message
	result := message
	if uppercase {
		result = fmt.Sprintf("PROCESSED: %s", result)
	} else {
		result = fmt.Sprintf("Processed: %s", result)
	}

	return result, nil
}

// Initialize sets up the tool
func (t *ExampleTool) Initialize(config map[string]interface{}) error {
	t.config = config
	// Perform any initialization here
	// For example: connect to databases, load configuration files, etc.
	return nil
}

// Cleanup cleans up resources
func (t ExampleTool) Cleanup() error {
	// Clean up any resources
	// For example: close database connections, flush buffers, etc.
	return nil
}

// GetCapabilities returns what this tool can do
func (t ExampleTool) GetCapabilities() tools.ToolCapabilities {
	return tools.ToolCapabilities{
		FileRead:      false, // This tool doesn't read files
		FileWrite:     false, // This tool doesn't write files
		NetworkAccess: false, // This tool doesn't access the network
		ProcessSpawn:  false, // This tool doesn't spawn processes
		WorkingDir:    "",    // No working directory restriction
	}
}
