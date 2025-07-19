package web

import (
	"fmt"
	"strings"
	"time"

	"github.com/rohanthewiz/element"
)

// FileTreeComponent represents the file tree UI component
type FileTreeComponent struct {
	Nodes []FileNode
}

// Render implements the element.Component interface
func (f FileTreeComponent) Render(b *element.Builder) (x any) {
	b.Div("class", "file-explorer").R(
		b.Div("class", "file-tree").R(
			element.ForEach(f.Nodes, func(node FileNode) {
				renderFileNode(b, node, 0)
			}),
		),
	)
	return
}

// renderFileNode recursively renders a file node
func renderFileNode(b *element.Builder, node FileNode, depth int) {
	// Calculate indentation based on depth
	indent := depth * 20
	
	// Node attributes
	attrs := []string{
		"class", "tree-node",
		"data-path", node.Path,
		"data-is-dir", fmt.Sprintf("%v", node.IsDir),
		"style", fmt.Sprintf("padding-left: %dpx", indent),
	}
	
	b.Div(attrs...).R(
		// Expand/collapse icon for directories
		func() (x any) {
			if node.IsDir {
				iconClass := "tree-icon folder-icon"
				if node.IsOpen {
					iconClass = "tree-icon folder-open-icon"
				}
				b.Span("class", iconClass, "data-action", "toggle")
				return
			}
			// File icon
			b.Span("class", fmt.Sprintf("tree-icon file-icon file-icon-%s", node.Icon))
			return
		}(),
		
		// Node name
		b.Span("class", "node-name").T(node.Name),
		
		// File size for files
		func() (x any) {
			if !node.IsDir && node.Size > 0 {
				b.Span("class", "node-size").T(formatFileSize(node.Size))
			}
			return
		}(),
	)
	
	// Render children if directory is open
	if node.IsDir && node.IsOpen && len(node.Children) > 0 {
		b.Div("class", "tree-children").R(
			element.ForEach(node.Children, func(child FileNode) {
				renderFileNode(b, child, depth+1)
			}),
		)
	}
}

// SessionInfo represents basic session information for the UI
type SessionInfo struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

// FileExplorerTabs represents the tabbed sidebar interface
type FileExplorerTabs struct {
	ActiveTab string // "sessions" or "files"
	Sessions  []SessionInfo
	FileTree  []FileNode
}

// Render implements the element.Component interface for tabs
func (f FileExplorerTabs) Render(b *element.Builder) (x any) {
	// Tab headers
	b.Div("class", "sidebar-tabs").R(
		b.Div("class", getTabClass("sessions", f.ActiveTab), "data-tab", "sessions").T("Sessions"),
		b.Div("class", getTabClass("files", f.ActiveTab), "data-tab", "files").T("Files"),
	)
	
	// Tab content
	b.Div("class", "sidebar-content").R(
		// Sessions tab
		b.Div("class", getTabContentClass("sessions", f.ActiveTab), "id", "sessions-tab").R(
			b.Div("id", "session-list").R(
				element.ForEach(f.Sessions, func(session SessionInfo) {
					b.Div("class", "session-item", "data-session-id", session.ID).R(
						b.Div("class", "session-name").T(session.Name),
						b.Div("class", "session-date").T(session.CreatedAt.Format("Jan 2, 15:04")),
					)
				}),
			),
		),
		
		// Files tab
		b.Div("class", getTabContentClass("files", f.ActiveTab), "id", "files-tab").R(
			b.Div("class", "file-search").R(
				b.Input("type", "text", "id", "file-search-input", "placeholder", "Search files...", "class", "search-input"),
			),
			b.Div("id", "file-tree-container").R(
				func() (x any) {
					if len(f.FileTree) > 0 {
						fileTreeComp := FileTreeComponent{Nodes: f.FileTree}
						element.RenderComponents(b, fileTreeComp)
					} else {
						b.Div("class", "empty-state").T("Loading file tree...")
					}
					return
				}(),
			),
		),
	)
	return
}

// getTabClass returns the CSS class for a tab
func getTabClass(tab, activeTab string) string {
	if tab == activeTab {
		return "sidebar-tab active"
	}
	return "sidebar-tab"
}

// getTabContentClass returns the CSS class for tab content
func getTabContentClass(tab, activeTab string) string {
	if tab == activeTab {
		return "tab-content active"
	}
	return "tab-content"
}

// formatFileSize formats file size in human-readable format
func formatFileSize(size int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	
	switch {
	case size >= GB:
		return fmt.Sprintf("%.1f GB", float64(size)/float64(GB))
	case size >= MB:
		return fmt.Sprintf("%.1f MB", float64(size)/float64(MB))
	case size >= KB:
		return fmt.Sprintf("%.1f KB", float64(size)/float64(KB))
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// FileViewerComponent represents the file viewer panel
type FileViewerComponent struct {
	OpenFiles []OpenFile
	ActiveFile string
}

// OpenFile represents an open file in the viewer
type OpenFile struct {
	Path    string
	Name    string
	Content string
	Language string
}

// Render implements the element.Component interface for file viewer
func (f FileViewerComponent) Render(b *element.Builder) (x any) {
	if len(f.OpenFiles) == 0 {
		return
	}
	
	b.Div("id", "file-viewer", "class", "file-viewer").R(
		// File tabs
		b.Div("class", "file-tabs").R(
			element.ForEach(f.OpenFiles, func(file OpenFile) {
				tabClass := "file-tab"
				if file.Path == f.ActiveFile {
					tabClass = "file-tab active"
				}
				b.Div("class", tabClass, "data-path", file.Path).R(
					b.Span("class", "tab-name").T(file.Name),
					b.Span("class", "tab-close", "data-action", "close-file").T("×"),
				)
			}),
		),
		
		// File content area
		b.Div("class", "file-content").R(
			func() (x any) {
				for _, file := range f.OpenFiles {
					if file.Path == f.ActiveFile {
						b.Div("id", "file-viewer-editor", "data-language", file.Language)
						break
					}
				}
				return
			}(),
		),
	)
	return
}

// getFileLanguage determines the language for syntax highlighting
func getFileLanguage(filename string) string {
	ext := strings.ToLower(strings.TrimPrefix(filename, "."))
	if idx := strings.LastIndex(filename, "."); idx >= 0 {
		ext = strings.ToLower(filename[idx+1:])
	}
	
	languageMap := map[string]string{
		"go":     "go",
		"js":     "javascript",
		"mjs":    "javascript",
		"cjs":    "javascript",
		"ts":     "typescript",
		"tsx":    "typescript",
		"jsx":    "javascript",
		"py":     "python",
		"java":   "java",
		"rb":     "ruby",
		"rs":     "rust",
		"c":      "c",
		"h":      "c",
		"cpp":    "cpp",
		"cxx":    "cpp",
		"cc":     "cpp",
		"hpp":    "cpp",
		"cs":     "csharp",
		"php":    "php",
		"html":   "html",
		"css":    "css",
		"scss":   "scss",
		"sass":   "scss",
		"less":   "less",
		"json":   "json",
		"xml":    "xml",
		"yaml":   "yaml",
		"yml":    "yaml",
		"md":     "markdown",
		"sql":    "sql",
		"sh":     "shell",
		"bash":   "shell",
		"vim":    "vim",
		"dockerfile": "dockerfile",
		"makefile":   "makefile",
	}
	
	if lang, ok := languageMap[ext]; ok {
		return lang
	}
	return "plaintext"
}