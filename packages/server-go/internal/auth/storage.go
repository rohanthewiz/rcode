package auth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/rohanthewiz/serr"
)

// Storage handles secure storage of authentication credentials
type Storage struct {
	mu       sync.RWMutex
	filePath string
}

// Credentials represents stored authentication credentials
type Credentials struct {
	Type    string `json:"type"`    // "oauth" or "apikey"
	Access  string `json:"access,omitempty"`
	Refresh string `json:"refresh,omitempty"`
	Expires int64  `json:"expires,omitempty"`
	APIKey  string `json:"apikey,omitempty"`
}

// NewStorage creates a new storage instance
func NewStorage() *Storage {
	// Use XDG_DATA_HOME or default to ~/.local/share
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, _ := os.UserHomeDir()
		dataHome = filepath.Join(home, ".local", "share")
	}

	return &Storage{
		filePath: filepath.Join(dataHome, "opencode", "auth.json"),
	}
}

// Get retrieves credentials for a provider
func (s *Storage) Get(provider string) (*Credentials, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Read auth file
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No credentials stored
		}
		return nil, serr.Wrap(err, "failed to read auth file")
	}

	// Parse JSON
	var auth map[string]*Credentials
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, serr.Wrap(err, "failed to parse auth file")
	}

	return auth[provider], nil
}

// Set stores credentials for a provider
func (s *Storage) Set(provider string, creds *Credentials) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return serr.Wrap(err, "failed to create auth directory")
	}

	// Read existing auth data
	auth := make(map[string]*Credentials)
	if data, err := os.ReadFile(s.filePath); err == nil {
		json.Unmarshal(data, &auth)
	}

	// Update credentials
	auth[provider] = creds

	// Write to file with restricted permissions (0600)
	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return serr.Wrap(err, "failed to marshal auth data")
	}

	if err := os.WriteFile(s.filePath, data, 0600); err != nil {
		return serr.Wrap(err, "failed to write auth file")
	}

	return nil
}

// Delete removes credentials for a provider
func (s *Storage) Delete(provider string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Read existing auth data
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to delete
		}
		return serr.Wrap(err, "failed to read auth file")
	}

	// Parse JSON
	var auth map[string]*Credentials
	if err := json.Unmarshal(data, &auth); err != nil {
		return serr.Wrap(err, "failed to parse auth file")
	}

	// Remove provider
	delete(auth, provider)

	// Write updated data
	updatedData, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return serr.Wrap(err, "failed to marshal auth data")
	}

	if err := os.WriteFile(s.filePath, updatedData, 0600); err != nil {
		return serr.Wrap(err, "failed to write auth file")
	}

	return nil
}

// List returns all stored provider names
func (s *Storage) List() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Read auth file
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, serr.Wrap(err, "failed to read auth file")
	}

	// Parse JSON
	var auth map[string]*Credentials
	if err := json.Unmarshal(data, &auth); err != nil {
		return nil, serr.Wrap(err, "failed to parse auth file")
	}

	// Extract provider names
	providers := make([]string, 0, len(auth))
	for provider := range auth {
		providers = append(providers, provider)
	}

	return providers, nil
}

// GetFilePath returns the auth file path (for debugging)
func (s *Storage) GetFilePath() string {
	return s.filePath
}