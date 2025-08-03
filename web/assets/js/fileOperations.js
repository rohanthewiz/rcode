// File Operations Module - handles create, rename, delete operations
const FileOperations = (function() {
    
    // API calls to backend
    async function createFile(path, type, content = '') {
        try {
            const response = await fetch('/api/files/create', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ path, type, content })
            });
            
            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to create');
            }
            
            return await response.json();
        } catch (error) {
            console.error('Create error:', error);
            throw error;
        }
    }
    
    async function renameFile(oldPath, newName) {
        try {
            const response = await fetch('/api/files/rename', {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ oldPath, newName })
            });
            
            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to rename');
            }
            
            return await response.json();
        } catch (error) {
            console.error('Rename error:', error);
            throw error;
        }
    }
    
    async function deleteFile(path) {
        try {
            const response = await fetch('/api/files/delete', {
                method: 'DELETE',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ path })
            });
            
            if (!response.ok) {
                const error = await response.json();
                throw new Error(error.error || 'Failed to delete');
            }
            
            return await response.json();
        } catch (error) {
            console.error('Delete error:', error);
            throw error;
        }
    }
    
    // Dialog functions
    function showCreateDialog(parentPath = '') {
        return new Promise((resolve, reject) => {
            // Remove any existing dialog
            const existing = document.getElementById('file-operation-dialog');
            if (existing) existing.remove();
            
            // Create dialog HTML
            const dialog = document.createElement('div');
            dialog.id = 'file-operation-dialog';
            dialog.className = 'modal-overlay';
            dialog.innerHTML = `
                <div class="modal-dialog">
                    <div class="modal-header">
                        <h3>Create New</h3>
                        <button class="close-btn" data-action="cancel">&times;</button>
                    </div>
                    <div class="modal-body">
                        <div class="form-group">
                            <label>Type:</label>
                            <div class="radio-group">
                                <label>
                                    <input type="radio" name="type" value="file" checked>
                                    <span>📄 File</span>
                                </label>
                                <label>
                                    <input type="radio" name="type" value="directory">
                                    <span>📁 Folder</span>
                                </label>
                            </div>
                        </div>
                        <div class="form-group">
                            <label for="name-input">Name:</label>
                            <input type="text" id="name-input" class="form-input" placeholder="Enter name..." autofocus>
                            <div class="help-text">Path: ${parentPath ? parentPath + '/' : ''}<span id="name-preview"></span></div>
                        </div>
                        <div class="form-group" id="content-group" style="display: none;">
                            <label for="content-input">Initial Content (optional):</label>
                            <textarea id="content-input" class="form-input" rows="10" placeholder="// Initial file content"></textarea>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" data-action="cancel">Cancel</button>
                        <button class="btn btn-primary" data-action="create">Create</button>
                    </div>
                </div>
            `;
            
            document.body.appendChild(dialog);
            
            // Get elements
            const nameInput = dialog.querySelector('#name-input');
            const namePreview = dialog.querySelector('#name-preview');
            const contentGroup = dialog.querySelector('#content-group');
            const contentInput = dialog.querySelector('#content-input');
            const typeRadios = dialog.querySelectorAll('input[name="type"]');
            
            // Update preview as user types
            nameInput.addEventListener('input', () => {
                namePreview.textContent = nameInput.value;
            });
            
            // Show/hide content field based on type
            typeRadios.forEach(radio => {
                radio.addEventListener('change', () => {
                    contentGroup.style.display = radio.value === 'file' ? 'block' : 'none';
                });
            });
            
            // Handle button clicks
            dialog.addEventListener('click', async (e) => {
                const action = e.target.dataset.action;
                
                if (action === 'cancel') {
                    dialog.remove();
                    reject(new Error('Cancelled'));
                } else if (action === 'create') {
                    const name = nameInput.value.trim();
                    if (!name) {
                        alert('Please enter a name');
                        return;
                    }
                    
                    const type = dialog.querySelector('input[name="type"]:checked').value;
                    const content = type === 'file' ? contentInput.value : '';
                    const fullPath = parentPath ? `${parentPath}/${name}` : name;
                    
                    try {
                        const result = await createFile(fullPath, type, content);
                        dialog.remove();
                        resolve(result);
                    } catch (error) {
                        alert(`Failed to create: ${error.message}`);
                    }
                }
            });
            
            // Handle Enter key
            nameInput.addEventListener('keydown', (e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    dialog.querySelector('[data-action="create"]').click();
                }
            });
            
            // Handle Escape key
            dialog.addEventListener('keydown', (e) => {
                if (e.key === 'Escape') {
                    dialog.remove();
                    reject(new Error('Cancelled'));
                }
            });
            
            // Focus name input
            nameInput.focus();
        });
    }
    
    function showRenameDialog(oldPath, oldName) {
        return new Promise((resolve, reject) => {
            // Remove any existing dialog
            const existing = document.getElementById('file-operation-dialog');
            if (existing) existing.remove();
            
            // Create dialog HTML
            const dialog = document.createElement('div');
            dialog.id = 'file-operation-dialog';
            dialog.className = 'modal-overlay';
            dialog.innerHTML = `
                <div class="modal-dialog">
                    <div class="modal-header">
                        <h3>Rename</h3>
                        <button class="close-btn" data-action="cancel">&times;</button>
                    </div>
                    <div class="modal-body">
                        <div class="form-group">
                            <label for="name-input">New Name:</label>
                            <input type="text" id="name-input" class="form-input" value="${oldName}" autofocus>
                            <div class="help-text">Current: ${oldName}</div>
                        </div>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" data-action="cancel">Cancel</button>
                        <button class="btn btn-primary" data-action="rename">Rename</button>
                    </div>
                </div>
            `;
            
            document.body.appendChild(dialog);
            
            // Get name input
            const nameInput = dialog.querySelector('#name-input');
            
            // Select all text on focus
            nameInput.select();
            
            // Handle button clicks
            dialog.addEventListener('click', async (e) => {
                const action = e.target.dataset.action;
                
                if (action === 'cancel') {
                    dialog.remove();
                    reject(new Error('Cancelled'));
                } else if (action === 'rename') {
                    const newName = nameInput.value.trim();
                    if (!newName) {
                        alert('Please enter a name');
                        return;
                    }
                    
                    if (newName === oldName) {
                        dialog.remove();
                        reject(new Error('No change'));
                        return;
                    }
                    
                    try {
                        const result = await renameFile(oldPath, newName);
                        dialog.remove();
                        resolve(result);
                    } catch (error) {
                        alert(`Failed to rename: ${error.message}`);
                    }
                }
            });
            
            // Handle Enter key
            nameInput.addEventListener('keydown', (e) => {
                if (e.key === 'Enter') {
                    e.preventDefault();
                    dialog.querySelector('[data-action="rename"]').click();
                }
            });
            
            // Handle Escape key
            dialog.addEventListener('keydown', (e) => {
                if (e.key === 'Escape') {
                    dialog.remove();
                    reject(new Error('Cancelled'));
                }
            });
        });
    }
    
    function showDeleteDialog(path, name, isDir) {
        return new Promise((resolve, reject) => {
            // Remove any existing dialog
            const existing = document.getElementById('file-operation-dialog');
            if (existing) existing.remove();
            
            // Create dialog HTML
            const dialog = document.createElement('div');
            dialog.id = 'file-operation-dialog';
            dialog.className = 'modal-overlay';
            dialog.innerHTML = `
                <div class="modal-dialog">
                    <div class="modal-header">
                        <h3>Confirm Delete</h3>
                        <button class="close-btn" data-action="cancel">&times;</button>
                    </div>
                    <div class="modal-body">
                        <p>Are you sure you want to delete ${isDir ? 'folder' : 'file'}:</p>
                        <p class="file-path"><strong>${name}</strong></p>
                        ${isDir ? '<p class="warning-text">⚠️ This will delete all files and subdirectories inside this folder!</p>' : ''}
                        <p class="warning-text">This action cannot be undone.</p>
                    </div>
                    <div class="modal-footer">
                        <button class="btn btn-secondary" data-action="cancel">Cancel</button>
                        <button class="btn btn-danger" data-action="delete">Delete</button>
                    </div>
                </div>
            `;
            
            document.body.appendChild(dialog);
            
            // Handle button clicks
            dialog.addEventListener('click', async (e) => {
                const action = e.target.dataset.action;
                
                if (action === 'cancel') {
                    dialog.remove();
                    reject(new Error('Cancelled'));
                } else if (action === 'delete') {
                    try {
                        const result = await deleteFile(path);
                        dialog.remove();
                        resolve(result);
                    } catch (error) {
                        alert(`Failed to delete: ${error.message}`);
                    }
                }
            });
            
            // Handle Escape key
            dialog.addEventListener('keydown', (e) => {
                if (e.key === 'Escape') {
                    dialog.remove();
                    reject(new Error('Cancelled'));
                }
            });
            
            // Focus delete button
            dialog.querySelector('[data-action="delete"]').focus();
        });
    }
    
    // Public API
    return {
        createFile,
        renameFile,
        deleteFile,
        showCreateDialog,
        showRenameDialog,
        showDeleteDialog
    };
})();

// Export for use in other modules
window.FileOperations = FileOperations;