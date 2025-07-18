# Phase 3 Implementation Plan: Agent Enhancement (Points 1-3)

## Overview
This document contains the detailed implementation plan for the first three points of Phase 3 from the rcode enhancement plan:
1. Task Planning System - Break down complex requests ✅
2. Multi-step Execution - Checkpoint-based operations ✅
3. Rollback Capabilities - Undo/redo support ✅

The planner package already exists with basic structure at `/planner/`, but needs significant enhancement and integration with the rest of the system.

## Implementation Status (Updated)

### ✅ Completed
1. **Database Migration** - Created migration #4 with all task planning tables
2. **Snapshot Manager** - Implemented content-addressed file snapshot system
3. **Task Persistence** - Created db/tasks.go with full CRUD operations
4. **API Endpoints** - Implemented all planning endpoints in web/planning.go
5. **Snapshot Integration** - Added file snapshot creation/restoration in planner
6. **Factory Pattern** - Created PlannerFactory to properly initialize dependencies
7. **Context-Aware Analysis** - Enhanced analyzer to use context intelligence

### 🚧 In Progress
1. **Parallel Execution** - Basic structure exists, needs implementation
2. **Execution Metrics** - Database tables created, collection not implemented
3. **Git Rollback** - Placeholder exists, needs proper implementation

### 📋 Todo
1. **Testing** - Comprehensive test coverage for all new features
2. **Documentation** - Update user-facing docs for planning features
3. **UI Enhancements** - Add planning UI to web interface

## Current State Analysis

### Existing Planner Structure
- **types.go**: Well-defined data structures for tasks, steps, checkpoints, and execution state
- **planner.go**: Basic task management with checkpoint creation (but no actual rollback implementation)
- **analyzer.go**: Simple pattern-based task analysis with basic keyword matching
- **executor.go**: Step execution with tool registry integration

### Key Gaps Identified
1. No persistence - plans are only in memory
2. No integration with the main session/message flow
3. Rollback is partially implemented (no file restoration)
4. No parallel execution support
5. No context intelligence integration
6. No real-time progress updates via SSE

## 1. Task Planning System Enhancement

### 1.1 Enhance Task Analysis (`planner/analyzer.go`)

#### Add NLP-based Task Understanding
```go
// Import context package for NLP capabilities
import (
    "rcode/context"
)

// Enhance TaskAnalyzer struct
type TaskAnalyzer struct {
    patterns      []TaskPattern
    contextMgr    *context.Manager
    prioritizer   *context.FilePrioritizer
}

// Enhanced AnalyzeTask method
func (a *TaskAnalyzer) AnalyzeTask(description string) ([]TaskStep, error) {
    // Use context intelligence for better understanding
    keywords := a.prioritizer.ExtractKeywords(description)
    
    // Get relevant files for the task
    taskCtx := &context.TaskContext{
        Task:        description,
        MaxFiles:    20,
        SearchTerms: keywords,
        FileScores:  make(map[string]float64),
    }
    
    relevantFiles, _ := a.prioritizer.Prioritize(a.contextMgr.GetContext(), taskCtx)
    
    // Use file context to enhance step generation
    // ...
}
```

#### Improve Pattern Matching
```go
// Add more sophisticated patterns
var advancedPatterns = []TaskPattern{
    {
        Keywords:    []string{"refactor", "extract", "method", "function"},
        ToolChain:   []string{"search", "read_file", "analyze_code", "edit_file", "test"},
        Description: "Code refactoring with extraction",
        Conditions:  []PatternCondition{
            {Type: "has_tests", Required: true},
        },
    },
    {
        Keywords:    []string{"add", "endpoint", "api", "route"},
        ToolChain:   []string{"read_file", "write_file", "edit_file", "test", "git_add"},
        Description: "API endpoint creation",
        Parallel:    [][]string{{"write_handler", "write_test"}}, // Parallel steps
    },
}
```

### 1.2 Add Planning Persistence

