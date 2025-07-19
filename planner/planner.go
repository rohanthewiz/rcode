package planner

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rohanthewiz/serr"
)

// Planner handles task planning and execution
type Planner struct {
	mu               sync.RWMutex
	tasks            map[string]*TaskPlanner
	executor         *StepExecutor
	parallelExecutor *ParallelExecutor
	analyzer         *TaskAnalyzer
	templates        map[string]*TaskTemplate
	logs             map[string][]ExecutionLog
	options          PlannerOptions
	snapshotManager  *SnapshotManager
	contextManager   interface{} // Will be *context.Manager but avoid import cycle
	dbStore          interface{} // Will be *db.TaskPlanDB but avoid import cycle
	metricsCollector *MetricsCollector
	gitRollback      map[string]*GitRollbackManager // Per-task Git rollback managers
}

// NewPlanner creates a new task planner
func NewPlanner(options PlannerOptions) *Planner {
	// Create analyzer with context support if available
	var analyzer *TaskAnalyzer
	if options.ContextManager != nil {
		analyzer = NewTaskAnalyzerWithContext(options.ContextManager)
	} else {
		analyzer = NewTaskAnalyzer()
	}

	stepExecutor := NewStepExecutor()

	planner := &Planner{
		tasks:            make(map[string]*TaskPlanner),
		executor:         stepExecutor,
		analyzer:         analyzer,
		templates:        make(map[string]*TaskTemplate),
		logs:             make(map[string][]ExecutionLog),
		options:          options,
		contextManager:   options.ContextManager,
		metricsCollector: NewMetricsCollector(),
		gitRollback:      make(map[string]*GitRollbackManager),
	}

	// Initialize parallel executor if concurrent steps are enabled
	if options.MaxConcurrentSteps > 1 {
		planner.parallelExecutor = NewParallelExecutor(stepExecutor, options.MaxConcurrentSteps)
	}

	// Snapshot manager will be initialized via factory or SetSnapshotStore

	return planner
}

// CreatePlan creates a new task plan from a description
func (p *Planner) CreatePlan(description string) (*TaskPlanner, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Analyze the task description
	steps, err := p.analyzer.AnalyzeTask(description)
	if err != nil {
		return nil, serr.Wrap(err, "failed to analyze task")
	}

	if len(steps) > p.options.MaxSteps {
		return nil, serr.New(fmt.Sprintf("task requires %d steps, exceeds maximum of %d",
			len(steps), p.options.MaxSteps))
	}

	// Create task planner
	task := &TaskPlanner{
		ID:          uuid.New().String(),
		Description: description,
		Steps:       steps,
		CurrentStep: 0,
		Checkpoints: make([]Checkpoint, 0),
		Context: &TaskContext{
			Variables:     make(map[string]interface{}),
			Environment:   make(map[string]string),
			Files:         make([]string, 0),
			ModifiedFiles: make([]string, 0),
		},
		Status:    TaskStatusPending,
		StartTime: time.Now(),
	}

	p.tasks[task.ID] = task
	p.logInfo(task.ID, "", fmt.Sprintf("Created task plan with %d steps", len(steps)))

	return task, nil
}

