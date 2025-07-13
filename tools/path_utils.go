package tools

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/rohanthewiz/serr"
)

// ExpandPath expands a file path, replacing ~ with the user's home directory
// This ensures that paths like ~/Documents/file.txt work correctly
func ExpandPath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Handle home directory expansion for Unix-like systems
	if strings.HasPrefix(path, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", serr.Wrap(err, "failed to get home directory")
		}

		if path == "~" {
			return homeDir, nil
		}

		// IMPORTANT SAFETY CHECK: Handle edge cases where path length is 2
		// Without this check, path[2:] at line 36 would panic with index out of range
		// We treat "~/" and "~/." the same as "~" (just the home directory)
		if len(path) == 2 {
			if path == "~/" || path == "~/." {
				return homeDir, nil
			} else {
				return "", serr.F("file path: %q is malformed", path)
			}
		}

		// Replace ~ with home directory
		path = filepath.Join(homeDir, path[2:])
	}

	// Clean the path to handle . and .. properly
	return filepath.Clean(path), nil
}
