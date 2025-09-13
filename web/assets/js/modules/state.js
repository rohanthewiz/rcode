/**
 * State Module - Centralized state management for the application
 * Manages all global state variables and provides controlled access
 */

// Initialize the state object with all global variables
function createState() {
  return {
    // Session and connection state
    currentSessionId: null,
    eventSource: null,
    connectionStatus: 'disconnected', // 'connected', 'disconnected', 'reconnecting'
    isManuallyDisconnected: false,
    
    // Editor state
    editor: null,
    
    // Request processing state
    pendingNewSession: false,
    hasReceivedFirstResponse: false,
    currentRequestController: null,
    isProcessing: false,
    
    // LLM processing state
    isLLMProcessing: false,
    thinkingReturnTimer: null,
    toolsAnnounced: false,
    
    // Streaming message state
    currentStreamingMessageDiv: null,
    currentStreamingContent: '',
    
    // Plan mode state
    isPlanMode: false,
    currentPlan: null,
    currentPlanId: null,
    planSteps: new Map(),
    
    // Plan history state
    planHistoryPage: 1,
    planHistoryLoading: false,
    planHistorySearch: '',
    planHistoryStatus: '',
    
    // Tool and permission tracking
    activeToolExecutions: new Map(),
    activePermissionRequests: new Map(),
    
    // Reconnection state
    reconnectAttempts: 0,
    reconnectDelay: 1000,
    
    // Configuration constants
    fileRefreshDelay: 9000,
    maxReconnectAttempts: 5,
    maxReconnectDelay: 30000,
    THINKING_RETURN_DELAY: 2000
  };
}

// Create the global state instance
const state = createState();

// State getters
function getState(key) {
  if (key) {
    return state[key];
  }
  return state;
}

// State setters with validation
function setState(key, value) {
  if (!state.hasOwnProperty(key)) {
    console.warn(`State key "${key}" does not exist`);
    return false;
  }
  
  const oldValue = state[key];
  state[key] = value;
  
  // Trigger state change event for observers
  if (window.StateEvents && window.StateEvents.emit) {
    window.StateEvents.emit('stateChange', { key, oldValue, newValue: value });
  }
  
  return true;
}

// Batch state updates
function setStateMultiple(updates) {
  const changes = [];
  
  for (const [key, value] of Object.entries(updates)) {
    if (!state.hasOwnProperty(key)) {
      console.warn(`State key "${key}" does not exist`);
      continue;
    }
    
    const oldValue = state[key];
    state[key] = value;
    changes.push({ key, oldValue, newValue: value });
  }
  
  // Trigger batch state change event
  if (window.StateEvents && window.StateEvents.emit && changes.length > 0) {
    window.StateEvents.emit('batchStateChange', changes);
  }
  
  return changes.length > 0;
}

// Reset state to initial values
function resetState(keys = null) {
  const initialState = createState();
  
  if (keys && Array.isArray(keys)) {
    // Reset only specified keys
    keys.forEach(key => {
      if (state.hasOwnProperty(key)) {
        state[key] = initialState[key];
      }
    });
  } else {
    // Reset all state
    Object.keys(state).forEach(key => {
      state[key] = initialState[key];
    });
  }
  
  // Trigger reset event
  if (window.StateEvents && window.StateEvents.emit) {
    window.StateEvents.emit('stateReset', keys);
  }
}

// Helper functions for specific state operations
function incrementReconnectAttempts() {
  state.reconnectAttempts++;
  state.reconnectDelay = Math.min(state.reconnectDelay * 2, state.maxReconnectDelay);
  return state.reconnectAttempts;
}

function resetReconnectState() {
  state.reconnectAttempts = 0;
  state.reconnectDelay = 1000;
}

function setConnectionStatus(status) {
  if (!['connected', 'disconnected', 'reconnecting'].includes(status)) {
    console.warn(`Invalid connection status: ${status}`);
    return false;
  }
  
  const oldStatus = state.connectionStatus;
  state.connectionStatus = status;
  
  // Trigger connection status change event
  if (window.StateEvents && window.StateEvents.emit) {
    window.StateEvents.emit('connectionStatusChange', { oldStatus, newStatus: status });
  }
  
  return true;
}

// Export all functions to global scope using IIFE pattern
window.AppState = {
  getState,
  setState,
  setStateMultiple,
  resetState,
  incrementReconnectAttempts,
  resetReconnectState,
  setConnectionStatus
};