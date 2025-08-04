package context

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
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

// findRelevantFilesWithRipgrep uses ripgrep to quickly find relevant source files
// This is much faster than walking the entire directory tree for large projects
// Returns nil if ripgrep is not available, allowing fallback to regular file walking
func (s *ProjectScanner) findRelevantFilesWithRipgrep(rootPath string) ([]string, error) {
	// Check if ripgrep is available
	if _, err := exec.LookPath("rg"); err != nil {
		// Ripgrep not available, return nil to fall back to regular file walking
		return nil, nil
	}

	// Use ripgrep to find all source files, respecting .gitignore by default
	// We look for common programming patterns to identify source files
	allFiles := []string{}

	// Get all files using ripgrep's built-in file type detection
	// This is more efficient than multiple type-specific searches
	cmd := exec.Command("rg", 
		"--files",           // List files that would be searched
		"--hidden",          // Include hidden files (but still respect .gitignore)
		"--no-ignore-vcs",   // Don't ignore VCS ignore files
		"--ignore-file", filepath.Join(rootPath, ".gitignore"), // Use project's gitignore
		rootPath,
	)

	output, err := cmd.Output()
	if err != nil {
		// If ripgrep fails, return nil to fall back to regular walking
		return nil, nil
	}

	files := strings.Split(string(output), "\n")
	for _, file := range files {
		if file != "" {
			// Filter to relevant source and config files
			ext := strings.ToLower(filepath.Ext(file))
			base := filepath.Base(file)
			
			// Check if it's a relevant file type
			relevantExts := map[string]bool{
				".go": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
				".py": true, ".java": true, ".cpp": true, ".c": true, ".h": true,
				".cs": true, ".rb": true, ".php": true, ".swift": true, ".kt": true,
				".rs": true, ".scala": true, ".json": true, ".yaml": true, ".yml": true,
				".toml": true, ".xml": true, ".md": true, ".txt": true, ".sql": true,
				".sh": true, ".bash": true, ".zsh": true, ".fish": true,
			}
			
			relevantFiles := map[string]bool{
				"Makefile": true, "Dockerfile": true, "docker-compose.yml": true,
				"package.json": true, "go.mod": true, "go.sum": true,
				"requirements.txt": true, "Pipfile": true, "Cargo.toml": true,
				"pom.xml": true, "build.gradle": true, ".gitignore": true,
			}
			
			if relevantExts[ext] || relevantFiles[base] {
				allFiles = append(allFiles, file)
			}
		}
	}

	return allFiles, nil
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

// extractGoMetadata extracts Go-specific metadata
func (s *ProjectScanner) extractGoMetadata(line string, metadata *FileMetadata) {
	// Import statements
	if strings.HasPrefix(line, "import ") {
		// Single import
		if strings.Contains(line, "\"") {
			start := strings.Index(line, "\"")
			end := strings.LastIndex(line, "\"")
			if start != -1 && end > start {
				importPath := line[start+1:end]
				metadata.Imports = append(metadata.Imports, importPath)
			}
		}
	} else if strings.HasPrefix(line, "func ") {
		// Function definitions
		// Extract function name from "func (receiver) Name(" or "func Name("
		funcStart := 5 // len("func ")
		parenIndex := strings.Index(line[funcStart:], "(")
		if parenIndex != -1 {
			// Check for receiver
			funcDecl := line[funcStart:]
			if strings.HasPrefix(funcDecl, "(") {
				// Has receiver, find closing paren
				recvEnd := strings.Index(funcDecl, ")")
				if recvEnd != -1 {
					funcDecl = funcDecl[recvEnd+1:]
					funcDecl = strings.TrimSpace(funcDecl)
				}
			}
			// Extract function name
			if spaceIdx := strings.Index(funcDecl, "("); spaceIdx > 0 {
				funcName := strings.TrimSpace(funcDecl[:spaceIdx])
				if funcName != "" && isExported(funcName) {
					metadata.Functions = append(metadata.Functions, funcName)
					metadata.Exports = append(metadata.Exports, funcName)
				} else if funcName != "" {
					metadata.Functions = append(metadata.Functions, funcName)
				}
			}
		}
	} else if strings.HasPrefix(line, "type ") && strings.Contains(line, " struct") {
		// Struct definitions (classes in Go)
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			typeName := parts[1]
			metadata.Classes = append(metadata.Classes, typeName)
			if isExported(typeName) {
				metadata.Exports = append(metadata.Exports, typeName)
			}
		}
	} else if strings.HasPrefix(line, "type ") && strings.Contains(line, " interface") {
		// Interface definitions
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			typeName := parts[1]
			metadata.Classes = append(metadata.Classes, typeName)
			if isExported(typeName) {
				metadata.Exports = append(metadata.Exports, typeName)
			}
		}
	}
}

