package web

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

// ZipRequest represents a request to zip files
type ZipRequest struct {
	Paths           []string `json:"paths"`           // Files/directories to zip
	OutputName      string   `json:"outputName"`      // Name for the zip file
	ExcludeDotFiles bool     `json:"excludeDotFiles"` // Exclude files starting with .
	UseGitignore    bool     `json:"useGitignore"`    // Respect .gitignore rules
}

// GitignoreParser handles .gitignore pattern matching
type GitignoreParser struct {
	patterns []gitignorePattern
	root     string
}

type gitignorePattern struct {
	pattern  string
	isNegate bool
	isDir    bool
}

// NewGitignoreParser creates a parser from .gitignore file
func NewGitignoreParser(root string) (*GitignoreParser, error) {
	parser := &GitignoreParser{
		patterns: []gitignorePattern{},
		root:     root,
	}

	gitignorePath := filepath.Join(root, ".gitignore")
	file, err := os.Open(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			// No .gitignore file, return empty parser
			return parser, nil
		}
		return nil, serr.Wrap(err, "failed to open .gitignore")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		pattern := gitignorePattern{
			pattern: line,
		}

		// Check for negation
		if strings.HasPrefix(line, "!") {
			pattern.isNegate = true
			pattern.pattern = line[1:]
		}

		// Check if pattern is directory-specific
		if strings.HasSuffix(pattern.pattern, "/") {
			pattern.isDir = true
			pattern.pattern = strings.TrimSuffix(pattern.pattern, "/")
		}

		parser.patterns = append(parser.patterns, pattern)
	}

	return parser, scanner.Err()
}

// ShouldIgnore checks if a path should be ignored based on gitignore rules
func (g *GitignoreParser) ShouldIgnore(path string) bool {
	// Convert to relative path from root
	relPath, err := filepath.Rel(g.root, path)
	if err != nil {
		return false
	}

	// Normalize path separators
	relPath = filepath.ToSlash(relPath)

	ignored := false
	for _, pattern := range g.patterns {
		if g.matchPattern(relPath, pattern) {
			if pattern.isNegate {
				ignored = false
			} else {
				ignored = true
			}
		}
	}

	return ignored
}

// matchPattern checks if a path matches a gitignore pattern
func (g *GitignoreParser) matchPattern(path string, pattern gitignorePattern) bool {
	// Simple pattern matching - can be enhanced with proper glob matching
	patternStr := pattern.pattern

	// Handle directory-only patterns
	if pattern.isDir {
		info, err := os.Stat(filepath.Join(g.root, path))
		if err != nil || !info.IsDir() {
			return false
		}
	}

	// Handle patterns starting with /
	if strings.HasPrefix(patternStr, "/") {
		patternStr = patternStr[1:]
		// Must match from root
		return strings.HasPrefix(path, patternStr)
	}

	// Handle wildcards (simplified)
	if strings.Contains(patternStr, "*") {
		// Convert simple wildcards to work with filepath.Match
		patternStr = strings.ReplaceAll(patternStr, "**", "*")
		matched, _ := filepath.Match(patternStr, path)
		if matched {
			return true
		}
		// Also check if any parent directory matches
		parts := strings.Split(path, "/")
		for i := range parts {
			subPath := strings.Join(parts[i:], "/")
			if matched, _ := filepath.Match(patternStr, subPath); matched {
				return true
			}
		}
		return false
	}

	// Check if path contains the pattern
	return strings.Contains(path, patternStr)
}

