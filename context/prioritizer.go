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
	
	// Function/method relevance
	if len(node.Metadata.Functions) > 0 {
		funcScore := p.scoreFunctions(node.Metadata.Functions, keywords)
		score += funcScore * p.weights.nameMatch // Use name match weight
	}
	
	// Class/type relevance
	if len(node.Metadata.Classes) > 0 {
		classScore := p.scoreClasses(node.Metadata.Classes, keywords)
		score += classScore * p.weights.nameMatch // Use name match weight
	}
	
	// Export relevance (public API)
	if len(node.Metadata.Exports) > 0 {
		exportScore := p.scoreExports(node.Metadata.Exports, keywords)
		score += exportScore * 1.2 // Slight boost for public API
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

// scoreFunctions scores based on function name relevance
func (p *FilePrioritizer) scoreFunctions(functions []string, keywords []string) float64 {
	score := 0.0
	
	for _, function := range functions {
		funcLower := strings.ToLower(function)
		for _, keyword := range keywords {
			keywordLower := strings.ToLower(keyword)
			
			// Exact match
			if funcLower == keywordLower {
				score += 2.0
			} else if strings.Contains(funcLower, keywordLower) {
				score += 1.0
			}
			
			// Check camelCase splits
			splits := splitCamelCase(function)
			for _, split := range splits {
				if split == keywordLower {
					score += 0.5
				}
			}
		}
	}
	
	return score
}

// scoreClasses scores based on class/type name relevance
func (p *FilePrioritizer) scoreClasses(classes []string, keywords []string) float64 {
	score := 0.0
	
	for _, class := range classes {
		classLower := strings.ToLower(class)
		for _, keyword := range keywords {
			keywordLower := strings.ToLower(keyword)
			
			// Exact match
			if classLower == keywordLower {
				score += 2.0
			} else if strings.Contains(classLower, keywordLower) {
				score += 1.0
			}
			
			// Check camelCase splits
			splits := splitCamelCase(class)
			for _, split := range splits {
				if split == keywordLower {
					score += 0.5
				}
			}
		}
	}
	
	return score
}

// scoreExports scores based on exported symbols relevance
func (p *FilePrioritizer) scoreExports(exports []string, keywords []string) float64 {
	score := 0.0
	
	for _, export := range exports {
		exportLower := strings.ToLower(export)
		for _, keyword := range keywords {
			keywordLower := strings.ToLower(keyword)
			
			// Exact match for exports gets higher score
			if exportLower == keywordLower {
				score += 2.5
			} else if strings.Contains(exportLower, keywordLower) {
				score += 1.5
			}
		}
	}
	
	return score
}

// extractKeywords extracts relevant keywords from a task description
func (p *FilePrioritizer) extractKeywords(task string) []string {
	// Enhanced NLP-based keyword extraction
	originalTask := task
	task = strings.ToLower(task)
	
	// Extended stop words list
	stopWords := map[string]bool{
		// Articles
		"the": true, "a": true, "an": true,
		// Conjunctions
		"and": true, "or": true, "but": true, "nor": true, "yet": true, "so": true,
		// Prepositions
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true, 
		"with": true, "by": true, "from": true, "up": true, "about": true, "into": true,
		"through": true, "during": true, "before": true, "after": true, "above": true,
		"below": true, "between": true, "under": true, "over": true,
		// Pronouns
		"i": true, "me": true, "my": true, "you": true, "your": true, "he": true,
		"she": true, "it": true, "its": true, "we": true, "our": true, "they": true,
		"their": true, "this": true, "that": true, "these": true, "those": true,
		// Verbs (common)
		"is": true, "are": true, "was": true, "were": true, "been": true, "be": true,
		"have": true, "has": true, "had": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "could": true, "should": true, "may": true, 
		"might": true, "must": true, "can": true, "need": true, "want": true,
		// Question words (but we'll extract them specially)
		"how": true, "what": true, "where": true, "when": true, "why": true, "which": true,
		// Common task words (we'll handle these specially)
		"please": true, "help": true, "me": true, "find": true, "show": true,
	}

	// Extract code-like patterns first (camelCase, snake_case, etc.)
	codePatterns := p.extractCodePatterns(originalTask)
	
	// Split into words and clean
	words := strings.FieldsFunc(task, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-')
	})
	
	keywordMap := make(map[string]bool)
	keywords := make([]string, 0)
	
	// Process each word
	for _, word := range words {
		// Skip if too short or stop word
		if len(word) < 2 || stopWords[word] {
			continue
		}
		
		// Skip numbers
		if isNumeric(word) {
			continue
		}
		
		// Add to keywords if not already present
		if !keywordMap[word] {
			keywordMap[word] = true
			keywords = append(keywords, word)
		}
	}
	
	// Add code patterns
	for _, pattern := range codePatterns {
		if !keywordMap[strings.ToLower(pattern)] {
			keywords = append(keywords, pattern)
		}
	}
	
	// Extract and expand domain-specific terms
	domainKeywords := p.extractDomainKeywords(task)
	for _, dk := range domainKeywords {
		if !keywordMap[dk] {
			keywords = append(keywords, dk)
		}
	}
	
	// Extract action-object pairs
	actionPairs := p.extractActionObjectPairs(task)
	for _, pair := range actionPairs {
		if !keywordMap[pair] {
			keywords = append(keywords, pair)
		}
	}
	
	// Add synonyms and related terms
	expandedKeywords := p.expandKeywords(keywords)
	for _, ek := range expandedKeywords {
		if !keywordMap[ek] {
			keywords = append(keywords, ek)
		}
	}
	
	return keywords
}

