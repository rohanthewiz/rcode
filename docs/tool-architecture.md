# RCode Tool Architecture

This document describes the tool system architecture in OpenCode, analyzing the TypeScript implementation and proposing a Go server implementation.

## TypeScript Tool System Overview

### Core Tool Interface

The tool system in TypeScript is built around a clean, type-safe interface defined in `/packages/opencode/src/tool/tool.ts`:

```typescript
export namespace Tool {
  interface Metadata {
    title: string
    [key: string]: any
  }

  export type Context<M extends Metadata = Metadata> = {
    sessionID: string
    messageID: string
    abort: AbortSignal
    metadata(meta: M): void
  }

  export interface Info<
    Parameters extends StandardSchemaV1 = StandardSchemaV1,
    M extends Metadata = Metadata,
  > {
    id: string
    description: string
    parameters: Parameters
    execute(
      args: StandardSchemaV1.InferOutput<Parameters>,
      ctx: Context,
    ): Promise<{
      metadata: M
      output: string
    }>
  }
}
```

### Tool Definition Pattern

Tools are created using the `Tool.define()` method, which provides type safety and consistent structure:

```typescript
export const ReadTool = Tool.define({
  id: "read",
  description: DESCRIPTION, // Loaded from read.txt
  parameters: z.object({
    filePath: z.string().describe("The path to the file to read"),
    offset: z.number().optional(),
    limit: z.number().optional(),
  }),
  async execute(params, ctx) {
    // Tool implementation logic
    return {
      output: formattedContent,
      metadata: {
        preview: preview,
        title: relativePath,
      }
    }
  }
})
```

### Available Built-in Tools

The system includes these built-in tools:

- **File Operations**:
  - `read` - Read file contents with line numbers
  - `write` - Write content to files
  - `edit` - Edit files with sophisticated text replacement
  - `multiedit` - Multiple edits in a single operation
  - `patch` - Apply diff patches

- **File Search**:
  - `glob` - Find files by pattern
  - `grep` - Search file contents
  - `ls` - List directory contents

- **Execution**:
  - `bash` - Execute shell commands

- **Web**:
  - `webfetch` - Fetch and process web content

- **Organization**:
  - `todoread` - Read task list
  - `todowrite` - Write task list

- **Development**:
  - `lsp-diagnostics` - Language server diagnostics
  - `lsp-hover` - Language server hover information

### Tool Execution Flow

1. **Provider Request**: AI provider requests tool execution as part of message generation
2. **Session Integration**: Tools are registered per-session with the AI SDK
3. **Parameter Validation**: Zod schemas validate input parameters
4. **Execution Context**: Each tool receives:
   - Session ID for scoping
   - Message ID for tracking
   - Abort signal for cancellation
   - Metadata callback for real-time updates
5. **Result Streaming**: Metadata updates stream to clients via SSE
6. **Error Handling**: Errors are caught and returned as tool output

### Provider-Specific Tool Mapping

Different AI providers have different tool capabilities:

```typescript
const TOOL_MAPPING: Record<string, Tool.Info[]> = {
  anthropic: TOOLS.filter((t) => t.id !== "patch"),
  openai: TOOLS.map((t) => ({
    ...t,
    parameters: optionalToNullable(t.parameters),
  })),
  azure: TOOLS.map((t) => ({
    ...t,
    parameters: optionalToNullable(t.parameters),
  })),
  google: TOOLS,
}
```

### Advanced Features

1. **MCP Integration**: External Model Context Protocol servers can provide additional tools
2. **Permission System**: Sensitive operations can require user permission
3. **Event Publishing**: Tools publish events for file changes
4. **Tool Documentation**: Each tool has a `.txt` file with detailed usage instructions

## Current Client-Server Architecture

### Go TUI as Client

The Go TUI (`packages/tui`) operates as a client application:

1. **Spawns TypeScript Server**: On startup, launches the backend server
2. **HTTP Communication**: Uses generated SDK for API calls
3. **SSE Subscription**: Receives real-time updates via Server-Sent Events
4. **Tool Result Display**: Renders tool execution results in chat interface

### TypeScript Backend Server

The server (`packages/opencode`) handles:

1. **HTTP API**: REST endpoints for sessions, messages, and configuration
2. **Tool Execution**: Runs tools server-side with proper sandboxing
3. **Provider Integration**: Manages different AI provider connections
4. **State Management**: Persists sessions and messages

## Proposed Go Server Implementation

### Architecture Goals

1. **API Compatibility**: Drop-in replacement for TypeScript server
2. **Performance**: Leverage Go's concurrency and efficiency
3. **Type Safety**: Strong typing without runtime overhead
4. **Maintainability**: Clear separation of concerns

### Core Components

#### Tool Interface

```go
type Tool interface {
    ID() string
    Description() string
    Parameters() Schema
    Execute(ctx ToolContext, params map[string]any) (ToolResult, error)
}

type ToolContext struct {
    SessionID string
    MessageID string
    Abort     context.Context
    Metadata  func(meta map[string]any)
}

type ToolResult struct {
    Output   string
    Metadata map[string]any
}
```

#### Tool Registry

```go
type Registry struct {
    tools map[string]Tool
    mu    sync.RWMutex
}

func (r *Registry) Register(tool Tool) error
func (r *Registry) Get(id string) (Tool, bool)
func (r *Registry) List() []Tool
func (r *Registry) ForProvider(provider string) []Tool
```

#### Schema Validation

Replace Zod with a Go-native solution:

```go
type Schema interface {
    Validate(value any) error
    Description() string
    ToJSON() map[string]any
}

type ObjectSchema struct {
    Properties map[string]Schema
    Required   []string
}

type StringSchema struct {
    MinLength *int
    MaxLength *int
    Pattern   *regexp.Regexp
}
```

### Implementation Strategy

1. **Phase 1: Foundation**
   - Server setup with rweb
   - Tool interface and registry
   - Basic schema validation

2. **Phase 2: Core Tools**
   - File operations (read, write, edit)
   - Bash execution
   - Search tools (glob, grep)

3. **Phase 3: Integration**
   - Session management
   - Provider abstraction
   - SSE event streaming

4. **Phase 4: Advanced Features**
   - Permission system
   - MCP protocol support
   - Tool documentation loading

### Key Design Decisions

1. **Context for Cancellation**: Use Go's context.Context for abort signals
2. **Concurrent Execution**: Support parallel tool calls with goroutines
3. **Error Handling**: Consistent error wrapping with serr
4. **Metadata Updates**: Channel-based communication for real-time updates
5. **File Operations**: Careful handling of paths and permissions

### Testing Approach

1. **Unit Tests**: Each tool implementation tested in isolation
2. **Integration Tests**: Full tool execution flow
3. **Compatibility Tests**: Ensure API matches TypeScript server
4. **Performance Tests**: Benchmark against TypeScript implementation

## Benefits of Go Implementation

1. **Performance**: Native compilation and efficient concurrency
2. **Resource Usage**: Lower memory footprint
3. **Deployment**: Single binary distribution
4. **Type Safety**: Compile-time checking without runtime overhead
5. **Stability**: Go's mature standard library

## Migration Path

1. **Parallel Development**: Build alongside TypeScript server
2. **Feature Parity**: Match all existing functionality
3. **Gradual Rollout**: Test with subset of users
4. **Full Migration**: Replace TypeScript server once stable

This architecture provides a solid foundation for implementing a high-performance, maintainable tool system in Go while preserving the excellent design of the TypeScript implementation.
