package config

import (
	"os"
	"path/filepath"
	"strings"
)

const (
	// Default Anthropic API URL
	defaultAnthropicAPIURL = "https://api.anthropic.com/v1/messages"
)

// Config holds application configuration
type Config struct {
	AnthropicAPIURL string
	// TLS configuration
	TLSEnabled  bool
	TLSPort     string
	TLSCertFile string
	TLSKeyFile  string
	// Custom tool configuration
	CustomToolsEnabled bool
	CustomToolsPaths   []string // Directories to search for custom tools
	CustomToolsConfig  string   // Path to custom tools config file
}

// globalConfig holds the application configuration instance
var globalConfig *Config

// Initialize sets up the configuration from environment variables
func Initialize() {
	globalConfig = &Config{
		AnthropicAPIURL:    getAnthropicAPIURL(),
		TLSEnabled:         getTLSEnabled(),
		TLSPort:            getTLSPort(),
		TLSCertFile:        getTLSCertFile(),
		TLSKeyFile:         getTLSKeyFile(),
		CustomToolsEnabled: getCustomToolsEnabled(),
		CustomToolsPaths:   getCustomToolsPaths(),
		CustomToolsConfig:  getCustomToolsConfig(),
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

// getTLSEnabled returns whether TLS is enabled from environment
func getTLSEnabled() bool {
	return os.Getenv("RCODE_TLS_ENABLED") == "true"
}

// getTLSPort returns the TLS port from environment or default
func getTLSPort() string {
	if port := os.Getenv("RCODE_TLS_PORT"); port != "" {
		return port
	}
	return ":8443" // Default HTTPS port for non-privileged
}

// getTLSCertFile returns the certificate file path from environment or default
func getTLSCertFile() string {
	if cert := os.Getenv("RCODE_TLS_CERT"); cert != "" {
		return cert
	}
	return "certs/localhost.crt" // Default certificate path
}

// getTLSKeyFile returns the key file path from environment or default
func getTLSKeyFile() string {
	if key := os.Getenv("RCODE_TLS_KEY"); key != "" {
		return key
	}
	return "certs/localhost.key" // Default key path
}

// getCustomToolsEnabled returns whether custom tools are enabled from environment
func getCustomToolsEnabled() bool {
	return os.Getenv("RCODE_CUSTOM_TOOLS_ENABLED") == "true"
}

// getCustomToolsPaths returns the directories to search for custom tools
func getCustomToolsPaths() []string {
	paths := []string{
		filepath.Join(os.Getenv("HOME"), ".rcode", "tools"),
		"/usr/local/lib/rcode/tools",
	}

	if envPaths := os.Getenv("RCODE_CUSTOM_TOOLS_PATHS"); envPaths != "" {
		paths = append(paths, strings.Split(envPaths, ":")...)
	}

	return paths
}

// getCustomToolsConfig returns the path to custom tools config file
func getCustomToolsConfig() string {
	if config := os.Getenv("RCODE_CUSTOM_TOOLS_CONFIG"); config != "" {
		return config
	}
	return filepath.Join(os.Getenv("HOME"), ".rcode", "tools.json")
}
