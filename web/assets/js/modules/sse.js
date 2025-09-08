/**
 * SSE Module - Server-Sent Events management
 * Handles real-time communication with the server via SSE
 */

// SSE connection management
function connectEventSource() {
  // Check if AppState is available
  if (!window.AppState || !window.AppState.getState) {
    console.warn('AppState not available, using fallback SSE connection');
    // Fallback implementation
    connectEventSourceFallback();
    return;
  }
  
  const state = window.AppState.getState();
  
  // Don't reconnect if manually disconnected
  if (state.isManuallyDisconnected) {
    console.log('SSE connection skipped - manually disconnected');
    return;
  }

  // Close existing connection if any
  if (state.eventSource) {
    console.log('Closing existing EventSource before connecting');
    state.eventSource.close();
    window.AppState.setState('eventSource', null);
  }

  console.log(`Attempting SSE connection (attempt ${state.reconnectAttempts + 1})`);

  // Update status to reconnecting if we're retrying
  if (state.reconnectAttempts > 0 || state.connectionStatus === 'reconnecting') {
    updateConnectionStatus('reconnecting');
  }

  const eventSource = new EventSource('/events');
  window.AppState.setState('eventSource', eventSource);

  eventSource.onopen = function() {
    console.log('SSE connection established');
    // Reset reconnection state on successful connection
    window.AppState.setStateMultiple({
      reconnectAttempts: 0,
      reconnectDelay: 1000
    });
    updateConnectionStatus('connected');
    
    // Emit connection open event
    if (window.SSEEvents) {
      window.SSEEvents.emit(window.StandardEvents.CONNECTION_OPEN);
    }
    
    // Refresh sessions in case server was restarted
    if (window.loadSessions) {
      window.loadSessions();
    }
  };

  eventSource.onmessage = function(eventPayload) {
    const dataObj = JSON.parse(eventPayload.data);
    handleServerEvent(dataObj);
  };

  eventSource.onerror = function(error) {
    console.error('SSE error:', error);
    
    // Close the current connection
    const currentEventSource = window.AppState.getState('eventSource');
    if (currentEventSource) {
      currentEventSource.close();
      window.AppState.setState('eventSource', null);
    }

    // Increment attempts
    const attempts = window.AppState.incrementReconnectAttempts();
    const maxAttempts = window.AppState.getState('maxReconnectAttempts');

    // Check if we've exceeded max reconnection attempts
    if (attempts > maxAttempts) {
      console.error('Max reconnection attempts reached. Stopping auto-reconnect.');
      updateConnectionStatus('disconnected');
      showConnectionError('Connection to server lost. Please refresh the page or click reconnect.');
      
      // Emit connection error event
      if (window.SSEEvents) {
        window.SSEEvents.emit(window.StandardEvents.CONNECTION_ERROR, { 
          reason: 'max_attempts_exceeded' 
        });
      }
      return;
    }

    // Update status to show we're reconnecting with attempt count
    updateConnectionStatus('reconnecting');

    // Calculate next delay with exponential backoff
    const currentDelay = window.AppState.getState('reconnectDelay');
    const maxDelay = window.AppState.getState('maxReconnectDelay');
    const nextDelay = Math.min(currentDelay * 2, maxDelay);
    
    console.log(`Reconnecting in ${currentDelay/1000} seconds... (attempt ${attempts}/${maxAttempts})`);
    
    setTimeout(() => {
      connectEventSource();
    }, currentDelay);
    
    // Update delay for next attempt
    window.AppState.setState('reconnectDelay', nextDelay);
  };
}

// Manually reconnect SSE
function reconnectSSE() {
  console.log('Manual SSE reconnection requested');
  
  // Close any existing connection first
  const eventSource = window.AppState.getState('eventSource');
  if (eventSource) {
    console.log('Closing existing EventSource before reconnecting');
    eventSource.close();
    window.AppState.setState('eventSource', null);
  }
  
  // Reset connection state
  window.AppState.setStateMultiple({
    isManuallyDisconnected: false,
    reconnectAttempts: 0,
    reconnectDelay: 1000
  });
  
  // Update status to show we're starting fresh
  // Set attempts to 1 temporarily for display purposes
  const tempAttempts = window.AppState.getState('reconnectAttempts');
  window.AppState.setState('reconnectAttempts', 1);
  updateConnectionStatus('reconnecting');
  window.AppState.setState('reconnectAttempts', tempAttempts);
  
  // Small delay to ensure UI updates before connection attempt
  setTimeout(() => {
    connectEventSource();
  }, 100);
}

