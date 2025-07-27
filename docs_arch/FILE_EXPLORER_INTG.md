# RCode File Explorer Integration Plan

## Date: 2025-07-19
## Phase 4, Step 1: File Explorer Feature

This document captures the comprehensive plan for implementing a File Explorer feature in RCode, based on analysis of the current UI architecture and best practices for integrating with the existing codebase.

## Current UI Architecture Analysis

### Technology Stack
- **Backend**: Go with `github.com/rohanthewiz/element` for HTML generation
- **Frontend**: Vanilla JavaScript with Monaco Editor
- **CSS**: Custom dark theme with CSS variables
- **Real-time**: Server-Sent Events (SSE) for live updates

### Key Components

1. **HTML Generation (web/ui.go)**
   - Uses the `element` package's builder pattern for type-safe HTML generation
   - Server-side rendering approach with dynamic content injection
   - Clean separation between authenticated and non-authenticated views

2. **Layout Structure**
   - Header: App title, plan mode toggle, auth status, and action buttons
   - Main area with two sections:
     - Left sidebar (250px): Session list
     - Right content area: Chat messages and Monaco editor input
   - Modal overlays for plan execution and history

3. **Monaco Editor Integration**
   - Loaded via CDN (v0.52.2)
   - Custom dark theme ("ro-dark")
   - Used as the main input for messages
   - Configured with markdown syntax highlighting

4. **JavaScript Architecture (web/assets/js/ui.js)**
   - Event-driven with SSE for real-time updates
   - Session management with automatic reconnection
   - Plan mode functionality for complex task execution
   - Tool usage summaries displayed inline

5. **CSS Framework (web/assets/css/ui.css)**
   - CSS variables for theming
   - Flexbox layout system
   - Dark theme with accent colors
   - Responsive design patterns

## File Explorer Design Specification

### 1. Sidebar Tab Conversion
Convert the current sidebar to a tabbed interface:
- **Tab 1**: Sessions (existing functionality)
- **Tab 2**: Files (new file explorer)

Features:
- File explorer displays project directory tree
- Expand/collapse functionality for folders
- File icons based on extension
- Search/filter capability

### 2. Backend API Endpoints

```go
// New endpoints to implement
GET /api/files/tree         // Get directory tree structure
GET /api/files/content/:path // Get file content
POST /api/files/open        // Open file in editor (track in session)
GET /api/files/recent       // Get recently accessed files
POST /api/files/search      // Search files by name or content
```

### 3. Frontend Components

Create new JavaScript module `fileExplorer.js`:
```javascript
// Core functionality
- Tree node rendering with expand/collapse
- File selection and double-click to view
- Integration with existing SSE for file change notifications
- Keyboard navigation (arrow keys)
- Context menu support (future enhancement)
```

### 4. File Viewer Integration

Add a collapsible panel above chat area:
- Use Monaco Editor in read-only mode for syntax highlighting
- Support split view: chat on right, file viewer on left
- Tab management for multiple open files
- Synchronized scrolling option
- File path breadcrumb navigation

### 5. Element Package Components

```go
// FileTreeNode component for recursive rendering
type FileTreeNode struct {
    Path     string
    Name     string
    IsDir    bool
    IsOpen   bool
    Children []FileTreeNode
    Icon     string
}

func (f FileTreeNode) Render(b *element.Builder) (x any) {
    iconClass := "file-icon"
    if f.IsDir {
        if f.IsOpen {
            iconClass = "folder-open-icon"
        } else {
            iconClass = "folder-icon"
        }
    }
    
    b.Div("data-path", f.Path, "class", "tree-node").R(
        b.Span("class", iconClass),
        b.Span("class", "node-name").T(f.Name),
        func() (x any) {
            if f.IsDir && f.IsOpen {
                b.Div("class", "tree-children").R(
                    element.ForEach(f.Children, func(child FileTreeNode) {
                        child.Render(b)
                    }),
                )
            }
            return
        }(),
    )
    return
}
```

### 6. CSS Enhancements

```css
/* File Explorer Styles */
.file-explorer {
    --indent-size: 20px;
    --line-height: 24px;
}

.tree-node {
    display: flex;
    align-items: center;
    padding: 2px 4px;
    cursor: pointer;
    user-select: none;
}

.tree-node:hover {
    background-color: var(--hover-bg);
}

.tree-node.selected {
    background-color: var(--selection-bg);
}

.file-icon::before { content: "📄"; }
.folder-icon::before { content: "📁"; }
.folder-open-icon::before { content: "📂"; }

/* Tab Navigation */
.sidebar-tabs {
    display: flex;
    border-bottom: 1px solid var(--border-color);
}

.sidebar-tab {
    flex: 1;
    padding: 8px;
    text-align: center;
    cursor: pointer;
}

.sidebar-tab.active {
    border-bottom: 2px solid var(--accent-color);
}
```

### 7. Interactive Features

- **Selection**: Click to select files/folders
- **Navigation**: Double-click to open files
- **Keyboard**: Arrow keys for navigation, Enter to open
- **Search**: Ctrl/Cmd+P for quick file search
- **Context**: Right-click for future context menu
- **Drag & Drop**: Future support for file operations

### 8. Chat Integration

