// File Browser with Context Menu functionality

class FileBrowser {
    constructor() {
        this.selectedFiles = new Set();
        this.clipboard = {
            mode: null, // 'cut' or 'copy'
            files: []
        };
        this.currentPath = '/';
        this.sessionId = null;
        this.contextMenu = null;
        this.init();
    }

    init() {
        // Create context menu element
        this.createContextMenu();
        
        // Get current session ID
        this.sessionId = window.currentSessionId || localStorage.getItem('currentSessionId');
        
        // Setup event listeners
        this.setupEventListeners();
        
        // Load initial file list
        this.loadFiles(this.currentPath);
    }

    createContextMenu() {
        // Remove existing context menu if any
        const existing = document.getElementById('file-context-menu');
        if (existing) existing.remove();

        // Create context menu HTML
        const menu = document.createElement('div');
        menu.id = 'file-context-menu';
        menu.className = 'context-menu';
        menu.innerHTML = `
            <div class="context-menu-item" data-action="cut">
                <span class="menu-icon">✂️</span>
                Cut
                <span class="menu-shortcut">Ctrl+X</span>
            </div>
            <div class="context-menu-item" data-action="copy">
                <span class="menu-icon">📋</span>
                Copy
                <span class="menu-shortcut">Ctrl+C</span>
            </div>
            <div class="context-menu-item" data-action="paste">
                <span class="menu-icon">📌</span>
                Paste
                <span class="menu-shortcut">Ctrl+V</span>
            </div>
            <div class="context-menu-separator"></div>
            <div class="context-menu-item" data-action="delete">
                <span class="menu-icon">🗑️</span>
                Delete
                <span class="menu-shortcut">Del</span>
            </div>
            <div class="context-menu-separator"></div>
            <div class="context-menu-item" data-action="rename">
                <span class="menu-icon">✏️</span>
                Rename
                <span class="menu-shortcut">F2</span>
            </div>
            <div class="context-menu-item" data-action="new-file">
                <span class="menu-icon">📄</span>
                New File
            </div>
            <div class="context-menu-item" data-action="new-folder">
                <span class="menu-icon">📁</span>
                New Folder
            </div>
            <div class="context-menu-separator"></div>
            <div class="context-menu-item" data-action="zip">
                <span class="menu-icon">🗜️</span>
                Create Zip...
            </div>
            <div class="context-menu-separator"></div>
            <div class="context-menu-item" data-action="refresh">
                <span class="menu-icon">🔄</span>
                Refresh
            </div>
        `;
        document.body.appendChild(menu);
        this.contextMenu = menu;
    }

    setupEventListeners() {
        // Right-click on file tree
        document.addEventListener('contextmenu', (e) => {
            const fileNode = e.target.closest('.tree-node');
            if (fileNode) {
                e.preventDefault();
                this.showContextMenu(e.pageX, e.pageY, fileNode);
            }
        });

        // Click on context menu items
        this.contextMenu.addEventListener('click', (e) => {
            const item = e.target.closest('.context-menu-item');
            if (item) {
                const action = item.dataset.action;
                this.handleContextMenuAction(action);
            }
        });

        // Hide context menu on click outside
        document.addEventListener('click', (e) => {
            if (!e.target.closest('#file-context-menu')) {
                this.hideContextMenu();
            }
        });

        // Keyboard shortcuts
        document.addEventListener('keydown', (e) => {
            if (e.target.closest('.file-tree')) {
                this.handleKeyboardShortcut(e);
            }
        });

        // File selection
        document.addEventListener('click', (e) => {
            const fileNode = e.target.closest('.tree-node');
            if (fileNode && !e.target.closest('[data-action="toggle"]')) {
                this.handleFileSelection(fileNode, e);
            }
        });
    }

    showContextMenu(x, y, fileNode) {
        // Store the context file
        this.contextFile = fileNode.dataset.path;
        
        // Update menu state based on clipboard
        this.updateContextMenuState();
        
        // Position and show menu
        this.contextMenu.style.display = 'block';
        this.contextMenu.style.left = x + 'px';
        this.contextMenu.style.top = y + 'px';
        
        // Adjust position if menu goes off-screen
        const rect = this.contextMenu.getBoundingClientRect();
        if (rect.right > window.innerWidth) {
            this.contextMenu.style.left = (window.innerWidth - rect.width - 10) + 'px';
        }
        if (rect.bottom > window.innerHeight) {
            this.contextMenu.style.top = (window.innerHeight - rect.height - 10) + 'px';
        }
    }

