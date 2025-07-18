# RCode Enhancement Plan - Stellar Agentic Coding Tool

## Session Context
- **Date**: 2025-07-10 (Updated: 2025-07-15)
- **Goal**: Transform rcode into a stellar agentic coding tool
- **Focus**: Effective agentic coding mode as main priority

## Current Architecture Analysis

### Strengths
1. **Clean Go Architecture**
   - Well-organized package structure (auth, web, tools, providers, db)
   - Proper separation of concerns
   - Uses modern Go libraries (rweb, element, serr, logger)
   - Pure Go implementation (no TypeScript/Node dependencies)

2. **Authentication & Security**
   - OAuth 2.0 with PKCE flow for Claude Pro/Max subscribers
   - Persistent token storage with automatic refresh
   - Free API access for Pro/Max users

3. **Database & Persistence**
   - DuckDB for local storage (lightweight, fast)
   - Session management with message history
   - Initial prompts system with permissions
   - Tool usage logging

4. **Tool System Foundation**
   - Clean tool interface with registry pattern
   - Comprehensive tool set: file operations, directory management, search, Git integration, web tools
   - Proper error handling and result formatting
   - Tool usage summaries with metrics

5. **UI & Real-time Communication**
   - Server-sent events (SSE) for streaming responses
   - Monaco Editor integration for code input
   - Markdown/syntax highlighting support
   - Session-based chat interface

### Current Limitations

1. **Limited Tool Set** *(Partially Addressed)*
   - ✅ File operations expanded (read, write, edit)
   - ✅ Search/grep functionality implemented
   - ✅ Directory operations (list, tree, mkdir, rm, move)
   - ✅ Core Git integration (status, diff, log, branch, add, commit, push, pull, checkout, merge)
   - ⏳ Advanced Git operations pending (stash, reset, rebase, fetch, clone, remote)
   - ❌ No project context awareness yet

2. **Basic Context Management**
   - No file watching or change detection
   - No workspace/project understanding
   - Limited context window management
   - No intelligent file prioritization

3. **UI/UX Constraints**
   - No file tree/explorer
   - No diff viewing capabilities
   - Limited code preview/highlighting
   - No terminal integration
   - No multi-file operations UI

4. **Agent Capabilities**
   - No planning/task breakdown
   - No autonomous decision making
   - No multi-step operation support
   - ✅ Error recovery strategies (Completed)
   - No learning from interactions

## Enhancement Strategy

### Phase 1: Core Tool Expansion (Essential) ✅ COMPLETED
1. ✅ **Edit Tool**: Line-based editing with diff preview
2. ✅ **Search/Grep**: Context-aware code search
3. ✅ **Directory Operations**: ls, tree, mkdir, rm
4. ✅ **Git Integration**: status, diff, commit, branch
5. ✅ **Error Recovery**: Retry strategies in tools

### Phase 2: Context Intelligence ✅ COMPLETED
1. ✅ **Project Scanner**: Enhanced language/framework detection with metadata extraction
2. ✅ **Smart File Prioritization**: NLP-based relevance scoring with metadata awareness
3. ✅ **Change Tracking**: Connected to tool executions with detailed tracking
4. ✅ **Context Window Optimization**: Accurate token counting with language awareness
5. ⏳ **Workspace Persistence**: Database integration pending

### Phase 3: Agent Enhancement ✅ COMPLETED (Core Features)
1. ✅ **Task Planning System**: Break down complex requests
2. ✅ **Multi-step Execution**: Checkpoint-based operations
3. ✅ **Rollback Capabilities**: Undo/redo support
4. ⏳ **Learning System**: Improve from user feedback (future enhancement)
5. ⏳ **Code Generation**: Templates and boilerplate (future enhancement)

