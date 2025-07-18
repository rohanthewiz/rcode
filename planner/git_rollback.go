package planner

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rohanthewiz/serr"
)

// GitOperation represents a Git operation that can be rolled back
type GitOperation struct {
	Type        string                 `json:"type"`         // commit, push, merge, checkout, branch
	CommitHash  string                 `json:"commit_hash"`  // For commit operations
	Branch      string                 `json:"branch"`       // Branch affected
	PrevBranch  string                 `json:"prev_branch"`  // Previous branch (for checkout)
	RemoteName  string                 `json:"remote_name"`  // Remote name (for push)
	MergeCommit string                 `json:"merge_commit"` // For merge operations
	Timestamp   time.Time              `json:"timestamp"`
	StepID      string                 `json:"step_id"`
	Params      map[string]interface{} `json:"params"` // Original parameters
}

// GitRollbackManager handles rollback of Git operations
type GitRollbackManager struct {
	operations []GitOperation
	workDir    string
}

// NewGitRollbackManager creates a new Git rollback manager
func NewGitRollbackManager(workDir string) *GitRollbackManager {
	return &GitRollbackManager{
		operations: make([]GitOperation, 0),
		workDir:    workDir,
	}
}

// TrackGitOperation records a Git operation for potential rollback
func (grm *GitRollbackManager) TrackGitOperation(step *TaskStep, result *StepResult) error {
	if step == nil || result == nil || !result.Success {
		return nil // Only track successful operations
	}

	// Extract Git operation type from tool name
	if !strings.HasPrefix(step.Tool, "git_") {
		return nil // Not a Git operation
	}

	operation := GitOperation{
		Type:      step.Tool,
		StepID:    step.ID,
		Timestamp: time.Now(),
		Params:    step.Params,
	}

	// Extract operation-specific details from result
	switch step.Tool {
	case "git_commit":
		// Extract commit hash from result
		if commitHash := extractCommitHash(result.Output); commitHash != "" {
			operation.CommitHash = commitHash
		}
		operation.Branch = getCurrentBranch(grm.workDir)

	case "git_push":
		operation.Branch = getParamString(step.Params, "branch", getCurrentBranch(grm.workDir))
		operation.RemoteName = getParamString(step.Params, "remote", "origin")
		// Track the commit that was pushed
		operation.CommitHash = getLatestCommit(grm.workDir)

	case "git_merge":
		operation.Branch = getCurrentBranch(grm.workDir)
		operation.MergeCommit = getLatestCommit(grm.workDir)
		// Store the branch that was merged
		if mergedBranch, ok := step.Params["branch"].(string); ok {
			operation.PrevBranch = mergedBranch
		}

	case "git_checkout":
		// Track branch switch
		if branch, ok := step.Params["branch"].(string); ok {
			operation.Branch = branch
			operation.PrevBranch = getPreviousBranch(result.Output)
		}

	case "git_branch":
		// Track branch creation
		if createBranch, ok := step.Params["create"].(string); ok {
			operation.Branch = createBranch
		}
	}

	grm.operations = append(grm.operations, operation)
	return nil
}

// RollbackToCheckpoint rolls back Git operations to a specific checkpoint
func (grm *GitRollbackManager) RollbackToCheckpoint(checkpointStepID string) error {
	// Find all operations after the checkpoint
	rollbackOps := make([]GitOperation, 0)
	for i := len(grm.operations) - 1; i >= 0; i-- {
		op := grm.operations[i]
		rollbackOps = append(rollbackOps, op)
		if op.StepID == checkpointStepID {
			break
		}
	}

	// Execute rollback in reverse order
	for _, op := range rollbackOps {
		if err := grm.rollbackOperation(op); err != nil {
			return serr.Wrap(err, fmt.Sprintf("failed to rollback %s operation", op.Type))
		}
	}

	// Remove rolled back operations from tracking
	grm.operations = grm.operations[:len(grm.operations)-len(rollbackOps)]

	return nil
}

// rollbackOperation rolls back a specific Git operation
func (grm *GitRollbackManager) rollbackOperation(op GitOperation) error {
	switch op.Type {
	case "git_commit":
		return grm.rollbackCommit(op)
	case "git_push":
		return grm.rollbackPush(op)
	case "git_merge":
		return grm.rollbackMerge(op)
	case "git_checkout":
		return grm.rollbackCheckout(op)
	case "git_branch":
		return grm.rollbackBranch(op)
	default:
		// Other Git operations don't need rollback or are read-only
		return nil
	}
}

// rollbackCommit reverts a commit operation
func (grm *GitRollbackManager) rollbackCommit(op GitOperation) error {
	if op.CommitHash == "" {
		return serr.New("no commit hash recorded for rollback")
	}

	// Check if this commit was already pushed
	if grm.wasCommitPushed(op.CommitHash) {
		// Create a revert commit instead of resetting
		cmd := exec.Command("git", "revert", "--no-edit", op.CommitHash)
		cmd.Dir = grm.workDir
		if err := cmd.Run(); err != nil {
			return serr.Wrap(err, "failed to revert commit")
		}
	} else {
		// Safe to reset if not pushed
		cmd := exec.Command("git", "reset", "--hard", "HEAD~1")
		cmd.Dir = grm.workDir
		if err := cmd.Run(); err != nil {
			return serr.Wrap(err, "failed to reset commit")
		}
	}

	return nil
}

