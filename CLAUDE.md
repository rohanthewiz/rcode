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
├── handlers/
│   ├── routes.go             # Route definitions
│   ├── ui.go                 # Main UI with element
│   ├── auth_callback.go      # OAuth callback UI
│   ├── session.go            # Session management with init prompt
│   ├── sse.go                # SSE implementation with reconnection
│   └── assets/
│       ├── js/
│       │   ├── ui.js         # Main UI logic with SSE handling
│       │   └── login.js      # Login flow logic
│       └── css/
│           └── ui.css        # Dark theme styles
├── providers/
│   └── anthropic.go          # Anthropic API client
├── tools/
│   ├── tool.go               # Tool interface & registry
│   ├── default.go            # Default tool implementations
│   ├── read_file.go          # File reading tool
│   ├── write_file.go         # File writing tool
│   └── bash.go               # Bash command tool
└── go.mod                    # Dependencies
```

## Core Technologies
- **Web Framework**: github.com/rohanthewiz/rweb
- **HTML Generation**: github.com/rohanthewiz/element
- **Error Handling**: github.com/rohanthewiz/serr
- **Logging**: github.com/rohanthewiz/logger
- **Server Port**: 8000

## Authentication System
- **OAuth Provider**: Anthropic (Claude.ai)
- **Client ID**: `9d1c250a-e61b-44d9-88ed-5944d1962f5e`
- **OAuth Flow**: PKCE-based with manual code entry
- **Token Storage**: `~/.local/share/rcode/auth.json`
- **Auto-refresh**: Tokens refresh automatically when expired
- **Free Usage**: OAuth tokens provide free API access for Pro/Max users

## Key Features
1. **Chat Interface**: Web-based UI with session management
2. **Tool System**: Extensible tools for file operations and bash commands
3. **Real-time Updates**: Server-sent events (SSE) with robust reconnection
4. **Dark Theme**: Modern dark-themed UI with CSS variables
5. **Session Management**: Create, list, and delete chat sessions
6. **Auto-initialization**: Sessions start with permission prompt for tools/files
7. **Connection Recovery**: Exponential backoff and manual reconnection for SSE

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
- `POST /api/session/:id/message` - Send message to session
- `GET /api/session/:id/messages` - Get session messages
- `GET /events` - SSE endpoint for real-time updates

## Development Notes

### Running the Server
```bash
go run main.go
```
Then visit http://localhost:8000

### OAuth Flow
1. User clicks login → Opens Claude.ai OAuth in new tab
2. User authorizes → Gets code from Anthropic
3. User pastes code → Server exchanges for tokens
4. Tokens stored with automatic refresh capability

### Important Implementation Details
- System prompt identifies as "Claude Code, Anthropic's official CLI"
- OAuth headers: `Authorization: Bearer {token}`, `anthropic-beta: oauth-2025-04-20`
- Messages use Anthropic's streaming API format
- Tool system supports read, write, bash operations
- Sessions currently use in-memory storage (temporary)
- Each session starts with: "Always ask before creating or writing files or using any tools"
- SSE reconnection: 5 attempts with exponential backoff (1s, 2s, 4s, 8s, 16s, max 30s)
- Session recovery: Automatic new session creation on 404 errors

### Recent Updates
- Migrated from TypeScript to Go implementation
- Fixed Monaco Editor issues by using native textarea
- Resolved API authentication with proper system prompts
- Fixed SSE streaming for real-time responses
- Implemented basic tool system architecture
- Added robust SSE reconnection with exponential backoff
- Implemented session auto-recovery for lost sessions
- Added initialization prompt to enforce permission requests
- Fixed connection status indicator visibility issues

## Next Steps
- Implement persistent session storage
- Add more tools (edit, search, etc.)
- Enhance streaming response handling
- Add provider abstraction for multiple AI models
- Implement MCP protocol support