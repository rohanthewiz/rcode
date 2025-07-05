package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rohanthewiz/serr"
	"github.com/sst/opencode/server-go/internal/schema"
	"github.com/sst/opencode/server-go/internal/tool"
)

// BashTool implements command execution functionality.
// It executes shell commands with timeout and output streaming.
type BashTool struct {
	description string
	defaultTimeout time.Duration
}

// NewBashTool creates a new bash tool instance
func NewBashTool() *BashTool {
	return &BashTool{
		description: `Executes bash commands in a shell environment.
Supports command execution with optional timeout.
Returns command output and exit status.
Use with caution as this executes arbitrary commands.`,
		defaultTimeout: 2 * time.Minute,
	}
}

func (t *BashTool) ID() string {
	return "bash"
}

func (t *BashTool) Description() string {
	return t.description
}

func (t *BashTool) Parameters() tool.Schema {
	return schema.Object(map[string]tool.Schema{
		"command": schema.String().Describe("The bash command to execute"),
		"timeout": schema.Optional(
			schema.Number().
				Describe("Timeout in milliseconds (default: 120000, max: 600000)").
				Minimum(0).
				Maximum(600000),
		),
		"description": schema.Optional(
			schema.String().Describe("Brief description of what this command does"),
		),
	}, "command")
}

func (t *BashTool) Execute(ctx tool.Context, params map[string]any) (tool.Result, error) {
	// Extract parameters
	command, _ := params["command"].(string)
	description, _ := params["description"].(string)
	
	// Handle timeout
	timeout := t.defaultTimeout
	if timeoutMs, ok := params["timeout"].(float64); ok {
		timeout = time.Duration(timeoutMs) * time.Millisecond
	}
	
	// Create command with timeout context
	cmdCtx, cancel := context.WithTimeout(ctx.Abort, timeout)
	defer cancel()
	
	// Log what we're executing
	if description != "" {
		ctx.Metadata(map[string]any{
			"description": description,
			"command":     command,
		})
	}
	
	// Create the command
	cmd := exec.CommandContext(cmdCtx, "bash", "-c", command)
	
	// Set up output buffers
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// Execute the command
	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)
	
	// Build output
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += "STDERR:\n" + stderr.String()
	}
	
	// Trim excessive output (limit to 30k characters like TypeScript version)
	const maxOutput = 30000
	truncated := false
	if len(output) > maxOutput {
		output = output[:maxOutput] + "\n... (output truncated)"
		truncated = true
	}
	
	// Prepare metadata
	metadata := map[string]any{
		"duration_ms": duration.Milliseconds(),
		"exit_code":   0,
		"truncated":   truncated,
	}
	
	// Handle errors
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Command executed but returned non-zero exit code
			metadata["exit_code"] = exitErr.ExitCode()
			
			// Include exit code in output
			output += fmt.Sprintf("\n\nProcess exited with code %d", exitErr.ExitCode())
		} else if cmdCtx.Err() == context.DeadlineExceeded {
			// Command timed out
			metadata["timeout"] = true
			return tool.Result{
				Output:   output + "\n\nCommand timed out",
				Metadata: metadata,
			}, nil
		} else {
			// Other execution error
			return tool.Result{}, serr.Wrap(err, "failed to execute command")
		}
	}
	
	// Send final metadata update
	ctx.Metadata(metadata)
	
	return tool.Result{
		Output:   strings.TrimRight(output, "\n"),
		Metadata: metadata,
	}, nil
}