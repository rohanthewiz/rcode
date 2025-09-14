package web

import (
	"embed"
	_ "embed"
	"fmt"
	"os"
	"strings"

	"rcode/auth"

	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/rweb"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
	"github.com/tdewolff/minify/v2/js"
)

// Embed all static assets
//
//go:embed assets/js/* assets/js/modules/* assets/css/*
var _ embed.FS // TODO

// Individual embeds for backward compatibility
//
//go:embed assets/js/ui.js
var uiJS string

//go:embed assets/js/login.js
var loginJS string

//go:embed assets/js/fileExplorer.js
var fileExplorerJS string

//go:embed assets/js/diffViewer.js
var diffViewerJS string

//go:embed assets/js/fileOperations.js
var fileOperationsJS string

//go:embed assets/js/file-browser.js
var fileBrowserJS string

//go:embed assets/js/modules/clipboard.js
var clipboardJS string

//go:embed assets/js/modules/state.js
var stateJS string

//go:embed assets/js/modules/events.js
var eventsJS string

//go:embed assets/js/modules/sse.js
var sseJS string

//go:embed assets/js/modules/messages.js
var messagesJS string

//go:embed assets/js/modules/session.js
var sessionJS string

//go:embed assets/js/modules/tools.js
var toolsJS string

//go:embed assets/js/modules/permissions.js
var permissionsJS string

//go:embed assets/js/modules/usage.js
var usageJS string

//go:embed assets/js/modules/markdown.js
var markdownJS string

//go:embed assets/js/modules/utils.js
var utilsJS string

//go:embed assets/js/modules/compaction.js
var compactionJS string

//go:embed assets/js/modules/tool-widget.js
var toolWidgetJS string

// //go:embed assets/js/monacoLoader.js
// var monacoLoaderJS string

//go:embed assets/css/ui.css
var uiCSS string

//go:embed assets/css/compaction.css
var compactionCSS string

//go:embed assets/css/file-browser.css
var fileBrowserCSS string

//go:embed assets/css/diffViewer.css
var diffViewerCSS string

//go:embed assets/css/fileOperations.css
var fileOperationsCSS string

// UIHandler serves the main chat interface using element package
func UIHandler(c rweb.Context) error {
	// Check if user is authenticated
	_, err := auth.GetAccessToken()
	isAuthenticated := err == nil

	return c.WriteHTML(generateMainUI(isAuthenticated))
}