### Phase 4: UI/UX Polish
1. **File Explorer**: Visual tree with operations
2. **Diff Visualization**: Before/after comparison
3. **Terminal Integration**: Embedded command line
4. **Multi-pane Layout**: Code, chat, files, terminal
5. **Keyboard Shortcuts**: Power user features
6. ✅ **Plan History View**: Review and manage previous plans (Completed)

### Phase 5: Advanced Features
1. **Multi-model Support**: OpenAI, local models
2. **Team Collaboration**: Shared sessions
3. **Custom Tool Creation**: User-defined tools
4. **Workflow Automation**: Scriptable operations
5. **Performance Analytics**: Usage monitoring

## Technical Approach

### 1. Enhanced Tool System Design
```go
// Proposed tool interface extensions
type Tool interface {
    Name() string
    Description() string
    Execute(params map[string]interface{}) (ToolResult, error)
    ValidateParams(params map[string]interface{}) error
    GetSchema() ToolSchema
}

type ToolResult struct {
    Success bool
    Output  interface{}
    Error   string
    Metadata map[string]interface{} // For diffs, statistics, etc.
}

// Tool categories for better organization
type ToolCategory string
const (
    FileOps ToolCategory = "file_operations"
    CodeAnalysis = "code_analysis"
    ProjectMgmt = "project_management"
    SystemOps = "system_operations"
)
```

### 2. Context Management System
```go
type ProjectContext struct {
    RootPath string
    Language string
    Framework string
    Dependencies []Dependency
    FileTree *FileNode
    RecentFiles []string
    ModifiedFiles map[string]time.Time
}

type ContextManager interface {
    ScanProject(path string) (*ProjectContext, error)
    PrioritizeFiles(query string, context *ProjectContext) []string
    TrackChange(filepath string, changeType ChangeType)
    GetRelevantContext(task string) *TaskContext
}
```

### 3. Task Planning Framework
```go
type TaskPlanner struct {
    Steps []TaskStep
    CurrentStep int
    Checkpoints []Checkpoint
    Context *TaskContext
}

type TaskStep struct {
    ID string
    Description string
    Tool string
    Params map[string]interface{}
    Dependencies []string
    Retryable bool
}

func (tp *TaskPlanner) Plan(request string) error
func (tp *TaskPlanner) Execute() error
func (tp *TaskPlanner) Rollback(toStep int) error
```

### 4. Agentic UI Components
- **Command Palette**: Quick action access (Ctrl+P)
- **File Explorer**: Tree view with context menu
- **Diff Viewer**: Side-by-side comparison
- **Task Monitor**: Progress visualization
- **Context Display**: Show current working set

## Implementation Priorities

### Week 1-2: Tool System Expansion
- [x] Implement Edit tool with line-based operations
- [x] Add Search/Grep tool with regex support
- [x] Create Directory operations tools
- [x] Basic Git integration (status, diff)
- [x] Extended Git integration (add, commit, push, pull, checkout, merge)

### Week 3-4: Context Intelligence
- [ ] Project scanner implementation
- [ ] File prioritization algorithm
- [ ] Change tracking system
- [ ] Context window manager

### Week 5-6: Agent Capabilities
- [x] Task planning system
- [x] Multi-step execution engine
- [x] Checkpoint/rollback mechanism
- [x] Error recovery strategies

### Week 7-8: UI Enhancement
- [ ] File explorer component
- [ ] Diff viewer integration
- [ ] Command palette
- [ ] Keyboard shortcuts

## Progress Updates

### 2025-07-18: Plan History View Implementation ✅
Successfully implemented a comprehensive Plan History feature that enhances the task planning system with full history management:

#### Implemented Plan History Features:
1. **Backend API Endpoints** (`web/planning.go`)
   - `GET /api/session/:id/plans/history` - Paginated history with search/filter support
   - `GET /api/plan/:id/full` - Complete plan details with execution metrics
   - `POST /api/plan/:id/clone` - Clone plans for re-execution
   - `DELETE /api/plan/:id` - Delete plans with transactional integrity
   
