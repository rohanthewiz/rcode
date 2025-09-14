/**
 * Tool Use Widget Module
 * Displays real-time tool execution status in LED-style display
 */
(function() {
  // Track active tool executions
  const activeTools = new Map();
  const FADE_DELAY = 20000; // n seconds after completion before fading
  const MAX_VISIBLE_TOOLS = 5; // Maximum tools before removing old completed ones
  let widgetElement = null;
  let containerElement = null;
  
  /**
   * Initialize the tool widget
   */
  function initToolWidget() {
    // Check if already initialized
    if (widgetElement) return;

    // Get the tool (use) widget
    widgetElement = document.getElementById('tool-use-widget');

    containerElement = widgetElement.querySelector('.tool-cards-container');
    
    setupEventListeners();
    
    // console.log('Tool widget initialized');
  }
  
  /**
   * Create the widget DOM element
   */
  function createWidgetElement() {
    const widget = document.createElement('div');
    widget.id = 'tool-use-widget';
    widget.className = 'tool-use-widget hidden'; // Start hidden
    widget.innerHTML = `
      <div class="widget-label">TOOLS</div>
      <div class="tool-cards-container"></div>
    `;
    return widget;
  }
  
  /**
   * Setup event listeners for SSE events
   */
  function setupEventListeners() {
    // Listen for SSE events if available
    if (window.SSEEvents) {
      window.SSEEvents.on('tool_use_start', handleToolStart);
      window.SSEEvents.on('tool_execution_start', handleToolExecutionStart);
      window.SSEEvents.on('tool_execution_complete', handleToolComplete);
    }
    
    // Also listen for direct window events (backward compatibility)
    window.addEventListener('tool_use_start', (e) => handleToolStart(e.detail));
    window.addEventListener('tool_execution_start', (e) => handleToolExecutionStart(e.detail));
    window.addEventListener('tool_execution_complete', (e) => handleToolComplete(e.detail));
  }
  
  /**
   * Handle tool use start event (from Anthropic API)
   */
  function handleToolStart(evtData) {
    if (!evtData || !evtData.data) return;
    
    const { toolUseId, toolName } = evtData.data;
    if (!toolUseId || !toolName) return;
    
    // Create card with toolUseId (will be mapped to toolId later)
    addToolCard(toolUseId, toolName);
  }
  
  /**
   * Handle tool execution start event (from backend)
   */
  function handleToolExecutionStart(evtData) {
    if (!evtData || !evtData.data) return;
    
    const { toolId, toolName } = evtData.data;
    if (!toolId || !toolName) return;
    
    // Add or update tool card
    addToolCard(toolId, toolName);
  }
  
  /**
   * Add a new tool card to the widget
   */
  function addToolCard(toolId, toolName) {
    // Show widget if hidden
    if (widgetElement && widgetElement.classList.contains('hidden')) {
      widgetElement.classList.remove('hidden');
    }
    
    // Check if tool already exists
    if (activeTools.has(toolId)) {
      return; // Tool already being tracked
    }
    
    // Create tool card
    const card = createToolCard(toolId, toolName);
    
    // Add to container
    if (containerElement) {
      containerElement.appendChild(card);
      
      // Track execution
      activeTools.set(toolId, {
        element: card,
        name: toolName,
        startTime: Date.now(),
        fadeTimeout: null
      });
      
      // Auto-scroll to show new tool
      containerElement.scrollLeft = containerElement.scrollWidth;
      
      // Manage overflow if too many tools
      manageToolOverflow();
    }
  }
  
  /**
   * Create a tool card element
   */
  function createToolCard(toolId, toolName) {
    const card = document.createElement('div');
    card.className = 'tool-card executing';
    card.dataset.toolId = toolId;
    card.innerHTML = `
      <div class="tool-card-name">${formatToolName(toolName)}</div>
      <div class="tool-card-status">EXECUTING</div>
      <div class="tool-card-summary">...</div>
      <div class="tool-card-metrics"></div>
    `;
    
    // Click to pin/unpin
    card.addEventListener('click', () => togglePin(card));
    
    return card;
  }
  
  /**
   * Handle tool execution complete event
   */
  function handleToolComplete(evtData) {
    if (!evtData || !evtData.data) return;
    
    const { toolId, status, summary, duration, metrics } = evtData.data;
    const tool = activeTools.get(toolId);
    if (!tool) return;
    
    // Update card status
    const card = tool.element;
    card.classList.remove('executing');
    card.classList.add(status === 'success' ? 'success' : 'failed');
    
    // Update card content
    updateCardContent(card, status, summary, duration || (Date.now() - tool.startTime), metrics);
    
    // Clear any existing fade timeout
    if (tool.fadeTimeout) {
      clearTimeout(tool.fadeTimeout);
    }
    
    // Schedule removal unless pinned
    if (!card.dataset.pinned) {
      tool.fadeTimeout = setTimeout(() => {
        removeToolCard(toolId);
      }, FADE_DELAY);
    }
  }
  
  /**
   * Update card content with completion data
   */
  function updateCardContent(card, status, summary, duration, metrics) {
    const statusEl = card.querySelector('.tool-card-status');
    const summaryEl = card.querySelector('.tool-card-summary');
    const metricsEl = card.querySelector('.tool-card-metrics');
    
    if (statusEl) {
      statusEl.textContent = status === 'success' ? 'SUCCESS' : 'FAILED';
    }
    
    if (summaryEl && summary) {
      summaryEl.textContent = extractSummaryText(summary);
      summaryEl.title = summary; // Full text on hover
    }
    
    if (metricsEl) {
      metricsEl.textContent = formatMetrics(metrics, duration);
    }
  }
  
  /**
   * Remove a tool card with fade animation
   */
  function removeToolCard(toolId) {
    const tool = activeTools.get(toolId);
    if (!tool) return;
    
    const card = tool.element;
    
    // Add fading class for animation
    card.classList.add('fading');
    
    // Remove after animation
    setTimeout(() => {
      if (card.parentNode) {
        card.remove();
      }
      
      // Clear fade timeout if exists
      if (tool.fadeTimeout) {
        clearTimeout(tool.fadeTimeout);
      }
      
      activeTools.delete(toolId);
      
      // Hide widget if no more tools
      if (activeTools.size === 0 && widgetElement) {
        widgetElement.classList.add('hidden');
      }
    }, 300);
  }
  
  /**
   * Toggle pin state of a card
   */
  function togglePin(card) {
    const isPinned = card.dataset.pinned === 'true';
    card.dataset.pinned = isPinned ? '' : 'true';
    card.classList.toggle('pinned');
    
    // If unpinning a completed card, schedule removal
    if (!isPinned && !card.classList.contains('executing')) {
      const toolId = card.dataset.toolId;
      const tool = activeTools.get(toolId);
      if (tool) {
        tool.fadeTimeout = setTimeout(() => {
          removeToolCard(toolId);
        }, FADE_DELAY);
      }
    }
  }
  
  /**
   * Manage tool overflow by removing old completed tools
   */
  function manageToolOverflow() {
    if (!containerElement) return;
    
    const cards = Array.from(containerElement.querySelectorAll('.tool-card'));
    
    // Remove old completed tools if too many
    if (cards.length > MAX_VISIBLE_TOOLS) {
      const completed = cards.filter(c => 
        !c.classList.contains('executing') && 
        c.dataset.pinned !== 'true'
      );
      
      // Remove oldest completed tools
      const toRemove = cards.length - MAX_VISIBLE_TOOLS;
      completed.slice(0, toRemove).forEach(card => {
        const toolId = card.dataset.toolId;
        removeToolCard(toolId);
      });
    }
  }
  
  /**
   * Format tool name for display
   */
  function formatToolName(toolName) {
    // Convert snake_case to readable format
    return toolName
      .replace(/_/g, ' ')
      .replace(/\b\w/g, c => c.toUpperCase())
      .substring(0, 15); // Limit length
  }
  
  /**
   * Extract meaningful summary text
   */
  function extractSummaryText(summary) {
    if (!summary) return '...';
    
    // Remove status symbols and extract core message
    let cleaned = summary.replace(/^[✓✗✔✖]\s*/, '');
    
    // Remove "Success: " or "Failed: " prefix
    cleaned = cleaned.replace(/^(Success|Failed):\s*/i, '');
    
    // Extract key information (file names, counts, etc.)
    const match = cleaned.match(/([^(]+)/);
    if (match) {
      cleaned = match[1].trim();
    }
    
    // Limit length
    if (cleaned.length > 25) {
      cleaned = cleaned.substring(0, 22) + '...';
    }
    
    return cleaned;
  }
  
  /**
   * Format metrics for display
   */
  function formatMetrics(metrics, duration) {
    const parts = [];
    
    // Format duration
    if (duration) {
      if (duration < 1000) {
        parts.push(`${Math.round(duration)}ms`);
      } else {
        parts.push(`${(duration / 1000).toFixed(1)}s`);
      }
    }
    
    // Add key metrics if available
    if (metrics) {
      if (metrics.bytesProcessed) {
        parts.push(formatBytes(metrics.bytesProcessed));
      } else if (metrics.linesProcessed) {
        parts.push(`${metrics.linesProcessed}L`);
      } else if (metrics.filesProcessed) {
        parts.push(`${metrics.filesProcessed}F`);
      }
    }
    
    return parts.join(' • ');
  }
  
  /**
   * Format bytes for display
   */
  function formatBytes(bytes) {
    if (bytes < 1024) return `${bytes}B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`;
    return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
  }
  
  /**
   * Clear all active tools (for cleanup)
   */
  function clearAllTools() {
    activeTools.forEach((tool, toolId) => {
      if (tool.fadeTimeout) {
        clearTimeout(tool.fadeTimeout);
      }
      if (tool.element && tool.element.parentNode) {
        tool.element.remove();
      }
    });
    activeTools.clear();
    
    if (widgetElement) {
      widgetElement.classList.add('hidden');
    }
  }
  
  // Export to global scope
  window.ToolWidget = {
    init: initToolWidget,
    handleToolStart,
    handleToolExecutionStart,
    handleToolComplete,
    clearAllTools,
    getActiveToolCount: () => activeTools.size
  };
  
  // Auto-initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', initToolWidget);
  } else {
    // DOM already loaded, initialize with slight delay to ensure other modules are ready
    setTimeout(initToolWidget, 100);
  }
})();