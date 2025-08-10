let currentSessionId = null;
let eventSource = null;
let messageInput;
let editor = null;
let pendingNewSession = false; // Track if we're waiting to create a new session
let hasReceivedFirstResponse = false; // Track first response per message
let fileRefreshDelay = 9000 // Warn: must be greater than the cacheTTL of the backend which is currently 7s
let currentRequestController = null; // AbortController for current request
let isProcessing = false; // Track if currently processing a request

// Function to toggle between Send and Stop buttons
function toggleStopButton(show) {
  const sendBtn = document.getElementById('send-btn');
  const stopBtn = document.getElementById('stop-btn');
  const createPlanBtn = document.getElementById('create-plan-btn');
  
  if (!stopBtn) {
    // Create stop button if it doesn't exist
    const stopButton = document.createElement('button');
    stopButton.id = 'stop-btn';
    stopButton.className = 'btn-warning';
    stopButton.textContent = 'Stop';
    stopButton.style.display = 'none';
    stopButton.onclick = stopCurrentOperation;
    
    // Insert stop button after send button
    if (sendBtn) {
      sendBtn.parentNode.insertBefore(stopButton, sendBtn.nextSibling);
    }
  }
  
  const stopBtnElement = document.getElementById('stop-btn');
  if (show) {
    if (sendBtn) sendBtn.style.display = 'none';
    if (createPlanBtn) createPlanBtn.style.display = 'none';
    if (stopBtnElement) stopBtnElement.style.display = 'inline-block';
  } else {
    if (sendBtn) sendBtn.style.display = 'inline-block';
    if (stopBtnElement) stopBtnElement.style.display = 'none';
    // Show create plan button if in plan mode
    const planModeSwitch = document.getElementById('plan-mode-switch');
    if (planModeSwitch && planModeSwitch.checked && createPlanBtn) {
      createPlanBtn.style.display = 'inline-block';
      if (sendBtn) sendBtn.style.display = 'none';
    }
  }
}

// Function to stop the current operation
function stopCurrentOperation() {
  console.log('Stopping current operation...');
  
  // Abort the current request if any
  if (currentRequestController) {
    currentRequestController.abort();
    currentRequestController = null;
  }
  
  // Reset UI state
  toggleStopButton(false);
  isProcessing = false;
  isLLMProcessing = false; // Reset LLM processing state
  toolsAnnounced = false; // Reset tools announced flag
  
  // Clear any pending thinking return timer
  if (thinkingReturnTimer) {
    clearTimeout(thinkingReturnTimer);
    thinkingReturnTimer = null;
  }
  
  // Remove any thinking indicators
  const thinkingIndicators = document.querySelectorAll('.message.thinking');
  thinkingIndicators.forEach(indicator => indicator.remove());
  
  // Mark any executing tools as cancelled
  const executingTools = document.querySelectorAll('.tool-item.executing');
  executingTools.forEach(tool => {
    tool.classList.remove('executing');
    tool.classList.add('cancelled');
    const statusIcon = tool.querySelector('.tool-status-icon');
    if (statusIcon) statusIcon.textContent = '⚠️';
    const metrics = tool.querySelector('.tool-metrics');
    if (metrics) metrics.textContent = 'Cancelled';
  });
  
  // Clear active tool executions
  activeToolExecutions.clear();
}

// SSE connection tracking
let reconnectAttempts = 0;
let reconnectDelay = 1000; // Start with 1 second
const maxReconnectAttempts = 5;
const maxReconnectDelay = 30000; // Max 30 seconds
let isManuallyDisconnected = false;
let connectionStatus = 'disconnected'; // 'connected', 'disconnected', 'reconnecting'

// LLM response state tracking
let isLLMProcessing = false; // Track if LLM is still processing
let thinkingReturnTimer = null; // Timer to return to thinking state after tool completion
const THINKING_RETURN_DELAY = 2000; // 2 seconds delay before returning to thinking
let toolsAnnounced = false; // Track if tools have been announced for this response

console.log('Initializing JavaScript...');

// Configure marked.js when available
function configureMarked() {
  if (typeof marked !== 'undefined' && typeof hljs !== 'undefined') {
    marked.setOptions({
      breaks: true,
      gfm: true,
      headerIds: false,
      mangle: false,
      highlight: function(code, lang) {
        if (lang && hljs.getLanguage(lang)) {
          try {
            return hljs.highlight(code, { language: lang }).value;
          } catch (err) {}
        }
        return hljs.highlightAuto(code).value;
      }
    });
  }
}

function connectEventSource() {
  // Don't connect if manually disconnected
  if (isManuallyDisconnected) {
    console.log('Not connecting - manually disconnected');
    return;
  }

  // Close any existing connection first
  if (eventSource) {
    console.log('Closing existing EventSource before creating new connection');
    eventSource.close();
    eventSource = null;
  }

  console.log(`Attempting SSE connection (attempt ${reconnectAttempts + 1})`);

  // Update status to reconnecting if we're retrying
  if (reconnectAttempts > 0 || connectionStatus === 'reconnecting') {
    updateConnectionStatus('reconnecting');
  }

  eventSource = new EventSource('/events');

  eventSource.onopen = function() {
    console.log('SSE connection established');
    // Reset reconnection state on successful connection
    reconnectAttempts = 0;
    reconnectDelay = 1000;
    updateConnectionStatus('connected');
    
    // Refresh sessions in case server was restarted
    loadSessions();
  };

  eventSource.onmessage = function(event) {
    // console.log('SSE msg received (raw): ', event);

    const data = JSON.parse(event.data);
    handleServerEvent(data);
  };

  eventSource.onerror = function(error) {
    console.error('SSE error:', error);
    
    // Close the current connection
    if (eventSource) {
      eventSource.close();
      eventSource = null;
    }

    // Increment attempts first
    reconnectAttempts++;

    // Check if we've exceeded max reconnection attempts
    if (reconnectAttempts > maxReconnectAttempts) {
      console.error('Max reconnection attempts reached. Stopping auto-reconnect.');
      updateConnectionStatus('disconnected');
      showConnectionError('Connection to server lost. Please refresh the page or click reconnect.');
      return;
    }

    // Update status to show we're reconnecting with attempt count
    updateConnectionStatus('reconnecting');

    // Calculate next delay with exponential backoff
    const nextDelay = Math.min(reconnectDelay * 2, maxReconnectDelay);
    
    console.log(`Reconnecting in ${reconnectDelay/1000} seconds... (attempt ${reconnectAttempts}/${maxReconnectAttempts})`);
    
    setTimeout(() => {
      connectEventSource();
    }, reconnectDelay);
    
    // Update delay for next attempt
    reconnectDelay = nextDelay;
  };
}

// Manually reconnect SSE
function reconnectSSE() {
  console.log('Manual SSE reconnection requested');
  
  // Close any existing connection first
  if (eventSource) {
    console.log('Closing existing EventSource before reconnecting');
    eventSource.close();
    eventSource = null;
  }
  
  isManuallyDisconnected = false;
  reconnectAttempts = 0;
  reconnectDelay = 1000;
  
  // Update status to show we're starting fresh
  // Set attempts to 1 temporarily for display purposes
  const tempAttempts = reconnectAttempts;
  reconnectAttempts = 1;
  updateConnectionStatus('reconnecting');
  reconnectAttempts = tempAttempts;
  
  // Small delay to ensure UI updates before connection attempt
  setTimeout(() => {
    connectEventSource();
  }, 100);
}

// Disconnect SSE
function disconnectSSE() {
  console.log('Disconnecting SSE');
  isManuallyDisconnected = true;
  if (eventSource) {
    eventSource.close();
    eventSource = null;
  }
  updateConnectionStatus('disconnected');
}

// Update connection status in UI
function updateConnectionStatus(status) {
  connectionStatus = status;
  const statusElement = document.getElementById('connection-status');
  if (!statusElement) {
    console.error('Connection status element not found');
    return;
  }

  console.log(`Updating connection status to: ${status}`);

  // Remove all status classes
  statusElement.classList.remove('connected', 'disconnected', 'reconnecting');
  
  // Add current status class
  statusElement.classList.add(status);
  
  // Update text and visibility
  switch(status) {
    case 'connected':
      statusElement.style.display = 'none'; // Hide when connected
      statusElement.textContent = ''; // Clear text content
      console.log('Status set to connected - indicator should be hidden');
      break;
    case 'reconnecting':
      statusElement.textContent = `Reconnecting... (${reconnectAttempts}/${maxReconnectAttempts})`;
      statusElement.style.display = 'block';
      break;
    case 'disconnected':
      if (reconnectAttempts >= maxReconnectAttempts) {
        statusElement.innerHTML = 'Connection lost. <a href="#" onclick="reconnectSSE(); return false;">Reconnect</a>';
      } else {
        statusElement.textContent = 'Disconnected';
      }
      statusElement.style.display = 'block';
      break;
  }
}