// extractJSMetadata extracts JavaScript/TypeScript metadata
func (s *ProjectScanner) extractJSMetadata(line string, metadata *FileMetadata) {
	// Import statements
	if strings.HasPrefix(line, "import ") {
		if strings.Contains(line, " from ") {
			// ES6 imports
			fromIdx := strings.Index(line, " from ")
			if fromIdx != -1 {
				modulePart := line[fromIdx+6:]
				modulePart = strings.Trim(modulePart, " ;")
				modulePart = strings.Trim(modulePart, "'\"")
				metadata.Imports = append(metadata.Imports, modulePart)
			}
		}
	} else if strings.HasPrefix(line, "const ") || strings.HasPrefix(line, "let ") || strings.HasPrefix(line, "var ") {
		// Check for function declarations
		if strings.Contains(line, " = function") || strings.Contains(line, " = (") || strings.Contains(line, " = async") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				varName := strings.TrimSuffix(parts[1], ":")
				metadata.Functions = append(metadata.Functions, varName)
			}
		}
	} else if strings.HasPrefix(line, "function ") {
		// Function declarations
		funcStart := 9 // len("function ")
		parenIdx := strings.Index(line[funcStart:], "(")
		if parenIdx > 0 {
			funcName := strings.TrimSpace(line[funcStart:funcStart+parenIdx])
			metadata.Functions = append(metadata.Functions, funcName)
		}
	} else if strings.HasPrefix(line, "class ") {
		// Class declarations
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			className := parts[1]
			if strings.Contains(className, "{") {
				className = strings.TrimSuffix(className, "{")
			}
			metadata.Classes = append(metadata.Classes, className)
		}
	} else if strings.HasPrefix(line, "export ") {
		// Export statements
		if strings.Contains(line, "function ") {
			// Export function
			funcIdx := strings.Index(line, "function ")
			if funcIdx != -1 {
				funcPart := line[funcIdx+9:]
				if parenIdx := strings.Index(funcPart, "("); parenIdx > 0 {
					funcName := strings.TrimSpace(funcPart[:parenIdx])
					metadata.Exports = append(metadata.Exports, funcName)
					metadata.Functions = append(metadata.Functions, funcName)
				}
			}
		} else if strings.Contains(line, "class ") {
			// Export class
			classIdx := strings.Index(line, "class ")
			if classIdx != -1 {
				classPart := line[classIdx+6:]
				parts := strings.Fields(classPart)
				if len(parts) > 0 {
					className := strings.TrimSuffix(parts[0], "{")
					metadata.Exports = append(metadata.Exports, className)
					metadata.Classes = append(metadata.Classes, className)
				}
			}
		} else if strings.Contains(line, "const ") || strings.Contains(line, "let ") {
			// Export const/let
			parts := strings.Fields(line)
			for i, part := range parts {
				if (part == "const" || part == "let") && i+1 < len(parts) {
					varName := strings.TrimSuffix(parts[i+1], ";")
					metadata.Exports = append(metadata.Exports, varName)
					break
				}
			}
		}
	}
}

