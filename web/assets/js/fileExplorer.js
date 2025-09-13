// File Explorer Module
//
// File Tree Refresh Mechanism:
// The file explorer automatically refreshes when file system changes occur through SSE events.
// 
// How it works:
// 1. When a file operation occurs (create/modify/delete), the server broadcasts a 'file_tree_update' SSE event
// 2. The event contains: { type: 'file_tree_update', sessionId: '...', data: { path: '...' } }
// 3. The UI checks if the sessionId matches the current session (or is empty for broadcast to all)
// 4. If matched, handleFileEvent() is called with the event data
// 5. refreshPath() is called with the path from the event:
//    - If path is empty (''), loadFileTree() refreshes the entire tree
//    - If path is not empty, it fetches and updates just that subtree
// 6. The tree is re-rendered to reflect the changes
//
// This ensures the file explorer stays in sync with file system changes made by tools or other operations.

const FileExplorer = (function() {
    let fileTree = [];
    let selectedPath = null;
    let openFolders = new Set();
    let openFiles = new Map(); // path -> {name, content, language}
    let activeFile = null;
    let fileViewerEditor = null;
    let modifiedFiles = new Set(); // Track files that have been modified
    let newFiles = new Set(); // Track files that have been newly created
    let currentDirectory = '/'; // Track the current directory being viewed

    // Initialize the file explorer
    async function init() {
        // Set up tab switching
        document.querySelectorAll('.sidebar-tab').forEach(tab => {
            tab.addEventListener('click', () => switchTab(tab.dataset.tab));
        });

        // Set up file search
        const searchInput = document.getElementById('file-search-input');
        if (searchInput) {
            searchInput.addEventListener('input', debounce(handleSearch, 300));
        }

        // Load initial file tree
        await loadFileTree();

        // Set up event delegation for tree interactions
        const treeContainer = document.getElementById('file-tree-container');
        if (treeContainer) {
            treeContainer.addEventListener('click', handleTreeClick);
            treeContainer.addEventListener('dblclick', handleTreeDoubleClick);
            treeContainer.addEventListener('contextmenu', handleTreeContextMenu);
        }

        // Create context menu
        createContextMenu();

        // Load recent files if we have a session
        if (window.currentSessionId) {
            loadRecentFiles();
        }
    }

    // Load recent files for the current session
    async function loadRecentFiles() {
        if (!window.currentSessionId) return;

        try {
            const response = await fetch(`/api/session/${window.currentSessionId}/files/recent`);
            if (response.ok) {
                const data = await response.json();
                // Could display these in a special section or highlight in tree
                console.log('Recent files:', data.files);
            }
        } catch (error) {
            console.error('Error loading recent files:', error);
        }
    }

    // Switch between tabs
    function switchTab(tabName) {
        // Update tab headers
        document.querySelectorAll('.sidebar-tab').forEach(tab => {
            tab.classList.toggle('active', tab.dataset.tab === tabName);
        });

        // Update tab content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.toggle('active', content.id === `${tabName}-tab`);
        });

        // Show/hide new session button
        const newSessionBtn = document.getElementById('new-session-btn');
        if (newSessionBtn) {
            newSessionBtn.parentElement.style.display = tabName === 'sessions' ? 'block' : 'none';
        }
        
        // Load tools when switching to tools tab
        if (tabName === 'tools' && window.currentSessionId) {
            loadSessionTools(window.currentSessionId);
        }
    }

    // Load file tree from server
    async function loadFileTree(path = '', depth = 2) {
        console.log('loadFileTree called with path:', path, 'depth:', depth);
        try {
            const url = `/api/files/tree?path=${encodeURIComponent(path)}&depth=${depth}`;
            console.log('Fetching from URL:', url);
            const response = await fetch(url);
            console.log('Response status:', response.status);
            if (!response.ok) throw new Error('Failed to load file tree');
            
            const data = await response.json();
            console.log('Received tree data:', data);
            
            if (path === '') {
                fileTree = data.children || [];
                console.log('Updated fileTree with', fileTree.length, 'items');
                // Update current directory based on the root path
                currentDirectory = data.path || await getCurrentWorkingDirectory();
                // Store the display path if provided
                if (data.displayPath) {
                    window.fileExplorerDisplayPath = data.displayPath;
                }
                updateCurrentDirectoryDisplay();
            }
            
            renderFileTree();
            console.log('Tree rendered');
            return data;
        } catch (error) {
            console.error('Error loading file tree:', error);
            showError('Failed to load file tree');
        }
    }
    
    // Get current working directory from server
    async function getCurrentWorkingDirectory() {
        try {
            const response = await fetch('/api/files/cwd');
            if (response.ok) {
                const data = await response.json();
                // Store both the actual path and display path
                if (data.displayPath) {
                    window.fileExplorerDisplayPath = data.displayPath;
                }
                return data.path || '/';
            }
        } catch (error) {
            console.error('Error getting current directory:', error);
        }
        return '/';
    }
    
    // Update the current directory display
    function updateCurrentDirectoryDisplay() {
        const dirElement = document.getElementById('current-directory-path');
        if (dirElement) {
            // Use the abbreviated display path if available
            let displayPath = window.fileExplorerDisplayPath || currentDirectory;
            
            // Further truncate if still too long (over 50 chars)
            if (displayPath.length > 50) {
                const parts = displayPath.split('/');
                if (parts.length > 4) {
                    // Keep the tilde if present
                    const prefix = displayPath.startsWith('~') ? '~' : '/';
                    const startIdx = displayPath.startsWith('~') ? 1 : 1;
                    displayPath = prefix + parts.slice(startIdx, startIdx + 1).join('/') + '/.../' + parts.slice(-2).join('/');
                }
            }
            dirElement.textContent = displayPath || '/';
            dirElement.title = currentDirectory; // Show full path on hover
        }
    }

    // Render the file tree
    function renderFileTree() {
        const container = document.getElementById('file-tree-container');
        if (!container) return;

        if (fileTree.length === 0) {
            container.innerHTML = '<div class="empty-state">No files found</div>';
            return;
        }

        const treeHtml = renderTreeNodes(fileTree, 0);
        container.innerHTML = `<div class="file-tree">${treeHtml}</div>`;
    }

    // Recursively render tree nodes
    function renderTreeNodes(nodes, depth) {
        return nodes.map(node => {
            const isOpen = openFolders.has(node.path);
            const isSelected = selectedPath === node.path;
            const indent = depth * 20;
            
            let iconClass = 'tree-icon ';
            if (node.isDir) {
                iconClass += isOpen ? 'folder-open-icon' : 'folder-icon';
            } else {
                iconClass += `file-icon file-icon-${node.icon || 'file'}`;
            }

            // Check file status (new or modified)
            const isNew = !node.isDir && newFiles.has(node.path);
            const isModified = !node.isDir && modifiedFiles.has(node.path);
            const fileStatus = isNew ? 'new' : (isModified ? 'modified' : '');
            
            let html = `
                <div class="tree-node ${isSelected ? 'selected' : ''} ${fileStatus}" 
                     data-path="${node.path}" 
                     data-is-dir="${node.isDir}"
                     style="padding-left: ${indent}px">
                    <span class="${iconClass}" data-action="toggle"></span>
                    <span class="node-name">${node.name}</span>
                    ${isNew ? '<span class="diff-indicator new" title="New file">●</span>' : 
                      isModified ? '<span class="diff-indicator modified" title="File has been modified">●</span>' : ''}
                    ${!node.isDir && node.size ? `<span class="node-size">${formatFileSize(node.size)}</span>` : ''}
                </div>
            `;

            if (node.isDir && isOpen && node.children) {
                html += `<div class="tree-children">${renderTreeNodes(node.children, depth + 1)}</div>`;
            }

            return html;
        }).join('');
    }

    // Handle tree node clicks
    async function handleTreeClick(event) {
        const node = event.target.closest('.tree-node');
        if (!node) return;

        const path = node.dataset.path;
        const isDir = node.dataset.isDir === 'true';

        // Handle folder toggle
        if (isDir && event.target.classList.contains('tree-icon')) {
            await toggleFolder(path);
            return;
        }

        // Select node
        selectNode(path);
    }

    // Handle double-click to open files or toggle folders
    async function handleTreeDoubleClick(event) {
        const node = event.target.closest('.tree-node');
        if (!node) return;

        const path = node.dataset.path;
        const isDir = node.dataset.isDir === 'true';

        if (isDir) {
            // Toggle folder open/closed on double-click
            await toggleFolder(path);
        } else {
            // Open file on double-click
            await openFile(path);
        }
    }

    // Toggle folder open/closed
    async function toggleFolder(path) {
        if (openFolders.has(path)) {
            openFolders.delete(path);
            updateFolderState(path, false);
        } else {
            openFolders.add(path);
            
            // Load children if not already loaded
            const node = findNodeByPath(fileTree, path);
            if (node && (!node.children || node.children.length === 0)) {
                const data = await loadFileTree(path, 2);
                if (data && data.children) {
                    node.children = data.children;
                    node.isOpen = true;
                }
            }
            
            updateFolderState(path, true);
        }
        
        renderFileTree();
    }

    // Update folder open/closed state
    function updateFolderState(path, isOpen) {
        const node = findNodeByPath(fileTree, path);
        if (node) {
            node.isOpen = isOpen;
        }
    }

    // Find node by path in tree
    function findNodeByPath(nodes, targetPath) {
        for (const node of nodes) {
            if (node.path === targetPath) {
                return node;
            }
            if (node.children) {
                const found = findNodeByPath(node.children, targetPath);
                if (found) return found;
            }
        }
        return null;
    }

    // Select a node
    function selectNode(path) {
        selectedPath = path;
        
        // Update visual selection
        document.querySelectorAll('.tree-node').forEach(node => {
            node.classList.toggle('selected', node.dataset.path === path);
        });
    }

    // Open a file
    async function openFile(path) {
        try {
            // Use encodeURI instead of encodeURIComponent to preserve slashes
            const response = await fetch(`/api/files/content/${encodeURI(path)}`);
            if (!response.ok) throw new Error('Failed to load file');
            
            const data = await response.json();
            
            if (data.isBinary) {
                showError('Cannot display binary files');
                return;
            }

            // Track opened file in session
            const sessionId = window.currentSessionId;
            if (sessionId) {
                await fetch(`/api/session/${sessionId}/files/open`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path })
                });
            }

            // Add to open files
            openFiles.set(path, {
                name: data.name,
                content: data.content,
                language: getLanguageFromFilename(data.name)
            });

            activeFile = path;
            showFileViewer(path);
            
        } catch (error) {
            console.error('Error opening file:', error);
            showError('Failed to open file');
        }
    }

    // Show file viewer with Monaco editor
    function showFileViewer(path) {
        const file = openFiles.get(path);
        if (!file) return;

        // Create or update file viewer
        let viewer = document.getElementById('file-viewer');
        if (!viewer) {
            // Create viewer structure
            const chatArea = document.getElementById('chat-area');
            viewer = document.createElement('div');
            viewer.id = 'file-viewer';
            viewer.className = 'file-viewer';
            viewer.innerHTML = `
                <div class="file-tabs"></div>
                <div class="file-content">
                    <div id="file-viewer-editor"></div>
                </div>
            `;
            chatArea.insertBefore(viewer, chatArea.firstChild);
        }

        // Update tabs
        updateFileTabs();
        
        // Show viewer
        viewer.classList.add('active');

        // Initialize or update Monaco editor
        if (!fileViewerEditor) {
            // Wait for Monaco to be available
            if (typeof monaco === 'undefined') {
                setTimeout(() => showFileViewer(path), 100);
                return;
            }

            fileViewerEditor = monaco.editor.create(document.getElementById('file-viewer-editor'), {
                value: file.content,
                language: file.language,
                theme: 'vs-dark',
                readOnly: true,
                minimap: { enabled: false },
                scrollBeyondLastLine: false,
                fontSize: 14,
                lineNumbers: 'on',
                renderWhitespace: 'selection',
                wordWrap: 'on'
            });

            // Handle editor resize
            window.addEventListener('resize', () => {
                if (fileViewerEditor) {
                    fileViewerEditor.layout();
                }
            });
        } else {
            // Update existing editor
            fileViewerEditor.setValue(file.content);
            monaco.editor.setModelLanguage(fileViewerEditor.getModel(), file.language);
        }

        // Layout editor
        setTimeout(() => {
            if (fileViewerEditor) {
                fileViewerEditor.layout();
            }
        }, 0);
    }

    // Update file tabs
    function updateFileTabs() {
        const tabsContainer = document.querySelector('.file-tabs');
        if (!tabsContainer) return;

        const tabsHtml = Array.from(openFiles.entries()).map(([path, file]) => {
            const isActive = path === activeFile;
            return `
                <div class="file-tab ${isActive ? 'active' : ''}" data-path="${path}">
                    <span class="tab-name">${file.name}</span>
                    <span class="tab-close" data-action="close-file">×</span>
                </div>
            `;
        }).join('');

        tabsContainer.innerHTML = tabsHtml;

        // Add event listeners
        tabsContainer.querySelectorAll('.file-tab').forEach(tab => {
            tab.addEventListener('click', (e) => {
                if (e.target.dataset.action === 'close-file') {
                    closeFile(tab.dataset.path);
                } else {
                    switchToFile(tab.dataset.path);
                }
            });
        });
    }

    // Switch to a different open file
    function switchToFile(path) {
        activeFile = path;
        showFileViewer(path);
    }

    // Close a file
    async function closeFile(path) {
        openFiles.delete(path);
        
        // Notify server that file was closed
        if (window.currentSessionId) {
            try {
                await fetch(`/api/session/${window.currentSessionId}/files/close`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ path })
                });
            } catch (error) {
                console.error('Error closing file on server:', error);
            }
        }
        
        if (openFiles.size === 0) {
            // Hide viewer if no files open
            const viewer = document.getElementById('file-viewer');
            if (viewer) {
                viewer.classList.remove('active');
            }
            activeFile = null;
        } else if (path === activeFile) {
            // Switch to another open file
            const nextFile = openFiles.keys().next().value;
            switchToFile(nextFile);
        } else {
            // Just update tabs
            updateFileTabs();
        }
    }

    // Handle file search
    async function handleSearch(event) {
        const query = event.target.value.trim();
        
        if (!query) {
            renderFileTree();
            return;
        }

        try {
            const response = await fetch('/api/files/search', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ query, searchContent: false })
            });

            if (!response.ok) throw new Error('Search failed');
            
            const data = await response.json();
            
            // Update tree with search results
            const container = document.getElementById('file-tree-container');
            if (data.results.length === 0) {
                container.innerHTML = '<div class="empty-state">No files found</div>';
            } else {
                const treeHtml = renderTreeNodes(data.results, 0);
                container.innerHTML = `<div class="file-tree">${treeHtml}</div>`;
            }
            
        } catch (error) {
            console.error('Search error:', error);
            showError('Search failed');
        }
    }

    // Get language for Monaco from filename
    function getLanguageFromFilename(filename) {
        const ext = filename.split('.').pop().toLowerCase();
        const languageMap = {
            'js': 'javascript',
            'mjs': 'javascript',
            'cjs': 'javascript',
            'jsx': 'javascript',
            'ts': 'typescript',
            'tsx': 'typescript',
            'json': 'json',
            'html': 'html',
            'htm': 'html',
            'css': 'css',
            'scss': 'scss',
            'sass': 'scss',
            'less': 'less',
            'py': 'python',
            'rb': 'ruby',
            'go': 'go',
            'rs': 'rust',
            'java': 'java',
            'c': 'c',
            'cpp': 'cpp',
            'cxx': 'cpp',
            'cc': 'cpp',
            'cs': 'csharp',
            'php': 'php',
            'sql': 'sql',
            'md': 'markdown',
            'yaml': 'yaml',
            'yml': 'yaml',
            'xml': 'xml',
            'sh': 'shell',
            'bash': 'shell',
            'dockerfile': 'dockerfile',
            'makefile': 'makefile'
        };
        
        return languageMap[ext] || 'plaintext';
    }

    // Format file size
    function formatFileSize(size) {
        const units = ['B', 'KB', 'MB', 'GB'];
        let unitIndex = 0;
        let formattedSize = size;
        
        while (formattedSize >= 1024 && unitIndex < units.length - 1) {
            formattedSize /= 1024;
            unitIndex++;
        }
        
        return `${formattedSize.toFixed(1)} ${units[unitIndex]}`;
    }

    // Show error message
    function showError(message) {
        console.error(message);
        // Could show a toast notification here
    }

    // Debounce utility
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

    // Handle file events from SSE
    function handleFileEvent(event) {
        console.log('File event received:', event);
        console.log('Event type:', event.type);
        console.log('Event data:', event.data);
        
        switch (event.type) {
            case 'file_opened':
                // Could highlight the file in the tree
                console.log('File opened:', event.data.path);
                break;
                
            case 'file_changed':
                console.log('File changed:', event.data);
                handleFileChange(event.data);
                break;
                
            case 'file_tree_update':
                console.log('Tree update needed for path:', event.data.path);
                // Refresh the tree or specific path
                refreshPath(event.data.path);
                break;
                
            default:
                console.log('Unknown file event type:', event.type);
        }
    }

    // Handle file change events
    function handleFileChange(data) {
        const { path, changeType } = data;
        
        // Handle different change types
        if (changeType === 'created') {
            markFileNew(path);
        } else if (changeType === 'modified') {
            // If file was previously new, keep it as new
            if (!newFiles.has(path)) {
                markFileModified(path);
            }
        } else if (changeType === 'deleted') {
            unmarkFileModified(path);
            unmarkFileNew(path);
        }
        
        // If the changed file is currently open, show a notification
        if (openFiles.has(path) && activeFile === path) {
            // Show notification that file was modified externally
            console.log(`Open file ${path} was ${changeType} externally`);
            // Could show a banner or dialog asking if user wants to reload
        }
        
        // Refresh the parent directory in the tree
        const parentPath = path.substring(0, path.lastIndexOf('/')) || '';
        refreshPath(parentPath);
    }

    // Refresh a specific path in the tree
    async function refreshPath(path) {
        console.log('refreshPath called with path:', path);
        
        // If it's the root or we're viewing the root, refresh everything
        if (!path || path === '') {
            console.log('Refreshing entire file tree...');
            await loadFileTree();
            console.log('File tree refresh complete');
            return;
        }
        
        // Otherwise, find and refresh the specific node
        console.log('Refreshing specific path:', path);
        const node = findNodeByPath(fileTree, path);
        if (node && node.isDir) {
            // For non-root paths, we need to fetch the subtree and update it
            try {
                const response = await fetch(`/api/files/tree?path=${encodeURIComponent(path)}&depth=2`);
                if (response.ok) {
                    const data = await response.json();
                    console.log('Fetched subtree data for path:', path, data);
                    if (data && data.children) {
                        node.children = data.children;
                        renderFileTree();
                        console.log('Subtree updated and rendered');
                    }
                }
            } catch (error) {
                console.error('Error refreshing path:', path, error);
                // Fall back to refreshing entire tree
                await loadFileTree();
            }
        } else {
            // If we can't find the node, refresh the whole tree
            console.log('Node not found, refreshing entire tree');
            await loadFileTree();
        }
    }

    // Mark a file as modified (has diff available)
    function markFileModified(path) {
        // Remove from new files if it was there
        newFiles.delete(path);
        modifiedFiles.add(path);
        renderFileTree();
    }
    
    // Mark a file as new
    function markFileNew(path) {
        // Remove from modified if it was there
        modifiedFiles.delete(path);
        newFiles.add(path);
        renderFileTree();
    }
    
    // Unmark a file as modified
    function unmarkFileModified(path) {
        modifiedFiles.delete(path);
        renderFileTree();
    }
    
    // Unmark a file as new
    function unmarkFileNew(path) {
        newFiles.delete(path);
        renderFileTree();
    }
    
    // Check if a file is modified
    function isFileModified(path) {
        return modifiedFiles.has(path);
    }
    
    // Check if a file is new
    function isFileNew(path) {
        return newFiles.has(path);
    }

    // Create context menu
    function createContextMenu() {
        // Remove existing context menu if any
        const existing = document.getElementById('file-context-menu');
        if (existing) {
            existing.remove();
        }

        // Create context menu element
        const menu = document.createElement('div');
        menu.id = 'file-context-menu';
        menu.className = 'context-menu';
        document.body.appendChild(menu);

        // Hide menu on click outside
        document.addEventListener('click', (e) => {
            if (!menu.contains(e.target)) {
                menu.classList.remove('active');
            }
        });

        // Hide menu on escape
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                menu.classList.remove('active');
            }
        });
    }

    // Handle context menu (right-click) on tree nodes
    function handleTreeContextMenu(event) {
        event.preventDefault();
        
        const node = event.target.closest('.tree-node');
        if (!node) return;

        const path = node.dataset.path;
        const isDir = node.dataset.isDir === 'true';
        const isModified = !isDir && modifiedFiles.has(path);

        // Select the node
        selectNode(path);

        // Build context menu items
        const menuItems = [];
        const nodeName = node.querySelector('.node-name')?.textContent || path.split('/').pop() || 'item';

        if (!isDir) {
            menuItems.push({
                label: 'Open',
                icon: '📄',
                action: () => openFile(path)
            });

            if (isModified) {
                menuItems.push({
                    label: 'View Changes',
                    icon: '📝',
                    action: () => viewChanges(path)
                });
            }

            menuItems.push({ separator: true });

            menuItems.push({
                label: 'Rename',
                icon: '✏️',
                action: () => handleRename(path, nodeName)
            });

            menuItems.push({
                label: 'Delete',
                icon: '🗑️',
                className: 'danger',
                action: () => handleDelete(path, nodeName, false)
            });

            menuItems.push({ separator: true });

            menuItems.push({
                label: 'Copy Path',
                icon: '📋',
                action: () => copyPath(path)
            });
        } else {
            const isOpen = openFolders.has(path);
            menuItems.push({
                label: isOpen ? 'Collapse' : 'Expand',
                icon: isOpen ? '📂' : '📁',
                action: () => toggleFolder(path)
            });

            menuItems.push({ separator: true });

            menuItems.push({
                label: 'New File',
                icon: '📄',
                action: () => handleCreateNew(path, 'file')
            });

            menuItems.push({
                label: 'New Folder',
                icon: '📁',
                action: () => handleCreateNew(path, 'directory')
            });

            menuItems.push({ separator: true });

            menuItems.push({
                label: 'Rename',
                icon: '✏️',
                action: () => handleRename(path, nodeName)
            });

            menuItems.push({
                label: 'Delete',
                icon: '🗑️',
                className: 'danger',
                action: () => handleDelete(path, nodeName, true)
            });

            menuItems.push({ separator: true });

            menuItems.push({
                label: 'Refresh',
                icon: '🔄',
                action: () => refreshPath(path)
            });

            menuItems.push({
                label: 'Copy Path',
                icon: '📋',
                action: () => copyPath(path)
            });
        }

        // Show context menu
        showContextMenu(event.clientX, event.clientY, menuItems);
    }

    // Show context menu at position
    function showContextMenu(x, y, items) {
        const menu = document.getElementById('file-context-menu');
        if (!menu) return;

        // Build menu HTML
        const menuHtml = items.map(item => {
            if (item.separator) {
                return '<div class="context-menu-item separator"></div>';
            }
            return `
                <div class="context-menu-item" data-action="${item.label}">
                    <span class="context-menu-icon">${item.icon}</span>
                    <span>${item.label}</span>
                </div>
            `;
        }).join('');

        menu.innerHTML = menuHtml;

        // Add click handlers
        menu.querySelectorAll('.context-menu-item:not(.separator)').forEach((menuItem, index) => {
            menuItem.addEventListener('click', () => {
                const itemData = items.filter(i => !i.separator)[index];
                if (itemData && itemData.action) {
                    itemData.action();
                }
                menu.classList.remove('active');
            });
        });

        // Position menu
        const menuRect = menu.getBoundingClientRect();
        const windowWidth = window.innerWidth;
        const windowHeight = window.innerHeight;

        // Adjust position if menu would go off screen
        if (x + menuRect.width > windowWidth) {
            x = windowWidth - menuRect.width - 10;
        }
        if (y + menuRect.height > windowHeight) {
            y = windowHeight - menuRect.height - 10;
        }

        menu.style.left = `${x}px`;
        menu.style.top = `${y}px`;
        menu.classList.add('active');
    }

    // View changes for a file
    function viewChanges(path) {
        if (window.diffViewer) {
            // Get the latest diff ID for this file
            const latestDiffId = window.diffViewer.getLatestDiff(path);
            if (latestDiffId) {
                window.diffViewer.showDiff(latestDiffId);
            } else {
                console.log('No diff available for', path);
                showError('No changes available for this file');
            }
        }
    }

    // Copy path to clipboard
    async function copyPath(path) {
        try {
            await navigator.clipboard.writeText(path);
            console.log('Path copied to clipboard:', path);
            // Could show a toast notification here
        } catch (error) {
            console.error('Failed to copy path:', error);
            showError('Failed to copy path to clipboard');
        }
    }

    // Handle create new file/folder
    async function handleCreateNew(parentPath, type) {
        if (!window.FileOperations) {
            showError('File operations not available');
            return;
        }

        try {
            const result = await window.FileOperations.showCreateDialog(parentPath);
            console.log(`Created ${type}:`, result);
            
            // Refresh the parent directory
            await refreshPath(parentPath);
            
            // If it's a file, optionally open it
            if (type === 'file' && result.path) {
                // Could open the newly created file
                // await openFile(result.path);
            }
        } catch (error) {
            if (error.message !== 'Cancelled') {
                console.error('Create failed:', error);
            }
        }
    }

    // Handle rename
    async function handleRename(path, oldName) {
        if (!window.FileOperations) {
            showError('File operations not available');
            return;
        }

        try {
            const result = await window.FileOperations.showRenameDialog(path, oldName);
            console.log('Renamed:', result);
            
            // Refresh the parent directory
            const parentPath = path.substring(0, path.lastIndexOf('/')) || '';
            await refreshPath(parentPath);
            
            // If the renamed file was open, update it
            if (openFiles.has(path)) {
                const fileData = openFiles.get(path);
                openFiles.delete(path);
                openFiles.set(result.newPath, {
                    ...fileData,
                    name: result.newPath.split('/').pop()
                });
                
                // If it was the active file, update that too
                if (activeFile === path) {
                    activeFile = result.newPath;
                }
                
                updateFileTabs();
            }
        } catch (error) {
            if (error.message !== 'Cancelled' && error.message !== 'No change') {
                console.error('Rename failed:', error);
            }
        }
    }

    // Handle delete
    async function handleDelete(path, name, isDir) {
        if (!window.FileOperations) {
            showError('File operations not available');
            return;
        }

        try {
            const result = await window.FileOperations.showDeleteDialog(path, name, isDir);
            console.log('Deleted:', result);
            
            // Close the file if it's open
            if (openFiles.has(path)) {
                await closeFile(path);
            }
            
            // Refresh the parent directory
            const parentPath = path.substring(0, path.lastIndexOf('/')) || '';
            await refreshPath(parentPath);
        } catch (error) {
            if (error.message !== 'Cancelled') {
                console.error('Delete failed:', error);
            }
        }
    }

    // Public API
    return {
        init,
        loadFileTree,
        openFile,
        getOpenFiles: () => openFiles,
        getActiveFile: () => activeFile,
        refreshTree: () => renderFileTree(),
        handleFileEvent,
        refreshPath,
        markFileModified,
        markFileNew,
        unmarkFileModified,
        unmarkFileNew,
        isFileModified,
        isFileNew,
        switchTab  // Export switchTab function for external use
    };
})();

// Export for use in other modules
window.FileExplorer = FileExplorer;