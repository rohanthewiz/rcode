// tools.js - Tool execution and display
// This module handles tool usage display and execution tracking

import { state } from './state.js';
import { escapeHtml } from './markdown.js';
import { formatDuration } from './utils.js';

// Add tool usage summary to UI
export function addToolUsageSummaryToUI(toolData) {
  const chatContainer = document.getElementById('chat-container');
  if (!chatContainer) return;
  
  // Create a container for tool usage
  const toolContainer = document.createElement('div');
  toolContainer.className = 'tool-usage-summary';
  toolContainer.dataset.toolId = toolData.id;
  
  // Format the tool name and parameters
  let summaryText = `🛠️ ${toolData.name}`;
  
  // Add relevant parameter summaries based on tool type
  if (toolData.input) {
    const params = toolData.input;
    
    switch(toolData.name) {
      case 'read_file':
        summaryText += `: ${params.file_path || 'file'}`;
        if (params.offset || params.limit) {
          summaryText += ` (lines ${params.offset || 0}-${(params.offset || 0) + (params.limit || 'end')})`;
        }
        break;
      case 'write_file':
        summaryText += `: ${params.file_path || 'file'}`;
        if (params.content) {
          const lines = params.content.split('\n').length;
          summaryText += ` (${lines} lines)`;
        }
        break;
      case 'edit_file':
        summaryText += `: ${params.file_path || 'file'}`;
        if (params.operation) {
          summaryText += ` (${params.operation})`;
        }
        break;
      case 'search':
        summaryText += `: "${params.pattern || ''}" in ${params.path || '.'}`;
        break;
      case 'bash':
        summaryText += `: ${params.command || 'command'}`;
        break;
      case 'list_dir':
      case 'tree':
        summaryText += `: ${params.path || '.'}`;
        break;
      case 'git_commit':
        summaryText += `: "${params.message || 'commit'}"`;
        break;
      case 'git_checkout':
      case 'git_branch':
        summaryText += params.branch ? `: ${params.branch}` : '';
        break;
      default:
        // For other tools, show first meaningful parameter
        if (params.path) summaryText += `: ${params.path}`;
        else if (params.file_path) summaryText += `: ${params.file_path}`;
        else if (params.url) summaryText += `: ${params.url}`;
    }
  }
  
  toolContainer.innerHTML = `
    <div class="tool-summary-header">
      <span class="tool-summary-text">${escapeHtml(summaryText)}</span>
      <span class="tool-status" id="tool-status-${toolData.id}">pending</span>
    </div>
  `;
  
  chatContainer.appendChild(toolContainer);
  chatContainer.scrollTop = chatContainer.scrollHeight;
}

// Update working indicator based on active tools
export function updateWorkingIndicator() {
  const hasActiveTools = state.activeToolExecutions.size > 0;
  const thinkingElements = document.querySelectorAll('.message.thinking');
  
  thinkingElements.forEach(element => {
    const statusElement = element.querySelector('.thinking-status');
    if (statusElement) {
      if (hasActiveTools) {
        // Get first active tool name for display
        const firstTool = Array.from(state.activeToolExecutions.values())[0];
        if (firstTool && firstTool.name) {
          statusElement.textContent = `Running ${firstTool.name}...`;
        } else {
          statusElement.textContent = 'Working...';
        }
      } else {
        statusElement.textContent = 'Thinking...';
      }
    }
  });
}

// Handle tool execution start event
export function handleToolExecutionStart(data) {
  console.log('Tool execution started:', data);
  
  // Store in active executions
  state.activeToolExecutions.set(data.tool_id, {
    tool_id: data.tool_id,
    name: data.tool_name,
    startTime: Date.now(),
    status: 'executing'
  });
  
  // Update working indicator
  updateWorkingIndicator();
  
  // Find and update the tool container
  const container = document.querySelector(`[data-tool-id="${data.tool_id}"]`);
  if (!container) {
    console.warn('Tool container not found for:', data.tool_id);
    return;
  }
  
  // Create or update the execution display
  let executionDisplay = container.querySelector('.tool-execution-display');
  if (!executionDisplay) {
    executionDisplay = document.createElement('div');
    executionDisplay.className = 'tool-execution-display';
    container.appendChild(executionDisplay);
  }
  
  // Add executing class for animation
  container.classList.add('executing');
  
  // Create the tool item display
  const toolItem = document.createElement('div');
  toolItem.className = 'tool-item executing';
  toolItem.dataset.toolId = data.tool_id;
  toolItem.innerHTML = `
    <div class="tool-header">
      <span class="tool-status-icon">⚡</span>
      <span class="tool-name">${escapeHtml(data.tool_name)}</span>
      <span class="tool-status-text">Executing...</span>
    </div>
    <div class="tool-details" style="display: none;">
      <div class="tool-params">${formatToolParams(data.params)}</div>
      <div class="tool-metrics">Starting...</div>
    </div>
    <button class="tool-toggle" onclick="toggleToolDetails(this)">
      <span class="toggle-icon">▶</span>
    </button>
  `;
  
  executionDisplay.innerHTML = '';
  executionDisplay.appendChild(toolItem);
}

