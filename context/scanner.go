package context

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rohanthewiz/serr"
)

// ProjectScanner scans projects to detect language, framework, and structure
type ProjectScanner struct {
	ignorePatterns []string
}

// NewProjectScanner creates a new project scanner
func NewProjectScanner() *ProjectScanner {
	return &ProjectScanner{
		ignorePatterns: []string{
			".git", "node_modules", "vendor", ".venv", "venv",
			"__pycache__", ".pytest_cache", "dist", "build",
			"target", ".idea", ".vscode", "*.pyc", "*.pyo",
		},
	}
}

// Scan analyzes a project directory and returns context
func (s *ProjectScanner) Scan(rootPath string) (*ProjectContext, error) {
	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, serr.Wrap(err, "failed to get absolute path")
	}

	ctx := &ProjectContext{
		RootPath:      absPath,
		Dependencies:  make([]Dependency, 0),
		ModifiedFiles: make(map[string]time.Time),
		Statistics: ProjectStats{
			FilesByLanguage: make(map[string]int),
		},
	}

	// Detect language and framework from config files
	if err := s.detectProjectType(ctx); err != nil {
		return nil, serr.Wrap(err, "failed to detect project type")
	}

	// Load ignore patterns from .gitignore
	s.loadGitignore(absPath)

	// Build file tree
	fileTree, err := s.buildFileTree(absPath, absPath)
	if err != nil {
		return nil, serr.Wrap(err, "failed to build file tree")
	}
	ctx.FileTree = fileTree

	// Detect project patterns
	ctx.Patterns = s.detectPatterns(ctx)

	// Calculate statistics
	s.calculateStats(ctx)

	return ctx, nil
}

// detectProjectType detects the primary language and framework
func (s *ProjectScanner) detectProjectType(ctx *ProjectContext) error {
	rootPath := ctx.RootPath

	// Check for Go
	if _, err := os.Stat(filepath.Join(rootPath, "go.mod")); err == nil {
		ctx.Language = "go"
		s.parseGoMod(ctx)
		return nil
	}

	// Check for Node.js/JavaScript/TypeScript
	if _, err := os.Stat(filepath.Join(rootPath, "package.json")); err == nil {
		ctx.Language = "javascript"
		s.parsePackageJSON(ctx)
		return nil
	}

	// Check for Python
	for _, file := range []string{"requirements.txt", "setup.py", "pyproject.toml", "Pipfile"} {
		if _, err := os.Stat(filepath.Join(rootPath, file)); err == nil {
			ctx.Language = "python"
			s.parsePythonDeps(ctx, file)
			return nil
		}
	}

	// Check for Rust
	if _, err := os.Stat(filepath.Join(rootPath, "Cargo.toml")); err == nil {
		ctx.Language = "rust"
		// TODO: Parse Cargo.toml
		return nil
	}

	// Check for Java
	if _, err := os.Stat(filepath.Join(rootPath, "pom.xml")); err == nil {
		ctx.Language = "java"
		ctx.Framework = "maven"
		return nil
	}
	if _, err := os.Stat(filepath.Join(rootPath, "build.gradle")); err == nil {
		ctx.Language = "java"
		ctx.Framework = "gradle"
		return nil
	}

	// Default: try to detect from file extensions
	s.detectFromExtensions(ctx)
	return nil
}

// parseGoMod parses go.mod file for dependencies
func (s *ProjectScanner) parseGoMod(ctx *ProjectContext) {
	modPath := filepath.Join(ctx.RootPath, "go.mod")
	file, err := os.Open(modPath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inRequire := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if strings.HasPrefix(line, "module ") {
			// Module name can help identify framework
			moduleName := strings.TrimPrefix(line, "module ")
			if strings.Contains(moduleName, "gin") {
				ctx.Framework = "gin"
			} else if strings.Contains(moduleName, "echo") {
				ctx.Framework = "echo"
			} else if strings.Contains(moduleName, "fiber") {
				ctx.Framework = "fiber"
			}
		}

		if line == "require (" {
			inRequire = true
			continue
		}
		if inRequire && line == ")" {
			inRequire = false
			continue
		}

		if inRequire && line != "" && !strings.HasPrefix(line, "//") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				dep := Dependency{
					Name:    parts[0],
					Version: parts[1],
					Type:    "go_module",
				}
				ctx.Dependencies = append(ctx.Dependencies, dep)

				// Detect framework from dependencies
				if strings.Contains(dep.Name, "gin-gonic/gin") {
					ctx.Framework = "gin"
				} else if strings.Contains(dep.Name, "labstack/echo") {
					ctx.Framework = "echo"
				} else if strings.Contains(dep.Name, "gofiber/fiber") {
					ctx.Framework = "fiber"
				}
			}
		}
	}
}

