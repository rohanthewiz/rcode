package planner

import (
	"fmt"
	"sync"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// ParallelExecutor handles parallel execution of task steps with dependency management
type ParallelExecutor struct {
	executor   *StepExecutor
	maxWorkers int
	mu         sync.Mutex
}

// NewParallelExecutor creates a new parallel executor
func NewParallelExecutor(executor *StepExecutor, maxWorkers int) *ParallelExecutor {
	if maxWorkers <= 0 {
		maxWorkers = 3 // Default to 3 concurrent workers
	}

	return &ParallelExecutor{
		executor:   executor,
		maxWorkers: maxWorkers,
	}
}

// DependencyGraph represents the dependency relationships between steps
type DependencyGraph struct {
	nodes     map[string]*TaskStep
	edges     map[string][]string // step ID -> dependent step IDs
	inDegree  map[string]int      // number of unresolved dependencies
	completed map[string]bool
	mu        sync.RWMutex
}

// ExecuteSteps executes multiple steps in parallel while respecting dependencies
func (pe *ParallelExecutor) ExecuteSteps(steps []TaskStep, context *TaskContext) (map[string]*StepResult, error) {
	if len(steps) == 0 {
		return make(map[string]*StepResult), nil
	}

	// Build dependency graph
	graph := pe.buildDependencyGraph(steps)

	// Channel to track results
	results := make(map[string]*StepResult)
	resultsMu := sync.Mutex{}

	// Channel for errors
	errChan := make(chan error, 1)

	// Worker pool
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, pe.maxWorkers)

	// Channel to signal when new steps might be ready
	checkReady := make(chan struct{}, len(steps))

	// Start with initial ready steps
	readySteps := pe.findReadySteps(graph)
	if len(readySteps) == 0 && len(steps) > 0 {
		return nil, serr.New("no steps are ready to execute - check for circular dependencies")
	}

	// Process steps until all are completed or an error occurs
	for {
		select {
		case err := <-errChan:
			// Wait for running workers to finish
			wg.Wait()
			return results, err

		default:
			// Get ready steps
			graph.mu.RLock()
			allCompleted := len(graph.completed) == len(steps)
			graph.mu.RUnlock()

			if allCompleted {
				wg.Wait()
				return results, nil
			}

			// Execute ready steps
			for _, step := range readySteps {
				wg.Add(1)

				go func(s TaskStep) {
					defer wg.Done()

					// Acquire semaphore
					semaphore <- struct{}{}
					defer func() { <-semaphore }()

					logger.Info("Executing step in parallel",
						"step_id", s.ID,
						"description", s.Description,
						"tool", s.Tool)

					// Execute the step
					result, err := pe.executor.Execute(&s, context)
					if err != nil {
						result = &StepResult{
							Success: false,
							Error:   err.Error(),
						}
						// Don't block on error channel
						select {
						case errChan <- serr.Wrap(err, fmt.Sprintf("step %s failed", s.ID)):
						default:
						}
					}

					// Store result
					resultsMu.Lock()
					results[s.ID] = result
					resultsMu.Unlock()

					// Mark as completed
					pe.markCompleted(graph, s.ID)

					// Signal that new steps might be ready
					select {
					case checkReady <- struct{}{}:
					default:
					}
				}(step)
			}

			// Wait for signal or check periodically
			if len(readySteps) == 0 {
				select {
				case <-checkReady:
					// A step completed, check for newly ready steps
				}
			}

			// Find newly ready steps
			readySteps = pe.findReadySteps(graph)
		}
	}
}

// buildDependencyGraph creates a dependency graph from the steps
func (pe *ParallelExecutor) buildDependencyGraph(steps []TaskStep) *DependencyGraph {
	graph := &DependencyGraph{
		nodes:     make(map[string]*TaskStep),
		edges:     make(map[string][]string),
		inDegree:  make(map[string]int),
		completed: make(map[string]bool),
	}

	// Add all nodes
	for i := range steps {
		step := &steps[i]
		graph.nodes[step.ID] = step
		graph.inDegree[step.ID] = len(step.Dependencies)
		graph.edges[step.ID] = []string{}
	}

	// Build edges (reverse of dependencies for easier traversal)
	for _, step := range steps {
		for _, depID := range step.Dependencies {
			if _, exists := graph.nodes[depID]; exists {
				graph.edges[depID] = append(graph.edges[depID], step.ID)
			}
		}
	}

	return graph
}

