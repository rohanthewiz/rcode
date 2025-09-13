// main.js - Main entry point for the modularized UI
// This module initializes and coordinates all other modules

import { state, setState } from './state.js';
import { connectEventSource, disconnectSSE } from './sse.js';
import { configureMarked } from './markdown.js';
import { setupClipboardHandling, setupDragAndDrop } from './clipboard.js';
import { switchToSession, createNewSession, loadSessions } from './session.js';
import { addMessage, addSystemMessageToUI, addThinkingIndicator } from './messages.js';
import { initializeUsagePanel } from './usage.js';
import { compactSession, updateCompactionStats, checkAutoCompaction } from './compaction.js';

// Store pastedImages globally for access
window.pastedImages = [];

// Initialize the application
async function initialize() {
  console.log('Initializing RCode UI...');
  
  // Configure marked.js for markdown rendering
  configureMarked();
  
  // Get message input element
  const messageInput = document.getElementById('message-input');
  setState('messageInput', messageInput);
  
  // Setup editor (Monaco or textarea)
  setupEditor();
  
  // Initialize SSE connection
  connectEventSource();
  
  // Load sessions
  await loadSessions();
  
  // Setup event listeners
  setupEventListeners();
  
  // Initialize usage panel
  initializeUsagePanel();
  
  // Check authentication status
  await checkAuthStatus();
  
  console.log('RCode UI initialized');
}

// Setup editor (Monaco or textarea fallback)
function setupEditor() {
  const editorContainer = document.getElementById('editor-container');
  const messageInput = document.getElementById('message-input');
  
  if (!editorContainer || !messageInput) {
    console.error('Editor container or message input not found');
    return;
  }
  
  // For now, use the textarea as the editor
  setState('editor', messageInput);
  
  // Setup clipboard handling
  setupClipboardHandling(messageInput, window.pastedImages);
  
  // Setup drag and drop
  setupDragAndDrop(messageInput, window.pastedImages);
  
  // Auto-resize textarea
  messageInput.addEventListener('input', () => {
    messageInput.style.height = 'auto';
    messageInput.style.height = messageInput.scrollHeight + 'px';
  });
}

// Setup event listeners
function setupEventListeners() {
  // Send button
  const sendBtn = document.getElementById('send-btn');
  if (sendBtn) {
    sendBtn.addEventListener('click', sendMessage);
  }
  
  // New session button
  const newSessionBtn = document.getElementById('new-session-btn');
  if (newSessionBtn) {
    newSessionBtn.addEventListener('click', createNewSession);
  }
  
  // Logout button
  const logoutBtn = document.getElementById('logout-btn');
  if (logoutBtn) {
    logoutBtn.addEventListener('click', logout);
  }
  
  // Message input - Enter to send
  const messageInput = state.messageInput;
  if (messageInput) {
    messageInput.addEventListener('keydown', (e) => {
      if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault();
        sendMessage();
      }
    });
  }
  
  // Plan mode switch
  const planModeSwitch = document.getElementById('plan-mode-switch');
  if (planModeSwitch) {
    planModeSwitch.addEventListener('change', (e) => {
      togglePlanMode(e.target.checked);
    });
  }
  
  // Plan history button
  const planHistoryBtn = document.getElementById('plan-history-btn');
  if (planHistoryBtn) {
    planHistoryBtn.addEventListener('click', showPlanHistory);
  }
  
  // Compact session button
  const compactBtn = document.getElementById('compact-session-btn');
  if (compactBtn) {
    compactBtn.addEventListener('click', () => compactSession());
  }
}