#### Database Schema (`db/migrations.go`)
```sql
-- Migration 006: Add task planning tables
CREATE TABLE IF NOT EXISTS task_plans (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    description TEXT NOT NULL,
    status TEXT NOT NULL,
    steps JSON NOT NULL,
    context JSON,
    checkpoints JSON,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS task_executions (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL,
    step_id TEXT NOT NULL,
    status TEXT NOT NULL,
    result JSON,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    retries INTEGER DEFAULT 0,
    FOREIGN KEY (plan_id) REFERENCES task_plans(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS file_snapshots (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL,
    checkpoint_id TEXT,
    file_path TEXT NOT NULL,
    content TEXT NOT NULL,
    hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (plan_id) REFERENCES task_plans(id) ON DELETE CASCADE
);

CREATE INDEX idx_task_plans_session ON task_plans(session_id);
CREATE INDEX idx_task_executions_plan ON task_executions(plan_id);
CREATE INDEX idx_file_snapshots_plan ON file_snapshots(plan_id);
```

#### Database Methods (`db/tasks.go` - new file)
```go
package db

// TaskPlanDB handles task plan persistence
type TaskPlanDB struct {
    db *sql.DB
}

// SavePlan saves a task plan to the database
func (t *TaskPlanDB) SavePlan(plan *planner.TaskPlanner) error {
    stepsJSON, _ := json.Marshal(plan.Steps)
    contextJSON, _ := json.Marshal(plan.Context)
    checkpointsJSON, _ := json.Marshal(plan.Checkpoints)
    
    _, err := t.db.Exec(`
        INSERT INTO task_plans (id, session_id, description, status, steps, context, checkpoints)
        VALUES (?, ?, ?, ?, ?, ?, ?)
        ON CONFLICT(id) DO UPDATE SET
            status = excluded.status,
            steps = excluded.steps,
            context = excluded.context,
            checkpoints = excluded.checkpoints,
            updated_at = CURRENT_TIMESTAMP
    `, plan.ID, plan.SessionID, plan.Description, plan.Status, 
       string(stepsJSON), string(contextJSON), string(checkpointsJSON))
    
    return err
}

// SaveExecution saves step execution result
func (t *TaskPlanDB) SaveExecution(planID, stepID string, result *planner.StepResult) error {
    resultJSON, _ := json.Marshal(result)
    
    _, err := t.db.Exec(`
        INSERT INTO task_executions (id, plan_id, step_id, status, result, duration_ms, retries)
        VALUES (?, ?, ?, ?, ?, ?, ?)
    `, uuid.New().String(), planID, stepID, result.Status, 
       string(resultJSON), result.Duration.Milliseconds(), result.Retries)
    
    return err
}
```

### 1.3 Integrate with Session

#### API Endpoints (`web/routes.go`)
```go
// Add task planning routes
s.Post("/api/session/:id/plan", createPlanHandler)
s.Get("/api/session/:id/plans", listPlansHandler)
s.Post("/api/plan/:id/execute", executePlanHandler)
s.Get("/api/plan/:id/status", getPlanStatusHandler)
s.Post("/api/plan/:id/rollback", rollbackPlanHandler)
s.Get("/api/plan/:id/checkpoints", listCheckpointsHandler)
```

#### Plan Creation Handler (`web/planning.go` - new file)
```go
package web

type CreatePlanRequest struct {
    Description string `json:"description"`
    AutoExecute bool   `json:"auto_execute"`
}

func createPlanHandler(c rweb.Context) error {
    sessionID := c.Request().Param("id")
    
    var req CreatePlanRequest
    if err := json.Unmarshal(c.Request().Body(), &req); err != nil {
        return c.WriteError(err, 400)
    }
    
    // Get planner instance (singleton or per-session)
    planner := getPlanner()
    
    // Create plan with context
    plan, err := planner.CreatePlan(req.Description)
    if err != nil {
        return c.WriteError(err, 500)
    }
    
    // Associate with session
    plan.SessionID = sessionID
    
    // Save to database
    if err := database.SavePlan(plan); err != nil {
        return c.WriteError(err, 500)
    }
    
    // Broadcast plan creation
    BroadcastPlanCreated(sessionID, plan)
    
    // Auto-execute if requested
    if req.AutoExecute {
        go func() {
            if err := planner.ExecutePlan(plan.ID); err != nil {
                logger.LogErr(err, "auto-execution failed")
            }
        }()
    }
    
    return c.WriteJSON(plan)
}
```

