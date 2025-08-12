package tools

import (
	"github.com/rohanthewiz/logger"
	"rcode/config"
)

// DefaultRegistry creates a registry with all default tools
func DefaultRegistry() *Registry {
	registry := NewRegistry()

	// Register read file tool
	readTool := &ReadFileTool{}
	registry.Register(readTool.GetDefinition(), readTool)

	// Register write file tool
	writeTool := &WriteFileTool{}
	registry.Register(writeTool.GetDefinition(), writeTool)

	// Register bash tool
	bashTool := &BashTool{}
	registry.Register(bashTool.GetDefinition(), bashTool)

	// Register edit file tool
	editTool := &EditFileTool{}
	registry.Register(editTool.GetDefinition(), editTool)

	// Register smart edit tool for token-efficient editing
	// Provides multiple modes: patch, replace, sed, line with optimized responses
	smartEditTool := &SmartEditTool{}
	registry.Register(smartEditTool.GetDefinition(), smartEditTool)

	// Register search tool
	searchTool := &SearchTool{}
	registry.Register(searchTool.GetDefinition(), searchTool)

	// Register ripgrep tool for high-performance search
	// Ripgrep offers better performance and token efficiency with multiple output modes
	ripgrepTool := &RipgrepTool{}
	registry.Register(ripgrepTool.GetDefinition(), ripgrepTool)

	// Register directory operation tools
	listDirTool := &ListDirTool{}
	registry.Register(listDirTool.GetDefinition(), listDirTool)

	makeDirTool := &MakeDirTool{}
	registry.Register(makeDirTool.GetDefinition(), makeDirTool)

	removeTool := &RemoveTool{}
	registry.Register(removeTool.GetDefinition(), removeTool)

	treeTool := &TreeTool{}
	registry.Register(treeTool.GetDefinition(), treeTool)

	moveTool := &MoveTool{}
	registry.Register(moveTool.GetDefinition(), moveTool)

	// Register git tools
	gitStatusTool := &GitStatusTool{}
	registry.Register(gitStatusTool.GetDefinition(), gitStatusTool)

	gitDiffTool := &GitDiffTool{}
	registry.Register(gitDiffTool.GetDefinition(), gitDiffTool)

	gitLogTool := &GitLogTool{}
	registry.Register(gitLogTool.GetDefinition(), gitLogTool)

	gitBranchTool := &GitBranchTool{}
	registry.Register(gitBranchTool.GetDefinition(), gitBranchTool)

	gitAddTool := &GitAddTool{}
	registry.Register(gitAddTool.GetDefinition(), gitAddTool)

	gitCommitTool := &GitCommitTool{}
	registry.Register(gitCommitTool.GetDefinition(), gitCommitTool)

	gitPushTool := &GitPushTool{}
	registry.Register(gitPushTool.GetDefinition(), gitPushTool)

	gitPullTool := &GitPullTool{}
	registry.Register(gitPullTool.GetDefinition(), gitPullTool)

	gitCheckoutTool := &GitCheckoutTool{}
	registry.Register(gitCheckoutTool.GetDefinition(), gitCheckoutTool)

	gitMergeTool := &GitMergeTool{}
	registry.Register(gitMergeTool.GetDefinition(), gitMergeTool)

	// Register web search tool
	webSearchTool := &WebSearchTool{}
	registry.Register(webSearchTool.GetDefinition(), webSearchTool)

	// Register web fetch tool
	webFetchTool := &WebFetchTool{}
	registry.Register(webFetchTool.GetDefinition(), webFetchTool)

	// Register clipboard paste tool for handling clipboard content
	clipboardTool := &ClipboardPasteTool{}
	registry.Register(clipboardTool.GetDefinition(), clipboardTool)

	return registry
}

// DefaultRegistryWithPlugins creates a registry with default tools and plugins
func DefaultRegistryWithPlugins(projectRoot string) (*Registry, error) {
	registry := DefaultRegistry()

	// Load custom tools if enabled
	cfg := config.Get()
	if cfg.CustomToolsEnabled {
		loader := NewPluginLoader(cfg.CustomToolsPaths)
		if err := loader.LoadPlugins(); err != nil {
			logger.LogErr(err, "failed to load custom tool plugins")
			// Continue with built-in tools only
		} else {
			if err := loader.RegisterWithRegistry(registry, projectRoot); err != nil {
				logger.LogErr(err, "failed to register custom tools")
			}
		}
	}

	return registry, nil
}