func generateMainUI(isAuthenticated bool) string {
	b := element.NewBuilder()

	b.Html().R(
		b.Head().R(
			b.Title().T("RCode - AI Coding Assistant"),
			b.Meta("charset", "UTF-8"),
			b.Meta("name", "viewport", "content", "width=device-width, initial-scale=1.0"),
			b.Style().T(generateCSS()),
			// Marked.js for markdown rendering
			b.Script("src", "https://cdn.jsdelivr.net/npm/marked/marked.min.js").R(),
			// Highlight.js for code syntax highlighting
			b.Link("rel", "stylesheet", "href", "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/styles/github-dark.min.css"),
			b.Script("src", "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/11.9.0/highlight.min.js").R(),
			// Monaco Editor CSS
			b.Link("rel", "stylesheet", "href", "https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.52.2/min/vs/editor/editor.main.min.css"),
			// Our custom styles
			// Define handleLogin function early
			b.Script().T(loginJS),
		),
		b.Body().R(
			b.Div("id", "app").R(
				// Header
				b.Header().R(
					b.Div("class", "header-content").R(
						b.H1().T("RCode"),
						b.Div("class", "header-center").R(
							func() any {
								if isAuthenticated {
									// Plan Mode Toggle
									b.Div("class", "plan-mode-toggle").R(
										b.Label("class", "switch").R(
											b.Input("type", "checkbox", "id", "plan-mode-switch"),
											b.Span("class", "slider round").R(),
										),
										b.Label("for", "plan-mode-switch", "class", "plan-mode-label").T("Plan Mode"),
									)
								}
								return nil
							}(),
						),
						b.Div("class", "header-right").R(
							func() any {
								if isAuthenticated {
									b.Span("class", "auth-status").T("Connected to Claude")
									b.Span("id", "connection-status", "class", "connection-status").R()
									b.Button("id", "plan-history-btn", "class", "btn-secondary").T("Plan History")
									b.Button("class", "btn-secondary", "onclick", "window.open('/prompts', '_blank')").T("Manage Prompts")
									b.Button("id", "usage-toggle-btn", "class", "btn-secondary").T("Usage")
									b.Button("id", "logout-btn", "class", "btn-secondary").T("Logout")
								} else {
									b.Button("class", "btn-primary", "onclick", "handleLogin()").T("Login with Claude Pro/Max")
								}
								return nil
							}(),
						),
					),
				),
				// Usage Panel (hidden by default, shown as dropdown)
				func() any {
					if isAuthenticated {
						b.Div("id", "usage-panel", "class", "usage-panel", "style", "display: none;").R(
							b.Div("class", "usage-content").R(
								// Session Usage
								b.Div("class", "usage-section").R(
									b.H3().T("Current Session"),
									b.Div("class", "usage-stats").R(
										b.Div("class", "stat-item").R(
											b.Span("class", "stat-label").T("Input:"),
											b.Span("id", "session-input-tokens", "class", "stat-value").T("0"),
										),
										b.Div("class", "stat-item").R(
											b.Span("class", "stat-label").T("Output:"),
											b.Span("id", "session-output-tokens", "class", "stat-value").T("0"),
										),
										b.Div("class", "stat-item").R(
											b.Span("class", "stat-label").T("Cost:"),
											b.Span("id", "session-cost", "class", "stat-value").T("$0.00"),
										),
									),
								),
								// Rate Limits
								b.Div("class", "usage-section").R(
									b.H3().T("Rate Limits"),
									b.Div("class", "rate-limits").R(
										b.Div("class", "limit-bar").R(
											b.Span("class", "limit-label").T("Requests"),
											b.Div("class", "progress-bar").R(
												b.Div("id", "requests-progress", "class", "progress-fill").R(),
											),
											b.Span("id", "requests-remaining", "class", "limit-text").T("--"),
										),
										b.Div("class", "limit-bar").R(
											b.Span("class", "limit-label").T("Input Tokens"),
											b.Div("class", "progress-bar").R(
												b.Div("id", "input-tokens-progress", "class", "progress-fill").R(),
											),
											b.Span("id", "input-tokens-remaining", "class", "limit-text").T("--"),
										),
										b.Div("class", "limit-bar").R(
											b.Span("class", "limit-label").T("Output Tokens"),
											b.Div("class", "progress-bar").R(
												b.Div("id", "output-tokens-progress", "class", "progress-fill").R(),
											),
											b.Span("id", "output-tokens-remaining", "class", "limit-text").T("--"),
										),
									),
								),
								// Daily Usage
								b.Div("class", "usage-section").R(
									b.H3().T("Today's Usage"),
									b.Div("id", "daily-usage", "class", "usage-stats").T("Loading..."),
								),
								// Model indicator
								b.Div("class", "usage-section").R(
									b.Div("class", "model-indicator").R(
										b.Span("class", "model-label").T("Current Model:"),
										b.Span("id", "current-model", "class", "model-name").T("--"),
									),
								),
							),
						)
					}
					return nil
				}(),
				// Main content area
				b.Main().R(
					// Sidebar with tabs
					b.Aside("id", "sidebar").R(
						// Render the tabbed interface
						func() any {
							// For initial render, we'll show empty sessions and file tree
							// These will be populated via JavaScript after page load
							tabs := FileExplorerTabs{
								ActiveTab: "sessions",
								Sessions:  []SessionInfo{},
								FileTree:  []FileNode{},
							}
							element.RenderComponents(b, tabs)
							return nil
						}(),
						// New session button and compaction controls (will be shown/hidden based on active tab)
						b.Div("class", "sidebar-footer").R(
							b.Button("id", "new-session-btn", "class", "btn-primary", "style", "width: 100%; margin-bottom: 0.5rem;").T("New Session"),
							b.Button("id", "compact-session-btn", "class", "btn-secondary", "style", "width: 100%; display: none;").T("Compact Conversation"),
						),
					),
					// Chat area
					b.Section("id", "chat-area").R(
						func() any {
							if !isAuthenticated {
								b.Div("class", "auth-prompt").R(
									b.H2().T("Welcome to RCode"),
									b.P().T("Please login with your Claude Pro/Max account to start coding."),
									b.Button("class", "btn-primary large", "onclick", "handleLogin()").T("Login with Claude Pro/Max"),
								)
							} else {
								// Plan execution area (hidden by default)
								b.Div("id", "plan-execution-area", "class", "plan-execution-area", "style", "display: none;").R(
									b.Div("class", "plan-header").R(
										b.H3().T("Task Plan Execution"),
										b.Button("id", "close-plan-btn", "class", "btn-secondary").T("×"),
									),
									b.Div("id", "plan-progress", "class", "plan-progress").R(
										b.Div("class", "progress-bar").R(
											b.Div("id", "progress-fill", "class", "progress-fill", "style", "width: 0%").R(),
										),
										b.Span("id", "progress-text", "class", "progress-text").T("0 / 0 steps"),
									),
									b.Div("id", "plan-steps", "class", "plan-steps").R(),
									b.Div("class", "plan-controls").R(
										b.Button("id", "execute-plan-btn", "class", "btn-primary").T("Execute Plan"),
										b.Button("id", "pause-plan-btn", "class", "btn-secondary", "disabled", "disabled").T("Pause"),
										b.Button("id", "rollback-plan-btn", "class", "btn-warning", "disabled", "disabled").T("Rollback"),
										b.Button("id", "view-metrics-btn", "class", "btn-secondary").T("View Metrics"),
									),
								)
								// Messages container
								b.Div("id", "messages", "class", "messages").R()
								// Input area
								b.Div("class", "input-area").R(
									b.DivClass("input-area-header").R(
										// Model selector
										b.Div("class", "model-selector-container").R(
											b.Label("for", "model-selector", "class", "model-label").T("Model:"),
											b.Select("id", "model-selector", "class", "model-selector").R(
												b.Option("value", "claude-opus-4-1-20250805").T("Opus 4.1"),
												b.Option("value", "claude-opus-4-20250514").T("Opus 4"),
												b.Option("value", "claude-sonnet-4-20250514").T("Sonnet 4"),
												b.Option("value", "claude-3-7-sonnet-20250219").T("3.7 Sonnet"),
												b.Option("value", "claude-3-5-sonnet-20241022").T("3.5 Sonnet"),
												b.Option("value", "claude-3-5-haiku-20240701").T("3.5 Haiku (Fast)"),
											),
										),
										b.DivClass("tool-use-widget hidden", "id", "tool-use-widget").R(
											b.DivClass("widget-label").T("TOOLS"),
											b.DivClass("tool-cards-container").R(),
										),
									),
									// Plan mode indicator
									b.Div("id", "plan-mode-indicator", "class", "plan-mode-indicator", "style", "display: none;").R(
										b.Span("class", "plan-icon").T("📋"),
										b.Span().T("Plan Mode Active - Describe a complex task to create a plan"),
									),
									// Monaco editor container
									b.Div("id", "monaco-container", "style", "height: 150px; border: 1px solid var(--border); border-radius: 4px; margin-bottom: 1rem;").R(),
									b.Div("class", "input-controls").R(
										b.Button("id", "send-btn", "class", "btn-primary").T("Send"),
										b.Button("id", "create-plan-btn", "class", "btn-primary", "style", "display: none;").T("Create Plan"),
										b.Button("id", "clear-btn", "class", "btn-secondary").T("Clear"),
									),
								)
							}
							return nil
						}(),
					),
				),
			),
			// Plan execution overlay
			b.Div("class", "plan-overlay").R(),
			// Plan History Panel
			b.Div("id", "plan-history-panel", "class", "plan-history-panel").R(
				b.Div("class", "panel-header").R(
					b.H3().T("Plan History"),
					b.Button("id", "close-history-btn", "class", "btn-close").T("×"),
				),
				b.Div("class", "panel-controls").R(
					b.Input("type", "text", "id", "plan-search", "class", "plan-search", "placeholder", "Search plans..."),
					b.Select("id", "plan-status-filter", "class", "plan-filter").R(
						b.Option("value", "").T("All Status"),
						b.Option("value", "completed").T("Completed"),
						b.Option("value", "failed").T("Failed"),
						b.Option("value", "executing").T("Running"),
						b.Option("value", "pending").T("Pending"),
					),
				),
				b.Div("id", "plan-history-list", "class", "plan-history-list").R(
					b.Div("class", "loading").T("Loading plan history..."),
				),
				b.Div("class", "panel-footer").R(
					b.Button("id", "load-more-plans", "class", "btn-secondary", "style", "display: none;").T("Load More"),
				),
			),
			// Plan Details Modal
			b.Div("id", "plan-details-modal", "class", "modal").R(
				b.Div("class", "modal-content").R(
					b.Div("class", "modal-header").R(
						b.H3().T("Plan Details"),
						b.Button("class", "btn-close", "onclick", "closePlanDetailsModal()").T("×"),
					),
					b.Div("id", "plan-details-content", "class", "modal-body").R(),
				),
			),
			// Diff Viewer Modal
			b.Div("id", "diff-modal", "class", "modal").R(
				b.Div("class", "modal-content diff-viewer-content").R(
					b.Div("class", "diff-header").R(
						b.H3().R(
							b.T("📄 "),
							b.Span("id", "diff-filename").T("filename"),
							b.T(" - Changes"),
						),
						b.Button("class", "btn-close").T("×"),
					),
					b.Div("class", "diff-toolbar").R(
						b.Div("class", "diff-mode-selector").R(
							b.Button("class", "diff-mode active", "data-mode", "monaco").T("Monaco"),
							b.Button("class", "diff-mode", "data-mode", "side-by-side").T("Side-by-Side"),
							b.Button("class", "diff-mode", "data-mode", "inline").T("Inline"),
							b.Button("class", "diff-mode", "data-mode", "unified").T("Unified"),
						),
						b.Div("class", "diff-options").R(
							b.Label().R(
								b.Input("type", "checkbox", "id", "word-wrap"),
								b.T(" Wrap"),
							),
							b.Select("id", "diff-theme").R(
								b.Option("value", "dark").T("Dark"),
								b.Option("value", "light").T("Light"),
							),
						),
						b.Div("class", "diff-stats").R(
							b.Span("class", "additions").R(
								b.T("+"),
								b.Span("id", "additions-count").T("0"),
							),
							b.Span("class", "deletions").R(
								b.T("-"),
								b.Span("id", "deletions-count").T("0"),
							),
						),
					),
					b.Div("id", "diff-container", "class", "diff-container").R(),
					b.Div("class", "diff-actions").R(
						b.Button("class", "btn-primary", "onclick", "window.diffViewer && window.diffViewer.applyDiff()").T("Apply Changes"),
						b.Button("class", "btn-secondary", "onclick", "window.diffViewer && window.diffViewer.revertDiff()").T("Revert"),
						b.Button("class", "btn-secondary", "onclick", "window.diffViewer && window.diffViewer.copyDiff()").T("Copy Diff"),
					),
				),
			),
			// Permission Dialog Modal
			b.Div("id", "permission-modal", "class", "modal").R(
				b.Div("class", "modal-content permission-dialog").R(
					b.Div("class", "modal-header").R(
						b.H3().R(
							b.Span("class", "permission-icon").T("🔐"),
							b.T(" Tool Permission Required"),
						),
					),
					b.Div("class", "modal-body").R(
						b.Div("class", "permission-info").R(
							b.P().R(
								b.T("The AI wants to use the "),
								b.Strong("id", "permission-tool-name").T(""),
								b.T(" tool with the following parameters:"),
							),
							b.Div("id", "permission-params", "class", "permission-params").R(),
						),
						// Diff preview section (hidden by default)
						b.Div("id", "permission-diff-section", "class", "permission-diff-section", "style", "display: none;").R(
							b.Div("class", "permission-diff-header").R(
								b.Button("id", "permission-diff-toggle", "class", "diff-toggle-btn").R(
									b.Span("class", "toggle-icon").T("▶"),
									b.T(" View Changes "),
									b.Span("id", "permission-diff-stats", "class", "diff-stats").T(""),
								),
							),
							b.Div("id", "permission-diff-container", "class", "permission-diff-container", "style", "display: none;").R(
								b.Div("id", "permission-diff-content", "class", "permission-diff-content").R(),
							),
						),
						b.Div("class", "permission-warning").R(
							b.P().T("⚠️ Please review the operation carefully before approving."),
						),
						b.Div("class", "permission-remember").R(
							b.Label().R(
								b.Input("type", "checkbox", "id", "permission-remember"),
								b.T(" Remember this choice for this session"),
							),
						),
					),
					b.Div("class", "modal-footer permission-actions").R(
						b.Button("id", "permission-deny", "class", "btn-secondary").T("Deny"),
						b.Button("id", "permission-approve", "class", "btn-primary").T("Approve"),
						b.Button("id", "permission-abort", "class", "btn-danger", "title", "Completely stop the current operation").T("ABORT"),
					),
				),
			),
			// Monaco Editor Scripts
			b.Script("src", "https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.52.2/min/vs/loader.min.js").R(),
			// b.Script().T(monacoLoaderJS),
			// Our application JavaScript
			b.Script().T(generateJavaScript(isAuthenticated)),
		),
	)

	return b.String()
}

