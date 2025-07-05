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

	return registry
}
