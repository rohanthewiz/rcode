package context

import (
	"time"
)

// ProjectContext represents the current project's context
type ProjectContext struct {
	RootPath      string                   `json:"root_path"`
	Language      string                   `json:"language"`
	Framework     string                   `json:"framework"`
	Dependencies  []Dependency             `json:"dependencies"`
	FileTree      *FileNode                `json:"file_tree"`
	RecentFiles   []string                 `json:"recent_files"`
	ModifiedFiles map[string]time.Time     `json:"modified_files"`
	Patterns      ProjectPatterns          `json:"patterns"`
	Statistics    ProjectStats             `json:"statistics"`
}

// Dependency represents a project dependency
type Dependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Type    string `json:"type"` // e.g., "go_module", "npm_package", "pip_package"
}

// FileNode represents a file or directory in the project tree
type FileNode struct {
	Name     string               `json:"name"`
	Path     string               `json:"path"`
	IsDir    bool                 `json:"is_dir"`
	Size     int64                `json:"size"`
	Modified time.Time            `json:"modified"`
	Children map[string]*FileNode `json:"children,omitempty"`
	Language string               `json:"language,omitempty"`
	Metadata FileMetadata         `json:"metadata,omitempty"`
}

// FileMetadata contains additional file information
type FileMetadata struct {
	Lines         int      `json:"lines"`
	Imports       []string `json:"imports,omitempty"`
	Exports       []string `json:"exports,omitempty"`
	Functions     []string `json:"functions,omitempty"`
	Classes       []string `json:"classes,omitempty"`
	IsTest        bool     `json:"is_test"`
	IsConfig      bool     `json:"is_config"`
	IsDocumentation bool   `json:"is_documentation"`
}

// ProjectPatterns contains detected project patterns
type ProjectPatterns struct {
	TestPattern      string   `json:"test_pattern"`      // e.g., "*_test.go", "*.test.js"
	SourceDirs       []string `json:"source_dirs"`       // e.g., ["src", "lib"]
	TestDirs         []string `json:"test_dirs"`         // e.g., ["tests", "test"]
	ConfigFiles      []string `json:"config_files"`      // e.g., ["go.mod", "package.json"]
	IgnorePatterns   []string `json:"ignore_patterns"`   // from .gitignore
	BuildArtifacts   []string `json:"build_artifacts"`   // e.g., ["dist", "build", "target"]
}

// ProjectStats contains project statistics
type ProjectStats struct {
	TotalFiles      int            `json:"total_files"`
	TotalLines      int            `json:"total_lines"`
	FilesByLanguage map[string]int `json:"files_by_language"`
	LargestFiles    []FileInfo     `json:"largest_files"`
}

// FileInfo represents basic file information
type FileInfo struct {
	Path  string `json:"path"`
	Size  int64  `json:"size"`
	Lines int    `json:"lines"`
}

// ChangeType represents the type of file change
type ChangeType string

const (
	ChangeTypeCreate ChangeType = "create"
	ChangeTypeModify ChangeType = "modify"
	ChangeTypeDelete ChangeType = "delete"
	ChangeTypeRename ChangeType = "rename"
)

// FileChange represents a file change event
type FileChange struct {
	Path       string                 `json:"path"`
	Type       ChangeType             `json:"type"`
	Timestamp  time.Time              `json:"timestamp"`
	OldPath    string                 `json:"old_path,omitempty"` // for renames
	Tool       string                 `json:"tool,omitempty"`      // which tool made the change
	Details    map[string]interface{} `json:"details,omitempty"`   // additional tool-specific details
}

// TaskContext represents context for a specific task
type TaskContext struct {
	Task          string        `json:"task"`
	RelevantFiles []string      `json:"relevant_files"`
	SearchTerms   []string      `json:"search_terms"`
	FileScores    map[string]float64 `json:"file_scores"`
	MaxFiles      int           `json:"max_files"`
}

// ContextWindow represents the current context window
type ContextWindow struct {
	Files        []ContextFile `json:"files"`
	TotalTokens  int           `json:"total_tokens"`
	MaxTokens    int           `json:"max_tokens"`
	Priority     string        `json:"priority"` // "relevance", "recent", "modified"
}

// ContextFile represents a file in the context window
type ContextFile struct {
	Path     string  `json:"path"`
	Content  string  `json:"content"`
	Tokens   int     `json:"tokens"`
	Score    float64 `json:"score"`
	Included bool    `json:"included"`
}