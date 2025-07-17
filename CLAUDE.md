
# RCode Go Server - Project Context

## Overview
RCode is a Go-based web server that provides an AI-powered coding assistant interface. It uses Anthropic's Claude API with OAuth authentication for Claude Pro/Max subscribers.

## Project Structure
```
rcode/
├── main.go                    # Entry point
├── auth/
│   ├── anthropic.go          # OAuth implementation & client
│   ├── exchange.go           # Code exchange handler
│   ├── oauth_url.go          # OAuth URL generation
│   ├── logout.go             # Logout handler
│   └── storage.go            # Token persistence
├── web/
│   ├── routes.go             # Route definitions
│   ├── ui.go                 # Main UI with element
│   ├── auth_callback.go      # OAuth callback UI
│   ├── session.go            # Session management with init prompt & tool summaries
│   ├── sse.go                # SSE implementation with reconnection
│   ├── context_handlers.go   # Context API endpoints
│   └── assets/
│       ├── js/
│       │   ├── ui.js         # Main UI logic with SSE handling & tool summaries
│       │   └── login.js      # Login flow logic
│       └── css/
│           └── ui.css        # Dark theme styles with tool summary styling
├── providers/
│   └── anthropic.go          # Anthropic API client with retry & context integration
├── tools/
│   ├── tool.go               # Tool interface & registry
│   ├── default.go            # Default tool implementations
│   ├── read_file.go          # File reading tool
│   ├── write_file.go         # File writing tool
│   ├── bash.go               # Bash command tool
│   ├── edit_file.go          # Line-based file editing tool
│   ├── search.go             # Regex-based file search tool
│   ├── directory.go          # Directory operations (list, tree, mkdir, rm, move)
│   ├── git.go                # Git operations (status, diff, log, branch, add, commit, push, pull, checkout, merge)
│   ├── validation.go         # Tool parameter validation
│   ├── enhanced_registry.go  # Enhanced registry with validation, metrics & retry
│   ├── context_aware.go      # Context-aware tool execution
│   ├── retry.go              # Retry utility with exponential backoff
│   ├── errors.go             # Error classification system
│   └── web_*.go              # Web tools (search, fetch)
├── context/
│   ├── types.go              # Core context data structures
│   ├── manager.go            # Context manager with file tracking
│   ├── scanner.go            # Project scanner for language/framework detection
│   ├── prioritizer.go        # Smart file prioritization algorithm
│   ├── tracker.go            # Change tracking system
│   └── window.go             # Context window optimization
├── planner/
│   ├── types.go              # Task planning data structures
│   ├── planner.go            # Multi-step task execution
│   ├── executor.go           # Step execution with tool integration
│   └── analyzer.go           # Task analysis and breakdown
├── db/
│   └── *.go                  # Database layer with DuckDB
└── go.mod                    # Dependencies
```

## Core Technologies
- **Web Framework**: github.com/rohanthewiz/rweb
- **HTML Generation**: github.com/rohanthewiz/element
- **Error Handling**: github.com/rohanthewiz/serr
- **Logging**: github.com/rohanthewiz/logger
- **Database**: DuckDB (embedded)
- **Server Port**: 8000 (HTTP), 8443 (HTTPS)

## Authentication System
- **OAuth Provider**: Anthropic (Claude.ai)
- **Client ID**: `9d1c250a-e61b-44d9-88ed-5944d1962f5e`
- **OAuth Flow**: PKCE-based with manual code entry
- **Token Storage**: `~/.local/share/rcode/auth.json`
- **Auto-refresh**: Tokens refresh automatically when expired
- **Free Usage**: OAuth tokens provide free API access for Pro/Max users

## Key Features
1. **Chat Interface**: Web-based UI with session management
2. **Enhanced Tool System**: 
   - File operations: read, write, edit (line-based)
   - Directory operations: list, tree, mkdir, rm, move
   - Search: regex-based file content search
   - Git integration: status, diff, log, branch, add, commit, push, pull, checkout, merge
   - Web operations: search (mock), fetch and convert pages
   - Bash command execution
   - Tool parameter validation and safety checks
3. **Error Recovery System**:
   - Automatic retry with exponential backoff for transient failures
   - Error classification (retryable, permanent, rate limit)
   - Per-tool retry policies (network vs filesystem)
   - API 529 "Overloaded" errors handled transparently