## 2. Multi-step Execution Enhancement

### 2.1 Enhance StepExecutor

#### Context-Aware Execution (`planner/executor.go`)
```go
type StepExecutor struct {
    toolRegistry    *tools.Registry
    contextManager  *context.Manager
    contextExecutor *tools.ContextAwareExecutor
    snapshotMgr     *SnapshotManager
}

func NewStepExecutor(contextMgr *context.Manager) *StepExecutor {
    registry := tools.DefaultRegistry()
    return &StepExecutor{
        toolRegistry:    registry,
        contextManager:  contextMgr,
        contextExecutor: tools.NewContextAwareExecutor(registry, contextMgr),
        snapshotMgr:     NewSnapshotManager(),
    }
}

func (e *StepExecutor) Execute(step *TaskStep, taskContext *TaskContext) (*StepResult, error) {
    // Take snapshots before file-modifying operations
    if e.shouldSnapshot(step.Tool) {
        if err := e.snapshotMgr.CreateSnapshot(step.ID, taskContext); err != nil {
            logger.LogErr(err, "failed to create snapshot")
        }
    }
    
    // Use context-aware executor
    toolUse := tools.ToolUse{
        Type:  "tool_use",
        ID:    step.ID,
        Name:  step.Tool,
        Input: e.prepareParams(step.Params, taskContext),
    }
    
    toolResult, err := e.contextExecutor.Execute(toolUse)
    
    // Track changes in context
    e.updateTaskContext(taskContext, step, toolResult)
    
    return e.createStepResult(toolResult, err), nil
}
```

#### Parallel Execution Support (`planner/parallel_executor.go` - new file)
```go
package planner

type ParallelExecutor struct {
    executor *StepExecutor
    maxWorkers int
}

func (pe *ParallelExecutor) ExecuteSteps(steps []TaskStep, context *TaskContext) map[string]*StepResult {
    // Build dependency graph
    graph := pe.buildDependencyGraph(steps)
    
    // Find steps that can run in parallel
    readySteps := pe.findReadySteps(graph)
    
    // Execute in parallel with worker pool
    results := make(map[string]*StepResult)
    var wg sync.WaitGroup
    semaphore := make(chan struct{}, pe.maxWorkers)
    
    for len(readySteps) > 0 {
        for _, step := range readySteps {
            wg.Add(1)
            semaphore <- struct{}{}
            
            go func(s TaskStep) {
                defer wg.Done()
                defer func() { <-semaphore }()
                
                result, err := pe.executor.Execute(&s, context)
                results[s.ID] = result
                
                // Update graph and find newly ready steps
                pe.markCompleted(graph, s.ID)
            }(step)
        }
        
        wg.Wait()
        readySteps = pe.findReadySteps(graph)
    }
    
    return results
}
```

### 2.2 Add Progress Tracking

#### SSE Events (`web/sse.go`)
```go
// New event types for planning
type PlanEvent struct {
    Type      string      `json:"type"`
    SessionID string      `json:"session_id"`
    PlanID    string      `json:"plan_id"`
    Data      interface{} `json:"data"`
}

func BroadcastPlanCreated(sessionID string, plan *planner.TaskPlanner) {
    event := PlanEvent{
        Type:      "plan_created",
        SessionID: sessionID,
        PlanID:    plan.ID,
        Data: map[string]interface{}{
            "description": plan.Description,
            "steps":       len(plan.Steps),
            "status":      plan.Status,
        },
    }
    broadcast(event)
}

func BroadcastStepProgress(sessionID, planID, stepID string, status StepStatus) {
    event := PlanEvent{
        Type:      "step_progress",
        SessionID: sessionID,
        PlanID:    planID,
        Data: map[string]interface{}{
            "step_id": stepID,
            "status":  status,
        },
    }
    broadcast(event)
}
```

