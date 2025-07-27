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
- 📜 **Plan History** - Review, re-run, and manage previous task plans
- 📁 **File Explorer** - Visual file browser with tabbed interface and real-time updates
- 🔍 **Diff Visualization** - Real-time file change tracking with Monaco-powered diff viewer
- ⚡ **Real-time Tool Execution** - Live visualization of tool operations as they happen

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

## Plan History

RCode includes a comprehensive Plan History feature that allows you to review, re-run, and manage all your previously created task plans:

### Accessing Plan History

Click the **"Plan History"** button in the header to open the history panel. The panel slides in from the right side of the screen.

### Features

#### Search and Filter
- **Search Bar**: Find plans by description using the search input
- **Status Filter**: Filter plans by status:
  - All Status (default)
  - Completed
  - Failed  
  - Running
  - Pending

#### Plan List View
Each plan in the history shows:
- **Status Icon**: Visual indicator of the plan's current state
- **Description**: The original task description
- **Status Badge**: Colored badge showing the execution status
- **Step Count**: Number of steps in the plan
- **Created Time**: When the plan was created (shown as relative time)
- **Duration**: Total execution time (if completed)

#### Actions
For each plan, you can:
- **View Details**: Opens a modal with comprehensive plan information
- **Re-run**: Clone and optionally execute the plan again
- **Delete**: Permanently remove the plan from history

### Plan Details Modal

The details view provides:

#### Overview Section
- Complete task description
- Current status with visual badge
- Creation and completion timestamps

#### Execution Statistics
- **Executions**: Total number of times the plan was run
- **Success Rate**: Percentage of successful executions
- **Total Time**: Cumulative execution duration
- **Total Steps**: Number of steps in the plan

#### Steps Breakdown
- Detailed view of each step with:
  - Step description
  - Tool used
  - Execution status
  - Error messages (if any)

#### Additional Information
- **Modified Files**: List of files changed during execution
- **Git Operations**: Summary of git commands executed

### Re-running Plans

When you re-run a plan:
1. The original plan is cloned with all steps reset to "pending"
2. The plan history panel closes automatically
3. The plan execution area opens with the cloned plan loaded
4. You're prompted whether to execute immediately or review first

### Pagination

Plan history loads 20 items at a time. Use the **"Load More"** button at the bottom to fetch additional plans.

### Real-time Updates

The plan history updates automatically when:
- New plans are created
- Existing plans change status
- Plans are deleted

## File Explorer

RCode includes a comprehensive file explorer that provides visual access to your project files with real-time synchronization:

### Accessing the File Explorer

The file explorer is integrated into the sidebar with a tabbed interface:
- **Sessions Tab**: Shows your chat sessions (default view)
- **Files Tab**: Shows the project file tree

Click the "Files" tab to switch to the file explorer view.

### Features

#### File Tree Navigation
- **Expand/Collapse**: Click folder icons to explore the directory structure
- **File Icons**: Visual indicators for different file types (Go, JavaScript, Python, etc.)
- **Ignore Patterns**: Respects `.gitignore` and `.rcodeIgnore` files
- **Smart Sorting**: Directories first, then files alphabetically

#### File Operations
- **Double-click to Open**: Opens files in a read-only Monaco editor
- **Syntax Highlighting**: Automatic language detection
- **Multiple Files**: Open multiple files in tabs
- **File Search**: Quick search by filename

#### Real-time Updates
The file explorer automatically refreshes when:
- Files are created, modified, or deleted by AI tools
- Directories are created or removed
- Git operations affect the file structure

#### Session Integration
- **File Tracking**: All opened files are tracked per session
- **Recent Files**: Access recently viewed files quickly
- **Persistent State**: File access history is stored in the database

### File Viewer

When you open a file:
- **Monaco Editor**: Full syntax highlighting with theme support
- **Read-only Mode**: Files are displayed for viewing (editing through AI chat)
- **Tab Management**: Switch between multiple open files
- **Auto-close**: Close files individually or all at once

### Keyboard Navigation (Coming Soon)
- Arrow keys for tree navigation
- Enter to open files
- Ctrl/Cmd+P for quick file search

### API Endpoints

The file explorer provides these endpoints:
- `GET /api/files/tree` - Get directory tree structure
- `GET /api/files/content/:path` - Get file content
- `POST /api/files/search` - Search for files
- `POST /api/session/:id/files/open` - Track file opening
- `GET /api/session/:id/files/recent` - Get recent files

## Diff Visualization

RCode includes a powerful diff visualization system that tracks all file changes during your coding sessions and displays them with Monaco Editor's professional diff viewer:

### How It Works

Whenever you modify a file using AI tools, RCode automatically:
1. **Creates a snapshot** of the file before changes
2. **Generates a diff** after modifications
3. **Broadcasts real-time notifications** via Server-Sent Events
4. **Updates the UI** with visual indicators

