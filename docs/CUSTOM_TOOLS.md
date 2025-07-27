# Custom Tools Guide for RCode

## Overview

RCode supports custom tools through a plugin system that allows you to extend the AI assistant's capabilities with your own tools. Custom tools are loaded dynamically at startup and integrate seamlessly with the built-in tools.

## Quick Start

1. **Enable custom tools:**
   ```bash
   export RCODE_CUSTOM_TOOLS_ENABLED=true
   ```

2. **Create a custom tool:**
   ```bash
   # Copy the template
   cp -r tools/plugin_template ~/.rcode/my_tool
   cd ~/.rcode/my_tool
   
   # Edit the tool implementation
   vim my_tool.go
   ```

3. **Build the plugin:**
   ```bash
   ./build.sh
   ```

4. **Restart RCode:**
   ```bash
   rcode
   ```

Your custom tool will now be available alongside the built-in tools!

## Architecture

### Plugin System

RCode uses Go's plugin system to load custom tools at runtime. Plugins are compiled as shared libraries (`.so` files) and loaded from configured directories.

**Default search paths:**
- `~/.rcode/tools/` - User-specific tools
- `/usr/local/lib/rcode/tools/` - System-wide tools
- Additional paths via `RCODE_CUSTOM_TOOLS_PATHS` environment variable

### Security Model

Custom tools run in a sandboxed environment with capability-based security:

1. **Capability Declaration**: Tools must declare what resources they need
2. **Path Sandboxing**: File operations are restricted to the project directory
3. **Resource Limits**: Execution time and output size are limited
4. **Validation**: All inputs are validated before execution

## Plugin Development

### Required Interface

Your plugin must implement the `ToolPlugin` interface:

```go
type ToolPlugin interface {
    GetDefinition() Tool              // Tool metadata and schema
    Execute(ctx, input) (string, error) // Tool execution
    Initialize(config) error          // Setup
    Cleanup() error                   // Teardown
    GetCapabilities() ToolCapabilities // Security capabilities
}
```

### Capabilities

Define what your tool needs access to:

```go
type ToolCapabilities struct {
    FileRead      bool   // Can read files
    FileWrite     bool   // Can write files
    NetworkAccess bool   // Can make network requests
    ProcessSpawn  bool   // Can spawn processes
    WorkingDir    string // Restricted directory (empty = project root)
}
```

### Input Schema

Define your tool's parameters using JSON Schema:

```go
InputSchema: map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "path": map[string]interface{}{
            "type": "string",
            "description": "File path to process",
        },
        "format": map[string]interface{}{
            "type": "string",
            "enum": []string{"json", "yaml", "toml"},
            "default": "json",
        },
    },
    "required": []string{"path"},
}
```

## Example Tools

### 1. JSON Formatter Tool

A tool that formats JSON files:

```go
package main

import (
    "context"
    "encoding/json"
    "io/ioutil"
    "rcode/tools"
)

var Tool JSONFormatter
var Metadata = &tools.PluginMetadata{
    Name:    "json_format",
    Version: "1.0.0",
    Author:  "Your Name",
}

type JSONFormatter struct{}

func (t JSONFormatter) GetDefinition() tools.Tool {
    return tools.Tool{
        Name:        "json_format",
        Description: "Format and validate JSON files",
        InputSchema: // ... schema definition
    }
}

func (t JSONFormatter) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
    path := input["path"].(string)
    
    // Read file
    data, err := ioutil.ReadFile(path)
    if err != nil {
        return "", err
    }
    
    // Parse and format JSON
    var obj interface{}
    if err := json.Unmarshal(data, &obj); err != nil {
        return "", err
    }
    
    formatted, err := json.MarshalIndent(obj, "", "  ")
    if err != nil {
        return "", err
    }
    
    // Write back
    if err := ioutil.WriteFile(path, formatted, 0644); err != nil {
        return "", err
    }
    
    return "JSON file formatted successfully", nil
}

func (t JSONFormatter) GetCapabilities() tools.ToolCapabilities {
    return tools.ToolCapabilities{
        FileRead:  true,
        FileWrite: true,
    }
}

// ... other interface methods
```

### 2. Database Query Tool

A tool that executes database queries:

```go
func (t DBQueryTool) GetCapabilities() tools.ToolCapabilities {
    return tools.ToolCapabilities{
        NetworkAccess: true,  // For database connections
        WorkingDir:    "db",  // Restrict to db directory for config files
    }
}
```

### 3. Code Generator Tool

A tool that generates boilerplate code:

```go
func (t CodeGenTool) GetCapabilities() tools.ToolCapabilities {
    return tools.ToolCapabilities{
        FileWrite:  true,
        WorkingDir: "src",  // Only write to src directory
    }
}
```

## Best Practices

### 1. Error Handling

Always return descriptive errors:

```go
if path == "" {
    return "", fmt.Errorf("path parameter is required")
}

if !strings.HasSuffix(path, ".json") {
    return "", fmt.Errorf("file must have .json extension, got %s", path)
}
```

### 2. Context Handling

Check for cancellation in long operations:

