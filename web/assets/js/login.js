window.handleLogin = async function() {
  console.log('handleLogin called');
  try {
    const response = await fetch('/auth/anthropic/oauth-url');
    const data = await response.json();
    console.log('OAuth URL response:', data);

    if (data.url) {
      // Open OAuth URL in new tab
      window.open(data.url, '_blank');
      // Redirect to callback page
      window.location.href = '/auth/callback';
    } else {
      alert('Failed to get authorization URL');
    }
  } catch (error) {
    console.error('Login error:', error);
    alert('Failed to initiate login');
  }
}
