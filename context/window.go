package context

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rohanthewiz/serr"
)

// WindowOptimizer manages context windows for AI interactions
type WindowOptimizer struct {
	// Token estimation rates (rough estimates)
	avgTokensPerLine   float64
	avgTokensPerChar   float64
	maxLinesPerFile    int
	priorityDecayRate  float64
}

// NewWindowOptimizer creates a new window optimizer
func NewWindowOptimizer() *WindowOptimizer {
	return &WindowOptimizer{
		avgTokensPerLine:  10.0,  // Rough estimate
		avgTokensPerChar:  0.25,  // ~4 chars per token
		maxLinesPerFile:   500,   // Truncate very large files
		priorityDecayRate: 0.9,   // Each lower priority file gets 90% of previous score
	}
}

// OptimizeWindow creates an optimized context window within token limits
func (wo *WindowOptimizer) OptimizeWindow(files []string, scores map[string]float64, maxTokens int) (*ContextWindow, error) {
	window := &ContextWindow{
		Files:     make([]ContextFile, 0),
		MaxTokens: maxTokens,
		Priority:  "optimized",
	}

	// Create file info list with scores
	type fileInfo struct {
		path   string
		score  float64
		size   int64
		tokens int
	}

	fileInfos := make([]fileInfo, 0, len(files))
	
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			continue // Skip files we can't read
		}

		// Skip directories and binary files
		if info.IsDir() || isBinaryFile(path) {
			continue
		}

		score := scores[path]
		if score == 0 {
			score = 1.0 // Default score
		}

		tokens := wo.estimateTokens(info.Size())
		
		fileInfos = append(fileInfos, fileInfo{
			path:   path,
			score:  score,
			size:   info.Size(),
			tokens: tokens,
		})
	}

	// Sort by score (highest first)
	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].score > fileInfos[j].score
	})

	// Pack files into window using greedy algorithm
	usedTokens := 0
	reservedTokens := maxTokens / 10 // Reserve 10% for system messages

	for i, fi := range fileInfos {
		if usedTokens+fi.tokens > maxTokens-reservedTokens {
			// Try to fit partial file
			remainingTokens := maxTokens - reservedTokens - usedTokens
			if remainingTokens > 100 { // Only include if we can fit meaningful content
				content, tokens := wo.truncateFile(fi.path, remainingTokens)
				if tokens > 0 {
					window.Files = append(window.Files, ContextFile{
						Path:     fi.path,
						Content:  content,
						Tokens:   tokens,
						Score:    fi.score * wo.priorityDecayRate * float64(i),
						Included: true,
					})
					usedTokens += tokens
				}
			}
			break
		}

		// Read full file
		content, tokens, err := wo.readFileWithTokenLimit(fi.path, fi.tokens)
		if err != nil {
			continue
		}

		window.Files = append(window.Files, ContextFile{
			Path:     fi.path,
			Content:  content,
			Tokens:   tokens,
			Score:    fi.score * wo.priorityDecayRate * float64(i),
			Included: true,
		})
		usedTokens += tokens
	}

	window.TotalTokens = usedTokens

	// Add excluded files for reference
	for _, fi := range fileInfos[len(window.Files):] {
		window.Files = append(window.Files, ContextFile{
			Path:     fi.path,
			Content:  "", // Not included
			Tokens:   fi.tokens,
			Score:    fi.score,
			Included: false,
		})
	}

	return window, nil
}

// estimateTokens estimates token count from file size
func (wo *WindowOptimizer) estimateTokens(size int64) int {
	// Use character-based estimation
	return int(float64(size) * wo.avgTokensPerChar)
}

// readFileWithTokenLimit reads a file up to a token limit
func (wo *WindowOptimizer) readFileWithTokenLimit(path string, maxTokens int) (string, int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", 0, serr.Wrap(err, "failed to read file")
	}

	// Convert to string and split into lines
	fullContent := string(content)
	lines := strings.Split(fullContent, "\n")

	// Limit lines if file is too large
	if len(lines) > wo.maxLinesPerFile {
		lines = lines[:wo.maxLinesPerFile]
		fullContent = strings.Join(lines, "\n") + "\n... (truncated)"
	}

	// Estimate tokens
	estimatedTokens := wo.estimateTokensFromContent(fullContent)
	
	// If within limit, return full content
	if estimatedTokens <= maxTokens {
		return fullContent, estimatedTokens, nil
	}

	// Otherwise, truncate to fit
	return wo.truncateContent(fullContent, maxTokens), maxTokens, nil
}