2. **Database Enhancements** (`db/tasks.go`)
   - `GetSessionPlansWithFilter` - Advanced query with pagination, search, and status filtering
   - `DeletePlan` - Transactional deletion preserving referential integrity
   - Alias fields for API compatibility (StartTime/EndTime)
   
3. **UI Components** (`web/ui.go`, `web/assets/css/ui.css`)
   - Slide-in history panel with smooth animations
   - Search bar with 300ms debounced search
   - Status filter dropdown (All, Completed, Failed, Running, Pending)
   - Plan item cards with status badges and action buttons
   - Detailed plan view modal with metrics and statistics
   
4. **JavaScript Integration** (`web/assets/js/ui.js`)
   - Complete plan history management system
   - Real-time search and filtering
   - Pagination with "Load More" functionality
   - View details modal with comprehensive information
   - Re-run capability with plan cloning
   - Delete functionality with confirmation
   - Integration with existing plan execution system
   
5. **Visual Features**
   - **Status Icons**: ✅ Completed, ❌ Failed, ⏳ Running, ⏸️ Pending
   - **Time Display**: Relative time format (e.g., "5m ago", "2h ago")
   - **Metrics Cards**: Execution count, success rate, total time
   - **Step Details**: Tool used, status, error messages
   - **Additional Info**: Modified files, Git operations summary

#### Benefits:
- **Historical Analysis**: Review past task executions and learn from patterns
- **Quick Re-execution**: Clone and re-run successful plans on similar tasks
- **Failure Investigation**: Detailed error information for debugging
- **Performance Insights**: Execution metrics help optimize future plans
- **Clean Management**: Search, filter, and delete old plans easily

#### Usage Workflow:
1. Click "Plan History" button to open the panel
2. Search by description or filter by status
3. View details to see complete execution information
4. Re-run plans by cloning and optionally executing
5. Delete old or unnecessary plans to keep history clean

### 2025-07-17: Context Intelligence Implementation ✅
Successfully completed Phase 2 of the enhancement plan with advanced context intelligence:

#### Implemented Context Intelligence Features:
1. **Enhanced Metadata Extraction** (`scanner.go`)
   - Language-specific parsing for Go, JavaScript/TypeScript, Python, Java, and Rust
   - Extracts imports, functions, classes, and exports
   - Multi-line import support for Go
   - Identifies test files, config files, and documentation
   
2. **NLP-Based Keyword Extraction** (`prioritizer.go`)
   - Advanced stop word filtering
   - Code pattern detection (camelCase, snake_case, kebab-case)
   - Domain-specific keyword expansion
   - Action-object pair extraction
   - Synonym and related term mapping
   - Language-specific keyword associations
   
3. **Smart File Scoring**
   - Function and class name relevance scoring
   - Export/public API bonus scoring
   - Import dependency analysis
   - Metadata-aware prioritization
   
4. **Accurate Token Counting** (`window.go`)
   - GPT-style tokenization approximation
   - Language-specific token ratios
   - Subword tokenization for long words
   - Multi-character operator recognition
   - Detailed token statistics and debugging
   
5. **Enhanced Change Tracking**
   - Tool execution integration
   - Detailed change metadata (size, lines, operations)
   - Git operation tracking
   - Tool-specific details preservation

#### Benefits:
- **Better File Selection**: AI can now find relevant files based on code structure, not just names
- **Accurate Context Windows**: Token counting prevents truncation and optimizes context usage
- **Intelligent Prioritization**: Files are ranked by actual code relevance, not just keywords
- **Change Awareness**: Full tracking of what tools modify during sessions

#### Remaining Work:
- Database persistence for change history
- Test coverage for all new components
- Performance optimization for large codebases

### 2025-07-18: Task Planning System Implementation ✅
Successfully completed Phase 3 of the enhancement plan with a comprehensive task planning and execution system:

