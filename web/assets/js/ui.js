let currentSessionId = null;
let eventSource = null;
let messageInput;
let editor = null;
let pendingNewSession = false; // Track if we're waiting to create a new session

// SSE connection tracking
let reconnectAttempts = 0;
let reconnectDelay = 1000; // Start with 1 second
const maxReconnectAttempts = 5;
const maxReconnectDelay = 30000; // Max 30 seconds
let isManuallyDisconnected = false;
let connectionStatus = 'disconnected'; // 'connected', 'disconnected', 'reconnecting'

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
  console.log('Received SSE event:', event);

  if (event.type === 'message' && event.sessionID === currentSessionId) {
    console.log('Adding assistant message to UI');
    // Add assistant message to UI
    addMessageToUI(event.data);
    // Scroll to bottom
    const messagesContainer = document.getElementById('messages');
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
  } else if (event.type === 'session_list_updated') {
    loadSessions();
  }
}

// Load sessions
async function loadSessions() {
  try {
    const response = await fetch('/api/session');
    const sessions = await response.json();

    const sessionList = document.getElementById('session-list');
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
  pendingNewSession = false; // Clear pending state when selecting existing session
  loadMessages();
  loadSessions(); // Refresh to update active state
}

// Load messages for current session
async function loadMessages() {
  if (!currentSessionId) return;

  try {
    const response = await fetch('/api/session/' + currentSessionId + '/messages');
    const messages = await response.json();

    const messagesContainer = document.getElementById('messages');
    messagesContainer.innerHTML = '';

    messages.forEach(msg => {
      addMessageToUI(msg);
    });

    messagesContainer.scrollTop = messagesContainer.scrollHeight;
  } catch (error) {
    console.error('Failed to load messages:', error);
  }
}

// Add message to UI
function addMessageToUI(message) {
  const messagesContainer = document.getElementById('messages');
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

    if (modelId.includes('opus-4')) {
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
    if (value.includes('opus-4')) {
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
    thinkingDiv.remove();
  }
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

  // Add user message to UI immediately
  addMessageToUI({ role: 'user', content: content });

  // Clear input
  editor.setValue('');

  // Add a thinking indicator
  const thinkingId = 'thinking-' + Date.now();
  addThinkingIndicator(thinkingId);

  try {
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
      })
    });

    console.log('Response status:', response.status);

    // Remove thinking indicator
    removeThinkingIndicator(thinkingId);

    if (!response.ok) {
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

    // Add the assistant's response directly from the API response
    if (result.content) {
      addMessageToUI({
        role: 'assistant',
        content: result.content,
        model: result.model
      });
    }
    
    // Reload sessions to show updated title (for first message)
    // The backend will have updated the session title based on the first user message
    loadSessions();
  } catch (error) {
    // Remove thinking indicator on error
    removeThinkingIndicator(thinkingId);
    console.error('Failed to send message:', error);
    alert('Failed to send message: ' + error.message);
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

// Wait for DOM to be ready
document.addEventListener('DOMContentLoaded', function() {
  console.log('DOM loaded, initializing...');

  // Configure marked.js
  configureMarked();

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
});