// Show connection error message
function showConnectionError(message) {
  const messagesContainer = document.getElementById('messages');
  if (!messagesContainer) return;

  const errorDiv = document.createElement('div');
  errorDiv.className = 'connection-error';
  errorDiv.innerHTML = `
    <div class="error-content">
      <strong>Connection Error</strong>
      <p>${message}</p>
      <button onclick="reconnectSSE()" class="btn-secondary">Reconnect</button>
    </div>
  `;
  
  messagesContainer.appendChild(errorDiv);
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

// Handle server events
function handleServerEvent(event) {
  // already logged at parent // console.log('Received SSE event:', event);
  console.log('Event sessionId:', event.sessionId, 'Current sessionId:', currentSessionId, 'Match:', event.sessionId === currentSessionId);
  console.log('Global currentSessionId:', window.currentSessionId);
  
  // Special logging for permission events
  if (event.type === 'permission_request') {
    console.warn('PERMISSION EVENT RECEIVED:', {
      type: event.type,
      eventSessionId: event.sessionId,
      currentSessionId: currentSessionId,
      windowSessionId: window.currentSessionId,
      sessionMatch: event.sessionId === currentSessionId,
      data: event.data
    });
  }

  // Auto-switch to Files tab on first response
  if (event.sessionId === currentSessionId && !hasReceivedFirstResponse) {
    // Check for events that indicate the LLM is starting to respond
    if (event.type === 'content_start' || 
        event.type === 'tool_execution_start' ||
        (event.type === 'message_delta' && event.data && event.data.delta)) {
      
      // Switch to Files tab on first response
      if (window.FileExplorer && window.FileExplorer.switchTab) {
        console.log('Auto-switching to Files tab on first response');
        window.FileExplorer.switchTab('files');
        hasReceivedFirstResponse = true;
      }
    }
  }

  if (event.type === 'message_start' && event.sessionId === currentSessionId) {
    console.log('Message streaming started');
    isLLMProcessing = true; // LLM is now processing
    toolsAnnounced = false; // Reset tools announced flag for new message
    // Clear any pending thinking return timer
    if (thinkingReturnTimer) {
      clearTimeout(thinkingReturnTimer);
      thinkingReturnTimer = null;
    }
  } else if (event.type === 'content_start' && event.sessionId === currentSessionId) {
    console.log('Content started (text or tool) - checking if tools announced:', toolsAnnounced);
    // Only handle thinking indicator if NOT expecting tools
    if (!toolsAnnounced) {
      // This is pure text content, remove thinking indicator
      const thinkingIndicator = document.querySelector('.message.thinking');
      if (thinkingIndicator) {
        console.log('Removing thinking indicator for pure text content');
        thinkingIndicator.remove();
      }
    }
    // If tools were announced, keep the thinking/tool execution indicator
  } else if (event.type === 'tool_use_start' && event.sessionId === currentSessionId) {
    console.log('Tool use started - tools announced');
    toolsAnnounced = true; // Mark that tools have been announced
    // Transform thinking indicator to show tools are coming
    const thinkingIndicator = document.querySelector('.message.thinking');
    if (thinkingIndicator) {
      const content = thinkingIndicator.querySelector('.message-content');
      if (content) {
        content.innerHTML = '<span class="tool-executing">🛠️ Executing tools...</span>';
      }
    }
  } else if (event.type === 'message_delta' && event.sessionId === currentSessionId) {
    console.log('Message delta received:', event.data.delta);
    // Create streaming message container if it doesn't exist
    if (!currentStreamingMessageDiv) {
      createStreamingMessage();
    }
    // Append delta to streaming message
    appendToStreamingMessage(event.data.delta);
  } else if (event.type === 'message_stop' && event.sessionId === currentSessionId) {
    console.log('Message streaming stopped');
    // Finalize streaming message
    finalizeStreamingMessage();
    // Hide stop button when message completes
    toggleStopButton(false);
    isProcessing = false;
    isLLMProcessing = false; // LLM has finished processing
    toolsAnnounced = false; // Reset tools announced flag
    
    // Clear any pending thinking return timer
    if (thinkingReturnTimer) {
      clearTimeout(thinkingReturnTimer);
      thinkingReturnTimer = null;
    }
    
    // Remove any remaining thinking indicators
    const thinkingIndicators = document.querySelectorAll('.message.thinking');
    thinkingIndicators.forEach(indicator => indicator.remove());
  } else if (event.type === 'tool_execution_start' && event.sessionId === currentSessionId) {
    console.log('Tool execution started:', event.data);
    handleToolExecutionStart(event.data);
  } else if (event.type === 'tool_execution_progress' && event.sessionId === currentSessionId) {
    console.log('Tool execution progress:', event.data);
    handleToolExecutionProgress(event.data);
  } else if (event.type === 'tool_execution_complete' && event.sessionId === currentSessionId) {
    console.log('Tool execution completed:', event.data);
    handleToolExecutionComplete(event.data);

    // Refresh file tree after successful file-related tool operations
    const fileTools = ['write_file', 'edit_file', 'make_dir', 'remove', 'move'];
    if (event.data && event.data.toolName && window.FileExplorer) {
      // Extract tool name from the summary (format: "✓ Tool operation...")
      const toolName = event.data.toolName;
      if (fileTools.includes(toolName)) {
        // console.log('File operation detected, refreshing file tree for tool:', toolName);

        // Cancel any pending refresh to debounce multiple operations
        if (window.fileTreeRefreshTimeout) {
          clearTimeout(window.fileTreeRefreshTimeout);
          console.log('Cancelled pending file tree refresh, will reschedule');
        }

        // Schedule refresh with longer delay to ensure file system operations are complete
        // Using 1.5 second delay for better reliability with larger operations
        window.fileTreeRefreshTimeout = setTimeout(() => {
          console.log('Refreshing file tree now...');
          window.FileExplorer.loadFileTree();
          window.fileTreeRefreshTimeout = null;
        }, fileRefreshDelay);
      }
    }

  } else if (event.type === 'tool_usage' && event.sessionId === currentSessionId) {
    console.log('Tool usage event received:', event.data);
    // Add tool usage summary to UI
    addToolUsageSummaryToUI(event.data);
  } else if (event.type === 'session_list_updated') {
    loadSessions();
  } else if (event.type && event.type.startsWith('plan_') && event.session_id === currentSessionId) {
    // Handle plan-related events
    handlePlanEvent(event);
  } else if (event.type && (event.type === 'file_opened' || event.type === 'file_changed' || event.type === 'file_tree_update')) {
    // Handle file explorer events
    if (window.FileExplorer && window.FileExplorer.handleFileEvent) {
      window.FileExplorer.handleFileEvent(event);
    }
  } else if (event.type === 'diff_available' && event.sessionId === currentSessionId) {
    // Handle diff available event
    console.log('Diff available:', event.data);
    if (event.data && event.data.filePath) {
      // For diff_available, we'll mark as modified since it implies the file was changed
      // The file_changed event will handle marking as new if it's a creation
      if (window.FileExplorer && window.FileExplorer.markFileModified) {
        window.FileExplorer.markFileModified(event.data.filePath);
      }
      
      // Store the diff in the diff viewer
      if (window.diffViewer && event.data.diffId) {
        // Show notification that diff is available
        const notification = `📝 Changes detected in ${event.data.filePath}`;
        addSystemMessageToUI(notification, 'info');
      }
    }
  } else if (event.type === 'tool_permission_update' && event.sessionId === currentSessionId) {
    // Handle tool permission update event
    console.log('Tool permission updated:', event.data);
    // Could refresh the tools list or update specific tool UI
    // For now, just log it as the UI is already updated optimistically
  } else if (event.type === 'permission_request') {
    // Handle permission request - check session ID match
    console.log('Permission request received:', event.data);
    console.log('Session check:', event.sessionId, '===', currentSessionId, '?', event.sessionId === currentSessionId);
    if (event.sessionId === currentSessionId || event.sessionId === window.currentSessionId) {
      handlePermissionRequest(event.data);
    } else {
      console.warn('Permission request for different session, ignoring');
    }
  } else if (event.type === 'file_diff' && event.sessionId === currentSessionId) {
    // Handle file diff display
    console.log('File diff received:', event.data);
    displayFileDiff(event.data);
  }
}

// Load sessions
async function loadSessions() {
  try {
    const response = await fetch('/api/session');
    
    // Check if response is ok
    if (!response.ok) {
      console.error('Failed to fetch sessions:', response.status);
      return;
    }
    
    const sessions = await response.json();
    
    // Check if sessions is null or not an array
    if (!sessions || !Array.isArray(sessions)) {
      console.log('No sessions returned or invalid format');
      return;
    }

    const sessionList = document.getElementById('session-list');
    
    // Check if sessionList exists (it won't when on tools tab)
    if (!sessionList) {
      console.log('Session list element not found - likely on a different tab');
      return;
    }
    
    sessionList.innerHTML = '';

    sessions.forEach(session => {
      const item = document.createElement('div');
      item.className = 'session-item' + (session.id === currentSessionId ? ' active' : '');
      item.textContent = session.title || 'Session ' + session.id.substring(0, 8);
      item.onclick = () => selectSession(session.id);
      sessionList.appendChild(item);
    });
  } catch (error) {
    console.error('Failed to load sessions:', error);
  }
}

// Select session
function selectSession(sessionId) {
  currentSessionId = sessionId;
  window.currentSessionId = sessionId; // make sure it is globally available
  pendingNewSession = false; // Clear pending state when selecting existing session
  loadMessages();
  loadSessions(); // Refresh to update active state
}

// Load messages for current session
async function loadMessages() {
  if (!currentSessionId) return;

  try {
    // Fetch messages and prompts in parallel
    const [messagesResponse, promptsResponse] = await Promise.all([
      fetch('/api/session/' + currentSessionId + '/messages'),
      fetch('/api/session/' + currentSessionId + '/prompts')
    ]);

    const messages = await messagesResponse.json();
    const prompts = await promptsResponse.json();

    const messagesContainer = document.getElementById('messages');
    messagesContainer.innerHTML = '';

    // Display initial prompts if any
    if (prompts && prompts.length > 0) {
      const promptsDiv = document.createElement('div');
      promptsDiv.className = 'initial-prompts';
      promptsDiv.innerHTML = `
        <div class="prompts-header">
          <span class="prompts-icon">📝</span>
          <span class="prompts-title">Initial Prompts</span>
        </div>
        <div class="prompts-content">
          ${prompts.map(prompt => `
            <div class="prompt-item">
              <div class="prompt-name">${prompt.name}</div>
              <div class="prompt-content">${prompt.content}</div>
              ${prompt.includes_permissions ? '<span class="prompt-badge">Includes Permissions</span>' : ''}
            </div>
          `).join('')}
        </div>
      `;
      messagesContainer.appendChild(promptsDiv);
    }

    messages.forEach(msg => {
      addMessageToUI(msg);
    });

    messagesContainer.scrollTop = messagesContainer.scrollHeight;
  } catch (error) {
    console.error('Failed to load messages:', error);
  }
}

// Add tool usage summary to UI
function addToolUsageSummaryToUI(toolData) {
  const messagesContainer = document.getElementById('messages');
  
  // Find the thinking indicator if it exists
  const thinkingIndicator = messagesContainer.querySelector('.message.thinking');
  
  // Create or find the tools summary container
  let toolsSummary = document.querySelector('.tools-summary.active');
  if (!toolsSummary) {
    toolsSummary = document.createElement('div');
    toolsSummary.className = 'tools-summary active';
    toolsSummary.innerHTML = '<div class="tools-header">🛠️ TOOL USE</div><div class="tools-list"></div>';
    
    // Insert before thinking indicator if it exists, otherwise append
    if (thinkingIndicator) {
      messagesContainer.insertBefore(toolsSummary, thinkingIndicator);
    } else {
      messagesContainer.appendChild(toolsSummary);
    }
  }
  
  // Add the tool usage to the list
  const toolsList = toolsSummary.querySelector('.tools-list');
  const toolItem = document.createElement('div');
  toolItem.className = 'tool-item';
  
  // Check if summary contains newlines (for diffs)
  if (toolData.summary && toolData.summary.includes('\n')) {
    // For multiline content, preserve the formatting
    const lines = toolData.summary.split('\n');
    const firstLine = lines[0];
    const diffContent = lines.slice(1).join('\n');
    
    // Create structure for expandable diff
    toolItem.innerHTML = `
      <div class="tool-summary-line">${escapeHtml(firstLine)}</div>
      ${diffContent ? `<pre class="tool-diff-content">${escapeHtml(diffContent)}</pre>` : ''}
    `;
  } else {
    // Single line summary
    toolItem.textContent = toolData.summary;
  }
  
  toolsList.appendChild(toolItem);
  
  // Scroll to bottom
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

// Helper function to quickly add a message
function addMessage(role, content) {
  addMessageToUI({ role: role, content: content });
}

// Add system message to UI
function addSystemMessageToUI(message, type = 'info') {
  const messagesContainer = document.getElementById('messages');
  
  const messageDiv = document.createElement('div');
  messageDiv.className = `system-message ${type}`;
  
  const icon = type === 'error' ? '❌' : type === 'warning' ? '⚠️' : 'ℹ️';
  
  messageDiv.innerHTML = `
    <span class="system-message-icon">${icon}</span>
    <span class="system-message-text">${message}</span>
  `;
  
  messagesContainer.appendChild(messageDiv);
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

// Add message to UI
function addMessageToUI(message) {
  const messagesContainer = document.getElementById('messages');
  
  // Mark any active tools summary as inactive
  const activeToolsSummary = document.querySelector('.tools-summary.active');
  if (activeToolsSummary) {
    activeToolsSummary.classList.remove('active');
  }
  
  const messageDiv = document.createElement('div');
  messageDiv.className = 'message ' + message.role;

  const header = document.createElement('div');
  header.className = 'message-header';

  if (message.role === 'user') {
    header.textContent = 'You';
  } else {
    // Show which model responded - use actual model from response if available
    let modelName = 'Assistant';
    const modelId = message.model || '';

    if (modelId.includes('opus-4-1')) {
      modelName = 'Claude Opus 4.1';
    } else if (modelId.includes('opus-4')) {
      modelName = 'Claude Opus 4';
    } else if (modelId.includes('sonnet-4')) {
      modelName = 'Claude Sonnet 4';
    } else if (modelId.includes('3-opus')) {
      modelName = 'Claude 3 Opus';
    } else if (modelId.includes('haiku')) {
      modelName = 'Claude 3 Haiku';
    } else if (modelId.includes('3-5-sonnet')) {
      modelName = 'Claude 3.5 Sonnet';
    } else if (modelId.includes('3-sonnet')) {
      modelName = 'Claude 3 Sonnet';
    }
    header.textContent = modelName;
    console.log('Model name:', modelName);
  }

  const content = document.createElement('div');
  content.className = 'message-content';

  if (message.role === 'assistant' && typeof marked !== 'undefined') {
    // Render markdown for assistant messages
    content.innerHTML = marked.parse(message.content);

    // Highlight code blocks if highlight.js is available
    if (typeof hljs !== 'undefined') {
      content.querySelectorAll('pre code').forEach((block) => {
        hljs.highlightElement(block);
      });
    }
  } else {
    // Plain text for user messages or if marked is not available
    content.textContent = message.content;
  }

  messageDiv.appendChild(header);
  messageDiv.appendChild(content);
  messagesContainer.appendChild(messageDiv);

  // Scroll to bottom
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

// Add thinking indicator
function addThinkingIndicator(id) {
  const messagesContainer = document.getElementById('messages');
  const thinkingDiv = document.createElement('div');
  thinkingDiv.id = id;
  thinkingDiv.className = 'message assistant thinking';

  const header = document.createElement('div');
  header.className = 'message-header';

  // Show which model is thinking
  const modelSelector = document.getElementById('model-selector');
  let modelName = 'Assistant';
  if (modelSelector) {
    const value = modelSelector.value;
    if (value.includes('opus-4-1')) {
      modelName = 'Claude Opus 4.1';
    } else if (value.includes('opus-4')) {
      modelName = 'Claude Opus 4';
    } else if (value.includes('sonnet-4')) {
      modelName = 'Claude Sonnet 4';
    } else if (value.includes('3-opus')) {
      modelName = 'Claude 3 Opus';
    } else if (value.includes('haiku')) {
      modelName = 'Claude 3 Haiku';
    } else if (value.includes('3-5-sonnet')) {
      modelName = 'Claude 3.5 Sonnet';
    } else if (value.includes('3-sonnet')) {
      modelName = 'Claude 3 Sonnet';
    }
  }
  header.textContent = modelName;

  const content = document.createElement('div');
  content.className = 'message-content';
  content.innerHTML = '<span class="thinking-dots">Thinking<span>.</span><span>.</span><span>.</span></span>';

  thinkingDiv.appendChild(header);
  thinkingDiv.appendChild(content);
  messagesContainer.appendChild(thinkingDiv);

  // Scroll to bottom
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

// Remove thinking indicator
function removeThinkingIndicator(id) {
  const thinkingDiv = document.getElementById(id);
  if (thinkingDiv) {
    console.log('Removing thinking indicator:', id);
    thinkingDiv.remove();
  }
}

// Variables to track streaming state
let currentStreamingMessageDiv = null;
let currentStreamingContent = '';

// Create streaming message container
function createStreamingMessage() {
  const messagesContainer = document.getElementById('messages');
  
  // Remove any existing thinking indicators
  const thinkingIndicators = messagesContainer.querySelectorAll('.message.thinking');
  thinkingIndicators.forEach(indicator => indicator.remove());
  
  // Create message container
  const messageDiv = document.createElement('div');
  messageDiv.className = 'message assistant streaming';
  
  const header = document.createElement('div');
  header.className = 'message-header';
  
  // Show which model is responding
  const modelSelector = document.getElementById('model-selector');
  let modelName = 'Assistant';
  if (modelSelector && modelSelector.value) {
    // Extract model name from value
    const parts = modelSelector.value.split('-');
    if (parts.length > 1) {
      modelName = parts[1].charAt(0).toUpperCase() + parts[1].slice(1);
    }
  }
  
  header.innerHTML = `<span class="role">${modelName}</span>`;
  
  const content = document.createElement('div');
  content.className = 'message-content';
  content.innerHTML = '<span class="streaming-cursor"></span>';
  
  messageDiv.appendChild(header);
  messageDiv.appendChild(content);
  messagesContainer.appendChild(messageDiv);
  
  // Store reference
  currentStreamingMessageDiv = messageDiv;
  currentStreamingContent = '';
  
  // Scroll to bottom
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

// Append text to streaming message
function appendToStreamingMessage(delta) {
  if (!currentStreamingMessageDiv) {
    createStreamingMessage();
  }
  
  const content = currentStreamingMessageDiv.querySelector('.message-content');
  currentStreamingContent += delta;
  
  // Process markdown and update content
  const processedContent = window.marked ? marked.parse(currentStreamingContent) : currentStreamingContent;
  content.innerHTML = processedContent + '<span class="streaming-cursor"></span>';
  
  // Highlight code blocks if they exist
  content.querySelectorAll('pre code').forEach((block) => {
    if (window.hljs) {
      hljs.highlightElement(block);
    }
  });
  
  // Smooth scroll to bottom
  const messagesContainer = document.getElementById('messages');
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

// Finalize streaming message
function finalizeStreamingMessage() {
  if (!currentStreamingMessageDiv) return;
  
  // Remove streaming class and cursor
  currentStreamingMessageDiv.classList.remove('streaming');
  const cursor = currentStreamingMessageDiv.querySelector('.streaming-cursor');
  if (cursor) cursor.remove();
  
  // Final markdown processing
  const content = currentStreamingMessageDiv.querySelector('.message-content');
  const processedContent = window.marked ? marked.parse(currentStreamingContent) : currentStreamingContent;
  content.innerHTML = processedContent;
  
  // Final code highlighting
  content.querySelectorAll('pre code').forEach((block) => {
    if (window.hljs) {
      hljs.highlightElement(block);
    }
  });
  
  // Reset streaming state
  currentStreamingMessageDiv = null;
  currentStreamingContent = '';
}

// Send message
async function sendMessage() {
  console.log('sendMessage called');

  if (!editor) {
    console.error('Monaco editor not initialized');
    return;
  }

  const content = editor.getValue().trim();
  console.log('Message content:', content);

  if (!content) {
    console.log('No content, returning');
    return;
  }

  // Create session if needed
  if (!currentSessionId || pendingNewSession) {
    console.log('Creating new session before sending message');
    try {
      await actuallyCreateSession();
      console.log('Session created:', currentSessionId);
    } catch (error) {
      console.error('Failed to create session:', error);
      alert('Failed to create session. Please try again.');
      return;
    }
  }

  console.log('Sending to session:', currentSessionId);
  console.log('Window session ID:', window.currentSessionId);

  // Reset first response flag for new message
  hasReceivedFirstResponse = false;

  // Add user message to UI immediately
  addMessageToUI({ role: 'user', content: content });

  // Clear input
  editor.setValue('');

  // Add a thinking indicator
  const thinkingId = 'thinking-' + Date.now();
  addThinkingIndicator(thinkingId);

  // Show stop button and hide send button
  toggleStopButton(true);
  isProcessing = true;
  isLLMProcessing = true; // Mark LLM as processing

  try {
    // Create abort controller for this request
    currentRequestController = new AbortController();

    // Get selected model
    const modelSelector = document.getElementById('model-selector');
    const selectedModel = modelSelector ? modelSelector.value : 'claude-sonnet-4-20250514';

    console.log('Making API request with model:', selectedModel);
    const response = await fetch('/api/session/' + currentSessionId + '/message', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        content: content,
        model: selectedModel
      }),
      signal: currentRequestController.signal
    });

    console.log('Response status:', response.status);

    // Don't remove thinking indicator here - let SSE events handle it during streaming

    if (!response.ok) {
      // Remove thinking indicator only on error
      removeThinkingIndicator(thinkingId);
      isLLMProcessing = false; // Reset LLM processing on error
      toolsAnnounced = false; // Reset tools announced flag
      
      // Clear any pending thinking return timer
      if (thinkingReturnTimer) {
        clearTimeout(thinkingReturnTimer);
        thinkingReturnTimer = null;
      }
      
      const errorText = await response.text();
      console.error('Response error:', errorText);
      
      // Handle session not found error
      if (response.status === 404) {
        console.log('Session not found, creating new session');
        // Clear current session and create a new one
        currentSessionId = null;
        await createNewSession();
        
        // Show error message to user
        addMessageToUI({
          role: 'assistant',
          content: 'Previous session was lost (server may have restarted). Created a new session. Please resend your message.'
        });
        
        // Restore the user's message to the input
        editor.setValue(content);
        return;
      }
      
      throw new Error('Failed to send message: ' + errorText);
    }

    const result = await response.json();
    console.log('Response data:', result);
    
    // Remove thinking indicator when we get the response
    removeThinkingIndicator(thinkingId);
    
    // Display tool summaries if any
    if (result.toolSummaries && result.toolSummaries.length > 0) {
      // Create tools summary container
      const messagesContainer = document.getElementById('messages');
      const toolsSummary = document.createElement('div');
      toolsSummary.className = 'tools-summary';
      toolsSummary.innerHTML = '<div class="tools-header">🛠️ TOOL USE</div><div class="tools-list"></div>';
      
      const toolsList = toolsSummary.querySelector('.tools-list');
      result.toolSummaries.forEach(summary => {
        const toolItem = document.createElement('div');
        toolItem.className = 'tool-item';
        toolItem.textContent = summary;
        toolsList.appendChild(toolItem);
      });
      
      messagesContainer.appendChild(toolsSummary);
    }

    // Response content already streamed via SSE deltas - no need to add again
    
    // Reload sessions to show updated title (for first message)
    // The backend will have updated the session title based on the first user message
    loadSessions();
  } catch (error) {
    // Remove thinking indicator on error
    removeThinkingIndicator(thinkingId);
    
    // Only show error if not aborted by user
    if (error.name !== 'AbortError') {
      console.error('Failed to send message:', error);
      alert('Failed to send message: ' + error.message);
    } else {
      console.log('Request cancelled by user');
      addMessageToUI({
        role: 'assistant',
        content: '⚠️ Operation cancelled by user'
      });
    }
  } finally {
    // Reset UI state
    toggleStopButton(false);
    isProcessing = false;
    isLLMProcessing = false; // Reset LLM processing state
    toolsAnnounced = false; // Reset tools announced flag
    currentRequestController = null;
    
    // Clear any pending thinking return timer
    if (thinkingReturnTimer) {
      clearTimeout(thinkingReturnTimer);
      thinkingReturnTimer = null;
    }
  }
}

// Actually create a new session in the database
async function actuallyCreateSession() {
  try {
    const response = await fetch('/api/session', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });

    const session = await response.json();
    currentSessionId = session.id;
    window.currentSessionId = session.id; // Ensure global is also set
    pendingNewSession = false;
    
    // Don't reload sessions immediately - wait for title to be set
    return session;
  } catch (error) {
    console.error('Failed to create session:', error);
    throw error;
  }
}

// Prepare UI for new session (called by New Session button)
async function createNewSession() {
  // Just prepare UI for new session
  currentSessionId = null;
  pendingNewSession = true;
  
  // Clear messages
  document.getElementById('messages').innerHTML = '';
  
  // Remove active class from all sessions
  document.querySelectorAll('.session-item').forEach(item => {
    item.classList.remove('active');
  });
  
  // Focus on input
  if (editor) {
    editor.focus();
  }
}

// Update message (for streaming)
function updateMessage(messageData) {
  // This will be implemented when we add streaming support
  console.log('Message update:', messageData);
}

// Logout
async function logout() {
  try {
    await fetch('/api/auth/logout', { method: 'POST' });
    window.location.reload();
  } catch (error) {
    console.error('Logout failed:', error);
  }
}

// Plan Mode Management
let isPlanMode = false;
let currentPlan = null;
let currentPlanId = null;
let planSteps = new Map(); // Map of step ID to step data

function initializePlanMode() {
  const planModeSwitch = document.getElementById('plan-mode-switch');
  const planModeIndicator = document.getElementById('plan-mode-indicator');
  const sendBtn = document.getElementById('send-btn');
  const createPlanBtn = document.getElementById('create-plan-btn');
  const planExecutionArea = document.getElementById('plan-execution-area');
  const closePlanBtn = document.getElementById('close-plan-btn');
  
  if (!planModeSwitch) return;
  
  // Toggle plan mode
  planModeSwitch.addEventListener('change', function() {
    isPlanMode = this.checked;
    document.body.classList.toggle('plan-mode', isPlanMode);
    
    if (isPlanMode) {
      planModeIndicator.style.display = 'block';
      sendBtn.style.display = 'none';
      createPlanBtn.style.display = 'inline-block';
      if (editor) {
        editor.updateOptions({ placeholder: 'Describe a complex task to create a plan...' });
      }
    } else {
      planModeIndicator.style.display = 'none';
      sendBtn.style.display = 'inline-block';
      createPlanBtn.style.display = 'none';
      if (editor) {
        editor.updateOptions({ placeholder: 'Type a message...' });
      }
    }
  });
  
  // Create plan button
  if (createPlanBtn) {
    createPlanBtn.addEventListener('click', createPlan);
  }
  
  // Close plan execution area
  if (closePlanBtn) {
    closePlanBtn.addEventListener('click', function() {
      planExecutionArea.style.display = 'none';
      document.body.classList.remove('plan-executing');
    });
  }
  
  // Plan control buttons
  const executePlanBtn = document.getElementById('execute-plan-btn');
  const pausePlanBtn = document.getElementById('pause-plan-btn');
  const rollbackPlanBtn = document.getElementById('rollback-plan-btn');
  const viewMetricsBtn = document.getElementById('view-metrics-btn');
  
  if (executePlanBtn) {
    executePlanBtn.addEventListener('click', executePlan);
  }
  
  if (pausePlanBtn) {
    pausePlanBtn.addEventListener('click', pausePlan);
  }
  
  if (rollbackPlanBtn) {
    rollbackPlanBtn.addEventListener('click', showRollbackDialog);
  }
  
  if (viewMetricsBtn) {
    viewMetricsBtn.addEventListener('click', viewPlanMetrics);
  }
}

async function createPlan() {
  const content = editor.getValue().trim();
  if (!content || !currentSessionId) return;
  
  try {
    const response = await fetch(`/api/session/${currentSessionId}/plan`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ 
        description: content,
        auto_execute: false 
      })
    });
    
    if (!response.ok) {
      throw new Error('Failed to create plan');
    }
    
    const plan = await response.json();
    currentPlan = plan;
    
    // Clear the editor
    editor.setValue('');
    
    // Show the plan in the messages area
    addMessage('assistant', `📋 **Task Plan Created**\n\nI've created a plan with ${plan.steps.length} steps to: ${plan.description}`);
    
    // Show plan execution area
    displayPlan(plan);
    
  } catch (error) {
    console.error('Error creating plan:', error);
    addMessage('assistant', '❌ Failed to create task plan. Please try again.');
  }
}