// extractPythonMetadata extracts Python metadata
func (s *ProjectScanner) extractPythonMetadata(line string, metadata *FileMetadata) {
	// Import statements
	if strings.HasPrefix(line, "import ") {
		importPart := strings.TrimPrefix(line, "import ")
		imports := strings.Split(importPart, ",")
		for _, imp := range imports {
			imp = strings.TrimSpace(imp)
			if asIdx := strings.Index(imp, " as "); asIdx != -1 {
				imp = imp[:asIdx]
			}
			metadata.Imports = append(metadata.Imports, imp)
		}
	} else if strings.HasPrefix(line, "from ") {
		// from X import Y
		if importIdx := strings.Index(line, " import "); importIdx != -1 {
			module := line[5:importIdx] // Skip "from "
			metadata.Imports = append(metadata.Imports, module)
		}
	} else if strings.HasPrefix(line, "def ") {
		// Function definitions
		defPart := strings.TrimPrefix(line, "def ")
		if parenIdx := strings.Index(defPart, "("); parenIdx > 0 {
			funcName := strings.TrimSpace(defPart[:parenIdx])
			metadata.Functions = append(metadata.Functions, funcName)
			// In Python, functions starting with _ are private
			if !strings.HasPrefix(funcName, "_") {
				metadata.Exports = append(metadata.Exports, funcName)
			}
		}
	} else if strings.HasPrefix(line, "class ") {
		// Class definitions
		classPart := strings.TrimPrefix(line, "class ")
		if colonIdx := strings.Index(classPart, ":"); colonIdx > 0 {
			classDef := classPart[:colonIdx]
			if parenIdx := strings.Index(classDef, "("); parenIdx > 0 {
				className := strings.TrimSpace(classDef[:parenIdx])
				metadata.Classes = append(metadata.Classes, className)
				if !strings.HasPrefix(className, "_") {
					metadata.Exports = append(metadata.Exports, className)
				}
			} else {
				className := strings.TrimSpace(classDef)
				metadata.Classes = append(metadata.Classes, className)
				if !strings.HasPrefix(className, "_") {
					metadata.Exports = append(metadata.Exports, className)
				}
			}
		}
	}
}

// extractJavaMetadata extracts Java metadata
func (s *ProjectScanner) extractJavaMetadata(line string, metadata *FileMetadata) {
	// Import statements
	if strings.HasPrefix(line, "import ") {
		importStmt := strings.TrimSuffix(strings.TrimPrefix(line, "import "), ";")
		metadata.Imports = append(metadata.Imports, strings.TrimSpace(importStmt))
	} else if strings.Contains(line, " class ") {
		// Class definitions
		classIdx := strings.Index(line, " class ")
		if classIdx != -1 {
			classPart := line[classIdx+7:]
			parts := strings.Fields(classPart)
			if len(parts) > 0 {
				className := parts[0]
				if strings.Contains(className, "{") {
					className = strings.TrimSuffix(className, "{")
				}
				metadata.Classes = append(metadata.Classes, className)
				if strings.Contains(line[:classIdx], "public") {
					metadata.Exports = append(metadata.Exports, className)
				}
			}
		}
	} else if strings.Contains(line, " interface ") {
		// Interface definitions
		intIdx := strings.Index(line, " interface ")
		if intIdx != -1 {
			intPart := line[intIdx+11:]
			parts := strings.Fields(intPart)
			if len(parts) > 0 {
				intName := parts[0]
				if strings.Contains(intName, "{") {
					intName = strings.TrimSuffix(intName, "{")
				}
				metadata.Classes = append(metadata.Classes, intName)
				if strings.Contains(line[:intIdx], "public") {
					metadata.Exports = append(metadata.Exports, intName)
				}
			}
		}
	} else if strings.Contains(line, "(") && strings.Contains(line, ")") && strings.Contains(line, "{") {
		// Method definitions (simplified)
		parenIdx := strings.Index(line, "(")
		if parenIdx > 0 {
			beforeParen := line[:parenIdx]
			parts := strings.Fields(beforeParen)
			if len(parts) >= 2 {
				methodName := parts[len(parts)-1]
				if methodName != "" && !strings.Contains(methodName, "if") && !strings.Contains(methodName, "for") && !strings.Contains(methodName, "while") {
					metadata.Functions = append(metadata.Functions, methodName)
					if strings.Contains(beforeParen, "public") {
						metadata.Exports = append(metadata.Exports, methodName)
					}
				}
			}
		}
	}
}