4. **Context Intelligence**:
   - Automatic project language/framework detection
   - Smart file prioritization for relevant context
   - Change tracking during sessions
   - Context-aware tool suggestions
5. **Tool Usage Summaries**: Concise display of tool operations with metrics
6. **Real-time Updates**: Server-sent events (SSE) with robust reconnection
7. **Dark Theme**: Modern dark-themed UI with CSS variables
8. **Session Management**: Persistent sessions with DuckDB storage
9. **Auto-initialization**: Sessions start with permission prompt for tools/files
10. **Connection Recovery**: Exponential backoff and manual reconnection for SSE
11. **HTTPS Support**: Built-in TLS/SSL with automatic HTTP-to-HTTPS redirect

## API Endpoints

### Authentication
- `GET /auth/anthropic/oauth-url` - Get OAuth authorization URL
- `POST /auth/anthropic/exchange` - Exchange code for tokens
- `POST /auth/anthropic/refresh` - Refresh access token
- `POST /auth/logout` - Clear authentication
- `GET /auth/callback` - Manual code entry page

### Session Management
- `GET /api/app` - Application info & auth status
- `GET /api/session` - List all sessions
- `POST /api/session` - Create new session
- `DELETE /api/session/:id` - Delete session
- `POST /api/session/:id/message` - Send message to session (includes tool summaries)
- `GET /api/session/:id/messages` - Get session messages
- `GET /api/session/:id/prompts` - Get initial prompts for session
- `GET /events` - SSE endpoint for real-time updates

### Context Management
- `GET /api/context` - Get current project context
- `POST /api/context/scan` - Scan project and update context
- `GET /api/context/files/:task` - Get relevant files for a task
- `GET /api/context/metrics` - Get context metrics

## Development Notes

### Running the Server

#### HTTP Mode (default):
```bash
go run main.go
```
Then visit http://localhost:8000

#### HTTPS Mode:
```bash
# Generate certificates first (for development)
cd scripts && ./generate-certs.sh && cd ..

# Run with TLS enabled
RCODE_TLS_ENABLED=true go run main.go
```
Then visit https://localhost:8443

The server will automatically redirect HTTP traffic to HTTPS when TLS is enabled.

### OAuth Flow
1. User clicks login → Opens Claude.ai OAuth in new tab
2. User authorizes → Gets code from Anthropic
3. User pastes code → Server exchanges for tokens
4. Tokens stored with automatic refresh capability

### Configuration

The server can be configured using environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `MSG_PROXY` | Proxy URL for Anthropic API | Direct connection |
| `RCODE_TLS_ENABLED` | Enable HTTPS ("true" to enable) | false |
| `RCODE_TLS_PORT` | HTTPS port | :8443 |
| `RCODE_TLS_CERT` | Path to TLS certificate | certs/localhost.crt |
| `RCODE_TLS_KEY` | Path to TLS private key | certs/localhost.key |

### Important Implementation Details
- System prompt remains exactly: "You are Claude Code, Anthropic's official CLI for Claude."
- Context information is added as part of the initial user prompt, not the system prompt
- OAuth headers: `Authorization: Bearer {token}`, `anthropic-beta: oauth-2025-04-20`
- Messages use Anthropic's streaming API format
- Comprehensive tool system with 22 tools across file, directory, search, git, and web operations
- Sessions persist in DuckDB at `~/.local/share/rcode/rcode.db`
- Each session starts with configurable prompts (default includes permission requirements)
- Tool usage summaries display as "🛠️ TOOL USE" with concise metrics
- SSE reconnection: 5 attempts with exponential backoff (1s, 2s, 4s, 8s, 16s, max 30s)
- Session recovery: Automatic new session creation on 404 errors

### Recent Updates
- Migrated from TypeScript to Go implementation
- Switched to DuckDB for persistent session storage
- Implemented comprehensive tool system with 22 tools:
  - File operations: read, write, edit (line-based)
  - Directory operations: list, tree, mkdir, remove, move
  - Search: regex-based file content search
  - Git integration: status, diff, log, branch, add, commit, push, pull, checkout, merge
  - Web operations: search (mock), fetch with HTML-to-markdown conversion
- Added error recovery system (Phase 1 Complete):
  - Retry utility with exponential backoff and jitter
  - Error classification for intelligent retry decisions
  - Per-tool retry policies based on operation type
  - Anthropic API 529 "Overloaded" errors now retry automatically
  - Comprehensive test coverage for reliability