// Send message to server
async function sendMessage() {
  const messageInput = state.messageInput;
  if (!messageInput) return;
  
  const content = messageInput.value.trim();
  if (!content) return;
  
  // Check if we have a session
  if (!state.currentSessionId) {
    console.log('No session selected, creating new session');
    await createNewSession();
    if (!state.currentSessionId) {
      addSystemMessageToUI('Failed to create session', 'error');
      return;
    }
  }
  
  // Check if already processing
  if (state.isProcessing) {
    console.log('Already processing a request');
    return;
  }
  
  // Set processing state
  setState('isProcessing', true);
  setState('hasReceivedFirstResponse', false);
  window.toggleStopButton(true);
  
  // Clear input
  messageInput.value = '';
  messageInput.style.height = 'auto';
  
  // Add user message to UI
  addMessage('user', content);
  
  // Show thinking indicator
  addThinkingIndicator('current');
  
  // Create abort controller
  const controller = new AbortController();
  setState('currentRequestController', controller);
  
  try {
    // Prepare message with images if any
    let messageContent = content;
    const imageAttachments = [];
    
    // Check for pasted images
    if (window.pastedImages && window.pastedImages.length > 0) {
      // Replace pasted image references and collect actual images
      window.pastedImages.forEach(img => {
        const pattern = new RegExp(`!\\[[^\\]]*\\]\\(pasted:${img.filename}\\)`, 'g');
        if (content.match(pattern)) {
          // Remove the markdown image reference from text
          messageContent = messageContent.replace(pattern, '');
          
          // Add image to attachments
          imageAttachments.push({
            type: 'image',
            source: {
              type: 'base64',
              media_type: img.mimeType,
              data: img.data.split(',')[1] // Remove data:image/png;base64, prefix
            }
          });
        }
      });
      
      // Clear pasted images after sending
      window.pastedImages = [];
    }
    
    // Build content array
    const contentArray = [];
    if (messageContent.trim()) {
      contentArray.push({
        type: 'text',
        text: messageContent.trim()
      });
    }
    contentArray.push(...imageAttachments);
    
    // Send to server
    const response = await fetch(`/api/session/${state.currentSessionId}/message`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        content: contentArray.length === 1 && contentArray[0].type === 'text' 
          ? contentArray[0].text 
          : contentArray
      }),
      signal: controller.signal
    });
    
    if (!response.ok) {
      throw new Error(`Failed to send message: ${response.statusText}`);
    }
    
    // Response will be handled via SSE
    
  } catch (error) {
    if (error.name === 'AbortError') {
      console.log('Request was aborted');
      addSystemMessageToUI('Request cancelled', 'info');
    } else {
      console.error('Error sending message:', error);
      addSystemMessageToUI('Failed to send message', 'error');
    }
  } finally {
    // Reset state
    setState('isProcessing', false);
    setState('currentRequestController', null);
    window.toggleStopButton(false);
  }
}

// Check authentication status
async function checkAuthStatus() {
  try {
    const response = await fetch('/api/app');
    if (response.ok) {
      const data = await response.json();
      if (!data.authenticated) {
        console.log('Not authenticated');
        // Could show login prompt here
      }
    }
  } catch (error) {
    console.error('Error checking auth status:', error);
  }
}

// Logout
async function logout() {
  if (!confirm('Are you sure you want to logout?')) {
    return;
  }
  
  try {
    const response = await fetch('/auth/logout', {
      method: 'POST'
    });
    
    if (response.ok) {
      // Disconnect SSE
      disconnectSSE();
      
      // Redirect to login
      window.location.reload();
    } else {
      throw new Error('Logout failed');
    }
  } catch (error) {
    console.error('Error during logout:', error);
    addSystemMessageToUI('Failed to logout', 'error');
  }
}

// Toggle plan mode
function togglePlanMode(enabled) {
  const sendBtn = document.getElementById('send-btn');
  const createPlanBtn = document.getElementById('create-plan-btn');
  
  if (enabled) {
    console.log('Plan mode enabled');
    if (sendBtn) sendBtn.style.display = 'none';
    if (createPlanBtn) {
      createPlanBtn.style.display = 'inline-block';
    } else {
      // Create plan button if it doesn't exist
      const planBtn = document.createElement('button');
      planBtn.id = 'create-plan-btn';
      planBtn.className = 'btn-primary';
      planBtn.textContent = 'Create Plan';
      planBtn.onclick = createPlan;
      
      if (sendBtn) {
        sendBtn.parentNode.insertBefore(planBtn, sendBtn.nextSibling);
      }
    }
  } else {
    console.log('Plan mode disabled');
    if (sendBtn) sendBtn.style.display = 'inline-block';
    if (createPlanBtn) createPlanBtn.style.display = 'none';
  }
}

// Create plan (placeholder)
async function createPlan() {
  const messageInput = state.messageInput;
  if (!messageInput) return;
  
  const content = messageInput.value.trim();
  if (!content) return;
  
  console.log('Creating plan for:', content);
  addSystemMessageToUI('Plan creation not yet implemented', 'info');
}

// Show plan history (placeholder)
function showPlanHistory() {
  console.log('Showing plan history');
  addSystemMessageToUI('Plan history not yet implemented', 'info');
}

// Initialize when DOM is ready
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initialize);
} else {
  initialize();
}

// Export functions that need global access
window.sendMessage = sendMessage;
window.createPlan = createPlan;