package server

import (
	"net/http"
	"os"

	"github.com/rohanthewiz/rweb"
	"github.com/sst/opencode/server-go/internal/auth"
)

// Provider represents an AI provider configuration
type ProviderConfig struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Models      []ModelConfig  `json:"models"`
}

// ModelConfig represents a model configuration
type ModelConfig struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Provider    string                 `json:"provider"`
	ToolSupport bool                   `json:"tool_support"`
	Info        map[string]interface{} `json:"info,omitempty"`
}

// configHandler returns the server configuration
func (s *Server) configHandler(c rweb.Context) error {
	config := map[string]interface{}{
		"version": "0.1.0",
		"server":  "opencode-go",
		"features": map[string]bool{
			"tools":     true,
			"streaming": true,
			"sessions":  true,
		},
	}
	
	return c.JSON(http.StatusOK, config)
}

// providersHandler returns available providers
func (s *Server) providersHandler(c rweb.Context) error {
	providers := []ProviderConfig{
		{
			ID:          "anthropic",
			Name:        "Anthropic",
			Description: "Claude AI models",
			Models: []ModelConfig{
				{
					ID:          "claude-3-opus-20240229",
					Name:        "Claude 3 Opus",
					Provider:    "anthropic",
					ToolSupport: true,
					Info: map[string]interface{}{
						"context_window": 200000,
						"max_output":     4096,
					},
				},
				{
					ID:          "claude-3-sonnet-20240229",
					Name:        "Claude 3 Sonnet",
					Provider:    "anthropic",
					ToolSupport: true,
					Info: map[string]interface{}{
						"context_window": 200000,
						"max_output":     4096,
					},
				},
				{
					ID:          "claude-3-haiku-20240307",
					Name:        "Claude 3 Haiku",
					Provider:    "anthropic",
					ToolSupport: true,
					Info: map[string]interface{}{
						"context_window": 200000,
						"max_output":     4096,
					},
				},
			},
		},
	}
	
	// Check authentication status
	storage := auth.NewStorage()
	hasAnthropicKey := os.Getenv("ANTHROPIC_API_KEY") != ""
	hasAnthropicOAuth := false
	isProMax := false
	
	if creds, err := storage.Get("anthropic"); err == nil && creds != nil {
		hasAnthropicOAuth = creds.Type == "oauth"
		isProMax = hasAnthropicOAuth // OAuth means Pro/Max user
	}
	
	authStatus := map[string]interface{}{
		"apikey": hasAnthropicKey,
		"oauth":  hasAnthropicOAuth,
		"promax": isProMax,
	}
	
	return c.JSON(http.StatusOK, map[string]interface{}{
		"providers": providers,
		"configured": map[string]bool{
			"anthropic": hasAnthropicKey || hasAnthropicOAuth,
		},
		"auth": map[string]interface{}{
			"anthropic": authStatus,
		},
	})
}

// modelsHandler returns available models for a provider
func (s *Server) modelsHandler(c rweb.Context) error {
	providerID := c.Param("provider")
	
	models := []ModelConfig{}
	
	switch providerID {
	case "anthropic":
		models = []ModelConfig{
			{
				ID:          "claude-3-opus-20240229",
				Name:        "Claude 3 Opus",
				Provider:    "anthropic",
				ToolSupport: true,
			},
			{
				ID:          "claude-3-sonnet-20240229",
				Name:        "Claude 3 Sonnet",
				Provider:    "anthropic",
				ToolSupport: true,
			},
			{
				ID:          "claude-3-haiku-20240307",
				Name:        "Claude 3 Haiku",
				Provider:    "anthropic",
				ToolSupport: true,
			},
		}
	}
	
	return c.JSON(http.StatusOK, models)
}