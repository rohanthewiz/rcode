package diff

import (
	"strings"
)

// diffAlgorithm implements a line-based diff algorithm for comparing text files.
// This is a simplified implementation suitable for the diff visualization feature.
type diffAlgorithm struct{}

// ComputeLineDiff generates diff hunks between two text strings.
// Uses a line-based approach to identify additions, deletions, and context lines.
func (da *diffAlgorithm) ComputeLineDiff(before, after string, contextLines int) ([]DiffHunk, error) {
	beforeLines := splitLines(before)
	afterLines := splitLines(after)
	
	// Use LCS (Longest Common Subsequence) based approach
	lcs := computeLCS(beforeLines, afterLines)
	
	// Build diff hunks from LCS
	hunks := buildDiffHunks(beforeLines, afterLines, lcs, contextLines)
	
	return hunks, nil
}

// splitLines splits text into lines, preserving empty lines.
func splitLines(text string) []string {
	if text == "" {
		return []string{}
	}
	// Split by newline but preserve the structure
	lines := strings.Split(text, "\n")
	// Remove last empty element if text didn't end with newline
	if len(lines) > 0 && lines[len(lines)-1] == "" && !strings.HasSuffix(text, "\n") {
		lines = lines[:len(lines)-1]
	}
	return lines
}

// lcsEntry represents an entry in the LCS computation matrix.
type lcsEntry struct {
	length int
	prev   string // "diag", "up", "left"
}

// computeLCS computes the Longest Common Subsequence between two line slices.
// Returns a 2D matrix used for backtracking to build the diff.
func computeLCS(before, after []string) [][]lcsEntry {
	m, n := len(before), len(after)
	
	// Initialize LCS matrix
	lcs := make([][]lcsEntry, m+1)
	for i := range lcs {
		lcs[i] = make([]lcsEntry, n+1)
	}
	
	// Fill LCS matrix
	for i := 1; i <= m; i++ {
		for j := 1; j <= n; j++ {
			if before[i-1] == after[j-1] {
				lcs[i][j].length = lcs[i-1][j-1].length + 1
				lcs[i][j].prev = "diag"
			} else if lcs[i-1][j].length >= lcs[i][j-1].length {
				lcs[i][j].length = lcs[i-1][j].length
				lcs[i][j].prev = "up"
			} else {
				lcs[i][j].length = lcs[i][j-1].length
				lcs[i][j].prev = "left"
			}
		}
	}
	
	return lcs
}

// diffOp represents a diff operation during backtracking.
type diffOp struct {
	opType  string // "equal", "delete", "add"
	oldLine int
	newLine int
	content string
}

// buildDiffHunks constructs diff hunks from the LCS matrix.
// Groups consecutive changes and adds context lines.
func buildDiffHunks(before, after []string, lcs [][]lcsEntry, contextLines int) []DiffHunk {
	// Backtrack through LCS to find operations
	ops := backtrackLCS(before, after, lcs)
	
	// Group operations into hunks with context
	hunks := groupIntoHunks(ops, contextLines)
	
	return hunks
}

// backtrackLCS traces through the LCS matrix to identify diff operations.
func backtrackLCS(before, after []string, lcs [][]lcsEntry) []diffOp {
	var ops []diffOp
	i, j := len(before), len(after)
	
	// Backtrack from bottom-right to top-left
	for i > 0 || j > 0 {
		if i == 0 {
			// Add remaining lines from 'after'
			j--
			ops = append([]diffOp{{
				opType:  "add",
				oldLine: i,
				newLine: j + 1,
				content: after[j],
			}}, ops...)
		} else if j == 0 {
			// Delete remaining lines from 'before'
			i--
			ops = append([]diffOp{{
				opType:  "delete",
				oldLine: i + 1,
				newLine: j,
				content: before[i],
			}}, ops...)
		} else if lcs[i][j].prev == "diag" {
			// Lines are equal
			i--
			j--
			ops = append([]diffOp{{
				opType:  "equal",
				oldLine: i + 1,
				newLine: j + 1,
				content: before[i],
			}}, ops...)
		} else if lcs[i][j].prev == "up" {
			// Delete line from 'before'
			i--
			ops = append([]diffOp{{
				opType:  "delete",
				oldLine: i + 1,
				newLine: j,
				content: before[i],
			}}, ops...)
		} else {
			// Add line from 'after'
			j--
			ops = append([]diffOp{{
				opType:  "add",
				oldLine: i,
				newLine: j + 1,
				content: after[j],
			}}, ops...)
		}
	}
	
	return ops
}

// groupIntoHunks groups diff operations into hunks with context lines.
func groupIntoHunks(ops []diffOp, contextLines int) []DiffHunk {
	if len(ops) == 0 {
		return []DiffHunk{}
	}
	
	var hunks []DiffHunk
	var currentHunk *DiffHunk
	lastChangeIdx := -1
	
	for i, op := range ops {
		if op.opType != "equal" {
			// This is a change operation
			if currentHunk == nil || i-lastChangeIdx > contextLines*2 {
				// Start a new hunk
				if currentHunk != nil {
					// Finalize the previous hunk
					hunks = append(hunks, *currentHunk)
				}
				
				// Create new hunk with context before
				currentHunk = &DiffHunk{
					OldStart: op.oldLine,
					NewStart: op.newLine,
					Lines:    []DiffLine{},
				}
				
				// Add context lines before the change
				contextStart := maxInt(0, i-contextLines)
				for j := contextStart; j < i; j++ {
					if ops[j].opType == "equal" {
						currentHunk.Lines = append(currentHunk.Lines, opToDiffLine(ops[j]))
						if j == contextStart {
							currentHunk.OldStart = ops[j].oldLine
							currentHunk.NewStart = ops[j].newLine
						}
					}
				}
			}
			
			// Add the change line
			currentHunk.Lines = append(currentHunk.Lines, opToDiffLine(op))
			lastChangeIdx = i
			
		} else if currentHunk != nil && i-lastChangeIdx <= contextLines {
			// Add context line after a change
			currentHunk.Lines = append(currentHunk.Lines, opToDiffLine(op))
		}
	}
	
	// Finalize the last hunk
	if currentHunk != nil {
		// Calculate line counts
		for _, line := range currentHunk.Lines {
			if line.OldLine != nil {
				currentHunk.OldLines++
			}
			if line.NewLine != nil {
				currentHunk.NewLines++
			}
		}
		hunks = append(hunks, *currentHunk)
	}
	
	return hunks
}

// opToDiffLine converts a diff operation to a DiffLine.
func opToDiffLine(op diffOp) DiffLine {
	line := DiffLine{
		Content: op.content,
	}
	
	switch op.opType {
	case "equal":
		line.Type = "context"
		line.OldLine = &op.oldLine
		line.NewLine = &op.newLine
	case "delete":
		line.Type = "delete"
		line.OldLine = &op.oldLine
	case "add":
		line.Type = "add"
		line.NewLine = &op.newLine
	}
	
	return line
}

// maxInt returns the maximum of two integers.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}