// rollbackPush attempts to undo a push operation (if possible)
func (grm *GitRollbackManager) rollbackPush(op GitOperation) error {
	// WARNING: This is a destructive operation and should be used with caution
	// Only attempt if we're certain no one else has pulled the changes

	// For safety, we'll just log a warning instead of force pushing
	return serr.New(fmt.Sprintf(
		"Cannot automatically rollback push to %s/%s. Manual intervention required:\n"+
			"If safe to do so, you can force push the previous state with:\n"+
			"git push --force-with-lease %s %s~1:%s",
		op.RemoteName, op.Branch, op.RemoteName, op.Branch, op.Branch))
}

// rollbackMerge reverts a merge operation
func (grm *GitRollbackManager) rollbackMerge(op GitOperation) error {
	if op.MergeCommit == "" {
		return serr.New("no merge commit recorded for rollback")
	}

	// Check if the merge was already pushed
	if grm.wasCommitPushed(op.MergeCommit) {
		// Create a revert commit for the merge
		// -m 1 means we're reverting to the first parent (the branch we merged into)
		cmd := exec.Command("git", "revert", "-m", "1", "--no-edit", op.MergeCommit)
		cmd.Dir = grm.workDir
		if err := cmd.Run(); err != nil {
			return serr.Wrap(err, "failed to revert merge commit")
		}
	} else {
		// Safe to reset if not pushed
		cmd := exec.Command("git", "reset", "--hard", "HEAD~1")
		cmd.Dir = grm.workDir
		if err := cmd.Run(); err != nil {
			return serr.Wrap(err, "failed to reset merge")
		}
	}

	return nil
}

// rollbackCheckout switches back to the previous branch
func (grm *GitRollbackManager) rollbackCheckout(op GitOperation) error {
	if op.PrevBranch == "" {
		return nil // No previous branch recorded
	}

	cmd := exec.Command("git", "checkout", op.PrevBranch)
	cmd.Dir = grm.workDir
	if err := cmd.Run(); err != nil {
		return serr.Wrap(err, "failed to checkout previous branch")
	}

	return nil
}

// rollbackBranch deletes a created branch (if safe)
func (grm *GitRollbackManager) rollbackBranch(op GitOperation) error {
	if op.Branch == "" {
		return nil
	}

	// Check if we're currently on this branch
	currentBranch := getCurrentBranch(grm.workDir)
	if currentBranch == op.Branch {
		// Switch to main/master first
		mainBranch := getMainBranch(grm.workDir)
		cmd := exec.Command("git", "checkout", mainBranch)
		cmd.Dir = grm.workDir
		if err := cmd.Run(); err != nil {
			return serr.Wrap(err, "failed to switch branch before deletion")
		}
	}

	// Delete the branch (use -D to force delete even if not merged)
	cmd := exec.Command("git", "branch", "-D", op.Branch)
	cmd.Dir = grm.workDir
	if err := cmd.Run(); err != nil {
		// Branch might have been pushed, just log warning
		return serr.New(fmt.Sprintf("could not delete branch %s - it may have been pushed to remote", op.Branch))
	}

	return nil
}

// Helper functions

func (grm *GitRollbackManager) wasCommitPushed(commitHash string) bool {
	// Check if commit exists on any remote
	cmd := exec.Command("git", "branch", "-r", "--contains", commitHash)
	cmd.Dir = grm.workDir
	output, err := cmd.Output()
	return err == nil && len(output) > 0
}

func getCurrentBranch(workDir string) string {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func getLatestCommit(workDir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = workDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func getMainBranch(workDir string) string {
	// Try to detect main branch name (could be main or master)
	for _, branch := range []string{"main", "master"} {
		cmd := exec.Command("git", "rev-parse", "--verify", branch)
		cmd.Dir = workDir
		if err := cmd.Run(); err == nil {
			return branch
		}
	}
	return "main" // Default
}

func extractCommitHash(output interface{}) string {
	// Extract commit hash from git commit output
	if str, ok := output.(string); ok {
		// Look for pattern like [branch-name abc1234]
		if idx := strings.Index(str, "["); idx >= 0 {
			if endIdx := strings.Index(str[idx:], "]"); endIdx > 0 {
				parts := strings.Fields(str[idx+1 : idx+endIdx])
				if len(parts) >= 2 {
					return parts[1]
				}
			}
		}
		// Also try to extract from "commit <hash>" pattern
		lines := strings.Split(str, "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "commit ") {
				return strings.TrimSpace(strings.TrimPrefix(line, "commit "))
			}
		}
	}
	return ""
}

func getPreviousBranch(output interface{}) string {
	// Try to extract previous branch from checkout output
	if str, ok := output.(string); ok {
		// Look for "Switched from branch 'xxx'"
		if idx := strings.Index(str, "from '"); idx >= 0 {
			start := idx + 6
			if endIdx := strings.Index(str[start:], "'"); endIdx > 0 {
				return str[start : start+endIdx]
			}
		}
	}
	return ""
}

func getParamString(params map[string]interface{}, key, defaultValue string) string {
	if val, ok := params[key].(string); ok {
		return val
	}
	return defaultValue
}

// GetOperations returns the tracked Git operations
func (grm *GitRollbackManager) GetOperations() []GitOperation {
	return grm.operations
}

// Clear removes all tracked operations
func (grm *GitRollbackManager) Clear() {
	grm.operations = make([]GitOperation, 0)
}
