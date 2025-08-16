// session.js - Session management
// This module handles session creation, selection, and management

import { state, setState } from './state.js';
import { addSystemMessageToUI } from './messages.js';

// Select and load a session
export async function selectSession(sessionId) {
  console.log('Selecting session:', sessionId);
  setState('currentSessionId', sessionId);
  
  // Update UI to show selected session
  const sessionItems = document.querySelectorAll('.session-item');
  sessionItems.forEach(item => {
    if (item.dataset.sessionId === sessionId) {
      item.classList.add('active');
    } else {
      item.classList.remove('active');
    }
  });
  
  // Clear chat area
  const chatContainer = document.getElementById('chat-container');
  if (chatContainer) {
    chatContainer.innerHTML = '';
  }
  
  // Load session messages
  try {
    const response = await fetch(`/api/session/${sessionId}/messages`);
    if (!response.ok) {
      // If session not found, create a new one
      if (response.status === 404) {
        console.log('Session not found, creating new session');
        await createNewSession();
        return;
      }
      throw new Error('Failed to load session messages');
    }
    
    const messages = await response.json();
    
    // Display messages
    const { addMessageToUI } = await import('./messages.js');
    messages.forEach(msg => {
      if (msg.role === 'user' || msg.role === 'assistant') {
        // Skip system messages as they're not shown in UI
        addMessageToUI(msg);
      } else if (msg.role === 'system' && msg.content && msg.content.includes('=== Compacted Conversation')) {
        // Handle compacted messages
        const { addCompactedMessageToUI } = await import('./compaction.js');
        addCompactedMessageToUI(msg.content);
      }
    });
    
    // Scroll to bottom
    if (chatContainer) {
      chatContainer.scrollTop = chatContainer.scrollHeight;
    }
    
    // Check compaction stats for this session
    const { updateCompactionStats } = await import('./compaction.js');
    await updateCompactionStats();
    
  } catch (error) {
    console.error('Error loading session:', error);
    addSystemMessageToUI('Failed to load session messages', 'error');
  }
}

// Create a new session
export async function createNewSession() {
  try {
    // Check if we're already creating a session
    if (state.pendingNewSession) {
      console.log('Already creating a new session...');
      return;
    }
    
    setState('pendingNewSession', true);
    
    const response = await fetch('/api/session', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        title: `Session ${new Date().toLocaleString()}`
      })
    });
    
    if (!response.ok) {
      throw new Error('Failed to create session');
    }
    
    const session = await response.json();
    console.log('Created new session:', session.id);
    
    // Add session to sidebar
    await loadSessions();
    
    // Select the new session
    await selectSession(session.id);
    
  } catch (error) {
    console.error('Error creating session:', error);
    addSystemMessageToUI('Failed to create new session', 'error');
  } finally {
    setState('pendingNewSession', false);
  }
}

// Load all sessions
export async function loadSessions() {
  try {
    const response = await fetch('/api/session');
    if (!response.ok) {
      throw new Error('Failed to load sessions');
    }
    
    const sessions = await response.json();
    
    // Update sessions list in sidebar
    const sessionsList = document.getElementById('sessions-list');
    if (sessionsList) {
      sessionsList.innerHTML = '';
      
      sessions.forEach(session => {
        const sessionItem = document.createElement('div');
        sessionItem.className = 'session-item';
        sessionItem.dataset.sessionId = session.id;
        if (session.id === state.currentSessionId) {
          sessionItem.classList.add('active');
        }
        
        // Format date
        const date = new Date(session.created_at);
        const timeStr = date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
        
        sessionItem.innerHTML = `
          <div class="session-info">
            <div class="session-title">${session.title || 'Untitled Session'}</div>
            <div class="session-time">${timeStr}</div>
          </div>
          <button class="delete-session-btn" onclick="deleteSession('${session.id}', event)">×</button>
        `;
        
        sessionItem.onclick = (e) => {
          if (!e.target.classList.contains('delete-session-btn')) {
            selectSession(session.id);
          }
        };
        
        sessionsList.appendChild(sessionItem);
      });
      
      // If no current session and sessions exist, select the first one
      if (!state.currentSessionId && sessions.length > 0) {
        selectSession(sessions[0].id);
      } else if (sessions.length === 0) {
        // No sessions exist, create one
        createNewSession();
      }
    }
  } catch (error) {
    console.error('Error loading sessions:', error);
  }
}

// Delete a session
export async function deleteSession(sessionId, event) {
  if (event) {
    event.stopPropagation();
  }
  
  if (!confirm('Are you sure you want to delete this session?')) {
    return;
  }
  
  try {
    const response = await fetch(`/api/session/${sessionId}`, {
      method: 'DELETE'
    });
    
    if (!response.ok) {
      throw new Error('Failed to delete session');
    }
    
    // If we deleted the current session, clear it
    if (sessionId === state.currentSessionId) {
      setState('currentSessionId', null);
      const chatContainer = document.getElementById('chat-container');
      if (chatContainer) {
        chatContainer.innerHTML = '';
      }
    }
    
    // Reload sessions
    await loadSessions();
    
  } catch (error) {
    console.error('Error deleting session:', error);
    addSystemMessageToUI('Failed to delete session', 'error');
  }
}

// Export for global access
window.selectSession = selectSession;
window.deleteSession = deleteSession;