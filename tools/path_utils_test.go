package tools

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	// Get home directory for comparison
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("Failed to get home directory: %v", err)
	}

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name:  "expand home directory with file",
			input: "~/xfr/hi.go",
			want:  filepath.Join(homeDir, "xfr/hi.go"),
		},
		{
			name:  "expand just tilde",
			input: "~",
			want:  homeDir,
		},
		{
			name:  "expand tilde slash",
			input: "~/",
			want:  homeDir,
		},
		{
			name:  "expand tilde dot",
			input: "~/.",
			want:  homeDir,
		},
		{
			name:  "expand home directory with subdirectory",
			input: "~/Documents",
			want:  filepath.Join(homeDir, "Documents"),
		},
		{
			name:  "absolute path unchanged",
			input: "/absolute/path",
			want:  "/absolute/path",
		},
		{
			name:  "relative path cleaned",
			input: "relative/path",
			want:  "relative/path",
		},
		{
			name:  "current directory path cleaned",
			input: "./current/path",
			want:  "current/path",
		},
		{
			name:  "empty path returns empty",
			input: "",
			want:  "",
		},
		{
			name:  "path with double dots cleaned",
			input: "~/test/../Documents",
			want:  filepath.Join(homeDir, "Documents"),
		},
		{
			name:    "malformed tilde path",
			input:   "~x",
			wantErr: true,
			errMsg:  "file path: \"~x\" is malformed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExpandPath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ExpandPath(%q) expected error containing %q, but got no error", tt.input, tt.errMsg)
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("ExpandPath(%q) error = %q, want error containing %q", tt.input, err.Error(), tt.errMsg)
				}
				return
			}
			if err != nil {
				t.Errorf("ExpandPath(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExpandPathWithReadFileTool(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello, World!"

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test reading a file using ReadFileTool
	tool := &ReadFileTool{}

	// Use the actual file path (not with ~)
	input := map[string]interface{}{
		"path": testFile,
	}

	// result should contain the file content with line numbers
	result, err := tool.Execute(input)
	if err != nil {
		t.Errorf("ReadFileTool.Execute() error = %v", err)
		return
	}

	// Check if the content was read correctly
	if result == "" {
		t.Errorf("ReadFileTool.Execute() returned empty result")
		return
	}

	// Check if the content contains our test string
	if !contains(result, testContent) {
		t.Errorf("ReadFileTool.Execute() result does not contain expected content %q", testContent)
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
