# Custom Tool Architecture Diagram

## System Architecture

```mermaid
graph TB
    subgraph "RCode Core"
        Registry[Tool Registry]
        EnhancedRegistry[Enhanced Registry]
        DefaultTools[Built-in Tools<br/>read_file, write_file, etc.]
        
        Registry --> EnhancedRegistry
        DefaultTools --> Registry
    end
    
    subgraph "Plugin System"
        PluginLoader[Plugin Loader]
        PluginInterface[ToolPlugin Interface]
        SandboxExecutor[Sandboxed Executor]
        
        PluginLoader --> |discovers| PluginFiles[.so Plugin Files]
        PluginLoader --> |validates| PluginInterface
        PluginInterface --> |wrapped by| SandboxExecutor
    end
    
    subgraph "Custom Tools"
        CustomTool1[Custom Tool 1<br/>example_analyzer.so]
        CustomTool2[Custom Tool 2<br/>code_formatter.so]
        CustomTool3[Custom Tool 3<br/>api_client.so]
        
        CustomTool1 --> |implements| PluginInterface
        CustomTool2 --> |implements| PluginInterface
        CustomTool3 --> |implements| PluginInterface
    end
    
    subgraph "Configuration"
        Config[Config System]
        EnvVars[Environment Variables<br/>RCODE_CUSTOM_TOOLS_ENABLED<br/>RCODE_CUSTOM_TOOLS_PATHS]
        ToolPaths[Search Paths<br/>~/.rcode/tools<br/>/usr/local/lib/rcode/tools]
        
        EnvVars --> Config
        Config --> PluginLoader
        ToolPaths --> PluginLoader
    end
    
    subgraph "Security Layer"
        Capabilities[Capability Model<br/>FileRead, FileWrite<br/>NetworkAccess, ProcessSpawn]
        PathValidator[Path Validator]
        ResourceLimits[Resource Limits<br/>CPU, Memory, Time]
        
        SandboxExecutor --> Capabilities
        SandboxExecutor --> PathValidator
        SandboxExecutor --> ResourceLimits
    end
    
    subgraph "Integration Points"
        Session[Session Handler]
        ContextManager[Context Manager]
        TaskPlanner[Task Planner]
        
        Session --> |uses| EnhancedRegistry
        ContextManager --> |uses| EnhancedRegistry
        TaskPlanner --> |uses| EnhancedRegistry
    end
    
    PluginLoader --> |registers with| Registry
    SandboxExecutor --> |executes| CustomTools
```

## Plugin Lifecycle

```mermaid
sequenceDiagram
    participant User
    participant RCode
    participant Config
    participant PluginLoader
    participant Plugin
    participant Registry
    participant Sandbox
    
    User->>RCode: Start RCode
    RCode->>Config: Load configuration
    Config->>RCode: Custom tools enabled
    
    RCode->>PluginLoader: Initialize with paths
    PluginLoader->>PluginLoader: Scan directories
    
    loop For each .so file
        PluginLoader->>Plugin: Load plugin
        Plugin->>PluginLoader: Return ToolPlugin
        PluginLoader->>Plugin: Initialize()
        Plugin->>PluginLoader: Success
        PluginLoader->>Registry: Register tool
    end
    
    User->>RCode: Use custom tool
    RCode->>Registry: Execute tool
    Registry->>Sandbox: Wrap execution
    Sandbox->>Sandbox: Validate capabilities
    Sandbox->>Plugin: Execute()
    Plugin->>Sandbox: Return result
    Sandbox->>Registry: Return result
    Registry->>RCode: Return result
    RCode->>User: Display result
```

## Data Flow

