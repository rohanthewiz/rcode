// events.js - Event handling
// This module handles server events and coordinates responses

import { state, setState } from './state.js';
import { 
  addMessage, 
  addSystemMessageToUI, 
  addThinkingIndicator, 
  removeThinkingIndicator,
  createStreamingMessage,
  appendToStreamingMessage,
  finalizeStreamingMessage
} from './messages.js';
import { 
  addToolUsageSummaryToUI,
  handleToolExecutionStart,
  handleToolExecutionProgress,
  handleToolExecutionComplete
} from './tools.js';
import { handlePermissionRequest } from './permissions.js';
import { handleUsageUpdateEvent } from './usage.js';
import { selectSession, loadSessions } from './session.js';

// Handle server events
export function handleServerEvent(event) {
  console.log('Server event:', event.type, event);
  
  // Filter events by session if we have a current session
  if (event.sessionId && state.currentSessionId && event.sessionId !== state.currentSessionId) {
    console.log('Event for different session, ignoring');
    return;
  }
  
  switch (event.type) {
    case 'session_created':
      loadSessions();
      if (event.sessionId) {
        selectSession(event.sessionId);
      }
      break;
      
    case 'session_deleted':
      loadSessions();
      break;
      
    case 'message_start':
      // Set flag to indicate we're receiving a response
      setState('hasReceivedFirstResponse', true);
      
      // Clear any existing thinking indicator
      removeThinkingIndicator('current');
      
      // Create streaming message container
      createStreamingMessage();
      
      // Mark LLM as processing
      setState('isLLMProcessing', true);
      setState('toolsAnnounced', false);
      break;
      
    case 'content_block_start':
      if (event.content_block?.type === 'text') {
        // Text content starting
        console.log('Text content block starting');
      } else if (event.content_block?.type === 'tool_use') {
        // Tool use starting
        console.log('Tool use starting:', event.content_block.name);
        setState('currentToolUseId', event.content_block.id);
        setState('currentToolName', event.content_block.name);
        setState('currentToolInput', '');
      }
      break;
      
    case 'content_block_delta':
      if (event.delta?.type === 'text_delta') {
        // Append text to streaming message
        appendToStreamingMessage(event.delta.text);
      } else if (event.delta?.type === 'input_json_delta') {
        // Accumulate tool input
        state.currentToolInput += event.delta.partial_json;
      }
      break;
      
    case 'content_block_stop':
      if (event.content_block?.type === 'tool_use' && state.currentToolUseId) {
        // Tool use completed, add summary
        try {
          const toolInput = state.currentToolInput ? JSON.parse(state.currentToolInput) : {};
          addToolUsageSummaryToUI({
            id: state.currentToolUseId,
            name: state.currentToolName,
            input: toolInput
          });
        } catch (e) {
          console.error('Error parsing tool input:', e);
        }
        
        // Reset tool state
        setState('currentToolUseId', null);
        setState('currentToolName', null);
        setState('currentToolInput', '');
      }
      break;
      
    case 'message_delta':
      // Handle stop reason or other message-level deltas
      if (event.delta?.stop_reason) {
        console.log('Message stop reason:', event.delta.stop_reason);
      }
      break;
      
    case 'message_stop':
      // Message completed
      finalizeStreamingMessage();
      
      // Mark LLM as no longer processing
      setState('isLLMProcessing', false);
      
      // Check if we should return to thinking state
      if (state.activeToolExecutions.size === 0 && !state.toolsAnnounced) {
        // No tools are running, remove thinking indicator
        removeThinkingIndicator('current');
      }
      break;
      
    case 'error':
      console.error('Server error:', event);
      
      // Check for specific error types
      if (event.error?.includes('session_not_found')) {
        // Session was deleted or doesn't exist
        console.log('Session not found, creating new session');
        setState('currentSessionId', null);
        loadSessions();
      } else {
        addSystemMessageToUI(event.error || 'An error occurred', 'error');
      }
      
      // Clean up any streaming state
      finalizeStreamingMessage();
      removeThinkingIndicator('current');
      
      // Reset processing state
      setState('isLLMProcessing', false);
      setState('isProcessing', false);
      toggleStopButton(false);
      break;
      
    case 'tool_execution_start':
      handleToolExecutionStart(event);
      break;
      
    case 'tool_execution_progress':
      handleToolExecutionProgress(event);
      break;
      
    case 'tool_execution_complete':
      handleToolExecutionComplete(event);
      
      // After tool completion, check if we should show thinking indicator
      if (state.activeToolExecutions.size === 0 && state.isLLMProcessing) {
        // All tools done but LLM still processing
        if (state.thinkingReturnTimer) {
          clearTimeout(state.thinkingReturnTimer);
        }
        
        // Set a timer to return to thinking state
        state.thinkingReturnTimer = setTimeout(() => {
          if (state.isLLMProcessing && state.activeToolExecutions.size === 0) {
            addThinkingIndicator('current');
          }
          state.thinkingReturnTimer = null;
        }, state.THINKING_RETURN_DELAY);
      }
      break;
      
    case 'permission_request':
      handlePermissionRequest(event);
      break;
      
    case 'usage_update':
      handleUsageUpdateEvent(event);
      break;
      
    default:
      console.log('Unhandled event type:', event.type);
  }
}

// Toggle stop button visibility
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

// Stop current operation
function stopCurrentOperation() {
  console.log('Stopping current operation...');
  
  // Abort the current request if any
  if (state.currentRequestController) {
    state.currentRequestController.abort();
    setState('currentRequestController', null);
  }
  
  // Reset UI state
  toggleStopButton(false);
  setState('isProcessing', false);
  setState('isLLMProcessing', false);
  setState('toolsAnnounced', false);
  
  // Clear any pending thinking return timer
  if (state.thinkingReturnTimer) {
    clearTimeout(state.thinkingReturnTimer);
    setState('thinkingReturnTimer', null);
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
  state.activeToolExecutions.clear();
}

// Export functions that need global access
window.toggleStopButton = toggleStopButton;
window.stopCurrentOperation = stopCurrentOperation;