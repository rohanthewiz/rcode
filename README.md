# RCode Agentic Coding

A pure Go, web-based agentic coding implementation for Anthropic. This is a work in progress --Alpha status.
Note: It is way better to use Claude Code, but if you want to see how deep the rabbit hole goes, take the red pill!

## Features

- 🌐 **Web UI** - Built with element package
- 📝 **Monaco Editor** - For graphical chat input editing
- 💬 **Real-time Chat** - Server-sent events for live updates
- 🎯 **Session Management** - Multiple chat sessions support

## Quick Start

```bash
# Run the server
go run cmd/main.go

# Visit http://localhost:8000
```

## Authentication

1. Authorize on Claude.ai (opens in new tab)

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

## Environment

- Go 1.22+
- Zero frontend framework
- Zero Nodejs
- Zero TypeScript