#### Execution Metrics (`planner/metrics.go` - new file)
```go
package planner

type ExecutionMetrics struct {
    PlanID         string
    TotalSteps     int
    CompletedSteps int
    FailedSteps    int
    TotalDuration  time.Duration
    StepMetrics    map[string]*StepMetric
}

type StepMetric struct {
    StepID       string
    Tool         string
    Duration     time.Duration
    RetryCount   int
    MemoryUsage  int64
    Success      bool
}

func (p *Planner) CollectMetrics(planID string) (*ExecutionMetrics, error) {
    // Aggregate metrics from executions
    // Store in database for analysis
}
```

## 3. Rollback Capabilities

### 3.1 Implement File Snapshots

#### Snapshot Manager (`planner/snapshots.go` - new file)
```go
package planner

import (
    "crypto/sha256"
    "encoding/hex"
    "io/ioutil"
    "os"
    "path/filepath"
)

type SnapshotManager struct {
    baseDir string
    db      *db.TaskPlanDB
}

func NewSnapshotManager() *SnapshotManager {
    homeDir, _ := os.UserHomeDir()
    baseDir := filepath.Join(homeDir, ".local", "share", "rcode", "snapshots")
    os.MkdirAll(baseDir, 0755)
    
    return &SnapshotManager{
        baseDir: baseDir,
        db:      db.GetTaskPlanDB(),
    }
}

func (sm *SnapshotManager) CreateSnapshot(planID, checkpointID string, files []string) error {
    for _, file := range files {
        content, err := ioutil.ReadFile(file)
        if err != nil {
            continue // File might not exist yet
        }
        
        // Calculate content hash
        hash := sha256.Sum256(content)
        hashStr := hex.EncodeToString(hash[:])
        
        // Store content using content-addressed storage
        snapPath := filepath.Join(sm.baseDir, hashStr[:2], hashStr)
        os.MkdirAll(filepath.Dir(snapPath), 0755)
        
        if _, err := os.Stat(snapPath); os.IsNotExist(err) {
            if err := ioutil.WriteFile(snapPath, content, 0644); err != nil {
                return err
            }
        }
        
        // Save snapshot metadata to database
        snapshot := FileSnapshot{
            ID:           uuid.New().String(),
            PlanID:       planID,
            CheckpointID: checkpointID,
            FilePath:     file,
            Hash:         hashStr,
        }
        
        if err := sm.db.SaveSnapshot(snapshot); err != nil {
            return err
        }
    }
    
    return nil
}

func (sm *SnapshotManager) RestoreSnapshot(checkpointID string) error {
    snapshots, err := sm.db.GetSnapshots(checkpointID)
    if err != nil {
        return err
    }
    
    for _, snapshot := range snapshots {
        // Read content from content-addressed storage
        snapPath := filepath.Join(sm.baseDir, snapshot.Hash[:2], snapshot.Hash)
        content, err := ioutil.ReadFile(snapPath)
        if err != nil {
            return err
        }
        
        // Restore file
        if err := ioutil.WriteFile(snapshot.FilePath, content, 0644); err != nil {
            return err
        }
    }
    
    return nil
}
```

### 3.2 Enhance Rollback Function

