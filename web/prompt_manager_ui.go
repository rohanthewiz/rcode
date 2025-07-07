package web

import (
	"rcode/auth"

	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/rweb"
)

// PromptManagerHandler serves the prompt management interface
func PromptManagerHandler(c rweb.Context) error {
	// Check if user is authenticated
	_, err := auth.GetAccessToken()
	if err != nil {
		return c.WriteError(err, 401)
	}

	return c.WriteHTML(generatePromptManagerUI())
}

func generatePromptManagerUI() string {
	b := element.NewBuilder()

	b.Html().R(
		b.Head().R(
			b.Title().T("Prompt Manager - RCode"),
			b.Meta("charset", "UTF-8"),
			b.Meta("name", "viewport", "content", "width=device-width, initial-scale=1.0"),
			b.Style().T(generatePromptManagerCSS()),
		),
		b.Body().R(
			b.Div("id", "prompt-manager").R(
				b.Header().R(
					b.Div("class", "header-content").R(
						b.H1().T("Initial Prompt Manager"),
						b.Button("id", "close-btn", "class", "btn-secondary", "onclick", "window.close()").T("Close"),
					),
				),
				b.Main().R(
					b.Div("class", "toolbar").R(
						b.Button("id", "add-prompt-btn", "class", "btn-primary", "onclick", "showAddPromptForm()").T("+ New Prompt"),
						b.Button("id", "back-to-chat-btn", "class", "btn-secondary", "onclick", "window.location.href='/'").T("Back to Chat"),
					),
					b.Div("id", "prompts-list", "class", "prompts-list").T("Loading prompts..."),
					b.Div("id", "prompt-form", "class", "prompt-form hidden").R(
						b.H2("id", "form-title").T("Add New Prompt"),
						b.Form("onsubmit", "savePrompt(event)").R(
							b.Div("class", "form-group").R(
								b.Label("for", "prompt-name").T("Name"),
								b.Input("type", "text", "id", "prompt-name", "name", "name", "required", "required", "placeholder", "e.g., go_language_prompt"),
							),
							b.Div("class", "form-group").R(
								b.Label("for", "prompt-description").T("Description"),
								b.Input("type", "text", "id", "prompt-description", "name", "description", "placeholder", "Brief description of this prompt"),
							),
							b.Div("class", "form-group").R(
								b.Label("for", "prompt-content").T("Content"),
								b.TextArea("id", "prompt-content", "name", "content", "required", "required", "rows", "4", "placeholder", "The actual prompt text...").R(),
							),
							b.Div("class", "form-group checkbox-group").R(
								b.Label().R(
									b.Input("type", "checkbox", "id", "prompt-includes-permissions", "name", "includes_permissions"),
									b.Span().T("Includes Permissions"),
								),
							),
							b.Div("class", "form-group checkbox-group").R(
								b.Label().R(
									b.Input("type", "checkbox", "id", "prompt-is-active", "name", "is_active", "checked", "checked"),
									b.Span().T("Active"),
								),
							),
							b.Div("class", "form-group checkbox-group").R(
								b.Label().R(
									b.Input("type", "checkbox", "id", "prompt-is-default", "name", "is_default"),
									b.Span().T("Default (automatically applied to new sessions)"),
								),
							),
							b.Div("class", "form-actions").R(
								b.Button("type", "submit", "class", "btn-primary").T("Save Prompt"),
								b.Button("type", "button", "class", "btn-secondary", "onclick", "hidePromptForm()").T("Cancel"),
							),
						),
					),
				),
			),
			b.Script().T(generatePromptManagerJS()),
		),
	)

	return b.String()
}

