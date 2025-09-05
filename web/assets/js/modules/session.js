/**
 * Session Module - Session management functionality
 * Handles session creation, loading, switching, and deletion
 */

(function() {
  'use strict';

  /**
   * Load all sessions from the server
   */
  async function loadSessions() {
    try {
      const response = await fetch('/api/session');
      if (!response.ok) {
        console.error('Failed to fetch sessions:', response.status);
        return;
      }
      
      const sessions = await response.json();
      
      // Ensure sessions is an array
      if (!Array.isArray(sessions)) {
        console.error('Invalid sessions response:', sessions);
        return;
      }
      
      const sessionsList = document.getElementById('sessions-list');
      if (!sessionsList) {
        console.error('Sessions list element not found');
        return;
      }
      
      sessionsList.innerHTML = '';
      
      // Get current session ID
      const currentSessionId = window.AppState ? 
        window.AppState.getState('currentSessionId') : window.currentSessionId;
      
      sessions.forEach(session => {
        const sessionItem = document.createElement('div');
        sessionItem.className = 'session-item';
        if (session.id === currentSessionId) {
          sessionItem.classList.add('active');
        }
        
        // Create session title element
        const sessionTitle = document.createElement('span');
        sessionTitle.className = 'session-title';
        sessionTitle.textContent = session.title || 'New Session';
        sessionTitle.onclick = () => switchToSession(session.id);
        
        // Create delete button
        const deleteBtn = document.createElement('button');
        deleteBtn.className = 'delete-session-btn';
        deleteBtn.innerHTML = '×';
        deleteBtn.title = 'Delete session';
        deleteBtn.onclick = (e) => {
          e.stopPropagation();
          deleteSession(session.id);
        };
        
        sessionItem.appendChild(sessionTitle);
        sessionItem.appendChild(deleteBtn);
        sessionsList.appendChild(sessionItem);
      });
    } catch (error) {
      console.error('Failed to load sessions:', error);
    }
  }

  /**
   * Switch to a different session
   * @param {string} sessionId - ID of the session to switch to
   */
  async function switchToSession(sessionId) {
    try {
      // Update current session ID
      if (window.AppState) {
        window.AppState.setStateMultiple({
          currentSessionId: sessionId,
          pendingNewSession: false
        });
      } else {
        window.currentSessionId = sessionId;
        window.pendingNewSession = false;
      }
      
      // Load messages for this session
      const response = await fetch(`/api/session/${sessionId}/messages`);
      const messages = await response.json();
      
      // Clear current messages
      const messagesContainer = document.getElementById('messages');
      if (messagesContainer) {
        messagesContainer.innerHTML = '';
        
        // Add all messages
        messages.forEach(msg => {
          if (window.addMessageToUI) {
            window.addMessageToUI(msg);
          }
        });
      }
      
      // Update active session in UI
      document.querySelectorAll('.session-item').forEach(item => {
        item.classList.remove('active');
      });
      
      // Find and activate the clicked session
      const sessionItems = document.querySelectorAll('.session-item');
      sessionItems.forEach(item => {
        const titleElement = item.querySelector('.session-title');
        if (titleElement && titleElement.textContent === 
            (messages[0]?.content?.substring(0, 50) || 'New Session')) {
          item.classList.add('active');
        }
      });
      
      // Emit session switch event
      if (window.AppEvents) {
        window.AppEvents.emit(window.StandardEvents?.SESSION_SWITCH || 'session_switch', { 
          sessionId 
        });
      }
      
    } catch (error) {
      console.error('Failed to switch session:', error);
    }
  }

  /**
   * Delete a session
   * @param {string} sessionId - ID of the session to delete
   */
  async function deleteSession(sessionId) {
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
      
      // Get current session ID
      const currentSessionId = window.AppState ? 
        window.AppState.getState('currentSessionId') : window.currentSessionId;
      
      // If we deleted the current session, clear it
      if (sessionId === currentSessionId) {
        if (window.AppState) {
          window.AppState.setState('currentSessionId', null);
        } else {
          window.currentSessionId = null;
        }
        
        // Clear messages
        const messagesContainer = document.getElementById('messages');
        if (messagesContainer) {
          messagesContainer.innerHTML = '';
        }
      }
      
      // Reload sessions list
      loadSessions();
      
      // Emit session delete event
      if (window.AppEvents) {
        window.AppEvents.emit(window.StandardEvents?.SESSION_DELETE || 'session_delete', { 
          sessionId 
        });
      }
      
    } catch (error) {
      console.error('Failed to delete session:', error);
      alert('Failed to delete session');
    }
  }

  /**
   * Actually create a new session in the database
   */
  async function actuallyCreateSession() {
    try {
      const response = await fetch('/api/session', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      });

      const session = await response.json();
      
      // Update session ID
      if (window.AppState) {
        window.AppState.setStateMultiple({
          currentSessionId: session.id,
          pendingNewSession: false
        });
      } else {
        window.currentSessionId = session.id;
        window.pendingNewSession = false;
      }
      
      // Emit session create event
      if (window.AppEvents) {
        window.AppEvents.emit(window.StandardEvents?.SESSION_CREATE || 'session_create', { 
          sessionId: session.id 
        });
      }
      
      return session;
    } catch (error) {
      console.error('Failed to create session:', error);
      throw error;
    }
  }

  /**
   * Prepare UI for new session (called by New Session button)
   */
  async function createNewSession() {
    // Clear current session
    if (window.AppState) {
      window.AppState.setStateMultiple({
        currentSessionId: null,
        pendingNewSession: true
      });
    } else {
      window.currentSessionId = null;
      window.pendingNewSession = true;
    }
    
    // Clear messages
    const messagesContainer = document.getElementById('messages');
    if (messagesContainer) {
      messagesContainer.innerHTML = '';
    }
    
    // Remove active class from all sessions
    document.querySelectorAll('.session-item').forEach(item => {
      item.classList.remove('active');
    });
    
    // Focus on input
    if (window.editor) {
      window.editor.focus();
    }
  }

  /**
   * Get the current session ID
   */
  function getCurrentSessionId() {
    return window.AppState ? 
      window.AppState.getState('currentSessionId') : window.currentSessionId;
  }

  /**
   * Check if there's a pending new session
   */
  function isPendingNewSession() {
    return window.AppState ? 
      window.AppState.getState('pendingNewSession') : window.pendingNewSession;
  }

  // Export to global scope
  window.SessionModule = {
    loadSessions,
    switchToSession,
    deleteSession,
    actuallyCreateSession,
    createNewSession,
    getCurrentSessionId,
    isPendingNewSession
  };

  // Also expose individual functions for backward compatibility
  window.loadSessions = loadSessions;
  window.switchToSession = switchToSession;
  window.deleteSession = deleteSession;
  window.actuallyCreateSession = actuallyCreateSession;
  window.createNewSession = createNewSession;

})();