/**
 * Events Module - Event bus for inter-module communication
 * Provides publish/subscribe pattern for decoupled module interaction
 */

// Create the event bus
function createEventBus() {
  const listeners = new Map();
  const oneTimeListeners = new Map();
  
  // Subscribe to an event
  function on(event, callback) {
    if (!listeners.has(event)) {
      listeners.set(event, new Set());
    }
    listeners.get(event).add(callback);
    
    // Return unsubscribe function
    return () => off(event, callback);
  }
  
  // Subscribe to an event once
  function once(event, callback) {
    if (!oneTimeListeners.has(event)) {
      oneTimeListeners.set(event, new Set());
    }
    oneTimeListeners.get(event).add(callback);
    
    // Return unsubscribe function
    return () => {
      if (oneTimeListeners.has(event)) {
        oneTimeListeners.get(event).delete(callback);
      }
    };
  }
  
  // Unsubscribe from an event
  function off(event, callback = null) {
    if (callback === null) {
      // Remove all listeners for this event
      listeners.delete(event);
      oneTimeListeners.delete(event);
    } else {
      // Remove specific callback
      if (listeners.has(event)) {
        listeners.get(event).delete(callback);
        if (listeners.get(event).size === 0) {
          listeners.delete(event);
        }
      }
      if (oneTimeListeners.has(event)) {
        oneTimeListeners.get(event).delete(callback);
        if (oneTimeListeners.get(event).size === 0) {
          oneTimeListeners.delete(event);
        }
      }
    }
  }
  
  // Emit an event
  function emit(event, data = null) {
    const results = [];
    
    // Call regular listeners
    if (listeners.has(event)) {
      listeners.get(event).forEach(callback => {
        try {
          const result = callback(data);
          results.push(result);
        } catch (error) {
          console.error(`Error in event listener for "${event}":`, error);
        }
      });
    }
    
    // Call one-time listeners and remove them
    if (oneTimeListeners.has(event)) {
      const callbacks = Array.from(oneTimeListeners.get(event));
      oneTimeListeners.delete(event);
      
      callbacks.forEach(callback => {
        try {
          const result = callback(data);
          results.push(result);
        } catch (error) {
          console.error(`Error in one-time event listener for "${event}":`, error);
        }
      });
    }
    
    return results;
  }
  
  // Wait for an event (returns a promise)
  function waitFor(event, timeout = null) {
    return new Promise((resolve, reject) => {
      let timeoutId = null;
      
      const cleanup = () => {
        if (timeoutId) {
          clearTimeout(timeoutId);
        }
      };
      
      // Set up one-time listener
      const unsubscribe = once(event, (data) => {
        cleanup();
        resolve(data);
      });
      
      // Set up timeout if provided
      if (timeout) {
        timeoutId = setTimeout(() => {
          unsubscribe();
          reject(new Error(`Timeout waiting for event "${event}"`));
        }, timeout);
      }
    });
  }
  
  // Get all registered events
  function getEvents() {
    const allEvents = new Set([
      ...listeners.keys(),
      ...oneTimeListeners.keys()
    ]);
    return Array.from(allEvents);
  }
  
  // Get listener count for an event
  function listenerCount(event) {
    let count = 0;
    if (listeners.has(event)) {
      count += listeners.get(event).size;
    }
    if (oneTimeListeners.has(event)) {
      count += oneTimeListeners.get(event).size;
    }
    return count;
  }
  
  // Clear all listeners
  function clear() {
    listeners.clear();
    oneTimeListeners.clear();
  }
  
  return {
    on,
    once,
    off,
    emit,
    waitFor,
    getEvents,
    listenerCount,
    clear
  };
}

// Create global event buses
const appEvents = createEventBus();
const stateEvents = createEventBus();
const sseEvents = createEventBus();
const uiEvents = createEventBus();

// Define standard events
const StandardEvents = {
  // Application lifecycle
  APP_INIT: 'app:init',
  APP_READY: 'app:ready',
  APP_ERROR: 'app:error',
  
  // Session events
  SESSION_CREATE: 'session:create',
  SESSION_CREATED: 'session:created',
  SESSION_DELETE: 'session:delete',
  SESSION_DELETED: 'session:deleted',
  SESSION_SWITCH: 'session:switch',
  SESSION_ERROR: 'session:error',
  
  // Message events
  MESSAGE_SEND: 'message:send',
  MESSAGE_SENT: 'message:sent',
  MESSAGE_RECEIVE: 'message:receive',
  MESSAGE_STREAM_START: 'message:stream:start',
  MESSAGE_STREAM_CHUNK: 'message:stream:chunk',
  MESSAGE_STREAM_END: 'message:stream:end',
  MESSAGE_ERROR: 'message:error',
  
  // Tool events
  TOOL_START: 'tool:start',
  TOOL_PROGRESS: 'tool:progress',
  TOOL_COMPLETE: 'tool:complete',
  TOOL_ERROR: 'tool:error',
  
  // Permission events
  PERMISSION_REQUEST: 'permission:request',
  PERMISSION_GRANTED: 'permission:granted',
  PERMISSION_DENIED: 'permission:denied',
  
  // Connection events
  CONNECTION_OPEN: 'connection:open',
  CONNECTION_CLOSE: 'connection:close',
  CONNECTION_ERROR: 'connection:error',
  CONNECTION_RECONNECT: 'connection:reconnect',
  
  // UI events
  UI_RESIZE: 'ui:resize',
  UI_THEME_CHANGE: 'ui:theme:change',
  UI_MODAL_OPEN: 'ui:modal:open',
  UI_MODAL_CLOSE: 'ui:modal:close',
  
  // Plan events
  PLAN_START: 'plan:start',
  PLAN_UPDATE: 'plan:update',
  PLAN_COMPLETE: 'plan:complete',
  PLAN_ERROR: 'plan:error',
  PLAN_STEP_START: 'plan:step:start',
  PLAN_STEP_COMPLETE: 'plan:step:complete'
};

// Export to global scope
window.AppEvents = appEvents;
window.StateEvents = stateEvents;
window.SSEEvents = sseEvents;
window.UIEvents = uiEvents;
window.StandardEvents = StandardEvents;