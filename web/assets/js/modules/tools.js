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
  }

  /**
   * Handle tool execution progress event
   * @param {Object} evtData - Tool execution progress event
   */
  function handleToolExecutionProgress(evtData) {
  }

  /**
   * Handle tool execution complete data
   * @param {Object} data - Tool execution complete event data
   */
  function handleToolExecutionComplete(evtData) {
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