function displayPlan(plan) {
  const planExecutionArea = document.getElementById('plan-execution-area');
  const planStepsContainer = document.getElementById('plan-steps');
  const progressText = document.getElementById('progress-text');
  
  // Clear previous steps
  planStepsContainer.innerHTML = '';
  planSteps.clear();
  
  // Update progress text
  progressText.textContent = `0 / ${plan.steps.length} steps`;
  
  // Display each step
  plan.steps.forEach((step, index) => {
    const stepElement = createStepElement(step, index + 1);
    planStepsContainer.appendChild(stepElement);
    planSteps.set(step.id, { element: stepElement, data: step });
  });
  
  // Show the plan execution area
  planExecutionArea.style.display = 'flex';
  document.body.classList.add('plan-executing');
  
  // Enable/disable buttons based on plan status
  updatePlanControls(plan.status);
}

function createStepElement(step, number) {
  const stepDiv = document.createElement('div');
  stepDiv.className = 'plan-step';
  stepDiv.id = `step-${step.id}`;
  
  stepDiv.innerHTML = `
    <div class="step-header">
      <div class="step-info">
        <span class="step-number">${number}</span>
        <span class="step-title">${step.description}</span>
      </div>
      <span class="step-status ${step.status || 'pending'}">${step.status || 'pending'}</span>
    </div>
    <div class="step-details">
      <span class="step-tool">Tool: ${step.tool}</span>
    </div>
    <div class="step-output" style="display: none;"></div>
    <div class="step-metrics" style="display: none;"></div>
  `;
  
  return stepDiv;
}

