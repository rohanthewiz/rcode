/**
 * Clipboard Module - Handles image paste and drag-drop operations
 * Provides functionality for pasting and dropping images into the editor
 */

// Setup clipboard and drag-drop handling for images
function setupClipboardHandling(editor) {
  // Variable to store pasted/dropped images
  let pastedImages = [];
  
  // Monaco Editor paste event handling
  const editorContainer = editor.getDomNode();
  if (editorContainer) {
    // Add paste event listener with capture to intercept before Monaco
    editorContainer.addEventListener('paste', async function(e) {
      // Check if clipboard contains an image
      const clipboardData = e.clipboardData || window.clipboardData;
      if (clipboardData && clipboardData.items) {
        for (let i = 0; i < clipboardData.items.length; i++) {
          const item = clipboardData.items[i];
          if (item.type.indexOf('image') !== -1) {
            // Handle image paste
            e.preventDefault(); // Prevent default Monaco paste
            e.stopPropagation(); // Stop event propagation
            handlePasteEvent(e, editor, pastedImages);
            return;
          }
        }
      }
      // Let Monaco handle non-image pastes normally
    }, true); // Use capture phase to intercept before Monaco
    
    // Also add a contextmenu handler to enable paste for images
    editorContainer.addEventListener('contextmenu', function(e) {
      // Check if clipboard might have an image
      // We can't directly check clipboard on contextmenu due to security
      // but we can ensure our paste handler is ready
      setTimeout(() => {
        // Focus the editor after context menu to ensure paste events work
        editor.focus();
      }, 100);
    });
    
    // Add keyboard shortcut handler for Cmd/Ctrl+V
    editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyV, function() {
      // Trigger a synthetic paste event
      const pasteEvent = new ClipboardEvent('paste', {
        clipboardData: new DataTransfer(),
        bubbles: true,
        cancelable: true
      });
      
      // Try to read from clipboard using async API if available
      if (navigator.clipboard && navigator.clipboard.read) {
        navigator.clipboard.read().then(items => {
          for (const item of items) {
            for (const type of item.types) {
              if (type.startsWith('image/')) {
                item.getType(type).then(blob => {
                  // Process the image blob
                  processImageBlob(blob, type, editor, pastedImages);
                });
                return false; // Prevent default paste
              }
            }
          }
        }).catch(err => {
          console.log('Clipboard API not available or permission denied:', err);
          // Fall back to letting default paste happen
        });
        return false; // Prevent default behavior while we process
      }
    });
  }
  
  // Also handle paste on the document for when editor might not have focus
  document.addEventListener('paste', async function(e) {
    // Only handle if we're in the main chat area but not in the editor itself
    const chatArea = document.getElementById('chat-area');
    const editorContainer = editor.getDomNode();
    if (chatArea && chatArea.contains(e.target) && !editorContainer.contains(e.target)) {
      handlePasteEvent(e, editor, pastedImages);
    }
  });
  
  // Setup drag and drop handling
  setupDragAndDrop(editor, pastedImages);
  
  // Store pasted images reference on the editor for access later
  editor.pastedImages = pastedImages;
}

// Helper function to process image blob
function processImageBlob(blob, mimeType, editor, pastedImages) {
  const reader = new FileReader();
  reader.onload = function(event) {
    const base64Data = event.target.result;
    // Remove the data URL prefix to get just the base64 string
    const base64String = base64Data.split(',')[1];
    
    // Store the image data
    const imageData = {
      type: 'image',
      mediaType: mimeType,
      data: base64String,
      timestamp: Date.now()
    };
    // Clear any previous images and add only the new one
    // This ensures we're only sending the most recently pasted image
    pastedImages.length = 0;
    pastedImages.push(imageData);
    
    // Add a visual indicator to the editor
    const currentValue = editor.getValue();
    // Use a clearer message that won't confuse the AI into thinking it should use clipboard tools
    const imageIndicator = currentValue ? `\n[Image attached - ${(blob.size / 1024).toFixed(1)}KB]` : `Here's an image (${(blob.size / 1024).toFixed(1)}KB):`;
    editor.setValue(currentValue + imageIndicator);
    
    // Show a notification
    showImagePastedNotification(mimeType, blob.size);
  };
  reader.readAsDataURL(blob);
}

