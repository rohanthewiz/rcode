# RoCode Go Server

A Go-based server implementation of RoCode with web UI, replacing the original TypeScript implementation.

## Features

- 🔐 **Anthropic OAuth Authentication** - Login with Claude Pro/Max for free API usage
- 🌐 **Web UI** - Built with element package for HTML generation
- 📝 **Monaco Editor** - Rich code editing experience
- 💬 **Real-time Chat** - Server-sent events for live updates
- 🎯 **Session Management** - Multiple chat sessions support

## Quick Start

```bash
# Run the server
go run cmd/main.go

# Visit http://localhost:8000
```

## Authentication

1. Click "Login with Claude Pro/Max"
2. Authorize on Claude.ai (opens in new tab)
3. Copy the authorization code
4. Paste it in the callback page
5. You're connected!

## Technical Stack

- **Web Framework**: github.com/rohanthewiz/rweb
- **HTML Generation**: github.com/rohanthewiz/element
- **Error Handling**: github.com/rohanthewiz/serr
- **Logging**: github.com/rohanthewiz/logger

## Architecture

- OAuth 2.0 with PKCE flow
- Persistent token storage with automatic refresh
- RESTful API with SSE for real-time updates
- Clean separation of concerns with auth, handlers, and providers

## Development

See [IMPLEMENTATION_STATUS.md](IMPLEMENTATION_STATUS.md) for detailed implementation notes and next steps.

## Environment

- Go 1.22+
- No API keys required with Claude Pro/Max OAuth
- Tokens stored in `~/.local/share/rcode/auth.json`
