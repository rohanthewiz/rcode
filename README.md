# RCode Agentic Coding

A pure Go, web-based agentic coding implementation for Anthropic. This is a work in progress --Alpha status.
Note: It is way better to use Claude Code, but if you want to see how deep the rabbit hole goes, take the red pill!

## Features

- 🌐 **Web UI** - Built with element package
- 📝 **Monaco Editor** - For graphical chat input editing
- 💬 **Real-time Chat** - Server-sent events for live updates
- 🎯 **Session Management** - Multiple chat sessions support
- 🔒 **HTTPS Support** - Built-in TLS/SSL for secure connections

## Quick Start

```bash
# Run the server
go run main.go

# Visit http://localhost:8000
```

### Using HTTPS (Optional)

To enable HTTPS:

```bash
# Generate self-signed certificates (for development)
cd scripts && ./generate-certs.sh && cd ..

# Run with TLS enabled
RCODE_TLS_ENABLED=true go run main.go

# Visit https://localhost:8443
```

See [docs/TLS.md](docs/TLS.md) for detailed TLS configuration options.

### Using a Proxy (Optional)

If you need to access the Anthropic API through a proxy 
(e.g., in environments where you cannot access api.anthropic.com),
you can use the MSG_PROXY environment variable:

```bash
# Start the proxy server (on a server with access)
cd proxy
go run proxy.go

# Run rcode with proxy configuration
MSG_PROXY=http://the-server:8001 go run main.go
```

The proxy server will:
- Listen on port 8001
- Forward all requests to api.anthropic.com
- Preserve OAuth tokens and headers
- Support both regular and streaming responses

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