// findReadySteps finds all steps that are ready to execute
func (pe *ParallelExecutor) findReadySteps(graph *DependencyGraph) []TaskStep {
	graph.mu.RLock()
	defer graph.mu.RUnlock()

	var ready []TaskStep

	for id, step := range graph.nodes {
		// Skip if already completed
		if graph.completed[id] {
			continue
		}

		// Check if all dependencies are satisfied
		if graph.inDegree[id] == 0 {
			ready = append(ready, *step)
		}
	}

	return ready
}

// markCompleted marks a step as completed and updates dependent steps
func (pe *ParallelExecutor) markCompleted(graph *DependencyGraph, stepID string) {
	graph.mu.Lock()
	defer graph.mu.Unlock()

	// Mark as completed
	graph.completed[stepID] = true

	// Update dependent steps
	for _, dependentID := range graph.edges[stepID] {
		graph.inDegree[dependentID]--

		if graph.inDegree[dependentID] < 0 {
			// This shouldn't happen, but log it
			logger.LogErr(nil, "negative in-degree detected",
				"step_id", dependentID,
				"dependency", stepID)
			graph.inDegree[dependentID] = 0
		}
	}
}

// AnalyzeParallelizability analyzes steps to determine parallelization opportunities
func (pe *ParallelExecutor) AnalyzeParallelizability(steps []TaskStep) *ParallelAnalysis {
	graph := pe.buildDependencyGraph(steps)

	analysis := &ParallelAnalysis{
		TotalSteps:     len(steps),
		MaxParallelism: 0,
		CriticalPath:   []string{},
		ParallelGroups: [][]string{},
	}

	// Find maximum parallelism by simulating execution
	remainingSteps := len(steps)
	simulatedCompleted := make(map[string]bool)

	for remainingSteps > 0 {
		// Find steps that would be ready
		var readyGroup []string

		for id, step := range graph.nodes {
			if simulatedCompleted[id] {
				continue
			}

			// Check if dependencies are satisfied
			ready := true
			for _, depID := range step.Dependencies {
				if !simulatedCompleted[depID] {
					ready = false
					break
				}
			}

			if ready {
				readyGroup = append(readyGroup, id)
			}
		}

		if len(readyGroup) == 0 && remainingSteps > 0 {
			// Circular dependency detected
			break
		}

		// Update max parallelism
		if len(readyGroup) > analysis.MaxParallelism {
			analysis.MaxParallelism = len(readyGroup)
		}

		// Add to parallel groups
		analysis.ParallelGroups = append(analysis.ParallelGroups, readyGroup)

		// Mark as completed
		for _, id := range readyGroup {
			simulatedCompleted[id] = true
			remainingSteps--
		}
	}

	// Calculate critical path (longest dependency chain)
	analysis.CriticalPath = pe.findCriticalPath(graph)
	analysis.EstimatedSpeedup = float64(len(steps)) / float64(len(analysis.CriticalPath))

	return analysis
}

// findCriticalPath finds the longest dependency chain
func (pe *ParallelExecutor) findCriticalPath(graph *DependencyGraph) []string {
	memo := make(map[string][]string)

	var dfs func(nodeID string) []string
	dfs = func(nodeID string) []string {
		if path, exists := memo[nodeID]; exists {
			return path
		}

		node := graph.nodes[nodeID]
		if len(node.Dependencies) == 0 {
			memo[nodeID] = []string{nodeID}
			return memo[nodeID]
		}

		var longestPath []string
		for _, depID := range node.Dependencies {
			if _, exists := graph.nodes[depID]; exists {
				path := dfs(depID)
				if len(path) > len(longestPath) {
					longestPath = path
				}
			}
		}

		// Append current node to the longest path from dependencies
		result := make([]string, len(longestPath)+1)
		copy(result, longestPath)
		result[len(longestPath)] = nodeID

		memo[nodeID] = result
		return result
	}

	var criticalPath []string
	for id := range graph.nodes {
		path := dfs(id)
		if len(path) > len(criticalPath) {
			criticalPath = path
		}
	}

	return criticalPath
}

// ParallelAnalysis contains analysis results for parallel execution
type ParallelAnalysis struct {
	TotalSteps       int        `json:"total_steps"`
	MaxParallelism   int        `json:"max_parallelism"`
	CriticalPath     []string   `json:"critical_path"`
	ParallelGroups   [][]string `json:"parallel_groups"`
	EstimatedSpeedup float64    `json:"estimated_speedup"`
}
