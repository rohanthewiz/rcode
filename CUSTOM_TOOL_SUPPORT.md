# Custom Tool Support Implementation Plan for RCode

## Overview
This document outlines the implementation plan for adding custom tool support to RCode, allowing users to extend the tool system with their own custom tools without modifying the core codebase.

## Current Architecture Analysis

### Tool System Components
1. **Tool Interface** (`tools/tool.go`):
   - `Tool` struct: Defines tool metadata (name, description, input schema)
   - `Executor` interface: Single method `Execute(input map[string]interface{}) (string, error)`
   - `Registry`: Manages tool registration and execution
   - `EnhancedRegistry`: Adds validation, metrics, retry policies, and hooks

2. **Tool Registration Flow**:
   - Tools are registered during startup in `DefaultRegistry()` or `DefaultEnhancedRegistry()`
   - Each tool implements `GetDefinition()` to return its `Tool` metadata
   - Tools implement the `Executor` interface
   - Registration is static - all tools are hardcoded in the default registry functions

3. **Tool Usage Points**:
   - `web/session.go`: Creates registry with `tools.DefaultRegistry()`
   - `web/context_handlers.go`: Uses registry for tool suggestions
   - `planner/executor.go`: Uses registry for multi-step execution

## Implementation Plan

### Phase 1: Core Infrastructure

#### 1.1 Plugin Interface Definition
Create `tools/plugin.go`:
```go
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
    FileRead     bool   // Can read files
    FileWrite    bool   // Can write files
    NetworkAccess bool  // Can make network requests
    ProcessSpawn  bool  // Can spawn processes
    WorkingDir   string // Restricted working directory (empty = project root)
}

// PluginMetadata contains information about a plugin
type PluginMetadata struct {
    Name        string
    Version     string
    Author      string
    Description string
    MinRCodeVersion string
    MaxRCodeVersion string
}
```

#### 1.2 Plugin Loader
Create `tools/loader.go`:
```go
package tools

import (
    "fmt"
    "path/filepath"
    "plugin"
    "github.com/rohanthewiz/logger"
    "github.com/rohanthewiz/serr"
)

// PluginLoader handles loading custom tools
type PluginLoader struct {
    searchPaths []string
    loadedPlugins map[string]*LoadedPlugin
}

// LoadedPlugin represents a loaded plugin instance
type LoadedPlugin struct {
    Path       string
    Plugin     ToolPlugin
    Metadata   PluginMetadata
    Enabled    bool
}

// NewPluginLoader creates a new plugin loader
func NewPluginLoader(searchPaths []string) *PluginLoader {
    return &PluginLoader{
        searchPaths: searchPaths,
        loadedPlugins: make(map[string]*LoadedPlugin),
    }
}

// LoadPlugins discovers and loads all plugins from search paths
func (pl *PluginLoader) LoadPlugins() error {
    for _, searchPath := range pl.searchPaths {
        // Find .so files (compiled Go plugins)
        matches, err := filepath.Glob(filepath.Join(searchPath, "*.so"))
        if err != nil {
            logger.LogErr(err, "failed to search for plugins", "path", searchPath)
            continue
        }
        
        for _, pluginPath := range matches {
            if err := pl.loadPlugin(pluginPath); err != nil {
                logger.LogErr(err, "failed to load plugin", "path", pluginPath)
                // Continue loading other plugins
            }
        }
    }
    return nil
}

// loadPlugin loads a single plugin
func (pl *PluginLoader) loadPlugin(path string) error {
    // Load the Go plugin
    p, err := plugin.Open(path)
    if err != nil {
        return serr.Wrap(err, "failed to open plugin")
    }
    
    // Look for the required symbol
    sym, err := p.Lookup("Tool")
    if err != nil {
        return serr.Wrap(err, "plugin missing 'Tool' symbol")
    }
    
    // Assert to ToolPlugin interface
    toolPlugin, ok := sym.(ToolPlugin)
    if !ok {
        return serr.New("plugin 'Tool' does not implement ToolPlugin interface")
    }
    
    // Get plugin metadata
    metaSym, err := p.Lookup("Metadata")
    if err != nil {
        return serr.Wrap(err, "plugin missing 'Metadata' symbol")
    }
    
    metadata, ok := metaSym.(PluginMetadata)
    if !ok {
        return serr.New("plugin 'Metadata' is not of type PluginMetadata")
    }
    
    // Initialize the plugin
    if err := toolPlugin.Initialize(nil); err != nil {
        return serr.Wrap(err, "plugin initialization failed")
    }
    
    // Store the loaded plugin
    pl.loadedPlugins[metadata.Name] = &LoadedPlugin{
        Path:     path,
        Plugin:   toolPlugin,
        Metadata: metadata,
        Enabled:  true,
    }
    
    logger.Info("Loaded custom tool plugin", 
        "name", metadata.Name,
        "version", metadata.Version,
        "author", metadata.Author)
    
    return nil
}

// GetPlugins returns all loaded plugins
func (pl *PluginLoader) GetPlugins() map[string]*LoadedPlugin {
    return pl.loadedPlugins
}

// RegisterWithRegistry adds all loaded plugins to a registry
func (pl *PluginLoader) RegisterWithRegistry(registry *Registry) error {
    for name, loadedPlugin := range pl.loadedPlugins {
        if !loadedPlugin.Enabled {
            continue
        }
        
        // Create an executor adapter
        executor := &PluginExecutorAdapter{
            plugin: loadedPlugin.Plugin,
        }
        
        // Register with the registry
        registry.Register(loadedPlugin.Plugin.GetDefinition(), executor)
        
        logger.Debug("Registered custom tool with registry", "tool", name)
    }
    return nil
}
```

