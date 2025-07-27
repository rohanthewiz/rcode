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
	// Token estimation parameters
	avgTokensPerLine   float64
	avgTokensPerChar   float64
	avgCharsPerToken   float64
	maxLinesPerFile    int
	priorityDecayRate  float64
	
	// Language-specific token ratios
	langTokenRatios map[string]float64
}

// NewWindowOptimizer creates a new window optimizer
func NewWindowOptimizer() *WindowOptimizer {
	return &WindowOptimizer{
		avgTokensPerLine:  10.0,  // Average for code
		avgTokensPerChar:  0.27,  // ~3.7 chars per token (common for GPT-style tokenizers)
		avgCharsPerToken:  3.7,   // Inverse for calculations
		maxLinesPerFile:   500,   // Truncate very large files
		priorityDecayRate: 0.9,   // Each lower priority file gets 90% of previous score
		
		// Language-specific adjustments based on typical tokenization
		langTokenRatios: map[string]float64{
			"go":         1.1,  // Go tends to have more tokens due to explicit syntax
			"python":     0.9,  // Python is more concise
			"javascript": 1.0,  // Baseline
			"typescript": 1.05, // Slightly more due to type annotations
			"java":       1.2,  // Verbose language
			"rust":       1.15, // More tokens due to ownership syntax
			"cpp":        1.15, // Templates and pointers add tokens
			"markdown":   0.8,  // Natural language is more efficient
			"json":       1.3,  // Lots of delimiters
			"yaml":       0.9,  // More concise than JSON
		},
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
	// Detect language from content or file extension
	lang := wo.detectLanguageFromContent(content)
	
	// Get language-specific ratio
	ratio := wo.langTokenRatios[lang]
	if ratio == 0 {
		ratio = 1.0 // Default ratio
	}
	
	// Count different token types
	tokens := wo.countTokenTypes(content)
	
	// Apply language-specific adjustment
	totalTokens := int(float64(tokens.Total) * ratio)
	
	return totalTokens
}

// TokenCount represents different types of tokens in content
type TokenCount struct {
	Words       int
	Numbers     int
	Symbols     int
	Whitespace  int
	Total       int
}

// countTokenTypes counts different types of tokens using GPT-style tokenization rules
func (wo *WindowOptimizer) countTokenTypes(content string) TokenCount {
	count := TokenCount{}
	
	// Split content into potential tokens
	// This approximates GPT tokenization patterns
	
	// Count whitespace runs as single tokens
	inWhitespace := false
	
	// Track current word/number
	currentWord := ""
	currentNumber := ""
	
	for i, ch := range content {
		switch {
		case isWhitespace(ch):
			// End current word/number
			if currentWord != "" {
				count.Words += wo.estimateWordTokens(currentWord)
				currentWord = ""
			}
			if currentNumber != "" {
				count.Numbers++
				currentNumber = ""
			}
			
			// Count whitespace runs
			if !inWhitespace {
				count.Whitespace++
				inWhitespace = true
			}
			
		case isLetter(ch):
			inWhitespace = false
			if currentNumber != "" {
				count.Numbers++
				currentNumber = ""
			}
			currentWord += string(ch)
			
		case isDigit(ch):
			inWhitespace = false
			if currentWord != "" {
				count.Words += wo.estimateWordTokens(currentWord)
				currentWord = ""
			}
			currentNumber += string(ch)
			
		default:
			// Symbols and punctuation
			inWhitespace = false
			
			// End current word/number
			if currentWord != "" {
				count.Words += wo.estimateWordTokens(currentWord)
				currentWord = ""
			}
			if currentNumber != "" {
				count.Numbers++
				currentNumber = ""
			}
			
			// Special handling for common multi-char operators
			if i+1 < len(content) {
				nextCh := rune(content[i+1])
				twoChar := string(ch) + string(nextCh)
				if isMultiCharOperator(twoChar) {
					count.Symbols++
					continue
				}
			}
			
			count.Symbols++
		}
	}
	
	// Handle any remaining
	if currentWord != "" {
		count.Words += wo.estimateWordTokens(currentWord)
	}
	if currentNumber != "" {
		count.Numbers++
	}
	
	count.Total = count.Words + count.Numbers + count.Symbols + count.Whitespace
	return count
}

// estimateWordTokens estimates tokens for a word (handling subword tokenization)
func (wo *WindowOptimizer) estimateWordTokens(word string) int {
	// Common words are usually single tokens
	if isCommonWord(word) {
		return 1
	}
	
	// Short words are typically single tokens
	if len(word) <= 4 {
		return 1
	}
	
	// Estimate based on character count for longer words
	// GPT tokenizers typically split long words
	if len(word) <= 8 {
		return 1
	} else if len(word) <= 12 {
		return 2
	} else {
		// Very long words get split multiple times
		return (len(word) + 5) / 6
	}
}

// detectLanguageFromContent attempts to detect programming language from content
func (wo *WindowOptimizer) detectLanguageFromContent(content string) string {
	// Simple heuristics based on syntax
	lines := strings.Split(content, "\n")
	
	for _, line := range lines[:min(50, len(lines))] {
		trimmed := strings.TrimSpace(line)
		
		// Go detection
		if strings.HasPrefix(trimmed, "package ") || strings.HasPrefix(trimmed, "func ") ||
		   strings.Contains(trimmed, " := ") {
			return "go"
		}
		
		// Python detection
		if strings.HasPrefix(trimmed, "def ") || strings.HasPrefix(trimmed, "class ") ||
		   strings.HasPrefix(trimmed, "import ") || strings.HasPrefix(trimmed, "from ") {
			return "python"
		}
		
		// JavaScript/TypeScript detection
		if strings.Contains(trimmed, "const ") || strings.Contains(trimmed, "let ") ||
		   strings.Contains(trimmed, "function ") || strings.Contains(trimmed, "=>") {
			if strings.Contains(content, ": string") || strings.Contains(content, ": number") {
				return "typescript"
			}
			return "javascript"
		}
		
		// Java detection
		if strings.Contains(trimmed, "public class") || strings.Contains(trimmed, "private ") ||
		   strings.Contains(trimmed, "public static void") {
			return "java"
		}
		
		// Rust detection
		if strings.HasPrefix(trimmed, "fn ") || strings.HasPrefix(trimmed, "impl ") ||
		   strings.Contains(trimmed, "mut ") || strings.HasPrefix(trimmed, "use ") {
			return "rust"
		}
	}
	
	// Check for common file patterns
	if strings.Contains(content, "# ") && strings.Contains(content, "\n## ") {
		return "markdown"
	}
	
	if strings.Contains(content, "{") && strings.Contains(content, "}") &&
	   strings.Contains(content, "\"") && strings.Contains(content, ":") {
		if strings.Contains(content, "[\n") || strings.Contains(content, "{\n") {
			return "json"
		}
	}
	
	return "unknown"
}

// Helper functions for token counting

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isLetter(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

func isMultiCharOperator(s string) bool {
	operators := map[string]bool{
		"==": true, "!=": true, "<=": true, ">=": true,
		"&&": true, "||": true, "<<": true, ">>": true,
		"++": true, "--": true, "+=": true, "-=": true,
		"*=": true, "/=": true, ":=": true, "->": true,
		"=>": true, "//": true, "/*": true, "*/": true,
	}
	return operators[s]
}

func isCommonWord(word string) bool {
	word = strings.ToLower(word)
	commonWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"if": true, "else": true, "for": true, "while": true, "do": true,
		"return": true, "function": true, "var": true, "let": true, "const": true,
		"class": true, "def": true, "import": true, "from": true, "package": true,
		"public": true, "private": true, "static": true, "void": true,
		"int": true, "string": true, "bool": true, "true": true, "false": true,
		"null": true, "nil": true, "none": true, "undefined": true,
	}
	return commonWords[word]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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

// GetDetailedTokenCount returns detailed token count information for content
func (wo *WindowOptimizer) GetDetailedTokenCount(content string) TokenCountDetails {
	lang := wo.detectLanguageFromContent(content)
	tokens := wo.countTokenTypes(content)
	ratio := wo.langTokenRatios[lang]
	if ratio == 0 {
		ratio = 1.0
	}
	
	return TokenCountDetails{
		Language:       lang,
		TokenCount:     tokens,
		AdjustedTotal:  int(float64(tokens.Total) * ratio),
		LanguageRatio:  ratio,
		Lines:          strings.Count(content, "\n") + 1,
		Characters:     len(content),
		AvgTokensPerLine: float64(tokens.Total) / float64(strings.Count(content, "\n") + 1),
	}
}

// TokenCountDetails provides detailed token counting information
type TokenCountDetails struct {
	Language         string
	TokenCount       TokenCount
	AdjustedTotal    int
	LanguageRatio    float64
	Lines            int
	Characters       int
	AvgTokensPerLine float64
}

// EstimateTokensForFile estimates tokens for a file with language detection
func (wo *WindowOptimizer) EstimateTokensForFile(path string) (int, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return 0, serr.Wrap(err, "failed to read file for token estimation")
	}
	
	// Override language detection based on file extension if available
	ext := strings.ToLower(filepath.Ext(path))
	lang := ""
	
	switch ext {
	case ".go":
		lang = "go"
	case ".py":
		lang = "python"
	case ".js", ".jsx":
		lang = "javascript"
	case ".ts", ".tsx":
		lang = "typescript"
	case ".java":
		lang = "java"
	case ".rs":
		lang = "rust"
	case ".cpp", ".cc", ".cxx", ".hpp", ".h":
		lang = "cpp"
	case ".md":
		lang = "markdown"
	case ".json":
		lang = "json"
	case ".yaml", ".yml":
		lang = "yaml"
	default:
		// Use content-based detection
		lang = wo.detectLanguageFromContent(string(content))
	}
	
	// Get language-specific ratio
	ratio := wo.langTokenRatios[lang]
	if ratio == 0 {
		ratio = 1.0
	}
	
	// Count tokens
	tokens := wo.countTokenTypes(string(content))
	
	return int(float64(tokens.Total) * ratio), nil
}