- Added context intelligence system (Phase 2 Complete):
  - Enhanced metadata extraction for code files:
    - Imports, exports, functions, classes for multiple languages
    - Language-specific parsing (Go, JS/TS, Python, Java, Rust)
  - NLP-based keyword extraction with:
    - Code pattern detection (camelCase, snake_case)
    - Domain-specific term expansion
    - Synonym mapping and action-object pairs
  - Smart file prioritization:
    - Metadata-aware scoring (functions, classes, exports)
    - Language and framework detection
    - Relevance-based file selection
  - Accurate token counting:
    - GPT-style tokenization approximation
    - Language-specific token ratios
    - Subword tokenization handling
  - Enhanced change tracking:
    - Detailed metadata for each change
    - Tool-specific operation tracking
    - Git operation awareness
- Implemented tool parameter validation for safety
- Added tool usage summaries in UI with metrics:
  - File operations show byte counts and line numbers
  - Directory operations show item counts
  - Git operations show change counts
- Fixed system prompt handling to maintain exact Claude Code identity
- Context information now added as initial user prompt
- Enhanced UI with tool summary display during execution
- Fixed SSE event handling for proper sessionId matching

## Tool System Details

### Available Tools
1. **read_file** - Read file contents with line numbers
2. **write_file** - Create new files with content
3. **edit_file** - Line-based editing (replace, insert_before, insert_after, delete)
4. **search** - Regex search across files with context lines
5. **list_dir** - List directory contents with filtering options
6. **tree** - Display directory tree structure
7. **make_dir** - Create directories (with parents option)
8. **remove** - Remove files/directories (with safety checks)
9. **move** - Move/rename files and directories
10. **bash** - Execute shell commands with timeout
11. **git_status** - Show git repository status
12. **git_diff** - Show git differences (staged/unstaged)
13. **git_log** - Show git commit history
14. **git_branch** - List git branches
15. **git_add** - Stage files for commit
16. **git_commit** - Create commits with messages
17. **git_push** - Push commits to remote repository
18. **git_pull** - Pull and merge changes from remote
19. **git_checkout** - Switch branches or restore files
20. **git_merge** - Merge branches with conflict handling
21. **web_search** - Search the web for information (mock implementation)
22. **web_fetch** - Fetch and convert web page content to markdown

### Web Tools Details
- **web_search**: Currently returns mock results. Ready for integration with search APIs (Google, Bing, DuckDuckGo)
- **web_fetch**: 
  - Fetches content from HTTP/HTTPS URLs
  - Converts HTML to readable markdown format
  - Pretty-prints JSON responses
  - Configurable timeout (1-120 seconds) and size limits (1KB-50MB)
  - Follows redirects with safety checks
  - Includes metadata (URL, status, content type, size)

### Tool Safety Features
- Path validation ensures operations stay within project scope
- Critical file protection (go.mod, package.json, etc.)
- Dangerous command detection in bash tool
- Parameter type validation and constraints
- Context-aware execution tracking

### Error Recovery Features
- **Retry Policies**:
  - Network tools: 5 attempts, 500ms initial delay, 30s max
  - Filesystem tools: 2 attempts, 50ms initial delay, 500ms max
  - API calls: 5 attempts, 1s initial delay, 60s max
- **Error Classification**:
  - Network errors: connection refused, timeout, DNS failures
  - Filesystem errors: EAGAIN, EBUSY, EINTR, lock issues
  - API errors: 429 (rate limit), 5xx (server errors), 529 (overloaded)
- **Automatic Recovery**: Transient failures handled transparently

## Next Steps
- Begin Phase 3: Agent Enhancement
  - Task planning system for complex operations
  - Multi-step execution with checkpoints
  - Rollback capabilities for operations
  - Learning from user feedback
  - Code generation templates
- Complete remaining Phase 2 items:
  - Persist change history in database
  - Add comprehensive test coverage
  - Performance optimization for large projects
- Integrate real search APIs for web_search tool (Google Custom Search, Bing, DuckDuckGo)
- Add remaining git operations (stash, reset, rebase, fetch, clone, remote)
- Implement code formatting tools
- Add test running capabilities
- Enhance streaming response handling
- Add provider abstraction for multiple AI models
- Implement MCP protocol support
- Add more sophisticated HTML-to-markdown conversion (tables, nested lists)