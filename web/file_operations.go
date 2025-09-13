package web

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

// FileOperationRequest represents a file operation request
type FileOperationRequest struct {
	Paths     []string `json:"paths"`
	Target    string   `json:"target,omitempty"`
	Overwrite bool     `json:"overwrite,omitempty"`
	Recursive bool     `json:"recursive,omitempty"`
}

// FileListResponse represents directory listing response
type FileListResponse struct {
	Path  string     `json:"path"`
	Files []FileInfo `json:"files"`
}

// Protected paths that should not be modified
var protectedPaths = map[string]bool{
	".git":              true,
	"go.mod":            true,
	"go.sum":            true,
	"package.json":      true,
	"package-lock.json": true,
	"yarn.lock":         true,
	".env":              true,
	".env.local":        true,
}

// ListFilesHandler handles directory listing requests
func ListFilesHandler(c rweb.Context) error {
	path := c.Request().QueryParam("path")
	if path == "" {
		path = "."
	}

	// Validate path is within project directory
	absPath, err := filepath.Abs(path)
	if err != nil {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "Invalid path"})
	}

	projectRoot, _ := os.Getwd()
	if !strings.HasPrefix(absPath, projectRoot) {
		c.Response().SetStatus(403)
		return c.WriteJSON(map[string]string{"error": "Path outside project directory"})
	}

	// Read directory contents
	entries, err := os.ReadDir(absPath)
	if err != nil {
		c.Response().SetStatus(500)
		return c.WriteJSON(map[string]string{"error": fmt.Sprintf("Failed to read directory: %v", err)})
	}

	files := make([]FileInfo, 0, len(entries))
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Skip hidden files starting with . (except .env files which we show but protect)
		name := entry.Name()
		if strings.HasPrefix(name, ".") && !strings.HasPrefix(name, ".env") && name != ".gitignore" {
			continue
		}

		files = append(files, FileInfo{
			Path:     filepath.Join(absPath, name),
			Name:     name,
			IsDir:    entry.IsDir(),
			Size:     info.Size(),
			Modified: info.ModTime(),
		})
	}

	return c.WriteJSON(FileListResponse{
		Path:  absPath,
		Files: files,
	})
}

// CopyFilesHandler handles copy to clipboard requests
func CopyFilesHandler(c rweb.Context) error {
	sessionID := c.Request().Header("X-Session-ID")
	if sessionID == "" {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "Session ID required"})
	}

	var req FileOperationRequest
	if err := json.Unmarshal(c.Request().Body(), &req); err != nil {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "Invalid request"})
	}

	// Validate paths and collect file info
	files := make([]FileInfo, 0, len(req.Paths))
	for _, path := range req.Paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		// Check if path is within project
		projectRoot, _ := os.Getwd()
		if !strings.HasPrefix(absPath, projectRoot) {
			c.Response().SetStatus(403)
			return c.WriteJSON(map[string]string{"error": fmt.Sprintf("Path outside project: %s", path)})
		}

		// Get file info
		info, err := os.Stat(absPath)
		if err != nil {
			c.Response().SetStatus(404)
			return c.WriteJSON(map[string]string{"error": fmt.Sprintf("File not found: %s", path)})
		}

		files = append(files, FileInfo{
			Path:     absPath,
			Name:     filepath.Base(absPath),
			IsDir:    info.IsDir(),
			Size:     info.Size(),
			Modified: info.ModTime(),
		})
	}

	// Update clipboard
	clipboardManager.Set(sessionID, ClipboardModeCopy, files)

	return c.WriteJSON(map[string]interface{}{
		"message": fmt.Sprintf("Copied %d items to clipboard", len(files)),
		"count":   len(files),
	})
}