- Reference open files in chat with `@filename`
- Highlight mentioned files in explorer
- Quick actions: "Edit this file", "Show in explorer"
- Track which files are being discussed in current session
- Auto-suggest relevant files based on conversation

## Implementation Roadmap

### Phase 1: Backend API (Week 1)
1. Create file system service with safety checks
   - Path validation and sanitization
   - Respect .gitignore and .rcodeIgnore
   - Implement file access permissions
2. Directory tree traversal with caching
   - Efficient recursive scanning
   - Incremental updates for large projects
   - Memory-efficient tree representation
3. File content serving with proper MIME types
4. Recent files tracking per session

### Phase 2: UI Structure (Week 1-2)
1. Modify sidebar HTML generation in `web/ui.go`
   - Add tab navigation structure
   - Create containers for both tabs
   - Maintain backward compatibility
2. Update CSS for tabbed interface
   - Smooth transitions
   - Active tab indicators
   - Responsive behavior
3. Add file viewer panel structure
   - Collapsible design
   - Resizable panels
   - Tab management

### Phase 3: File Tree Rendering (Week 2)
1. Implement FileTreeNode component
   - Recursive rendering logic
   - Lazy loading for deep structures
   - Virtual scrolling for performance
2. Add expand/collapse functionality
   - State management
   - Animation transitions
   - Keyboard shortcuts
3. File type detection and icons
   - Extension-based icons
   - Special folder detection
   - Custom icon support

### Phase 4: File Viewer (Week 3)
1. Integrate Monaco Editor for viewing
   - Read-only mode configuration
   - Syntax highlighting
   - Theme consistency
2. Tab management system
   - Multiple file support
   - Tab overflow handling
   - Close/save indicators
3. Split view implementation
   - Draggable divider
   - Synchronized scrolling option
   - Layout persistence

### Phase 5: Integration (Week 3-4)
1. Connect file selection to chat context
   - Context injection
   - File reference parsing
   - Auto-completion
2. SSE events for file changes
   - Watch for external changes
   - Update tree dynamically
   - Notify open file modifications
3. Session-based file tracking
   - Remember open files
   - Recent files per session
   - Quick access shortcuts

## Technical Considerations

### Performance Optimization
- **Lazy Loading**: Load directory contents on demand
- **Virtualization**: Render only visible tree nodes
- **Caching**: Cache directory structures with TTL
- **Debouncing**: Debounce search and filter operations
- **Web Workers**: Offload heavy operations (future)

### Security Measures
- **Path Validation**: Ensure all paths stay within project root
- **Access Control**: Respect file permissions
- **Ignore Patterns**: Honor .gitignore and custom patterns
- **Symbolic Links**: Handle safely, prevent escaping
- **File Size Limits**: Prevent loading huge files

### UX Best Practices
- **Responsive**: Collapsible sidebar on mobile
- **Accessibility**: Keyboard navigation, ARIA labels
- **Feedback**: Loading states, error messages
- **Persistence**: Remember user preferences
- **Shortcuts**: Common operations via keyboard

### State Management
- **Session State**: Track per-session file access
- **UI State**: Expanded folders, selected files
- **View State**: Split positions, tab order
- **Preference State**: User settings persistence

### Integration Points
- **Context System**: Feed file info to AI context
- **Tool System**: Quick access to file tools
- **Search System**: Integrate with existing search
- **Git Integration**: Show git status in tree

## Future Enhancements

1. **Advanced Features**
   - File creation/deletion from UI
   - Rename operations
   - Drag & drop support
   - Multi-select operations
   - File preview (images, markdown)

2. **Context Menu**
   - Common file operations
   - Git operations
   - Tool shortcuts
   - Custom actions

3. **Search Enhancements**
   - Full-text search
   - Regex support
   - Search history
   - Find & replace

4. **Visual Indicators**
   - Git status badges
   - Changed file markers
   - Error/warning indicators
   - File size display

5. **Collaboration**
   - Show which files others are viewing
   - Shared cursors
   - File locking
   - Change notifications

## Dependencies and Resources

### Required Packages
- Existing: `element`, `rweb`, `serr`, `logger`
- New: None required (using existing stack)

### References
- Current UI: `/web/ui.go`, `/web/assets/js/ui.js`
- CSS Framework: `/web/assets/css/ui.css`
- Monaco Editor: v0.52.2 via CDN
- Element Examples: See CLAUDE.md for patterns

### Testing Strategy
- Unit tests for file service
- Integration tests for API endpoints
- UI tests for tree interactions
- Performance tests for large directories
- Security tests for path validation

## Success Metrics

1. **Performance**
   - Tree renders <100ms for 1000 files
   - File content loads <50ms
   - Smooth animations at 60fps

2. **Usability**
   - Intuitive navigation
   - Fast file access
   - Minimal clicks to common tasks

3. **Integration**
   - Seamless chat integration
   - Context awareness
   - Tool accessibility

4. **Reliability**
   - Handle large projects
   - Graceful error handling
   - Consistent state management

## Conclusion

This File Explorer implementation will significantly enhance RCode's capabilities as an AI coding assistant by providing direct visual access to project files. The design integrates seamlessly with the existing architecture while adding powerful new functionality that complements the chat-based interface.

The phased approach ensures we can deliver value incrementally while maintaining system stability and performance. Each phase builds upon the previous one, allowing for testing and refinement along the way.