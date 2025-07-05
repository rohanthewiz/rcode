package handlers

import (
	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/rweb"
)

// AuthCallbackHandler displays a page for users to enter their authorization code
func AuthCallbackHandler(c rweb.Context) error {
	html := generateAuthCallbackHTML()
	return c.WriteHTML(html)
}

func generateAuthCallbackHTML() string {
	b := element.NewBuilder()

	b.Html().R(
		b.Head().R(
			b.Title().T("RCode - Enter Authorization Code"),
			b.Meta("charset", "UTF-8"),
			b.Meta("name", "viewport", "content", "width=device-width, initial-scale=1.0"),
			b.Style().T(`
				body {
					font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
					background: #1a1a1a;
					color: #ffffff;
					display: flex;
					justify-content: center;
					align-items: center;
					height: 100vh;
					margin: 0;
				}
				.container {
					background: #2a2a2a;
					padding: 2rem;
					border-radius: 8px;
					max-width: 500px;
					width: 90%;
				}
				h1 {
					color: #4a9eff;
					margin-bottom: 1rem;
				}
				p {
					color: #b0b0b0;
					margin-bottom: 1.5rem;
					line-height: 1.6;
				}
				.code-input {
					width: 100%;
					padding: 0.75rem;
					background: #3a3a3a;
					border: 1px solid #404040;
					color: #ffffff;
					border-radius: 4px;
					font-family: monospace;
					margin-bottom: 1rem;
				}
				.code-input:focus {
					outline: none;
					border-color: #4a9eff;
				}
				.btn {
					width: 100%;
					padding: 0.75rem;
					background: #4a9eff;
					color: white;
					border: none;
					border-radius: 4px;
					cursor: pointer;
					font-size: 1rem;
				}
				.btn:hover {
					background: #3a8eef;
				}
				.btn:disabled {
					background: #404040;
					cursor: not-allowed;
				}
				.error {
					color: #f44336;
					margin-top: 1rem;
					display: none;
				}
				.success {
					color: #4caf50;
					margin-top: 1rem;
					display: none;
				}
			`),
		),
		b.Body().R(
			b.Div("class", "container").R(
				b.H1().T("Authorization Required"),
				b.P().T("After authorizing RCode on Claude.ai, you should see an authorization code. Please copy and paste it below:"),
				b.Input("type", "text", "id", "code-input", "class", "code-input", "placeholder", "Paste authorization code here"),
				b.Button("id", "submit-btn", "class", "btn", "onclick", "submitCode()").T("Submit Code"),
				b.Div("id", "error-msg", "class", "error").T("Invalid code. Please try again."),
				b.Div("id", "success-msg", "class", "success").T("Success! Redirecting..."),
			),
		),
		b.Script().T(`
			async function submitCode() {
				const codeInput = document.getElementById('code-input');
				const submitBtn = document.getElementById('submit-btn');
				const errorMsg = document.getElementById('error-msg');
				const successMsg = document.getElementById('success-msg');
				
				const code = codeInput.value.trim();
				if (!code) {
					return;
				}
				
				// Disable button and show loading state
				submitBtn.disabled = true;
				submitBtn.textContent = 'Processing...';
				errorMsg.style.display = 'none';
				successMsg.style.display = 'none';
				
				try {
					// The code from Anthropic contains both code and state separated by #
					const response = await fetch('/auth/anthropic/exchange', {
						method: 'POST',
						headers: {
							'Content-Type': 'application/json',
						},
						body: JSON.stringify({ code: code })
					});
					
					const data = await response.json();
					
					if (response.ok) {
						successMsg.style.display = 'block';
						setTimeout(() => {
							window.location.href = '/';
						}, 1500);
					} else {
						errorMsg.style.display = 'block';
						submitBtn.disabled = false;
						submitBtn.textContent = 'Submit Code';
					}
				} catch (error) {
					console.error('Error:', error);
					errorMsg.style.display = 'block';
					submitBtn.disabled = false;
					submitBtn.textContent = 'Submit Code';
				}
			}
			
			// Allow Enter key to submit
			document.getElementById('code-input').addEventListener('keypress', function(event) {
				if (event.key === 'Enter') {
					submitCode();
				}
			});
		`),
	)

	return b.String()
}