// CutFilesHandler handles cut to clipboard requests
func CutFilesHandler(c rweb.Context) error {
	sessionID := c.Request().Header("X-Session-ID")
	if sessionID == "" {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "Session ID required"})
	}

	var req FileOperationRequest
	if err := json.Unmarshal(c.Request().Body(), &req); err != nil {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "Invalid request"})
	}

	// Validate paths and collect file info
	files := make([]FileInfo, 0, len(req.Paths))
	for _, path := range req.Paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		// Check if path is within project
		projectRoot, _ := os.Getwd()
		if !strings.HasPrefix(absPath, projectRoot) {
			c.Response().SetStatus(403)
			return c.WriteJSON(map[string]string{"error": fmt.Sprintf("Path outside project: %s", path)})
		}

		// Check if file is protected
		if isProtectedPath(absPath) {
			c.Response().SetStatus(403)
			return c.WriteJSON(map[string]string{"error": fmt.Sprintf("Cannot cut protected file: %s", path)})
		}

		// Get file info
		info, err := os.Stat(absPath)
		if err != nil {
			c.Response().SetStatus(404)
			return c.WriteJSON(map[string]string{"error": fmt.Sprintf("File not found: %s", path)})
		}

		files = append(files, FileInfo{
			Path:     absPath,
			Name:     filepath.Base(absPath),
			IsDir:    info.IsDir(),
			Size:     info.Size(),
			Modified: info.ModTime(),
		})
	}

	// Update clipboard
	clipboardManager.Set(sessionID, ClipboardModeCut, files)

	return c.WriteJSON(map[string]interface{}{
		"message": fmt.Sprintf("Cut %d items to clipboard", len(files)),
		"count":   len(files),
	})
}

// PasteFilesHandler handles paste from clipboard requests
func PasteFilesHandler(c rweb.Context) error {
	sessionID := c.Request().Header("X-Session-ID")
	if sessionID == "" {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "Session ID required"})
	}

	var req FileOperationRequest
	if err := json.Unmarshal(c.Request().Body(), &req); err != nil {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "Invalid request"})
	}

	// Get clipboard content
	clipboard := clipboardManager.Get(sessionID)
	if clipboard == nil || len(clipboard.Files) == 0 {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "Clipboard is empty"})
	}

	// Validate target directory
	targetPath, err := filepath.Abs(req.Target)
	if err != nil {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "Invalid target path"})
	}

	projectRoot, _ := os.Getwd()
	if !strings.HasPrefix(targetPath, projectRoot) {
		c.Response().SetStatus(403)
		return c.WriteJSON(map[string]string{"error": "Target outside project directory"})
	}

	// Check if target exists and is a directory
	targetInfo, err := os.Stat(targetPath)
	if err != nil {
		c.Response().SetStatus(404)
		return c.WriteJSON(map[string]string{"error": "Target directory not found"})
	}
	if !targetInfo.IsDir() {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "Target must be a directory"})
	}

	// Perform paste operation
	successCount := 0
	errors := []string{}

	for _, file := range clipboard.Files {
		destPath := filepath.Join(targetPath, filepath.Base(file.Path))

		// Check if destination already exists
		if _, err := os.Stat(destPath); err == nil && !req.Overwrite {
			errors = append(errors, fmt.Sprintf("%s already exists", filepath.Base(file.Path)))
			continue
		}

		// Perform operation based on clipboard mode
		if clipboard.Mode == ClipboardModeCopy {
			if err := copyPath(file.Path, destPath); err != nil {
				errors = append(errors, fmt.Sprintf("Failed to copy %s: %v", file.Name, err))
			} else {
				successCount++
			}
		} else if clipboard.Mode == ClipboardModeCut {
			if err := os.Rename(file.Path, destPath); err != nil {
				// If rename fails (e.g., cross-device), try copy and delete
				if err := copyPath(file.Path, destPath); err != nil {
					errors = append(errors, fmt.Sprintf("Failed to move %s: %v", file.Name, err))
				} else {
					if err := os.RemoveAll(file.Path); err != nil {
						logger.LogErr(err, "Failed to remove source after copy", "path", file.Path)
					}
					successCount++
				}
			} else {
				successCount++
			}
		}
	}

	// Clear clipboard if it was a cut operation
	if clipboard.Mode == ClipboardModeCut && successCount > 0 {
		clipboardManager.Clear(sessionID)
	}

	response := map[string]interface{}{
		"message": fmt.Sprintf("Pasted %d of %d items", successCount, len(clipboard.Files)),
		"success": successCount,
		"total":   len(clipboard.Files),
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	return c.WriteJSON(response)
}

