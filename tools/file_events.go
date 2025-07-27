package tools

import (
	"path/filepath"
	"strings"
)

// FileChangeNotifier is an interface for notifying file changes
type FileChangeNotifier interface {
	NotifyFileChanged(path string, changeType string)
	NotifyFileTreeUpdate(path string)
}

// Global file change notifier
var fileChangeNotifier FileChangeNotifier

// SetFileChangeNotifier sets the global file change notifier
func SetFileChangeNotifier(notifier FileChangeNotifier) {
	fileChangeNotifier = notifier
}

// NotifyFileChange broadcasts file change events through the notifier
func NotifyFileChange(path string, changeType string) {
	// If no notifier is set, just return
	if fileChangeNotifier == nil {
		return
	}

	// Convert to relative path if it's absolute
	relPath := path
	if filepath.IsAbs(path) {
		if cwd, err := filepath.Abs("."); err == nil {
			if rel, err := filepath.Rel(cwd, path); err == nil {
				relPath = rel
			}
		}
	}

	// Clean the path
	relPath = filepath.Clean(relPath)

	// Don't broadcast changes to certain files/directories
	ignorePaths := []string{
		".git/",
		"node_modules/",
		".DS_Store",
		"*.log",
		"*.tmp",
		".env",
	}

	for _, ignore := range ignorePaths {
		if strings.Contains(relPath, ignore) {
			return
		}
	}

	// Notify the file change event
	fileChangeNotifier.NotifyFileChanged(relPath, changeType)

	// Also notify tree update for the parent directory
	parentDir := filepath.Dir(relPath)
	if parentDir != "." && parentDir != "/" {
		fileChangeNotifier.NotifyFileTreeUpdate(parentDir)
	} else {
		fileChangeNotifier.NotifyFileTreeUpdate("")
	}
}
