package tools

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

	// Register search tool
	searchTool := &SearchTool{}
	registry.Register(searchTool.GetDefinition(), searchTool)

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

	// Register web search tool
	webSearchTool := &WebSearchTool{}
	registry.Register(webSearchTool.GetDefinition(), webSearchTool)

	// Register web fetch tool
	webFetchTool := &WebFetchTool{}
	registry.Register(webFetchTool.GetDefinition(), webFetchTool)

	return registry
}
