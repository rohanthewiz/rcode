package tool

import (
	"context"
	"sync"

	"github.com/rohanthewiz/serr"
)

// Tool represents a capability that can be invoked by the AI assistant.
// Each tool has a unique ID, description, parameter schema, and execution logic.
type Tool interface {
	// ID returns the unique identifier for this tool
	ID() string
	
	// Description returns a detailed description of what this tool does
	Description() string
	
	// Parameters returns the schema for validating input parameters
	Parameters() Schema
	
	// Execute runs the tool with the given parameters and context
	Execute(ctx Context, params map[string]any) (Result, error)
}

// Context provides runtime information for tool execution.
// It includes session/message identifiers, cancellation support, and metadata updates.
type Context struct {
	// SessionID identifies the current chat session
	SessionID string
	
	// MessageID identifies the message that triggered this tool execution
	MessageID string
	
	// Abort provides cancellation support via context
	Abort context.Context
	
	// Metadata allows tools to send real-time updates during execution
	Metadata func(meta map[string]any)
}

// Result contains the output from a tool execution.
// Output is the main text result, while Metadata provides structured data.
type Result struct {
	// Output is the text result that will be shown to the user
	Output string
	
	// Metadata contains structured data about the execution
	Metadata map[string]any
}

// Registry manages the collection of available tools.
// It provides thread-safe registration and retrieval of tools.
type Registry struct {
	tools map[string]Tool
	mu    sync.RWMutex
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry.
// Returns an error if a tool with the same ID already exists.
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.tools[tool.ID()]; exists {
		return serr.New("tool with ID %s already registered", tool.ID())
	}
	
	r.tools[tool.ID()] = tool
	return nil
}

// Get retrieves a tool by ID.
// Returns the tool and true if found, nil and false otherwise.
func (r *Registry) Get(id string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tool, ok := r.tools[id]
	return tool, ok
}

// List returns all registered tools
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ForProvider returns tools available for a specific AI provider.
// This allows customizing which tools are available for different providers.
func (r *Registry) ForProvider(provider string) []Tool {
	// For now, return all tools. In the future, we can add provider-specific filtering
	// based on tool capabilities and provider limitations.
	return r.List()
}

// Schema represents a parameter validation schema.
// It defines the structure and constraints for tool input parameters.
type Schema interface {
	// Validate checks if the given value conforms to this schema
	Validate(value any) error
	
	// Description returns a human-readable description of this schema
	Description() string
	
	// ToJSON converts the schema to a JSON-compatible representation
	ToJSON() map[string]any
}