```go
select {
case <-ctx.Done():
    return "", ctx.Err()
default:
    // Continue processing
}
```

### 3. Resource Management

Clean up resources properly:

```go
func (t MyTool) Initialize(config map[string]interface{}) error {
    // Open database connection
    db, err := sql.Open("postgres", connStr)
    t.db = db
    return err
}

func (t MyTool) Cleanup() error {
    if t.db != nil {
        return t.db.Close()
    }
    return nil
}
```

### 4. Input Validation

Validate all inputs thoroughly:

```go
func validatePath(path string) error {
    if path == "" {
        return errors.New("path cannot be empty")
    }
    
    if strings.Contains(path, "..") {
        return errors.New("path traversal not allowed")
    }
    
    if filepath.IsAbs(path) {
        return errors.New("absolute paths not allowed")
    }
    
    return nil
}
```

## Configuration

### Environment Variables

- `RCODE_CUSTOM_TOOLS_ENABLED` - Enable/disable custom tools (default: false)
- `RCODE_CUSTOM_TOOLS_PATHS` - Colon-separated list of directories to search
- `RCODE_CUSTOM_TOOLS_CONFIG` - Path to tools configuration file

### Configuration File

Create `~/.rcode/tools.json` to configure tools:

```json
{
  "tools": {
    "json_format": {
      "enabled": true,
      "config": {
        "indent": 2,
        "sort_keys": true
      }
    },
    "db_query": {
      "enabled": false,
      "reason": "Disabled for security audit"
    }
  }
}
```

## Troubleshooting

### Plugin Won't Load

1. **Check build mode:**
   ```bash
   go build -buildmode=plugin -o tool.so tool.go
   ```

2. **Verify exports:**
   - Must export `Tool` variable (implements ToolPlugin)
   - Must export `Metadata` variable (type *PluginMetadata)

3. **Check Go version:**
   - Plugin and RCode must use same Go version
   - Check with `go version`

### Plugin Crashes

1. **Enable debug logging:**
   ```bash
   RCODE_DEBUG=true rcode
   ```

2. **Check capabilities:**
   - Ensure tool has required capabilities
   - File operations need FileRead/FileWrite
   - Network operations need NetworkAccess

3. **Validate inputs:**
   - Check for nil pointers
   - Validate type assertions

### Permission Denied

1. **Check sandboxing:**
   - File paths must be within project directory
   - Use relative paths, not absolute

2. **Check capabilities:**
   - Tool must declare needed capabilities
   - WorkingDir restricts to subdirectory

## Security Considerations

### Do's

- ✅ Validate all inputs
- ✅ Use minimal capabilities
- ✅ Sanitize file paths
- ✅ Handle errors gracefully
- ✅ Respect context cancellation
- ✅ Clean up resources

### Don'ts

- ❌ Accept arbitrary commands
- ❌ Use absolute file paths
- ❌ Bypass sandboxing
- ❌ Store credentials in code
- ❌ Ignore resource limits
- ❌ Leave connections open

## Advanced Topics

### Tool Composition

Tools can work together through the context system:

```go
// Tool A writes analysis to context
ctx.SetVariable("analysis_result", result)

// Tool B reads from context
if analysis, ok := ctx.GetVariable("analysis_result"); ok {
    // Use analysis result
}
```

### Async Operations

For long-running operations:

```go
func (t MyTool) Execute(ctx context.Context, input map[string]interface{}) (string, error) {
    // Start async operation
    resultChan := make(chan string)
    errChan := make(chan error)
    
    go func() {
        // Long operation
        result, err := performOperation()
        if err != nil {
            errChan <- err
        } else {
            resultChan <- result
        }
    }()
    
    // Wait with timeout
    select {
    case result := <-resultChan:
        return result, nil
    case err := <-errChan:
        return "", err
    case <-ctx.Done():
        return "", ctx.Err()
    }
}
```

### Testing Your Plugin

Create a test file:

```go
package main

import (
    "testing"
    "rcode/tools"
)

func TestMyTool(t *testing.T) {
    tool := MyTool{}
    tp := tools.NewTestPlugin(t, tool)
    
    // Test definition
    tp.TestDefinition()
    
    // Test execution
    tp.TestExecution([]tools.TestCase{
        {
            Name: "valid input",
            Input: map[string]interface{}{
                "path": "test.json",
            },
            ExpectedOutput: "Success",
        },
        {
            Name: "missing path",
            Input: map[string]interface{}{},
            ExpectError: true,
        },
    })
}
```

## Contributing

If you create a useful custom tool, consider contributing it to the RCode community:

1. Ensure your tool follows best practices
2. Add comprehensive tests
3. Document the tool thoroughly
4. Submit a pull request to the [rcode-community-tools](https://github.com/rcode/community-tools) repository

## Support

- **Documentation**: This guide and the plugin template
- **Examples**: See `tools/plugin_examples/` directory
- **Community**: Share tools and get help at [RCode Discussions](https://github.com/rcode/discussions)
- **Issues**: Report bugs at [RCode Issues](https://github.com/rcode/issues)