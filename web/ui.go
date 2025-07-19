package web

import (
	_ "embed"

	"rcode/auth"

	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/rweb"
)

//go:embed assets/js/ui.js
var uiJS string

//go:embed assets/js/login.js
var loginJS string

//go:embed assets/js/fileExplorer.js
var fileExplorerJS string

// //go:embed assets/js/monacoLoader.js
// var monacoLoaderJS string

//go:embed assets/css/ui.css
var uiCSS string

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
									b.Span("class", "auth-status").T("Connected to Claude Pro/Max")
									b.Span("id", "connection-status", "class", "connection-status").R()
									b.Button("id", "plan-history-btn", "class", "btn-secondary").T("Plan History")
									b.Button("class", "btn-secondary", "onclick", "window.open('/prompts', '_blank')").T("Manage Prompts")
									b.Button("id", "logout-btn", "class", "btn-secondary").T("Logout")
								} else {
									b.Button("class", "btn-primary", "onclick", "handleLogin()").T("Login with Claude Pro/Max")
								}
								return nil
							}(),
						),
					),
				),
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
						// New session button (will be shown/hidden based on active tab)
						b.Div("class", "sidebar-footer").R(
							b.Button("id", "new-session-btn", "class", "btn-primary", "style", "width: 100%;").T("New Session"),
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
									// Model selector
									b.Div("class", "model-selector-container").R(
										b.Label("for", "model-selector", "class", "model-label").T("Model:"),
										b.Select("id", "model-selector", "class", "model-selector").R(
											b.Option("value", "claude-opus-4-20250514").T("Claude Opus 4 (Latest)"),
											b.Option("value", "claude-sonnet-4-20250514").T("Claude Sonnet 4 (Latest)"),
											b.Option("value", "claude-3-5-sonnet-20241022").T("Claude 3.5 Sonnet"),
											b.Option("value", "claude-3-opus-20240229").T("Claude 3 Opus"),
											b.Option("value", "claude-3-sonnet-20240229").T("Claude 3 Sonnet"),
											b.Option("value", "claude-3-haiku-20240307").T("Claude 3 Haiku (Fast)"),
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
			// Monaco Editor Scripts
			b.Script("src", "https://cdnjs.cloudflare.com/ajax/libs/monaco-editor/0.52.2/min/vs/loader.min.js").R(),
			// b.Script().T(monacoLoaderJS),
			// Our application JavaScript
			b.Script().T(generateJavaScript(isAuthenticated)),
		),
	)

	return b.String()
}

func generateCSS() string {
	return uiCSS
}

func generateJavaScript(isAuthenticated bool) string {
	if !isAuthenticated {
		// Return login JS for non-authenticated users
		return loginJS + `
			// Non-authenticated view
			document.addEventListener('DOMContentLoaded', function() {
				console.log('RCode initialized - Please login to continue');
			});
		`
	}

	// Include file explorer for authenticated users
	return fileExplorerJS + "\n\n" + uiJS + `
		// Initialize file explorer after UI is ready
		document.addEventListener('DOMContentLoaded', function() {
			// Initialize file explorer after a short delay to ensure Monaco is loaded
			setTimeout(() => {
				if (window.FileExplorer) {
					window.FileExplorer.init();
				}
			}, 500);
		});
	`
}
