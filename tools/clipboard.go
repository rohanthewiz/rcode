package tools

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rohanthewiz/serr"
)

// ClipboardPasteTool handles clipboard content from the frontend
type ClipboardPasteTool struct{}

// GetDefinition returns the tool definition for the AI
func (t *ClipboardPasteTool) GetDefinition() Tool {
	return Tool{
		Name:        "clipboard_paste",
		Description: "Handle pasted content from clipboard, including images and text",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The content from clipboard (base64 for images, plain text for text)",
				},
				"contentType": map[string]interface{}{
					"type":        "string",
					"description": "Type of content: 'image' or 'text'",
				},
				"mediaType": map[string]interface{}{
					"type":        "string",
					"description": "MIME type for images (e.g., 'image/png', 'image/jpeg')",
				},
				"saveToFile": map[string]interface{}{
					"type":        "boolean",
					"description": "Whether to save the content to a temporary file",
				},
			},
			"required": []string{"content", "contentType"},
		},
	}
}

// ClipboardResult represents the result of handling clipboard content
type ClipboardResult struct {
	Type      string `json:"type"`                // "image" or "text"
	Content   string `json:"content,omitempty"`   // Text content or base64 image (if not saved to file)
	FilePath  string `json:"filePath,omitempty"`  // Path to saved file (if saveToFile was true)
	MediaType string `json:"mediaType,omitempty"` // MIME type for images
	Size      int    `json:"size"`                // Size in bytes
}

// Execute processes the clipboard content
func (t *ClipboardPasteTool) Execute(input map[string]interface{}) (string, error) {
	content, ok := GetString(input, "content")
	if !ok || content == "" {
		return "", serr.New("content is required")
	}

	contentType, ok := GetString(input, "contentType")
	if !ok || contentType == "" {
		return "", serr.New("contentType is required")
	}

	// Validate content type
	if contentType != "image" && contentType != "text" {
		return "", serr.New("contentType must be 'image' or 'text'")
	}

	mediaType, _ := GetString(input, "mediaType")
	saveToFile, _ := GetBool(input, "saveToFile")

	result := ClipboardResult{
		Type: contentType,
	}

	if contentType == "image" {
		// Validate base64 encoding for images
		imageData, err := base64.StdEncoding.DecodeString(content)
		if err != nil {
			return "", serr.Wrap(err, "invalid base64 image data")
		}

		result.Size = len(imageData)
		result.MediaType = mediaType
		if result.MediaType == "" {
			// Try to detect media type from image data
			result.MediaType = detectImageType(imageData)
		}

		if saveToFile {
			// Save image to temporary file
			tempDir := os.TempDir()
			timestamp := time.Now().Format("20060102_150405")
			ext := getExtensionFromMimeType(result.MediaType)
			fileName := fmt.Sprintf("clipboard_image_%s%s", timestamp, ext)
			filePath := filepath.Join(tempDir, "rcode", fileName)

			// Create directory if it doesn't exist
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return "", serr.Wrap(err, "failed to create temp directory")
			}

			// Write image data to file
			if err := ioutil.WriteFile(filePath, imageData, 0644); err != nil {
				return "", serr.Wrap(err, "failed to save image to file")
			}

			result.FilePath = filePath

			// Return success message with file path
			return fmt.Sprintf("Image saved successfully to: %s\nSize: %d bytes\nType: %s",
				filePath, result.Size, result.MediaType), nil
		} else {
			// Keep image as base64 in result
			result.Content = content

			// Return success message
			return fmt.Sprintf("Image received from clipboard\nSize: %d bytes\nType: %s\n[Image data preserved as base64]",
				result.Size, result.MediaType), nil
		}
	} else {
		// Handle text content
		result.Content = content
		result.Size = len(content)

		if saveToFile {
			// Save text to temporary file
			tempDir := os.TempDir()
			timestamp := time.Now().Format("20060102_150405")
			fileName := fmt.Sprintf("clipboard_text_%s.txt", timestamp)
			filePath := filepath.Join(tempDir, "rcode", fileName)

			// Create directory if it doesn't exist
			if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
				return "", serr.Wrap(err, "failed to create temp directory")
			}

			// Write text to file
			if err := ioutil.WriteFile(filePath, []byte(content), 0644); err != nil {
				return "", serr.Wrap(err, "failed to save text to file")
			}

			result.FilePath = filePath

			// Return the text content with file info
			preview := content
			if len(preview) > 200 {
				preview = preview[:200] + "..."
			}
			return fmt.Sprintf("Text saved to: %s\nSize: %d bytes\n\nContent preview:\n%s",
				filePath, result.Size, preview), nil
		} else {
			// Return the text content directly
			return fmt.Sprintf("Text from clipboard (%d bytes):\n%s", result.Size, content), nil
		}
	}
}

// detectImageType attempts to detect image type from binary data
func detectImageType(data []byte) string {
	// Check common image signatures
	if len(data) < 4 {
		return "application/octet-stream"
	}

	// PNG signature: 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return "image/png"
	}

	// JPEG signature: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return "image/jpeg"
	}

	// GIF signature: 47 49 46 38 (GIF8)
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return "image/gif"
	}

	// WebP signature: 52 49 46 46 ... 57 45 42 50 (RIFF...WEBP)
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return "image/webp"
	}

	// BMP signature: 42 4D (BM)
	if data[0] == 0x42 && data[1] == 0x4D {
		return "image/bmp"
	}

	// SVG (XML-based, check for common SVG patterns)
	if strings.Contains(string(data[:min(1000, len(data))]), "<svg") {
		return "image/svg+xml"
	}

	return "application/octet-stream"
}

// getExtensionFromMimeType returns file extension for a MIME type
func getExtensionFromMimeType(mimeType string) string {
	extensions := map[string]string{
		"image/png":     ".png",
		"image/jpeg":    ".jpg",
		"image/gif":     ".gif",
		"image/webp":    ".webp",
		"image/svg+xml": ".svg",
		"image/bmp":     ".bmp",
		"image/tiff":    ".tiff",
		"image/x-icon":  ".ico",
	}
	if ext, ok := extensions[mimeType]; ok {
		return ext
	}
	return ".bin"
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