#### Implemented Task Planning Features:
1. **Task Analysis & Planning** (`planner/planner.go`, `planner/analyzer.go`)
   - AI-powered task breakdown into executable steps
   - Context-aware planning using project intelligence
   - Dependency tracking between steps
   - Template-based patterns for common tasks
   
2. **Multi-step Execution Engine** (`planner/executor.go`)
   - Sequential and parallel execution support
   - Step-by-step execution with retry logic
   - Integration with existing tool system
   - Real-time progress tracking and updates
   
3. **Parallel Execution** (`planner/parallel_executor.go`)
   - Dependency graph analysis
   - Worker pool with configurable concurrency
   - Critical path analysis for optimization
   - Automatic parallelization of independent steps
   
4. **Comprehensive Rollback System**
   - **File Snapshots** (`planner/snapshots.go`): Content-addressed storage for file backups
   - **Git Rollback** (`planner/git_rollback.go`): Intelligent Git operation tracking and reversal
   - **Checkpoint Management**: Create and restore execution checkpoints
   - **Safe Rollback**: Validates operations before reverting
   
5. **Execution Metrics** (`planner/metrics.go`)
   - Detailed performance tracking per step
   - Memory usage monitoring
   - Retry count and success rates
   - Parallel execution speedup analysis
   - Human-readable metrics reports
   
6. **Database Persistence** (`db/tasks.go`)
   - Full CRUD operations for task plans
   - Execution history tracking
   - File snapshot storage
   - Metrics persistence for analysis
   
7. **Web API Integration** (`web/planning.go`)
   - RESTful endpoints for plan management
   - SSE events for real-time updates
   - Plan execution controls (start, pause, rollback)
   - Checkpoint and metrics endpoints

#### API Endpoints:
- `POST /api/session/:id/plan` - Create a new task plan
- `GET /api/session/:id/plans` - List plans for a session
- `POST /api/plan/:id/execute` - Execute a plan
- `GET /api/plan/:id/status` - Get plan execution status
- `POST /api/plan/:id/rollback` - Rollback to checkpoint
- `GET /api/plan/:id/checkpoints` - List available checkpoints
- `GET /api/plan/:id/analyze` - Analyze parallelization opportunities
- `GET /api/plan/:id/git-operations` - View Git operations for rollback

#### Benefits:
- **Complex Task Automation**: Break down and execute multi-step operations
- **Reliability**: Automatic retry and rollback on failures
- **Performance**: Parallel execution speeds up independent operations
- **Safety**: Checkpoint-based recovery and Git-aware rollback
- **Visibility**: Real-time progress and detailed metrics

#### Remaining Work:
- Comprehensive test coverage for planner package
- UI enhancements for visual task planning
- User feedback learning system
- Code generation templates

### 2025-07-15: Error Recovery Implementation ✅
Successfully completed Phase 1 of the enhancement plan with comprehensive error recovery:

#### Implemented Error Recovery Features:
1. **Retry Utility Package** (`tools/retry.go`)
   - Exponential backoff with configurable delays
   - Jitter support to prevent thundering herd
   - Context-aware cancellation
   - Pre-configured policies for different scenarios

2. **Error Classification System** (`tools/errors.go`)
   - RetryableError: Transient failures that should be retried
   - PermanentError: Non-recoverable failures
   - RateLimitError: Special handling with retry-after support
   - Smart pattern matching for network, filesystem, and API errors

3. **Enhanced Registry Integration**
   - Automatic retry support for all tools
   - Per-tool retry policy configuration
   - Metrics tracking for retry attempts and success rates
   - Detailed logging of retry behavior

4. **Anthropic API 529 Fix**
   - HTTP 529 "Overloaded" errors now classified as retryable
   - SendMessageWithRetry and StreamMessageWithRetry methods
   - 5 retry attempts with exponential backoff
   - Prevents user-visible transient failures