#### 1.3 Plugin Executor Adapter
Add to `tools/loader.go`:
```go
// PluginExecutorAdapter adapts a ToolPlugin to the Executor interface
type PluginExecutorAdapter struct {
    plugin ToolPlugin
}

// Execute implements the Executor interface
func (a *PluginExecutorAdapter) Execute(input map[string]interface{}) (string, error) {
    // Create a context with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    // Execute the plugin
    return a.plugin.Execute(ctx, input)
}
```

### Phase 2: Configuration & Integration

#### 2.1 Extend Configuration
Update `config/config.go`:
```go
type Config struct {
    // ... existing fields ...
    
    // Custom tool configuration
    CustomToolsEnabled bool
    CustomToolsPaths   []string // Directories to search for custom tools
    CustomToolsConfig  string   // Path to custom tools config file
}

// Add getter functions
func getCustomToolsEnabled() bool {
    return os.Getenv("RCODE_CUSTOM_TOOLS_ENABLED") == "true"
}

func getCustomToolsPaths() []string {
    paths := []string{
        filepath.Join(os.Getenv("HOME"), ".rcode", "tools"),
        "/usr/local/lib/rcode/tools",
    }
    
    if envPaths := os.Getenv("RCODE_CUSTOM_TOOLS_PATHS"); envPaths != "" {
        paths = append(paths, strings.Split(envPaths, ":")...)
    }
    
    return paths
}
```

#### 2.2 Modify Registry Creation
Update `tools/default.go`:
```go
// DefaultRegistryWithPlugins creates a registry with default tools and plugins
func DefaultRegistryWithPlugins() (*Registry, error) {
    registry := DefaultRegistry()
    
    // Load custom tools if enabled
    cfg := config.Get()
    if cfg.CustomToolsEnabled {
        loader := NewPluginLoader(cfg.CustomToolsPaths)
        if err := loader.LoadPlugins(); err != nil {
            logger.LogErr(err, "failed to load custom tool plugins")
            // Continue with built-in tools only
        } else {
            if err := loader.RegisterWithRegistry(registry); err != nil {
                logger.LogErr(err, "failed to register custom tools")
            }
        }
    }
    
    return registry, nil
}
```

#### 2.3 Update Tool Registry Usage
Update initialization points to use the new registry function:
- `web/session.go`
- `web/context_handlers.go`
- `planner/executor.go`

### Phase 3: Safety & Sandboxing

