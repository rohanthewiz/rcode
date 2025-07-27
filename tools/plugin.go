package tools

import "context"

// ToolPlugin represents a custom tool that can be loaded dynamically
type ToolPlugin interface {
	// GetDefinition returns the tool metadata
	GetDefinition() Tool

	// Execute runs the tool with the given input
	Execute(ctx context.Context, input map[string]interface{}) (string, error)

	// Initialize is called when the plugin is loaded
	Initialize(config map[string]interface{}) error

	// Cleanup is called when the plugin is unloaded
	Cleanup() error

	// GetCapabilities returns what the tool can do (for sandboxing)
	GetCapabilities() ToolCapabilities
}

// ToolCapabilities defines what a tool is allowed to do
type ToolCapabilities struct {
	FileRead      bool   // Can read files
	FileWrite     bool   // Can write files
	NetworkAccess bool   // Can make network requests
	ProcessSpawn  bool   // Can spawn processes
	WorkingDir    string // Restricted working directory (empty = project root)
}

// PluginMetadata contains information about a plugin
type PluginMetadata struct {
	Name            string
	Version         string
	Author          string
	Description     string
	MinRCodeVersion string
	MaxRCodeVersion string
}
