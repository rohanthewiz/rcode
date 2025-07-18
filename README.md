# RCode Agentic Coding

A pure Go, web-based agentic coding implementation for Anthropic. This is a work in progress --Alpha status.
Note: It is way better to use Claude Code, but if you want to see how deep the rabbit hole goes, take the red pill!

## Features

- 🌐 **Web UI** - Built with element package
- 📝 **Monaco Editor** - For graphical chat input editing
- 💬 **Real-time Chat** - Server-sent events for live updates
- 🎯 **Session Management** - Multiple chat sessions support
- 🔒 **HTTPS Support** - Built-in TLS/SSL for secure connections
- 🤖 **Task Planning** - AI-powered task breakdown and execution
- ⚡ **Parallel Execution** - Smart dependency analysis and concurrent operations
- 🔄 **Rollback System** - Checkpoint-based recovery with Git awareness
- 📊 **Execution Metrics** - Performance tracking and reporting
- 🎨 **Plan Mode UI** - Visual interface for creating and executing complex tasks

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

## Task Planning System

RCode includes an advanced task planning system that can break down complex requests into executable steps:

### Creating a Plan

```bash
# API endpoint to create a plan
POST /api/session/{sessionId}/plan
{
  "description": "Refactor the authentication module to use JWT tokens",
  "auto_execute": false
}
```

### Plan Execution

Plans are executed step-by-step with:
- **Dependency Analysis**: Steps that don't depend on each other run in parallel
- **Checkpoints**: Automatic save points after each successful step
- **Retry Logic**: Transient failures are automatically retried
- **Progress Tracking**: Real-time updates via Server-Sent Events

### Rollback Capabilities

If something goes wrong, you can rollback to any checkpoint:

```bash
POST /api/plan/{planId}/rollback
{
  "checkpoint_id": "checkpoint-123"
}
```

The rollback system:
- Restores file contents to their state at the checkpoint
- Tracks and can reverse Git operations (commits, merges, etc.)
- Provides safe rollback with validation

### Execution Metrics

Get detailed metrics about plan execution:

```bash
GET /api/plan/{planId}/status
```

Returns:
- Step-by-step execution times
- Memory usage per operation
- Retry counts and success rates
- Parallel execution speedup analysis

### Example Workflow

1. **Create a complex task plan**:
   ```json
   {
     "description": "Add user authentication with login, registration, and password reset"
   }
   ```

2. **Review the generated plan** - AI breaks it down into steps like:
   - Create user model and database schema
   - Implement registration endpoint
   - Add login functionality
   - Create password reset flow
   - Write tests for each component

3. **Execute with confidence** - Each step is checkpointed and can be rolled back

4. **Monitor progress** - Real-time updates show which steps are running, completed, or failed

5. **Rollback if needed** - Restore to any checkpoint if issues arise

## Plan Mode UI

RCode includes a visual Plan Mode interface that makes it easy to create and execute complex multi-step tasks:

### Activating Plan Mode

1. **Toggle the Plan Mode switch** in the header - The UI will change to a purple theme
2. **Type your complex task** in the input area - The placeholder text guides you
3. **Click "Create Plan"** instead of "Send" - This generates a task plan

### Visual Features

#### Plan Mode Indicators
- **Purple Theme**: The entire UI adopts a purple color scheme when Plan Mode is active
- **Visual Border**: The app gets a purple border with a subtle glow effect
- **Mode Indicator**: A banner appears above the input showing "📋 Plan Mode Active"

#### Plan Execution Display
When you create a plan, a modal window appears showing:

- **Progress Bar**: Visual representation of overall completion percentage
- **Step Cards**: Each step is displayed as a card with:
  - Step number in a purple circle
  - Description of what the step will do
  - Current status (pending/running/completed/failed)
  - Tool being used
  - Collapsible output section
  - Execution metrics (duration, retries)

#### Interactive Controls
- **Execute Plan**: Start the plan execution
- **Pause**: Temporarily halt execution (coming soon)
- **Rollback**: Restore to a previous checkpoint
- **View Metrics**: See detailed performance statistics
- **Close**: Hide the execution window (plan continues in background)

### Real-time Updates

The UI updates in real-time as the plan executes:
- Steps change color when running (purple glow effect)
- Progress bar fills as steps complete
- Output appears live for each step
- Status badges update instantly
- Metrics appear when steps finish

### Using Plan Mode Effectively

1. **Complex Tasks**: Use Plan Mode for tasks with multiple steps like:
   - "Refactor the authentication module to use JWT tokens"
   - "Add a complete user management system with CRUD operations"
   - "Implement a REST API with full test coverage"

2. **Simple Tasks**: Use regular chat mode for quick questions or single operations:
   - "What does this function do?"
   - "Fix this syntax error"
   - "Add a comment to this method"

3. **Monitoring Execution**: 
   - Watch the progress in real-time
   - Check outputs for each step
   - Review metrics to understand performance
   - Use rollback if something goes wrong

### Keyboard Shortcuts (Coming Soon)
- `Ctrl/Cmd + P`: Toggle Plan Mode
- `Ctrl/Cmd + Enter`: Create Plan (in Plan Mode) or Send Message (in Chat Mode)
- `Esc`: Close plan execution window

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