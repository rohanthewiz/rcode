# Go Server Development Notes

## Current Status (2025-01-04)

### What We Built
- Created foundation for Go-based tool execution server
- Implemented core tool interface matching TypeScript design
- Built schema validation system (similar to Zod)
- Created tool registry and execution engine with proper context handling
- Implemented two example tools: `read` and `bash`
- Set up HTTP server with rweb and SSE support for real-time updates

### Architecture Decisions
1. **Tool Interface**: Mirrors TypeScript with ID, Description, Parameters, and Execute methods
2. **Schema Validation**: Native Go implementation providing type safety without runtime overhead
3. **Concurrent Execution**: Leverages goroutines for parallel tool execution
4. **Context Cancellation**: Uses Go's context.Context for abort signals
5. **API Compatibility**: Maintains same REST endpoints as TypeScript server

### Next Implementation Steps

#### Remaining Core Tools
1. **Write Tool**: File writing with safety checks
2. **Edit Tool**: Sophisticated text replacement (most complex)
3. **MultiEdit Tool**: Batch edit operations
4. **Glob Tool**: File pattern matching
5. **Grep Tool**: Content search
6. **LS Tool**: Directory listing
7. **Todo Tools**: Task management (read/write)
8. **WebFetch Tool**: HTTP content fetching

#### Provider Integration
- Need to implement provider abstraction layer
- Connect to AI providers (Anthropic, OpenAI, Google, etc.)
- Handle provider-specific tool mappings
- Implement streaming responses

#### Advanced Features
1. **Permission System**: Add security checks for sensitive operations
2. **MCP Protocol**: Support external tool servers
3. **Tool Documentation**: Load from .txt files like TypeScript
4. **State Persistence**: Save sessions to disk/database
5. **Metrics/Monitoring**: Add observability

### Technical Considerations

#### Edit Tool Implementation
The Edit tool is the most complex, requiring:
- Multiple text matching strategies (exact, trimmed, block anchor)
- Indentation handling
- Whitespace normalization
- Context-aware replacement

Consider porting the TypeScript logic carefully to maintain compatibility.

#### Session Management
Current implementation is in-memory. Need to add:
- Persistent storage (file system or database)
- Session expiration
- Cleanup routines
- Parent/child session relationships

#### Provider Communication
Need to implement:
- SDK client for each provider
- Streaming response handling
- Token counting
- Rate limiting
- Error recovery

### Testing Strategy
1. **Unit Tests**: Each tool in isolation
2. **Integration Tests**: Full request/response flow
3. **Compatibility Tests**: Ensure API matches TypeScript
4. **Performance Tests**: Benchmark vs TypeScript implementation
5. **Load Tests**: Concurrent request handling

### Performance Goals
- Single binary deployment (< 50MB)
- Memory usage < 100MB for typical workload
- Response time < 10ms for tool execution start
- Support 1000+ concurrent sessions

### Development Environment
- Go 1.22+ required
- Uses rohanthewiz packages (rweb, serr, logger, element)
- No external dependencies for core functionality
- Optional dependencies for specific tools (e.g., git for version control tools)

## Questions to Resolve
1. Should we maintain exact API compatibility or optimize for Go patterns?
2. How to handle tool versioning as we add features?
3. Should provider integration be a plugin system?
4. Database choice for session persistence (SQLite, PostgreSQL, etc.)?
5. How to handle file system permissions and sandboxing?

## Resources
- TypeScript implementation: `/packages/opencode/src/tool/`
- Tool descriptions: `/packages/opencode/src/tool/tools/*.txt`
- Provider logic: `/packages/opencode/src/provider/`
- Session handling: `/packages/opencode/src/session/`