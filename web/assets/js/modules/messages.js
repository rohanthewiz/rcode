// messages.js - Message handling and display
// This module manages chat messages, streaming, and UI updates

import { state, setState, getState } from './state.js';
import { renderMarkdown, escapeHtml } from './markdown.js';
import { formatDuration } from './utils.js';

// Add a new message to the chat
export function addMessage(role, content) {
  if (!state.currentSessionId) return;
  
  const message = {
    role: role,
    content: content,
    timestamp: new Date().toISOString()
  };
  
  addMessageToUI(message);
}

// Add system message to UI
export function addSystemMessageToUI(message, type = 'info') {
  const chatContainer = document.getElementById('chat-container');
  if (!chatContainer) return;
  
  const messageElement = document.createElement('div');
  messageElement.className = `message system-message system-${type}`;
  messageElement.innerHTML = `
    <div class="message-content">
      <span class="system-icon">${type === 'error' ? '⚠️' : 'ℹ️'}</span>
      ${escapeHtml(message)}
    </div>
  `;
  
  chatContainer.appendChild(messageElement);
  chatContainer.scrollTop = chatContainer.scrollHeight;
}

// Add message to UI
export function addMessageToUI(message) {
  const chatContainer = document.getElementById('chat-container');
  if (!chatContainer) return;
  
  const messageElement = document.createElement('div');
  messageElement.className = `message ${message.role}`;
  messageElement.dataset.messageId = message.id || '';
  
  let content = message.content;
  
  // Handle different content types
  if (typeof content === 'string') {
    // Process text content
    content = processMessageContent(content, message.role);
  } else if (Array.isArray(content)) {
    // Handle content array (text and images)
    content = processContentArray(content, message.role);
  }
  
  // For assistant messages, render markdown
  if (message.role === 'assistant') {
    content = renderMarkdown(content);
  } else if (message.role === 'user') {
    // For user messages, just escape HTML but preserve line breaks
    content = escapeHtml(content).replace(/\n/g, '<br>');
  }
  
  messageElement.innerHTML = `
    <div class="message-header">
      <span class="message-role">${message.role === 'user' ? 'You' : 'Claude'}</span>
      <span class="message-time">${formatTime(message.timestamp)}</span>
    </div>
    <div class="message-content">${content}</div>
  `;
  
  chatContainer.appendChild(messageElement);
  
  // Add copy buttons to code blocks
  addCopyButtonsToCodeBlocks(messageElement);
  
  // Handle images in messages
  handleMessageImages(messageElement);
  
  chatContainer.scrollTop = chatContainer.scrollHeight;
}

// Process message content (handle pasted images)
function processMessageContent(content, role) {
  if (role === 'user') {
    // Check for pasted image references and replace with actual images
    return content.replace(/!\[([^\]]*)\]\(pasted:([^)]+)\)/g, (match, alt, filename) => {
      // Find the image in pastedImages
      const imageData = window.pastedImages?.find(img => img.filename === filename);
      if (imageData) {
        return `<img src="${imageData.data}" alt="${alt || filename}" class="pasted-image" data-filename="${filename}" />`;
      }
      return match; // Return original if image not found
    });
  }
  return content;
}

// Process content array (text and images)
function processContentArray(contentArray, role) {
  return contentArray.map(item => {
    if (item.type === 'text') {
      return processMessageContent(item.text, role);
    } else if (item.type === 'image') {
      const imageData = item.source?.data || '';
      const mediaType = item.source?.media_type || 'image/png';
      const imgSrc = imageData.startsWith('data:') ? imageData : `data:${mediaType};base64,${imageData}`;
      return `<img src="${imgSrc}" alt="Image" class="message-image" />`;
    }
    return '';
  }).join('\n');
}

// Add copy buttons to code blocks
function addCopyButtonsToCodeBlocks(container) {
  const codeBlocks = container.querySelectorAll('pre code');
  codeBlocks.forEach(block => {
    const pre = block.parentElement;
    if (!pre.querySelector('.copy-button')) {
      const copyButton = document.createElement('button');
      copyButton.className = 'copy-button';
      copyButton.textContent = 'Copy';
      copyButton.onclick = () => {
        navigator.clipboard.writeText(block.textContent);
        copyButton.textContent = 'Copied!';
        setTimeout(() => {
          copyButton.textContent = 'Copy';
        }, 2000);
      };
      pre.style.position = 'relative';
      pre.appendChild(copyButton);
    }
  });
}

// Handle images in messages (add click to expand)
function handleMessageImages(container) {
  const images = container.querySelectorAll('img.pasted-image, img.message-image');
  images.forEach(img => {
    img.style.cursor = 'pointer';
    img.onclick = () => {
      showImageModal(img.src, img.alt);
    };
  });
}

