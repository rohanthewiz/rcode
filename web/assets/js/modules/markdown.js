// markdown.js - Markdown configuration and rendering
// This module handles markdown parsing configuration using marked.js

// Configure marked.js for markdown rendering
export function configureMarked() {
  if (typeof marked === 'undefined') {
    console.error('marked.js is not loaded');
    return;
  }
  
  // Custom renderer for marked to handle code blocks with hljs
  const renderer = new marked.Renderer();
  
  // Override code block rendering
  renderer.code = function(code, language) {
    if (language && hljs.getLanguage(language)) {
      try {
        const highlighted = hljs.highlight(code, { language: language }).value;
        return `<pre><code class="hljs language-${language}">${highlighted}</code></pre>`;
      } catch (e) {
        console.error('Highlight error:', e);
      }
    }
    // Fallback to auto-detection
    const highlighted = hljs.highlightAuto(code).value;
    return `<pre><code class="hljs">${highlighted}</code></pre>`;
  };
  
  // Configure marked options
  marked.setOptions({
    renderer: renderer,
    gfm: true,
    breaks: true,
    pedantic: false,
    sanitize: false,
    smartLists: true,
    smartypants: false
  });
}

// Render markdown content
export function renderMarkdown(content) {
  if (typeof marked === 'undefined') {
    console.error('marked.js is not loaded');
    return content;
  }
  
  try {
    return marked.parse(content);
  } catch (error) {
    console.error('Markdown parsing error:', error);
    return content;
  }
}

// Escape HTML for safe display
export function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}