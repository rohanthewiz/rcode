/**
 * Markdown Module - Markdown parsing and configuration
 * Handles markdown rendering with syntax highlighting
 */

(function() {
  'use strict';

  /**
   * Configure marked.js library for markdown parsing
   */
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
            } catch (err) {
              console.warn('Highlight error for language:', lang, err);
            }
          }
          return hljs.highlightAuto(code).value;
        }
      });
      
      console.log('Marked.js configured with syntax highlighting');
    } else {
      console.warn('Marked.js or hljs not available, markdown rendering disabled');
    }
  }

  /**
   * Process markdown content
   * @param {string} content - Markdown content
   * @returns {string} Processed HTML
   */
  function processMarkdown(content) {
    if (typeof marked === 'undefined') {
      // Fallback to basic text processing if marked is not available
      return escapeHtml(content);
    }
    
    try {
      return marked.parse(content);
    } catch (error) {
      console.error('Error processing markdown:', error);
      return escapeHtml(content);
    }
  }

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
   * Apply syntax highlighting to code blocks
   * @param {HTMLElement} container - Container element with code blocks
   */
  function highlightCodeBlocks(container) {
    if (typeof hljs === 'undefined') return;
    
    container.querySelectorAll('pre code').forEach((block) => {
      // Skip if already highlighted
      if (block.classList.contains('hljs')) return;
      
      try {
        hljs.highlightElement(block);
      } catch (error) {
        console.warn('Error highlighting code block:', error);
      }
    });
  }

  /**
   * Check if markdown is available
   * @returns {boolean} True if marked.js is loaded
   */
  function isMarkdownAvailable() {
    return typeof marked !== 'undefined';
  }

  /**
   * Check if syntax highlighting is available
   * @returns {boolean} True if highlight.js is loaded
   */
  function isSyntaxHighlightingAvailable() {
    return typeof hljs !== 'undefined';
  }

  // Export to global scope
  window.MarkdownModule = {
    configureMarked,
    processMarkdown,
    escapeHtml,
    highlightCodeBlocks,
    isMarkdownAvailable,
    isSyntaxHighlightingAvailable
  };

  // Also expose individual functions for backward compatibility
  window.configureMarked = configureMarked;
  window.processMarkdown = processMarkdown;

})();