### Visual Indicators

#### File Explorer Integration
- **Orange Dot (●)**: Appears next to modified files in the file tree
- **System Messages**: Notifications appear when files are changed: "📝 Changes detected in filename"
- **Context Menu**: Right-click modified files to see "View Changes" option

### Diff Viewer Features

The diff viewer provides multiple ways to visualize changes:

#### View Modes
1. **Monaco (Default)**: Professional side-by-side diff editor from VS Code
   - Syntax highlighting for all supported languages
   - Synchronized scrolling between panes
   - Inline change indicators
   - Minimap navigation

2. **Side-by-Side**: Custom implementation showing before/after
   - Line numbers for easy reference
   - Color-coded additions and deletions
   - Synchronized scrolling

3. **Inline**: Sequential view of changes
   - Deleted lines in red
   - Added lines in green
   - Ideal for reviewing small changes

4. **Unified**: Traditional unified diff format
   - Compact representation
   - Standard diff notation (+/-)

#### Interactive Controls
- **Theme Toggle**: Switch between dark and light themes
- **Word Wrap**: Enable/disable text wrapping
- **Statistics**: View addition/deletion counts
- **Actions**:
  - **Apply Changes**: Save the modifications permanently
  - **Revert**: Restore the original file content
  - **Copy Diff**: Export diff to clipboard

### Usage

1. **Automatic Tracking**: All file modifications are tracked automatically
2. **View Changes**: 
   - Look for orange indicators in File Explorer
   - Right-click and select "View Changes"
   - Or click when you see the system notification
3. **Review**: Use different view modes to review changes
4. **Decide**: Apply changes to keep them or revert to original

### Keyboard Shortcuts
- `Esc`: Close diff viewer
- `Tab`: Switch between view modes (when focused)

### Technical Details

The diff system uses:
- **LCS Algorithm**: Longest Common Subsequence for accurate line-based diffs
- **Snapshot Management**: Efficient storage of file versions
- **DuckDB Storage**: Persistent diff history tied to sessions
- **SSE Broadcasting**: Real-time updates across all connected clients

### API Endpoints

Diff visualization endpoints:
- `GET /api/session/:id/diff/:diffId` - Get diff details
- `POST /api/session/:id/diff/:diffId/apply` - Apply changes
- `POST /api/session/:id/diff/:diffId/revert` - Revert changes
- `GET /api/session/:id/diffs` - List all diffs for a session

## Real-time Tool Execution Display

RCode provides immediate visual feedback when AI tools are executing, replacing the generic "Thinking..." indicator with detailed real-time status updates:

### How It Works

When the AI uses tools to complete your request:
1. **"Thinking..." disappears** as soon as the first tool starts
2. **Tool execution container appears** showing active operations
3. **Real-time status updates** for each tool (executing → success/failed)
4. **Progress tracking** for long-running operations
5. **Persistent history** - summaries remain in chat after completion

### Visual Features

#### Tool Execution Container
- **Header**: "🛠️ Executing tools..." with collapsible toggle
- **Tool List**: Each active tool shows:
  - Status icon (⏳ executing, ✓ success, ❌ failed)
  - Tool name and operation details
  - Progress bar for operations that support it
  - Execution metrics (duration, bytes processed, etc.)

#### Status Indicators
- **⏳ Executing**: Animated spinner showing tool is running
- **✓ Success**: Green checkmark with summary (e.g., "✓ Wrote main.go (523 bytes)")
- **❌ Failed**: Red X with error message
- **🔄 Retrying**: For transient failures being retried

#### Animations
- Smooth fade-in when tools start
- Pulsing effect on executing tools
- Color transitions on status changes
- Progress bar animations for long operations

### Tool-Specific Features

Different tools provide specialized feedback:
- **File Operations**: Show bytes read/written and line counts
- **Directory Operations**: Display item counts
- **Search Operations**: Report number of matches found
- **Git Operations**: Show commit counts, changed files
- **Web Operations**: Display download progress

### Benefits

1. **Transparency**: See exactly what operations are being performed
2. **Progress Tracking**: No more wondering if the AI is stuck
3. **Debugging**: Clear visibility into tool failures
4. **Professional UX**: Matches modern CI/CD pipeline interfaces

### Technical Implementation

The feature uses:
- **Server-Sent Events**: Real-time updates from backend to frontend
- **Event Types**:
  - `tool_execution_start`: Tool begins execution
  - `tool_execution_progress`: Progress updates for long operations
  - `tool_execution_complete`: Tool finishes with status and metrics
- **Frontend State Management**: Tracks active executions in real-time
- **CSS Animations**: Smooth transitions and visual feedback

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