func generatePromptManagerCSS() string {
	return `
		:root {
			--bg-primary: #1a1a1a;
			--bg-secondary: #2a2a2a;
			--bg-tertiary: #3a3a3a;
			--text-primary: #ffffff;
			--text-secondary: #b0b0b0;
			--accent: #4a9eff;
			--accent-hover: #3a8eef;
			--border: #404040;
			--success: #4caf50;
			--error: #f44336;
			--warning: #ff9800;
		}

		* {
			margin: 0;
			padding: 0;
			box-sizing: border-box;
		}

		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
			background: var(--bg-primary);
			color: var(--text-primary);
			height: 100vh;
			overflow: hidden;
		}

		#prompt-manager {
			display: flex;
			flex-direction: column;
			height: 100vh;
		}

		header {
			background: var(--bg-secondary);
			border-bottom: 1px solid var(--border);
			padding: 1rem;
		}

		.header-content {
			display: flex;
			justify-content: space-between;
			align-items: center;
			max-width: 1200px;
			margin: 0 auto;
			width: 100%;
		}

		main {
			flex: 1;
			overflow-y: auto;
			padding: 2rem;
			max-width: 1200px;
			margin: 0 auto;
			width: 100%;
		}

		.toolbar {
			display: flex;
			gap: 1rem;
			margin-bottom: 2rem;
		}

		.btn-primary, .btn-secondary {
			padding: 0.5rem 1rem;
			border: none;
			border-radius: 6px;
			cursor: pointer;
			font-size: 0.9rem;
			font-weight: 500;
			transition: all 0.2s;
		}

		.btn-primary {
			background: var(--accent);
			color: white;
		}

		.btn-primary:hover {
			background: var(--accent-hover);
		}

		.btn-secondary {
			background: var(--bg-tertiary);
			color: var(--text-primary);
			border: 1px solid var(--border);
		}

		.btn-secondary:hover {
			background: var(--border);
		}

		.prompts-list {
			display: grid;
			gap: 1rem;
		}

		.prompt-card {
			background: var(--bg-secondary);
			border: 1px solid var(--border);
			border-radius: 8px;
			padding: 1.5rem;
			transition: all 0.2s;
		}

		.prompt-card:hover {
			border-color: var(--accent);
		}

		.prompt-card-header {
			display: flex;
			justify-content: space-between;
			align-items: flex-start;
			margin-bottom: 1rem;
		}

		.prompt-card-title {
			font-size: 1.1rem;
			font-weight: 600;
			color: var(--accent);
		}

		.prompt-card-badges {
			display: flex;
			gap: 0.5rem;
		}

		.badge {
			font-size: 0.75rem;
			padding: 0.2rem 0.5rem;
			border-radius: 4px;
			font-weight: 500;
		}

		.badge-default {
			background: var(--success);
			color: white;
		}

		.badge-permissions {
			background: var(--warning);
			color: white;
		}

		.badge-inactive {
			background: var(--bg-tertiary);
			color: var(--text-secondary);
		}

		.prompt-card-description {
			color: var(--text-secondary);
			font-size: 0.9rem;
			margin-bottom: 0.5rem;
		}

		.prompt-card-content {
			background: var(--bg-tertiary);
			padding: 0.75rem;
			border-radius: 6px;
			font-family: monospace;
			font-size: 0.85rem;
			margin-bottom: 1rem;
			line-height: 1.4;
		}

		.prompt-card-actions {
			display: flex;
			gap: 0.5rem;
		}

		.btn-small {
			padding: 0.25rem 0.75rem;
			font-size: 0.85rem;
		}

		.btn-danger {
			background: var(--error);
			color: white;
		}

		.btn-danger:hover {
			background: #d32f2f;
		}

		.prompt-form {
			background: var(--bg-secondary);
			border: 1px solid var(--border);
			border-radius: 8px;
			padding: 2rem;
			margin-bottom: 2rem;
		}

		.prompt-form.hidden {
			display: none;
		}

		.form-group {
			margin-bottom: 1.5rem;
		}

		.form-group label {
			display: block;
			margin-bottom: 0.5rem;
			font-weight: 500;
			color: var(--text-secondary);
		}

		.form-group input[type="text"],
		.form-group textarea {
			width: 100%;
			padding: 0.75rem;
			background: var(--bg-tertiary);
			border: 1px solid var(--border);
			border-radius: 6px;
			color: var(--text-primary);
			font-size: 0.9rem;
		}

		.form-group input[type="text"]:focus,
		.form-group textarea:focus {
			outline: none;
			border-color: var(--accent);
		}

		.checkbox-group {
			display: flex;
			align-items: center;
		}

		.checkbox-group label {
			display: flex;
			align-items: center;
			margin-bottom: 0;
			cursor: pointer;
		}

		.checkbox-group input[type="checkbox"] {
			margin-right: 0.5rem;
		}

		.form-actions {
			display: flex;
			gap: 1rem;
			justify-content: flex-end;
		}
	`
}

