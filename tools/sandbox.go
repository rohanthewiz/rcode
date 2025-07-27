package tools

import (
	"path/filepath"
	"strings"

	"github.com/rohanthewiz/serr"
)

// SandboxedExecutor wraps a plugin executor with safety checks
type SandboxedExecutor struct {
	executor     Executor
	capabilities ToolCapabilities
	projectRoot  string
}

// NewSandboxedExecutor creates a sandboxed executor
func NewSandboxedExecutor(executor Executor, capabilities ToolCapabilities, projectRoot string) *SandboxedExecutor {
	return &SandboxedExecutor{
		executor:     executor,
		capabilities: capabilities,
		projectRoot:  projectRoot,
	}
}

// Execute runs the tool with sandbox restrictions
func (s *SandboxedExecutor) Execute(input map[string]interface{}) (string, error) {
	// Pre-execution validation based on capabilities
	if err := s.validateInput(input); err != nil {
		return "", err
	}

	// Execute with monitoring
	result, err := s.executor.Execute(input)

	// Post-execution validation
	if err := s.validateOutput(result); err != nil {
		return "", err
	}

	return result, err
}

// validateInput checks if the input is allowed based on capabilities
func (s *SandboxedExecutor) validateInput(input map[string]interface{}) error {
	// Check file paths if file operations are involved
	if path, ok := GetString(input, "path"); ok {
		if !s.capabilities.FileRead && !s.capabilities.FileWrite {
			return serr.New("tool does not have file access capability")
		}

		// Ensure path is within allowed directory
		if err := s.validatePath(path); err != nil {
			return err
		}
	}

	// Check for file_path parameter (some tools use this)
	if path, ok := GetString(input, "file_path"); ok {
		if !s.capabilities.FileRead && !s.capabilities.FileWrite {
			return serr.New("tool does not have file access capability")
		}

		// Ensure path is within allowed directory
		if err := s.validatePath(path); err != nil {
			return err
		}
	}

	// Check network URLs if present
	if _, ok := GetString(input, "url"); ok {
		if !s.capabilities.NetworkAccess {
			return serr.New("tool does not have network access capability")
		}
	}

	// Check for command execution
	if _, ok := GetString(input, "command"); ok {
		if !s.capabilities.ProcessSpawn {
			return serr.New("tool does not have process spawn capability")
		}
	}

	return nil
}

// validatePath ensures the path is within allowed boundaries
func (s *SandboxedExecutor) validatePath(path string) error {
	// If path is relative, make it absolute relative to project root
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.projectRoot, path)
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return serr.Wrap(err, "invalid path")
	}

	// Clean the path to remove any .. components
	absPath = filepath.Clean(absPath)

	allowedRoot := s.projectRoot
	if s.capabilities.WorkingDir != "" {
		allowedRoot = filepath.Join(s.projectRoot, s.capabilities.WorkingDir)
	}

	// Ensure allowed root is also clean
	allowedRoot = filepath.Clean(allowedRoot)

	// Check if the path is within the allowed directory
	if !strings.HasPrefix(absPath, allowedRoot) {
		return serr.New("path is outside allowed directory")
	}

	return nil
}

// validateOutput performs post-execution validation
func (s *SandboxedExecutor) validateOutput(output string) error {
	// Check output size limits (prevent memory exhaustion)
	const maxOutputSize = 10 * 1024 * 1024 // 10MB
	if len(output) > maxOutputSize {
		return serr.New("tool output exceeds maximum allowed size")
	}

	return nil
}

// WrapWithSandbox wraps an executor with sandbox restrictions if needed
func WrapWithSandbox(executor Executor, plugin ToolPlugin, projectRoot string) Executor {
	capabilities := plugin.GetCapabilities()

	// If the tool has no restrictions, wrap it in sandbox
	if !capabilities.FileRead && !capabilities.FileWrite &&
		!capabilities.NetworkAccess && !capabilities.ProcessSpawn {
		// Tool has no capabilities, can run without sandbox
		return executor
	}

	// Wrap with sandbox for capability enforcement
	return NewSandboxedExecutor(executor, capabilities, projectRoot)
}