#### 3.1 Capability-Based Security
Create `tools/sandbox.go`:
```go
package tools

import (
    "os"
    "path/filepath"
    "strings"
)

// SandboxedExecutor wraps a plugin executor with safety checks
type SandboxedExecutor struct {
    executor     Executor
    capabilities ToolCapabilities
    projectRoot  string
}

// NewSandboxedExecutor creates a sandboxed executor
func NewSandboxedExecutor(executor Executor, capabilities ToolCapabilities, projectRoot string) *SandboxedExecutor {
    return &SandboxedExecutor{
        executor:     executor,
        capabilities: capabilities,
        projectRoot:  projectRoot,
    }
}

// Execute runs the tool with sandbox restrictions
func (s *SandboxedExecutor) Execute(input map[string]interface{}) (string, error) {
    // Pre-execution validation based on capabilities
    if err := s.validateInput(input); err != nil {
        return "", err
    }
    
    // Execute with monitoring
    result, err := s.executor.Execute(input)
    
    // Post-execution validation
    if err := s.validateOutput(result); err != nil {
        return "", err
    }
    
    return result, err
}

// validateInput checks if the input is allowed based on capabilities
func (s *SandboxedExecutor) validateInput(input map[string]interface{}) error {
    // Check file paths if file operations are involved
    if path, ok := GetString(input, "path"); ok {
        if !s.capabilities.FileRead && !s.capabilities.FileWrite {
            return serr.New("tool does not have file access capability")
        }
        
        // Ensure path is within allowed directory
        if err := s.validatePath(path); err != nil {
            return err
        }
    }
    
    // Check network URLs if present
    if url, ok := GetString(input, "url"); ok {
        if !s.capabilities.NetworkAccess {
            return serr.New("tool does not have network access capability")
        }
    }
    
    return nil
}

// validatePath ensures the path is within allowed boundaries
func (s *SandboxedExecutor) validatePath(path string) error {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return serr.Wrap(err, "invalid path")
    }
    
    allowedRoot := s.projectRoot
    if s.capabilities.WorkingDir != "" {
        allowedRoot = filepath.Join(s.projectRoot, s.capabilities.WorkingDir)
    }
    
    if !strings.HasPrefix(absPath, allowedRoot) {
        return serr.New("path is outside allowed directory")
    }
    
    return nil
}
```

### Phase 4: Plugin Development Support

#### 4.1 Plugin Template
Create `tools/plugin_template/example_tool.go`:
```go
package main

import (
    "context"
    "fmt"
    "rcode/tools"
)

// Tool is the exported symbol that RCode looks for
var Tool ExampleTool

// Metadata provides information about this plugin
var Metadata = tools.PluginMetadata{
    Name:        "example_tool",
    Version:     "1.0.0",
    Author:      "Your Name",
    Description: "An example custom tool",
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
            },
            "required": []string{"message"},
        },
    }
}

// Execute runs the tool
func (t ExampleTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
    message, ok := input["message"].(string)
    if !ok {
        return "", fmt.Errorf("message parameter is required")
    }
    
    // Check context for cancellation
    select {
    case <-ctx.Done():
        return "", ctx.Err()
    default:
    }
    
    // Process the message
    result := fmt.Sprintf("Processed: %s", message)
    
    return result, nil
}

// Initialize sets up the tool
func (t *ExampleTool) Initialize(config map[string]interface{}) error {
    t.config = config
    return nil
}

// Cleanup cleans up resources
func (t ExampleTool) Cleanup() error {
    // Clean up any resources
    return nil
}

// GetCapabilities returns what this tool can do
func (t ExampleTool) GetCapabilities() tools.ToolCapabilities {
    return tools.ToolCapabilities{
        FileRead:     false,
        FileWrite:    false,
        NetworkAccess: false,
        ProcessSpawn:  false,
    }
}
```

#### 4.2 Build Script
Create `tools/plugin_template/build.sh`:
```bash
#!/bin/bash
# Build script for RCode custom tool plugins

TOOL_NAME="example_tool"
OUTPUT_DIR="$HOME/.rcode/tools"

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"

# Build the plugin
go build -buildmode=plugin -o "$OUTPUT_DIR/$TOOL_NAME.so" $TOOL_NAME.go

if [ $? -eq 0 ]; then
    echo "Successfully built $TOOL_NAME plugin"
    echo "Plugin location: $OUTPUT_DIR/$TOOL_NAME.so"
else
    echo "Failed to build plugin"
    exit 1
fi
```

### Phase 5: Testing & Documentation