    hideContextMenu() {
        if (this.contextMenu) {
            this.contextMenu.style.display = 'none';
        }
    }

    updateContextMenuState() {
        const pasteItem = this.contextMenu.querySelector('[data-action="paste"]');
        if (pasteItem) {
            if (this.clipboard.files.length === 0) {
                pasteItem.classList.add('disabled');
            } else {
                pasteItem.classList.remove('disabled');
            }
        }
    }

    handleFileSelection(fileNode, event) {
        const path = fileNode.dataset.path;
        
        if (event.ctrlKey || event.metaKey) {
            // Multi-select with Ctrl/Cmd
            if (this.selectedFiles.has(path)) {
                this.selectedFiles.delete(path);
                fileNode.classList.remove('selected');
            } else {
                this.selectedFiles.add(path);
                fileNode.classList.add('selected');
            }
        } else if (event.shiftKey && this.lastSelectedFile) {
            // Range select with Shift
            this.selectRange(this.lastSelectedFile, path);
        } else {
            // Single select
            this.clearSelection();
            this.selectedFiles.add(path);
            fileNode.classList.add('selected');
        }
        
        this.lastSelectedFile = path;
    }

    clearSelection() {
        document.querySelectorAll('.tree-node.selected').forEach(node => {
            node.classList.remove('selected');
        });
        this.selectedFiles.clear();
    }

    selectRange(startPath, endPath) {
        const nodes = Array.from(document.querySelectorAll('.tree-node'));
        const startIndex = nodes.findIndex(n => n.dataset.path === startPath);
        const endIndex = nodes.findIndex(n => n.dataset.path === endPath);
        
        if (startIndex !== -1 && endIndex !== -1) {
            const [from, to] = startIndex < endIndex ? [startIndex, endIndex] : [endIndex, startIndex];
            
            this.clearSelection();
            for (let i = from; i <= to; i++) {
                const path = nodes[i].dataset.path;
                this.selectedFiles.add(path);
                nodes[i].classList.add('selected');
            }
        }
    }

    handleKeyboardShortcut(e) {
        if (e.ctrlKey || e.metaKey) {
            switch(e.key.toLowerCase()) {
                case 'c':
                    e.preventDefault();
                    this.copyFiles();
                    break;
                case 'x':
                    e.preventDefault();
                    this.cutFiles();
                    break;
                case 'v':
                    e.preventDefault();
                    this.pasteFiles();
                    break;
                case 'a':
                    e.preventDefault();
                    this.selectAll();
                    break;
            }
        } else if (e.key === 'Delete') {
            e.preventDefault();
            this.deleteFiles();
        } else if (e.key === 'F2') {
            e.preventDefault();
            this.renameFile();
        }
    }

    handleContextMenuAction(action) {
        this.hideContextMenu();
        
        switch(action) {
            case 'cut':
                this.cutFiles();
                break;
            case 'copy':
                this.copyFiles();
                break;
            case 'paste':
                this.pasteFiles();
                break;
            case 'delete':
                this.deleteFiles();
                break;
            case 'rename':
                this.renameFile();
                break;
            case 'new-file':
                this.createNewFile();
                break;
            case 'new-folder':
                this.createNewFolder();
                break;
            case 'zip':
                this.createZip();
                break;
            case 'refresh':
                this.refreshFiles();
                break;
        }
    }

    async copyFiles() {
        const paths = this.getSelectedPaths();
        if (paths.length === 0) return;

        try {
            const response = await fetch('/api/files/copy', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Session-ID': this.sessionId
                },
                body: JSON.stringify({ paths })
            });

