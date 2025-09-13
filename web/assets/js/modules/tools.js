/**
 * Tools Module - Tool execution display and handling
 * Manages tool execution UI, progress tracking, and real-time updates
 */

(function() {
  'use strict';

  // Module state
  const activeToolExecutions = new Map();
  let toolExecutionCounter = 0;

  /**
   * Handle tool execution start event data
   * @param {Object} evtData - Tool execution start event data
   */
  function handleToolExecutionStart(evtData) {
    const { toolId, toolName, parameters } = evtData.data;
    console.log('Tool execution started:', toolName, toolId);
    
    // Remove thinking indicator immediately when tools start
    const thinkingIndicator = document.querySelector('.message.thinking');
    if (thinkingIndicator) {
      console.log('Removing thinking indicator as tool execution started');
      thinkingIndicator.remove();
    }
    
    // Create tool execution display
    const messagesContainer = document.getElementById('messages');
    if (!messagesContainer) return;
    
    const executionDiv = document.createElement('div');
    executionDiv.className = 'tool-execution';
    executionDiv.id = `tool-execution-${toolId}`;
    executionDiv.dataset.toolName = toolName;
    
    const headerDiv = document.createElement('div');
    headerDiv.className = 'tool-header';
    
    const statusIndicator = document.createElement('span');
    statusIndicator.className = 'tool-status executing';
    statusIndicator.innerHTML = '⚡';
    
    const toolNameSpan = document.createElement('span');
    toolNameSpan.className = 'tool-name';
    toolNameSpan.textContent = toolName;
    
    const statusText = document.createElement('span');
    statusText.className = 'tool-status-text';
    statusText.textContent = ' Executing...';
    
    headerDiv.appendChild(statusIndicator);
    headerDiv.appendChild(toolNameSpan);
    headerDiv.appendChild(statusText);
    
    const paramsDiv = document.createElement('div');
    paramsDiv.className = 'tool-params';
    paramsDiv.innerHTML = formatToolParameters(toolName, parameters);
    
    const progressDiv = document.createElement('div');
    progressDiv.className = 'tool-progress';
    progressDiv.style.display = 'none';
    
    executionDiv.appendChild(headerDiv);
    executionDiv.appendChild(paramsDiv);
    executionDiv.appendChild(progressDiv);
    
    messagesContainer.appendChild(executionDiv);
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
    
    // Store execution info
    activeToolExecutions.set(toolId, {
      element: executionDiv,
      toolName: toolName,
      startTime: Date.now(),
      parameters: parameters
    });
  }

  /**
   * Handle tool execution progress event
   * @param {Object} evtData - Tool execution progress event
   */
  function handleToolExecutionProgress(evtData) {
    const { toolId, progress, message } = evtData.data;
    const execution = activeToolExecutions.get(toolId);
    if (!execution) return;
    
    const progressDiv = execution.element.querySelector('.tool-progress');
    if (progressDiv) {
      progressDiv.style.display = 'block';
      
      // Update or create progress bar
      let progressBar = progressDiv.querySelector('.progress-bar');
      if (!progressBar) {
        const progressContainer = document.createElement('div');
        progressContainer.className = 'progress-container';
        
        progressBar = document.createElement('div');
        progressBar.className = 'progress-bar';
        
        progressContainer.appendChild(progressBar);
        progressDiv.appendChild(progressContainer);
      }
      
      // Update progress
      if (progress !== undefined) {
        progressBar.style.width = `${Math.min(100, Math.max(0, progress))}%`;
      }
      
      // Add progress message
      if (message) {
        let messageDiv = progressDiv.querySelector('.progress-message');
        if (!messageDiv) {
          messageDiv = document.createElement('div');
          messageDiv.className = 'progress-message';
          progressDiv.appendChild(messageDiv);
        }
        messageDiv.textContent = message;
      }
    }
  }

  /**
   * Handle tool execution complete data
   * @param {Object} data - Tool execution complete event data
   */
  function handleToolExecutionComplete(evtData) {
    const { toolId, status, duration } = evtData.data;
    const execution = activeToolExecutions.get(toolId);
    if (!execution) return;
    
    // const duration = Date.now() - execution.startTime;
    const headerDiv = execution.element.querySelector('.tool-header');
    const statusIndicator = headerDiv.querySelector('.tool-status');
    const statusText = headerDiv.querySelector('.tool-status-text');
    
    // Update status based on success
    if (status === 'success') {
      statusIndicator.className = 'tool-status success';
      statusIndicator.innerHTML = '✓';
      statusText.textContent = `Success (${formatDuration(duration)})`;
    } else {
      statusIndicator.className = 'tool-status failed';
      statusIndicator.innerHTML = '✗';
      statusText.textContent = error ? `Failed: ${error}` : 'Failed';
    }
    
    // Hide progress
    const progressDiv = execution.element.querySelector('.tool-progress');
    if (progressDiv) {
      progressDiv.style.display = 'none';
    }
    
    // // Add metrics if available -- Maybe we will mess with this later
    // if (metrics && Object.keys(metrics).length > 0) {
    //   const metricsDiv = document.createElement('div');
    //   metricsDiv.className = 'tool-metrics';
    //   metricsDiv.innerHTML = formatToolMetrics(metrics);
    //   execution.element.appendChild(metricsDiv);
    // }

    // Remove from active executions
    activeToolExecutions.delete(toolId);
    
    // Scroll to bottom
    const messagesContainer = document.getElementById('messages');
    if (messagesContainer) {
      messagesContainer.scrollTop = messagesContainer.scrollHeight;
    }
  }


  /**
   * Format tool parameters for display
   * @param {string} toolName - Name of the tool
   * @param {Object} parameters - Tool parameters
   * @returns {string} Formatted HTML
   */
  function formatToolParameters(toolName, parameters) {
    if (!parameters || Object.keys(parameters).length === 0) {
        return `<em>No parameters for ${toolName}</em>`;
    }
    
    // Special formatting for specific tools
    switch (toolName) {
      case 'read_file':
        return `<code>${parameters.path || parameters.file_path}</code>`;
      
      case 'write_file':
        return `<code>${parameters.path || parameters.file_path}</code> (${
          parameters.content ? parameters.content.length + ' chars' : 'empty'
        })`;
      
      case 'edit_file':
        const ops = parameters.operations || parameters.edits || [];
        return `<code>${parameters.path || parameters.file_path}</code> (${ops.length} edit${ops.length !== 1 ? 's' : ''})`;
      
      case 'search':
        return `Pattern: <code>${parameters.pattern}</code> in ${parameters.path || '.'}`;
      
      case 'bash':
        const cmd = parameters.command || '';
        const truncated = cmd.length > 100 ? cmd.substring(0, 100) + '...' : cmd;
        return `<code>${truncated}</code>`;
      
      case 'git_commit':
        const msg = parameters.message || '';
        const truncatedMsg = msg.length > 80 ? msg.substring(0, 80) + '...' : msg;
        return `Message: "${truncatedMsg}"`;
      
      case 'list_dir':
      case 'tree':
        return `<code>${parameters.path || '.'}</code>`;
      
      default:
        // Generic parameter display
        const entries = Object.entries(parameters)
          .filter(([key]) => !key.startsWith('_'))
          .slice(0, 3)
          .map(([key, value]) => {
            const displayValue = typeof value === 'string' && value.length > 50 
              ? value.substring(0, 50) + '...' 
              : JSON.stringify(value);
            return `${key}: ${displayValue}`;
          });
        
        if (Object.keys(parameters).length > 3) {
          entries.push('...');
        }
        
        return entries.join(', ');
    }
  }

  /**
   * Format tool metrics for display
   * @param {Object} metrics - Tool execution metrics
   * @returns {string} Formatted HTML
   */
  function formatToolMetrics(metrics) {
    const items = [];
    
    if (metrics.filesCreated) items.push(`${metrics.filesCreated} file${metrics.filesCreated !== 1 ? 's' : ''} created`);
    if (metrics.filesModified) items.push(`${metrics.filesModified} file${metrics.filesModified !== 1 ? 's' : ''} modified`);
    if (metrics.filesDeleted) items.push(`${metrics.filesDeleted} file${metrics.filesDeleted !== 1 ? 's' : ''} deleted`);
    if (metrics.bytesRead) items.push(`${formatBytes(metrics.bytesRead)} read`);
    if (metrics.bytesWritten) items.push(`${formatBytes(metrics.bytesWritten)} written`);
    if (metrics.linesAdded) items.push(`+${metrics.linesAdded} lines`);
    if (metrics.linesDeleted) items.push(`-${metrics.linesDeleted} lines`);
    if (metrics.matchesFound) items.push(`${metrics.matchesFound} match${metrics.matchesFound !== 1 ? 'es' : ''} found`);
    
    return items.length > 0 ? items.join(' • ') : '';
  }

  /**
   * Format duration in human-readable format
   * @param {number} ms - Duration in milliseconds
   * @returns {string} Formatted duration
   */
  function formatDuration(ms) {
    if (ms < 1000) return `${ms}ms`;
    if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
    return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
  }

  /**
   * Format bytes in human-readable format
   * @param {number} bytes - Number of bytes
   * @returns {string} Formatted size
   */
  function formatBytes(bytes) {
    if (bytes < 1024) return `${bytes}B`;
    if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`;
    if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
    return `${(bytes / (1024 * 1024 * 1024)).toFixed(1)}GB`;
  }

  /**
   * Clear all active tool executions
   */
  function clearActiveExecutions() {
    activeToolExecutions.clear();
  }

  /**
   * Get active tool execution count
   */
  function getActiveExecutionCount() {
    return activeToolExecutions.size;
  }

  // Export to global scope
  window.ToolsModule = {
    handleToolExecutionStart,
    handleToolExecutionProgress,
    handleToolExecutionComplete,
    formatToolParameters,
    formatToolMetrics,
    formatDuration,
    formatBytes,
    clearActiveExecutions,
    getActiveExecutionCount
  };

  // Also expose handler functions for backward compatibility
  window.handleToolExecutionStart = handleToolExecutionStart;
  window.handleToolExecutionProgress = handleToolExecutionProgress;
  window.handleToolExecutionComplete = handleToolExecutionComplete;

})();