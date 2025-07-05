# Running the OpenCode Go Server

## Quick Start Guide

The Go server is an alternative implementation of the OpenCode server that includes tool support for Anthropic's Claude.

### Authentication Options

You have two ways to authenticate with Anthropic:

#### Option 1: Claude Pro/Max OAuth (Recommended for Pro/Max subscribers)

If you have a Claude Pro or Claude Max subscription, you can use OAuth authentication for free API usage:

```bash
# Start the server
cd packages/server-go
go run ./cmd/main.go

# The server will provide instructions to authenticate via browser
# Follow the OAuth flow to connect your Claude Pro/Max account
```

#### Option 2: API Key

```bash
export ANTHROPIC_API_KEY="your-anthropic-api-key"
```

### 2. Build and Run the Server

```bash
# From the project root
cd packages/server-go

# Run directly
go run ./cmd/main.go

# Or build first
go build -o opencode-server ./cmd/main.go
./opencode-server
```

### 3. Run the TUI with Go Server

In a separate terminal:

```bash
# The TUI will connect to the Go server on port 4096
cd packages/tui
go run ./cmd/main.go
```

## Server Options

- `-port`: Server port (default: 4096)
- `-verbose`: Enable verbose logging

Example:
```bash
go run ./cmd/main.go -port 8080 -verbose
```

## Verifying Tool Availability

Once the server is running, you can verify tools are available:

```bash
# Check server health
curl http://localhost:4096/health

# List available tools
curl http://localhost:4096/tools

# Check providers
curl http://localhost:4096/config/providers
```

## Current Tool Support

The Go server currently supports:
- **File Read Tool**: Claude can read files from your filesystem
- **More tools coming soon**: bash, write, edit, etc.

When you chat with Claude through the TUI connected to the Go server, Claude will automatically have access to these tools and can use them when needed.

## OAuth Authentication

### Starting OAuth Flow

1. Make a POST request to start authentication:
   ```bash
   curl -X POST http://localhost:4096/auth/anthropic/start
   ```

2. The server will open your browser to Claude's OAuth page
3. Log in with your Claude Pro/Max account
4. Authorize the application
5. Copy the authorization code from the callback page
6. Complete authentication with the code:
   ```bash
   curl -X POST http://localhost:4096/auth/anthropic/callback \
     -H "Content-Type: application/json" \
     -d '{"code":"your-code-here","verifier":"verifier-from-start"}'
   ```

### Benefits of OAuth Authentication

- **Free Usage**: No API charges for Claude Pro/Max subscribers
- **Automatic Token Refresh**: Tokens are refreshed automatically
- **Secure Storage**: Credentials stored securely in `~/.local/share/opencode/auth.json`

### Checking Authentication Status

```bash
curl http://localhost:4096/auth/status
```

### Logging Out

```bash
curl -X POST http://localhost:4096/auth/anthropic/logout
```

## Troubleshooting

If Claude says it can't read files:
1. Ensure the Go server is running (not the TypeScript server)
2. Check authentication:
   - For OAuth: Run `curl http://localhost:4096/auth/status`
   - For API key: Check that `ANTHROPIC_API_KEY` is set
3. Verify the TUI is connected to port 4096
4. Check server logs for any errors

## Example Usage

Once everything is running, you can ask Claude:
- "Read the contents of README.md"
- "What files are in the current directory?"
- "Show me the code in main.go"

Claude will use the file read tool to fulfill these requests.