            const result = await response.json();
            if (response.ok) {
                this.clipboard.mode = 'copy';
                this.clipboard.files = paths;
                this.showNotification(`Copied ${result.count} item(s) to clipboard`, 'success');
            } else {
                this.showNotification(result.error || 'Failed to copy files', 'error');
            }
        } catch (error) {
            console.error('Copy error:', error);
            this.showNotification('Failed to copy files', 'error');
        }
    }

    async cutFiles() {
        const paths = this.getSelectedPaths();
        if (paths.length === 0) return;

        try {
            const response = await fetch('/api/files/cut', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Session-ID': this.sessionId
                },
                body: JSON.stringify({ paths })
            });

            const result = await response.json();
            if (response.ok) {
                this.clipboard.mode = 'cut';
                this.clipboard.files = paths;
                // Mark files as cut visually
                paths.forEach(path => {
                    const node = document.querySelector(`.tree-node[data-path="${path}"]`);
                    if (node) node.classList.add('cut');
                });
                this.showNotification(`Cut ${result.count} item(s) to clipboard`, 'success');
            } else {
                this.showNotification(result.error || 'Failed to cut files', 'error');
            }
        } catch (error) {
            console.error('Cut error:', error);
            this.showNotification('Failed to cut files', 'error');
        }
    }

    async pasteFiles() {
        if (this.clipboard.files.length === 0) {
            this.showNotification('Clipboard is empty', 'info');
            return;
        }

        // Get target directory (current context or current path)
        const targetPath = this.contextFile || this.currentPath;

        try {
            const response = await fetch('/api/files/paste', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Session-ID': this.sessionId
                },
                body: JSON.stringify({ 
                    target: targetPath,
                    overwrite: false 
                })
            });

            const result = await response.json();
            if (response.ok) {
                this.showNotification(`Pasted ${result.success} item(s)`, 'success');
                
                // Clear cut visual state
                if (this.clipboard.mode === 'cut') {
                    document.querySelectorAll('.tree-node.cut').forEach(node => {
                        node.classList.remove('cut');
                    });
                    this.clipboard.files = [];
                    this.clipboard.mode = null;
                }
                
                // Refresh file tree
                this.refreshFiles();
            } else {
                this.showNotification(result.error || 'Failed to paste files', 'error');
            }
        } catch (error) {
            console.error('Paste error:', error);
            this.showNotification('Failed to paste files', 'error');
        }
    }

    async deleteFiles() {
        const paths = this.getSelectedPaths();
        if (paths.length === 0) return;

        // Confirm deletion
        if (!confirm(`Are you sure you want to delete ${paths.length} item(s)?`)) {
            return;
        }

        try {
            const response = await fetch('/api/files', {
                method: 'DELETE',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Session-ID': this.sessionId
                },
                body: JSON.stringify({ 
                    paths,
                    recursive: true 
                })
            });

            const result = await response.json();
            if (response.ok) {
                this.showNotification(`Deleted ${result.success} item(s)`, 'success');
                this.clearSelection();
                this.refreshFiles();
            } else {
                this.showNotification(result.error || 'Failed to delete files', 'error');
            }
        } catch (error) {
            console.error('Delete error:', error);
            this.showNotification('Failed to delete files', 'error');
        }
    }

    renameFile() {
        const paths = this.getSelectedPaths();
        if (paths.length !== 1) {
            this.showNotification('Please select exactly one file to rename', 'info');
            return;
        }

        const path = paths[0];
        const fileName = path.split('/').pop();
        const newName = prompt('Enter new name:', fileName);
        
        if (newName && newName !== fileName) {
            // Use existing rename endpoint if available
            // For now, show notification
            this.showNotification('Rename functionality coming soon', 'info');
        }
    }

    createNewFile() {
        const name = prompt('Enter file name:');
        if (name) {
            // Use existing create file endpoint if available
            this.showNotification('New file functionality coming soon', 'info');
        }
    }

    createNewFolder() {
        const name = prompt('Enter folder name:');
        if (name) {
            // Use existing create folder endpoint if available
            this.showNotification('New folder functionality coming soon', 'info');
        }
    }

    async createZip() {
        const paths = this.getSelectedPaths();
        if (paths.length === 0) {
            this.showNotification('Please select files to zip', 'info');
            return;
        }

        // Create zip options dialog
        const dialog = this.createZipDialog();
        document.body.appendChild(dialog);

        // Handle dialog submission
        const form = dialog.querySelector('#zip-form');
        const cancelBtn = dialog.querySelector('#zip-cancel');
        const overlay = dialog.querySelector('.dialog-overlay');

        const closeDialog = () => {
            dialog.remove();
        };

        cancelBtn.addEventListener('click', closeDialog);
        overlay.addEventListener('click', closeDialog);

        form.addEventListener('submit', async (e) => {
            e.preventDefault();
            
            const outputName = form.querySelector('#zip-name').value;
            const excludeDotFiles = form.querySelector('#exclude-dotfiles').checked;
            const useGitignore = form.querySelector('#use-gitignore').checked;

            closeDialog();

            // Show loading notification
            this.showNotification('Creating zip archive...', 'info');

            try {
                const response = await fetch('/api/files/zip', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-Session-ID': this.sessionId
                    },
                    body: JSON.stringify({
                        paths,
                        outputName,
                        excludeDotFiles,
                        useGitignore
                    })
                });

                const result = await response.json();
                if (response.ok) {
                    this.showNotification(
                        `Created ${result.outputPath.split('/').pop()}: ${result.filesAdded} files, ${result.compression} compression`,
                        'success'
                    );
                    this.refreshFiles();
                } else {
                    this.showNotification(result.error || 'Failed to create zip', 'error');
                }
            } catch (error) {
                console.error('Zip error:', error);
                this.showNotification('Failed to create zip archive', 'error');
            }
        });
    }

    createZipDialog() {
        const selectedCount = this.getSelectedPaths().length;
        const defaultName = `archive_${Date.now()}`;
        
        const dialog = document.createElement('div');
        dialog.className = 'zip-dialog';
        dialog.innerHTML = `
            <div class="dialog-overlay"></div>
            <div class="dialog-content">
                <h3>Create Zip Archive</h3>
                <p class="dialog-info">Creating archive of ${selectedCount} selected item(s)</p>
                
                <form id="zip-form">
                    <div class="form-group">
                        <label for="zip-name">Archive Name:</label>
                        <input 
                            type="text" 
                            id="zip-name" 
                            name="zipName" 
                            value="${defaultName}" 
                            required
                            placeholder="Enter archive name"
                        />
                        <span class="form-hint">.zip extension will be added automatically</span>
                    </div>
                    
                    <div class="form-group">
                        <label class="checkbox-label">
                            <input type="checkbox" id="exclude-dotfiles" checked />
                            Exclude dot files (files starting with .)
                        </label>
                    </div>
                    
                    <div class="form-group">
                        <label class="checkbox-label">
                            <input type="checkbox" id="use-gitignore" checked />
                            Respect .gitignore rules
                        </label>
                    </div>
                    
                    <div class="dialog-buttons">
                        <button type="submit" class="btn-primary">Create Zip</button>
                        <button type="button" id="zip-cancel" class="btn-secondary">Cancel</button>
                    </div>
                </form>
            </div>
        `;
        
        return dialog;
    }

    selectAll() {
        document.querySelectorAll('.tree-node').forEach(node => {
            const path = node.dataset.path;
            this.selectedFiles.add(path);
            node.classList.add('selected');
        });
    }

    getSelectedPaths() {
        if (this.selectedFiles.size > 0) {
            return Array.from(this.selectedFiles);
        } else if (this.contextFile) {
            return [this.contextFile];
        }
        return [];
    }

    async loadFiles(path) {
        try {
            const response = await fetch(`/api/files?path=${encodeURIComponent(path)}`, {
                headers: {
                    'X-Session-ID': this.sessionId
                }
            });

            if (response.ok) {
                const data = await response.json();
                this.renderFileList(data.files);
            }
        } catch (error) {
            console.error('Failed to load files:', error);
        }
    }

    renderFileList(files) {
        // This would integrate with the existing file tree rendering
        // For now, we'll rely on the existing file tree component
        console.log('Files loaded:', files);
    }

    refreshFiles() {
        // Trigger file tree refresh
        if (window.loadFileTree) {
            window.loadFileTree();
        } else {
            this.loadFiles(this.currentPath);
        }
    }

    showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.textContent = message;
        
        // Add to body
        document.body.appendChild(notification);
        
        // Show notification
        setTimeout(() => notification.classList.add('show'), 10);
        
        // Remove after 3 seconds
        setTimeout(() => {
            notification.classList.remove('show');
            setTimeout(() => notification.remove(), 300);
        }, 3000);
    }
}

// Initialize file browser when DOM is ready
if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
        window.fileBrowser = new FileBrowser();
    });
} else {
    window.fileBrowser = new FileBrowser();
}