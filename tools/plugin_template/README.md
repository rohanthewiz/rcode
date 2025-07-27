# RCode Custom Tool Plugin Template

This directory contains a template for creating custom tools for RCode.

## Quick Start

1. Copy this directory to create your own tool:
   ```bash
   cp -r tools/plugin_template ~/.rcode/my_tool
   cd ~/.rcode/my_tool
   ```

2. Rename and edit `example_tool.go`:
   - Change the tool name in the code
   - Implement your custom logic in the `Execute` method
   - Update the input schema to match your requirements
   - Set appropriate capabilities

3. Update `build.sh`:
   - Change the `TOOL_NAME` variable to match your tool

4. Build your plugin:
   ```bash
   ./build.sh
   ```

5. Enable custom tools and restart RCode:
   ```bash
   export RCODE_CUSTOM_TOOLS_ENABLED=true
   rcode
   ```

## Plugin Structure

### Required Exports

Your plugin must export two symbols:

1. `Tool` - An instance of your tool type that implements `tools.ToolPlugin`
2. `Metadata` - A pointer to `tools.PluginMetadata` with plugin information

### ToolPlugin Interface

Your tool must implement these methods:

- `GetDefinition()` - Returns tool metadata and input schema
- `Execute(ctx, input)` - Executes the tool with given input
- `Initialize(config)` - Called when plugin is loaded
- `Cleanup()` - Called when plugin is unloaded
- `GetCapabilities()` - Declares what the tool can access

### Capabilities

Define what your tool needs access to:

- `FileRead` - Can read files
- `FileWrite` - Can write files
- `NetworkAccess` - Can make network requests
- `ProcessSpawn` - Can spawn processes
- `WorkingDir` - Restrict to specific directory (empty = project root)

### Input Schema

Define your tool's parameters using JSON Schema format:

```go
InputSchema: map[string]interface{}{
    "type": "object",
    "properties": map[string]interface{}{
        "param1": map[string]interface{}{
            "type": "string",
            "description": "Description of param1",
        },
        "param2": map[string]interface{}{
            "type": "number",
            "description": "Description of param2",
            "default": 10,
        },
    },
    "required": []string{"param1"},
}
```

## Example Tools

See the `example_tool.go` for a basic example. More examples:

### File Processing Tool

```go
// A tool that counts lines in files
func (t LineCounterTool) GetCapabilities() tools.ToolCapabilities {
    return tools.ToolCapabilities{
        FileRead: true,  // Needs file read access
        WorkingDir: "",  // Can read from anywhere in project
    }
}
```

### Network Tool

```go
// A tool that fetches data from APIs
func (t APIFetcherTool) GetCapabilities() tools.ToolCapabilities {
    return tools.ToolCapabilities{
        NetworkAccess: true,  // Needs network access
    }
}
```

### Command Runner Tool

```go
// A tool that runs specific commands
func (t CommandRunnerTool) GetCapabilities() tools.ToolCapabilities {
    return tools.ToolCapabilities{
        ProcessSpawn: true,  // Needs to spawn processes
        WorkingDir: "scripts",  // Restricted to scripts directory
    }
}
```

## Best Practices

1. **Error Handling**: Always return descriptive errors
2. **Context Checking**: Check for cancellation in long-running operations
3. **Input Validation**: Validate all input parameters
4. **Resource Cleanup**: Clean up resources in the `Cleanup` method
5. **Minimal Capabilities**: Only request capabilities you actually need
6. **Logging**: Use appropriate logging for debugging

## Troubleshooting

### Plugin won't load
- Check that you're exporting the correct symbols
- Ensure the plugin implements all required methods
- Verify Go version compatibility

### Plugin crashes
- Add proper error handling
- Check for nil pointers
- Validate input parameters

### Permission denied
- Ensure your capabilities match what you're trying to do
- Check file paths are within allowed directories