async function executePlan() {
  if (!currentPlan) return;
  
  try {
    const response = await fetch(`/api/plan/${currentPlan.id}/execute`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });
    
    if (!response.ok) {
      throw new Error('Failed to execute plan');
    }
    
    // Update controls
    document.getElementById('execute-plan-btn').disabled = true;
    document.getElementById('pause-plan-btn').disabled = false;
    document.getElementById('rollback-plan-btn').disabled = false;
    
    addMessage('assistant', '🚀 Plan execution started...');
    
  } catch (error) {
    console.error('Error executing plan:', error);
    addMessage('assistant', '❌ Failed to execute plan. Please try again.');
  }
}

function pausePlan() {
  // TODO: Implement pause functionality
  console.log('Pause plan - not yet implemented');
}

function showRollbackDialog() {
  // TODO: Show checkpoint selection dialog
  console.log('Rollback - not yet implemented');
}

async function viewPlanMetrics() {
  if (!currentPlan) return;
  
  try {
    const response = await fetch(`/api/plan/${currentPlan.id}/status`);
    if (!response.ok) throw new Error('Failed to get metrics');
    
    const status = await response.json();
    
    // Display metrics in a message
    let metricsText = `📊 **Plan Execution Metrics**\n\n`;
    metricsText += `Total Steps: ${status.total_steps}\n`;
    metricsText += `Completed: ${status.completed_steps}\n`;
    metricsText += `Failed: ${status.failed_steps}\n`;
    if (status.metrics && status.metrics.total_duration) {
      metricsText += `Duration: ${formatDuration(status.metrics.total_duration)}\n`;
    }
    
    addMessage('assistant', metricsText);
    
  } catch (error) {
    console.error('Error getting metrics:', error);
  }
}