// Handle paste events
async function handlePasteEvent(e, editor, pastedImages) {
  const clipboardData = e.clipboardData || window.clipboardData;
  if (!clipboardData) return;
  
  const items = clipboardData.items;
  if (!items) return;
  
  // Check for images in clipboard
  for (let i = 0; i < items.length; i++) {
    const item = items[i];
    
    // Check if the item is an image
    if (item.type.indexOf('image') !== -1) {
      e.preventDefault(); // Prevent default paste behavior
      
      const blob = item.getAsFile();
      if (!blob) continue;
      
      // Convert blob to base64
      const reader = new FileReader();
      reader.onload = function(event) {
        const base64Data = event.target.result;
        // Remove the data URL prefix to get just the base64 string
        const base64String = base64Data.split(',')[1];
        
        // Store the image data
        const imageData = {
          type: 'image',
          mediaType: item.type,
          data: base64String,
          timestamp: Date.now()
        };
        // Clear any previous images and add only the new one
        // This ensures we're only sending the most recently pasted image
        pastedImages.length = 0;
        pastedImages.push(imageData);
        
        // Add a visual indicator to the editor
        const currentValue = editor.getValue();
        // Use a clearer message that won't confuse the AI
        const imageIndicator = currentValue ? `\n[Image attached - ${(blob.size / 1024).toFixed(1)}KB]` : `Here's an image (${(blob.size / 1024).toFixed(1)}KB):`;
        editor.setValue(currentValue + imageIndicator);
        
        // Show a notification
        showImagePastedNotification(item.type, blob.size);
      };
      reader.readAsDataURL(blob);
    }
  }
}

// Show notification when image is pasted or dropped
function showImagePastedNotification(mimeType, size, filename) {
  const messagesContainer = document.getElementById('messages');
  const notification = document.createElement('div');
  notification.className = 'image-paste-notification';
  notification.style.cssText = `
    position: fixed;
    bottom: 200px;
    right: 20px;
    background: var(--success);
    color: white;
    padding: 10px 15px;
    border-radius: 4px;
    box-shadow: 0 2px 10px rgba(0,0,0,0.3);
    z-index: 1000;
    animation: slideIn 0.3s ease-out;
  `;
  const action = filename ? 'attached' : 'pasted';
  const name = filename ? ` (${filename})` : '';
  notification.textContent = `Image ${action}${name} - ${(size / 1024).toFixed(1)}KB`;
  
  document.body.appendChild(notification);
  
  // Remove notification after 3 seconds
  setTimeout(() => {
    notification.style.animation = 'slideOut 0.3s ease-out';
    setTimeout(() => notification.remove(), 300);
  }, 3000);
}

// Setup drag and drop for images
function setupDragAndDrop(editor, pastedImages) {
  const chatArea = document.getElementById('chat-area');
  if (!chatArea) return;
  
  // Create drop zone overlay
  const dropZone = document.createElement('div');
  dropZone.id = 'drop-zone';
  dropZone.className = 'drop-zone';
  dropZone.innerHTML = `
    <div class="drop-zone-content">
      <div class="drop-icon">📎</div>
      <div class="drop-text">Drop images here to attach</div>
    </div>
  `;
  dropZone.style.display = 'none';
  chatArea.appendChild(dropZone);
  
  // Prevent default drag behaviors
  ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
    chatArea.addEventListener(eventName, preventDefaults, false);
    document.body.addEventListener(eventName, preventDefaults, false);
  });
  
  function preventDefaults(e) {
    e.preventDefault();
    e.stopPropagation();
  }
  
  // Highlight drop zone when item is dragged over
  ['dragenter', 'dragover'].forEach(eventName => {
    chatArea.addEventListener(eventName, highlight, false);
  });
  
  ['dragleave', 'drop'].forEach(eventName => {
    chatArea.addEventListener(eventName, unhighlight, false);
  });
  
  function highlight(e) {
    dropZone.style.display = 'flex';
    chatArea.classList.add('drag-over');
  }
  
  function unhighlight(e) {
    dropZone.style.display = 'none';
    chatArea.classList.remove('drag-over');
  }
  
  // Handle dropped files
  chatArea.addEventListener('drop', handleDrop, false);
  
  function handleDrop(e) {
    const dt = e.dataTransfer;
    const files = dt.files;
    
    handleFiles(files, editor, pastedImages);
  }
}

// Handle dropped files
function handleFiles(files, editor, pastedImages) {
  ([...files]).forEach(file => {
    if (file.type.startsWith('image/')) {
      processImageFile(file, editor, pastedImages);
    }
  });
}

// Process image file
function processImageFile(file, editor, pastedImages) {
  const reader = new FileReader();
  
  reader.onload = function(e) {
    const base64Data = e.target.result;
    // Remove the data URL prefix to get just the base64 string
    const base64String = base64Data.split(',')[1];
    
    // Store the image data
    const imageData = {
      type: 'image',
      mediaType: file.type,
      data: base64String,
      filename: file.name,
      timestamp: Date.now()
    };
    pastedImages.push(imageData);
    
    // Add a visual indicator to the editor
    const currentValue = editor.getValue();
    const imageIndicator = `\n[Image attached: ${file.name} - ${(file.size / 1024).toFixed(1)}KB]`;
    editor.setValue(currentValue + imageIndicator);
    
    // Show a notification
    showImagePastedNotification(file.type, file.size, file.name);
  };
  
  reader.readAsDataURL(file);
}