// Handle tool execution progress event
export function handleToolExecutionProgress(data) {
  const execution = state.activeToolExecutions.get(data.tool_id);
  if (execution) {
    execution.progress = data.progress;
  }
  
  // Update the tool display
  const toolItem = document.querySelector(`.tool-item[data-tool-id="${data.tool_id}"]`);
  if (toolItem) {
    const metricsEl = toolItem.querySelector('.tool-metrics');
    if (metricsEl && data.message) {
      metricsEl.textContent = data.message;
    }
  }
}

// Handle tool execution complete event
export function handleToolExecutionComplete(data) {
  console.log('Tool execution completed:', data);
  
  // Remove from active executions
  const execution = state.activeToolExecutions.get(data.tool_id);
  if (execution) {
    state.activeToolExecutions.delete(data.tool_id);
  }
  
  // Update working indicator
  updateWorkingIndicator();
  
  // Update the tool display
  const container = document.querySelector(`[data-tool-id="${data.tool_id}"]`);
  if (!container) return;
  
  // Remove executing class
  container.classList.remove('executing');
  
  const toolItem = container.querySelector('.tool-item');
  if (!toolItem) return;
  
  // Update status
  toolItem.classList.remove('executing');
  toolItem.classList.add(data.status === 'success' ? 'success' : 'failed');
  
  const statusIcon = toolItem.querySelector('.tool-status-icon');
  const statusText = toolItem.querySelector('.tool-status-text');
  
  if (statusIcon) {
    statusIcon.textContent = data.status === 'success' ? '✅' : '❌';
  }
  
  if (statusText) {
    if (data.status === 'success') {
      statusText.textContent = 'Complete';
    } else {
      statusText.textContent = data.error || 'Failed';
    }
  }
  
  // Update metrics if available
  if (data.metrics && execution) {
    const metricsEl = toolItem.querySelector('.tool-metrics');
    if (metricsEl) {
      const duration = Date.now() - execution.startTime;
      let metricsText = formatDuration(duration);
      
      // Add specific metrics based on tool type
      if (data.metrics.lines_read) {
        metricsText += ` • ${data.metrics.lines_read} lines`;
      }
      if (data.metrics.bytes_written) {
        metricsText += ` • ${formatBytes(data.metrics.bytes_written)}`;
      }
      if (data.metrics.files_found) {
        metricsText += ` • ${data.metrics.files_found} files`;
      }
      if (data.metrics.matches_found) {
        metricsText += ` • ${data.metrics.matches_found} matches`;
      }
      
      metricsEl.textContent = metricsText;
    }
  }
}

// Toggle tool details visibility
export function toggleToolDetails(button) {
  const toolItem = button.closest('.tool-item');
  const details = toolItem.querySelector('.tool-details');
  const icon = button.querySelector('.toggle-icon');
  
  if (details.style.display === 'none') {
    details.style.display = 'block';
    icon.textContent = '▼';
  } else {
    details.style.display = 'none';
    icon.textContent = '▶';
  }
}

// Format tool parameters for display
function formatToolParams(params) {
  if (!params) return '';
  
  const items = [];
  for (const [key, value] of Object.entries(params)) {
    if (value !== null && value !== undefined) {
      let displayValue = value;
      if (typeof value === 'string' && value.length > 100) {
        displayValue = value.substring(0, 100) + '...';
      } else if (typeof value === 'object') {
        displayValue = JSON.stringify(value, null, 2);
      }
      items.push(`<div class="param-item"><strong>${escapeHtml(key)}:</strong> ${escapeHtml(String(displayValue))}</div>`);
    }
  }
  
  return items.join('');
}

// Format bytes for display
function formatBytes(bytes) {
  if (bytes < 1024) return bytes + ' B';
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(2) + ' KB';
  return (bytes / (1024 * 1024)).toFixed(2) + ' MB';
}

// Export for global access
window.toggleToolDetails = toggleToolDetails;