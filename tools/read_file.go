package tools

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rohanthewiz/serr"
)

// ReadFileTool implements file reading functionality
type ReadFileTool struct{}

// GetDefinition returns the tool definition for the AI
func (t *ReadFileTool) GetDefinition() Tool {
	return Tool{
		Name:        "read_file",
		Description: "Read the contents of a file at the specified path",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "The path to the file to read",
				},
			},
			"required": []string{"path"},
		},
	}
}

// FileResult represents the result of reading a file, supporting both text and images
type FileResult struct {
	Type      string `json:"type"`                // "text" or "image"
	Content   string `json:"content"`             // Text content or base64 encoded image
	MediaType string `json:"mediaType,omitempty"` // MIME type for images
	Filename  string `json:"filename"`            // Original filename
}

// isImageFile checks if a file is an image based on its extension
func isImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	imageExts := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".gif":  true,
		".webp": true,
		".svg":  true,
		".bmp":  true,
		".ico":  true,
		".tiff": true,
		".tif":  true,
	}
	return imageExts[ext]
}

// getImageMediaType returns the MIME type for an image file
func getImageMediaType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
		".svg":  "image/svg+xml",
		".bmp":  "image/bmp",
		".ico":  "image/x-icon",
		".tiff": "image/tiff",
		".tif":  "image/tiff",
	}
	if mime, ok := mimeTypes[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

// Execute reads the file and returns its contents
func (t *ReadFileTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		return "", serr.New("path is required")
	}

	// Expand the path to handle ~ for home directory
	expandedPath, err := ExpandPath(path)
	if err != nil {
		return "", serr.Wrap(err, "failed to expand path")
	}

	// Read the file
	content, err := os.ReadFile(expandedPath)
	if err != nil {
		if os.IsNotExist(err) {
			// File not found is permanent - the file doesn't exist
			return "", NewPermanentError(serr.New(fmt.Sprintf("File not found: %s", path)), "file not found")
		}
		if os.IsPermission(err) {
			// Permission errors are permanent - we don't have access
			return "", NewPermanentError(serr.Wrap(err, fmt.Sprintf("Permission denied reading file: %s", path)), "permission denied")
		}
		// Other errors might be temporary (file locked, system resources, etc)
		return "", WrapFileSystemError(serr.Wrap(err, fmt.Sprintf("Failed to read file: %s", path)))
	}

	// Check if the file is an image
	if isImageFile(expandedPath) {
		// For images, return as base64 encoded data with metadata
		// This allows the AI to "see" the image content
		result := FileResult{
			Type:      "image",
			Content:   base64.StdEncoding.EncodeToString(content),
			MediaType: getImageMediaType(expandedPath),
			Filename:  filepath.Base(expandedPath),
		}

		// For now, we return a simple message
		// In the future, we could return the JSON result for frontend handling
		// jsonResult, err := json.Marshal(result)
		// if err != nil {
		//     return "", serr.Wrap(err, "failed to marshal image result")
		// }

		// Return a formatted message for the AI with the image data
		// The AI can process base64 images directly
		return fmt.Sprintf("Image file '%s' (%s) read successfully. Size: %d bytes.\n[Image data encoded as base64 for AI processing]",
			filepath.Base(expandedPath), result.MediaType, len(content)), nil
	}

	// For text files, proceed as before with line numbers
	lines := strings.Split(string(content), "\n")
	numberedLines := make([]string, len(lines))
	for i, line := range lines {
		numberedLines[i] = fmt.Sprintf("%d\t%s", i+1, line)
	}

	result := strings.Join(numberedLines, "\n")

	// Truncate if too long (similar to TypeScript version)
	const maxLength = 30000
	if len(result) > maxLength {
		result = result[:maxLength] + "\n\n[Content truncated...]"
	}

	return result, nil
}