// minifyJavaScript minifies JavaScript code without obfuscation
func minifyJavaScript(jsCode string) string {
	// Check if minification is disabled
	if os.Getenv("RCODE_MINIFY") == "false" {
		return jsCode
	}

	m := minify.New()
	// Configure JavaScript minifier with safe settings (no obfuscation)
	m.Add("text/javascript", &js.Minifier{
		KeepVarNames: true, // Don't obfuscate variable names
		Precision:    0,    // Keep all decimal precision
	})

	minified, err := m.String("text/javascript", jsCode)
	if err != nil {
		// If minification fails, return original code
		fmt.Printf("JavaScript minification failed: %v\n", err)
		return jsCode
	}
	return minified
}

// minifyCSS minifies CSS code
func minifyCSS(cssCode string) string {
	// Check if minification is disabled
	if os.Getenv("RCODE_MINIFY") == "false" {
		return cssCode
	}

	m := minify.New()
	// Configure CSS minifier
	m.Add("text/css", &css.Minifier{
		Precision: 0, // Keep all decimal precision
	})

	minified, err := m.String("text/css", cssCode)
	if err != nil {
		// If minification fails, return original code
		fmt.Printf("CSS minification failed: %v\n", err)
		return cssCode
	}
	return minified
}

func generateCSS() string {
	combinedCSS := uiCSS + "\n\n" + diffViewerCSS + "\n\n" + fileOperationsCSS + "\n\n" + fileBrowserCSS + "\n\n" + compactionCSS
	return minifyCSS(combinedCSS)
}

