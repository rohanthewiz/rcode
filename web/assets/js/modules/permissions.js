// permissions.js - Permission handling for tool usage
// This module manages permission requests and user consent

import { escapeHtml } from './markdown.js';

let currentPermissionRequest = null;
let permissionCallback = null;

// Handle permission request from server
export function handlePermissionRequest(data) {
  console.log('Permission request received:', data);
  currentPermissionRequest = data;
  showPermissionModal(data);
}

// Show permission modal
function showPermissionModal(data) {
  const modal = document.getElementById('permission-modal');
  if (!modal) {
    console.error('Permission modal not found');
    return;
  }
  
  // Update tool name
  const toolNameEl = document.getElementById('permission-tool-name');
  if (toolNameEl) {
    toolNameEl.textContent = data.tool_name || 'Unknown Tool';
  }
  
  // Update parameters display
  const paramsEl = document.getElementById('permission-params');
  if (paramsEl && data.params) {
    paramsEl.innerHTML = formatPermissionParams(data.params);
  }
  
  // Handle diff display if available
  const diffSection = document.getElementById('permission-diff-section');
  const diffContainer = document.getElementById('permission-diff-container');
  const diffContent = document.getElementById('permission-diff-content');
  
  if (data.diff_preview && diffSection && diffContent) {
    // Show diff section
    diffSection.style.display = 'block';
    
    // Update diff stats
    const diffStats = document.getElementById('permission-diff-stats');
    if (diffStats && data.diff_preview.stats) {
      const stats = data.diff_preview.stats;
      let statsText = '';
      if (stats.additions > 0) statsText += `+${stats.additions} `;
      if (stats.deletions > 0) statsText += `-${stats.deletions}`;
      diffStats.textContent = statsText;
    }
    
    // Render diff content
    renderPermissionDiff(diffContent, data.diff_preview);
    
    // Setup toggle button
    const toggleBtn = document.getElementById('permission-diff-toggle');
    if (toggleBtn) {
      toggleBtn.onclick = () => {
        const icon = toggleBtn.querySelector('.toggle-icon');
        if (diffContainer.style.display === 'none') {
          diffContainer.style.display = 'block';
          icon.textContent = '▼';
        } else {
          diffContainer.style.display = 'none';
          icon.textContent = '▶';
        }
      };
    }
  } else if (diffSection) {
    diffSection.style.display = 'none';
  }
  
  // Show modal
  modal.style.display = 'flex';
  
  // Setup button handlers
  setupPermissionButtons(data);
}

// Format permission parameters for display
function formatPermissionParams(params) {
  if (!params) return '<div class="param-empty">No parameters</div>';
  
  const items = [];
  for (const [key, value] of Object.entries(params)) {
    if (value !== null && value !== undefined) {
      let displayValue = value;
      let valueClass = 'param-value';
      
      // Special formatting for certain parameters
      if (key === 'file_path' || key === 'path') {
        valueClass += ' param-path';
      } else if (key === 'command') {
        valueClass += ' param-command';
        displayValue = `<code>${escapeHtml(String(value))}</code>`;
      } else if (key === 'content' && typeof value === 'string') {
        // Truncate long content
        if (value.length > 200) {
          displayValue = escapeHtml(value.substring(0, 200)) + '...';
        } else {
          displayValue = escapeHtml(value);
        }
        displayValue = `<pre class="param-content">${displayValue}</pre>`;
      } else if (typeof value === 'object') {
        displayValue = `<pre>${escapeHtml(JSON.stringify(value, null, 2))}</pre>`;
      } else {
        displayValue = escapeHtml(String(value));
      }
      
      items.push(`
        <div class="param-item">
          <span class="param-key">${escapeHtml(key)}:</span>
          <span class="${valueClass}">${displayValue}</span>
        </div>
      `);
    }
  }
  
  return items.join('') || '<div class="param-empty">No parameters</div>';
}

// Render permission diff
function renderPermissionDiff(container, diffResult) {
  if (!diffResult || !diffResult.hunks) {
    container.innerHTML = '<div class="diff-empty">No changes to preview</div>';
    return;
  }
  
  let html = '<div class="diff-content">';
  
  // Add file header if available
  if (diffResult.file_path) {
    html += `<div class="diff-file-header">${escapeHtml(diffResult.file_path)}</div>`;
  }
  
  // Render hunks
  diffResult.hunks.forEach(hunk => {
    html += '<div class="diff-hunk">';
    
    // Add hunk header
    if (hunk.header) {
      html += `<div class="diff-hunk-header">${escapeHtml(hunk.header)}</div>`;
    }
    
    // Add lines
    html += '<div class="diff-lines">';
    hunk.lines.forEach(line => {
      let lineClass = 'diff-line';
      let prefix = ' ';
      
      if (line.startsWith('+')) {
        lineClass += ' diff-add';
        prefix = '+';
      } else if (line.startsWith('-')) {
        lineClass += ' diff-delete';
        prefix = '-';
      } else if (line.startsWith('@')) {
        lineClass += ' diff-header';
        prefix = '@';
      }
      
      const content = line.substring(1); // Remove the prefix
      html += `
        <div class="${lineClass}">
          <span class="diff-prefix">${prefix}</span>
          <span class="diff-content">${escapeHtml(content)}</span>
        </div>
      `;
    });
    html += '</div>'; // diff-lines
    html += '</div>'; // diff-hunk
  });
  
  html += '</div>'; // diff-content
  container.innerHTML = html;
}

// Setup permission button handlers
function setupPermissionButtons(data) {
  const approveBtn = document.getElementById('permission-approve');
  const denyBtn = document.getElementById('permission-deny');
  const abortBtn = document.getElementById('permission-abort');
  const rememberCheck = document.getElementById('permission-remember');
  
  if (approveBtn) {
    approveBtn.onclick = async () => {
      const remember = rememberCheck?.checked || false;
      await sendPermissionResponse(data.request_id, true, remember);
      closePermissionModal();
    };
  }
  
  if (denyBtn) {
    denyBtn.onclick = async () => {
      const remember = rememberCheck?.checked || false;
      await sendPermissionResponse(data.request_id, false, remember);
      closePermissionModal();
    };
  }
  
  if (abortBtn) {
    abortBtn.onclick = async () => {
      // Send abort signal
      await sendPermissionResponse(data.request_id, false, false, true);
      closePermissionModal();
      
      // Also trigger stop operation
      if (window.stopCurrentOperation) {
        window.stopCurrentOperation();
      }
    };
  }
}

// Send permission response to server
async function sendPermissionResponse(requestId, approved, remember = false, abort = false) {
  try {
    const response = await fetch('/api/permission/respond', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        request_id: requestId,
        approved: approved,
        remember: remember,
        abort: abort
      })
    });
    
    if (!response.ok) {
      console.error('Failed to send permission response');
    }
  } catch (error) {
    console.error('Error sending permission response:', error);
  }
}

// Close permission modal
function closePermissionModal() {
  const modal = document.getElementById('permission-modal');
  if (modal) {
    modal.style.display = 'none';
  }
  
  // Reset state
  currentPermissionRequest = null;
  
  // Reset diff display
  const diffContainer = document.getElementById('permission-diff-container');
  if (diffContainer) {
    diffContainer.style.display = 'none';
  }
  
  // Reset remember checkbox
  const rememberCheck = document.getElementById('permission-remember');
  if (rememberCheck) {
    rememberCheck.checked = false;
  }
}