// truncateFile reads and truncates a file to fit within token limit
func (wo *WindowOptimizer) truncateFile(path string, maxTokens int) (string, int) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", 0
	}

	fullContent := string(content)
	return wo.truncateContent(fullContent, maxTokens), maxTokens
}

// truncateContent truncates content to fit within token limit
func (wo *WindowOptimizer) truncateContent(content string, maxTokens int) string {
	lines := strings.Split(content, "\n")
	
	// Binary search for the right number of lines
	left, right := 0, len(lines)
	result := ""
	
	for left < right {
		mid := (left + right + 1) / 2
		truncated := strings.Join(lines[:mid], "\n")
		tokens := wo.estimateTokensFromContent(truncated)
		
		if tokens <= maxTokens {
			result = truncated
			left = mid
		} else {
			right = mid - 1
		}
	}

	if result != "" && len(result) < len(content) {
		result += "\n... (truncated)"
	}

	return result
}

// estimateTokensFromContent estimates tokens from actual content
func (wo *WindowOptimizer) estimateTokensFromContent(content string) int {
	// Use line-based estimation for better accuracy
	lines := strings.Split(content, "\n")
	
	// Count non-empty lines and characters
	nonEmptyLines := 0
	totalChars := 0
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			nonEmptyLines++
			totalChars += len(trimmed)
		}
	}

	// Use hybrid estimation
	lineBasedEstimate := float64(nonEmptyLines) * wo.avgTokensPerLine
	charBasedEstimate := float64(totalChars) * wo.avgTokensPerChar
	
	// Return average of both estimates
	return int((lineBasedEstimate + charBasedEstimate) / 2)
}

// isBinaryFile checks if a file is likely binary
func isBinaryFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".pdf": true, ".zip": true, ".tar": true, ".gz": true,
		".db": true, ".sqlite": true, ".bin": true, ".dat": true,
		".ico": true, ".icns": true, ".mp3": true, ".mp4": true,
		".wav": true, ".avi": true, ".mov": true, ".mkv": true,
	}
	
	return binaryExts[ext]
}

// GetSummary returns a summary of the context window
func (wo *WindowOptimizer) GetSummary(window *ContextWindow) string {
	includedCount := 0
	excludedCount := 0
	
	for _, file := range window.Files {
		if file.Included {
			includedCount++
		} else {
			excludedCount++
		}
	}

	return fmt.Sprintf(
		"Context Window Summary:\n"+
		"- Total tokens: %d / %d (%.1f%%)\n"+
		"- Files included: %d\n"+
		"- Files excluded: %d\n"+
		"- Priority mode: %s",
		window.TotalTokens,
		window.MaxTokens,
		float64(window.TotalTokens)/float64(window.MaxTokens)*100,
		includedCount,
		excludedCount,
		window.Priority,
	)
}

// RecommendTokenLimit recommends a token limit based on the task
func (wo *WindowOptimizer) RecommendTokenLimit(taskComplexity string) int {
	switch taskComplexity {
	case "simple":
		return 4000
	case "moderate":
		return 8000
	case "complex":
		return 16000
	case "very_complex":
		return 32000
	default:
		return 8000
	}
}

// AnalyzeTokenUsage analyzes token usage patterns
func (wo *WindowOptimizer) AnalyzeTokenUsage(windows []*ContextWindow) TokenUsageStats {
	stats := TokenUsageStats{
		WindowCount: len(windows),
	}

	if len(windows) == 0 {
		return stats
	}

	totalTokens := 0
	totalFiles := 0
	tokenCounts := make([]int, len(windows))

	for i, window := range windows {
		totalTokens += window.TotalTokens
		for _, file := range window.Files {
			if file.Included {
				totalFiles++
			}
		}
		tokenCounts[i] = window.TotalTokens
	}

	stats.AvgTokensPerWindow = totalTokens / len(windows)
	stats.AvgFilesPerWindow = totalFiles / len(windows)
	stats.TotalTokensUsed = totalTokens

	// Calculate median
	sort.Ints(tokenCounts)
	if len(tokenCounts)%2 == 0 {
		stats.MedianTokens = (tokenCounts[len(tokenCounts)/2-1] + tokenCounts[len(tokenCounts)/2]) / 2
	} else {
		stats.MedianTokens = tokenCounts[len(tokenCounts)/2]
	}

	return stats
}

// TokenUsageStats contains statistics about token usage
type TokenUsageStats struct {
	WindowCount        int
	TotalTokensUsed    int
	AvgTokensPerWindow int
	MedianTokens       int
	AvgFilesPerWindow  int
}