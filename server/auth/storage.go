package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/rohanthewiz/serr"
)

// TokenStorage handles persistent storage of OAuth tokens
type TokenStorage struct {
	filePath string
}

// NewTokenStorage creates a new token storage instance
func NewTokenStorage() (*TokenStorage, error) {
	// Use ~/.local/share/opencode/auth.json similar to the TypeScript version
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, serr.Wrap(err, "failed to get home directory")
	}

	dataDir := filepath.Join(homeDir, ".local", "share", "opencode")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, serr.Wrap(err, "failed to create data directory")
	}

	return &TokenStorage{
		filePath: filepath.Join(dataDir, "auth.json"),
	}, nil
}

// AuthData represents the stored authentication data
type AuthData struct {
	Anthropic *StoredTokens `json:"anthropic,omitempty"`
}

// StoredTokens represents stored OAuth tokens
type StoredTokens struct {
	TokenResponse
	UpdatedAt time.Time `json:"updated_at"`
}

// Load reads tokens from persistent storage
func (ts *TokenStorage) Load() (*AuthData, error) {
	data, err := os.ReadFile(ts.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty auth data if file doesn't exist
			return &AuthData{}, nil
		}
		return nil, serr.Wrap(err, "failed to read auth file")
	}

	var authData AuthData
	if err := json.Unmarshal(data, &authData); err != nil {
		return nil, serr.Wrap(err, "failed to parse auth data")
	}

	return &authData, nil
}

// Save writes tokens to persistent storage
func (ts *TokenStorage) Save(authData *AuthData) error {
	data, err := json.MarshalIndent(authData, "", "  ")
	if err != nil {
		return serr.Wrap(err, "failed to marshal auth data")
	}

	// Write with restricted permissions (0600)
	if err := os.WriteFile(ts.filePath, data, 0600); err != nil {
		return serr.Wrap(err, "failed to write auth file")
	}

	return nil
}

// GetAnthropicTokens retrieves Anthropic tokens from storage
func (ts *TokenStorage) GetAnthropicTokens() (*StoredTokens, error) {
	authData, err := ts.Load()
	if err != nil {
		return nil, err
	}

	if authData.Anthropic == nil {
		return nil, serr.New("no Anthropic tokens found - please authenticate first")
	}

	return authData.Anthropic, nil
}

// SaveAnthropicTokens saves Anthropic tokens to storage
func (ts *TokenStorage) SaveAnthropicTokens(tokens *TokenResponse) error {
	authData, err := ts.Load()
	if err != nil {
		return err
	}

	authData.Anthropic = &StoredTokens{
		TokenResponse: *tokens,
		UpdatedAt:     time.Now(),
	}

	return ts.Save(authData)
}

// RemoveAnthropicTokens removes Anthropic tokens from storage
func (ts *TokenStorage) RemoveAnthropicTokens() error {
	authData, err := ts.Load()
	if err != nil {
		return err
	}

	authData.Anthropic = nil

	return ts.Save(authData)
}
