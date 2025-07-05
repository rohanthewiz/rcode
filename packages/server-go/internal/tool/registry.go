package tool

import (
	"fmt"
	"sync"
)

// Registry manages available tools
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

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tool == nil {
		return fmt.Errorf("cannot register nil tool")
	}

	id := tool.ID()
	if id == "" {
		return fmt.Errorf("tool has empty ID")
	}

	if _, exists := r.tools[id]; exists {
		return fmt.Errorf("tool with ID %q already registered", id)
	}

	r.tools[id] = tool
	return nil
}

// Get retrieves a tool by ID
func (r *Registry) Get(id string) Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.tools[id]
}

// GetAll returns all registered tools
func (r *Registry) GetAll() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tools := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// GetFiltered returns tools that pass the filter function
func (r *Registry) GetFiltered(filter func(Tool) bool) []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var filtered []Tool
	for _, tool := range r.tools {
		if filter(tool) {
			filtered = append(filtered, tool)
		}
	}
	return filtered
}

// Has checks if a tool with the given ID exists
func (r *Registry) Has(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, exists := r.tools[id]
	return exists
}

// Remove removes a tool from the registry
func (r *Registry) Remove(id string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[id]; exists {
		delete(r.tools, id)
		return true
	}
	return false
}

// Count returns the number of registered tools
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.tools)
}

// Clear removes all tools from the registry
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tools = make(map[string]Tool)
}