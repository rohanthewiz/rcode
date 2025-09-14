# Tool Use Widget - Design Plan

## Overview
Create a real-time tool execution monitor widget that displays active tools in a retro green LED display style, positioned to the right of the model selector.

## Design Requirements

### Visual Design
- **Style**: Retro green LED display aesthetic (#00ff00 on dark background)
- **Location**: Header area, to the right of the model selector
- **Layout**: Horizontal scrolling container with individual tool cards
- **Animations**: Smooth pulse/glow effects for active tools, fade transitions

### Functionality
1. **Real-time Updates**
   - Listen for `tool_use_start` events to add tools to the widget
   - Listen for `tool_execution_complete` events to update status
   - Show active/completed/failed status with different visual states

2. **Tool Display Format**
   - Show tool name in LED-style font
   - Display execution status (executing → success/failed)
   - Show summary text from `tool_execution_complete` event
   - Include execution metrics (duration, bytes, etc.)

3. **Interaction**
   - Horizontal scroll for multiple concurrent tools
   - Auto-scroll to newest tool when added
   - Tools fade out after completion (configurable delay)
   - Click to pin/unpin tools to prevent auto-removal

## Technical Implementation

### HTML Structure
```html
<div id="tool-use-widget" class="tool-use-widget">
  <div class="widget-label">TOOLS</div>
  <div class="tool-cards-container">
    <div class="tool-card executing" data-tool-id="tool_123">
      <div class="tool-card-name">read_file</div>
      <div class="tool-card-status">EXECUTING</div>
      <div class="tool-card-summary">Reading main.go...</div>
      <div class="tool-card-metrics">2.3s</div>
    </div>
  </div>
</div>
```

### CSS Styling
```css
.tool-use-widget {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: #0a0a0a;
  border: 1px solid #00ff00;
  border-radius: 4px;
  padding: 0.25rem 0.5rem;
  font-family: 'Courier New', monospace;
  max-width: 600px;
  margin-left: 1rem;
}

.widget-label {
  color: #00ff00;
  font-size: 0.75rem;
  font-weight: bold;
  letter-spacing: 1px;
  opacity: 0.8;
}

.tool-cards-container {
  display: flex;
  gap: 0.5rem;
  overflow-x: auto;
  flex: 1;
  scrollbar-width: thin;
  scrollbar-color: #00ff00 #0a0a0a;
}

.tool-card {
  min-width: 120px;
  padding: 0.25rem 0.5rem;
  background: #001100;
  border: 1px solid #00ff00;
  border-radius: 3px;
  position: relative;
  transition: all 0.3s ease;
}

.tool-card.executing {
  animation: ledPulse 1s ease-in-out infinite;
  box-shadow: 0 0 10px rgba(0, 255, 0, 0.5);
}

.tool-card.success {
  border-color: #00ff00;
  background: #002200;
}

.tool-card.failed {
  border-color: #ff3300;
  background: #220000;
}

@keyframes ledPulse {
  0%, 100% { opacity: 0.8; }
  50% { opacity: 1; box-shadow: 0 0 15px rgba(0, 255, 0, 0.8); }
}

.tool-card-name {
  color: #00ff00;
  font-size: 0.7rem;
  font-weight: bold;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.tool-card-status {
  color: #00cc00;
  font-size: 0.6rem;
  opacity: 0.9;
  margin-top: 2px;
}

.tool-card-summary {
  color: #00aa00;
  font-size: 0.65rem;
  margin-top: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 150px;
}

.tool-card-metrics {
  color: #008800;
  font-size: 0.6rem;
  margin-top: 2px;
  opacity: 0.8;
}
```

### JavaScript Implementation

#### Module Structure
Create new module: `/web/assets/js/modules/tool-widget.js`

```javascript
(function() {
  // Track active tool executions
  const activeTools = new Map();
  const FADE_DELAY = 3000; // 3 seconds after completion
  const MAX_VISIBLE_TOOLS = 5;
  
  // Initialize widget
  function initToolWidget() {
    const container = document.querySelector('.model-selector-container');
    if (!container) return;
    
    const widget = createWidgetElement();
    container.appendChild(widget);
    
    // Listen for SSE events
    if (window.SSEEvents) {
      window.SSEEvents.on('tool_use_start', handleToolStart);
      window.SSEEvents.on('tool_execution_complete', handleToolComplete);
    }
  }
  
  function createWidgetElement() {
    const widget = document.createElement('div');
    widget.id = 'tool-use-widget';
    widget.className = 'tool-use-widget';
    widget.style.display = 'none'; // Hidden initially
    widget.innerHTML = `
      <div class="widget-label">TOOLS</div>
      <div class="tool-cards-container"></div>
    `;
    return widget;
  }
  
  function handleToolStart(evtData) {
    const { toolId, toolName, parameters } = evtData.data;
    
    // Show widget
    const widget = document.getElementById('tool-use-widget');
    if (widget) widget.style.display = 'flex';
    
    // Create tool card
    const card = createToolCard(toolId, toolName);
    const container = document.querySelector('.tool-cards-container');
    container.appendChild(card);
    
    // Track execution
    activeTools.set(toolId, {
      element: card,
      name: toolName,
      startTime: Date.now()
    });
    
    // Auto-scroll to show new tool
    container.scrollLeft = container.scrollWidth;
    
    // Manage overflow
    manageToolOverflow();
  }
  
  function handleToolComplete(evtData) {
    const { toolId, status, summary, metrics } = evtData.data;
    const tool = activeTools.get(toolId);
    if (!tool) return;
    
    // Update card status
    const card = tool.element;
    card.classList.remove('executing');
    card.classList.add(status === 'success' ? 'success' : 'failed');
    
    // Update card content
    const statusEl = card.querySelector('.tool-card-status');
    const summaryEl = card.querySelector('.tool-card-summary');
    const metricsEl = card.querySelector('.tool-card-metrics');
    
    statusEl.textContent = status.toUpperCase();
    summaryEl.textContent = extractSummaryText(summary);
    metricsEl.textContent = formatMetrics(metrics, Date.now() - tool.startTime);
    
    // Schedule removal unless pinned
    if (!card.dataset.pinned) {
      setTimeout(() => {
        removeToolCard(toolId);
      }, FADE_DELAY);
    }
  }
  
  function createToolCard(toolId, toolName) {
    const card = document.createElement('div');
    card.className = 'tool-card executing';
    card.dataset.toolId = toolId;
    card.innerHTML = `
      <div class="tool-card-name">${toolName}</div>
      <div class="tool-card-status">EXECUTING</div>
      <div class="tool-card-summary">...</div>
      <div class="tool-card-metrics"></div>
    `;
    
    // Click to pin/unpin
    card.addEventListener('click', () => togglePin(card));
    
    return card;
  }
  
  function removeToolCard(toolId) {
    const tool = activeTools.get(toolId);
    if (!tool) return;
    
    tool.element.style.opacity = '0';
    setTimeout(() => {
      tool.element.remove();
      activeTools.delete(toolId);
      
      // Hide widget if no tools
      if (activeTools.size === 0) {
        const widget = document.getElementById('tool-use-widget');
        if (widget) widget.style.display = 'none';
      }
    }, 300);
  }
  
  function extractSummaryText(summary) {
    // Extract meaningful part from summary
    // e.g., "✓ Read file main.go (1234 bytes)" → "Read main.go"
    const cleaned = summary.replace(/^[✓✗]\s*/, '');
    const match = cleaned.match(/^(\w+)\s+(.+?)(?:\s*\(|$)/);
    return match ? `${match[1]} ${match[2]}` : cleaned;
  }
  
  function formatMetrics(metrics, duration) {
    // Format duration and key metrics
    const time = duration < 1000 ? `${duration}ms` : `${(duration/1000).toFixed(1)}s`;
    
    if (metrics) {
      if (metrics.bytesProcessed) {
        return `${time} • ${formatBytes(metrics.bytesProcessed)}`;
      }
      if (metrics.linesProcessed) {
        return `${time} • ${metrics.linesProcessed}L`;
      }
    }
    
    return time;
  }
  
  function manageToolOverflow() {
    const container = document.querySelector('.tool-cards-container');
    const cards = Array.from(container.querySelectorAll('.tool-card'));
    
    // Remove old completed tools if too many
    if (cards.length > MAX_VISIBLE_TOOLS) {
      const completed = cards.filter(c => 
        !c.classList.contains('executing') && !c.dataset.pinned
      );
      
      // Remove oldest completed tools
      completed.slice(0, cards.length - MAX_VISIBLE_TOOLS).forEach(card => {
        const toolId = card.dataset.toolId;
        removeToolCard(toolId);
      });
    }
  }
  
  function togglePin(card) {
    card.dataset.pinned = card.dataset.pinned ? '' : 'true';
    card.classList.toggle('pinned');
  }
  
  // Export to global
  window.ToolWidget = {
    init: initToolWidget,
    handleToolStart,
    handleToolComplete
  };
  
  // Auto-initialize on DOM ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initToolWidget);
  } else {
    initToolWidget();
  }
})();
```

## Integration Steps

1. **Add CSS to `/web/assets/css/tools.css`**
   - Add the LED-style widget CSS
   - Ensure proper dark theme integration

2. **Create JavaScript module**
   - Create `/web/assets/js/modules/tool-widget.js`
   - Implement tool tracking and display logic

3. **Update SSE event handling**
   - Ensure `tool_use_start` events are properly broadcast
   - Map `tool_execution_complete` events to widget updates

4. **Update HTML loading**
   - Include tool-widget.js in the main UI
   - Ensure it loads after SSE module

5. **Test scenarios**
   - Single tool execution
   - Multiple concurrent tools
   - Tool completion states (success/failure)
   - Horizontal scrolling with many tools
   - Pin/unpin functionality

## Event Flow

1. **Tool Start**
   ```
   tool_use_start → Create card → Add to widget → Show widget → Track in Map
   ```

2. **Tool Complete**
   ```
   tool_execution_complete → Find card → Update status/summary → Schedule removal
   ```

3. **User Interaction**
   ```
   Click card → Toggle pin → Prevent auto-removal if pinned
   ```

## Future Enhancements

1. **Advanced Features**
   - Tool execution history dropdown
   - Filter by tool type
   - Execution time statistics
   - Error details on hover

2. **Visual Improvements**
   - Matrix-style falling characters background
   - Scanline effects
   - Custom LED font
   - Sound effects (optional)

3. **Performance**
   - Virtual scrolling for many tools
   - WebGL-based rendering for effects
   - Efficient DOM updates

## Testing Checklist

- [ ] Widget appears on tool execution
- [ ] Multiple tools display correctly
- [ ] Horizontal scrolling works
- [ ] Tool status updates properly
- [ ] Summary text is extracted correctly
- [ ] Metrics display accurately
- [ ] Auto-removal after completion
- [ ] Pin/unpin prevents removal
- [ ] Widget hides when no tools
- [ ] LED styling is readable
- [ ] Responsive on different screen sizes