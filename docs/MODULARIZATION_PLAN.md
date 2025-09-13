# UI.js Modularization Plan

## Overview
This document outlines the plan to break up the monolithic `ui.js` file (3524 lines) into smaller, focused JavaScript modules for better maintainability, testability, and code organization.

## Current State
- **File**: `/web/assets/js/ui.js`
- **Size**: 3524 lines
- **Issues**: 
  - Difficult to maintain and test
  - All functionality in a single file
  - Hard to understand code boundaries
  - Global state scattered throughout

## Target Architecture

### Module Structure
```
web/assets/js/
├── ui.js (main orchestrator - reduced)
└── modules/
    ├── clipboard.js      # Image paste/drop handling
    ├── compaction.js     # Conversation compaction
    ├── events.js         # Central event bus
    ├── fileMention.js    # File mention detection
    ├── main.js           # Core initialization
    ├── markdown.js       # Markdown configuration
    ├── messages.js       # Message handling
    ├── permissions.js    # Permission management
    ├── session.js        # Session management
    ├── sse.js           # Server-sent events
    ├── state.js         # Global state management
    ├── tools.js         # Tool execution & display
    ├── usage.js         # Usage metrics display
    └── utils.js         # Utility functions
```

## Implementation Phases

### Phase 1: Infrastructure & Proof of Concept (Clipboard Module)
**Goal**: Establish the module system and prove it works with a simple module

#### Tasks:
1. Create `/web/assets/js/modules/` directory
2. Extract clipboard functionality into `clipboard.js`:
   - `setupClipboardHandling()`
   - `processImageBlob()`
   - `handlePasteEvent()`
   - `showImagePastedNotification()`
   - `setupDragAndDrop()`
   - `handleFiles()`
   - `processImageFile()`
3. Create module exports/imports pattern
4. Write tests for clipboard module
5. Update ui.js to import and use clipboard module
6. Update Go backend to embed new module

#### Success Criteria:
- Image paste/drop functionality works as before
- Module loads correctly
- Tests pass
- No console errors

### Phase 2: Core Communication Modules
**Goal**: Extract foundational modules that other modules depend on

#### Modules to Extract:
1. **state.js**: Global state management
   - All global variables
   - State getters/setters
   - State change notifications

2. **events.js**: Event bus for inter-module communication
   - Event registration
   - Event dispatching
   - Decouples module dependencies

3. **sse.js**: Server-sent events handling
   - `connectEventSource()`
   - `reconnectSSE()`
   - `disconnectSSE()`
   - `updateConnectionStatus()`
   - `showConnectionError()`
   - `handleServerEvent()`

#### Success Criteria:
- SSE reconnection works
- State changes propagate correctly
- Event system enables module decoupling

### Phase 3: Feature Modules
**Goal**: Extract remaining functionality into focused modules

#### Modules to Extract:

1. **messages.js** (484 lines estimated):
   - Message loading and display
   - Streaming message handling
   - Thinking indicators
   - Message sending

2. **session.js** (100 lines estimated):
   - Session loading
   - Session selection
   - Session creation

3. **tools.js** (388 lines estimated):
   - Tool usage summaries
   - Tool execution tracking
   - Tool management UI

4. **permissions.js** (240 lines estimated):
   - Permission request handling
   - Permission modal UI
   - Permission diff rendering

5. **usage.js** (219 lines estimated):
   - Usage panel initialization
   - Usage data loading
   - Usage display updates
   - Cost calculations

6. **markdown.js** (20 lines estimated):
   - Marked.js configuration
   - Markdown rendering utilities

7. **utils.js** (150 lines estimated):
   - Utility functions
   - Formatting helpers
   - Common operations

8. **compaction.js** (TBD):
   - Conversation compaction logic
   - Message history management

9. **fileMention.js** (TBD):
   - File path detection
   - File mention handling

10. **main.js** (remaining code):
    - Main initialization
    - DOM setup
    - Module orchestration

## Module Communication Pattern

### Export/Import Pattern
```javascript
// Module export (clipboard.js)
export function setupClipboardHandling(editor) {
  // ... implementation
}

export function handlePasteEvent(e, editor, pastedImages) {
  // ... implementation  
}

// Main import (ui.js)
import { setupClipboardHandling } from './modules/clipboard.js';
```

### Event Bus Pattern
```javascript
// events.js
class EventBus {
  constructor() {
    this.events = {};
  }
  
  on(event, callback) {
    if (!this.events[event]) {
      this.events[event] = [];
    }
    this.events[event].push(callback);
  }
  
  emit(event, data) {
    if (this.events[event]) {
      this.events[event].forEach(callback => callback(data));
    }
  }
}

export const eventBus = new EventBus();
```

### State Management Pattern
```javascript
// state.js
class AppState {
  constructor() {
    this.currentSessionId = null;
    this.editor = null;
    // ... other state
  }
  
  setSessionId(id) {
    this.currentSessionId = id;
    eventBus.emit('sessionChanged', id);
  }
  
  getSessionId() {
    return this.currentSessionId;
  }
}

export const appState = new AppState();
```

## Testing Strategy

### Unit Tests
- Test each module's exported functions in isolation
- Mock dependencies using test doubles
- Focus on edge cases and error handling

### Integration Tests  
- Test module communication via event bus
- Test state propagation across modules
- Verify module initialization order

### E2E Tests
- Test critical user flows
- Verify no functionality regression
- Performance benchmarking

## Migration Strategy

1. **Incremental Extraction**: Extract one module at a time
2. **Backward Compatibility**: Keep ui.js functional during migration
3. **Progressive Enhancement**: Add improvements while extracting
4. **Testing at Each Step**: Ensure no regression after each extraction

## Risk Mitigation

### Risks:
1. **Breaking existing functionality**: Mitigated by comprehensive testing
2. **Performance degradation**: Mitigated by bundling and lazy loading
3. **Complex dependencies**: Mitigated by event bus pattern
4. **Browser compatibility**: Mitigated by using ES6 modules with fallbacks

## Metrics for Success

- **Code Quality**: 
  - Reduced file sizes (target: < 500 lines per module)
  - Clear module boundaries
  - Improved testability

- **Performance**:
  - No increase in load time
  - No increase in memory usage
  - Maintained responsiveness

- **Developer Experience**:
  - Easier to find and fix bugs
  - Simpler to add new features
  - Better code reusability

## Timeline Estimate

- **Phase 1**: 1-2 days (Infrastructure + Clipboard)
- **Phase 2**: 2-3 days (Core modules)
- **Phase 3**: 3-4 days (Feature modules)
- **Testing & Refinement**: 2-3 days

**Total**: ~10-12 days

## Notes

- Maintain backward compatibility throughout the process
- Document module interfaces and dependencies
- Consider using a module bundler (webpack/rollup) in the future
- Keep the module size balanced (not too small, not too large)