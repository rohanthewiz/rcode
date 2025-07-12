package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rohanthewiz/serr"
	"golang.org/x/net/html"
)

// WebFetchTool implements web page fetching functionality
// This tool allows fetching and converting web page content
type WebFetchTool struct{}

// GetDefinition returns the tool definition
func (t *WebFetchTool) GetDefinition() Tool {
	return Tool{
		Name:        "web_fetch",
		Description: "Fetch and extract content from a web page URL",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "The URL to fetch content from",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Request timeout in seconds (default: 30, max: 120)",
					"default":     30,
				},
				"max_size": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum content size in bytes (default: 10MB)",
					"default":     10485760, // 10 MB
				},
				"follow_redirects": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to follow redirects (default: true)",
					"default":     true,
				},
			},
			"required": []string{"url"},
		},
	}
}

// Execute performs the web page fetch
func (t *WebFetchTool) Execute(input map[string]interface{}) (string, error) {
	// Extract and validate parameters
	urlStr, ok := GetString(input, "url")
	if !ok {
		return "", serr.New("url parameter is required")
	}

	// Validate URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", serr.Wrap(err, "invalid URL")
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", serr.New("only HTTP and HTTPS URLs are supported")
	}

	// Get optional parameters
	timeout := 30
	if timeoutParam, ok := GetInt(input, "timeout"); ok {
		timeout = timeoutParam
		if timeout > 120 {
			timeout = 120
		}
		if timeout < 1 {
			timeout = 1
		}
	}

	maxSize := 10485760 // 10MB default
	if maxSizeParam, ok := GetInt(input, "max_size"); ok {
		maxSize = maxSizeParam
		if maxSize > 52428800 { // 50MB hard limit
			maxSize = 52428800
		}
		if maxSize < 1024 { // 1KB minimum
			maxSize = 1024
		}
	}

	followRedirects := true
	if followParam, ok := input["follow_redirects"].(bool); ok {
		followRedirects = followParam
	}

	// Create HTTP client
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	if !followRedirects {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// Create request
	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", serr.Wrap(err, "failed to create request")
	}

	// Set user agent to avoid blocking
	req.Header.Set("User-Agent", "RCode-WebFetch/1.0 (compatible; like Mozilla/5.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	// Perform request
	resp, err := client.Do(req)
	if err != nil {
		return "", serr.Wrap(err, "failed to fetch URL")
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode >= 400 {
		return "", serr.New(fmt.Sprintf("HTTP error: %d %s", resp.StatusCode, resp.Status))
	}

	// Read response body with size limit
	limitedReader := io.LimitReader(resp.Body, int64(maxSize))
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", serr.Wrap(err, "failed to read response body")
	}

	// Get content type
	contentType := resp.Header.Get("Content-Type")
	contentTypeBase := strings.Split(contentType, ";")[0]
	contentTypeBase = strings.TrimSpace(strings.ToLower(contentTypeBase))

	// Process content based on type
	var content string
	switch {
	case strings.Contains(contentTypeBase, "html"):
		content = t.htmlToMarkdown(string(bodyBytes))
	case strings.Contains(contentTypeBase, "json"):
		content = t.formatJSON(bodyBytes)
	case strings.Contains(contentTypeBase, "text"):
		content = string(bodyBytes)
	default:
		content = string(bodyBytes)
	}

	// Format output
	var output strings.Builder
	output.WriteString(fmt.Sprintf("# Web Page Content\n\n"))
	output.WriteString(fmt.Sprintf("**URL:** %s\n", urlStr))
	output.WriteString(fmt.Sprintf("**Status:** %d %s\n", resp.StatusCode, resp.Status))
	output.WriteString(fmt.Sprintf("**Content-Type:** %s\n", contentType))
	output.WriteString(fmt.Sprintf("**Size:** %d bytes\n", len(bodyBytes)))
	if resp.Request.URL.String() != urlStr {
		output.WriteString(fmt.Sprintf("**Final URL:** %s (redirected)\n", resp.Request.URL.String()))
	}
	output.WriteString("\n---\n\n")
	output.WriteString(content)

	return output.String(), nil
}

// htmlToMarkdown converts HTML to markdown format
func (t *WebFetchTool) htmlToMarkdown(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return "Error parsing HTML: " + err.Error()
	}

	var buf bytes.Buffer
	t.convertNode(&buf, doc, 0)
	return buf.String()
}

// convertNode recursively converts HTML nodes to markdown
func (t *WebFetchTool) convertNode(buf *bytes.Buffer, n *html.Node, depth int) {
	switch n.Type {
	case html.TextNode:
		text := strings.TrimSpace(n.Data)
		if text != "" {
			buf.WriteString(text)
		}
	case html.ElementNode:
		t.convertElement(buf, n, depth)
	}

	// Process children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		t.convertNode(buf, c, depth)
	}

	// Add closing formatting
	switch n.Type {
	case html.ElementNode:
		t.addClosingFormat(buf, n, depth)
	}
}

// convertElement handles specific HTML elements
func (t *WebFetchTool) convertElement(buf *bytes.Buffer, n *html.Node, depth int) {
	switch n.Data {
	case "h1":
		buf.WriteString("\n# ")
	case "h2":
		buf.WriteString("\n## ")
	case "h3":
		buf.WriteString("\n### ")
	case "h4":
		buf.WriteString("\n#### ")
	case "h5":
		buf.WriteString("\n##### ")
	case "h6":
		buf.WriteString("\n###### ")
	case "p":
		buf.WriteString("\n\n")
	case "br":
		buf.WriteString("\n")
	case "strong", "b":
		buf.WriteString("**")
	case "em", "i":
		buf.WriteString("*")
	case "code":
		buf.WriteString("`")
	case "pre":
		buf.WriteString("\n```\n")
	case "ul", "ol":
		buf.WriteString("\n")
	case "li":
		buf.WriteString("\n- ")
	case "a":
		buf.WriteString("[")
	case "img":
		for _, attr := range n.Attr {
			if attr.Key == "alt" {
				buf.WriteString(fmt.Sprintf("![%s]", attr.Val))
			}
		}
	case "blockquote":
		buf.WriteString("\n> ")
	case "hr":
		buf.WriteString("\n---\n")
	}
}

// addClosingFormat adds closing markdown formatting
func (t *WebFetchTool) addClosingFormat(buf *bytes.Buffer, n *html.Node, depth int) {
	switch n.Data {
	case "h1", "h2", "h3", "h4", "h5", "h6":
		buf.WriteString("\n")
	case "strong", "b":
		buf.WriteString("**")
	case "em", "i":
		buf.WriteString("*")
	case "code":
		buf.WriteString("`")
	case "pre":
		buf.WriteString("\n```\n")
	case "a":
		buf.WriteString("]")
		// Add URL if present
		for _, attr := range n.Attr {
			if attr.Key == "href" {
				buf.WriteString(fmt.Sprintf("(%s)", attr.Val))
				break
			}
		}
	case "img":
		// Add src URL if present
		for _, attr := range n.Attr {
			if attr.Key == "src" {
				buf.WriteString(fmt.Sprintf("(%s)", attr.Val))
				break
			}
		}
	case "ul", "ol":
		buf.WriteString("\n")
	}
}

// formatJSON pretty-prints JSON content
func (t *WebFetchTool) formatJSON(jsonBytes []byte) string {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, jsonBytes, "", "  ")
	if err != nil {
		return string(jsonBytes) // Return original if formatting fails
	}
	return "```json\n" + prettyJSON.String() + "\n```"
}