```mermaid
graph LR
    subgraph "Input"
        UserRequest[User Request<br/>"use my_custom_tool"]
        ToolParams[Tool Parameters<br/>{message: "test"}]
    end
    
    subgraph "Validation"
        SchemaValidation[Schema Validation]
        CapabilityCheck[Capability Check]
        PathValidation[Path Validation]
    end
    
    subgraph "Execution"
        ContextSetup[Context Setup<br/>Timeout, Cancel]
        PluginExecute[Plugin.Execute()]
        ResultCapture[Result Capture]
    end
    
    subgraph "Output"
        SuccessResult[Success Result]
        ErrorResult[Error Result]
        Metrics[Metrics Update]
    end
    
    UserRequest --> SchemaValidation
    ToolParams --> SchemaValidation
    SchemaValidation --> CapabilityCheck
    CapabilityCheck --> PathValidation
    PathValidation --> ContextSetup
    ContextSetup --> PluginExecute
    PluginExecute --> ResultCapture
    ResultCapture --> SuccessResult
    ResultCapture --> ErrorResult
    ResultCapture --> Metrics
```

## Security Model

```mermaid
graph TB
    subgraph "Trust Levels"
        BuiltIn[Built-in Tools<br/>Full Trust]
        Verified[Verified Plugins<br/>High Trust]
        Community[Community Plugins<br/>Medium Trust]
        Unknown[Unknown Plugins<br/>Low Trust]
    end
    
    subgraph "Capabilities"
        NoAccess[No Access<br/>Computation Only]
        ReadOnly[Read Only<br/>File Read]
        ReadWrite[Read/Write<br/>File Operations]
        Network[Network<br/>HTTP/HTTPS]
        Process[Process<br/>Spawn Commands]
    end
    
    subgraph "Restrictions"
        PathRestrict[Path Restrictions<br/>Project Directory Only]
        TimeLimit[Time Limits<br/>5 minute max]
        ResourceLimit[Resource Limits<br/>Memory, CPU]
        AuditLog[Audit Logging<br/>All Operations]
    end
    
    BuiltIn --> ReadWrite
    BuiltIn --> Network
    BuiltIn --> Process
    
    Verified --> ReadWrite
    Verified --> Network
    Verified --> PathRestrict
    
    Community --> ReadOnly
    Community --> PathRestrict
    Community --> TimeLimit
    
    Unknown --> NoAccess
    Unknown --> TimeLimit
    Unknown --> ResourceLimit
    Unknown --> AuditLog
```

## File Structure

```
rcode/
├── tools/
│   ├── plugin.go              # Plugin interface definition
│   ├── loader.go              # Plugin loading logic
│   ├── sandbox.go             # Security sandboxing
│   ├── plugin_template/       # Template for new plugins
│   │   ├── example_tool.go
│   │   ├── build.sh
│   │   └── test_tool.go
│   └── plugin_examples/       # Example plugins
│       ├── json_validator/
│       ├── api_tester/
│       └── log_analyzer/
├── config/
│   └── config.go             # Extended with plugin config
└── ~/.rcode/
    ├── tools/                # User's custom tools
    │   ├── my_tool.so
    │   └── another_tool.so
    └── tools.json           # Tool configuration
```

## Configuration Example

```json
{
  "custom_tools": {
    "enabled": true,
    "search_paths": [
      "~/.rcode/tools",
      "/usr/local/lib/rcode/tools"
    ],
    "tools": {
      "my_analyzer": {
        "enabled": true,
        "capabilities": {
          "file_read": true,
          "file_write": false,
          "network_access": false,
          "process_spawn": false,
          "working_dir": "src/"
        },
        "config": {
          "max_file_size": "10MB",
          "timeout": "30s"
        }
      }
    }
  }
}
```

## Key Design Decisions

1. **Go Plugins**: Use Go's plugin system for native performance
2. **Interface-Based**: Clean separation between core and plugins
3. **Capability Model**: Explicit permissions for security
4. **Sandboxed Execution**: Prevent malicious behavior
5. **Configuration-Driven**: Easy to enable/disable tools
6. **Backward Compatible**: Doesn't break existing tools
7. **Developer-Friendly**: Templates and examples provided