// DeleteFilesHandler handles delete files requests
func DeleteFilesHandler(c rweb.Context) error {
	var req FileOperationRequest
	if err := json.Unmarshal(c.Request().Body(), &req); err != nil {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "Invalid request"})
	}

	successCount := 0
	errors := []string{}

	for _, path := range req.Paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Invalid path: %s", path))
			continue
		}

		// Check if path is within project
		projectRoot, _ := os.Getwd()
		if !strings.HasPrefix(absPath, projectRoot) {
			errors = append(errors, fmt.Sprintf("Path outside project: %s", path))
			continue
		}

		// Check if file is protected
		if isProtectedPath(absPath) {
			errors = append(errors, fmt.Sprintf("Cannot delete protected file: %s", path))
			continue
		}

		// Check if it's a directory and recursive flag is needed
		info, err := os.Stat(absPath)
		if err != nil {
			errors = append(errors, fmt.Sprintf("File not found: %s", path))
			continue
		}

		if info.IsDir() && !req.Recursive {
			errors = append(errors, fmt.Sprintf("Directory requires recursive flag: %s", path))
			continue
		}

		// Delete the file or directory
		if info.IsDir() {
			err = os.RemoveAll(absPath)
		} else {
			err = os.Remove(absPath)
		}

		if err != nil {
			errors = append(errors, fmt.Sprintf("Failed to delete %s: %v", path, err))
		} else {
			successCount++
		}
	}

	response := map[string]interface{}{
		"message": fmt.Sprintf("Deleted %d of %d items", successCount, len(req.Paths)),
		"success": successCount,
		"total":   len(req.Paths),
	}

	if len(errors) > 0 {
		response["errors"] = errors
	}

	return c.WriteJSON(response)
}

// GetClipboardHandler returns current clipboard status
func GetClipboardHandler(c rweb.Context) error {
	sessionID := c.Request().Header("X-Session-ID")
	if sessionID == "" {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "Session ID required"})
	}

	clipboard := clipboardManager.Get(sessionID)
	if clipboard == nil {
		return c.WriteJSON(map[string]interface{}{
			"empty": true,
		})
	}

	return c.WriteJSON(clipboard)
}

// ClearClipboardHandler clears the clipboard
func ClearClipboardHandler(c rweb.Context) error {
	sessionID := c.Request().Header("X-Session-ID")
	if sessionID == "" {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "Session ID required"})
	}

	clipboardManager.Clear(sessionID)
	return c.WriteJSON(map[string]string{"message": "Clipboard cleared"})
}

// Helper function to check if a path is protected
func isProtectedPath(path string) bool {
	base := filepath.Base(path)
	return protectedPaths[base]
}

// Helper function to copy a file or directory
func copyPath(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return serr.Wrap(err, "failed to stat source")
	}

	if srcInfo.IsDir() {
		return copyDir(src, dst)
	}
	return copyFile(src, dst)
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return serr.Wrap(err, "failed to open source file")
	}
	defer sourceFile.Close()

	// Get source file info for permissions
	srcInfo, err := sourceFile.Stat()
	if err != nil {
		return serr.Wrap(err, "failed to stat source file")
	}

	// Create destination file
	destFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return serr.Wrap(err, "failed to create destination file")
	}
	defer destFile.Close()

	// Copy content
	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return serr.Wrap(err, "failed to copy file content")
	}

	return nil
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	// Get source directory info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return serr.Wrap(err, "failed to stat source directory")
	}

	// Create destination directory
	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return serr.Wrap(err, "failed to create destination directory")
	}

	// Read source directory contents
	entries, err := os.ReadDir(src)
	if err != nil {
		return serr.Wrap(err, "failed to read source directory")
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
