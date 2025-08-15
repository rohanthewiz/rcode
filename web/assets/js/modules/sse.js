// sse.js - Server-Sent Events handling
// This module manages SSE connections, reconnection logic, and event handling

import { state, setState, getState } from './state.js';
import { handleServerEvent } from './events.js';
import { updateConnectionStatus, showConnectionError } from './ui-utils.js';

// Connect to SSE endpoint
export function connectEventSource() {
  // Don't connect if manually disconnected
  if (state.isManuallyDisconnected) {
    console.log('SSE connection manually disconnected, not reconnecting');
    return;
  }
  
  // Clean up any existing connection
  if (state.eventSource) {
    state.eventSource.close();
  }
  
  console.log('Connecting to SSE endpoint...');
  updateConnectionStatus('reconnecting');
  
  const eventSource = new EventSource('/events');
  setState('eventSource', eventSource);
  
  eventSource.onopen = function() {
    console.log('SSE connection established');
    setState('connectionStatus', 'connected');
    updateConnectionStatus('connected');
    setState('reconnectAttempts', 0);
    setState('reconnectDelay', 1000); // Reset delay
    
    // Clear any error messages
    const errorBanner = document.getElementById('connection-error');
    if (errorBanner) {
      errorBanner.remove();
    }
  };
  
  eventSource.onmessage = function(event) {
    try {
      const data = JSON.parse(event.data);
      handleServerEvent(data);
    } catch (error) {
      console.error('Error parsing SSE message:', error);
    }
  };
  
  eventSource.onerror = function(error) {
    console.error('SSE connection error:', error);
    eventSource.close();
    setState('eventSource', null);
    setState('connectionStatus', 'disconnected');
    updateConnectionStatus('disconnected');
    
    // Try to reconnect with exponential backoff
    reconnectSSE();
  };
  
  // Custom event listeners for specific event types
  eventSource.addEventListener('message_start', function(event) {
    const data = JSON.parse(event.data);
    handleServerEvent({ type: 'message_start', ...data });
  });
  
  eventSource.addEventListener('content_block_delta', function(event) {
    const data = JSON.parse(event.data);
    handleServerEvent({ type: 'content_block_delta', ...data });
  });
  
  eventSource.addEventListener('message_complete', function(event) {
    const data = JSON.parse(event.data);
    handleServerEvent({ type: 'message_complete', ...data });
  });
  
  eventSource.addEventListener('error', function(event) {
    const data = JSON.parse(event.data);
    handleServerEvent({ type: 'error', ...data });
  });
}

// Reconnect with exponential backoff
export function reconnectSSE() {
  // Don't reconnect if manually disconnected
  if (state.isManuallyDisconnected) {
    console.log('SSE manually disconnected, not reconnecting');
    return;
  }
  
  if (state.reconnectAttempts >= state.maxReconnectAttempts) {
    console.log('Max reconnection attempts reached');
    showConnectionError('Connection lost. Please refresh the page or click to reconnect.');
    return;
  }
  
  const attempts = state.reconnectAttempts + 1;
  setState('reconnectAttempts', attempts);
  
  console.log(`Reconnecting SSE in ${state.reconnectDelay}ms (attempt ${attempts}/${state.maxReconnectAttempts})...`);
  updateConnectionStatus('reconnecting');
  
  setTimeout(() => {
    connectEventSource();
  }, state.reconnectDelay);
  
  // Exponential backoff with max delay
  const newDelay = Math.min(state.reconnectDelay * 2, state.maxReconnectDelay);
  setState('reconnectDelay', newDelay);
}

// Disconnect SSE
export function disconnectSSE() {
  setState('isManuallyDisconnected', true);
  if (state.eventSource) {
    state.eventSource.close();
    setState('eventSource', null);
  }
  setState('connectionStatus', 'disconnected');
  updateConnectionStatus('disconnected');
}

// Manual reconnection
export function manualReconnect() {
  console.log('Manual reconnection triggered');
  setState('isManuallyDisconnected', false);
  setState('reconnectAttempts', 0);
  setState('reconnectDelay', 1000);
  connectEventSource();
}