// extractCodePatterns extracts code-like patterns from text
func (p *FilePrioritizer) extractCodePatterns(text string) []string {
	patterns := make([]string, 0)
	
	// Regular expression patterns for code elements
	// CamelCase: UserController, getData
	camelCaseWords := strings.FieldsFunc(text, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'))
	})
	
	for _, word := range camelCaseWords {
		if len(word) > 1 && containsUpperAndLower(word) {
			patterns = append(patterns, word)
			// Also add split version
			splits := splitCamelCase(word)
			patterns = append(patterns, splits...)
		}
	}
	
	// Snake_case and kebab-case
	snakeKebabWords := strings.FieldsFunc(text, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || 
		        (r >= '0' && r <= '9') || r == '_' || r == '-')
	})
	
	for _, word := range snakeKebabWords {
		if strings.Contains(word, "_") || strings.Contains(word, "-") {
			patterns = append(patterns, word)
			// Also add parts
			parts := strings.FieldsFunc(word, func(r rune) bool {
				return r == '_' || r == '-'
			})
			patterns = append(patterns, parts...)
		}
	}
	
	// File extensions
	words := strings.Fields(text)
	for _, word := range words {
		if strings.Contains(word, ".") && len(word) > 2 {
			ext := filepath.Ext(word)
			if len(ext) > 1 && len(ext) < 6 {
				patterns = append(patterns, strings.TrimPrefix(ext, "."))
			}
		}
	}
	
	return patterns
}

// extractDomainKeywords extracts domain-specific keywords based on context
func (p *FilePrioritizer) extractDomainKeywords(task string) []string {
	keywords := make([]string, 0)
	
	// Development domain mappings
	domainMappings := map[string][]string{
		"api":          {"endpoint", "route", "handler", "rest", "graphql", "controller"},
		"database":     {"db", "model", "schema", "migration", "query", "table", "sql"},
		"auth":         {"authentication", "authorization", "login", "token", "jwt", "oauth", "session"},
		"ui":           {"component", "view", "page", "template", "style", "css", "layout"},
		"frontend":     {"react", "vue", "angular", "component", "state", "props", "dom"},
		"backend":      {"server", "service", "middleware", "controller", "model"},
		"test":         {"spec", "unit", "integration", "mock", "assert", "expect", "coverage"},
		"performance":  {"optimize", "cache", "speed", "latency", "memory", "cpu"},
		"security":     {"vulnerability", "encryption", "ssl", "https", "cors", "xss", "csrf"},
		"deployment":   {"docker", "kubernetes", "ci", "cd", "pipeline", "build", "release"},
		"logging":      {"log", "logger", "debug", "error", "trace", "monitoring"},
		"config":       {"configuration", "settings", "environment", "env", "options", "yaml", "json"},
		"validation":   {"validate", "validator", "check", "verify", "sanitize", "rules"},
		"error":        {"exception", "handling", "catch", "throw", "stack", "trace"},
		"async":        {"promise", "async", "await", "callback", "concurrent", "parallel"},
		"cache":        {"redis", "memcached", "storage", "ttl", "invalidate"},
		"search":       {"elasticsearch", "solr", "index", "query", "filter", "facet"},
		"message":      {"queue", "pubsub", "kafka", "rabbitmq", "event", "broker"},
		"payment":      {"stripe", "paypal", "checkout", "billing", "subscription", "invoice"},
	}
	
	// Check each domain
	for domain, terms := range domainMappings {
		if strings.Contains(task, domain) {
			keywords = append(keywords, terms...)
		}
	}
	
	// Programming language specific
	if strings.Contains(task, "go") || strings.Contains(task, "golang") {
		keywords = append(keywords, "goroutine", "channel", "interface", "struct", "package")
	}
	if strings.Contains(task, "javascript") || strings.Contains(task, "js") {
		keywords = append(keywords, "function", "class", "module", "npm", "node")
	}
	if strings.Contains(task, "python") || strings.Contains(task, "py") {
		keywords = append(keywords, "def", "class", "module", "pip", "django", "flask")
	}
	
	return keywords
}