func generatePromptManagerJS() string {
	return `
		let editingPromptId = null;

		// Load prompts on page load
		document.addEventListener('DOMContentLoaded', function() {
			loadPrompts();
		});

		async function loadPrompts() {
			try {
				const response = await fetch('/api/prompts');
				const prompts = await response.json();
				displayPrompts(prompts);
			} catch (error) {
				console.error('Failed to load prompts:', error);
				document.getElementById('prompts-list').innerHTML = '<p>Failed to load prompts</p>';
			}
		}

		function displayPrompts(prompts) {
			const container = document.getElementById('prompts-list');
			
			if (prompts.length === 0) {
				container.innerHTML = '<p>No prompts configured yet.</p>';
				return;
			}

			container.innerHTML = prompts.map(prompt => {
				const badges = [];
				if (prompt.is_default) badges.push('<span class="badge badge-default">Default</span>');
				if (prompt.includes_permissions) badges.push('<span class="badge badge-permissions">Permissions</span>');
				if (!prompt.is_active) badges.push('<span class="badge badge-inactive">Inactive</span>');

				return ` + "`" + `
					<div class="prompt-card">
						<div class="prompt-card-header">
							<div class="prompt-card-title">${escapeHtml(prompt.name)}</div>
							<div class="prompt-card-badges">${badges.join('')}</div>
						</div>
						${prompt.description ? ` + "`" + `<div class="prompt-card-description">${escapeHtml(prompt.description)}</div>` + "`" + ` : ''}
						<div class="prompt-card-content">${escapeHtml(prompt.content)}</div>
						<div class="prompt-card-actions">
							<button class="btn-secondary btn-small" onclick="editPrompt(${prompt.id})">Edit</button>
							<button class="btn-danger btn-small" onclick="deletePrompt(${prompt.id}, '${escapeHtml(prompt.name)}')">Delete</button>
						</div>
					</div>
				` + "`" + `;
			}).join('');
		}

		function showAddPromptForm() {
			editingPromptId = null;
			document.getElementById('form-title').textContent = 'Add New Prompt';
			document.getElementById('prompt-form').classList.remove('hidden');
			document.getElementById('prompt-form').reset();
			document.getElementById('prompt-is-active').checked = true;
		}

		function hidePromptForm() {
			document.getElementById('prompt-form').classList.add('hidden');
			editingPromptId = null;
		}

		async function editPrompt(id) {
			try {
				const response = await fetch('/api/prompts/' + id);
				const prompt = await response.json();
				
				editingPromptId = id;
				document.getElementById('form-title').textContent = 'Edit Prompt';
				document.getElementById('prompt-name').value = prompt.name;
				document.getElementById('prompt-description').value = prompt.description || '';
				document.getElementById('prompt-content').value = prompt.content;
				document.getElementById('prompt-includes-permissions').checked = prompt.includes_permissions;
				document.getElementById('prompt-is-active').checked = prompt.is_active;
				document.getElementById('prompt-is-default').checked = prompt.is_default;
				
				document.getElementById('prompt-form').classList.remove('hidden');
			} catch (error) {
				console.error('Failed to load prompt:', error);
				alert('Failed to load prompt for editing');
			}
		}

		async function savePrompt(event) {
			event.preventDefault();
			
			const formData = new FormData(event.target);
			const prompt = {
				name: formData.get('name'),
				description: formData.get('description'),
				content: formData.get('content'),
				includes_permissions: formData.get('includes_permissions') === 'on',
				is_active: formData.get('is_active') === 'on',
				is_default: formData.get('is_default') === 'on'
			};

			try {
				const url = editingPromptId ? '/api/prompts/' + editingPromptId : '/api/prompts';
				const method = editingPromptId ? 'PUT' : 'POST';
				
				const response = await fetch(url, {
					method: method,
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(prompt)
				});

				if (!response.ok) {
					const errorText = await response.text();
					console.error('Server error:', errorText);
					throw new Error(errorText || 'Failed to save prompt');
				}

				hidePromptForm();
				loadPrompts();
			} catch (error) {
				console.error('Failed to save prompt:', error);
				alert('Failed to save prompt: ' + error.message);
			}
		}

		async function deletePrompt(id, name) {
			if (!confirm('Are you sure you want to delete the prompt "' + name + '"?')) {
				return;
			}

			try {
				const response = await fetch('/api/prompts/' + id, {
					method: 'DELETE'
				});

				if (!response.ok) {
					throw new Error('Failed to delete prompt');
				}

				loadPrompts();
			} catch (error) {
				console.error('Failed to delete prompt:', error);
				alert('Failed to delete prompt: ' + error.message);
			}
		}

		function escapeHtml(text) {
			const div = document.createElement('div');
			div.textContent = text;
			return div.innerHTML;
		}
	`
}
