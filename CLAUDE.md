# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

OpenCode is an open-source AI coding agent built for the terminal. It provides a client/server architecture with a Go-based TUI (Terminal User Interface) and a TypeScript/Bun backend. The project is provider-agnostic, supporting Anthropic, OpenAI, Google, and local models.

## Commands

### Development Setup
```bash
# Install dependencies for all packages
bun install

# Set up Git hooks
./scripts/hooks
```

### Running the Application
```bash
# Run the main opencode CLI in development mode
cd packages/opencode
bun run dev

# Or from root directory
bun run packages/opencode/src/index.ts

# Run the TUI (requires server to be running)
cd packages/tui
go run ./cmd/main.go
```

### Build and Type Checking
```bash
# Run TypeScript type checking across all packages
bun run typecheck

# Build the web documentation site
cd packages/web
npm run build
```

### Testing
```bash
# Run Go tests for TUI
go test ./packages/tui/internal/theme/

# Run TypeScript tests (using Bun's test runner)
cd packages/opencode
bun test
```

### Release Process
```bash
# Create a new release
./scripts/release

# Create a minor version bump
./scripts/release --minor
```

## Architecture

### Package Structure
- **packages/opencode**: Main TypeScript application with server, providers, and tools
- **packages/tui**: Go-based Terminal UI using Bubble Tea framework
- **packages/web**: Astro-based documentation website
- **packages/function**: Cloudflare Worker API endpoint

### Core Components

**Server (packages/opencode/src/server/)**
- HTTP server using Hono framework on port 4096
- REST API for session management, chat, and configuration
- Server-Sent Events (SSE) for real-time updates via `/event` endpoint

**Provider System (packages/opencode/src/provider/)**
- Abstracts different AI model providers
- Dynamic provider loading based on environment/config
- Handles authentication (API keys, OAuth)
- Each provider implements standard interface for chat completion

**Tool System (packages/opencode/src/tool/)**
- Extensible framework for AI assistant capabilities
- Built-in tools: bash, edit, read, write, grep, glob, todo, webfetch, etc.
- Tools have Zod schema definitions and execution logic
- Tools are mapped to providers based on compatibility

**Session Management (packages/opencode/src/session/)**
- Sessions represent AI conversations
- Persistent storage using file system
- Support for parent/child session relationships
- Shareable via URLs

**Event Bus (packages/opencode/src/bus/)**
- Pub/sub system for internal communication
- Typed events with Zod schemas
- SSE endpoint subscribes to all events for client updates

### Client/Server Communication Flow
1. TUI spawns local server instance on startup
2. User messages sent via POST to `/session/:id/message`
3. Server processes with selected provider/model
4. Tool executions happen server-side
5. Responses streamed back via SSE
6. TUI updates display in real-time

### Key Integration Points
- **MCP (Model Context Protocol)**: Supports external MCP servers for extended tools
- **Provider Authentication**: Multiple methods including API keys and OAuth flows
- **State Management**: Application state via `App.state()` pattern with persistent storage

### Anthropic Pro/Max Authentication

The app provides OAuth authentication for Claude Pro/Max subscribers, enabling free usage without API keys.

**OAuth Implementation (packages/opencode/src/auth/anthropic.ts)**
- Uses OAuth 2.0 with PKCE flow
- Client ID: `9d1c250a-e61b-44d9-88ed-5944d1962f5e`
- Authorization endpoint: `https://claude.ai/oauth/authorize`
- Token exchange: `https://console.anthropic.com/v1/oauth/token`
- Scopes: `org:create_api_key user:profile user:inference`

**Authentication Flow**
1. User selects "Claude Pro/Max" in auth command (packages/opencode/src/cli/cmd/auth.ts:33-42)
2. Browser opens to Claude.ai OAuth page
3. User authorizes and receives code
4. Code exchanged for access/refresh tokens
5. Tokens stored in `~/.local/share/opencode/auth.json` with 0600 permissions

**Pro/Max Benefits (packages/opencode/src/provider/provider.ts:107-123)**
- Cost set to 0 for all models when OAuth detected
- No API usage charges for Pro/Max subscribers
- Automatic token refresh when access token expires

**OAuth Headers (packages/opencode/src/provider/provider.ts:126-146)**
- Adds `Authorization: Bearer {access_token}` header
- Includes `anthropic-beta: oauth-2025-04-20` header
- Removes `x-api-key` header when using OAuth

**Token Management (packages/opencode/src/auth/anthropic.ts:70-104)**
- Access tokens automatically refreshed using refresh token
- Credentials stored securely with restricted file permissions
- Token expiration handled transparently

## Development Guidelines

When working on this codebase:
- The project uses Bun as the JavaScript runtime and package manager
- TypeScript code follows strict type checking - run `bun run typecheck` before committing
- Go code for TUI uses Bubble Tea patterns - maintain consistency with existing components
- Tools must have proper Zod schemas and follow the established tool interface
- Provider implementations should handle their specific authentication and API quirks
- After modifying server endpoints, the Stainless SDK may need regeneration
- SST framework handles infrastructure deployment to Cloudflare Workers