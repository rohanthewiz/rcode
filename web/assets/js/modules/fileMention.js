// File mention system for @ command support in the chat input
// This module provides autocomplete functionality for easily selecting and mentioning files

class FileMentionSystem {
  constructor(editor) {
    this.editor = editor; // Monaco editor instance
    this.isActive = false;
    this.currentQuery = '';
    this.files = [];
    this.selectedIndex = 0;
    this.dropdown = null;
    this.position = null;
    this.triggerPosition = null;
    
    // Initialize the system
    this.init();
  }
  
  init() {
    // Create the dropdown element
    this.createDropdown();
    
    // Listen for text changes in the editor
    this.editor.onDidChangeModelContent((e) => {
      this.handleContentChange(e);
    });
    
    // Listen for cursor position changes
    this.editor.onDidChangeCursorPosition((e) => {
      if (this.isActive && !this.isWithinMention(e.position)) {
        this.hide();
      }
    });
    
    // Handle keyboard navigation
    this.editor.addCommand(monaco.KeyCode.UpArrow, () => {
      if (this.isActive) {
        this.selectPrevious();
        return true;
      }
    });
    
    this.editor.addCommand(monaco.KeyCode.DownArrow, () => {
      if (this.isActive) {
        this.selectNext();
        return true;
      }
    });
    
    this.editor.addCommand(monaco.KeyCode.Enter, () => {
      if (this.isActive) {
        this.insertSelectedFile();
        return true;
      }
    });
    
    this.editor.addCommand(monaco.KeyCode.Tab, () => {
      if (this.isActive) {
        this.insertSelectedFile();
        return true;
      }
    });
    
    this.editor.addCommand(monaco.KeyCode.Escape, () => {
      if (this.isActive) {
        this.hide();
        return true;
      }
    });
  }
  
  createDropdown() {
    // Create dropdown container
    this.dropdown = document.createElement('div');
    this.dropdown.className = 'file-mention-dropdown';
    this.dropdown.style.display = 'none';
    document.body.appendChild(this.dropdown);
  }
  
  handleContentChange(e) {
    const position = this.editor.getPosition();
    const model = this.editor.getModel();
    const lineContent = model.getLineContent(position.lineNumber);
    const textBeforeCursor = lineContent.substring(0, position.column - 1);
    
    // Check if we're typing after an @ symbol
    const atMatch = textBeforeCursor.match(/@([^\s@]*)$/);
    
    if (atMatch) {
      // Found @ mention in progress
      this.currentQuery = atMatch[1].toLowerCase();
      this.triggerPosition = {
        lineNumber: position.lineNumber,
        column: position.column - atMatch[0].length + 1
      };
      this.show();
      this.updateFileList();
    } else if (this.isActive) {
      // No longer in @ mention context
      this.hide();
    }
  }
  
  isWithinMention(position) {
    if (!this.triggerPosition) return false;
    
    const model = this.editor.getModel();
    const lineContent = model.getLineContent(position.lineNumber);
    const textBeforeCursor = lineContent.substring(0, position.column - 1);
    
    // Check if we're still within an @ mention
    return textBeforeCursor.match(/@[^\s@]*$/);
  }
  
  async updateFileList() {
    try {
      // Fetch the file tree from the server
      const response = await fetch('/api/files/tree');
      if (!response.ok) throw new Error('Failed to fetch file tree');
      
      const data = await response.json();
      this.files = this.flattenFileTree(data.tree || []);
      
      // Filter files based on current query
      const filteredFiles = this.filterFiles(this.files, this.currentQuery);
      
      // Update dropdown content
      this.renderDropdown(filteredFiles);
    } catch (error) {
      console.error('Error fetching file tree:', error);
      // Try to use cached file tree from sidebar if available
      this.useCachedFileTree();
    }
  }
  
  useCachedFileTree() {
    // Try to get file tree from the sidebar if it's already loaded
    const fileTreeElement = document.getElementById('file-tree');
    if (fileTreeElement) {
      const files = [];
      const fileNodes = fileTreeElement.querySelectorAll('.file-item');
      fileNodes.forEach(node => {
        const path = node.dataset.path;
        if (path) {
          files.push(path);
        }
      });
      this.files = files;
      const filteredFiles = this.filterFiles(this.files, this.currentQuery);
      this.renderDropdown(filteredFiles);
    }
  }
  
  flattenFileTree(tree, basePath = '') {
    const files = [];
    
    const traverse = (nodes, currentPath) => {
      nodes.forEach(node => {
        const fullPath = currentPath ? `${currentPath}/${node.name}` : node.name;
        
        if (node.type === 'file') {
          files.push(fullPath);
        } else if (node.type === 'directory' && node.children) {
          traverse(node.children, fullPath);
        }
      });
    };
    
    traverse(tree, basePath);
    return files;
  }
  
  filterFiles(files, query) {
    if (!query) {
      // Return top 10 most relevant files when no query
      return files.slice(0, 10);
    }
    
    // Score each file based on query match
    const scored = files.map(file => {
      const fileName = file.split('/').pop().toLowerCase();
      const filePath = file.toLowerCase();
      let score = 0;
      
      // Exact filename match
      if (fileName === query) {
        score += 100;
      }
      // Filename starts with query
      else if (fileName.startsWith(query)) {
        score += 50;
      }
      // Filename contains query
      else if (fileName.includes(query)) {
        score += 25;
      }
      // Path contains query
      else if (filePath.includes(query)) {
        score += 10;
      }
      
      // Fuzzy match
      if (this.fuzzyMatch(query, fileName)) {
        score += 5;
      }
      
      return { file, score };
    });
    
    // Sort by score and return top results
    return scored
      .filter(item => item.score > 0)
      .sort((a, b) => b.score - a.score)
      .slice(0, 10)
      .map(item => item.file);
  }
  
