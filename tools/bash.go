package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rohanthewiz/serr"
)

// BashTool implements bash command execution
type BashTool struct{}

// GetDefinition returns the tool definition for the AI
func (t *BashTool) GetDefinition() Tool {
	return Tool{
		Name:        "bash",
		Description: "Execute a bash command and return the output",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"command": map[string]interface{}{
					"type":        "string",
					"description": "The bash command to execute",
				},
				"timeout": map[string]interface{}{
					"type":        "integer",
					"description": "Optional timeout in milliseconds (default: 120000)",
				},
			},
			"required": []string{"command"},
		},
	}
}

// Execute runs the bash command and returns the output
func (t *BashTool) Execute(input map[string]interface{}) (string, error) {
	command, ok := GetString(input, "command")
	if !ok || command == "" {
		return "", serr.New("command is required")
	}

	// Get timeout (default to 2 minutes)
	timeout := 120000
	if timeoutVal, ok := GetInt(input, "timeout"); ok && timeoutVal > 0 {
		timeout = timeoutVal
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeout)*time.Millisecond)
	defer cancel()

	// Create command
	cmd := exec.CommandContext(ctx, "bash", "-c", command)

	// Run command and capture output
	output, err := cmd.CombinedOutput()

	// Handle timeout
	if ctx.Err() == context.DeadlineExceeded {
		return string(output), serr.New(fmt.Sprintf("Command timed out after %dms", timeout))
	}

	// Build result
	result := string(output)

	// Add exit code if command failed
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result += fmt.Sprintf("\n\nExit code: %d", exitErr.ExitCode())
		}
	}

	// Truncate if too long
	const maxLength = 30000
	if len(result) > maxLength {
		result = result[:maxLength] + "\n\n[Output truncated...]"
	}

	// Clean up any trailing whitespace
	result = strings.TrimRight(result, "\n\r")

	return result, nil
}
