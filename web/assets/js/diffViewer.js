/**
 * DiffViewer - Handles diff visualization for file changes
 * Supports Monaco Editor integration for side-by-side diff view
 */
class DiffViewer {
    constructor() {
        this.modal = null;
        this.currentDiff = null;
        this.diffEditor = null;
        this.viewMode = 'monaco'; // 'monaco', 'side-by-side', 'inline', 'unified'
        this.theme = 'dark';
        this.wordWrap = false;
        this.latestDiffs = new Map(); // path -> diffId mapping
        this.init();
    }

    init() {
        // Modal is created by the HTML in ui.go
        this.modal = document.getElementById('diff-modal');
        
        if (this.modal) {
            this.bindEvents();
        }
    }

    bindEvents() {
        // Close button
        const closeBtn = this.modal.querySelector('.btn-close');
        if (closeBtn) {
            closeBtn.addEventListener('click', () => this.close());
        }

        // View mode buttons
        this.modal.querySelectorAll('.diff-mode').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.setViewMode(e.target.dataset.mode);
            });
        });

        // Word wrap checkbox
        const wordWrapCheckbox = document.getElementById('word-wrap');
        if (wordWrapCheckbox) {
            wordWrapCheckbox.addEventListener('change', (e) => {
                this.wordWrap = e.target.checked;
                if (this.diffEditor) {
                    this.diffEditor.updateOptions({ wordWrap: this.wordWrap ? 'on' : 'off' });
                }
            });
        }

        // Theme selector
        const themeSelector = document.getElementById('diff-theme');
        if (themeSelector) {
            themeSelector.addEventListener('change', (e) => {
                this.theme = e.target.value;
                if (this.diffEditor && this.viewMode === 'monaco') {
                    monaco.editor.setTheme(this.theme === 'dark' ? 'vs-dark' : 'vs');
                }
            });
        }

        // Click outside modal to close
        this.modal.addEventListener('click', (e) => {
            if (e.target === this.modal) {
                this.close();
            }
        });

        // Escape key to close
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape' && this.modal.classList.contains('active')) {
                this.close();
            }
        });
    }

    setViewMode(mode) {
        this.viewMode = mode;
        
        // Update active button
        this.modal.querySelectorAll('.diff-mode').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.mode === mode);
        });

        // Re-render if we have a diff
        if (this.currentDiff) {
            this.renderDiff();
        }
    }

    // Store latest diff ID for a file path
    setLatestDiff(path, diffId) {
        this.latestDiffs.set(path, diffId);
    }

    // Get latest diff ID for a file path
    getLatestDiff(path) {
        return this.latestDiffs.get(path);
    }

    async showDiff(diffId) {
        try {
            // Show loading state
            const container = document.getElementById('diff-container');
            container.innerHTML = '<div class="diff-loading">Loading diff...</div>';
            
            // Show modal
            this.modal.classList.add('active');

            // Fetch diff data from server
            const response = await fetch(`/api/session/${window.currentSessionId}/diff/${diffId}`);
            if (!response.ok) {
                throw new Error('Failed to fetch diff');
            }

            this.currentDiff = await response.json();
            
            // Update UI elements
            this.updateModalHeader();
            
            // Render the diff
            await this.renderDiff();

        } catch (error) {
            console.error('Error loading diff:', error);
            const container = document.getElementById('diff-container');
            container.innerHTML = `<div class="diff-error">Failed to load diff: ${error.message}</div>`;
        }
    }

    updateModalHeader() {
        if (!this.currentDiff) return;

        // Update filename
        const filenameEl = document.getElementById('diff-filename');
        if (filenameEl) {
            filenameEl.textContent = this.currentDiff.path || 'Unknown file';
        }

        // Update stats
        const stats = this.currentDiff.stats || {};
        const additionsEl = document.getElementById('additions-count');
        const deletionsEl = document.getElementById('deletions-count');
        
        if (additionsEl) additionsEl.textContent = stats.additions || 0;
        if (deletionsEl) deletionsEl.textContent = stats.deletions || 0;
    }

    async renderDiff() {
        const container = document.getElementById('diff-container');
        if (!container || !this.currentDiff) return;

        // Clean up previous editor if exists
        if (this.diffEditor) {
            this.diffEditor.dispose();
            this.diffEditor = null;
        }

        switch (this.viewMode) {
            case 'monaco':
                await this.renderMonacoDiff(container);
                break;
            case 'side-by-side':
                this.renderSideBySide(container);
                break;
            case 'inline':
                this.renderInline(container);
                break;
            case 'unified':
                this.renderUnified(container);
                break;
        }
    }

    async renderMonacoDiff(container) {
        // Wait for Monaco to be available
        if (typeof monaco === 'undefined') {
            container.innerHTML = '<div class="diff-loading">Loading Monaco Editor...</div>';
            
            // Wait and retry
            setTimeout(() => {
                if (this.currentDiff && this.viewMode === 'monaco') {
                    this.renderMonacoDiff(container);
                }
            }, 500);
            return;
        }

        // Clear container
        container.innerHTML = '';
        container.style.height = '100%';

        // Create the diff editor
        const originalModel = monaco.editor.createModel(
            this.currentDiff.before || '',
            this.getLanguageForPath(this.currentDiff.path)
        );
        
        const modifiedModel = monaco.editor.createModel(
            this.currentDiff.after || '',
            this.getLanguageForPath(this.currentDiff.path)
        );

        this.diffEditor = monaco.editor.createDiffEditor(container, {
            enableSplitViewResizing: true,
            renderSideBySide: true,
            readOnly: true,
            automaticLayout: true,
            minimap: { enabled: false },
            scrollBeyondLastLine: false,
            theme: this.theme === 'dark' ? 'vs-dark' : 'vs',
            wordWrap: this.wordWrap ? 'on' : 'off',
            fontSize: 14,
            renderWhitespace: 'selection',
            scrollbar: {
                vertical: 'visible',
                horizontal: 'visible',
                useShadows: false,
                verticalHasArrows: false,
                horizontalHasArrows: false,
                verticalScrollbarSize: 10,
                horizontalScrollbarSize: 10,
                arrowSize: 30
            }
        });

        // Set the diff
        this.diffEditor.setModel({
            original: originalModel,
            modified: modifiedModel
        });

        // Enable synchronized scrolling
        this.enableMonacoSyncScroll();

        // Layout the editor
        setTimeout(() => {
            if (this.diffEditor) {
                this.diffEditor.layout();
            }
        }, 100);
    }

    enableMonacoSyncScroll() {
        if (!this.diffEditor) return;

        // Monaco diff editor has built-in synchronized scrolling,
        // but we can enhance it by ensuring both sides stay perfectly in sync
        const editors = this.diffEditor.getOriginalEditor && this.diffEditor.getModifiedEditor 
            ? [this.diffEditor.getOriginalEditor(), this.diffEditor.getModifiedEditor()]
            : [];

        if (editors.length === 2) {
            let isScrolling = false;

            editors.forEach((editor, index) => {
                editor.onDidScrollChange(() => {
                    if (!isScrolling) {
                        isScrolling = true;
                        const otherEditor = editors[1 - index];
                        const scrollTop = editor.getScrollTop();
                        const scrollLeft = editor.getScrollLeft();
                        
                        otherEditor.setScrollPosition({
                            scrollTop: scrollTop,
                            scrollLeft: scrollLeft
                        });
                        
                        setTimeout(() => {
                            isScrolling = false;
                        }, 0);
                    }
                });
            });
        }
    }

    renderSideBySide(container) {
        const before = this.currentDiff.before || '';
        const after = this.currentDiff.after || '';
        
        const beforeLines = before.split('\n');
        const afterLines = after.split('\n');
        
        let html = '<div class="diff-side-by-side">';
        
        // Before side
        html += '<div class="diff-side before">';
        html += '<div class="diff-side-header">Before</div>';
        html += '<div class="diff-content' + (this.wordWrap ? ' wrap' : '') + '" id="diff-content-before">';
        
        beforeLines.forEach((line, i) => {
            html += `<div class="diff-line">`;
            html += `<span class="diff-line-number">${i + 1}</span>`;
            html += `<span class="diff-line-content">${this.escapeHtml(line)}</span>`;
            html += '</div>';
        });
        
        html += '</div></div>';
        
        // After side
        html += '<div class="diff-side after">';
        html += '<div class="diff-side-header">After</div>';
        html += '<div class="diff-content' + (this.wordWrap ? ' wrap' : '') + '" id="diff-content-after">';
        
        afterLines.forEach((line, i) => {
            html += `<div class="diff-line">`;
            html += `<span class="diff-line-number">${i + 1}</span>`;
            html += `<span class="diff-line-content">${this.escapeHtml(line)}</span>`;
            html += '</div>';
        });
        
        html += '</div></div>';
        html += '</div>';
        
        container.innerHTML = html;
        
        // Enable synchronized scrolling for custom side-by-side view
        this.enableCustomSyncScroll();
    }

    enableCustomSyncScroll() {
        const beforeContent = document.getElementById('diff-content-before');
        const afterContent = document.getElementById('diff-content-after');
        
        if (!beforeContent || !afterContent) return;
        
        let isScrolling = false;
        
        // Sync scroll from before to after
        beforeContent.addEventListener('scroll', () => {
            if (!isScrolling) {
                isScrolling = true;
                afterContent.scrollTop = beforeContent.scrollTop;
                afterContent.scrollLeft = beforeContent.scrollLeft;
                setTimeout(() => {
                    isScrolling = false;
                }, 10);
            }
        });
        
        // Sync scroll from after to before
        afterContent.addEventListener('scroll', () => {
            if (!isScrolling) {
                isScrolling = true;
                beforeContent.scrollTop = afterContent.scrollTop;
                beforeContent.scrollLeft = afterContent.scrollLeft;
                setTimeout(() => {
                    isScrolling = false;
                }, 10);
            }
        });
    }

    renderInline(container) {
        // For inline view, we'll show a simple before/after comparison
        const before = this.currentDiff.before || '';
        const after = this.currentDiff.after || '';
        
        let html = '<div class="diff-inline">';
        
        if (before) {
            html += '<div class="diff-section deleted">';
            html += '<div class="diff-header-line">--- Before</div>';
            html += '<div class="diff-content' + (this.wordWrap ? ' wrap' : '') + '">';
            before.split('\n').forEach((line, i) => {
                html += `<div class="diff-line deleted">`;
                html += `<span class="diff-line-number">${i + 1}</span>`;
                html += `<span class="diff-line-content">${this.escapeHtml(line)}</span>`;
                html += '</div>';
            });
            html += '</div></div>';
        }
        
        if (after) {
            html += '<div class="diff-section added">';
            html += '<div class="diff-header-line">+++ After</div>';
            html += '<div class="diff-content' + (this.wordWrap ? ' wrap' : '') + '">';
            after.split('\n').forEach((line, i) => {
                html += `<div class="diff-line added">`;
                html += `<span class="diff-line-number">${i + 1}</span>`;
                html += `<span class="diff-line-content">${this.escapeHtml(line)}</span>`;
                html += '</div>';
            });
            html += '</div></div>';
        }
        
        html += '</div>';
        container.innerHTML = html;
    }

    renderUnified(container) {
        // Simple unified diff view
        const before = (this.currentDiff.before || '').split('\n');
        const after = (this.currentDiff.after || '').split('\n');
        
        let html = '<div class="diff-unified">';
        html += '<div class="diff-header-line">@@ Diff @@</div>';
        html += '<div class="diff-content' + (this.wordWrap ? ' wrap' : '') + '">';
        
        // Simple line-by-line comparison
        const maxLines = Math.max(before.length, after.length);
        
        for (let i = 0; i < maxLines; i++) {
            if (i < before.length && i < after.length) {
                if (before[i] !== after[i]) {
                    // Changed line
                    html += `<div class="diff-line deleted">`;
                    html += `<span class="diff-line-content">- ${this.escapeHtml(before[i])}</span>`;
                    html += '</div>';
                    html += `<div class="diff-line added">`;
                    html += `<span class="diff-line-content">+ ${this.escapeHtml(after[i])}</span>`;
                    html += '</div>';
                } else {
                    // Unchanged line
                    html += `<div class="diff-line">`;
                    html += `<span class="diff-line-content">  ${this.escapeHtml(before[i])}</span>`;
                    html += '</div>';
                }
            } else if (i < before.length) {
                // Deleted line
                html += `<div class="diff-line deleted">`;
                html += `<span class="diff-line-content">- ${this.escapeHtml(before[i])}</span>`;
                html += '</div>';
            } else {
                // Added line
                html += `<div class="diff-line added">`;
                html += `<span class="diff-line-content">+ ${this.escapeHtml(after[i])}</span>`;
                html += '</div>';
            }
        }
        
        html += '</div></div>';
        container.innerHTML = html;
    }

    getLanguageForPath(path) {
        if (!path) return 'plaintext';
        
        const ext = path.split('.').pop().toLowerCase();
        const languageMap = {
            'js': 'javascript',
            'mjs': 'javascript',
            'jsx': 'javascript',
            'ts': 'typescript',
            'tsx': 'typescript',
            'json': 'json',
            'html': 'html',
            'css': 'css',
            'scss': 'scss',
            'go': 'go',
            'rs': 'rust',
            'py': 'python',
            'java': 'java',
            'cpp': 'cpp',
            'c': 'c',
            'cs': 'csharp',
            'php': 'php',
            'rb': 'ruby',
            'swift': 'swift',
            'kt': 'kotlin',
            'scala': 'scala',
            'sh': 'shell',
            'yaml': 'yaml',
            'yml': 'yaml',
            'xml': 'xml',
            'md': 'markdown',
            'sql': 'sql',
            'dockerfile': 'dockerfile',
            'makefile': 'makefile'
        };
        
        return languageMap[ext] || 'plaintext';
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text || '';
        return div.innerHTML;
    }

    async applyDiff() {
        if (!this.currentDiff) return;
        
        if (!confirm('Apply these changes to the file?')) {
            return;
        }

        try {
            const response = await fetch(`/api/session/${window.currentSessionId}/diff/${this.currentDiff.id}/apply`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) {
                throw new Error('Failed to apply diff');
            }

            const result = await response.json();
            
            // Show success message
            if (window.addSystemMessageToUI) {
                window.addSystemMessageToUI(`✅ Changes applied to ${this.currentDiff.path}`, 'success');
            }

            // Close the diff viewer
            this.close();

            // Remove the modified indicator from file explorer
            if (window.FileExplorer && window.FileExplorer.unmarkFileModified) {
                window.FileExplorer.unmarkFileModified(this.currentDiff.path);
            }

        } catch (error) {
            console.error('Error applying diff:', error);
            if (window.addSystemMessageToUI) {
                window.addSystemMessageToUI(`❌ Failed to apply changes: ${error.message}`, 'error');
            }
        }
    }

    async revertDiff() {
        if (!this.currentDiff) return;
        
        if (!confirm('Revert to the original version? This will discard the changes.')) {
            return;
        }

        try {
            const response = await fetch(`/api/session/${window.currentSessionId}/diff/${this.currentDiff.id}/revert`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json'
                }
            });

            if (!response.ok) {
                throw new Error('Failed to revert diff');
            }

            const result = await response.json();
            
            // Show success message
            if (window.addSystemMessageToUI) {
                window.addSystemMessageToUI(`↩️ Reverted changes in ${this.currentDiff.path}`, 'info');
            }

            // Close the diff viewer
            this.close();

            // Remove the modified indicator from file explorer
            if (window.FileExplorer && window.FileExplorer.unmarkFileModified) {
                window.FileExplorer.unmarkFileModified(this.currentDiff.path);
            }

        } catch (error) {
            console.error('Error reverting diff:', error);
            if (window.addSystemMessageToUI) {
                window.addSystemMessageToUI(`❌ Failed to revert changes: ${error.message}`, 'error');
            }
        }
    }

    async copyDiff() {
        if (!this.currentDiff) return;

        try {
            // Generate a unified diff format
            let diffText = `--- ${this.currentDiff.path} (before)\n`;
            diffText += `+++ ${this.currentDiff.path} (after)\n`;
            diffText += `@@ Changes @@\n`;
            
            const before = (this.currentDiff.before || '').split('\n');
            const after = (this.currentDiff.after || '').split('\n');
            const maxLines = Math.max(before.length, after.length);
            
            for (let i = 0; i < maxLines; i++) {
                if (i < before.length && i < after.length) {
                    if (before[i] !== after[i]) {
                        diffText += `- ${before[i]}\n`;
                        diffText += `+ ${after[i]}\n`;
                    } else {
                        diffText += `  ${before[i]}\n`;
                    }
                } else if (i < before.length) {
                    diffText += `- ${before[i]}\n`;
                } else {
                    diffText += `+ ${after[i]}\n`;
                }
            }

            await navigator.clipboard.writeText(diffText);
            
            if (window.addSystemMessageToUI) {
                window.addSystemMessageToUI('📋 Diff copied to clipboard', 'info');
            }
        } catch (error) {
            console.error('Failed to copy diff:', error);
            if (window.addSystemMessageToUI) {
                window.addSystemMessageToUI('❌ Failed to copy diff to clipboard', 'error');
            }
        }
    }

    close() {
        this.modal.classList.remove('active');
        
        // Clean up Monaco editor
        if (this.diffEditor) {
            this.diffEditor.dispose();
            this.diffEditor = null;
        }
        
        this.currentDiff = null;
    }
}

// Export for other modules
window.DiffViewer = DiffViewer;