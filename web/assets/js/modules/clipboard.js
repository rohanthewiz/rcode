// clipboard.js - Clipboard and paste handling
// This module handles clipboard operations, image pasting, and drag-and-drop

// Setup clipboard handling for the editor
export function setupClipboardHandling(editor, pastedImages) {
  if (!editor) return;
  
  // Handle paste events
  editor.addEventListener('paste', async (e) => {
    const items = e.clipboardData?.items;
    if (!items) return;
    
    let hasImage = false;
    
    for (const item of items) {
      // Check if the item is an image
      if (item.type.startsWith('image/')) {
        hasImage = true;
        e.preventDefault(); // Prevent default paste behavior for images
        
        const blob = item.getAsFile();
        if (blob) {
          await processImageBlob(blob, item.type, editor, pastedImages);
        }
      }
    }
    
    // If no image was found, let the default paste behavior handle text
    if (!hasImage) {
      // Text will be handled by default behavior
      return;
    }
  });
  
  // Handle keyboard shortcuts
  editor.addEventListener('keydown', (e) => {
    // Cmd/Ctrl + V is already handled by paste event
    // Add any additional keyboard shortcuts here if needed
  });
}

// Process image blob from clipboard or file
async function processImageBlob(blob, mimeType, editor, pastedImages) {
  try {
    // Generate a unique filename
    const timestamp = Date.now();
    const extension = mimeType.split('/')[1] || 'png';
    const filename = `pasted_image_${timestamp}.${extension}`;
    
    // Read the blob as base64
    const reader = new FileReader();
    reader.onload = async function(e) {
      const base64Data = e.target.result;
      
      // Store the image data
      const imageData = {
        filename: filename,
        mimeType: mimeType,
        data: base64Data,
        size: blob.size,
        timestamp: timestamp
      };
      
      // Add to pastedImages array
      pastedImages.push(imageData);
      
      // Create a reference to insert into the editor
      const imageRef = `![${filename}](pasted:${filename})`;
      
      // Get current cursor position
      const currentValue = editor.value;
      const cursorPos = editor.selectionStart;
      
      // Insert the image reference at cursor position
      const newValue = 
        currentValue.substring(0, cursorPos) + 
        imageRef + 
        currentValue.substring(cursorPos);
      
      editor.value = newValue;
      
      // Move cursor after the inserted text
      const newCursorPos = cursorPos + imageRef.length;
      editor.setSelectionRange(newCursorPos, newCursorPos);
      
      // Focus back on the editor
      editor.focus();
      
      // Show notification
      showImagePastedNotification(mimeType, blob.size, filename);
      
      // Log for debugging
      console.log('Image pasted:', {
        filename: filename,
        size: blob.size,
        type: mimeType
      });
    };
    
    reader.onerror = function(error) {
      console.error('Error reading image:', error);
      showNotification('Failed to process image', 'error');
    };
    
    // Start reading the blob
    reader.readAsDataURL(blob);
    
  } catch (error) {
    console.error('Error processing image:', error);
    showNotification('Failed to paste image', 'error');
  }
}

// Show notification when image is pasted
function showImagePastedNotification(mimeType, size, filename) {
  // Format size for display
  const sizeKB = (size / 1024).toFixed(2);
  const sizeText = sizeKB > 1024 
    ? `${(sizeKB / 1024).toFixed(2)} MB` 
    : `${sizeKB} KB`;
  
  // Create notification element
  const notification = document.createElement('div');
  notification.className = 'image-paste-notification';
  notification.innerHTML = `
    <div class="notification-content">
      <span class="notification-icon">📎</span>
      <div class="notification-text">
        <div class="notification-title">Image pasted successfully</div>
        <div class="notification-details">${filename} (${sizeText})</div>
      </div>
    </div>
  `;
  
  // Add to page
  document.body.appendChild(notification);
  
  // Animate in
  setTimeout(() => {
    notification.classList.add('show');
  }, 10);
  
  // Remove after 3 seconds
  setTimeout(() => {
    notification.classList.remove('show');
    setTimeout(() => {
      notification.remove();
    }, 300);
  }, 3000);
}

// Setup drag and drop for the editor
export function setupDragAndDrop(editor, pastedImages) {
  if (!editor) return;
  
  const dropZone = editor.parentElement;
  
  // Prevent default drag behaviors
  ['dragenter', 'dragover', 'dragleave', 'drop'].forEach(eventName => {
    dropZone.addEventListener(eventName, preventDefaults, false);
    document.body.addEventListener(eventName, preventDefaults, false);
  });
  
  // Highlight drop zone when item is dragged over it
  ['dragenter', 'dragover'].forEach(eventName => {
    dropZone.addEventListener(eventName, highlight, false);
  });
  
  ['dragleave', 'drop'].forEach(eventName => {
    dropZone.addEventListener(eventName, unhighlight, false);
  });
  
  // Handle dropped files
  dropZone.addEventListener('drop', handleDrop, false);
  
  function preventDefaults(e) {
    e.preventDefault();
    e.stopPropagation();
  }
  
  function highlight(e) {
    dropZone.classList.add('drag-highlight');
  }
  
  function unhighlight(e) {
    dropZone.classList.remove('drag-highlight');
  }
  
  function handleDrop(e) {
    const dt = e.dataTransfer;
    const files = dt.files;
    
    handleFiles(files, editor, pastedImages);
  }
}

// Handle dropped or selected files
function handleFiles(files, editor, pastedImages) {
  ([...files]).forEach(file => {
    if (file.type.startsWith('image/')) {
      processImageFile(file, editor, pastedImages);
    }
  });
}

// Process image file
function processImageFile(file, editor, pastedImages) {
  processImageBlob(file, file.type, editor, pastedImages);
}

// Show generic notification
function showNotification(message, type = 'info') {
  const notification = document.createElement('div');
  notification.className = `notification notification-${type}`;
  notification.textContent = message;
  
  document.body.appendChild(notification);
  
  setTimeout(() => {
    notification.classList.add('show');
  }, 10);
  
  setTimeout(() => {
    notification.classList.remove('show');
    setTimeout(() => {
      notification.remove();
    }, 300);
  }, 3000);
}