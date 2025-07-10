package config

import (
	"os"
)

const (
	// Default Anthropic API URL
	defaultAnthropicAPIURL = "https://api.anthropic.com/v1/messages"
)

// Config holds application configuration
type Config struct {
	AnthropicAPIURL string
}

// globalConfig holds the application configuration instance
var globalConfig *Config

// Initialize sets up the configuration from environment variables
func Initialize() {
	globalConfig = &Config{
		AnthropicAPIURL: getAnthropicAPIURL(),
	}
}

// Get returns the global configuration instance
func Get() *Config {
	if globalConfig == nil {
		Initialize()
	}
	return globalConfig
}

// getAnthropicAPIURL returns the API URL from environment or default
func getAnthropicAPIURL() string {
	// Check for MSG_PROXY environment variable
	if proxyURL := os.Getenv("MSG_PROXY"); proxyURL != "" {
		// If MSG_PROXY is set, append the messages endpoint
		return proxyURL + "/v1/messages"
	}
	// Otherwise use the direct Anthropic API URL
	return defaultAnthropicAPIURL
}