func generateJavaScript(isAuthenticated bool) string {
	if !isAuthenticated {
		// Return minified login JS for non-authenticated users
		nonAuthJS := loginJS + `
			// Non-authenticated view
			document.addEventListener('DOMContentLoaded', function() {
				console.log('RCode initialized - Please login to continue');
			});
		`
		return minifyJavaScript(nonAuthJS)
	}

	// Include file explorer, file operations, and diff viewer for authenticated users
	// Wrap all modules in IIFE pattern for browser compatibility
	stateModule := `
// State module wrapped for non-module usage
(function() {
` + stateJS + `
})();
`

	eventsModule := `
// Events module wrapped for non-module usage  
(function() {
` + eventsJS + `
})();
`

	sseModule := `
// SSE module wrapped for non-module usage
(function() {
` + sseJS + `
})();
`

	messagesModule := `
// Messages module wrapped for non-module usage
(function() {
` + messagesJS + `
})();
`

	sessionModule := `
// Session module wrapped for non-module usage  
(function() {
` + sessionJS + `
})();
`

	toolsModule := `
// Tools module wrapped for non-module usage
(function() {
` + toolsJS + `
})();
`

	toolWidgetModule := `
// Tool Widget module wrapped for non-module usage
(function() {
` + toolWidgetJS + `
})();
`

	permissionsModule := `
// Permissions module wrapped for non-module usage
(function() {
` + permissionsJS + `
})();
`

	usageModule := `
// Usage module wrapped for non-module usage
(function() {
` + usageJS + `
})();
`

	markdownModule := `
// Markdown module wrapped for non-module usage
(function() {
` + markdownJS + `
})();
`

	utilsModule := `
// Utils module wrapped for non-module usage
(function() {
` + utilsJS + `
})();
`

	clipboardModule := `
// Clipboard module wrapped for non-module usage
(function() {
	const ClipboardModule = {};` + "\n" +
		clipboardJS + "\n" + `
	// Export functions to global ClipboardModule object
	window.ClipboardModule = {
		setupClipboardHandling,
		processImageBlob,
		handlePasteEvent,
		showImagePastedNotification,
		setupDragAndDrop,
		handleFiles,
		processImageFile
	};
})();
`
	// Load core modules first, then feature modules, then main UI
	// Order: utils -> markdown -> state -> events -> sse -> messages -> session -> tools -> tool-widget -> permissions -> usage -> other modules -> ui
	combinedJS := utilsModule + "\n\n" + markdownModule + "\n\n" + stateModule + "\n\n" +
		eventsModule + "\n\n" + sseModule + "\n\n" + messagesModule + "\n\n" +
		sessionModule + "\n\n" + toolsModule + "\n\n" + toolWidgetModule + "\n\n" + permissionsModule + "\n\n" +
		usageModule + "\n\n" + fileOperationsJS + "\n\n" + fileExplorerJS + "\n\n" +
		diffViewerJS + "\n\n" + clipboardModule + "\n\n" + uiJS + `
		// Initialize file explorer and diff viewer after UI is ready
		document.addEventListener('DOMContentLoaded', function() {
			// Initialize file explorer after a short delay to ensure Monaco is loaded
			setTimeout(() => {
				if (window.FileExplorer) {
					window.FileExplorer.init();
				}
				// Initialize file browser with context menu
				if (window.FileBrowser) {
					window.fileBrowser = new window.FileBrowser();
				}
				// Initialize diff viewer
				if (window.DiffViewer) {
					window.diffViewer = new window.DiffViewer();
				}
			}, 500);
		});
	`

	// Minify the combined JavaScript
	return minifyJavaScript(combinedJS)
}