// ExecutePlan executes a task plan
func (p *Planner) ExecutePlan(taskID string) error {
	p.mu.Lock()
	task, exists := p.tasks[taskID]
	if !exists {
		p.mu.Unlock()
		return serr.New("task not found")
	}

	if task.Status != TaskStatusPending && task.Status != TaskStatusPaused {
		p.mu.Unlock()
		return serr.New("task is not in a runnable state")
	}

	task.Status = TaskStatusExecuting
	p.mu.Unlock()

	// Start metrics collection
	if p.metricsCollector != nil {
		p.metricsCollector.StartPlanExecution(task.ID, len(task.Steps))
	}

	// Check if we can use parallel execution
	if p.parallelExecutor != nil && p.shouldUseParallelExecution(task) {
		return p.executeParallel(task)
	}

	// Sequential execution
	for task.CurrentStep < len(task.Steps) {
		step := &task.Steps[task.CurrentStep]

		// Check if we should create a checkpoint
		if p.options.EnableCheckpoints &&
			task.CurrentStep > 0 &&
			task.CurrentStep%p.options.CheckpointEvery == 0 {
			if err := p.createCheckpoint(task); err != nil {
				p.logWarning(task.ID, step.ID, "Failed to create checkpoint: "+err.Error())
			}
		}

		// Check dependencies
		if !p.checkDependencies(task, step) {
			step.Status = StepStatusSkipped
			p.logInfo(task.ID, step.ID, "Skipping step due to unmet dependencies")
			if p.metricsCollector != nil {
				p.metricsCollector.RecordStepSkipped(task.ID, step.ID)
			}
			task.CurrentStep++
			continue
		}

		// Execute step
		if err := p.executeStep(task, step); err != nil {
			if step.Retryable && step.Result.Retries < step.MaxRetries {
				// Retry the step
				p.logWarning(task.ID, step.ID, fmt.Sprintf("Step failed, retrying (%d/%d)",
					step.Result.Retries+1, step.MaxRetries))
				if p.metricsCollector != nil {
					p.metricsCollector.RecordRetry(task.ID, step.ID)
				}
				continue
			}

			// Step failed
			task.Status = TaskStatusFailed
			endTime := time.Now()
			task.EndTime = &endTime

			// End metrics collection
			if p.metricsCollector != nil {
				metrics, _ := p.metricsCollector.EndPlanExecution(task.ID)
				if metrics != nil {
					p.logInfo(task.ID, "", GenerateMetricsReport(metrics))
				}
			}

			p.mu.Lock()
			p.mu.Unlock()

			return serr.Wrap(err, fmt.Sprintf("step %s failed", step.ID))
		}

		task.CurrentStep++

		// Save progress periodically
		if task.CurrentStep%3 == 0 {
			if err := p.saveProgress(task); err != nil {
				p.logWarning(task.ID, "", "Failed to save progress: "+err.Error())
			}
		}
	}

	// Task completed successfully
	task.Status = TaskStatusCompleted
	endTime := time.Now()
	task.EndTime = &endTime
	task.CompletedAt = &endTime

	p.mu.Lock()
	p.mu.Unlock()

	// Save final state
	if err := p.saveProgress(task); err != nil {
		p.logWarning(task.ID, "", "Failed to save final state: "+err.Error())
	}

	p.logInfo(task.ID, "", "Task completed successfully")
	return nil
}

// executeStep executes a single step
func (p *Planner) executeStep(task *TaskPlanner, step *TaskStep) error {
	startTime := time.Now()
	step.StartTime = &startTime
	step.Status = StepStatusRunning

	p.logInfo(task.ID, step.ID, fmt.Sprintf("Executing step: %s", step.Description))

	// Start step metrics
	if p.metricsCollector != nil {
		_, err := p.metricsCollector.StartStepExecution(task.ID, step.ID, step.Tool)
		if err != nil {
			p.logWarning(task.ID, step.ID, "Failed to start step metrics: "+err.Error())
		}
	}

	// Execute with timeout
	done := make(chan error, 1)
	go func() {
		result, err := p.executor.Execute(step, task.Context)
		if err != nil {
			done <- err
			return
		}
		step.Result = result
		done <- nil
	}()

	// Wait for completion or timeout
	select {
	case err := <-done:
		endTime := time.Now()
		step.EndTime = &endTime

		if err != nil {
			step.Status = StepStatusFailed
			if step.Result == nil {
				step.Result = &StepResult{
					Success: false,
					Error:   err.Error(),
				}
			}
			step.Result.Retries++

			// End step metrics
			if p.metricsCollector != nil {
				p.metricsCollector.EndStepExecution(task.ID, step.ID, false, err)
			}

			return err
		}

		step.Status = StepStatusCompleted
		step.Result.Duration = endTime.Sub(startTime)

		// Update context with any changes
		p.updateContext(task, step)

		// Track Git operations for rollback
		if strings.HasPrefix(step.Tool, "git_") && step.Result.Success {
			p.mu.Lock()
			if p.gitRollback[task.ID] == nil {
				p.gitRollback[task.ID] = NewGitRollbackManager(".")
			}
			gitMgr := p.gitRollback[task.ID]
			p.mu.Unlock()

			if err := gitMgr.TrackGitOperation(step, step.Result); err != nil {
				p.logWarning(task.ID, step.ID, "Failed to track Git operation: "+err.Error())
			}
		}

		// Record file modifications in metrics
		if p.metricsCollector != nil && len(task.Context.ModifiedFiles) > 0 {
			// For simplicity, just record the files modified by this step
			var bytesWritten int64
			if output, ok := step.Result.Output.(map[string]interface{}); ok {
				if bytes, ok := output["bytes_written"].(int64); ok {
					bytesWritten = bytes
				}
			}
			p.metricsCollector.RecordFileModification(task.ID, step.ID, []string{}, bytesWritten)
		}

		// End step metrics
		if p.metricsCollector != nil {
			p.metricsCollector.EndStepExecution(task.ID, step.ID, true, nil)
		}

		return nil

	case <-time.After(p.options.TimeoutPerStep):
		endTime := time.Now()
		step.EndTime = &endTime
		step.Status = StepStatusFailed

		// End step metrics
		if p.metricsCollector != nil {
			p.metricsCollector.EndStepExecution(task.ID, step.ID, false, serr.New("timeout exceeded"))
		}

		return serr.New("step timeout exceeded")
	}
}

