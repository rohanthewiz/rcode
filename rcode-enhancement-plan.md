# RCode Enhancement Plan - Stellar Agentic Coding Tool

## Session Context
- **Date**: 2025-07-10
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
   - Basic tools implemented: read_file, write_file, bash
   - Proper error handling and result formatting

5. **UI & Real-time Communication**
   - Server-sent events (SSE) for streaming responses
   - Monaco Editor integration for code input
   - Markdown/syntax highlighting support
   - Session-based chat interface

### Current Limitations

1. **Limited Tool Set**
   - Only 3 basic tools (read, write, bash)
   - No file editing capabilities (only overwrite)
   - No search/grep functionality
   - No directory operations
   - No git integration
   - No project context awareness

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
- [ ] Implement Edit tool with line-based operations
- [ ] Add Search/Grep tool with regex support
- [ ] Create Directory operations tools
- [ ] Basic Git integration (status, diff)

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

## Success Metrics
1. **Tool Coverage**: Support 80% of common coding operations
2. **Context Accuracy**: 90% relevant file selection
3. **Task Success Rate**: 85% completion without errors
4. **Response Time**: <2s for most operations
5. **User Satisfaction**: Positive feedback on efficiency

## Next Session Tasks
1. Review and refine the enhancement plan
2. Begin implementation of Edit tool
3. Design detailed schemas for new tools
4. Create project scanner prototype
5. Plan UI mockups for file explorer

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