// ZipFilesHandler handles requests to zip files
func ZipFilesHandler(c rweb.Context) error {
	var req ZipRequest
	if err := json.Unmarshal(c.Request().Body(), &req); err != nil {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "Invalid request"})
	}

	// Validate paths
	if len(req.Paths) == 0 {
		c.Response().SetStatus(400)
		return c.WriteJSON(map[string]string{"error": "No files selected"})
	}

	// Generate output name if not provided
	if req.OutputName == "" {
		req.OutputName = fmt.Sprintf("archive_%d.zip", time.Now().Unix())
	}

	// Ensure .zip extension
	if !strings.HasSuffix(req.OutputName, ".zip") {
		req.OutputName += ".zip"
	}

	// Get project root
	projectRoot, _ := os.Getwd()

	// Create gitignore parser if needed
	var gitignoreParser *GitignoreParser
	if req.UseGitignore {
		parser, err := NewGitignoreParser(projectRoot)
		if err != nil {
			logger.LogErr(err, "Failed to parse .gitignore, continuing without it")
		} else {
			gitignoreParser = parser
		}
	}

	// Create output zip file
	outputPath := filepath.Join(projectRoot, req.OutputName)
	zipFile, err := os.Create(outputPath)
	if err != nil {
		c.Response().SetStatus(500)
		return c.WriteJSON(map[string]string{"error": fmt.Sprintf("Failed to create zip file: %v", err)})
	}
	defer zipFile.Close()

	// Create zip writer
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Statistics
	filesAdded := 0
	filesSkipped := 0
	totalSize := int64(0)

	// Process each selected path
	for _, path := range req.Paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		// Validate path is within project
		if !strings.HasPrefix(absPath, projectRoot) {
			filesSkipped++
			continue
		}

		// Get file info
		info, err := os.Stat(absPath)
		if err != nil {
			filesSkipped++
			continue
		}

		if info.IsDir() {
			// Walk directory
			err = filepath.Walk(absPath, func(filePath string, fileInfo os.FileInfo, err error) error {
				if err != nil {
					return nil // Skip files with errors
				}

				// Skip directories themselves (only process files)
				if fileInfo.IsDir() {
					return nil
				}

				// Check exclusion rules
				if shouldExcludeFile(filePath, req.ExcludeDotFiles, gitignoreParser, projectRoot) {
					filesSkipped++
					return nil
				}

				// Add file to zip
				if err := addFileToZip(zipWriter, filePath, projectRoot); err != nil {
					logger.LogErr(err, "Failed to add file to zip", "file", filePath)
					filesSkipped++
				} else {
					filesAdded++
					totalSize += fileInfo.Size()
				}

				return nil
			})
			if err != nil {
				logger.LogErr(err, "Failed to walk directory", "path", absPath)
			}
		} else {
			// Single file
			if shouldExcludeFile(absPath, req.ExcludeDotFiles, gitignoreParser, projectRoot) {
				filesSkipped++
				continue
			}

			if err := addFileToZip(zipWriter, absPath, projectRoot); err != nil {
				logger.LogErr(err, "Failed to add file to zip", "file", absPath)
				filesSkipped++
			} else {
				filesAdded++
				totalSize += info.Size()
			}
		}
	}

	// Close the zip writer to finalize the archive
	if err := zipWriter.Close(); err != nil {
		c.Response().SetStatus(500)
		return c.WriteJSON(map[string]string{"error": fmt.Sprintf("Failed to finalize zip: %v", err)})
	}

	// Get final zip file size
	zipInfo, _ := os.Stat(outputPath)
	zipSize := int64(0)
	if zipInfo != nil {
		zipSize = zipInfo.Size()
	}

	return c.WriteJSON(map[string]interface{}{
		"message":      fmt.Sprintf("Created %s with %d files", req.OutputName, filesAdded),
		"outputPath":   outputPath,
		"filesAdded":   filesAdded,
		"filesSkipped": filesSkipped,
		"originalSize": totalSize,
		"zipSize":      zipSize,
		"compression":  calculateCompressionRatio(totalSize, zipSize),
	})
}

// shouldExcludeFile checks if a file should be excluded from the zip
func shouldExcludeFile(path string, excludeDotFiles bool, gitignoreParser *GitignoreParser, projectRoot string) bool {
	fileName := filepath.Base(path)

	// Never include the zip file itself or git directory
	if strings.HasSuffix(path, ".zip") || strings.Contains(path, ".git") {
		return true
	}

	// Check dot files
	if excludeDotFiles && strings.HasPrefix(fileName, ".") {
		return true
	}

	// Check gitignore
	if gitignoreParser != nil && gitignoreParser.ShouldIgnore(path) {
		return true
	}

	return false
}

// addFileToZip adds a single file to the zip archive
func addFileToZip(zipWriter *zip.Writer, filePath string, baseDir string) error {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return serr.Wrap(err, "failed to open file")
	}
	defer file.Close()

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return serr.Wrap(err, "failed to stat file")
	}

	// Create zip header
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return serr.Wrap(err, "failed to create zip header")
	}

	// Set the name as relative path from base directory
	relPath, err := filepath.Rel(baseDir, filePath)
	if err != nil {
		return serr.Wrap(err, "failed to get relative path")
	}
	header.Name = filepath.ToSlash(relPath) // Use forward slashes in zip

	// Set compression method
	header.Method = zip.Deflate

	// Create writer for this file
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return serr.Wrap(err, "failed to create zip entry")
	}

	// Copy file content
	_, err = io.Copy(writer, file)
	if err != nil {
		return serr.Wrap(err, "failed to write file to zip")
	}

	return nil
}

// calculateCompressionRatio calculates the compression percentage
func calculateCompressionRatio(originalSize, compressedSize int64) string {
	if originalSize == 0 {
		return "0%"
	}
	ratio := float64(originalSize-compressedSize) / float64(originalSize) * 100
	return fmt.Sprintf("%.1f%%", ratio)
}