// parsePackageJSON parses package.json for dependencies
func (s *ProjectScanner) parsePackageJSON(ctx *ProjectContext) {
	pkgPath := filepath.Join(ctx.RootPath, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}

	// Check for TypeScript
	if _, hasTS := pkg.DevDependencies["typescript"]; hasTS {
		ctx.Language = "typescript"
	}

	// Detect framework
	frameworks := map[string]string{
		"react":     "react",
		"vue":       "vue",
		"@angular/": "angular",
		"express":   "express",
		"next":      "nextjs",
		"nuxt":      "nuxt",
		"svelte":    "svelte",
	}

	for dep := range pkg.Dependencies {
		for key, framework := range frameworks {
			if strings.Contains(dep, key) {
				ctx.Framework = framework
				break
			}
		}
	}

	// Add dependencies
	for name, version := range pkg.Dependencies {
		ctx.Dependencies = append(ctx.Dependencies, Dependency{
			Name:    name,
			Version: version,
			Type:    "npm_package",
		})
	}
}

// parsePythonDeps parses Python dependency files
func (s *ProjectScanner) parsePythonDeps(ctx *ProjectContext, filename string) {
	depPath := filepath.Join(ctx.RootPath, filename)
	
	switch filename {
	case "requirements.txt":
		s.parseRequirementsTxt(ctx, depPath)
	case "pyproject.toml":
		// TODO: Parse pyproject.toml
	case "Pipfile":
		// TODO: Parse Pipfile
	}

	// Detect common Python frameworks
	for _, dep := range ctx.Dependencies {
		switch {
		case strings.Contains(dep.Name, "django"):
			ctx.Framework = "django"
		case strings.Contains(dep.Name, "flask"):
			ctx.Framework = "flask"
		case strings.Contains(dep.Name, "fastapi"):
			ctx.Framework = "fastapi"
		case strings.Contains(dep.Name, "pytest"):
			// Testing framework, not main framework
		}
	}
}

// parseRequirementsTxt parses requirements.txt
func (s *ProjectScanner) parseRequirementsTxt(ctx *ProjectContext, path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Simple parsing - doesn't handle all edge cases
		parts := strings.Split(line, "==")
		name := parts[0]
		version := ""
		if len(parts) > 1 {
			version = parts[1]
		}

		ctx.Dependencies = append(ctx.Dependencies, Dependency{
			Name:    name,
			Version: version,
			Type:    "pip_package",
		})
	}
}

// detectFromExtensions detects language from file extensions
func (s *ProjectScanner) detectFromExtensions(ctx *ProjectContext) {
	extCounts := make(map[string]int)

	filepath.Walk(ctx.RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != "" {
			extCounts[ext]++
		}
		return nil
	})

	// Language detection based on most common extension
	langMap := map[string]string{
		".go":   "go",
		".js":   "javascript",
		".ts":   "typescript",
		".jsx":  "javascript",
		".tsx":  "typescript",
		".py":   "python",
		".java": "java",
		".rs":   "rust",
		".cpp":  "cpp",
		".c":    "c",
		".cs":   "csharp",
		".rb":   "ruby",
		".php":  "php",
	}

	maxCount := 0
	for ext, count := range extCounts {
		if lang, ok := langMap[ext]; ok && count > maxCount {
			ctx.Language = lang
			maxCount = count
		}
	}
}

