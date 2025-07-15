# RCode Enhancement Plan - Stellar Agentic Coding Tool

## Session Context
- **Date**: 2025-07-10 (Updated: 2025-07-14)
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
   - No error recovery strategies
   - No learning from interactions

## Enhancement Strategy

### Phase 1: Core Tool Expansion (Essential)
1. **Edit Tool**: Line-based editing with diff preview
2. **Search/Grep**: Context-aware code search
3. **Directory Operations**: ls, tree, mkdir, rm
4. **Git Integration**: status, diff, commit, branch
5. **Error Recovery**: Retry strategies in tools

### Phase 2: Context Intelligence
1. **Project Scanner**: Language/framework detection
2. **Smart File Prioritization**: Relevance-based selection
3. **Change Tracking**: Monitor modifications during session
4. **Context Window Optimization**: Intelligent truncation
5. **Workspace Persistence**: Maintain state across sessions

### Phase 3: Agent Enhancement
1. **Task Planning System**: Break down complex requests
2. **Multi-step Execution**: Checkpoint-based operations
3. **Rollback Capabilities**: Undo/redo support
4. **Learning System**: Improve from user feedback
5. **Code Generation**: Templates and boilerplate

### Phase 4: UI/UX Polish
1. **File Explorer**: Visual tree with operations
2. **Diff Visualization**: Before/after comparison
3. **Terminal Integration**: Embedded command line
4. **Multi-pane Layout**: Code, chat, files, terminal
5. **Keyboard Shortcuts**: Power user features

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
- [ ] Task planning system
- [ ] Multi-step execution engine
- [ ] Checkpoint/rollback mechanism
- [ ] Error recovery strategies

### Week 7-8: UI Enhancement
- [ ] File explorer component
- [ ] Diff viewer integration
- [ ] Command palette
- [ ] Keyboard shortcuts

## Progress Updates

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

#### Current Tool Count: 22 tools total
- File operations: 5 (read, write, edit, search, etc.)
- Directory operations: 5 (list, tree, mkdir, rm, move)
- Git operations: 10 (status, diff, log, branch, add, commit, push, pull, checkout, merge)
- System operations: 1 (bash)
- Web operations: 2 (search, fetch)

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
4. Complete remaining Git tools (stash, reset, rebase, fetch, clone, remote)
5. Begin Context Intelligence implementation:
   - Create project scanner prototype
   - Implement language/framework detection
   - Design file prioritization algorithm
6. Enhance UI components:
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