function updatePlanControls(status) {
  const executeBtn = document.getElementById('execute-plan-btn');
  const pauseBtn = document.getElementById('pause-plan-btn');
  const rollbackBtn = document.getElementById('rollback-plan-btn');
  
  switch (status) {
    case 'pending':
      executeBtn.disabled = false;
      pauseBtn.disabled = true;
      rollbackBtn.disabled = true;
      break;
    case 'executing':
      executeBtn.disabled = true;
      pauseBtn.disabled = false;
      rollbackBtn.disabled = false;
      break;
    case 'completed':
    case 'failed':
      executeBtn.disabled = true;
      pauseBtn.disabled = true;
      rollbackBtn.disabled = false;
      break;
  }
}

// Handle plan-related SSE events
function handlePlanEvent(event) {
  switch (event.type) {
    case 'plan_created':
      handlePlanCreated(event.data);
      break;
    case 'step_progress':
      handleStepProgress(event.data);
      break;
    case 'plan_completed':
      handlePlanCompleted(event.data);
      break;
  }
}

function handlePlanCreated(data) {
  console.log('Plan created:', data);
  // Plan creation is already handled in createPlan()
}

function handleStepProgress(data) {
  const stepInfo = planSteps.get(data.step_id);
  if (!stepInfo) return;
  
  const { element } = stepInfo;
  const statusElement = element.querySelector('.step-status');
  const outputElement = element.querySelector('.step-output');
  
  // Update step status
  element.className = `plan-step ${data.status}`;
  statusElement.className = `step-status ${data.status}`;
  statusElement.textContent = data.status;
  
  // Show output if available
  if (data.output) {
    outputElement.style.display = 'block';
    outputElement.textContent = typeof data.output === 'string' ? data.output : JSON.stringify(data.output, null, 2);
  }
  
  // Update progress bar
  updateProgressBar();
  
  // Show metrics if step completed
  if (data.status === 'completed' && data.metrics) {
    const metricsElement = element.querySelector('.step-metrics');
    metricsElement.style.display = 'flex';
    metricsElement.innerHTML = `
      <span>Duration: ${formatDuration(data.metrics.duration)}</span>
      ${data.metrics.retry_count > 0 ? `<span>Retries: ${data.metrics.retry_count}</span>` : ''}
    `;
  }
}

function handlePlanCompleted(data) {
  updatePlanControls(data.status);
  
  if (data.status === 'completed') {
    addMessage('assistant', '✅ Plan execution completed successfully!');
  } else if (data.status === 'failed') {
    addMessage('assistant', '❌ Plan execution failed. You can try to rollback to a previous checkpoint.');
  }
}

function updateProgressBar() {
  const progressFill = document.getElementById('progress-fill');
  const progressText = document.getElementById('progress-text');
  
  let completed = 0;
  let total = planSteps.size;
  
  planSteps.forEach(step => {
    const status = step.element.querySelector('.step-status').textContent;
    if (status === 'completed' || status === 'failed') {
      completed++;
    }
  });
  
  const percentage = total > 0 ? (completed / total) * 100 : 0;
  progressFill.style.width = `${percentage}%`;
  progressText.textContent = `${completed} / ${total} steps`;
}

function formatDuration(duration) {
  // Duration is in nanoseconds, convert to readable format
  const ms = duration / 1000000;
  if (ms < 1000) return `${ms.toFixed(0)}ms`;
  const s = ms / 1000;
  if (s < 60) return `${s.toFixed(1)}s`;
  const m = s / 60;
  return `${m.toFixed(1)}m`;
}

// Wait for DOM to be ready
document.addEventListener('DOMContentLoaded', function() {
  console.log('DOM loaded, initializing...');

  // Configure marked.js
  configureMarked();

  // Initialize Plan Mode
  initializePlanMode();

  // Initialize model selector
  const modelSelector = document.getElementById('model-selector');
  if (modelSelector) {
    // Load saved model preference
    const savedModel = localStorage.getItem('selectedModel');
    if (savedModel) {
      modelSelector.value = savedModel;
    }

    // Save model preference on change
    modelSelector.addEventListener('change', function() {
      localStorage.setItem('selectedModel', modelSelector.value);
      console.log('Model changed to:', modelSelector.value);
    });
  }

  // Connect SSE
  connectEventSource();

  // Load initial data
  loadSessions();

  // Button handlers
  const sendBtn = document.getElementById('send-btn');
  if (sendBtn) {
    sendBtn.onclick = () => {
      console.log('Send button clicked');
      sendMessage();
    };
  } else {
    console.error('Send button not found!');
  }

  const clearBtn = document.getElementById('clear-btn');
  if (clearBtn) {
    clearBtn.onclick = () => {
      console.log('Clear button clicked');
      if (editor) editor.setValue('');
    };
  }

  const newSessionBtn = document.getElementById('new-session-btn');
  if (newSessionBtn) {
    newSessionBtn.onclick = createNewSession;
  }

  const logoutBtn = document.getElementById('logout-btn');
  if (logoutBtn) {
    logoutBtn.onclick = logout;
  }

  // CONFIGURE MONACO loader
  require.config({ paths: { 'vs': 'https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.52.2/min/vs' }});

// Store it globally so our main script can use it
  // window.monacoReady = new Promise((resolve) => {
  require(['vs/editor/editor.main'], function() {

    // Make our own theme
    monaco.editor.defineTheme('ro-dark', {
      base: 'vs-dark',
      inherit: true,
      rules: [
        { background: '1d1f21' },
        { token: 'comment', foreground: '909090' },
        { token: 'string', foreground: 'b5bd68' },
        { token: 'variable', foreground: 'c5c8c6' },
        { token: 'keyword', foreground: 'ba7d57' },
        { token: 'number', foreground: 'de935f' },
      ],
      colors: {
        'editorBackground': '#1d1f21',
        // 'editorForeground': '#c5c8c6',
        // 'editor.selectionBackground': '#373b41',
        'editorCursor.foreground': '#6DDADA',
        'editor.lineHighlightBackground': '#606060',
      }
    });

    // var init_val = document.getElementById("note_body").value;
    editor = monaco.editor.create(document.getElementById('monaco-container'), {
      value: '',
      language: 'markdown',
      theme: 'ro-dark',
      minimap: { enabled: false },
      lineNumbers: 'off',
      glyphMargin: false,
      folding: false,
      lineDecorationsWidth: 0,
      lineNumbersMinChars: 0,
      renderLineHighlight: 'gutter',
      scrollBeyondLastLine: false,
      wordWrap: 'on',
      automaticLayout: true,
      fontSize: 14,
      fontFamily: 'Monaco, Menlo, Consolas, "Courier New", monospace',
      padding: { top: 10, bottom: 10 },
      overviewRulerLanes: 0,
      hideCursorInOverviewRuler: true,
      scrollbar: {
        vertical: 'auto',
        horizontal: 'hidden'
      },
      contextmenu: false,
      quickSuggestions: false,
      parameterHints: { enabled: false },
      suggestOnTriggerCharacters: false,
      acceptSuggestionOnEnter: 'off',
      tabCompletion: 'off',
      wordBasedSuggestions: false
    });

    // Add keyboard shortcut for Ctrl/Cmd+Enter
    editor.addCommand(monaco.KeyCode.Enter, function() {
      sendMessage();
    });

    // Focus the editor
    editor.focus();
    // console.log('Monaco editor initialized successfully');

    console.log('Monaco is ready');
    // resolve();
  });
  //});

  // Initialize Monaco Editor
  // initializeMonacoEditor();

  console.log('JavaScript Initialization complete');
  
  // Initialize Plan History
  initializePlanHistory();
});

// Plan History functionality
let planHistoryPage = 1;
let planHistoryLoading = false;
let planHistorySearch = '';
let planHistoryStatus = '';

