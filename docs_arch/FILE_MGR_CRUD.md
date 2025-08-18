# File Manager UI - CRUD Operations Plan

## Overview
Add file management capabilities (cut, copy, paste, delete) to the RCode web UI with a file browser interface and backend API endpoints.

## Architecture Design

### 1. Backend Components

#### Clipboard Manager (`web/clipboard.go`)
- Session-based clipboard storage
- Support for cut/copy operations
- In-memory storage with session isolation

#### File Operations Handler (`web/file_operations.go`)
- API endpoints for file operations
- Path validation and security checks
- Error handling and response formatting

### 2. API Endpoints

```
GET    /api/files                 - List directory contents
POST   /api/files/copy            - Copy files to clipboard
POST   /api/files/cut             - Cut files to clipboard  
POST   /api/files/paste           - Paste from clipboard
DELETE /api/files                 - Delete files
GET    /api/files/clipboard       - Get clipboard status
POST   /api/files/clipboard/clear - Clear clipboard
```

### 3. UI Components

#### File Browser Panel
- Tree view or list view of project files
- Right-click context menu
- Multi-select support
- Visual indicators for cut files

#### Context Menu Options
- Cut (Ctrl+X)
- Copy (Ctrl+C)
- Paste (Ctrl+V)
- Delete (Del)
- Rename (F2)
- New File
- New Folder

### 4. Implementation Steps

#### Phase 1: Backend Infrastructure
1. Create clipboard manager with session support
2. Implement file operations handler
3. Add API routes in `web/routes.go`
4. Path validation and security

#### Phase 2: UI Implementation
1. Add file browser panel to main UI
2. Implement tree/list view component
3. Add context menu functionality
4. Keyboard shortcuts

#### Phase 3: Frontend Logic
1. File selection management
2. API integration for operations
3. Real-time updates via SSE
4. Error handling and notifications

## Technical Details

### Backend Data Structures

```go
// Clipboard manager
type Clipboard struct {
    SessionID string
    Mode      string      // "cut" or "copy"
    Files     []FileInfo
    Timestamp time.Time
}

type FileInfo struct {
    Path     string
    Name     string
    IsDir    bool
    Size     int64
    Modified time.Time
}

// API Request/Response
type FileOperation struct {
    Action string   `json:"action"` // copy, cut, paste, delete
    Paths  []string `json:"paths"`
    Target string   `json:"target,omitempty"`
}
```

### Frontend Components

```javascript
// File browser state
const fileBrowser = {
    currentPath: '/',
    selectedFiles: [],
    clipboard: {
        mode: null, // 'cut' or 'copy'
        files: []
    },
    contextMenu: {
        visible: false,
        x: 0,
        y: 0
    }
};

// File operations
async function copyFiles(paths) { /* ... */ }
async function cutFiles(paths) { /* ... */ }
async function pasteFiles(targetPath) { /* ... */ }
async function deleteFiles(paths) { /* ... */ }
```

### UI Layout

```
┌─────────────────────────────────────┐
│ RCode AI Assistant                  │
├──────────┬──────────────────────────┤
│          │                          │
│  File    │     Chat Interface      │
│  Browser │                          │
│          │                          │
│  ▼ /     │                          │
│  ├─ src  │                          │
│  │  ├─ main.go                      │
│  │  └─ ...                          │
│  └─ ...  │                          │
│          │                          │
└──────────┴──────────────────────────┘
```

## Safety Features

### Protected Files/Directories
- `.git` directory
- `go.mod`, `go.sum`
- `package.json`, `package-lock.json`
- `.env` files
- System directories

### Validation Rules
1. Operations restricted to project directory
2. No parent directory traversal
3. Confirmation for destructive operations
4. Size limits for operations

## UI/UX Considerations

### Visual Feedback
- Loading spinners during operations
- Success/error notifications
- Progress bars for large operations
- Grayed out appearance for cut files

### Keyboard Shortcuts
- Ctrl+C: Copy selected files
- Ctrl+X: Cut selected files  
- Ctrl+V: Paste files
- Delete: Delete selected files
- F2: Rename file
- Ctrl+A: Select all

### Context Menu
- Right-click on files/folders
- Disabled options when not applicable
- Submenu for advanced operations

## Success Metrics
- Intuitive file management interface
- Fast and responsive operations
- Clear error messages
- No data loss during operations
- Proper permission handling

## Files to Modify/Create

1. **Backend:**
   - `web/clipboard.go` - New clipboard manager
   - `web/file_operations.go` - New file operations handler
   - `web/routes.go` - Add new API routes

2. **Frontend:**
   - `web/ui.go` - Update UI to include file browser
   - `web/assets/js/file-browser.js` - New file browser logic
   - `web/assets/js/ui.js` - Integrate file browser
   - `web/assets/css/file-browser.css` - File browser styles
   - `web/assets/css/ui.css` - Update main styles

## Timeline
- Backend implementation: 2 hours
- Frontend UI: 2 hours
- Integration & testing: 1 hour
Total: ~5 hours