// Check if modular JavaScript files exist
func hasModules() bool {
	_, err := assetsFS.ReadFile("assets/js/modules/main.js")
	return err == nil
}

// Generate modular JavaScript that uses ES6 modules
func generateModularJavaScript() string {
	// For ES6 modules, we need to serve them as separate files and use import
	// This requires serving the modules directory and using type="module" in script tags
	// For now, we'll concatenate them in dependency order as a transitional approach

	moduleFiles := []string{
		"assets/js/modules/state.js",
		"assets/js/modules/markdown.js",
		"assets/js/modules/utils.js",
		"assets/js/modules/clipboard.js",
		"assets/js/modules/fileMention.js",
		"assets/js/modules/usage.js",
		"assets/js/modules/permissions.js",
		"assets/js/modules/messages.js",
		"assets/js/modules/tools.js",
		"assets/js/modules/session.js",
		"assets/js/modules/compaction.js",
		"assets/js/modules/sse.js",
		"assets/js/modules/events.js",
		"assets/js/modules/main.js",
	}

	var jsContent strings.Builder

	// Add supporting files first
	jsContent.WriteString(fileOperationsJS + "\n\n")
	jsContent.WriteString(fileExplorerJS + "\n\n")
	jsContent.WriteString(diffViewerJS + "\n\n")

	// Wrap modules in an IIFE to avoid global pollution
	jsContent.WriteString("(function() {\n")
	jsContent.WriteString("'use strict';\n\n")

	// Read and concatenate module files, converting ES6 imports/exports
	for _, file := range moduleFiles {
		content, err := assetsFS.ReadFile(file)
		if err != nil {
			fmt.Printf("Warning: Could not read module %s: %v\n", file, err)
			continue
		}

		// Convert ES6 module syntax to compatible format
		moduleContent := convertES6Module(string(content), file)
		jsContent.WriteString(fmt.Sprintf("// Module: %s\n", file))
		jsContent.WriteString(moduleContent)
		jsContent.WriteString("\n\n")
	}

	jsContent.WriteString("})();\n")

	return jsContent.String()
}

// Convert ES6 module syntax to browser-compatible format
func convertES6Module(content, filename string) string {
	// This is a simplified conversion that wraps modules in a way they can work
	// In production, you'd want to use a proper bundler like esbuild or webpack

	// Remove import statements (they'll be loaded in order)
	lines := strings.Split(content, "\n")
	var result []string

	for _, line := range lines {
		// Skip import statements
		if strings.HasPrefix(strings.TrimSpace(line), "import ") {
			continue
		}

		// Convert export statements to window assignments for global access
		if strings.HasPrefix(strings.TrimSpace(line), "export ") {
			line = strings.Replace(line, "export const ", "window.", 1)
			line = strings.Replace(line, "export function ", "window.", 1)
			line = strings.Replace(line, "export {", "// Export: {", 1)
			line = strings.Replace(line, "export default ", "window.default_", 1)
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
