package handlers

import (
	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/rweb"
)

// TestUIHandler serves a minimal test interface
func TestUIHandler(c rweb.Context) error {
	html := generateTestUI()
	return c.WriteHTML(html)
}

func generateTestUI() string {
	b := element.NewBuilder()

	b.Html().R(
		b.Head().R(
			b.Title().T("OpenCode Test - Simple Textarea"),
			b.Meta("charset", "UTF-8"),
			b.Style().T(`
				body {
					background: #1a1a1a;
					color: white;
					font-family: sans-serif;
					padding: 20px;
				}
				#container {
					width: 800px;
					height: 200px;
					margin: 0 auto;
					border: 1px solid #444;
					padding: 10px;
					background: #2a2a2a;
				}
				textarea {
					width: 100%;
					height: 100%;
					background: #333;
					color: white;
					border: none;
					padding: 10px;
					font-size: 14px;
					resize: none;
					outline: none;
				}
				button {
					margin-top: 10px;
					padding: 10px 20px;
					background: #4a9eff;
					color: white;
					border: none;
					cursor: pointer;
					border-radius: 4px;
				}
				#output {
					margin-top: 20px;
					padding: 10px;
					background: #333;
					border-radius: 4px;
					min-height: 100px;
				}
			`),
		),
		b.Body().R(
			b.H1().T("OpenCode Test Interface"),
			b.Div("id", "container").R(
				b.TextArea("id", "test-input", "placeholder", "Type your message here...").T(""),
			),
			b.Button("onclick", "sendTest()").T("Send"),
			b.Button("onclick", "clearTest()").T("Clear"),
			b.Div("id", "output").T("Output will appear here..."),
			b.Script().T(`
				const textarea = document.getElementById('test-input');
				const output = document.getElementById('output');
				
				function sendTest() {
					const value = textarea.value;
					output.textContent = 'You typed: ' + value;
					console.log('Send clicked, value:', value);
				}
				
				function clearTest() {
					textarea.value = '';
					output.textContent = 'Cleared';
					console.log('Clear clicked');
				}
				
				// Test Ctrl+Enter
				textarea.addEventListener('keydown', function(e) {
					if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
						e.preventDefault();
						sendTest();
					}
				});
				
				// Focus on load
				window.addEventListener('DOMContentLoaded', function() {
					textarea.focus();
					console.log('Test page loaded, textarea focused');
				});
			`),
		),
	)

	return b.String()
}