// extractActionObjectPairs extracts action-object pairs from task
func (p *FilePrioritizer) extractActionObjectPairs(task string) []string {
	pairs := make([]string, 0)
	
	// Common action verbs in development tasks
	actionVerbs := map[string]bool{
		"create": true, "add": true, "implement": true, "build": true,
		"update": true, "modify": true, "change": true, "edit": true,
		"fix": true, "repair": true, "debug": true, "resolve": true,
		"remove": true, "delete": true, "clean": true, "refactor": true,
		"optimize": true, "improve": true, "enhance": true,
		"test": true, "validate": true, "check": true, "verify": true,
		"integrate": true, "connect": true, "link": true,
		"migrate": true, "upgrade": true, "deploy": true,
		"configure": true, "setup": true, "install": true,
	}
	
	words := strings.Fields(strings.ToLower(task))
	for i, word := range words {
		if actionVerbs[word] && i+1 < len(words) {
			// Get the next word as potential object
			obj := words[i+1]
			if len(obj) > 2 && !isStopWord(obj) {
				pairs = append(pairs, obj)
				
				// Also check for compound objects
				if i+2 < len(words) && !isStopWord(words[i+2]) {
					compound := obj + "_" + words[i+2]
					pairs = append(pairs, compound)
				}
			}
		}
	}
	
	return pairs
}

// expandKeywords adds synonyms and related terms
func (p *FilePrioritizer) expandKeywords(keywords []string) []string {
	expanded := make([]string, 0)
	
	// Common synonyms and related terms in software development
	synonyms := map[string][]string{
		"api":        {"endpoint", "service"},
		"function":   {"func", "method", "procedure"},
		"class":      {"type", "struct", "object"},
		"test":       {"spec", "testing"},
		"config":     {"configuration", "settings"},
		"auth":       {"authentication", "authorization"},
		"db":         {"database", "storage"},
		"error":      {"exception", "err"},
		"handler":    {"controller", "processor"},
		"route":      {"path", "endpoint"},
		"model":      {"schema", "entity"},
		"component":  {"widget", "element"},
		"service":    {"provider", "manager"},
		"util":       {"utility", "helper"},
		"lib":        {"library", "package"},
	}
	
	for _, keyword := range keywords {
		if syns, exists := synonyms[keyword]; exists {
			expanded = append(expanded, syns...)
		}
		
		// Also check reverse mapping
		for key, syns := range synonyms {
			for _, syn := range syns {
				if syn == keyword {
					expanded = append(expanded, key)
					break
				}
			}
		}
	}
	
	return expanded
}

// Helper functions for keyword extraction

func containsUpperAndLower(s string) bool {
	hasUpper := false
	hasLower := false
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			hasUpper = true
		}
		if r >= 'a' && r <= 'z' {
			hasLower = true
		}
		if hasUpper && hasLower {
			return true
		}
	}
	return false
}

func splitCamelCase(s string) []string {
	var parts []string
	var current []rune
	
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			if len(current) > 0 {
				parts = append(parts, strings.ToLower(string(current)))
			}
			current = []rune{r}
		} else {
			current = append(current, r)
		}
	}
	
	if len(current) > 0 {
		parts = append(parts, strings.ToLower(string(current)))
	}
	
	return parts
}

func isNumeric(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "is": true, "are": true,
	}
	return stopWords[word]
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