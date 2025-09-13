/**
 * Permissions Module - Tool permission request handling
 * Manages permission dialogs, approvals, and diff previews
 */

(function() {
  'use strict';

  // Module state
  const activePermissionRequests = new Map();

  /**
   * Handle permission request from server
   * @param {Object} evtData - Permission request data
   */
  function handlePermissionRequest(evtData) {
    console.warn('PERMISSION REQUEST:', evtData);
    
    // Extract the actual data from the event
    const llmData = evtData.data;
    
    // Store the request with the correct requestId
    activePermissionRequests.set(llmData.requestId, llmData);
    
    // Get current session ID
    const currentSessionId = window.AppState ? 
      window.AppState.getState('currentSessionId') : window.currentSessionId;
    
    // Show permission modal with the actual data
    showPermissionModal(llmData);
    
    // Emit permission request event
    if (window.AppEvents) {
      window.AppEvents.emit(window.StandardEvents?.PERMISSION_REQUEST || 'permission_request', evtData);
    }
  }

  /**
   * Show permission modal dialog
   * @param {Object} llmData - Permission request data
   */
  function showPermissionModal(llmData) {
    const modal = document.getElementById('permission-modal');
    const toolNameElement = document.getElementById('permission-tool-name');
    const paramsElement = document.getElementById('permission-params');
    const rememberCheckbox = document.getElementById('permission-remember');
    
    if (!modal || !toolNameElement || !paramsElement) {
      console.error('Permission modal elements not found');
      return;
    }
    
    // Set tool name
    toolNameElement.textContent = llmData.toolName;
    
    // Display parameters
    displayPermissionParameters(paramsElement, llmData);
    
    // Handle diff preview if available
    handleDiffPreview(llmData);
    
    // Reset checkbox
    if (rememberCheckbox) {
      rememberCheckbox.checked = false;
    }
    
    // Set up button handlers
    setupPermissionButtons(llmData.requestId);
    
    // Show modal
    modal.style.display = 'block';
  }

  /**
   * Display permission parameters
   * @param {HTMLElement} container - Container element
   * @param {Object} data - Permission request data
   */
  function displayPermissionParameters(container, data) {
    container.innerHTML = '';
    
    if (data.parameterDisplay) {
      const paramDiv = document.createElement('div');
      paramDiv.className = 'param-display';
      paramDiv.textContent = data.parameterDisplay;
      container.appendChild(paramDiv);
    } else {
      // Fallback to showing raw parameters
      const paramList = document.createElement('ul');
      const params = data.parameters || {};
      const paramKeys = Object.keys(params).filter(k => !k.startsWith('_'));
      
      if (paramKeys.length === 0) {
        const li = document.createElement('li');
        li.innerHTML = '<em>No parameters provided (this might be an error)</em>';
        li.style.color = 'var(--warning)';
        paramList.appendChild(li);
      } else {
        for (const key of paramKeys) {
          const li = document.createElement('li');
          const value = params[key];
          // Truncate long values
          let displayValue = JSON.stringify(value);
          if (displayValue.length > 100) {
            displayValue = displayValue.substring(0, 100) + '...';
          }
          li.innerHTML = `<strong>${key}:</strong> ${displayValue}`;
          paramList.appendChild(li);
        }
      }
      container.appendChild(paramList);
    }
  }

  /**
   * Handle diff preview for file operations
   * @param {Object} data - Permission request data
   */
  function handleDiffPreview(data) {
    const diffSection = document.getElementById('permission-diff-section');
    const diffToggle = document.getElementById('permission-diff-toggle');
    const diffContainer = document.getElementById('permission-diff-container');
    const diffContent = document.getElementById('permission-diff-content');
    const diffStats = document.getElementById('permission-diff-stats');
    
    if (!diffSection || !diffToggle || !diffContainer || !diffContent) {
      return;
    }
    
    const fileTools = ['write_file', 'edit_file', 'smart_edit'];
    if (data.diffPreview && fileTools.includes(data.toolName)) {
      // Show diff section
      diffSection.style.display = 'block';
      
      // Set diff stats
      const stats = data.diffPreview.stats;
      if (stats && diffStats) {
        diffStats.textContent = `(+${stats.added || 0}, -${stats.deleted || 0} lines)`;
      }
      
      // Render diff content
      renderPermissionDiff(diffContent, data.diffPreview);
      
      // Set up toggle handler
      const toggleIcon = diffToggle.querySelector('.toggle-icon');
      if (toggleIcon) {
        diffToggle.onclick = function() {
          if (diffContainer.style.display === 'none') {
            diffContainer.style.display = 'block';
            diffToggle.classList.add('expanded');
            toggleIcon.textContent = '▼';
          } else {
            diffContainer.style.display = 'none';
            diffToggle.classList.remove('expanded');
            toggleIcon.textContent = '▶';
          }
        };
      }
      
      // Automatically expand diff
      diffContainer.style.display = 'block';
      diffToggle.classList.add('expanded');
      if (toggleIcon) {
        toggleIcon.textContent = '▼';
      }
    } else {
      // Hide diff section
      diffSection.style.display = 'none';
    }
  }

  /**
   * Render diff content in permission modal
   * @param {HTMLElement} container - Container element
   * @param {Object} diffResult - Diff result object
   */
  function renderPermissionDiff(container, diffResult) {
    container.innerHTML = '';
    
    if (!diffResult.hunks || diffResult.hunks.length === 0) {
      container.innerHTML = '<div class="diff-line context">No changes detected</div>';
      return;
    }
    
    // Render unified diff format
    diffResult.hunks.forEach(hunk => {
      // Add hunk header
      const header = document.createElement('div');
      header.className = 'diff-header';
      header.textContent = `@@ -${hunk.oldStart},${hunk.oldLines} +${hunk.newStart},${hunk.newLines} @@`;
      container.appendChild(header);
      
      // Add diff lines
      hunk.lines.forEach(line => {
        const lineDiv = document.createElement('div');
        lineDiv.className = `diff-line ${line.type}`;
        
        // Add prefix based on type
        let prefix = ' ';
        if (line.type === 'add') prefix = '+';
        else if (line.type === 'delete') prefix = '-';
        
        lineDiv.textContent = prefix + line.content;
        container.appendChild(lineDiv);
      });
    });
  }

  /**
   * Setup permission button handlers
   * @param {string} requestId - Request ID
   */
  function setupPermissionButtons(requestId) {
    const approveBtn = document.getElementById('permission-approve');
    const denyBtn = document.getElementById('permission-deny');
    const abortBtn = document.getElementById('permission-abort');
    
    if (!approveBtn || !denyBtn || !abortBtn) {
      console.error('Permission buttons not found');
      return;
    }
    
    // Remove old handlers by cloning
    const newApproveBtn = approveBtn.cloneNode(true);
    const newDenyBtn = denyBtn.cloneNode(true);
    const newAbortBtn = abortBtn.cloneNode(true);
    approveBtn.parentNode.replaceChild(newApproveBtn, approveBtn);
    denyBtn.parentNode.replaceChild(newDenyBtn, denyBtn);
    abortBtn.parentNode.replaceChild(newAbortBtn, abortBtn);
    
    // Get fresh references to the newly inserted buttons
    const finalApproveBtn = document.getElementById('permission-approve');
    const finalDenyBtn = document.getElementById('permission-deny');
    const finalAbortBtn = document.getElementById('permission-abort');
    
    // Add new handlers to the actual DOM elements
    finalApproveBtn.addEventListener('click', () => {
      console.log('Approve button clicked for request:', requestId);
      handlePermissionResponse(requestId, true);
    });
    
    finalDenyBtn.addEventListener('click', () => {
      console.log('Deny button clicked for request:', requestId);
      handlePermissionResponse(requestId, false);
    });
    
    finalAbortBtn.addEventListener('click', () => {
      console.log('Abort button clicked for request:', requestId);
      handlePermissionAbort(requestId);
    });
  }

  /**
   * Handle permission response
   * @param {string} requestId - Request ID
   * @param {boolean} approved - Whether approved or denied
   */
  async function handlePermissionResponse(requestId, approved) {
    const request = activePermissionRequests.get(requestId);
    if (!request) return;
    
    const rememberCheckbox = document.getElementById('permission-remember');
    const remember = rememberCheckbox ? rememberCheckbox.checked : false;
    
    // Hide modal
    const modal = document.getElementById('permission-modal');
    if (modal) {
      modal.style.display = 'none';
    }
    
    // Remove from active requests
    activePermissionRequests.delete(requestId);
    
    // Get current session ID
    const currentSessionId = window.AppState ? 
      window.AppState.getState('currentSessionId') : window.currentSessionId;
    
    // Send response to backend
    try {
      const response = await fetch('/api/permission-response', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          requestId: requestId,
          sessionId: currentSessionId,
          approved: approved,
          rememberChoice: remember
        })
      });
      
      if (!response.ok) {
        console.error('Failed to send permission response:', response.status);
      }
    } catch (error) {
      console.error('Error sending permission response:', error);
    }
    
    // Emit response event
    if (window.AppEvents) {
      window.AppEvents.emit(window.StandardEvents?.PERMISSION_RESPONSE || 'permission_response', {
        requestId,
        approved,
        remember
      });
    }
  }

  /**
   * Handle permission abort - completely stop the current operation
   * @param {string} requestId - Request ID
   */
  async function handlePermissionAbort(requestId) {
    const request = activePermissionRequests.get(requestId);
    if (!request) return;
    
    // Hide modal
    const modal = document.getElementById('permission-modal');
    if (modal) {
      modal.style.display = 'none';
    }
    
    // Remove from active requests
    activePermissionRequests.delete(requestId);
    
    // Get current session ID
    const currentSessionId = window.AppState ? 
      window.AppState.getState('currentSessionId') : window.currentSessionId;
    
    // Send abort signal to backend
    try {
      const response = await fetch('/api/permission-abort', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json'
        },
        body: JSON.stringify({
          request_id: requestId,
          session_id: currentSessionId
        })
      });
      
      if (!response.ok) {
        console.error('Failed to send permission abort:', response.status);
      }
      
      // Send abort message
      if (window.sendMessage) {
        await window.sendMessage('Important: ABORT!');
      }
      
      // Show notification
      if (window.addSystemMessageToUI) {
        window.addSystemMessageToUI('🛑 Operation completely aborted by user', 'error');
      }
    } catch (error) {
      console.error('Error sending permission abort:', error);
    }
    
    // Emit abort event
    if (window.AppEvents) {
      window.AppEvents.emit(window.StandardEvents?.PERMISSION_ABORT || 'permission_abort', {
        requestId
      });
    }
  }

  /**
   * Display file diff in a closable frame
   * @param {Object} data - Diff data
   */
  function displayFileDiff(data) {
    const { filePath, toolName, diff } = data;
    
    // Create or find the diff container
    let diffContainer = document.getElementById('file-diff-container');
    if (!diffContainer) {
      diffContainer = document.createElement('div');
      diffContainer.id = 'file-diff-container';
      diffContainer.className = 'file-diff-container';
      document.body.appendChild(diffContainer);
    }
    
    // Create the diff frame
    const diffFrame = document.createElement('div');
    diffFrame.className = 'diff-frame';
    diffFrame.innerHTML = `
      <div class="diff-header">
        <div class="diff-title">
          <span class="diff-icon">📝</span>
          <span class="diff-file-path">${escapeHtml(filePath)}</span>
          <span class="diff-tool">(${toolName})</span>
        </div>
        <button class="diff-close-btn" onclick="this.closest('.diff-frame').remove()">✕</button>
      </div>
      <div class="diff-content">
        <pre class="diff-text">${escapeHtml(diff)}</pre>
      </div>
    `;
    
    // Add the frame to the container
    diffContainer.appendChild(diffFrame);
    
    // Show the container
    diffContainer.style.display = 'block';
    
    // Auto-hide after 30 seconds
    setTimeout(() => {
      diffFrame.style.opacity = '0.7';
    }, 30000);
  }

  /**
   * Escape HTML for safe display
   * @param {string} text - Text to escape
   * @returns {string} Escaped HTML
   */
  function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  /**
   * Clear all active permission requests
   */
  function clearActiveRequests() {
    activePermissionRequests.clear();
  }

  /**
   * Get active permission request count
   */
  function getActiveRequestCount() {
    return activePermissionRequests.size;
  }

  // Export to global scope
  window.PermissionsModule = {
    handlePermissionRequest,
    showPermissionModal,
    handlePermissionResponse,
    handlePermissionAbort,
    displayFileDiff,
    clearActiveRequests,
    getActiveRequestCount
  };

  // Also expose main handler for backward compatibility
  window.handlePermissionRequest = handlePermissionRequest;

})();