#### Complete Rollback Implementation (`planner/planner.go`)
```go
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
    
    // Pause execution if running
    if task.Status == TaskStatusExecuting {
        task.Status = TaskStatusPaused
    }
    
    // Restore file snapshots
    if err := p.snapshotManager.RestoreSnapshot(checkpointID); err != nil {
        return serr.Wrap(err, "failed to restore file snapshots")
    }
    
    // Handle git operations rollback
    if err := p.rollbackGitOperations(task, checkpoint); err != nil {
        p.logWarning(taskID, "", "Failed to rollback git operations: " + err.Error())
    }
    
    // Reset task state
    task.CurrentStep = len(checkpoint.State.CompletedSteps)
    task.Context.Variables = copyMap(checkpoint.State.Variables)
    task.Context.ModifiedFiles = checkpoint.State.ModifiedFiles
    
    // Reset step statuses
    for i, step := range task.Steps {
        if i >= task.CurrentStep {
            task.Steps[i].Status = StepStatusPending
            task.Steps[i].Result = nil
            task.Steps[i].StartTime = nil
            task.Steps[i].EndTime = nil
        }
    }
    
    // Remove newer checkpoints
    task.Checkpoints = task.Checkpoints[:checkpointIndex+1]
    
    // Update context manager
    if p.contextManager != nil {
        p.contextManager.GetTracker().Clear()
        for _, file := range checkpoint.State.ModifiedFiles {
            p.contextManager.TrackChange(context.FileChange{
                Path: file,
                Type: context.ChangeTypeModify,
                Tool: "rollback",
            })
        }
    }
    
    // Save to database
    if err := p.db.SavePlan(task); err != nil {
        return serr.Wrap(err, "failed to save rollback state")
    }
    
    // Broadcast rollback event
    BroadcastRollback(task.SessionID, taskID, checkpointID)
    
    p.logInfo(taskID, "", fmt.Sprintf("Rolled back to checkpoint: %s", checkpointID))
    
    return nil
}

func (p *Planner) rollbackGitOperations(task *TaskPlanner, checkpoint *Checkpoint) error {
    // Check if any git operations were performed after checkpoint
    gitOps := []string{"git_add", "git_commit", "git_push"}
    hasGitOps := false
    
    for i := len(checkpoint.State.CompletedSteps); i < task.CurrentStep; i++ {
        step := task.Steps[i]
        for _, op := range gitOps {
            if step.Tool == op {
                hasGitOps = true
                break
            }
        }
    }
    
    if !hasGitOps {
        return nil
    }
    
    // Use git reflog to find the state at checkpoint time
    // Execute git reset if safe
    // This is a simplified version - real implementation would be more careful
    
    return nil
}
```

### 3.3 Add Selective Rollback

#### Partial Rollback Support (`planner/rollback.go` - new file)
```go
package planner

type RollbackOptions struct {
    Files      []string // Specific files to rollback
    Steps      []string // Specific steps to undo
    PreserveGit bool    // Don't rollback git operations
}

func (p *Planner) SelectiveRollback(taskID string, options RollbackOptions) error {
    task, err := p.GetPlan(taskID)
    if err != nil {
        return err
    }
    
    // Create a new checkpoint for current state (for redo)
    currentCheckpoint, err := p.createCheckpoint(task)
    if err != nil {
        return err
    }
    
    // Determine which files to rollback
    filesToRestore := make(map[string]string) // file -> checkpoint ID
    
    for _, file := range options.Files {
        // Find the most recent checkpoint that has this file
        for i := len(task.Checkpoints) - 1; i >= 0; i-- {
            cp := task.Checkpoints[i]
            if snapshot, exists := cp.State.FileSnapshots[file]; exists {
                filesToRestore[file] = snapshot
                break
            }
        }
    }
    
    // Restore selected files
    for file, snapshotID := range filesToRestore {
        if err := p.snapshotManager.RestoreFile(file, snapshotID); err != nil {
            return err
        }
    }
    
    // Update task state
    task.Checkpoints = append(task.Checkpoints, currentCheckpoint)
    
    return p.db.SavePlan(task)
}
```

## 4. Integration Points

### 4.1 Provider Integration