function initializePlanHistory() {
  const historyBtn = document.getElementById('plan-history-btn');
  const historyPanel = document.getElementById('plan-history-panel');
  const closeHistoryBtn = document.getElementById('close-history-btn');
  const searchInput = document.getElementById('plan-search');
  const statusFilter = document.getElementById('plan-status-filter');
  const loadMoreBtn = document.getElementById('load-more-plans');
  
  if (!historyBtn || !historyPanel) {
    console.log('Plan history elements not found');
    return;
  }
  
  // Toggle panel
  historyBtn.addEventListener('click', () => {
    historyPanel.classList.toggle('open');
    if (historyPanel.classList.contains('open')) {
      planHistoryPage = 1;
      loadPlanHistory(true);
    }
  });
  
  // Close panel
  closeHistoryBtn.addEventListener('click', () => {
    historyPanel.classList.remove('open');
  });
  
  // Search functionality
  let searchTimeout;
  searchInput.addEventListener('input', (e) => {
    clearTimeout(searchTimeout);
    planHistorySearch = e.target.value;
    searchTimeout = setTimeout(() => {
      planHistoryPage = 1;
      loadPlanHistory(true);
    }, 300);
  });
  
  // Filter functionality
  statusFilter.addEventListener('change', (e) => {
    planHistoryStatus = e.target.value;
    planHistoryPage = 1;
    loadPlanHistory(true);
  });
  
  // Load more
  loadMoreBtn.addEventListener('click', () => {
    planHistoryPage++;
    loadPlanHistory(false);
  });
}

async function loadPlanHistory(reset = false) {
  if (planHistoryLoading || !currentSessionId) return;
  
  planHistoryLoading = true;
  const historyList = document.getElementById('plan-history-list');
  const loadMoreBtn = document.getElementById('load-more-plans');
  
  if (reset) {
    historyList.innerHTML = '<div class="loading">Loading plan history...</div>';
  }
  
  try {
    const params = new URLSearchParams({
      page: planHistoryPage,
      limit: 20
    });
    
    if (planHistorySearch) {
      params.append('search', planHistorySearch);
    }
    
    if (planHistoryStatus) {
      params.append('status', planHistoryStatus);
    }
    
    const response = await fetch(`/api/session/${currentSessionId}/plans/history?${params}`);
    if (!response.ok) {
      throw new Error('Failed to load plan history');
    }
    
    const data = await response.json();
    
    if (reset) {
      historyList.innerHTML = '';
    }
    
    if (data.plans.length === 0 && reset) {
      historyList.innerHTML = '<div class="loading">No plans found</div>';
    } else {
      data.plans.forEach(plan => {
        historyList.appendChild(createPlanHistoryItem(plan));
      });
    }
    
    // Show/hide load more button
    if (data.page * data.limit < data.total) {
      loadMoreBtn.style.display = 'block';
    } else {
      loadMoreBtn.style.display = 'none';
    }
    
  } catch (error) {
    console.error('Error loading plan history:', error);
    if (reset) {
      historyList.innerHTML = '<div class="loading">Error loading plan history</div>';
    }
  } finally {
    planHistoryLoading = false;
  }
}

function createPlanHistoryItem(plan) {
  const item = document.createElement('div');
  item.className = 'plan-history-item';
  
  const statusIcon = getStatusIcon(plan.status);
  const timeAgo = formatTimeAgo(new Date(plan.created_at));
  const duration = plan.duration ? formatDuration(plan.duration) : 'N/A';
  
  item.innerHTML = `
    <div class="plan-item-header">
      <span class="plan-icon">${statusIcon}</span>
      <div class="plan-item-content">
        <div class="plan-description">${escapeHtml(plan.description)}</div>
        <div class="plan-metadata">
          <span class="plan-status-badge ${plan.status}">${plan.status}</span>
          <span class="plan-step-count">📋 ${plan.step_count} steps</span>
          <span class="plan-time">⏱️ ${timeAgo}</span>
          ${plan.duration ? `<span class="plan-duration">⏳ ${duration}</span>` : ''}
        </div>
        <div class="plan-actions">
          <button class="plan-action-btn" onclick="viewPlanDetails('${plan.id}')">View Details</button>
          <button class="plan-action-btn" onclick="rerunPlan('${plan.id}')">Re-run</button>
          <button class="plan-action-btn" onclick="deletePlan('${plan.id}')">Delete</button>
        </div>
      </div>
    </div>
  `;
  
  return item;
}

async function viewPlanDetails(planId) {
  try {
    const response = await fetch(`/api/plan/${planId}/full`);
    if (!response.ok) {
      throw new Error('Failed to load plan details');
    }
    
    const data = await response.json();
    showPlanDetailsModal(data);
  } catch (error) {
    console.error('Error loading plan details:', error);
    alert('Failed to load plan details');
  }
}

function showPlanDetailsModal(data) {
  const modal = document.getElementById('plan-details-modal');
  const content = document.getElementById('plan-details-content');
  
  const plan = data.plan;
  const stats = data.stats || {};
  const metrics = data.metrics || {};
  
  content.innerHTML = `
    <div class="plan-detail-section">
      <h4>Plan Overview</h4>
      <p><strong>Description:</strong> ${escapeHtml(plan.description)}</p>
      <p><strong>Status:</strong> <span class="plan-status-badge ${plan.status}">${plan.status}</span></p>
      <p><strong>Created:</strong> ${new Date(plan.created_at).toLocaleString()}</p>
      ${plan.completed_at ? `<p><strong>Completed:</strong> ${new Date(plan.completed_at).toLocaleString()}</p>` : ''}
    </div>
    
    <div class="plan-detail-section">
      <h4>Execution Statistics</h4>
      <div class="plan-metrics">
        <div class="metric-card">
          <div class="metric-value">${stats.execution_count || 0}</div>
          <div class="metric-label">Executions</div>
        </div>
        <div class="metric-card">
          <div class="metric-value">${Math.round(stats.success_rate || 0)}%</div>
          <div class="metric-label">Success Rate</div>
        </div>
        <div class="metric-card">
          <div class="metric-value">${formatDuration(stats.total_duration * 1000 || 0)}</div>
          <div class="metric-label">Total Time</div>
        </div>
        <div class="metric-card">
          <div class="metric-value">${plan.steps.length}</div>
          <div class="metric-label">Total Steps</div>
        </div>
      </div>
    </div>
    
    <div class="plan-detail-section">
      <h4>Steps</h4>
      <div class="plan-steps-detailed">
        ${plan.steps.map((step, index) => `
          <div class="plan-step-detailed ${step.status}">
            <div class="step-header">
              <span class="step-name">Step ${index + 1}: ${escapeHtml(step.description)}</span>
              <span class="step-status">${step.status}</span>
            </div>
            <div class="step-details">
              <strong>Tool:</strong> ${step.tool}<br>
              ${step.error ? `<strong>Error:</strong> ${escapeHtml(step.error)}` : ''}
            </div>
          </div>
        `).join('')}
      </div>
    </div>
    
    ${data.modified_files && data.modified_files.length > 0 ? `
      <div class="plan-detail-section">
        <h4>Modified Files</h4>
        <ul>
          ${data.modified_files.map(file => `<li>${escapeHtml(file)}</li>`).join('')}
        </ul>
      </div>
    ` : ''}
    
    ${data.git_operations && data.git_operations.length > 0 ? `
      <div class="plan-detail-section">
        <h4>Git Operations</h4>
        <ul>
          ${data.git_operations.map(op => `<li>${op.tool}: ${op.status}</li>`).join('')}
        </ul>
      </div>
    ` : ''}
  `;
  
  modal.classList.add('open');
}

function closePlanDetailsModal() {
  const modal = document.getElementById('plan-details-modal');
  modal.classList.remove('open');
}

async function rerunPlan(planId) {
  if (!confirm('Are you sure you want to re-run this plan?')) {
    return;
  }
  
  try {
    const response = await fetch(`/api/plan/${planId}/clone`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });
    
    if (!response.ok) {
      throw new Error('Failed to clone plan');
    }
    
    const newPlan = await response.json();
    
    // Close history panel
    document.getElementById('plan-history-panel').classList.remove('open');
    
    // Load and display the new plan
    currentPlanId = newPlan.id;
    await loadPlanDetails(newPlan.id);
    showPlanExecutionArea();
    
    // Optionally auto-execute
    if (confirm('Execute the cloned plan now?')) {
      executePlan();
    }
    
  } catch (error) {
    console.error('Error re-running plan:', error);
    alert('Failed to re-run plan');
  }
}

async function deletePlan(planId) {
  if (!confirm('Are you sure you want to delete this plan? This action cannot be undone.')) {
    return;
  }
  
  try {
    const response = await fetch(`/api/plan/${planId}`, {
      method: 'DELETE'
    });
    
    if (!response.ok) {
      throw new Error('Failed to delete plan');
    }
    
    // Reload the history
    planHistoryPage = 1;
    loadPlanHistory(true);
    
  } catch (error) {
    console.error('Error deleting plan:', error);
    alert('Failed to delete plan');
  }
}

function getStatusIcon(status) {
  switch (status) {
    case 'completed': return '✅';
    case 'failed': return '❌';
    case 'executing': return '⏳';
    case 'pending': return '⏸️';
    default: return '📋';
  }
}

function formatTimeAgo(date) {
  const seconds = Math.floor((new Date() - date) / 1000);
  
  if (seconds < 60) return 'just now';
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;
  
  return date.toLocaleDateString();
}

function formatDuration(ms) {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  if (ms < 3600000) return `${Math.floor(ms / 60000)}m ${Math.floor((ms % 60000) / 1000)}s`;
  
  const hours = Math.floor(ms / 3600000);
  const minutes = Math.floor((ms % 3600000) / 60000);
  return `${hours}h ${minutes}m`;
}

