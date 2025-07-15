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
		// Git status errors might be temporary (index lock, etc)
		return "", WrapFileSystemError(serr.Wrap(err, fmt.Sprintf("Git status failed: %s", stderr.String())))
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
		// Git diff errors might be temporary (index lock, etc)
		return "", WrapFileSystemError(serr.Wrap(err, fmt.Sprintf("Git diff failed: %s", stderr.String())))
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
		// Git log errors are usually permanent
		return "", NewPermanentError(serr.Wrap(err, fmt.Sprintf("Git log failed: %s", stderr.String())), "git error")
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
		// Git branch errors might be temporary (lock issues)
		return "", WrapFileSystemError(serr.Wrap(err, fmt.Sprintf("Git branch failed: %s", stderr.String())))
	}

	output := stdout.String()
	if output == "" && len(args) == 1 {
		output = "No branches found."
	}

	return strings.TrimSpace(output), nil
}

// GitAddTool implements git add functionality
type GitAddTool struct{}

// GetDefinition returns the tool definition for git add
func (t *GitAddTool) GetDefinition() Tool {
	return Tool{
		Name:        "git_add",
		Description: "Add file contents to the staging area",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Repository path (defaults to current directory)",
				},
				"files": map[string]interface{}{
					"type":        "array",
					"description": "List of files to add (if empty, use all option)",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"all": map[string]interface{}{
					"type":        "boolean",
					"description": "Add all changed files (git add -A)",
				},
				"update": map[string]interface{}{
					"type":        "boolean",
					"description": "Update tracked files only (git add -u)",
				},
				"interactive": map[string]interface{}{
					"type":        "boolean",
					"description": "Interactive mode (git add -i) - not recommended for automation",
				},
				"patch": map[string]interface{}{
					"type":        "boolean",
					"description": "Patch mode to stage hunks (git add -p) - not recommended for automation",
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs git add command
func (t *GitAddTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		path = "."
	}

	// Build git command
	args := []string{"add"}
	
	// Check for interactive modes and warn
	if interactive, ok := input["interactive"].(bool); ok && interactive {
		return "", serr.New("Interactive mode (git add -i) is not supported in automated contexts")
	}
	if patch, ok := input["patch"].(bool); ok && patch {
		return "", serr.New("Patch mode (git add -p) is not supported in automated contexts")
	}

	// Handle different add modes
	filesAdded := false
	
	if all, ok := input["all"].(bool); ok && all {
		args = append(args, "-A")
		filesAdded = true
	} else if update, ok := input["update"].(bool); ok && update {
		args = append(args, "-u")
		filesAdded = true
	} else if files, ok := input["files"].([]interface{}); ok && len(files) > 0 {
		// Add specific files
		for _, file := range files {
			if fileStr, ok := file.(string); ok {
				args = append(args, fileStr)
				filesAdded = true
			}
		}
	}

	// If no files specified and no flags, default to adding all
	if !filesAdded {
		args = append(args, ".")
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
		// Git add often exits with non-zero for warnings, check if there's actual error
		errMsg := stderr.String()
		if errMsg != "" && !strings.Contains(errMsg, "warning:") {
			return "", WrapFileSystemError(serr.Wrap(err, fmt.Sprintf("Git add failed: %s", errMsg)))
		}
	}

	// Get the status to show what was added
	statusCmd := exec.Command("git", "status", "--short")
	statusCmd.Dir = path
	
	var statusOut bytes.Buffer
	statusCmd.Stdout = &statusOut
	statusCmd.Run()

	result := "Files staged successfully.\n\nCurrent status:\n" + statusOut.String()
	
	// Include any warnings from the add command
	if stderr.Len() > 0 {
		result += "\nWarnings:\n" + stderr.String()
	}

	return result, nil
}

// GitCommitTool implements git commit functionality
type GitCommitTool struct{}

// GetDefinition returns the tool definition for git commit
func (t *GitCommitTool) GetDefinition() Tool {
	return Tool{
		Name:        "git_commit",
		Description: "Record changes to the repository",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Repository path (defaults to current directory)",
				},
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Commit message (required unless using amend with no message change)",
				},
				"amend": map[string]interface{}{
					"type":        "boolean",
					"description": "Amend the last commit",
				},
				"all": map[string]interface{}{
					"type":        "boolean",
					"description": "Automatically stage all modified files before commit",
				},
				"allow_empty": map[string]interface{}{
					"type":        "boolean",
					"description": "Allow creating an empty commit",
				},
				"no_verify": map[string]interface{}{
					"type":        "boolean",
					"description": "Skip pre-commit hooks",
				},
				"author": map[string]interface{}{
					"type":        "string",
					"description": "Override commit author (format: 'Name <email>')",
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs git commit command
func (t *GitCommitTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		path = "."
	}

	// Build git command
	args := []string{"commit"}

	// Get commit message
	message, hasMessage := GetString(input, "message")
	amend, isAmend := input["amend"].(bool)
	
	// Validate message requirement
	if !hasMessage && (!isAmend || !amend) {
		return "", serr.New("Commit message is required unless amending")
	}

	// Add message if provided
	if hasMessage && message != "" {
		args = append(args, "-m", message)
	}

	// Handle options
	if amend {
		args = append(args, "--amend")
		if !hasMessage {
			// Amending without changing the message
			args = append(args, "--no-edit")
		}
	}

	if all, ok := input["all"].(bool); ok && all {
		args = append(args, "-a")
	}

	if allowEmpty, ok := input["allow_empty"].(bool); ok && allowEmpty {
		args = append(args, "--allow-empty")
	}

	if noVerify, ok := input["no_verify"].(bool); ok && noVerify {
		args = append(args, "--no-verify")
	}

	if author, ok := GetString(input, "author"); ok && author != "" {
		args = append(args, "--author", author)
	}

	// Execute git command
	cmd := exec.Command("git", args...)
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if strings.Contains(errMsg, "not a git repository") {
			return "", NewPermanentError(serr.New(fmt.Sprintf("Not a git repository: %s", path)), "invalid repository")
		}
		if strings.Contains(errMsg, "nothing to commit") {
			return "Nothing to commit, working tree clean", nil
		}
		if strings.Contains(errMsg, "no changes added to commit") {
			return "", NewPermanentError(serr.New("No changes staged for commit. Use git_add first or use the 'all' option"), "no changes")
		}
		// Commit errors might be temporary (index lock, hooks failing, etc)
		return "", WrapFileSystemError(serr.Wrap(err, fmt.Sprintf("Git commit failed: %s", errMsg)))
	}

	// Get the commit info
	result := stdout.String()
	
	// Get the latest commit details
	logCmd := exec.Command("git", "log", "-1", "--oneline")
	logCmd.Dir = path
	
	var logOut bytes.Buffer
	logCmd.Stdout = &logOut
	if logCmd.Run() == nil {
		result += "\n\nLatest commit: " + logOut.String()
	}

	return result, nil
}

// GitPushTool implements git push functionality
type GitPushTool struct{}

// GetDefinition returns the tool definition for git push
func (t *GitPushTool) GetDefinition() Tool {
	return Tool{
		Name:        "git_push",
		Description: "Update remote refs along with associated objects",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Repository path (defaults to current directory)",
				},
				"remote": map[string]interface{}{
					"type":        "string",
					"description": "Remote name (defaults to 'origin')",
				},
				"branch": map[string]interface{}{
					"type":        "string",
					"description": "Branch to push (defaults to current branch)",
				},
				"all": map[string]interface{}{
					"type":        "boolean",
					"description": "Push all branches",
				},
				"tags": map[string]interface{}{
					"type":        "boolean",
					"description": "Push tags",
				},
				"force": map[string]interface{}{
					"type":        "boolean",
					"description": "Force push (use with caution!)",
				},
				"force_with_lease": map[string]interface{}{
					"type":        "boolean",
					"description": "Safer force push that checks remote hasn't changed",
				},
				"set_upstream": map[string]interface{}{
					"type":        "boolean",
					"description": "Set upstream branch (-u flag)",
				},
				"dry_run": map[string]interface{}{
					"type":        "boolean",
					"description": "Perform a dry run without actually pushing",
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs git push command
func (t *GitPushTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		path = "."
	}

	// Build git command
	args := []string{"push"}

	// Handle remote and branch
	remote, hasRemote := GetString(input, "remote")
	if !hasRemote || remote == "" {
		remote = "origin"
	}

	branch, hasBranch := GetString(input, "branch")
	
	// Handle options
	if all, ok := input["all"].(bool); ok && all {
		args = append(args, "--all")
		// Don't add remote/branch if pushing all
	} else {
		// Add remote
		args = append(args, remote)
		
		// Add branch if specified
		if hasBranch && branch != "" {
			args = append(args, branch)
		}
	}

	if tags, ok := input["tags"].(bool); ok && tags {
		args = append(args, "--tags")
	}

	// Handle force options (mutually exclusive)
	if force, ok := input["force"].(bool); ok && force {
		args = append(args, "--force")
	} else if forceWithLease, ok := input["force_with_lease"].(bool); ok && forceWithLease {
		args = append(args, "--force-with-lease")
	}

	if setUpstream, ok := input["set_upstream"].(bool); ok && setUpstream {
		args = append(args, "-u")
	}

	if dryRun, ok := input["dry_run"].(bool); ok && dryRun {
		args = append(args, "--dry-run")
	}

	// Execute git command
	cmd := exec.Command("git", args...)
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if strings.Contains(errMsg, "not a git repository") {
			return "", NewPermanentError(serr.New(fmt.Sprintf("Not a git repository: %s", path)), "invalid repository")
		}
		if strings.Contains(errMsg, "Could not read from remote repository") {
			return "", NewRetryableError(serr.New("Failed to connect to remote repository. Check your authentication and network connection"), "network error")
		}
		if strings.Contains(errMsg, "Connection refused") || strings.Contains(errMsg, "Connection timed out") ||
		   strings.Contains(errMsg, "Could not resolve host") || strings.Contains(errMsg, "Network is unreachable") {
			return "", NewRetryableError(serr.New(fmt.Sprintf("Network error during push: %s", errMsg)), "network error")
		}
		if strings.Contains(errMsg, "failed to push") || strings.Contains(errMsg, "rejected") {
			// Include the full error for push failures as they often contain important info
			// Most push rejections are permanent (non-fast-forward, permissions, etc)
			return "", NewPermanentError(serr.New(fmt.Sprintf("Push failed: %s", errMsg)), "push rejected")
		}
		// Default to retryable for unknown errors as they might be transient
		return "", NewRetryableError(serr.Wrap(err, fmt.Sprintf("Git push failed: %s", errMsg)), "unknown error")
	}

	// Combine stdout and stderr for push (git often puts progress to stderr)
	result := stdout.String()
	if stderr.String() != "" {
		result += stderr.String()
	}

	if result == "" {
		result = "Push completed successfully"
	}

	// Add warning for force push
	if force, ok := input["force"].(bool); ok && force {
		result = "⚠️  FORCE PUSH COMPLETED ⚠️\n\n" + result
	}

	return result, nil
}

// GitPullTool implements git pull functionality
type GitPullTool struct{}

// GetDefinition returns the tool definition for git pull
func (t *GitPullTool) GetDefinition() Tool {
	return Tool{
		Name:        "git_pull",
		Description: "Fetch from and integrate with another repository or a local branch",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Repository path (defaults to current directory)",
				},
				"remote": map[string]interface{}{
					"type":        "string",
					"description": "Remote name (defaults to 'origin')",
				},
				"branch": map[string]interface{}{
					"type":        "string",
					"description": "Branch to pull (defaults to current branch)",
				},
				"rebase": map[string]interface{}{
					"type":        "boolean",
					"description": "Rebase instead of merge",
				},
				"no_commit": map[string]interface{}{
					"type":        "boolean",
					"description": "Perform the merge but don't commit",
				},
				"no_ff": map[string]interface{}{
					"type":        "boolean",
					"description": "Create a merge commit even for fast-forward",
				},
				"strategy": map[string]interface{}{
					"type":        "string",
					"description": "Merge strategy (e.g., 'ours', 'theirs', 'recursive')",
				},
				"autostash": map[string]interface{}{
					"type":        "boolean",
					"description": "Automatically stash and pop local changes",
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs git pull command
func (t *GitPullTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		path = "."
	}

	// Build git command
	args := []string{"pull"}

	// Handle options
	if rebase, ok := input["rebase"].(bool); ok && rebase {
		args = append(args, "--rebase")
	}

	if noCommit, ok := input["no_commit"].(bool); ok && noCommit {
		args = append(args, "--no-commit")
	}

	if noFF, ok := input["no_ff"].(bool); ok && noFF {
		args = append(args, "--no-ff")
	}

	if strategy, ok := GetString(input, "strategy"); ok && strategy != "" {
		args = append(args, "--strategy", strategy)
	}

	if autostash, ok := input["autostash"].(bool); ok && autostash {
		args = append(args, "--autostash")
	}

	// Handle remote and branch
	remote, hasRemote := GetString(input, "remote")
	if !hasRemote || remote == "" {
		remote = "origin"
	}
	
	// Only add remote/branch if explicitly specified
	if hasRemote {
		args = append(args, remote)
		
		if branch, ok := GetString(input, "branch"); ok && branch != "" {
			args = append(args, branch)
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
		errMsg := stderr.String()
		if strings.Contains(errMsg, "not a git repository") {
			return "", NewPermanentError(serr.New(fmt.Sprintf("Not a git repository: %s", path)), "invalid repository")
		}
		if strings.Contains(errMsg, "Could not read from remote repository") {
			return "", NewRetryableError(serr.New("Failed to connect to remote repository. Check your authentication and network connection"), "network error")
		}
		if strings.Contains(errMsg, "Connection refused") || strings.Contains(errMsg, "Connection timed out") ||
		   strings.Contains(errMsg, "Could not resolve host") || strings.Contains(errMsg, "Network is unreachable") {
			return "", NewRetryableError(serr.New(fmt.Sprintf("Network error during pull: %s", errMsg)), "network error")
		}
		if strings.Contains(errMsg, "Automatic merge failed") {
			// Merge conflict - provide helpful information
			conflictInfo := "\n\nMERGE CONFLICT detected!\n"
			conflictInfo += "You need to:\n"
			conflictInfo += "1. Resolve conflicts in the affected files\n"
			conflictInfo += "2. Stage the resolved files with git_add\n"
			conflictInfo += "3. Complete the merge with git_commit\n\n"
			conflictInfo += "Affected files:\n"
			
			// Get conflict status
			statusCmd := exec.Command("git", "status", "--short")
			statusCmd.Dir = path
			var statusOut bytes.Buffer
			statusCmd.Stdout = &statusOut
			if statusCmd.Run() == nil {
				conflictInfo += statusOut.String()
			}
			
			// Merge conflicts are not retryable - they need manual intervention
			return "", NewPermanentError(serr.New(conflictInfo), "merge conflict")
		}
		// Default to retryable for unknown errors as they might be transient
		return "", NewRetryableError(serr.Wrap(err, fmt.Sprintf("Git pull failed: %s", errMsg)), "unknown error")
	}

	// Combine output
	result := stdout.String()
	if stderr.String() != "" {
		result += stderr.String()
	}

	if result == "" {
		result = "Already up to date."
	}

	// Get a summary of what changed
	if !strings.Contains(result, "Already up to date") {
		logCmd := exec.Command("git", "log", "--oneline", "-5")
		logCmd.Dir = path
		
		var logOut bytes.Buffer
		logCmd.Stdout = &logOut
		if logCmd.Run() == nil {
			result += "\n\nRecent commits:\n" + logOut.String()
		}
	}

	return result, nil
}

// GitCheckoutTool implements git checkout functionality
type GitCheckoutTool struct{}

// GetDefinition returns the tool definition for git checkout
func (t *GitCheckoutTool) GetDefinition() Tool {
	return Tool{
		Name:        "git_checkout",
		Description: "Switch branches or restore working tree files",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Repository path (defaults to current directory)",
				},
				"branch": map[string]interface{}{
					"type":        "string",
					"description": "Branch name to checkout",
				},
				"create": map[string]interface{}{
					"type":        "boolean",
					"description": "Create a new branch (-b flag)",
				},
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Specific file to restore from HEAD",
				},
				"force": map[string]interface{}{
					"type":        "boolean",
					"description": "Force checkout, discarding local changes",
				},
				"track": map[string]interface{}{
					"type":        "string",
					"description": "Set up tracking for remote branch",
				},
				"orphan": map[string]interface{}{
					"type":        "boolean",
					"description": "Create new orphan branch",
				},
				"detach": map[string]interface{}{
					"type":        "boolean",
					"description": "Detach HEAD at named commit",
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs git checkout command
func (t *GitCheckoutTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		path = "."
	}

	// Build git command
	args := []string{"checkout"}

	// Check what operation we're doing
	branch, hasBranch := GetString(input, "branch")
	file, hasFile := GetString(input, "file")
	
	if !hasBranch && !hasFile {
		return "", serr.New("Either 'branch' or 'file' parameter is required")
	}

	// Handle options
	if create, ok := input["create"].(bool); ok && create {
		if !hasBranch {
			return "", serr.New("Cannot use 'create' flag without specifying a branch")
		}
		args = append(args, "-b")
	}

	if orphan, ok := input["orphan"].(bool); ok && orphan {
		if !hasBranch {
			return "", serr.New("Cannot use 'orphan' flag without specifying a branch")
		}
		args = []string{"checkout", "--orphan"}
	}

	if force, ok := input["force"].(bool); ok && force {
		args = append(args, "-f")
	}

	if track, ok := GetString(input, "track"); ok && track != "" {
		args = append(args, "--track", track)
	}

	if detach, ok := input["detach"].(bool); ok && detach {
		args = append(args, "--detach")
	}

	// Add branch or file
	if hasBranch && branch != "" {
		args = append(args, branch)
	} else if hasFile && file != "" {
		// When checking out a file, we need to add -- before the filename
		args = append(args, "--", file)
	}

	// First, let's check if there are uncommitted changes
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = path
	
	var statusOut bytes.Buffer
	statusCmd.Stdout = &statusOut
	statusCmd.Run()
	
	hasUncommittedChanges := statusOut.Len() > 0

	// Execute git command
	cmd := exec.Command("git", args...)
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if strings.Contains(errMsg, "not a git repository") {
			return "", NewPermanentError(serr.New(fmt.Sprintf("Not a git repository: %s", path)), "invalid repository")
		}
		if strings.Contains(errMsg, "pathspec") && strings.Contains(errMsg, "did not match") {
			return "", NewPermanentError(serr.New(fmt.Sprintf("Branch or file not found: %s", branch)), "not found")
		}
		if strings.Contains(errMsg, "Your local changes") {
			return "", NewPermanentError(serr.New("Cannot checkout: You have uncommitted changes. Commit, stash, or use 'force' option"), "uncommitted changes")
		}
		if strings.Contains(errMsg, "already exists") {
			return "", serr.New(fmt.Sprintf("Branch '%s' already exists. Use a different name or checkout the existing branch", branch))
		}
		// Checkout errors might be temporary (lock issues)
		return "", WrapFileSystemError(serr.Wrap(err, fmt.Sprintf("Git checkout failed: %s", errMsg)))
	}

	// Build result message
	result := stdout.String()
	if stderr.String() != "" {
		result += stderr.String()
	}

	// Get current branch info
	branchCmd := exec.Command("git", "branch", "--show-current")
	branchCmd.Dir = path
	
	var branchOut bytes.Buffer
	branchCmd.Stdout = &branchOut
	if branchCmd.Run() == nil {
		currentBranch := strings.TrimSpace(branchOut.String())
		if currentBranch != "" {
			result += fmt.Sprintf("\n\nNow on branch: %s", currentBranch)
		} else {
			// Might be in detached HEAD state
			result += "\n\nNow in detached HEAD state"
		}
	}

	// If we switched branches and there were uncommitted changes, warn about it
	if hasBranch && hasUncommittedChanges && !hasFile {
		force, _ := input["force"].(bool)
		if force {
			result += "\n\n⚠️  Warning: Local changes were discarded due to force flag"
		}
	}

	// Show recent commits on the new branch
	if hasBranch {
		logCmd := exec.Command("git", "log", "--oneline", "-5")
		logCmd.Dir = path
		
		var logOut bytes.Buffer
		logCmd.Stdout = &logOut
		if logCmd.Run() == nil {
			result += "\n\nRecent commits on this branch:\n" + logOut.String()
		}
	}

	return result, nil
}

// GitMergeTool implements git merge functionality
type GitMergeTool struct{}

// GetDefinition returns the tool definition for git merge
func (t *GitMergeTool) GetDefinition() Tool {
	return Tool{
		Name:        "git_merge",
		Description: "Join two or more development histories together",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Repository path (defaults to current directory)",
				},
				"branch": map[string]interface{}{
					"type":        "string",
					"description": "Branch name to merge into current branch (required)",
				},
				"no_ff": map[string]interface{}{
					"type":        "boolean",
					"description": "Create a merge commit even for fast-forward",
				},
				"ff_only": map[string]interface{}{
					"type":        "boolean",
					"description": "Refuse to merge unless fast-forward is possible",
				},
				"squash": map[string]interface{}{
					"type":        "boolean",
					"description": "Squash commits into a single commit",
				},
				"strategy": map[string]interface{}{
					"type":        "string",
					"description": "Merge strategy (e.g., 'ours', 'theirs', 'recursive')",
				},
				"message": map[string]interface{}{
					"type":        "string",
					"description": "Set the commit message for the merge",
				},
				"abort": map[string]interface{}{
					"type":        "boolean",
					"description": "Abort the current conflict resolution process",
				},
				"continue": map[string]interface{}{
					"type":        "boolean",
					"description": "Continue after resolving conflicts",
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs git merge command
func (t *GitMergeTool) Execute(input map[string]interface{}) (string, error) {
	path, ok := GetString(input, "path")
	if !ok || path == "" {
		path = "."
	}

	// Check for special operations first
	if abort, ok := input["abort"].(bool); ok && abort {
		cmd := exec.Command("git", "merge", "--abort")
		cmd.Dir = path
		
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		
		if err := cmd.Run(); err != nil {
			errMsg := stderr.String()
			if strings.Contains(errMsg, "not a git repository") {
				return "", serr.New(fmt.Sprintf("Not a git repository: %s", path))
			}
			if strings.Contains(errMsg, "There is no merge to abort") {
				return "", NewPermanentError(serr.New("No merge in progress to abort"), "no merge")
			}
			return "", WrapFileSystemError(serr.Wrap(err, fmt.Sprintf("Failed to abort merge: %s", errMsg)))
		}
		
		return "Merge aborted successfully", nil
	}

	if cont, ok := input["continue"].(bool); ok && cont {
		cmd := exec.Command("git", "merge", "--continue")
		cmd.Dir = path
		
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		
		if err := cmd.Run(); err != nil {
			errMsg := stderr.String()
			if strings.Contains(errMsg, "not a git repository") {
				return "", serr.New(fmt.Sprintf("Not a git repository: %s", path))
			}
			if strings.Contains(errMsg, "There is no merge in progress") {
				return "", NewPermanentError(serr.New("No merge in progress to continue"), "no merge")
			}
			if strings.Contains(errMsg, "Committing is not possible") {
				return "", NewPermanentError(serr.New("Cannot continue merge: conflicts still exist"), "conflicts")
			}
			return "", WrapFileSystemError(serr.Wrap(err, fmt.Sprintf("Failed to continue merge: %s", errMsg)))
		}
		
		result := stdout.String() + stderr.String()
		return result + "\n\nMerge completed successfully", nil
	}

	// For normal merge, branch is required
	branch, hasBranch := GetString(input, "branch")
	if !hasBranch || branch == "" {
		return "", serr.New("Branch parameter is required for merge")
	}

	// Build git command
	args := []string{"merge"}

	// Handle conflicting options
	noFF, hasNoFF := input["no_ff"].(bool)
	ffOnly, hasFFOnly := input["ff_only"].(bool)
	
	if hasNoFF && noFF && hasFFOnly && ffOnly {
		return "", serr.New("Cannot use both 'no_ff' and 'ff_only' options")
	}

	if noFF {
		args = append(args, "--no-ff")
	}
	
	if ffOnly {
		args = append(args, "--ff-only")
	}

	if squash, ok := input["squash"].(bool); ok && squash {
		args = append(args, "--squash")
	}

	if strategy, ok := GetString(input, "strategy"); ok && strategy != "" {
		args = append(args, "--strategy", strategy)
	}

	if message, ok := GetString(input, "message"); ok && message != "" {
		args = append(args, "-m", message)
	}

	// Add the branch to merge
	args = append(args, branch)

	// Check current branch first
	currentBranchCmd := exec.Command("git", "branch", "--show-current")
	currentBranchCmd.Dir = path
	
	var currentBranchOut bytes.Buffer
	currentBranchCmd.Stdout = &currentBranchOut
	currentBranchCmd.Run()
	currentBranch := strings.TrimSpace(currentBranchOut.String())

	// Execute git command
	cmd := exec.Command("git", args...)
	cmd.Dir = path

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		errMsg := stderr.String()
		if strings.Contains(errMsg, "not a git repository") {
			return "", NewPermanentError(serr.New(fmt.Sprintf("Not a git repository: %s", path)), "invalid repository")
		}
		if strings.Contains(errMsg, "not something we can merge") {
			return "", NewPermanentError(serr.New(fmt.Sprintf("Cannot merge: '%s' is not a valid branch or commit", branch)), "invalid branch")
		}
		if strings.Contains(errMsg, "Automatic merge failed") || strings.Contains(errMsg, "CONFLICT") {
			// Merge conflict - provide helpful information
			conflictInfo := fmt.Sprintf("MERGE CONFLICT while merging '%s' into '%s'\n\n", branch, currentBranch)
			conflictInfo += "You need to:\n"
			conflictInfo += "1. Resolve conflicts in the affected files\n"
			conflictInfo += "2. Stage the resolved files with git_add\n"
			conflictInfo += "3. Complete the merge with git_merge --continue\n"
			conflictInfo += "   (or abort with git_merge --abort)\n\n"
			conflictInfo += "Conflicted files:\n"
			
			// Get conflict status
			statusCmd := exec.Command("git", "status", "--short")
			statusCmd.Dir = path
			var statusOut bytes.Buffer
			statusCmd.Stdout = &statusOut
			if statusCmd.Run() == nil {
				conflictInfo += statusOut.String()
			}
			
			return "", NewPermanentError(serr.New(conflictInfo), "merge conflict")
		}
		if strings.Contains(errMsg, "Not possible to fast-forward") {
			return "", NewPermanentError(serr.New("Cannot merge: Fast-forward not possible and ff_only was specified"), "ff not possible")
		}
		// Merge errors might be temporary (lock issues)
		return "", WrapFileSystemError(serr.Wrap(err, fmt.Sprintf("Git merge failed: %s", errMsg)))
	}

	// Build success message
	result := stdout.String()
	if stderr.String() != "" {
		result += stderr.String()
	}

	if result == "" {
		result = fmt.Sprintf("Successfully merged '%s' into '%s'", branch, currentBranch)
	}

	// Show the merge commit
	logCmd := exec.Command("git", "log", "-1", "--oneline")
	logCmd.Dir = path
	
	var logOut bytes.Buffer
	logCmd.Stdout = &logOut
	if logCmd.Run() == nil {
		result += "\n\nMerge commit: " + logOut.String()
	}

	// Show what changed
	diffStatCmd := exec.Command("git", "diff", "--stat", "HEAD~1..HEAD")
	diffStatCmd.Dir = path
	
	var diffStatOut bytes.Buffer
	diffStatCmd.Stdout = &diffStatOut
	if diffStatCmd.Run() == nil && diffStatOut.Len() > 0 {
		result += "\n\nChanges merged:\n" + diffStatOut.String()
	}

	return result, nil
}