  fuzzyMatch(query, text) {
    let queryIndex = 0;
    for (let i = 0; i < text.length && queryIndex < query.length; i++) {
      if (text[i] === query[queryIndex]) {
        queryIndex++;
      }
    }
    return queryIndex === query.length;
  }
  
  renderDropdown(files) {
    if (files.length === 0) {
      this.dropdown.innerHTML = '<div class="file-mention-empty">No files found</div>';
      return;
    }
    
    this.dropdown.innerHTML = files.map((file, index) => {
      const fileName = file.split('/').pop();
      const directory = file.substring(0, file.lastIndexOf('/'));
      const isSelected = index === this.selectedIndex;
      
      return `
        <div class="file-mention-item ${isSelected ? 'selected' : ''}" data-index="${index}" data-path="${file}">
          <span class="file-icon">${this.getFileIcon(fileName)}</span>
          <div class="file-info">
            <div class="file-name">${this.highlightMatch(fileName, this.currentQuery)}</div>
            ${directory ? `<div class="file-path">${directory}</div>` : ''}
          </div>
        </div>
      `;
    }).join('');
    
    // Add click handlers
    this.dropdown.querySelectorAll('.file-mention-item').forEach(item => {
      item.addEventListener('click', () => {
        this.selectedIndex = parseInt(item.dataset.index);
        this.insertSelectedFile();
      });
      
      item.addEventListener('mouseenter', () => {
        this.selectedIndex = parseInt(item.dataset.index);
        this.updateSelection();
      });
    });
  }
  
  highlightMatch(text, query) {
    if (!query) return text;
    
    const regex = new RegExp(`(${query.split('').join('.*?')})`, 'gi');
    return text.replace(regex, '<strong>$1</strong>');
  }
  
  getFileIcon(fileName) {
    const ext = fileName.split('.').pop().toLowerCase();
    const iconMap = {
      'js': '📜',
      'ts': '📘',
      'jsx': '⚛️',
      'tsx': '⚛️',
      'go': '🐹',
      'py': '🐍',
      'java': '☕',
      'rb': '💎',
      'php': '🐘',
      'html': '🌐',
      'css': '🎨',
      'scss': '🎨',
      'json': '📋',
      'md': '📝',
      'txt': '📄',
      'yml': '⚙️',
      'yaml': '⚙️',
      'xml': '📰',
      'sql': '🗄️',
      'sh': '🖥️',
      'bash': '🖥️',
      'dockerfile': '🐳',
      'gitignore': '🚫',
      'env': '🔐',
    };
    
    return iconMap[ext] || '📄';
  }
  
  show() {
    this.isActive = true;
    this.selectedIndex = 0;
    this.dropdown.style.display = 'block';
    this.positionDropdown();
  }
  
  hide() {
    this.isActive = false;
    this.dropdown.style.display = 'none';
    this.currentQuery = '';
    this.triggerPosition = null;
  }
  
  positionDropdown() {
    // Get editor position and dimensions
    const editorDom = this.editor.getDomNode();
    const editorRect = editorDom.getBoundingClientRect();
    
    // Get cursor position in pixels
    const position = this.editor.getPosition();
    const cursorCoords = this.editor.getScrolledVisiblePosition(position);
    
    if (cursorCoords) {
      // Position dropdown below the cursor
      const left = editorRect.left + cursorCoords.left;
      const top = editorRect.top + cursorCoords.top + cursorCoords.height;
      
      this.dropdown.style.left = `${left}px`;
      this.dropdown.style.top = `${top}px`;
      
      // Ensure dropdown doesn't go off screen
      const dropdownRect = this.dropdown.getBoundingClientRect();
      if (dropdownRect.right > window.innerWidth) {
        this.dropdown.style.left = `${window.innerWidth - dropdownRect.width - 10}px`;
      }
      if (dropdownRect.bottom > window.innerHeight) {
        // Show above cursor instead
        this.dropdown.style.top = `${editorRect.top + cursorCoords.top - dropdownRect.height}px`;
      }
    }
  }
  
  selectNext() {
    const items = this.dropdown.querySelectorAll('.file-mention-item');
    if (this.selectedIndex < items.length - 1) {
      this.selectedIndex++;
      this.updateSelection();
    }
  }
  
  selectPrevious() {
    if (this.selectedIndex > 0) {
      this.selectedIndex--;
      this.updateSelection();
    }
  }
  
  updateSelection() {
    const items = this.dropdown.querySelectorAll('.file-mention-item');
    items.forEach((item, index) => {
      if (index === this.selectedIndex) {
        item.classList.add('selected');
        // Scroll into view if needed
        item.scrollIntoView({ block: 'nearest' });
      } else {
        item.classList.remove('selected');
      }
    });
  }
  
  insertSelectedFile() {
    const selectedItem = this.dropdown.querySelector(`.file-mention-item[data-index="${this.selectedIndex}"]`);
    if (!selectedItem) return;
    
    const filePath = selectedItem.dataset.path;
    
    // Replace the @ mention with the file path
    const model = this.editor.getModel();
    const endPosition = this.editor.getPosition();
    
    const range = new monaco.Range(
      this.triggerPosition.lineNumber,
      this.triggerPosition.column,
      endPosition.lineNumber,
      endPosition.column
    );
    
    // Insert the file path wrapped in backticks for clarity
    const edit = {
      range: range,
      text: `\`${filePath}\` `,
      forceMoveMarkers: true
    };
    
    this.editor.executeEdits('file-mention', [edit]);
    
    // Hide the dropdown
    this.hide();
    
    // Focus back on editor
    this.editor.focus();
  }
}

// Export for use in main UI
window.FileMentionSystem = FileMentionSystem;