#### 5.1 Plugin Testing Framework
Create `tools/plugin_test_utils.go`:
```go
package tools

import (
    "testing"
    "context"
)

// TestPlugin provides utilities for testing custom tools
type TestPlugin struct {
    t      *testing.T
    plugin ToolPlugin
}

// NewTestPlugin creates a test helper for a plugin
func NewTestPlugin(t *testing.T, plugin ToolPlugin) *TestPlugin {
    return &TestPlugin{
        t:      t,
        plugin: plugin,
    }
}

// TestDefinition validates the plugin definition
func (tp *TestPlugin) TestDefinition() {
    def := tp.plugin.GetDefinition()
    
    if def.Name == "" {
        tp.t.Error("Plugin name cannot be empty")
    }
    
    if def.Description == "" {
        tp.t.Error("Plugin description cannot be empty")
    }
    
    if def.InputSchema == nil {
        tp.t.Error("Plugin must define an input schema")
    }
}

// TestExecution tests plugin execution with sample inputs
func (tp *TestPlugin) TestExecution(testCases []TestCase) {
    for _, tc := range testCases {
        result, err := tp.plugin.Execute(context.Background(), tc.Input)
        
        if tc.ExpectError && err == nil {
            tp.t.Errorf("Expected error for input %v, but got none", tc.Input)
        }
        
        if !tc.ExpectError && err != nil {
            tp.t.Errorf("Unexpected error for input %v: %v", tc.Input, err)
        }
        
        if tc.ExpectedOutput != "" && result != tc.ExpectedOutput {
            tp.t.Errorf("Expected output %s, got %s", tc.ExpectedOutput, result)
        }
    }
}

// TestCase represents a test case for plugin execution
type TestCase struct {
    Name           string
    Input          map[string]interface{}
    ExpectedOutput string
    ExpectError    bool
}
```

#### 5.2 Documentation
Create `docs/CUSTOM_TOOLS.md`:
```markdown
# Custom Tools Guide for RCode

## Overview
RCode supports custom tools through a plugin system that allows you to extend the AI assistant's capabilities with your own tools.

## Quick Start

1. Enable custom tools:
   ```bash
   export RCODE_CUSTOM_TOOLS_ENABLED=true
   ```

2. Create a custom tool using the template:
   ```bash
   cp -r tools/plugin_template ~/.rcode/my_tool
   cd ~/.rcode/my_tool
   # Edit example_tool.go to implement your tool
   ```

3. Build your tool:
   ```bash
   ./build.sh
   ```

4. Restart RCode - your tool will be automatically loaded

## Plugin Development

### Required Interface
Your plugin must implement the `ToolPlugin` interface:
- `GetDefinition()` - Tool metadata
- `Execute()` - Tool logic
- `Initialize()` - Setup
- `Cleanup()` - Teardown
- `GetCapabilities()` - Security capabilities

### Input Schema
Define your tool's parameters using JSON Schema format...

### Error Handling
Return appropriate errors with context...

### Testing Your Plugin
Use the provided test utilities...

## Security Considerations
- Tools run with restricted capabilities
- File access is sandboxed to project directory
- Network access requires explicit capability
- Process spawning is disabled by default

## Examples
See the `tools/plugin_examples/` directory for more examples.
```

## Implementation Priority

1. **Phase 1 (Core)**: Plugin interface, loader, and basic integration
2. **Phase 2 (Config)**: Configuration and registry updates
3. **Phase 3 (Security)**: Sandboxing and capability-based security
4. **Phase 4 (Developer Experience)**: Templates and build tools
5. **Phase 5 (Polish)**: Testing framework and documentation

## Migration Strategy

1. The existing tool system remains unchanged
2. Custom tools are additive - they don't replace built-in tools
3. If a custom tool has the same name as a built-in tool, the built-in takes precedence
4. Custom tools are loaded after built-in tools

## Future Enhancements

1. **Plugin Marketplace**: Central repository for sharing tools
2. **JavaScript/Python Plugins**: Support for tools written in other languages
3. **Hot Reloading**: Reload plugins without restarting RCode
4. **Plugin Dependencies**: Allow plugins to depend on npm/pip packages
5. **GUI Configuration**: Web UI for managing custom tools
6. **Tool Composition**: Allow tools to call other tools
7. **Event Hooks**: Allow plugins to hook into RCode events

## Security Considerations

1. **Capability Model**: Tools declare what they need access to
2. **Sandboxing**: Execution is restricted based on capabilities
3. **Path Validation**: File operations are restricted to project directory
4. **Resource Limits**: CPU, memory, and time limits for tool execution
5. **Audit Logging**: All custom tool executions are logged
6. **User Approval**: Option to require user approval for custom tool operations

## Testing Strategy

1. **Unit Tests**: Test each component in isolation
2. **Integration Tests**: Test plugin loading and execution
3. **Security Tests**: Verify sandboxing and capability enforcement
4. **Example Plugins**: Provide well-tested example plugins
5. **Load Testing**: Ensure custom tools don't impact performance

## Documentation Requirements

1. **Developer Guide**: How to create custom tools
2. **API Reference**: Detailed interface documentation
3. **Security Guide**: Best practices for safe tool development
4. **Examples**: Multiple example tools showing different capabilities
5. **Troubleshooting**: Common issues and solutions