// Show image in modal
export function showImageModal(src, alt) {
  // Remove any existing modal
  const existingModal = document.getElementById('image-modal');
  if (existingModal) {
    existingModal.remove();
  }
  
  // Create modal
  const modal = document.createElement('div');
  modal.id = 'image-modal';
  modal.className = 'image-modal';
  modal.innerHTML = `
    <div class="image-modal-content">
      <span class="image-modal-close">&times;</span>
      <img src="${src}" alt="${alt || 'Image'}" />
      <div class="image-modal-caption">${alt || ''}</div>
    </div>
  `;
  
  document.body.appendChild(modal);
  
  // Add close handlers
  const closeBtn = modal.querySelector('.image-modal-close');
  closeBtn.onclick = () => modal.remove();
  
  modal.onclick = (e) => {
    if (e.target === modal) {
      modal.remove();
    }
  };
  
  // Add keyboard handler
  const handleEscape = (e) => {
    if (e.key === 'Escape') {
      modal.remove();
      document.removeEventListener('keydown', handleEscape);
    }
  };
  document.addEventListener('keydown', handleEscape);
}

// Add thinking indicator
export function addThinkingIndicator(id) {
  const chatContainer = document.getElementById('chat-container');
  if (!chatContainer) return;
  
  // Check if thinking indicator already exists
  if (document.querySelector('.message.thinking')) {
    return;
  }
  
  const thinkingElement = document.createElement('div');
  thinkingElement.className = 'message thinking assistant';
  thinkingElement.dataset.thinkingId = id;
  thinkingElement.innerHTML = `
    <div class="message-header">
      <span class="message-role">Claude</span>
      <span class="thinking-status">Thinking...</span>
    </div>
    <div class="message-content">
      <div class="thinking-indicator">
        <span></span>
        <span></span>
        <span></span>
      </div>
    </div>
  `;
  
  chatContainer.appendChild(thinkingElement);
  chatContainer.scrollTop = chatContainer.scrollHeight;
}

// Remove thinking indicator
export function removeThinkingIndicator(id) {
  const thinkingElement = document.querySelector(`.message.thinking[data-thinking-id="${id}"]`);
  if (thinkingElement) {
    thinkingElement.remove();
  }
  
  // Also remove any thinking indicator without specific ID as fallback
  const genericThinking = document.querySelector('.message.thinking');
  if (genericThinking) {
    genericThinking.remove();
  }
}

// Create streaming message container
export function createStreamingMessage() {
  const chatContainer = document.getElementById('chat-container');
  if (!chatContainer) return null;
  
  const messageElement = document.createElement('div');
  messageElement.className = 'message assistant streaming';
  messageElement.innerHTML = `
    <div class="message-header">
      <span class="message-role">Claude</span>
      <span class="message-time">${formatTime(new Date())}</span>
    </div>
    <div class="message-content"></div>
  `;
  
  chatContainer.appendChild(messageElement);
  setState('currentStreamingMessage', messageElement);
  setState('currentStreamingContent', '');
  
  return messageElement;
}

// Append content to streaming message
export function appendToStreamingMessage(delta) {
  if (!state.currentStreamingMessage) {
    createStreamingMessage();
  }
  
  state.currentStreamingContent += delta;
  const contentElement = state.currentStreamingMessage.querySelector('.message-content');
  
  if (contentElement) {
    // Render markdown content
    contentElement.innerHTML = renderMarkdown(state.currentStreamingContent);
    
    // Add copy buttons to new code blocks
    addCopyButtonsToCodeBlocks(state.currentStreamingMessage);
    
    // Scroll to bottom
    const chatContainer = document.getElementById('chat-container');
    if (chatContainer) {
      chatContainer.scrollTop = chatContainer.scrollHeight;
    }
  }
}

// Finalize streaming message
export function finalizeStreamingMessage() {
  if (state.currentStreamingMessage) {
    state.currentStreamingMessage.classList.remove('streaming');
    setState('currentStreamingMessage', null);
    setState('currentStreamingContent', '');
  }
}

// Format time for display
function formatTime(timestamp) {
  if (!timestamp) return '';
  
  const date = new Date(timestamp);
  const now = new Date();
  const diff = now - date;
  
  // If today, show time
  if (date.toDateString() === now.toDateString()) {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
  
  // If yesterday
  const yesterday = new Date(now);
  yesterday.setDate(yesterday.getDate() - 1);
  if (date.toDateString() === yesterday.toDateString()) {
    return 'Yesterday ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
  }
  
  // Otherwise show date and time
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}