// Disconnect SSE
function disconnectSSE() {
  console.log('Disconnecting SSE');
  window.AppState.setState('isManuallyDisconnected', true);
  
  const eventSource = window.AppState.getState('eventSource');
  if (eventSource) {
    eventSource.close();
    window.AppState.setState('eventSource', null);
  }
  
  updateConnectionStatus('disconnected');
  
  // Emit connection close event
  if (window.SSEEvents) {
    window.SSEEvents.emit(window.StandardEvents.CONNECTION_CLOSE);
  }
}

// Update connection status in UI
function updateConnectionStatus(status) {
  window.AppState.setConnectionStatus(status);
  
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
  
  const reconnectAttempts = window.AppState.getState('reconnectAttempts');
  const maxReconnectAttempts = window.AppState.getState('maxReconnectAttempts');
  
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
        statusElement.innerHTML = 'Connection lost. <a href="#" onclick="SSEModule.reconnectSSE(); return false;">Reconnect</a>';
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
      <button onclick="SSEModule.reconnectSSE()" class="btn-secondary">Reconnect</button>
    </div>
  `;
  
  messagesContainer.appendChild(errorDiv);
  messagesContainer.scrollTop = messagesContainer.scrollHeight;
}

// Handle server events
function handleServerEvent(evtData) {
  // Get current session ID with fallback
  const currentSessionId = (window.AppState && window.AppState.getState) ? 
    window.AppState.getState('currentSessionId') : window.currentSessionId;
  
  console.log('Event sessionId:', evtData.sessionId, 'Current sessionId:', currentSessionId, 'Match:', evtData.sessionId === currentSessionId);
  
  // Special logging for permission events
  if (evtData.type === 'permission_request') {
    console.warn('PERMISSION EVENT RECEIVED:', {
      type: evtData.type,
      eventSessionId: evtData.sessionId,
      currentSessionId: currentSessionId,
      sessionMatch: evtData.sessionId === currentSessionId,
      data: evtData.data
    });
  }

  // Emit events for different event types
  if (window.SSEEvents) {
    window.SSEEvents.emit(evtData.type, evtData);
    
    // Also emit generic event
    window.SSEEvents.emit('server_event', evtData);
  }

  // Auto-switch to Files tab on first response
  const hasReceivedFirstResponse = window.AppState.getState('hasReceivedFirstResponse');
  if (evtData.sessionId === currentSessionId && !hasReceivedFirstResponse) {
    // Check for events that indicate the LLM is starting to respond
    if (evtData.type === 'content_start' ||
        evtData.type === 'tool_execution_start' ||
        (evtData.type === 'message_delta' && evtData.data && evtData.data.delta)) {
      
      // Switch to Files tab on first response
      if (window.FileExplorer && window.FileExplorer.switchTab) {
        console.log('Auto-switching to Files tab on first response');
        window.FileExplorer.switchTab('files');
        window.AppState.setState('hasReceivedFirstResponse', true);
      }
    }
  }

  // Handle specific event types
  if (evtData.sessionId === currentSessionId) {
    switch (evtData.type) {
      case 'message_start':
        handleMessageStart(evtData);
        break;
      case 'content_start':
        handleContentStart(evtData);
        break;
      case 'tool_use_start':
        handleToolUseStart(evtData);
        break;
      case 'message_delta':
        handleMessageDelta(evtData);
        break;
      case 'message_stop':
        handleMessageStop(evtData);
        break;
      case 'tool_execution_start':
        handleToolExecutionStart(evtData);
        break;
      case 'tool_execution_progress':
        handleToolExecutionProgress(evtData);
        break;
      case 'tool_execution_complete':
        handleToolExecutionComplete(evtData);
        break;
      case 'tool_usage':
        handleToolUsage(evtData);
        break;
      case 'permission_request':
        handlePermissionRequest(evtData);
        break;
      case 'plan_update':
        handlePlanUpdate(evtData);
        break;
      case 'plan_step_update':
        handlePlanStepUpdate(evtData);
        break;
      case 'plan_complete':
        handlePlanComplete(evtData);
        break;
      case 'usage_update':
        handleUsageUpdate(evtData);
        break;
      case 'error':
        handleErrorEvent(evtData);
        break;
    }
  }
}

// Event handlers
function handleMessageStart(evtData) {
  console.log('Message streaming started');
  if (window.AppState && window.AppState.setStateMultiple) {
    window.AppState.setStateMultiple({
      isLLMProcessing: true,
      toolsAnnounced: false
    });
  } else {
    // Fallback to global variables
    window.isLLMProcessing = true;
    window.toolsAnnounced = false;
  }
  
  // Clear any pending thinking return timer
  const timer = (window.AppState && window.AppState.getState) ?
    window.AppState.getState('thinkingReturnTimer') : window.thinkingReturnTimer;
  if (timer) {
    clearTimeout(timer);
    window.AppState.setState('thinkingReturnTimer', null);
  }
}

function handleContentStart(evtData) {
  const toolsAnnounced = window.AppState.getState('toolsAnnounced');
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
}

function handleToolUseStart(evtData) {
  console.log('Tool use started - tools announced');
  window.AppState.setState('toolsAnnounced', true);
  
  // Transform thinking indicator to show tools are coming
  const thinkingIndicator = document.querySelector('.message.thinking');
  if (thinkingIndicator) {
    const content = thinkingIndicator.querySelector('.message-content');
    if (content) {
      content.innerHTML = '<span class="tool-executing">🛠️ Executing tools...</span>';
    }
  }
}

function handleMessageDelta(evtData) {
  console.log('Message delta received:', evtData.data.delta);
  
  // Delegate to UI module functions
  if (window.createStreamingMessage && !window.AppState.getState('currentStreamingMessageDiv')) {
    window.createStreamingMessage();
  }
  if (window.appendToStreamingMessage) {
    window.appendToStreamingMessage(evtData.data.delta);
  }
}

function handleMessageStop(evtData) {
  console.log('Message streaming stopped');
  
  // Delegate to UI module functions
  if (window.finalizeStreamingMessage) {
    window.finalizeStreamingMessage();
  }
  if (window.toggleStopButton) {
    window.toggleStopButton(false);
  }
  
  window.AppState.setStateMultiple({
    isProcessing: false,
    isLLMProcessing: false,
    toolsAnnounced: false
  });
  
  // Update compaction stats after message is complete
  if (window.updateCompactionStats) {
    window.updateCompactionStats();
  }
  
  // Clear any pending thinking return timer
  const timer = window.AppState.getState('thinkingReturnTimer');
  if (timer) {
    clearTimeout(timer);
    window.AppState.setState('thinkingReturnTimer', null);
  }
}

// Delegate handlers - these will call into other modules
function handleToolExecutionStart(evtData) {
  if (window.handleToolExecutionStart) {
    window.handleToolExecutionStart(evtData);
  }
}

function handleToolExecutionProgress(evtData) {
  if (window.handleToolExecutionProgress) {
    window.handleToolExecutionProgress(evtData);
  }
}

function handleToolExecutionComplete(evtData) {
  if (window.handleToolExecutionComplete) {
    window.handleToolExecutionComplete(evtData);
  }
}

function handleToolUsage(evtData) {
  if (window.addToolUsageSummaryToUI) {
    window.addToolUsageSummaryToUI(evtData);
  }
}

function handlePermissionRequest(evtData) {
  if (window.handlePermissionRequest) {
    window.handlePermissionRequest(evtData);
  } else {
    console.error('handlePermissionRequest function not found in window');
  }
}

function handlePlanUpdate(evtData) {
  if (window.handlePlanUpdate) {
    window.handlePlanUpdate(evtData);
  }
}

function handlePlanStepUpdate(evtData) {
  if (window.handlePlanStepUpdate) {
    window.handlePlanStepUpdate(evtData);
  }
}

function handlePlanComplete(evtData) {
  if (window.handlePlanComplete) {
    window.handlePlanComplete(evtData);
  }
}

function handleUsageUpdate(evtData) {
  if (window.updateUsageDisplay) {
    window.updateUsageDisplay(evtData.data.usage);
  }
}

function handleErrorEvent(evtData) {
  console.error('Server error event:', evtData);
  if (window.showError) {
    window.showError(evtData.message || 'An error occurred');
  }
}

// Fallback implementation when AppState is not available
function connectEventSourceFallback() {
  console.log('Using fallback SSE connection');
  
  // Use global variables as fallback
  if (window.eventSource) {
    window.eventSource.close();
    window.eventSource = null;
  }
  
  window.eventSource = new EventSource('/events');
  
  window.eventSource.onopen = function() {
    console.log('SSE connection established (fallback)');
    if (window.loadSessions) {
      window.loadSessions();
    }
  };
  
  window.eventSource.onmessage = function(sseEvent) {
    try {
      const evtData = JSON.parse(sseEvent.data);
      if (window.handleServerEvent) {
        window.handleServerEvent(evtData);
      }
    } catch (e) {
      console.error('Error parsing SSE message:', e);
    }
  };
  
  window.eventSource.onerror = function(error) {
    console.error('SSE error (fallback):', error);
    if (window.eventSource) {
      window.eventSource.close();
      window.eventSource = null;
    }
    
    // Simple retry after 5 seconds
    setTimeout(() => {
      connectEventSourceFallback();
    }, 5000);
  };
}

// Export to global scope
window.SSEModule = {
  connectEventSource,
  reconnectSSE,
  disconnectSSE,
  updateConnectionStatus,
  showConnectionError,
  handleServerEvent
};