# OpenCode Go Server

This is a Go implementation of the OpenCode server, designed to be a high-performance alternative to the TypeScript server while maintaining API compatibility.

## Architecture

The Go server follows the same architectural patterns as the TypeScript implementation:

- **Tool System**: Extensible framework for AI assistant capabilities
- **HTTP API**: REST endpoints compatible with existing clients
- **SSE Events**: Real-time updates via Server-Sent Events
- **Session Management**: Conversation state management

## Project Structure

```
server-go/
├── cmd/
│   └── main.go              # Server entry point
├── internal/
│   ├── server/
│   │   ├── server.go        # Main server using rweb
│   │   └── event.go         # Event bus for SSE
│   ├── tool/
│   │   ├── tool.go          # Tool interface & registry
│   │   └── execute.go       # Tool execution engine
│   ├── tools/               # Individual tool implementations
│   │   ├── read.go          # File reading tool
│   │   └── bash.go          # Command execution tool
│   ├── schema/
│   │   └── schema.go        # Parameter validation (like Zod)
│   └── provider/            # AI provider integrations (TODO)
└── go.mod
```

## Tool Interface

Tools implement a simple interface:

```go
type Tool interface {
    ID() string
    Description() string
    Parameters() Schema
    Execute(ctx Context, params map[string]any) (Result, error)
}
```

The execution context provides:
- Session and message IDs
- Cancellation support via context
- Metadata callback for real-time updates

## Running the Server

```bash
# From the server-go directory
go run cmd/main.go -port 4096 -verbose

# Or build and run
go build -o opencode-server cmd/main.go
./opencode-server
```

## API Endpoints

The server implements these endpoints for compatibility:

- `GET /health` - Health check
- `POST /session` - Create new session
- `GET /session/:id` - Get session details
- `POST /session/:id/message` - Send message
- `GET /tools` - List available tools
- `GET /event` - SSE event stream
- `GET /provider` - List providers (stub)

## Tool Implementation Status

- [x] Read - Read files with line numbers
- [x] Bash - Execute shell commands
- [ ] Write - Write files
- [ ] Edit - Edit files with text replacement
- [ ] MultiEdit - Multiple edits in one operation
- [ ] Glob - Find files by pattern
- [ ] Grep - Search file contents
- [ ] LS - List directory contents
- [ ] TodoRead/Write - Task management
- [ ] WebFetch - Fetch web content
- [ ] LSP tools - Language server integration

## Development

### Adding a New Tool

1. Create a new file in `internal/tools/`
2. Implement the `Tool` interface
3. Register in `server.go`:
   ```go
   registry.Register(tools.NewYourTool())
   ```

### Schema Validation

The schema package provides type-safe parameter validation:

```go
schema.Object(map[string]tool.Schema{
    "path": schema.String().Describe("File path"),
    "limit": schema.Optional(schema.Number()),
}, "path") // "path" is required
```

## Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...
```

## Performance Considerations

- Concurrent tool execution with goroutines
- Efficient SSE event broadcasting
- Minimal memory allocations
- Context-based cancellation

## Future Enhancements

1. **Provider Integration**: Connect to AI providers (Anthropic, OpenAI, etc.)
2. **Permission System**: Add permission checks for sensitive operations
3. **MCP Protocol**: Support Model Context Protocol servers
4. **Tool Documentation**: Load descriptions from .txt files
5. **State Persistence**: Save sessions to disk
6. **Metrics**: Add performance monitoring

## Differences from TypeScript Implementation

While maintaining API compatibility, the Go implementation offers:

- Single binary deployment
- Lower memory footprint
- Better concurrent execution
- Native performance
- Simplified dependency management

The goal is to provide a drop-in replacement that can scale better for production use cases.