package context

import (
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rohanthewiz/serr"
)

// FilePrioritizer prioritizes files based on relevance to a task
type FilePrioritizer struct {
	// Weights for different scoring factors
	weights struct {
		nameMatch      float64
		pathMatch      float64
		recentlyUsed   float64
		recentlyModified float64
		fileType       float64
		imports        float64
		size           float64
	}
}

// NewFilePrioritizer creates a new file prioritizer with default weights
func NewFilePrioritizer() *FilePrioritizer {
	p := &FilePrioritizer{}
	
	// Default weights (can be tuned based on usage patterns)
	p.weights.nameMatch = 3.0
	p.weights.pathMatch = 2.0
	p.weights.recentlyUsed = 2.5
	p.weights.recentlyModified = 2.0
	p.weights.fileType = 1.5
	p.weights.imports = 1.5
	p.weights.size = -0.5 // Negative weight for size (prefer smaller files)
	
	return p
}

// Prioritize returns a prioritized list of files for a given task
func (p *FilePrioritizer) Prioritize(ctx *ProjectContext, taskCtx *TaskContext) ([]string, error) {
	if ctx == nil || ctx.FileTree == nil {
		return nil, serr.New("invalid project context")
	}

	// Extract keywords from task
	keywords := p.extractKeywords(taskCtx.Task)
	taskCtx.SearchTerms = keywords

	// Score all files
	fileScores := make(map[string]float64)
	p.scoreFileTree(ctx.FileTree, ctx, taskCtx, keywords, fileScores)

	// Sort files by score
	type scoredFile struct {
		path  string
		score float64
	}
	
	scoredFiles := make([]scoredFile, 0, len(fileScores))
	for path, score := range fileScores {
		scoredFiles = append(scoredFiles, scoredFile{path, score})
		taskCtx.FileScores[path] = score
	}

	sort.Slice(scoredFiles, func(i, j int) bool {
		return scoredFiles[i].score > scoredFiles[j].score
	})

	// Return top files
	result := make([]string, 0, taskCtx.MaxFiles)
	for i, sf := range scoredFiles {
		if i >= taskCtx.MaxFiles {
			break
		}
		result = append(result, sf.path)
	}

	return result, nil
}

// scoreFileTree recursively scores files in the tree
func (p *FilePrioritizer) scoreFileTree(node *FileNode, ctx *ProjectContext, taskCtx *TaskContext, keywords []string, scores map[string]float64) {
	if node == nil {
		return
	}

	// Score this file if it's not a directory
	if !node.IsDir {
		score := p.scoreFile(node, ctx, taskCtx, keywords)
		if score > 0 {
			scores[node.Path] = score
		}
	}

	// Recurse into children
	if node.Children != nil {
		for _, child := range node.Children {
			p.scoreFileTree(child, ctx, taskCtx, keywords, scores)
		}
	}
}

// scoreFile calculates the relevance score for a single file
func (p *FilePrioritizer) scoreFile(node *FileNode, ctx *ProjectContext, taskCtx *TaskContext, keywords []string) float64 {
	score := 0.0

	// Skip non-code files unless they're relevant
	if !isRelevantFile(node) {
		return 0
	}

	// Name matching
	nameScore := p.scoreNameMatch(node.Name, keywords)
	score += nameScore * p.weights.nameMatch

	// Path matching
	pathScore := p.scorePathMatch(node.Path, keywords)
	score += pathScore * p.weights.pathMatch

	// Recently used bonus
	if isRecentlyUsed(node.Path, ctx.RecentFiles) {
		score += p.weights.recentlyUsed
	}

	// Recently modified bonus
	if isRecentlyModified(node.Modified) {
		score += p.weights.recentlyModified
	}

	// File type relevance
	typeScore := p.scoreFileType(node, taskCtx.Task)
	score += typeScore * p.weights.fileType

	// Import/dependency relevance
	if len(node.Metadata.Imports) > 0 {
		importScore := p.scoreImports(node.Metadata.Imports, keywords)
		score += importScore * p.weights.imports
	}

	// Size penalty (prefer smaller files)
	if node.Size > 0 {
		sizePenalty := math.Log10(float64(node.Size)) / 10.0
		score += sizePenalty * p.weights.size
	}

	// Boost test files if task mentions testing
	if node.Metadata.IsTest && containsTestKeywords(taskCtx.Task) {
		score *= 2.0
	}

	// Boost config files if task mentions configuration
	if node.Metadata.IsConfig && containsConfigKeywords(taskCtx.Task) {
		score *= 1.5
	}

	return score
}

// scoreNameMatch scores how well a filename matches keywords
func (p *FilePrioritizer) scoreNameMatch(filename string, keywords []string) float64 {
	filename = strings.ToLower(filename)
	score := 0.0

	for _, keyword := range keywords {
		keyword = strings.ToLower(keyword)
		
		// Exact match
		if filename == keyword || filename == keyword+".go" || 
		   filename == keyword+".js" || filename == keyword+".py" {
			score += 3.0
		} else if strings.Contains(filename, keyword) {
			// Partial match
			score += 1.0
			
			// Bonus for match at start
			if strings.HasPrefix(filename, keyword) {
				score += 0.5
			}
		}
	}

	return score
}