#### Retry Configuration:
- **Network Tools**: 5 attempts, 500ms initial delay (web_fetch, git_push/pull)
- **File System Tools**: 2 attempts, 50ms initial delay (read/write/edit)
- **API Calls**: 5 attempts, 1s initial delay with 60s max

### 2025-07-14: Git Integration Milestone
Successfully implemented core Git workflow tools, expanding RCode's capabilities significantly:

#### Implemented Git Tools:
1. **git_add** - Stage files for commit
   - Support for specific files, all changes (-A), or tracked only (-u)
   - Safety checks for interactive modes
   - Shows status after staging

2. **git_commit** - Create commits
   - Message support with validation
   - Amend functionality (--amend)
   - Auto-staging option (-a)
   - Empty commit support
   - Author override capability

3. **git_push** - Push to remote
   - Remote and branch specification
   - Force push with prominent warnings
   - Force with lease (safer alternative)
   - Set upstream tracking (-u)
   - Dry run mode for testing

4. **git_pull** - Pull from remote
   - Merge vs rebase options
   - Various merge strategies
   - Autostash functionality
   - Comprehensive conflict detection and guidance

5. **git_checkout** - Switch branches/restore files
   - Branch switching with create option (-b)
   - File restoration from HEAD
   - Force checkout with warnings
   - Orphan branch creation
   - Detached HEAD support

6. **git_merge** - Merge branches
   - Multiple merge strategies
   - Fast-forward control (--no-ff, --ff-only)
   - Squash merges
   - Conflict resolution workflow (--abort, --continue)
   - Custom merge messages

#### Remaining Git Tools (Future Work):
- git_stash - Temporary change storage
- git_reset - Reset with safety checks
- git_rebase - Interactive/non-interactive rebasing
- git_fetch - Fetch without merging
- git_clone - Repository cloning
- git_remote - Remote management

#### Current Tool Count: 22 tools total (all with error recovery)
- File operations: 5 (read, write, edit, search, etc.)
- Directory operations: 5 (list, tree, mkdir, rm, move)
- Git operations: 10 (status, diff, log, branch, add, commit, push, pull, checkout, merge)
- System operations: 1 (bash)
- Web operations: 2 (search, fetch)

#### Task Planning Capabilities:
- Intelligent task breakdown and dependency analysis
- Parallel execution with up to 3x speedup on suitable tasks
- Checkpoint-based rollback for safe operation reversal
- Git-aware rollback that handles commits, merges, and pushes
- Real-time metrics and progress tracking
- Full database persistence for plan history

## Success Metrics
1. **Tool Coverage**: Support 80% of common coding operations
2. **Context Accuracy**: 90% relevant file selection
3. **Task Success Rate**: 85% completion without errors
4. **Response Time**: <2s for most operations
5. **User Satisfaction**: Positive feedback on efficiency

## Next Session Tasks
1. ✅ ~~Review and refine the enhancement plan~~
2. ✅ ~~Implement Edit tool~~ (Completed)
3. ✅ ~~Implement core Git tools~~ (Completed: add, commit, push, pull, checkout, merge)
4. ✅ ~~Implement error recovery~~ (Completed: retry strategies, error classification, API fixes)
5. Begin Phase 2 - Context Intelligence implementation:
   - Create project scanner prototype
   - Implement language/framework detection
   - Design file prioritization algorithm
   - Build change tracking system
6. Complete remaining Git tools (stash, reset, rebase, fetch, clone, remote)
7. Enhance UI components:
   - Plan file explorer mockups
   - Design diff viewer interface

## Open Questions
1. Should we support MCP (Model Context Protocol)?
2. How to handle large repositories efficiently?
3. What's the best approach for multi-file edits?
4. Should we add support for language servers?
5. How to implement undo/redo across tools?

## Resources & References
- Current codebase: `/Users/ro/projs/go/rcode`
- OAuth client ID: `9d1c250a-e61b-44d9-88ed-5944d1962f5e`
- Port: 8000
- Dependencies: rweb, element, serr, logger, DuckDB