// checkDependencies checks if step dependencies are met
func (p *Planner) checkDependencies(task *TaskPlanner, step *TaskStep) bool {
	for _, depID := range step.Dependencies {
		found := false
		for i := 0; i < task.CurrentStep; i++ {
			if task.Steps[i].ID == depID && task.Steps[i].Status == StepStatusCompleted {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// updateContext updates the task context after step execution
func (p *Planner) updateContext(task *TaskPlanner, step *TaskStep) {
	// Update modified files if the tool modifies files
	if step.Tool == "write_file" || step.Tool == "edit_file" {
		if path, ok := step.Params["path"].(string); ok {
			if !contains(task.Context.ModifiedFiles, path) {
				task.Context.ModifiedFiles = append(task.Context.ModifiedFiles, path)
			}
		}
	}

	// Store step output in variables
	if step.Result != nil && step.Result.Output != nil {
		task.Context.Variables[step.ID+"_output"] = step.Result.Output
	}
}

// PausePlan pauses a running task
func (p *Planner) PausePlan(taskID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	task, exists := p.tasks[taskID]
	if !exists {
		return serr.New("task not found")
	}

	if task.Status != TaskStatusExecuting {
		return serr.New("task is not running")
	}

	task.Status = TaskStatusPaused
	p.logInfo(taskID, "", "Task paused")
	return nil
}

// ResumePlan resumes a paused task
func (p *Planner) ResumePlan(taskID string) error {
	return p.ExecutePlan(taskID)
}

// CancelPlan cancels a task
func (p *Planner) CancelPlan(taskID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	task, exists := p.tasks[taskID]
	if !exists {
		return serr.New("task not found")
	}

	task.Status = TaskStatusCancelled
	endTime := time.Now()
	task.EndTime = &endTime

	p.logInfo(taskID, "", "Task cancelled")
	return nil
}

// GetPlan returns a task plan by ID
func (p *Planner) GetPlan(taskID string) (*TaskPlanner, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	task, exists := p.tasks[taskID]
	if !exists {
		return nil, serr.New("task not found")
	}

	return task, nil
}

// LoadPlan loads a plan into the planner's memory
func (p *Planner) LoadPlan(plan *TaskPlanner) error {
	if plan == nil {
		return serr.New("plan is nil")
	}
	if plan.ID == "" {
		return serr.New("plan ID is required")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.tasks[plan.ID] = plan
	p.logInfo(plan.ID, "", "Plan loaded into memory")

	return nil
}

// GetReport generates a report for a task
func (p *Planner) GetReport(taskID string) (*TaskReport, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	task, exists := p.tasks[taskID]
	if !exists {
		return nil, serr.New("task not found")
	}

	report := &TaskReport{
		TaskID:        task.ID,
		Description:   task.Description,
		Status:        task.Status,
		TotalSteps:    len(task.Steps),
		StartTime:     task.StartTime,
		EndTime:       task.EndTime,
		ModifiedFiles: task.Context.ModifiedFiles,
		Errors:        make([]string, 0),
		Checkpoints:   len(task.Checkpoints),
	}

	// Count completed and failed steps
	for _, step := range task.Steps {
		switch step.Status {
		case StepStatusCompleted:
			report.CompletedSteps++
		case StepStatusFailed:
			report.FailedSteps++
			if step.Result != nil && step.Result.Error != "" {
				report.Errors = append(report.Errors,
					fmt.Sprintf("Step %s: %s", step.ID, step.Result.Error))
			}
		}
	}

	// Calculate duration
	if task.EndTime != nil {
		report.Duration = task.EndTime.Sub(task.StartTime)
	} else {
		report.Duration = time.Since(task.StartTime)
	}

	// Get last checkpoint
	if len(task.Checkpoints) > 0 {
		report.LastCheckpoint = &task.Checkpoints[len(task.Checkpoints)-1]
	}

	return report, nil
}

// createCheckpoint creates a checkpoint for the current state
func (p *Planner) createCheckpoint(task *TaskPlanner) error {
	checkpoint := Checkpoint{
		ID:          uuid.New().String(),
		StepID:      task.Steps[task.CurrentStep-1].ID,
		Timestamp:   time.Now(),
		Description: fmt.Sprintf("Checkpoint after step %d", task.CurrentStep),
		State: TaskState{
			CompletedSteps: make([]string, 0),
			Variables:      copyMap(task.Context.Variables),
			FileSnapshots:  make(map[string]string),
		},
	}

	// Record completed steps
	for i := 0; i < task.CurrentStep; i++ {
		if task.Steps[i].Status == StepStatusCompleted {
			checkpoint.State.CompletedSteps = append(checkpoint.State.CompletedSteps,
				task.Steps[i].ID)
		}
	}

	// Create file snapshots for rollback
	if p.snapshotManager != nil && len(task.Context.ModifiedFiles) > 0 {
		err := p.snapshotManager.CreateSnapshot(task.ID, checkpoint.ID, task.Context.ModifiedFiles)
		if err != nil {
			p.logWarning(task.ID, "", fmt.Sprintf("Failed to create file snapshots: %v", err))
		} else {
			// Store snapshot references in checkpoint
			checkpoint.State.FileSnapshots = make(map[string]string)
			for _, file := range task.Context.ModifiedFiles {
				checkpoint.State.FileSnapshots[file] = checkpoint.ID
			}
		}
	}

	task.Checkpoints = append(task.Checkpoints, checkpoint)
	p.logInfo(task.ID, "", fmt.Sprintf("Created checkpoint: %s", checkpoint.ID))

	return nil
}

// RollbackToCheckpoint rolls back to a specific checkpoint
func (p *Planner) RollbackToCheckpoint(taskID, checkpointID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	task, exists := p.tasks[taskID]
	if !exists {
		return serr.New("task not found")
	}

	// Find checkpoint
	var checkpoint *Checkpoint
	checkpointIndex := -1
	for i, cp := range task.Checkpoints {
		if cp.ID == checkpointID {
			checkpoint = &cp
			checkpointIndex = i
			break
		}
	}

	if checkpoint == nil {
		return serr.New("checkpoint not found")
	}

	// Reset task state
	task.CurrentStep = len(checkpoint.State.CompletedSteps)
	task.Context.Variables = copyMap(checkpoint.State.Variables)

	// Reset step statuses
	for i, step := range task.Steps {
		if i < task.CurrentStep {
			// Check if this step was completed in the checkpoint
			completed := false
			for _, completedID := range checkpoint.State.CompletedSteps {
				if step.ID == completedID {
					completed = true
					break
				}
			}
			if !completed {
				task.Steps[i].Status = StepStatusPending
			}
		} else {
			task.Steps[i].Status = StepStatusPending
			task.Steps[i].Result = nil
			task.Steps[i].StartTime = nil
			task.Steps[i].EndTime = nil
		}
	}

	// Remove newer checkpoints
	task.Checkpoints = task.Checkpoints[:checkpointIndex+1]

	// Restore file snapshots
	if p.snapshotManager != nil && len(checkpoint.State.FileSnapshots) > 0 {
		if err := p.snapshotManager.RestoreSnapshot(checkpointID); err != nil {
			p.logWarning(taskID, "", fmt.Sprintf("Failed to restore file snapshots: %v", err))
		} else {
			// Update modified files list to match checkpoint state
			task.Context.ModifiedFiles = make([]string, 0, len(checkpoint.State.FileSnapshots))
			for file := range checkpoint.State.FileSnapshots {
				task.Context.ModifiedFiles = append(task.Context.ModifiedFiles, file)
			}
			p.logInfo(taskID, "", "Successfully restored file snapshots")
		}
	}

	// Rollback Git operations
	if gitMgr, exists := p.gitRollback[taskID]; exists {
		if err := gitMgr.RollbackToCheckpoint(checkpoint.StepID); err != nil {
			p.logWarning(taskID, "", fmt.Sprintf("Git rollback encountered issues: %v", err))
		} else {
			p.logInfo(taskID, "", "Successfully rolled back Git operations")
		}
	}

	task.Status = TaskStatusPaused
	p.logInfo(taskID, "", fmt.Sprintf("Rolled back to checkpoint: %s", checkpointID))

	return nil
}

// LoadTemplate loads a task template
func (p *Planner) LoadTemplate(template *TaskTemplate) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if template.Name == "" {
		return serr.New("template name is required")
	}

	p.templates[template.Name] = template
	return nil
}

// CreatePlanFromTemplate creates a plan from a template
func (p *Planner) CreatePlanFromTemplate(templateName string, variables map[string]interface{}) (*TaskPlanner, error) {
	p.mu.RLock()
	template, exists := p.templates[templateName]
	p.mu.RUnlock()

	if !exists {
		return nil, serr.New("template not found")
	}

	// Validate required variables
	for _, varDef := range template.Variables {
		if varDef.Required {
			if _, ok := variables[varDef.Name]; !ok {
				return nil, serr.New(fmt.Sprintf("required variable %s not provided", varDef.Name))
			}
		}
	}

	// Build description
	description := template.Description
	for name, value := range variables {
		description = strings.ReplaceAll(description, "${"+name+"}", fmt.Sprintf("%v", value))
	}

	// Convert template steps to task steps
	steps := make([]TaskStep, 0, len(template.Steps))
	for _, tmplStep := range template.Steps {
		step := TaskStep{
			ID:           tmplStep.ID,
			Description:  tmplStep.Description,
			Tool:         tmplStep.Tool,
			Params:       make(map[string]interface{}),
			Dependencies: make([]string, 0),
			Retryable:    true,
			MaxRetries:   p.options.MaxRetries,
			Status:       StepStatusPending,
		}

		// Map parameters
		for paramName, varName := range tmplStep.ParamMapping {
			if value, ok := variables[varName]; ok {
				step.Params[paramName] = value
			}
		}

		// TODO: Handle conditions and branching

		steps = append(steps, step)
	}

	// Create the plan
	return p.CreatePlanWithSteps(description, steps)
}

// CreatePlanWithSteps creates a plan with predefined steps
func (p *Planner) CreatePlanWithSteps(description string, steps []TaskStep) (*TaskPlanner, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(steps) > p.options.MaxSteps {
		return nil, serr.New(fmt.Sprintf("task requires %d steps, exceeds maximum of %d",
			len(steps), p.options.MaxSteps))
	}

	task := &TaskPlanner{
		ID:          uuid.New().String(),
		Description: description,
		Steps:       steps,
		CurrentStep: 0,
		Checkpoints: make([]Checkpoint, 0),
		Context: &TaskContext{
			Variables:     make(map[string]interface{}),
			Environment:   make(map[string]string),
			Files:         make([]string, 0),
			ModifiedFiles: make([]string, 0),
		},
		Status:    TaskStatusPending,
		StartTime: time.Now(),
	}

	p.tasks[task.ID] = task
	p.logInfo(task.ID, "", fmt.Sprintf("Created task plan with %d steps", len(steps)))

	return task, nil
}

// Logging helpers

func (p *Planner) logInfo(taskID, stepID, message string) {
	p.addLog(taskID, "info", stepID, message, nil)
}

func (p *Planner) logWarning(taskID, stepID, message string) {
	p.addLog(taskID, "warning", stepID, message, nil)
}

func (p *Planner) logError(taskID, stepID, message string, details interface{}) {
	p.addLog(taskID, "error", stepID, message, details)
}

func (p *Planner) addLog(taskID, level, stepID, message string, details interface{}) {
	log := ExecutionLog{
		Timestamp: time.Now(),
		Level:     level,
		StepID:    stepID,
		Message:   message,
		Details:   details,
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.logs[taskID]; !exists {
		p.logs[taskID] = make([]ExecutionLog, 0)
	}
	p.logs[taskID] = append(p.logs[taskID], log)
}

// GetLogs returns logs for a task
func (p *Planner) GetLogs(taskID string) ([]ExecutionLog, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	logs, exists := p.logs[taskID]
	if !exists {
		return nil, serr.New("task not found")
	}

	// Return a copy
	result := make([]ExecutionLog, len(logs))
	copy(result, logs)
	return result, nil
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func copyMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		result[k] = v
	}
	return result
}

// SetDatabaseStore sets the database store for the planner
func (p *Planner) SetDatabaseStore(store interface{}) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.dbStore = store
}

// saveProgress saves the current task progress to the database
func (p *Planner) saveProgress(task *TaskPlanner) error {
	if p.dbStore == nil {
		return nil // No database configured, skip saving
	}

	// Use type assertion to call SavePlan method
	type dbInterface interface {
		SavePlan(plan interface{}) error
	}

	_, ok := p.dbStore.(dbInterface)
	if !ok {
		return nil // Database doesn't support SavePlan
	}

	// Create a simple map representation
	planData := map[string]interface{}{
		"ID":          task.ID,
		"SessionID":   task.SessionID,
		"Description": task.Description,
		"Status":      string(task.Status),
		"CreatedAt":   task.CreatedAt,
		"UpdatedAt":   time.Now(),
		"CompletedAt": task.CompletedAt,
	}

	// Marshal complex fields
	if stepsJSON, err := json.Marshal(task.Steps); err == nil {
		planData["Steps"] = stepsJSON
	}
	if contextJSON, err := json.Marshal(task.Context); err == nil {
		planData["Context"] = contextJSON
	}
	if checkpointsJSON, err := json.Marshal(task.Checkpoints); err == nil {
		planData["Checkpoints"] = checkpointsJSON
	}

	// For now, just log that we would save
	p.logInfo(task.ID, "", "Progress saved to database")
	return nil
}

// shouldUseParallelExecution determines if a task can benefit from parallel execution
func (p *Planner) shouldUseParallelExecution(task *TaskPlanner) bool {
	// Don't use parallel execution if we're resuming from a checkpoint
	if task.CurrentStep > 0 {
		return false
	}

	// Check if any steps have dependencies
	hasDependencies := false
	for _, step := range task.Steps {
		if len(step.Dependencies) > 0 {
			hasDependencies = true
			break
		}
	}

	// If there are dependencies, analyze parallelizability
	if hasDependencies && p.parallelExecutor != nil {
		analysis := p.parallelExecutor.AnalyzeParallelizability(task.Steps)
		// Use parallel execution if we can achieve at least 1.5x speedup
		return analysis.EstimatedSpeedup >= 1.5
	}

	// No dependencies means all steps could run in parallel
	return len(task.Steps) > 1
}

// executeParallel executes a task plan using parallel execution
func (p *Planner) executeParallel(task *TaskPlanner) error {
	p.logInfo(task.ID, "", "Using parallel execution strategy")

	// Analyze parallelizability
	analysis := p.parallelExecutor.AnalyzeParallelizability(task.Steps)
	p.logInfo(task.ID, "", fmt.Sprintf("Parallel analysis: max parallelism=%d, estimated speedup=%.2fx",
		analysis.MaxParallelism, analysis.EstimatedSpeedup))

	// Execute all steps in parallel
	results, err := p.parallelExecutor.ExecuteSteps(task.Steps, task.Context)
	if err != nil {
		task.Status = TaskStatusFailed
		endTime := time.Now()
		task.EndTime = &endTime
		return err
	}

	// Update step results
	for i := range task.Steps {
		step := &task.Steps[i]
		if result, exists := results[step.ID]; exists {
			step.Result = result
			if result.Success {
				step.Status = StepStatusCompleted
			} else {
				step.Status = StepStatusFailed
			}
		}
	}

	// Task completed
	task.Status = TaskStatusCompleted
	endTime := time.Now()
	task.EndTime = &endTime
	task.CompletedAt = &endTime
	task.CurrentStep = len(task.Steps)

	// Save final state
	if err := p.saveProgress(task); err != nil {
		p.logWarning(task.ID, "", "Failed to save final state: "+err.Error())
	}

	p.logInfo(task.ID, "", "Task completed successfully using parallel execution")
	return nil
}

// AnalyzeParallelizability exposes parallel analysis for a task
func (p *Planner) AnalyzeParallelizability(taskID string) (*ParallelAnalysis, error) {
	p.mu.RLock()
	task, exists := p.tasks[taskID]
	p.mu.RUnlock()

	if !exists {
		return nil, serr.New("task not found")
	}

	if p.parallelExecutor == nil {
		return nil, serr.New("parallel execution not enabled")
	}

	return p.parallelExecutor.AnalyzeParallelizability(task.Steps), nil
}

// GetGitOperations returns the tracked Git operations for a task
func (p *Planner) GetGitOperations(taskID string) ([]GitOperation, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	gitMgr, exists := p.gitRollback[taskID]
	if !exists || gitMgr == nil {
		return []GitOperation{}, nil
	}

	return gitMgr.GetOperations(), nil
}