// extractRustMetadata extracts Rust metadata
func (s *ProjectScanner) extractRustMetadata(line string, metadata *FileMetadata) {
	// Use statements
	if strings.HasPrefix(line, "use ") {
		usePart := strings.TrimSuffix(strings.TrimPrefix(line, "use "), ";")
		metadata.Imports = append(metadata.Imports, strings.TrimSpace(usePart))
	} else if strings.HasPrefix(line, "fn ") {
		// Function definitions
		fnPart := strings.TrimPrefix(line, "fn ")
		if parenIdx := strings.Index(fnPart, "("); parenIdx > 0 {
			funcName := strings.TrimSpace(fnPart[:parenIdx])
			metadata.Functions = append(metadata.Functions, funcName)
			// Check if previous line had pub
			metadata.Exports = append(metadata.Exports, funcName)
		}
	} else if strings.HasPrefix(line, "pub fn ") {
		// Public function definitions
		fnPart := strings.TrimPrefix(line, "pub fn ")
		if parenIdx := strings.Index(fnPart, "("); parenIdx > 0 {
			funcName := strings.TrimSpace(fnPart[:parenIdx])
			metadata.Functions = append(metadata.Functions, funcName)
			metadata.Exports = append(metadata.Exports, funcName)
		}
	} else if strings.Contains(line, "struct ") {
		// Struct definitions
		structIdx := strings.Index(line, "struct ")
		if structIdx != -1 {
			structPart := line[structIdx+7:]
			parts := strings.Fields(structPart)
			if len(parts) > 0 {
				structName := strings.TrimSuffix(parts[0], "{")
				metadata.Classes = append(metadata.Classes, structName)
				if strings.HasPrefix(line, "pub ") {
					metadata.Exports = append(metadata.Exports, structName)
				}
			}
		}
	} else if strings.Contains(line, "enum ") {
		// Enum definitions
		enumIdx := strings.Index(line, "enum ")
		if enumIdx != -1 {
			enumPart := line[enumIdx+5:]
			parts := strings.Fields(enumPart)
			if len(parts) > 0 {
				enumName := strings.TrimSuffix(parts[0], "{")
				metadata.Classes = append(metadata.Classes, enumName)
				if strings.HasPrefix(line, "pub ") {
					metadata.Exports = append(metadata.Exports, enumName)
				}
			}
		}
	}
}

// isExported checks if a Go identifier is exported (starts with uppercase)
func isExported(name string) bool {
	if name == "" {
		return false
	}
	r := []rune(name)
	return r[0] >= 'A' && r[0] <= 'Z'
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

	// Read file and extract metadata based on language
	file, err := os.Open(path)
	if err != nil {
		return metadata
	}
	defer file.Close()

	// Detect language
	lang := s.detectFileLanguage(path)
	
	scanner := bufio.NewScanner(file)
	lines := 0
	inImportBlock := false // For Go multi-line imports
	
	for scanner.Scan() {
		lines++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		
		// Skip empty lines and comments for analysis
		if trimmed == "" {
			continue
		}
		
		// Handle Go import blocks
		if lang == "go" {
			if trimmed == "import (" {
				inImportBlock = true
				continue
			} else if inImportBlock && trimmed == ")" {
				inImportBlock = false
				continue
			} else if inImportBlock {
				// Extract import from block
				imp := strings.Trim(trimmed, "\t\"")
				if imp != "" && !strings.HasPrefix(imp, "//") {
					metadata.Imports = append(metadata.Imports, imp)
				}
				continue
			}
		}
		
		// Extract based on language
		switch lang {
		case "go":
			s.extractGoMetadata(trimmed, &metadata)
		case "javascript", "typescript":
			s.extractJSMetadata(trimmed, &metadata)
		case "python":
			s.extractPythonMetadata(trimmed, &metadata)
		case "java":
			s.extractJavaMetadata(trimmed, &metadata)
		case "rust":
			s.extractRustMetadata(trimmed, &metadata)
		}
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