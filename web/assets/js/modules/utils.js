// utils.js - Utility functions
// This module contains helper functions used across the application

// Format duration in milliseconds to human-readable format
export function formatDuration(ms) {
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  const minutes = Math.floor(ms / 60000);
  const seconds = Math.floor((ms % 60000) / 1000);
  return `${minutes}m ${seconds}s`;
}

// Format time ago
export function formatTimeAgo(date) {
  const seconds = Math.floor((new Date() - new Date(date)) / 1000);
  
  if (seconds < 60) return 'just now';
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
  if (seconds < 604800) return `${Math.floor(seconds / 86400)}d ago`;
  
  return new Date(date).toLocaleDateString();
}

// Update connection status indicator
export function updateConnectionStatus(status) {
  const statusElement = document.getElementById('connection-status');
  if (!statusElement) return;
  
  // Remove all status classes
  statusElement.classList.remove('connected', 'disconnected', 'reconnecting');
  
  switch (status) {
    case 'connected':
      statusElement.classList.add('connected');
      statusElement.textContent = '● Connected';
      statusElement.title = 'Real-time connection active';
      break;
    case 'disconnected':
      statusElement.classList.add('disconnected');
      statusElement.textContent = '○ Disconnected';
      statusElement.title = 'Real-time connection lost';
      break;
    case 'reconnecting':
      statusElement.classList.add('reconnecting');
      statusElement.textContent = '◐ Reconnecting...';
      statusElement.title = 'Attempting to reconnect...';
      break;
    default:
      statusElement.textContent = '';
  }
}

// Show connection error with reconnect option
export function showConnectionError(message) {
  // Remove any existing error banner
  const existingBanner = document.getElementById('connection-error');
  if (existingBanner) {
    existingBanner.remove();
  }
  
  // Create error banner
  const banner = document.createElement('div');
  banner.id = 'connection-error';
  banner.className = 'connection-error-banner';
  banner.innerHTML = `
    <span class="error-message">${message}</span>
    <button class="reconnect-btn" onclick="manualReconnect()">Reconnect</button>
    <button class="dismiss-btn" onclick="this.parentElement.remove()">×</button>
  `;
  
  // Insert at top of body
  document.body.insertBefore(banner, document.body.firstChild);
}

// Get status icon for different states
export function getStatusIcon(status) {
  switch(status) {
    case 'completed': return '✅';
    case 'failed': return '❌';
    case 'running': return '⚡';
    case 'paused': return '⏸️';
    case 'pending': return '⏳';
    default: return '○';
  }
}

// Debounce function
export function debounce(func, wait) {
  let timeout;
  return function executedFunction(...args) {
    const later = () => {
      clearTimeout(timeout);
      func(...args);
    };
    clearTimeout(timeout);
    timeout = setTimeout(later, wait);
  };
}

// Throttle function
export function throttle(func, limit) {
  let inThrottle;
  return function(...args) {
    if (!inThrottle) {
      func.apply(this, args);
      inThrottle = true;
      setTimeout(() => inThrottle = false, limit);
    }
  };
}

// Parse file path to get directory and filename
export function parsePath(path) {
  const lastSlash = path.lastIndexOf('/');
  if (lastSlash === -1) {
    return { dir: '', filename: path };
  }
  return {
    dir: path.substring(0, lastSlash),
    filename: path.substring(lastSlash + 1)
  };
}

// Format file size
export function formatFileSize(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

// Check if a string is valid JSON
export function isValidJSON(str) {
  try {
    JSON.parse(str);
    return true;
  } catch (e) {
    return false;
  }
}

// Deep clone object
export function deepClone(obj) {
  if (obj === null || typeof obj !== 'object') return obj;
  if (obj instanceof Date) return new Date(obj.getTime());
  if (obj instanceof Array) return obj.map(item => deepClone(item));
  if (obj instanceof Object) {
    const cloned = {};
    for (const key in obj) {
      if (obj.hasOwnProperty(key)) {
        cloned[key] = deepClone(obj[key]);
      }
    }
    return cloned;
  }
}

// Generate unique ID
export function generateId() {
  return `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

// Export for global access if needed
window.manualReconnect = async function() {
  const { manualReconnect } = await import('./sse.js');
  manualReconnect();
};