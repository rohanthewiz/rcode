/**
 * Utils Module - Common utility functions
 * Provides shared helper functions for the application
 */

(function() {
  'use strict';

  /**
   * Escape HTML for safe display
   * @param {string} text - Text to escape
   * @returns {string} Escaped HTML
   */
  function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  /**
   * Format duration from nanoseconds to human-readable format
   * @param {number} duration - Duration in nanoseconds
   * @returns {string} Formatted duration
   */
  function formatDuration(duration) {
    // Duration is in nanoseconds, convert to readable format
    const ms = duration / 1000000;
    if (ms < 1000) return `${ms.toFixed(0)}ms`;
    const s = ms / 1000;
    if (s < 60) return `${s.toFixed(1)}s`;
    const m = s / 60;
    return `${m.toFixed(1)}m`;
  }

  /**
   * Format bytes to human-readable format
   * @param {number} bytes - Number of bytes
   * @returns {string} Formatted size
   */
  function formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
  }

  /**
   * Debounce function to limit execution rate
   * @param {Function} func - Function to debounce
   * @param {number} wait - Wait time in milliseconds
   * @returns {Function} Debounced function
   */
  function debounce(func, wait) {
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

  /**
   * Throttle function to limit execution rate
   * @param {Function} func - Function to throttle
   * @param {number} limit - Minimum time between executions in milliseconds
   * @returns {Function} Throttled function
   */
  function throttle(func, limit) {
    let inThrottle;
    return function(...args) {
      if (!inThrottle) {
        func.apply(this, args);
        inThrottle = true;
        setTimeout(() => inThrottle = false, limit);
      }
    };
  }

  /**
   * Generate a unique ID
   * @param {string} prefix - Optional prefix for the ID
   * @returns {string} Unique ID
   */
  function generateId(prefix = 'id') {
    return `${prefix}-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`;
  }

  /**
   * Deep clone an object
   * @param {*} obj - Object to clone
   * @returns {*} Cloned object
   */
  function deepClone(obj) {
    if (obj === null || typeof obj !== 'object') return obj;
    if (obj instanceof Date) return new Date(obj.getTime());
    if (obj instanceof Array) return obj.map(item => deepClone(item));
    if (obj instanceof Object) {
      const clonedObj = {};
      for (const key in obj) {
        if (obj.hasOwnProperty(key)) {
          clonedObj[key] = deepClone(obj[key]);
        }
      }
      return clonedObj;
    }
  }

  /**
   * Check if a value is empty (null, undefined, empty string, empty array, empty object)
   * @param {*} value - Value to check
   * @returns {boolean} True if empty
   */
  function isEmpty(value) {
    if (value == null) return true;
    if (typeof value === 'string' || Array.isArray(value)) return value.length === 0;
    if (typeof value === 'object') return Object.keys(value).length === 0;
    return false;
  }

  /**
   * Format a date to a readable string
   * @param {Date|number|string} date - Date to format
   * @param {boolean} includeTime - Whether to include time
   * @returns {string} Formatted date string
   */
  function formatDate(date, includeTime = false) {
    const d = new Date(date);
    if (isNaN(d.getTime())) return 'Invalid date';
    
    const options = {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    };
    
    if (includeTime) {
      options.hour = '2-digit';
      options.minute = '2-digit';
    }
    
    return d.toLocaleString(undefined, options);
  }

  /**
   * Parse query parameters from URL
   * @param {string} url - URL to parse (defaults to current URL)
   * @returns {Object} Object with query parameters
   */
  function parseQueryParams(url = window.location.href) {
    const params = {};
    const queryString = url.split('?')[1];
    
    if (queryString) {
      queryString.split('&').forEach(param => {
        const [key, value] = param.split('=');
        params[decodeURIComponent(key)] = decodeURIComponent(value || '');
      });
    }
    
    return params;
  }

  /**
   * Copy text to clipboard
   * @param {string} text - Text to copy
   * @returns {Promise<boolean>} Promise that resolves to success status
   */
  async function copyToClipboard(text) {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(text);
        return true;
      } else {
        // Fallback for older browsers
        const textArea = document.createElement('textarea');
        textArea.value = text;
        textArea.style.position = 'fixed';
        textArea.style.left = '-999999px';
        document.body.appendChild(textArea);
        textArea.focus();
        textArea.select();
        const success = document.execCommand('copy');
        document.body.removeChild(textArea);
        return success;
      }
    } catch (error) {
      console.error('Failed to copy to clipboard:', error);
      return false;
    }
  }

  /**
   * Scroll element into view smoothly
   * @param {HTMLElement|string} element - Element or selector to scroll to
   * @param {Object} options - Scroll options
   */
  function scrollIntoView(element, options = {}) {
    const el = typeof element === 'string' ? document.querySelector(element) : element;
    if (el) {
      el.scrollIntoView({
        behavior: 'smooth',
        block: 'nearest',
        inline: 'nearest',
        ...options
      });
    }
  }

  // Export to global scope
  window.UtilsModule = {
    escapeHtml,
    formatDuration,
    formatBytes,
    debounce,
    throttle,
    generateId,
    deepClone,
    isEmpty,
    formatDate,
    parseQueryParams,
    copyToClipboard,
    scrollIntoView
  };

  // Also expose individual functions for backward compatibility
  window.escapeHtml = escapeHtml;
  window.formatDuration = formatDuration;
  window.formatBytes = formatBytes;
  window.debounce = debounce;
  window.throttle = throttle;
  window.generateId = generateId;

})();