// buildFileTree builds the file tree structure
func (s *ProjectScanner) buildFileTree(rootPath, currentPath string) (*FileNode, error) {
	info, err := os.Stat(currentPath)
	if err != nil {
		return nil, err
	}

	relPath, _ := filepath.Rel(rootPath, currentPath)
	if relPath == "." {
		relPath = filepath.Base(currentPath)
	}

	node := &FileNode{
		Name:     filepath.Base(currentPath),
		Path:     currentPath,
		IsDir:    info.IsDir(),
		Size:     info.Size(),
		Modified: info.ModTime(),
	}

	if info.IsDir() {
		node.Children = make(map[string]*FileNode)
		
		entries, err := os.ReadDir(currentPath)
		if err != nil {
			return node, nil // Return partial node
		}

		for _, entry := range entries {
			name := entry.Name()
			
			// Skip ignored patterns
			if s.shouldIgnore(name) {
				continue
			}

			childPath := filepath.Join(currentPath, name)
			child, err := s.buildFileTree(rootPath, childPath)
			if err != nil {
				continue // Skip problematic entries
			}

			node.Children[name] = child
		}
	} else {
		// Detect file language
		node.Language = s.detectFileLanguage(currentPath)
		
		// For code files, extract metadata
		if isCodeFile(currentPath) {
			node.Metadata = s.extractFileMetadata(currentPath)
		}
	}

	return node, nil
}

// shouldIgnore checks if a path should be ignored
func (s *ProjectScanner) shouldIgnore(name string) bool {
	for _, pattern := range s.ignorePatterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
		if pattern == name {
			return true
		}
	}
	return false
}

// detectFileLanguage detects the language of a file
func (s *ProjectScanner) detectFileLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	
	langMap := map[string]string{
		".go":   "go",
		".js":   "javascript",
		".ts":   "typescript",
		".jsx":  "javascript",
		".tsx":  "typescript",
		".py":   "python",
		".java": "java",
		".rs":   "rust",
		".cpp":  "cpp",
		".c":    "c",
		".cs":   "csharp",
		".rb":   "ruby",
		".php":  "php",
		".md":   "markdown",
		".json": "json",
		".yaml": "yaml",
		".yml":  "yaml",
		".xml":  "xml",
		".html": "html",
		".css":  "css",
		".scss": "scss",
		".sql":  "sql",
		".sh":   "shell",
		".bash": "bash",
	}

	if lang, ok := langMap[ext]; ok {
		return lang
	}
	return ""
}

// extractFileMetadata extracts metadata from a code file
func (s *ProjectScanner) extractFileMetadata(path string) FileMetadata {
	metadata := FileMetadata{
		Imports:   make([]string, 0),
		Exports:   make([]string, 0),
		Functions: make([]string, 0),
		Classes:   make([]string, 0),
	}

	// Check if it's a test file
	basename := filepath.Base(path)
	metadata.IsTest = strings.Contains(basename, "_test") || 
		strings.Contains(basename, ".test.") ||
		strings.Contains(basename, ".spec.")

	// Check if it's a config file
	metadata.IsConfig = strings.Contains(basename, "config") ||
		strings.Contains(basename, ".conf") ||
		strings.Contains(basename, "rc")

	// Check if it's documentation
	ext := filepath.Ext(path)
	metadata.IsDocumentation = ext == ".md" || ext == ".rst" || 
		ext == ".txt" || strings.HasPrefix(basename, "README")

	// Count lines
	file, err := os.Open(path)
	if err != nil {
		return metadata
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := 0
	for scanner.Scan() {
		lines++
		// TODO: Extract imports, functions, classes based on language
	}
	metadata.Lines = lines

	return metadata
}

// loadGitignore loads patterns from .gitignore
func (s *ProjectScanner) loadGitignore(rootPath string) {
	gitignorePath := filepath.Join(rootPath, ".gitignore")
	file, err := os.Open(gitignorePath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			s.ignorePatterns = append(s.ignorePatterns, line)
		}
	}
}

