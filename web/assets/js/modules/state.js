// state.js - Global state management
// This module manages all global state variables used across the application

export const state = {
  currentSessionId: null,
  eventSource: null,
  messageInput: null,
  editor: null,
  pendingNewSession: false, // Track if we're waiting to create a new session
  hasReceivedFirstResponse: false, // Track first response per message
  fileRefreshDelay: 9000, // Warn: must be greater than the cacheTTL of the backend which is currently 7s
  currentRequestController: null, // AbortController for current request
  isProcessing: false, // Track if currently processing a request
  
  // SSE connection tracking
  reconnectAttempts: 0,
  reconnectDelay: 1000, // Start with 1 second
  maxReconnectAttempts: 5,
  maxReconnectDelay: 30000, // Max 30 seconds
  isManuallyDisconnected: false,
  connectionStatus: 'disconnected', // 'connected', 'disconnected', 'reconnecting'
  
  // LLM response state tracking
  isLLMProcessing: false, // Track if LLM is still processing
  thinkingReturnTimer: null, // Timer to return to thinking state after tool completion
  THINKING_RETURN_DELAY: 2000, // 2 seconds delay before returning to thinking
  toolsAnnounced: false, // Track if tools have been announced
  
  // Streaming message state
  currentStreamingMessage: null,
  currentStreamingContent: '',
  currentToolUseId: null,
  currentToolName: null,
  currentToolInput: '',
  
  // Tool execution tracking
  activeToolExecutions: new Map(), // Map of tool_id -> execution data
  
  // Plan mode state
  planMode: {
    currentPlan: null,
    isExecuting: false,
    isPaused: false,
    currentStepIndex: -1
  }
};

// Helper functions to update state
export function setState(key, value) {
  if (key in state) {
    state[key] = value;
  } else {
    console.warn(`Attempting to set unknown state key: ${key}`);
  }
}

export function getState(key) {
  return state[key];
}

export function resetState() {
  // Reset to initial values
  state.currentSessionId = null;
  state.eventSource = null;
  state.messageInput = null;
  state.editor = null;
  state.pendingNewSession = false;
  state.hasReceivedFirstResponse = false;
  state.currentRequestController = null;
  state.isProcessing = false;
  state.isLLMProcessing = false;
  state.thinkingReturnTimer = null;
  state.toolsAnnounced = false;
  state.currentStreamingMessage = null;
  state.currentStreamingContent = '';
  state.currentToolUseId = null;
  state.currentToolName = null;
  state.currentToolInput = '';
  state.activeToolExecutions.clear();
}