function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// Helper function to close plan details modal
function closePlanDetailsModal() {
  const modal = document.getElementById('plan-details-modal');
  modal.classList.remove('open');
}

// Load plan details and prepare for execution
async function loadPlanDetails(planId) {
  try {
    const response = await fetch(`/api/plan/${planId}/full`);
    if (!response.ok) {
      throw new Error(`Failed to load plan: ${response.statusText}`);
    }
    
    const data = await response.json();
    
    // Extract the plan object from the response
    const plan = data.plan;
    
    // Update currentPlan to reference the loaded plan
    currentPlan = plan;
    
    // Display the plan in the execution area
    displayPlan(plan);
    
    return plan;
  } catch (error) {
    console.error('Error loading plan details:', error);
    alert('Failed to load plan details. Please try again.');
    throw error;
  }
}

// Show the plan execution area
function showPlanExecutionArea() {
  const planExecutionArea = document.getElementById('plan-execution-area');
  if (planExecutionArea) {
    planExecutionArea.style.display = 'flex';
    document.body.classList.add('plan-executing');
  }
}

// Tool management functions
async function loadSessionTools(sessionId) {
  const toolsList = document.getElementById('tools-list');
  if (!toolsList) return;
  
  // Check if sessionId is provided
  if (!sessionId) {
    toolsList.innerHTML = '<div class="empty-state">Please select a session to view and manage tool permissions.</div>';
    return;
  }
  
  // Show loading state
  toolsList.innerHTML = '<div class="loading">Loading tools...</div>';
  
  try {
    const response = await fetch(`/api/session/${sessionId}/tools`);
    if (!response.ok) throw new Error('Failed to load tools');
    
    const tools = await response.json();
    
    // Group tools by category
    const toolsByCategory = {};
    tools.forEach(tool => {
      if (!toolsByCategory[tool.category]) {
        toolsByCategory[tool.category] = [];
      }
      toolsByCategory[tool.category].push(tool);
    });
    
    // Render tools grouped by category
    let html = '';
    Object.entries(toolsByCategory).forEach(([category, categoryTools]) => {
      html += `
        <div class="tool-category">
          <div class="tool-category-header">${category}</div>
          <div class="tool-items">
            ${categoryTools.map(tool => renderTool(tool)).join('')}
          </div>
        </div>
      `;
    });
    
    toolsList.innerHTML = html;
    
    // Add event listeners for tool controls
    toolsList.querySelectorAll('.tool-toggle input').forEach(toggle => {
      toggle.addEventListener('change', handleToolToggle);
    });
    
    toolsList.querySelectorAll('.mode-radio input').forEach(radio => {
      radio.addEventListener('change', handleModeChange);
    });
    
  } catch (error) {
    console.error('Error loading tools:', error);
    toolsList.innerHTML = '<div class="error">Failed to load tools</div>';
  }
}

function renderTool(tool) {
  const toolId = `tool-${tool.name}`;
  const modeGroupName = `mode-${tool.name}`;
  
  return `
    <div class="tool-item ${!tool.enabled ? 'disabled' : ''}" data-tool="${tool.name}">
      <div class="tool-info">
        <div class="tool-name">${tool.name}</div>
        <div class="tool-description">${escapeHtml(tool.description)}</div>
      </div>
      <div class="tool-controls">
        <label class="tool-toggle">
          <input type="checkbox" id="${toolId}" data-tool="${tool.name}" ${tool.enabled ? 'checked' : ''}>
          <span class="toggle-slider"></span>
        </label>
        <div class="tool-mode" ${!tool.enabled ? 'style="opacity: 0.3; pointer-events: none;"' : ''}>
          <div class="mode-radio">
            <input type="radio" id="${toolId}-ask" name="${modeGroupName}" value="ask" 
                   data-tool="${tool.name}" ${tool.mode === 'ask' ? 'checked' : ''}>
            <label for="${toolId}-ask">Ask</label>
          </div>
          <div class="mode-radio">
            <input type="radio" id="${toolId}-auto" name="${modeGroupName}" value="auto" 
                   data-tool="${tool.name}" ${tool.mode === 'auto' ? 'checked' : ''}>
            <label for="${toolId}-auto">Auto</label>
          </div>
        </div>
      </div>
    </div>
  `;
}

