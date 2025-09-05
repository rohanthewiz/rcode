/**
 * Messages Module - Message handling and display functionality
 * Handles message creation, streaming, formatting, and UI updates
 */

(function() {
  'use strict';

  // Module state
  let currentStreamingMessageDiv = null;
  let currentStreamingContent = '';
  let currentRequestController = null;
  let thinkingReturnTimer = null;

  /**
   * Add a message to the UI
   * @param {Object} message - Message object with role and content
   */
  function addMessageToUI(message) {
    const messagesContainer = document.getElementById('messages');
    if (!messagesContainer) {
      console.error('Messages container not found');
      return;
    }

    const messageDiv = document.createElement('div');
    messageDiv.className = `message ${message.role}`;
    
    const contentDiv = document.createElement('div');
    contentDiv.className = 'message-content';
    
    // Process markdown content
    const processedContent = window.processMarkdown ? 
      window.processMarkdown(message.content) : message.content;
    contentDiv.innerHTML = processedContent;
    
    messageDiv.appendChild(contentDiv);
    messagesContainer.appendChild(messageDiv);
    
    // Highlight code blocks if hljs is available
    if (window.hljs) {
      contentDiv.querySelectorAll('pre code').forEach((block) => {
        window.hljs.highlightElement(block);
      });
    }
    
    // Scroll to bottom
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
  }

  /**
   * Add a thinking indicator to show processing
   * @param {string} id - Unique identifier for the indicator
   */
  function addThinkingIndicator(id) {
    const messagesContainer = document.getElementById('messages');
    if (!messagesContainer) return;

    const thinkingDiv = document.createElement('div');
    thinkingDiv.className = 'message assistant thinking';
    thinkingDiv.id = id;
    
    const contentDiv = document.createElement('div');
    contentDiv.className = 'message-content';
    contentDiv.innerHTML = '<span class="thinking-indicator">Thinking<span class="dots">...</span></span>';
    
    thinkingDiv.appendChild(contentDiv);
    messagesContainer.appendChild(thinkingDiv);
    messagesContainer.scrollTop = messagesContainer.scrollHeight;
  }

  /**
   * Remove a thinking indicator
   * @param {string} id - Identifier of the indicator to remove
   */
  function removeThinkingIndicator(id) {
    const indicator = document.getElementById(id);
    if (indicator) {
      indicator.remove();
    }
  }

  /**
   * Create a streaming message container
   */
  function createStreamingMessage() {
    // Remove any existing thinking indicator
    const thinkingIndicator = document.querySelector('.message.thinking');
    if (thinkingIndicator) {
      console.log('Removing thinking indicator for streaming message');
      thinkingIndicator.remove();
    }

    const messagesContainer = document.getElementById('messages');
    if (!messagesContainer) {
      console.error('Messages container not found');
      return;
    }

    // Create new streaming message if needed
    if (!currentStreamingMessageDiv) {
      currentStreamingMessageDiv = document.createElement('div');
      currentStreamingMessageDiv.className = 'message assistant streaming';
      
      const contentDiv = document.createElement('div');
      contentDiv.className = 'message-content';
      
      currentStreamingMessageDiv.appendChild(contentDiv);
      messagesContainer.appendChild(currentStreamingMessageDiv);
    }
  }

  /**
   * Append content to the streaming message
   * @param {string} delta - Content to append
   */
  function appendToStreamingMessage(delta) {
    if (!currentStreamingMessageDiv) {
      createStreamingMessage();
    }

    currentStreamingContent += delta;
    
    const contentDiv = currentStreamingMessageDiv.querySelector('.message-content');
    if (contentDiv) {
      // Process markdown
      const processedContent = window.processMarkdown ? 
        window.processMarkdown(currentStreamingContent) : currentStreamingContent;
      contentDiv.innerHTML = processedContent;
      
      // Auto-scroll
      const messagesContainer = document.getElementById('messages');
      if (messagesContainer) {
        messagesContainer.scrollTop = messagesContainer.scrollHeight;
      }
    }
  }

  /**
   * Finalize the streaming message
   */
  function finalizeStreamingMessage() {
    if (!currentStreamingMessageDiv) return;

    // Remove streaming class
    currentStreamingMessageDiv.classList.remove('streaming');
    
    // Apply syntax highlighting
    const content = currentStreamingMessageDiv.querySelector('.message-content');
    if (content && window.hljs) {
      content.querySelectorAll('pre code').forEach((block) => {
        window.hljs.highlightElement(block);
      });
    }
    
    // Reset streaming state
    currentStreamingMessageDiv = null;
    currentStreamingContent = '';
  }

  /**
   * Detect file paths in message and offer to load images
   * @param {string} content - Message content to process
   * @returns {Promise<string>} Processed content with image instructions if needed
   */
  async function detectAndHandleFilePaths(content) {
    // Regular expressions to detect file paths
    const filePathPatterns = [
      // [Image attached: ...] pattern
      /\[Image attached: (.+?\.(?:png|jpg|jpeg|gif|webp|svg|bmp|ico|tiff?)) - .*?\]/gi,
      // Absolute paths
      /(?:^|\s)(\/[\w\-_. \/]+\.(?:png|jpg|jpeg|gif|webp|svg|bmp|ico|tiff?))\b/gi,
      // Home directory paths
      /(?:^|\s)(~\/[\w\-_. \/]+\.(?:png|jpg|jpeg|gif|webp|svg|bmp|ico|tiff?))\b/gi,
      // Relative paths
      /(?:^|\s)(\.{1,2}\/[\w\-_. \/]+\.(?:png|jpg|jpeg|gif|webp|svg|bmp|ico|tiff?))\b/gi
    ];
    
    let detectedPaths = [];
    let processedContent = content;
    
    // Find all file paths
    filePathPatterns.forEach(pattern => {
      const matches = content.matchAll(pattern);
      for (const match of matches) {
        const path = match[1];
        if (!detectedPaths.includes(path)) {
          detectedPaths.push(path);
        }
      }
    });
    
    // Ask if user wants to load images
    if (detectedPaths.length > 0) {
      const loadImages = confirm(
        `Found ${detectedPaths.length} image path(s) in your message:\n\n` +
        detectedPaths.join('\n') + 
        '\n\nWould you like to load these images?'
      );
      
      if (loadImages) {
        const imageInstructions = detectedPaths.map(path => 
          `Please use the read_file tool to load the image at: ${path}`
        ).join('\n');
        
        processedContent = content + '\n\n' + imageInstructions;
      }
    }
    
    return processedContent;
  }

  /**
   * Send a message to the current session
   */
  async function sendMessage() {
    console.log('sendMessage called');

    if (!window.editor) {
      console.error('Monaco editor not initialized');
      return;
    }

    let content = window.editor.getValue().trim();
    console.log('Message content:', content);
    
    // Check for file paths
    content = await detectAndHandleFilePaths(content);

    if (!content) {
      console.log('No content, returning');
      return;
    }

    // Get current session ID
    const currentSessionId = window.AppState ? 
      window.AppState.getState('currentSessionId') : window.currentSessionId;
    const pendingNewSession = window.AppState ? 
      window.AppState.getState('pendingNewSession') : window.pendingNewSession;

    // Create session if needed
    if (!currentSessionId || pendingNewSession) {
      console.log('Creating new session before sending message');
      try {
        if (window.actuallyCreateSession) {
          await window.actuallyCreateSession();
          console.log('Session created:', currentSessionId);
        }
      } catch (error) {
        console.error('Failed to create session:', error);
        alert('Failed to create session. Please try again.');
        return;
      }
    }

    const sessionId = window.AppState ? 
      window.AppState.getState('currentSessionId') : window.currentSessionId;
    
    console.log('Sending to session:', sessionId);

    // Reset first response flag
    if (window.AppState) {
      window.AppState.setState('hasReceivedFirstResponse', false);
    } else {
      window.hasReceivedFirstResponse = false;
    }

    // Add user message to UI
    addMessageToUI({ role: 'user', content: content });

    // Clear input
    window.editor.setValue('');

    // Add thinking indicator
    const thinkingId = 'thinking-' + Date.now();
    addThinkingIndicator(thinkingId);

    // Show stop button
    if (window.toggleStopButton) {
      window.toggleStopButton(true);
    }
    
    // Set processing state
    if (window.AppState) {
      window.AppState.setStateMultiple({
        isProcessing: true,
        isLLMProcessing: true
      });
    } else {
      window.isProcessing = true;
      window.isLLMProcessing = true;
    }

    try {
      // Create abort controller
      currentRequestController = new AbortController();

      // Get selected model
      const modelSelector = document.getElementById('model-selector');
      const selectedModel = modelSelector ? modelSelector.value : 'claude-sonnet-4-20250514';

      // Prepare request body
      let requestBody = {
        content: content,
        model: selectedModel
      };
      
      // Include pasted images if any
      if (window.editor.pastedImages && window.editor.pastedImages.length > 0) {
        requestBody.images = window.editor.pastedImages;
        console.log(`Including ${window.editor.pastedImages.length} pasted image(s) with message`);
        window.editor.pastedImages = [];
      }

      console.log('Making API request with model:', selectedModel);
      const response = await fetch('/api/session/' + sessionId + '/message', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(requestBody),
        signal: currentRequestController.signal
      });

      console.log('Response status:', response.status);

      if (!response.ok) {
        // Remove thinking indicator on error
        removeThinkingIndicator(thinkingId);
        
        // Reset processing state
        if (window.AppState) {
          window.AppState.setStateMultiple({
            isLLMProcessing: false,
            toolsAnnounced: false
          });
        } else {
          window.isLLMProcessing = false;
          window.toolsAnnounced = false;
        }
        
        // Clear thinking timer
        if (thinkingReturnTimer) {
          clearTimeout(thinkingReturnTimer);
          thinkingReturnTimer = null;
        }
        
        const errorText = await response.text();
        console.error('Response error:', errorText);
        
        // Handle session not found
        if (response.status === 404) {
          console.log('Session not found, creating new session');
          
          // Clear current session
          if (window.AppState) {
            window.AppState.setState('currentSessionId', null);
          } else {
            window.currentSessionId = null;
          }
          
          if (window.createNewSession) {
            await window.createNewSession();
          }
          
          addMessageToUI({
            role: 'assistant',
            content: 'Previous session was lost (server may have restarted). Created a new session. Please resend your message.'
          });
          
          // Restore the message
          window.editor.setValue(content);
          return;
        }
        
        throw new Error('Failed to send message: ' + errorText);
      }

      const result = await response.json();
      console.log('Response data:', result);
      
      // Remove thinking indicator
      removeThinkingIndicator(thinkingId);
      
      // Display tool summaries if any
      displayToolSummaries(result.toolSummaries);

      // Reload sessions to update title
      if (window.loadSessions) {
        window.loadSessions();
      }
      
    } catch (error) {
      removeThinkingIndicator(thinkingId);
      
      // Handle abort vs other errors
      if (error.name !== 'AbortError') {
        console.error('Failed to send message:', error);
        alert('Failed to send message: ' + error.message);
      } else {
        console.log('Request cancelled by user');
        addMessageToUI({
          role: 'assistant',
          content: '⚠️ Operation cancelled by user'
        });
      }
    } finally {
      // Reset UI state
      if (window.toggleStopButton) {
        window.toggleStopButton(false);
      }
      
      // Reset processing state
      if (window.AppState) {
        window.AppState.setStateMultiple({
          isProcessing: false,
          isLLMProcessing: false,
          toolsAnnounced: false
        });
      } else {
        window.isProcessing = false;
        window.isLLMProcessing = false;
        window.toolsAnnounced = false;
      }
      
      currentRequestController = null;
      
      // Clear thinking timer
      if (thinkingReturnTimer) {
        clearTimeout(thinkingReturnTimer);
        thinkingReturnTimer = null;
      }
    }
  }

  /**
   * Display tool summaries
   * @param {Array} toolSummaries - Array of tool summary strings
   */
  function displayToolSummaries(toolSummaries) {
    if (!toolSummaries || toolSummaries.length === 0) return;

    const messagesContainer = document.getElementById('messages');
    if (!messagesContainer) return;

    const toolsSummary = document.createElement('div');
    toolsSummary.className = 'tools-summary';
    toolsSummary.innerHTML = '<div class="tools-header">🛠️ TOOL USE</div><div class="tools-list"></div>';
    
    const toolsList = toolsSummary.querySelector('.tools-list');
    toolSummaries.forEach(summary => {
      const toolItem = document.createElement('div');
      toolItem.className = 'tool-item';
      toolItem.textContent = summary;
      toolsList.appendChild(toolItem);
    });
    
    messagesContainer.appendChild(toolsSummary);
  }

  /**
   * Stop the current request
   */
  function stopCurrentRequest() {
    if (currentRequestController) {
      currentRequestController.abort();
      console.log('Request aborted');
    }
  }

  /**
   * Get current streaming state
   */
  function getStreamingState() {
    return {
      isStreaming: currentStreamingMessageDiv !== null,
      content: currentStreamingContent
    };
  }

  // Export to global scope
  window.MessagesModule = {
    addMessageToUI,
    addThinkingIndicator,
    removeThinkingIndicator,
    createStreamingMessage,
    appendToStreamingMessage,
    finalizeStreamingMessage,
    detectAndHandleFilePaths,
    sendMessage,
    displayToolSummaries,
    stopCurrentRequest,
    getStreamingState
  };

  // Also expose individual functions for backward compatibility
  window.addMessageToUI = addMessageToUI;
  window.addThinkingIndicator = addThinkingIndicator;
  window.removeThinkingIndicator = removeThinkingIndicator;
  window.createStreamingMessage = createStreamingMessage;
  window.appendToStreamingMessage = appendToStreamingMessage;
  window.finalizeStreamingMessage = finalizeStreamingMessage;
  window.detectAndHandleFilePaths = detectAndHandleFilePaths;
  window.sendMessage = sendMessage;

})();