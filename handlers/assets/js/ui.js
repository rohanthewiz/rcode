let currentSessionId = null;
let eventSource = null;
let messageInput;
let editor = null;

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
  eventSource = new EventSource('/events');

  eventSource.onmessage = function(event) {
    const data = JSON.parse(event.data);
    handleServerEvent(data);
  };

  eventSource.onerror = function(error) {
    console.error('SSE error:', error);
    setTimeout(connectEventSource, 5000); // Reconnect after 5 seconds
  };
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
    const htmlContent = marked.parse(message.content);
    content.innerHTML = htmlContent;

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

  if (!currentSessionId) {
    console.log('No session, creating new one');
    // Create new session if none selected
    await createNewSession();
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
  } catch (error) {
    // Remove thinking indicator on error
    removeThinkingIndicator(thinkingId);
    console.error('Failed to send message:', error);
    alert('Failed to send message: ' + error.message);
  }
}

// Create new session
async function createNewSession() {
  try {
    const response = await fetch('/api/session', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });

    const session = await response.json();
    currentSessionId = session.id;
    await loadSessions();

    // Clear messages
    document.getElementById('messages').innerHTML = '';
  } catch (error) {
    console.error('Failed to create session:', error);
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
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter, function() {
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
