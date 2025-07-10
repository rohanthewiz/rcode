package tools

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/rohanthewiz/serr"
)

// GitStatusTool implements git status functionality
type GitStatusTool struct{}

// GetDefinition returns the tool definition for git status
func (t *GitStatusTool) GetDefinition() Tool {
	return Tool{
		Name:        "git_status",
		Description: "Show the working tree status of a git repository",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Repository path (defaults to current directory)",
				},
				"short": map[string]interface{}{
					"type":        "boolean",
					"description": "Show status in short format",
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs git status command
func (t *GitStatusTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		path = "."
	}

	shortFormat := false
	if val, exists := input["short"]; exists {
		if boolVal, ok := val.(bool); ok {
			shortFormat = boolVal
		}
	}

	// Build git command
	args := []string{"status"}
	if shortFormat {
		args = append(args, "-s")
	}

	// Execute git command
	cmd := exec.Command("git", args...)
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if it's not a git repository
		if strings.Contains(stderr.String(), "not a git repository") {
			return "", serr.New(fmt.Sprintf("Not a git repository: %s", path))
		}
		return "", serr.Wrap(err, fmt.Sprintf("Git status failed: %s", stderr.String()))
	}

	output := stdout.String()
	if output == "" && !shortFormat {
		output = "No changes detected. Working tree is clean."
	}

	return output, nil
}

// GitDiffTool implements git diff functionality
type GitDiffTool struct{}

// GetDefinition returns the tool definition for git diff
func (t *GitDiffTool) GetDefinition() Tool {
	return Tool{
		Name:        "git_diff",
		Description: "Show changes between commits, commit and working tree, etc",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Repository path (defaults to current directory)",
				},
				"staged": map[string]interface{}{
					"type":        "boolean",
					"description": "Show staged changes (--cached)",
				},
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Specific file to show diff for",
				},
				"stat": map[string]interface{}{
					"type":        "boolean",
					"description": "Show only stats (--stat)",
				},
				"name_only": map[string]interface{}{
					"type":        "boolean",
					"description": "Show only file names",
				},
				"commit": map[string]interface{}{
					"type":        "string",
					"description": "Show changes in a specific commit",
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs git diff command
func (t *GitDiffTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		path = "."
	}

	// Build git command
	args := []string{"diff"}

	// Handle options
	if staged, ok := input["staged"].(bool); ok && staged {
		args = append(args, "--cached")
	}

	if stat, ok := input["stat"].(bool); ok && stat {
		args = append(args, "--stat")
	}

	if nameOnly, ok := input["name_only"].(bool); ok && nameOnly {
		args = append(args, "--name-only")
	}

	if commit, ok := GetString(input, "commit"); ok && commit != "" {
		args = []string{"show", commit}
	}

	if file, ok := GetString(input, "file"); ok && file != "" {
		args = append(args, "--", file)
	}

	// Execute git command
	cmd := exec.Command("git", args...)
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if strings.Contains(stderr.String(), "not a git repository") {
			return "", serr.New(fmt.Sprintf("Not a git repository: %s", path))
		}
		return "", serr.Wrap(err, fmt.Sprintf("Git diff failed: %s", stderr.String()))
	}

	output := stdout.String()
	if output == "" {
		output = "No changes to display."
	}

	return output, nil
}

// GitLogTool implements git log functionality
type GitLogTool struct{}

// GetDefinition returns the tool definition for git log
func (t *GitLogTool) GetDefinition() Tool {
	return Tool{
		Name:        "git_log",
		Description: "Show commit logs",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Repository path (defaults to current directory)",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Limit the number of commits to show",
				},
				"oneline": map[string]interface{}{
					"type":        "boolean",
					"description": "Show each commit on one line",
				},
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Show commits affecting specific file",
				},
				"author": map[string]interface{}{
					"type":        "string",
					"description": "Filter commits by author",
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs git log command
func (t *GitLogTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		path = "."
	}

	// Build git command
	args := []string{"log"}

	// Handle options
	if limit, ok := GetInt(input, "limit"); ok && limit > 0 {
		args = append(args, fmt.Sprintf("-n%d", limit))
	}

	if oneline, ok := input["oneline"].(bool); ok && oneline {
		args = append(args, "--oneline")
	}

	if author, ok := GetString(input, "author"); ok && author != "" {
		args = append(args, "--author="+author)
	}

	if file, ok := GetString(input, "file"); ok && file != "" {
		args = append(args, "--", file)
	}

	// Execute git command
	cmd := exec.Command("git", args...)
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if strings.Contains(stderr.String(), "not a git repository") {
			return "", serr.New(fmt.Sprintf("Not a git repository: %s", path))
		}
		return "", serr.Wrap(err, fmt.Sprintf("Git log failed: %s", stderr.String()))
	}

	output := stdout.String()
	if output == "" {
		output = "No commits found."
	}

	return output, nil
}

// GitBranchTool implements git branch functionality
type GitBranchTool struct{}

// GetDefinition returns the tool definition for git branch
func (t *GitBranchTool) GetDefinition() Tool {
	return Tool{
		Name:        "git_branch",
		Description: "List, create, or delete branches",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Repository path (defaults to current directory)",
				},
				"all": map[string]interface{}{
					"type":        "boolean",
					"description": "List both local and remote branches",
				},
				"create": map[string]interface{}{
					"type":        "string",
					"description": "Create a new branch with this name",
				},
				"delete": map[string]interface{}{
					"type":        "string",
					"description": "Delete branch with this name",
				},
				"current": map[string]interface{}{
					"type":        "boolean",
					"description": "Show only the current branch name",
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs git branch command
func (t *GitBranchTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		path = "."
	}

	// Build git command based on operation
	var args []string

	if createBranch, ok := GetString(input, "create"); ok && createBranch != "" {
		args = []string{"branch", createBranch}
	} else if deleteBranch, ok := GetString(input, "delete"); ok && deleteBranch != "" {
		args = []string{"branch", "-d", deleteBranch}
	} else if current, ok := input["current"].(bool); ok && current {
		args = []string{"branch", "--show-current"}
	} else {
		args = []string{"branch"}
		if all, ok := input["all"].(bool); ok && all {
			args = append(args, "-a")
		}
	}

	// Execute git command
	cmd := exec.Command("git", args...)
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		if strings.Contains(stderr.String(), "not a git repository") {
			return "", serr.New(fmt.Sprintf("Not a git repository: %s", path))
		}
		return "", serr.Wrap(err, fmt.Sprintf("Git branch failed: %s", stderr.String()))
	}

	output := stdout.String()
	if output == "" && len(args) == 1 {
		output = "No branches found."
	}

	return strings.TrimSpace(output), nil
}