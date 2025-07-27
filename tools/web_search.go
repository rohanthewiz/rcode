package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rohanthewiz/serr"
)

// WebSearchTool implements web search functionality
// This tool allows searching the web and retrieving results
type WebSearchTool struct{}

// WebSearchResult represents a single web search result
type WebSearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

// WebSearchResponse represents the response from a search API
type WebSearchResponse struct {
	Results []WebSearchResult `json:"results"`
	Total   int               `json:"total"`
}

// GetDefinition returns the tool definition
func (t *WebSearchTool) GetDefinition() Tool {
	return Tool{
		Name:        "web_search",
		Description: "Search the web for information using a search query",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "The search query",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return (default: 10, max: 50)",
					"default":     10,
				},
			},
			"required": []string{"query"},
		},
	}
}

// Execute performs the web search
func (t *WebSearchTool) Execute(input map[string]interface{}) (string, error) {
	// Extract and validate parameters
	query, ok := GetString(input, "query")
	if !ok {
		return "", serr.New("invalid query parameter")
	}

	maxResults := 10
	if _, exists := input["max_results"]; exists {
		if maxResultsInt, ok := GetInt(input, "max_results"); ok {
			maxResults = maxResultsInt
			if maxResults > 50 {
				maxResults = 50
			}
			if maxResults < 1 {
				maxResults = 1
			}
		}
	}

	// Perform the search
	results, err := t.performSearch(query, maxResults)
	if err != nil {
		// Already classified in performSearch
		return "", err
	}

	// Format results
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Web Search Results for: \"%s\"\n", query))
	output.WriteString(fmt.Sprintf("Found %d results:\n\n", len(results.Results)))

	for i, result := range results.Results {
		output.WriteString(fmt.Sprintf("%d. %s\n", i+1, result.Title))
		output.WriteString(fmt.Sprintf("   URL: %s\n", result.URL))
		output.WriteString(fmt.Sprintf("   %s\n\n", result.Snippet))
	}

	return output.String(), nil
}

// performSearch executes the actual search using DuckDuckGo API
// We use DuckDuckGo as it doesn't require API keys
func (t *WebSearchTool) performSearch(query string, maxResults int) (*WebSearchResponse, error) {
	// Using DuckDuckGo's HTML API and parsing it
	// For a production system, you might want to use a proper search API with authentication
	searchURL := fmt.Sprintf("https://html.duckduckgo.com/html/?q=%s", url.QueryEscape(query))

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, NewPermanentError(serr.Wrap(err, "failed to create request"), "invalid request")
	}

	// Set user agent to avoid blocking
	req.Header.Set("User-Agent", "RCode-Search/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, WrapNetworkError(serr.Wrap(err, "failed to perform search request"))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		httpErr := serr.New(fmt.Sprintf("search request failed with status: %d", resp.StatusCode))
		switch resp.StatusCode {
		case 429:
			return nil, NewRateLimitError(httpErr, 60)
		case 500, 502, 503, 504:
			return nil, NewRetryableError(httpErr, "server error")
		case 400, 401, 403, 404:
			return nil, NewPermanentError(httpErr, "client error")
		default:
			if resp.StatusCode >= 500 {
				return nil, NewRetryableError(httpErr, "server error")
			}
			return nil, NewPermanentError(httpErr, "client error")
		}
	}

	// For now, we'll use a mock response since parsing HTML from DuckDuckGo
	// would require additional HTML parsing dependencies
	// In a production system, you'd want to use a proper search API
	return t.getMockResults(query, maxResults), nil
}

// getMockResults returns mock search results for demonstration
// In production, this would be replaced with actual API parsing
func (t *WebSearchTool) getMockResults(query string, maxResults int) *WebSearchResponse {
	// This is a placeholder implementation
	// In a real implementation, you would:
	// 1. Use a proper search API (Google Custom Search, Bing Search API, etc.)
	// 2. Parse the actual results
	// 3. Handle pagination if needed

	results := []WebSearchResult{
		{
			Title:   fmt.Sprintf("Search results for: %s", query),
			URL:     fmt.Sprintf("https://example.com/search?q=%s", url.QueryEscape(query)),
			Snippet: "This is a mock search result. In production, this would contain actual search results from a web search API.",
		},
	}

	// Add a note about implementation
	if len(results) > 0 {
		results = append(results, WebSearchResult{
			Title:   "Note: Mock Implementation",
			URL:     "https://developers.google.com/custom-search/v1/overview",
			Snippet: "This web search tool currently returns mock results. For production use, integrate with Google Custom Search API, Bing Search API, or similar service.",
		})
	}

	return &WebSearchResponse{
		Results: results,
		Total:   len(results),
	}
}

// Alternative implementation using Google Custom Search API (requires API key)
func (t *WebSearchTool) performGoogleSearch(query string, maxResults int, apiKey string, searchEngineID string) (*WebSearchResponse, error) {
	searchURL := fmt.Sprintf("https://www.googleapis.com/customsearch/v1?key=%s&cx=%s&q=%s&num=%d",
		apiKey, searchEngineID, url.QueryEscape(query), maxResults)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(searchURL)
	if err != nil {
		return nil, WrapNetworkError(serr.Wrap(err, "failed to perform Google search"))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		httpErr := serr.New(fmt.Sprintf("Google search failed with status %d: %s", resp.StatusCode, string(body)))
		switch resp.StatusCode {
		case 429:
			return nil, NewRateLimitError(httpErr, 60)
		case 500, 502, 503, 504:
			return nil, NewRetryableError(httpErr, "server error")
		case 400, 401, 403, 404:
			return nil, NewPermanentError(httpErr, "client error")
		default:
			if resp.StatusCode >= 500 {
				return nil, NewRetryableError(httpErr, "server error")
			}
			return nil, NewPermanentError(httpErr, "client error")
		}
	}

	var googleResponse struct {
		Items []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"items"`
		SearchInformation struct {
			TotalResults string `json:"totalResults"`
		} `json:"searchInformation"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&googleResponse); err != nil {
		// JSON decode errors might be temporary (truncated response)
		return nil, NewRetryableError(serr.Wrap(err, "failed to decode Google search response"), "decode error")
	}

	results := make([]WebSearchResult, 0, len(googleResponse.Items))
	for _, item := range googleResponse.Items {
		results = append(results, WebSearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: item.Snippet,
		})
	}

	return &WebSearchResponse{
		Results: results,
		Total:   len(results),
	}, nil
}
