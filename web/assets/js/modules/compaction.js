// compaction.js - Conversation compaction functionality
// This module handles compacting long conversations to reduce token usage

import { state, getState } from './state.js';
import { addSystemMessageToUI } from './messages.js';

// Compact the current session
export async function compactSession(options = {}) {
  const sessionId = getState('currentSessionId');
  if (!sessionId) {
    addSystemMessageToUI('No session selected', 'error');
    return;
  }

  // Show confirmation dialog
  const messageCount = document.querySelectorAll('.message').length;
  if (messageCount < 20) {
    addSystemMessageToUI('Not enough messages to compact (need at least 20)', 'warning');
    return;
  }

  if (!confirm(`This will compact the conversation, reducing approximately ${Math.floor(messageCount * 0.6)} messages into summaries. Continue?`)) {
    return;
  }

  try {
    // Show loading state
    const compactBtn = document.getElementById('compact-session-btn');
    if (compactBtn) {
      compactBtn.disabled = true;
      compactBtn.textContent = 'Compacting...';
    }

    // Default options
    const compactionOptions = {
      preserve_recent: options.preserveRecent || 10,
      preserve_initial: options.preserveInitial || 2,
      strategy: options.strategy || 'conservative',
      max_summary_tokens: options.maxSummaryTokens || 500,
      min_messages_to_compact: options.minMessagesToCompact || 20
    };

    const response = await fetch(`/api/session/${sessionId}/compact`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify(compactionOptions)
    });

    const data = await response.json();

    if (response.ok && data.success) {
      addSystemMessageToUI(
        `Conversation compacted successfully! Saved ${data.tokens_saved} tokens by compacting ${data.messages_compacted} messages.`,
        'success'
      );
      
      // Reload messages to show compacted view
      await reloadMessages();
      
      // Update compaction stats
      await updateCompactionStats();
    } else {
      throw new Error(data.error || 'Failed to compact conversation');
    }
  } catch (error) {
    console.error('Compaction error:', error);
    addSystemMessageToUI(`Failed to compact conversation: ${error.message}`, 'error');
  } finally {
    // Reset button state
    const compactBtn = document.getElementById('compact-session-btn');
    if (compactBtn) {
      compactBtn.disabled = false;
      compactBtn.textContent = 'Compact Conversation';
    }
  }
}

// Reload messages after compaction
async function reloadMessages() {
  const sessionId = getState('currentSessionId');
  if (!sessionId) return;

  try {
    const response = await fetch(`/api/session/${sessionId}/messages`);
    const messages = await response.json();

    // Clear chat container
    const chatContainer = document.getElementById('chat-container');
    if (chatContainer) {
      chatContainer.innerHTML = '';
    }

    // Re-add messages (compacted messages will show as system messages)
    messages.forEach(msg => {
      if (msg.role === 'system' && msg.content.includes('=== Compacted Conversation')) {
        // Special styling for compacted messages
        addCompactedMessageToUI(msg.content);
      } else {
        // Regular message
        const { addMessageToUI } = await import('./messages.js');
        addMessageToUI(msg);
      }
    });
  } catch (error) {
    console.error('Failed to reload messages:', error);
  }
}

// Add compacted message to UI with special styling
export function addCompactedMessageToUI(content) {
  const chatContainer = document.getElementById('chat-container');
  if (!chatContainer) return;

  const messageElement = document.createElement('div');
  messageElement.className = 'message compacted-message';
  
  // Parse the compacted message content
  const lines = content.split('\n');
  const title = lines[0]; // "=== Compacted Conversation (X messages) ==="
  const summary = lines.slice(1).join('\n');

  messageElement.innerHTML = `
    <div class="compacted-header">
      <span class="compacted-icon">📦</span>
      <span class="compacted-title">${escapeHtml(title)}</span>
      <button class="expand-btn" onclick="toggleCompactedMessage(this)">▼</button>
    </div>
    <div class="compacted-content" style="display: none;">
      <pre>${escapeHtml(summary)}</pre>
    </div>
  `;

  chatContainer.appendChild(messageElement);
}

// Toggle compacted message visibility
window.toggleCompactedMessage = function(button) {
  const content = button.closest('.compacted-message').querySelector('.compacted-content');
  if (content.style.display === 'none') {
    content.style.display = 'block';
    button.textContent = '▲';
  } else {
    content.style.display = 'none';
    button.textContent = '▼';
  }
};

// Update compaction statistics
export async function updateCompactionStats() {
  const sessionId = getState('currentSessionId');
  if (!sessionId) return;

  try {
    const response = await fetch(`/api/session/${sessionId}/compaction/stats`);
    const stats = await response.json();

    // Update UI with stats if needed
    console.log('Compaction stats:', stats);
    
    // Show/hide compact button based on message count
    const compactBtn = document.getElementById('compact-session-btn');
    if (compactBtn) {
      if (stats.current_message_count >= 20) {
        compactBtn.style.display = 'block';
      } else {
        compactBtn.style.display = 'none';
      }
    }
  } catch (error) {
    console.error('Failed to get compaction stats:', error);
  }
}

// Restore compacted messages
export async function restoreCompactedMessages(compactionId) {
  const sessionId = getState('currentSessionId');
  if (!sessionId) return;

  try {
    const response = await fetch(`/api/session/${sessionId}/compaction/${compactionId}/restore`, {
      method: 'POST'
    });

    const data = await response.json();
    if (response.ok && data.success) {
      addSystemMessageToUI('Messages restored successfully', 'success');
      await reloadMessages();
    } else {
      throw new Error(data.error || 'Failed to restore messages');
    }
  } catch (error) {
    console.error('Failed to restore messages:', error);
    addSystemMessageToUI(`Failed to restore messages: ${error.message}`, 'error');
  }
}

// Helper function to escape HTML
function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// Auto-compaction check
export async function checkAutoCompaction() {
  const sessionId = getState('currentSessionId');
  if (!sessionId) return;

  try {
    const response = await fetch(`/api/session/${sessionId}/compaction/stats`);
    const stats = await response.json();

    if (stats.auto_compact_enabled && stats.current_message_count > stats.compact_threshold) {
      // Trigger auto-compaction
      console.log('Auto-compacting session due to threshold');
      await compactSession({
        strategy: 'aggressive',
        preserveRecent: 5,
        preserveInitial: 1
      });
    }
  } catch (error) {
    console.error('Auto-compaction check failed:', error);
  }
}