#### Plan Mode Detection (`providers/anthropic.go`)
```go
// Add to message handling
func (c *AnthropicClient) detectPlanMode(message string) bool {
    planIndicators := []string{
        "create a plan",
        "break down",
        "step by step",
        "task list",
        "let's plan",
    }
    
    messageLower := strings.ToLower(message)
    for _, indicator := range planIndicators {
        if strings.Contains(messageLower, indicator) {
            return true
        }
    }
    
    return false
}

// Enhanced message handling
func (c *AnthropicClient) SendMessageWithPlanning(request CreateMessageRequest) (*MessageResponse, *planner.TaskPlanner, error) {
    // Check if this is a planning request
    if c.detectPlanMode(request.Messages[len(request.Messages)-1].Content) {
        // Add planning instructions to system prompt
        request.System += "\nWhen asked to create a plan, provide a structured response with clear steps."
    }
    
    response, err := c.SendMessageWithRetry(request)
    if err != nil {
        return nil, nil, err
    }
    
    // Parse plan from response if applicable
    var plan *planner.TaskPlanner
    if c.detectPlanMode(request.Messages[len(request.Messages)-1].Content) {
        plan = c.parsePlanFromResponse(response)
    }
    
    return response, plan, nil
}
```

### 4.2 Tool Integration

#### Dry Run Support (`tools/registry.go`)
```go
// Add to Registry
func (r *Registry) ExecuteDryRun(toolUse ToolUse) (ToolResult, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()
    
    executor, exists := r.executors[toolUse.Name]
    if !exists {
        return ToolResult{}, fmt.Errorf("unknown tool: %s", toolUse.Name)
    }
    
    // Check if tool supports dry run
    if dryRunner, ok := executor.(DryRunner); ok {
        return dryRunner.DryRun(toolUse.Input)
    }
    
    // Default dry run response
    return ToolResult{
        Content: fmt.Sprintf("DRY RUN: Would execute %s with params: %v", 
                            toolUse.Name, toolUse.Input),
        IsError: false,
    }, nil
}

// DryRunner interface
type DryRunner interface {
    DryRun(params map[string]interface{}) (ToolResult, error)
}
```

## 5. Testing Strategy

### Unit Tests (`planner/planner_test.go`)
```go
func TestPlanCreation(t *testing.T) {
    p := NewPlanner(DefaultPlannerOptions())
    
    plan, err := p.CreatePlan("Add a new feature to handle user authentication")
    assert.NoError(t, err)
    assert.NotEmpty(t, plan.ID)
    assert.True(t, len(plan.Steps) > 0)
}

func TestParallelExecution(t *testing.T) {
    // Test that independent steps run in parallel
}

func TestRollback(t *testing.T) {
    // Test complete rollback scenario
}
```

### Integration Tests (`integration/planner_test.go`)
```go
func TestFullPlanExecution(t *testing.T) {
    // Create a plan
    // Execute it
    // Verify results
    // Rollback
    // Verify rollback
}
```

## 6. UI Enhancements (Minimal)

### Frontend Updates (`web/assets/js/ui.js`)
```javascript
// Add plan mode toggle
function togglePlanMode() {
    const planMode = document.getElementById('plan-mode-toggle').checked;
    if (planMode) {
        messageInput.placeholder = "Describe a task to plan...";
    } else {
        messageInput.placeholder = "Type a message...";
    }
}

// Handle plan events
eventSource.addEventListener('plan_created', (event) => {
    const data = JSON.parse(event.data);
    showPlanNotification(data);
});

eventSource.addEventListener('step_progress', (event) => {
    const data = JSON.parse(event.data);
    updateStepProgress(data);
});
```

## Implementation Timeline

### Week 1: Foundation
- Day 1-2: Database migrations and persistence layer
- Day 3-4: Enhance analyzer with context intelligence  
- Day 5: Integration tests setup

### Week 2: Execution
- Day 1-2: Parallel execution support
- Day 3-4: Progress tracking and SSE events
- Day 5: API endpoints and session integration

### Week 3: Rollback
- Day 1-2: Snapshot system implementation
- Day 3-4: Complete rollback functionality
- Day 5: Selective rollback and UI

### Week 4: Polish
- Day 1-2: Comprehensive testing
- Day 3: UI enhancements
- Day 4-5: Documentation and examples

## Next Session Starting Points

1. Begin with database migrations - create the schema
2. Implement the snapshot manager for file backups
3. Enhance the analyzer with context intelligence
4. Create the API endpoints for plan management
5. Test the basic flow end-to-end

This plan provides a solid foundation for implementing a sophisticated task planning and execution system with full rollback capabilities.