async function handleToolToggle(event) {
  const toggle = event.target;
  const toolName = toggle.dataset.tool;
  const enabled = toggle.checked;
  const toolItem = toggle.closest('.tool-item');
  const modeControls = toolItem.querySelector('.tool-mode');
  
  // Update UI immediately
  toolItem.classList.toggle('disabled', !enabled);
  if (modeControls) {
    modeControls.style.opacity = enabled ? '1' : '0.3';
    modeControls.style.pointerEvents = enabled ? 'auto' : 'none';
  }
  
  // Get current mode
  const modeRadio = toolItem.querySelector('.mode-radio input:checked');
  const mode = modeRadio ? modeRadio.value : 'ask';
  
  // Update on server
  try {
    const response = await fetch(`/api/session/${currentSessionId}/tools/${toolName}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ enabled, mode })
    });
    
    if (!response.ok) {
      throw new Error('Failed to update tool permission');
    }
    
    console.log(`Tool ${toolName} ${enabled ? 'enabled' : 'disabled'}`);
  } catch (error) {
    console.error('Error updating tool:', error);
    // Revert UI change
    toggle.checked = !enabled;
    toolItem.classList.toggle('disabled', enabled);
    if (modeControls) {
      modeControls.style.opacity = !enabled ? '1' : '0.3';
      modeControls.style.pointerEvents = !enabled ? 'auto' : 'none';
    }
    alert('Failed to update tool permission. Please try again.');
  }
}

async function handleModeChange(event) {
  const radio = event.target;
  const toolName = radio.dataset.tool;
  const mode = radio.value;
  const toolItem = radio.closest('.tool-item');
  const enableToggle = toolItem.querySelector('.tool-toggle input');
  const enabled = enableToggle.checked;
  
  // Update on server
  try {
    const response = await fetch(`/api/session/${currentSessionId}/tools/${toolName}`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ enabled, mode })
    });
    
    if (!response.ok) {
      throw new Error('Failed to update tool mode');
    }
    
    console.log(`Tool ${toolName} mode changed to ${mode}`);
  } catch (error) {
    console.error('Error updating tool mode:', error);
    // Revert to previous selection
    const otherRadio = toolItem.querySelector(`.mode-radio input[value="${mode === 'ask' ? 'auto' : 'ask'}"]`);
    if (otherRadio) otherRadio.checked = true;
    alert('Failed to update tool mode. Please try again.');
  }
}

// Escape HTML to prevent XSS
function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// Export the loadSessionTools function to window so it can be called from fileExplorer.js
window.loadSessionTools = loadSessionTools;

// Active tool executions tracker
const activeToolExecutions = new Map();

// Function to update working indicator
function updateWorkingIndicator() {
  const toolCount = activeToolExecutions.size;
  let indicator = document.getElementById('working-indicator');
  
  if (toolCount > 0) {
    // Create indicator if it doesn't exist
    if (!indicator) {
      indicator = document.createElement('div');
      indicator.id = 'working-indicator';
      indicator.className = 'working-indicator';
      document.body.appendChild(indicator);
    }
    
    // Update content with animated gear and count
    indicator.innerHTML = `
      <span class="working-text">Working</span>
      <span class="gear-icon rotating">⚙️</span>
      <span class="tool-count">(${toolCount} tool${toolCount !== 1 ? 's' : ''})</span>
    `;
    indicator.style.display = 'flex';
  } else {
    // Hide indicator when no tools are running
    if (indicator) {
      indicator.style.display = 'none';
    }
  }
}

// Handle tool execution start event
function handleToolExecutionStart(data) {
  const messagesContainer = document.getElementById('messages');
  
  // Clear any pending thinking return timer when new tool starts
  if (thinkingReturnTimer) {
    clearTimeout(thinkingReturnTimer);
    thinkingReturnTimer = null;
  }
  
  // Transform thinking indicator into tool execution display instead of removing it
  const thinkingIndicator = messagesContainer.querySelector('.message.thinking');
  if (thinkingIndicator) {
    // Keep the indicator but change its content to show tool execution
    const content = thinkingIndicator.querySelector('.message-content');
    if (content) {
      content.innerHTML = '<span class="tool-executing">🛠️ Executing tools...</span>';
    }
  }
  
  // Find or create the tool execution container
  let toolsContainer = document.querySelector('.tool-execution-container.active');
  if (!toolsContainer) {
    toolsContainer = document.createElement('div');
    toolsContainer.className = 'tool-execution-container active';
    toolsContainer.innerHTML = `
      <div class="tool-execution-header">
        <span class="tool-icon">🛠️</span>
        <span class="tool-title">Executing tools...</span>
        <button class="tool-toggle" onclick="toggleToolDetails(this)">▼</button>
      </div>
      <div class="tool-execution-list"></div>
    `;
    messagesContainer.appendChild(toolsContainer);
  }
  
  // Add tool to the execution list
  const toolsList = toolsContainer.querySelector('.tool-execution-list');
  const toolItem = document.createElement('div');
  toolItem.className = 'tool-item executing';
  toolItem.id = `tool-${data.toolId}`;
  toolItem.innerHTML = `
    <span class="tool-status-icon">⏳</span>
    <span class="tool-name">${data.toolName}</span>
    <div class="tool-progress" style="display: none;">
      <div class="tool-progress-bar" style="width: 0%"></div>
    </div>
    <span class="tool-metrics"></span>
  `;
  
  toolsList.appendChild(toolItem);
  
  // Track the active tool execution
  activeToolExecutions.set(data.toolId, {
    name: data.toolName,
    startTime: data.startTime,
    element: toolItem
  });
  
  // Update working indicator
  updateWorkingIndicator();
  
  // Scroll to bottom
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

// Handle tool execution progress event
function handleToolExecutionProgress(data) {
  const toolInfo = activeToolExecutions.get(data.toolId);
  if (!toolInfo) return;
  
  const toolItem = toolInfo.element;
  const progressContainer = toolItem.querySelector('.tool-progress');
  const progressBar = toolItem.querySelector('.tool-progress-bar');
  
  // Show progress bar
  progressContainer.style.display = 'block';
  progressBar.style.width = `${data.progress}%`;
  
  // Update metrics if provided
  if (data.message) {
    const metricsSpan = toolItem.querySelector('.tool-metrics');
    metricsSpan.textContent = data.message;
  }
}

// Handle tool execution complete event
function handleToolExecutionComplete(data) {
  // Clear any existing thinking return timer
  if (thinkingReturnTimer) {
    clearTimeout(thinkingReturnTimer);
    thinkingReturnTimer = null;
  }
  
  const toolInfo = activeToolExecutions.get(data.toolId);
  if (!toolInfo) return;
  
  const toolItem = toolInfo.element;
  const statusIcon = toolItem.querySelector('.tool-status-icon');
  const metricsSpan = toolItem.querySelector('.tool-metrics');
  const progressContainer = toolItem.querySelector('.tool-progress');
  
  // Update status
  toolItem.classList.remove('executing');
  toolItem.classList.add(data.status);
  
  // Update icon based on status
  if (data.status === 'success') {
    statusIcon.textContent = '✓';
  } else if (data.status === 'failed') {
    statusIcon.textContent = '❌';
  }
  
  // Hide progress bar
  progressContainer.style.display = 'none';
  
  // Update summary/metrics
  if (data.summary) {
    metricsSpan.textContent = data.summary.replace(/^[✓❌]\s*/, ''); // Remove status icon from summary
  } else if (data.metrics && data.metrics.duration) {
    metricsSpan.textContent = `(${data.metrics.duration}ms)`;
  }
  
  // Remove from active executions
  activeToolExecutions.delete(data.toolId);
  
  // Update working indicator
  updateWorkingIndicator();
  
  // If no more active tools, handle state transition
  if (activeToolExecutions.size === 0) {
    const toolsContainer = document.querySelector('.tool-execution-container.active');
    if (toolsContainer) {
      // Remove the active class to hide it visually
      toolsContainer.classList.remove('active');
      // Optionally remove the container entirely after a short delay
      // to allow the last tool animation to complete
      setTimeout(() => {
        if (toolsContainer.parentNode) {
          toolsContainer.remove();
        }
      }, 500);
    }
    
    // If LLM is still processing, return to thinking state after delay
    if (isLLMProcessing) {
      console.log('All tools complete, scheduling return to thinking state');
      thinkingReturnTimer = setTimeout(() => {
        // Only show thinking if LLM is still processing and no new tools/content started
        if (isLLMProcessing && activeToolExecutions.size === 0 && !currentStreamingMessageDiv) {
          console.log('Returning to thinking state after tool completion');
          const messagesContainer = document.getElementById('messages');
          
          // Check if thinking indicator already exists
          let thinkingIndicator = messagesContainer.querySelector('.message.thinking');
          if (!thinkingIndicator) {
            // Create new thinking indicator
            const thinkingId = 'thinking-return-' + Date.now();
            addThinkingIndicator(thinkingId);
          } else {
            // Update existing thinking indicator back to thinking state
            const content = thinkingIndicator.querySelector('.message-content');
            if (content) {
              content.innerHTML = '<span class="thinking-dots">Thinking<span>.</span><span>.</span><span>.</span></span>';
            }
          }
        }
        thinkingReturnTimer = null;
      }, THINKING_RETURN_DELAY);
    } else {
      // LLM not processing, remove any remaining thinking indicators
      const thinkingIndicators = document.querySelectorAll('.message.thinking');
      thinkingIndicators.forEach(indicator => indicator.remove());
    }
  }
}

// Toggle tool execution details visibility
function toggleToolDetails(button) {
  const container = button.closest('.tool-execution-container');
  const list = container.querySelector('.tool-execution-list');
  
  if (list.style.display === 'none') {
    list.style.display = 'block';
    button.textContent = '▼';
  } else {
    list.style.display = 'none';
    button.textContent = '▶';
  }
}

// Permission Request Handling
const activePermissionRequests = new Map();

// Handle incoming permission request
function handlePermissionRequest(data) {
  console.error('HANDLE PERMISSION REQUEST CALLED:', data);
  
  // Store the request
  activePermissionRequests.set(data.requestId, data);
  
  // Show the permission modal
  showPermissionModal(data);
}

// Show permission modal dialog
function showPermissionModal(data) {
  const modal = document.getElementById('permission-modal');
  const toolNameElement = document.getElementById('permission-tool-name');
  const paramsElement = document.getElementById('permission-params');
  const rememberCheckbox = document.getElementById('permission-remember');
  
  // Set tool name
  toolNameElement.textContent = data.toolName;
  
  // Display parameters
  paramsElement.innerHTML = '';
  if (data.parameterDisplay) {
    const paramDiv = document.createElement('div');
    paramDiv.className = 'param-display';
    paramDiv.textContent = data.parameterDisplay;
    paramsElement.appendChild(paramDiv);
  } else {
    // Fallback to showing raw parameters
    const paramList = document.createElement('ul');
    const params = data.parameters || {};
    const paramKeys = Object.keys(params).filter(k => !k.startsWith('_'));
    
    if (paramKeys.length === 0) {
      // No parameters to display
      const li = document.createElement('li');
      li.innerHTML = '<em>No parameters provided (this might be an error)</em>';
      li.style.color = 'var(--warning)';
      paramList.appendChild(li);
    } else {
      for (const key of paramKeys) {
        const li = document.createElement('li');
        const value = params[key];
        // Truncate long values for display
        let displayValue = JSON.stringify(value);
        if (displayValue.length > 100) {
          displayValue = displayValue.substring(0, 100) + '...';
        }
        li.innerHTML = `<strong>${key}:</strong> ${displayValue}`;
        paramList.appendChild(li);
      }
    }
    paramsElement.appendChild(paramList);
  }
  
  // Handle diff preview if available
  const diffSection = document.getElementById('permission-diff-section');
  const diffToggle = document.getElementById('permission-diff-toggle');
  const diffContainer = document.getElementById('permission-diff-container');
  const diffContent = document.getElementById('permission-diff-content');
  const diffStats = document.getElementById('permission-diff-stats');
  
  if (data.diffPreview && (data.toolName === 'write_file' || data.toolName === 'edit_file' || data.toolName === 'smart_edit')) {
    // Show diff section
    diffSection.style.display = 'block';
    
    // Set diff stats
    const stats = data.diffPreview.stats;
    if (stats) {
      diffStats.textContent = `(+${stats.added || 0}, -${stats.deleted || 0} lines)`;
    }
    
    // Render diff content
    renderPermissionDiff(diffContent, data.diffPreview);
    
    // Set up toggle handler
    const toggleIcon = diffToggle.querySelector('.toggle-icon');
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
    
    // Automatically expand diff for file write operations
    diffContainer.style.display = 'block';
    diffToggle.classList.add('expanded');
    toggleIcon.textContent = '▼';
  } else {
    // Hide diff section
    diffSection.style.display = 'none';
  }
  
  // Reset checkbox
  rememberCheckbox.checked = false;
  
  // Set up button handlers
  const approveBtn = document.getElementById('permission-approve');
  const denyBtn = document.getElementById('permission-deny');
  const abortBtn = document.getElementById('permission-abort');
  
  // Remove old handlers
  const newApproveBtn = approveBtn.cloneNode(true);
  const newDenyBtn = denyBtn.cloneNode(true);
  const newAbortBtn = abortBtn.cloneNode(true);
  approveBtn.parentNode.replaceChild(newApproveBtn, approveBtn);
  denyBtn.parentNode.replaceChild(newDenyBtn, denyBtn);
  abortBtn.parentNode.replaceChild(newAbortBtn, abortBtn);
  
  // Add new handlers
  newApproveBtn.addEventListener('click', () => {
    handlePermissionResponse(data.requestId, true);
  });
  
  newDenyBtn.addEventListener('click', () => {
    handlePermissionResponse(data.requestId, false);
  });
  
  newAbortBtn.addEventListener('click', () => {
    handlePermissionAbort(data.requestId);
  });
  
  // Show modal
  modal.style.display = 'block';
}

// Render diff content in permission modal
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

// Handle permission response
async function handlePermissionResponse(requestId, approved) {
  const request = activePermissionRequests.get(requestId);
  if (!request) return;
  
  const rememberCheckbox = document.getElementById('permission-remember');
  const remember = rememberCheckbox.checked;
  
  // Hide modal
  document.getElementById('permission-modal').style.display = 'none';
  
  // Remove from active requests
  activePermissionRequests.delete(requestId);
  
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
}

// Handle permission abort - completely stop the current operation
async function handlePermissionAbort(requestId) {
  const request = activePermissionRequests.get(requestId);
  if (!request) return;
  
  // Hide modal
  document.getElementById('permission-modal').style.display = 'none';
  
  // Remove from active requests
  activePermissionRequests.delete(requestId);
  
  // Send abort signal to backend
  try {
    const response = await fetch('/api/permission-abort', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        requestId: requestId,
        sessionId: window.currentSessionId
      })
    });
    
    if (!response.ok) {
      console.error('Failed to send permission abort:', response.status);
    }
    
    // Send abort message to the LLM
    await sendMessage('Important: ABORT!');
    
    // Show notification that the operation was aborted
    addSystemMessageToUI('🛑 Operation completely aborted by user', 'error');
  } catch (error) {
    console.error('Error sending permission abort:', error);
  }
}

// Display file diff in a closable frame
function displayFileDiff(data) {
  const { filePath, toolName, diff } = data;
  
  // Create or find the diff container
  let diffContainer = document.getElementById('file-diff-container');
  if (!diffContainer) {
    // Create the diff container
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