// scorePathMatch scores how well a file path matches keywords
func (p *FilePrioritizer) scorePathMatch(path string, keywords []string) float64 {
	path = strings.ToLower(path)
	score := 0.0

	for _, keyword := range keywords {
		keyword = strings.ToLower(keyword)
		
		if strings.Contains(path, keyword) {
			// Count occurrences in path
			count := strings.Count(path, keyword)
			score += float64(count) * 0.5
			
			// Bonus for directory name match
			dir := filepath.Dir(path)
			if strings.Contains(filepath.Base(dir), keyword) {
				score += 1.0
			}
		}
	}

	return score
}

// scoreFileType scores based on file type relevance to task
func (p *FilePrioritizer) scoreFileType(node *FileNode, task string) float64 {
	task = strings.ToLower(task)
	score := 0.0

	// Map task keywords to relevant file types
	if strings.Contains(task, "test") && node.Metadata.IsTest {
		score += 2.0
	}
	
	if strings.Contains(task, "config") && node.Metadata.IsConfig {
		score += 2.0
	}
	
	if strings.Contains(task, "doc") && node.Metadata.IsDocumentation {
		score += 1.5
	}

	// Language-specific boosts
	switch node.Language {
	case "go":
		if strings.Contains(task, "handler") && strings.Contains(node.Name, "handler") {
			score += 1.0
		}
		if strings.Contains(task, "service") && strings.Contains(node.Name, "service") {
			score += 1.0
		}
	case "javascript", "typescript":
		if strings.Contains(task, "component") && strings.Contains(node.Name, "component") {
			score += 1.0
		}
		if strings.Contains(task, "api") && strings.Contains(node.Path, "api") {
			score += 1.0
		}
	}

	return score
}

// scoreImports scores based on import relevance
func (p *FilePrioritizer) scoreImports(imports []string, keywords []string) float64 {
	score := 0.0

	for _, imp := range imports {
		imp = strings.ToLower(imp)
		for _, keyword := range keywords {
			if strings.Contains(imp, keyword) {
				score += 0.5
			}
		}
	}

	return score
}

// extractKeywords extracts relevant keywords from a task description
func (p *FilePrioritizer) extractKeywords(task string) []string {
	// Simple keyword extraction - can be improved with NLP
	task = strings.ToLower(task)
	
	// Remove common words
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "is": true, "are": true,
		"was": true, "were": true, "been": true, "be": true, "have": true,
		"has": true, "had": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "could": true, "should": true,
		"may": true, "might": true, "must": true, "can": true,
		"how": true, "what": true, "where": true, "when": true, "why": true,
		"implement": true, "create": true, "add": true, "update": true,
		"fix": true, "change": true, "modify": true, "edit": true,
	}

	// Split into words
	words := strings.Fields(task)
	keywords := make([]string, 0)
	
	for _, word := range words {
		// Clean word
		word = strings.Trim(word, ".,!?;:'\"")
		
		// Skip stop words and very short words
		if len(word) < 3 || stopWords[word] {
			continue
		}
		
		keywords = append(keywords, word)
	}

	// Add compound keywords for common patterns
	if strings.Contains(task, "api") {
		keywords = append(keywords, "api", "endpoint", "route", "handler")
	}
	if strings.Contains(task, "database") || strings.Contains(task, "db") {
		keywords = append(keywords, "database", "db", "model", "schema")
	}
	if strings.Contains(task, "auth") {
		keywords = append(keywords, "auth", "authentication", "login", "token")
	}
	if strings.Contains(task, "ui") || strings.Contains(task, "frontend") {
		keywords = append(keywords, "ui", "component", "view", "page")
	}

	return keywords
}

// Helper functions

func isRelevantFile(node *FileNode) bool {
	// Skip binary files
	ext := strings.ToLower(filepath.Ext(node.Name))
	binaryExts := map[string]bool{
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".pdf": true, ".zip": true, ".tar": true, ".gz": true,
	}
	
	if binaryExts[ext] {
		return false
	}

	// Include code files, configs, and docs
	return node.Language != "" || node.Metadata.IsConfig || 
	       node.Metadata.IsDocumentation || ext == ".md" || ext == ".txt"
}

func isRecentlyUsed(path string, recentFiles []string) bool {
	for i, recent := range recentFiles {
		if recent == path {
			// More recent = higher in list = more relevant
			return i < 10
		}
	}
	return false
}

func isRecentlyModified(modified time.Time) bool {
	// Consider files modified in the last 7 days as recent
	return time.Since(modified) < 7*24*time.Hour
}

func containsTestKeywords(task string) bool {
	task = strings.ToLower(task)
	testKeywords := []string{"test", "spec", "unit test", "integration test", 
	                        "testing", "tests", "tdd", "bdd"}
	
	for _, keyword := range testKeywords {
		if strings.Contains(task, keyword) {
			return true
		}
	}
	return false
}

func containsConfigKeywords(task string) bool {
	task = strings.ToLower(task)
	configKeywords := []string{"config", "configuration", "settings", "setup",
	                          "environment", "env", "options", "preferences"}
	
	for _, keyword := range configKeywords {
		if strings.Contains(task, keyword) {
			return true
		}
	}
	return false
}

// SetWeights allows customizing the scoring weights
func (p *FilePrioritizer) SetWeights(nameMatch, pathMatch, recentlyUsed, 
	recentlyModified, fileType, imports, size float64) {
	p.weights.nameMatch = nameMatch
	p.weights.pathMatch = pathMatch
	p.weights.recentlyUsed = recentlyUsed
	p.weights.recentlyModified = recentlyModified
	p.weights.fileType = fileType
	p.weights.imports = imports
	p.weights.size = size
}