// detectPatterns detects common project patterns
func (s *ProjectScanner) detectPatterns(ctx *ProjectContext) ProjectPatterns {
	patterns := ProjectPatterns{
		SourceDirs:     make([]string, 0),
		TestDirs:       make([]string, 0),
		ConfigFiles:    make([]string, 0),
		IgnorePatterns: s.ignorePatterns,
		BuildArtifacts: make([]string, 0),
	}

	// Common source directories
	commonSrcDirs := []string{"src", "lib", "app", "pkg", "internal", "cmd"}
	commonTestDirs := []string{"test", "tests", "spec", "specs", "__tests__"}
	
	// Check which directories exist
	for _, dir := range commonSrcDirs {
		if _, err := os.Stat(filepath.Join(ctx.RootPath, dir)); err == nil {
			patterns.SourceDirs = append(patterns.SourceDirs, dir)
		}
	}

	for _, dir := range commonTestDirs {
		if _, err := os.Stat(filepath.Join(ctx.RootPath, dir)); err == nil {
			patterns.TestDirs = append(patterns.TestDirs, dir)
		}
	}

	// Language-specific patterns
	switch ctx.Language {
	case "go":
		patterns.TestPattern = "*_test.go"
		patterns.BuildArtifacts = []string{"vendor", "bin"}
		patterns.ConfigFiles = []string{"go.mod", "go.sum"}
	case "javascript", "typescript":
		patterns.TestPattern = "*.test.js"
		patterns.BuildArtifacts = []string{"dist", "build", "node_modules"}
		patterns.ConfigFiles = []string{"package.json", "tsconfig.json", "webpack.config.js"}
	case "python":
		patterns.TestPattern = "test_*.py"
		patterns.BuildArtifacts = []string{"__pycache__", "*.pyc", "build", "dist", ".egg-info"}
		patterns.ConfigFiles = []string{"setup.py", "requirements.txt", "pyproject.toml"}
	}

	return patterns
}

// calculateStats calculates project statistics
func (s *ProjectScanner) calculateStats(ctx *ProjectContext) {
	stats := &ctx.Statistics
	s.walkFileTree(ctx.FileTree, func(node *FileNode) {
		if !node.IsDir {
			stats.TotalFiles++
			stats.TotalLines += node.Metadata.Lines
			
			if node.Language != "" {
				stats.FilesByLanguage[node.Language]++
			}
		}
	})

	// Find largest files
	var allFiles []FileInfo
	s.walkFileTree(ctx.FileTree, func(node *FileNode) {
		if !node.IsDir && node.Size > 0 {
			allFiles = append(allFiles, FileInfo{
				Path:  node.Path,
				Size:  node.Size,
				Lines: node.Metadata.Lines,
			})
		}
	})

	// Sort by size and keep top 10
	if len(allFiles) > 10 {
		// Simple selection of top 10
		stats.LargestFiles = allFiles[:10]
	} else {
		stats.LargestFiles = allFiles
	}
}

// walkFileTree walks the file tree and applies a function to each node
func (s *ProjectScanner) walkFileTree(node *FileNode, fn func(*FileNode)) {
	if node == nil {
		return
	}
	
	fn(node)
	
	if node.Children != nil {
		for _, child := range node.Children {
			s.walkFileTree(child, fn)
		}
	}
}

// RefreshFile refreshes information about a specific file
func (s *ProjectScanner) RefreshFile(ctx *ProjectContext, path string) error {
	// Find the parent directory
	dir := filepath.Dir(path)
	
	// Find the parent node
	parentNode := findFileNode(ctx.FileTree, dir)
	if parentNode == nil {
		return serr.New("parent directory not found in context")
	}

	// Rebuild just this file's node
	newNode, err := s.buildFileTree(ctx.RootPath, path)
	if err != nil {
		return serr.Wrap(err, "failed to refresh file")
	}

	// Update in parent's children
	filename := filepath.Base(path)
	parentNode.Children[filename] = newNode

	return nil
}

// isCodeFile checks if a file is a code file based on extension
func isCodeFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	codeExts := map[string]bool{
		".go": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
		".py": true, ".java": true, ".rs": true, ".cpp": true, ".c": true,
		".cs": true, ".rb": true, ".php": true, ".swift": true, ".kt": true,
		".scala": true, ".r": true, ".m": true, ".h": true, ".hpp": true,
		".cc": true, ".cxx": true, ".lua": true, ".dart": true, ".ex": true,
		".exs": true, ".clj": true, ".cljs": true, ".elm": true, ".ml": true,
		".mli": true, ".fs": true, ".fsx": true, ".pas": true, ".pl": true,
		".pm": true, ".tcl": true, ".groovy": true, ".gradle": true,
	}
	return codeExts[ext]
}