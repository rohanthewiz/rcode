package web

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/rohanthewiz/element"
	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
	"rcode/context"
	"rcode/db"
	"rcode/planner"
)

// CreatePlanRequest represents a request to create a task plan
type CreatePlanRequest struct {
	Description string `json:"description"`
	AutoExecute bool   `json:"auto_execute"`
}

// PlanResponse represents a task plan in API responses
type PlanResponse struct {
	ID          string                 `json:"id"`
	SessionID   string                 `json:"session_id"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	Steps       []planner.TaskStep     `json:"steps"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
}

// createPlanHandler creates a new task plan
func createPlanHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")
	if sessionID == "" {
		return c.WriteError(serr.New("session ID required"), 400)
	}
	
	var req CreatePlanRequest
	if err := json.Unmarshal(c.Request().Body(), &req); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}
	
	if req.Description == "" {
		return c.WriteError(serr.New("description required"), 400)
	}
	
	// Get context manager
	contextMgr := context.NewManager()
	
	// Create planner instance with context using factory
	plannerOpts := planner.PlannerOptions{
		MaxConcurrentSteps: 3,
		EnableCheckpoints:  true,
		CheckpointInterval: 5,
		ContextManager:     contextMgr,
	}
	factory := planner.NewPlannerFactory()
	taskPlanner := factory.CreatePlanner(plannerOpts)
	
	// Create plan
	plan, err := taskPlanner.CreatePlan(req.Description)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to create plan"), 500)
	}
	
	// Associate with session
	plan.SessionID = sessionID
	
	// Save to database
	taskDB := db.GetTaskPlanDB()
	dbPlan := &db.TaskPlan{
		ID:          plan.ID,
		SessionID:   plan.SessionID,
		Description: plan.Description,
		Status:      db.PlanStatus(plan.Status),
		CreatedAt:   plan.CreatedAt,
		UpdatedAt:   plan.UpdatedAt,
	}
	
	// Marshal plan details
	stepsJSON, _ := json.Marshal(plan.Steps)
	contextJSON, _ := json.Marshal(plan.Context)
	checkpointsJSON, _ := json.Marshal(plan.Checkpoints)
	
	dbPlan.Steps = stepsJSON
	dbPlan.Context = contextJSON
	dbPlan.Checkpoints = checkpointsJSON
	
	if err := taskDB.SavePlan(dbPlan); err != nil {
		return c.WriteError(serr.Wrap(err, "failed to save plan"), 500)
	}
	
	// Broadcast plan creation event
	broadcastPlanEvent("plan_created", sessionID, plan.ID, map[string]interface{}{
		"description": plan.Description,
		"steps":       len(plan.Steps),
		"status":      plan.Status,
	})
	
	// Auto-execute if requested
	if req.AutoExecute {
		go func() {
			logger.Info("Starting auto-execution of plan", "plan_id", plan.ID)
			if err := taskPlanner.ExecutePlan(plan.ID); err != nil {
				logger.LogErr(err, "auto-execution failed", "plan_id", plan.ID)
			}
		}()
	}
	
	// Create response
	response := PlanResponse{
		ID:          plan.ID,
		SessionID:   plan.SessionID,
		Description: plan.Description,
		Status:      string(plan.Status),
		Steps:       plan.Steps,
		CreatedAt:   plan.CreatedAt,
		UpdatedAt:   plan.UpdatedAt,
		CompletedAt: plan.CompletedAt,
	}
	
	return c.WriteJSON(response)
}

// listPlansHandler lists all plans for a session
func listPlansHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")
	if sessionID == "" {
		return c.WriteError(serr.New("session ID required"), 400)
	}
	
	taskDB := db.GetTaskPlanDB()
	plans, err := taskDB.GetSessionPlans(sessionID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get plans"), 500)
	}
	
	// Convert to response format
	responses := make([]PlanResponse, len(plans))
	for i, plan := range plans {
		var steps []planner.TaskStep
		if err := json.Unmarshal(plan.Steps, &steps); err != nil {
			logger.LogErr(err, "failed to unmarshal steps", "plan_id", plan.ID)
			steps = []planner.TaskStep{}
		}
		
		responses[i] = PlanResponse{
			ID:          plan.ID,
			SessionID:   plan.SessionID,
			Description: plan.Description,
			Status:      string(plan.Status),
			Steps:       steps,
			CreatedAt:   plan.CreatedAt,
			UpdatedAt:   plan.UpdatedAt,
			CompletedAt: plan.CompletedAt,
		}
	}
	
	return c.WriteJSON(responses)
}

// executePlanHandler executes a task plan
func executePlanHandler(c rweb.Context) error {
	planID := c.Request().Param("id")
	if planID == "" {
		return c.WriteError(serr.New("plan ID required"), 400)
	}
	
	// Get plan from database
	taskDB := db.GetTaskPlanDB()
	dbPlan, err := taskDB.GetPlan(planID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get plan"), 404)
	}
	
	// Create planner instance using factory
	contextMgr := context.NewManager()
	plannerOpts := planner.PlannerOptions{
		MaxConcurrentSteps: 3,
		EnableCheckpoints:  true,
		CheckpointInterval: 5,
		ContextManager:     contextMgr,
	}
	factory := planner.NewPlannerFactory()
	taskPlanner := factory.CreatePlanner(plannerOpts)
	
	// Execute plan asynchronously
	go func() {
		logger.Info("Starting plan execution", "plan_id", planID)
		
		// Update status to executing
		dbPlan.Status = db.PlanStatusExecuting
		if err := taskDB.SavePlan(dbPlan); err != nil {
			logger.LogErr(err, "failed to update plan status", "plan_id", planID)
		}
		
		broadcastPlanEvent("plan_executing", dbPlan.SessionID, planID, nil)
		
		// Convert DB plan to planner.TaskPlanner
		var steps []planner.TaskStep
		if err := json.Unmarshal(dbPlan.Steps, &steps); err != nil {
			logger.LogErr(err, "failed to unmarshal steps", "plan_id", planID)
			return
		}
		
		var checkpoints []planner.Checkpoint
		if dbPlan.Checkpoints != nil {
			if err := json.Unmarshal(dbPlan.Checkpoints, &checkpoints); err != nil {
				logger.LogErr(err, "failed to unmarshal checkpoints", "plan_id", planID)
			}
		}
		
		var ctx *planner.TaskContext
		if dbPlan.Context != nil {
			if err := json.Unmarshal(dbPlan.Context, &ctx); err != nil {
				logger.LogErr(err, "failed to unmarshal context", "plan_id", planID)
				ctx = &planner.TaskContext{
					Variables:     make(map[string]interface{}),
					Environment:   make(map[string]string),
					Files:         make([]string, 0),
					ModifiedFiles: make([]string, 0),
				}
			}
		} else {
			ctx = &planner.TaskContext{
				Variables:     make(map[string]interface{}),
				Environment:   make(map[string]string),
				Files:         make([]string, 0),
				ModifiedFiles: make([]string, 0),
			}
		}
		
		// Create planner.TaskPlanner from DB data
		plan := &planner.TaskPlanner{
			ID:          dbPlan.ID,
			SessionID:   dbPlan.SessionID,
			Description: dbPlan.Description,
			Status:      planner.TaskStatus(dbPlan.Status),
			Steps:       steps,
			CurrentStep: 0,
			Checkpoints: checkpoints,
			Context:     ctx,
			StartTime:   dbPlan.CreatedAt, // Use CreatedAt as StartTime
			CreatedAt:   dbPlan.CreatedAt,
			UpdatedAt:   dbPlan.UpdatedAt,
			CompletedAt: dbPlan.CompletedAt,
		}
		
		// Load the plan into the planner's memory
		if err := taskPlanner.LoadPlan(plan); err != nil {
			logger.LogErr(err, "failed to load plan into planner", "plan_id", planID)
			return
		}
		
		// Execute the plan
		if err := taskPlanner.ExecutePlan(planID); err != nil {
			logger.LogErr(err, "plan execution failed", "plan_id", planID)
			
			// Update status to failed
			dbPlan.Status = db.PlanStatusFailed
			now := time.Now()
			dbPlan.CompletedAt = &now
			if err := taskDB.SavePlan(dbPlan); err != nil {
				logger.LogErr(err, "failed to update plan status", "plan_id", planID)
			}
			
			broadcastPlanEvent("plan_failed", dbPlan.SessionID, planID, map[string]interface{}{
				"error": err.Error(),
			})
		} else {
			// Update status to completed
			dbPlan.Status = db.PlanStatusCompleted
			now := time.Now()
			dbPlan.CompletedAt = &now
			if err := taskDB.SavePlan(dbPlan); err != nil {
				logger.LogErr(err, "failed to update plan status", "plan_id", planID)
			}
			
			broadcastPlanEvent("plan_completed", dbPlan.SessionID, planID, nil)
		}
	}()
	
	return c.WriteJSON(map[string]string{
		"status": "execution_started",
		"plan_id": planID,
	})
}

// getPlanStatusHandler gets the current status of a plan
func getPlanStatusHandler(c rweb.Context) error {
	planID := c.Request().Param("id")
	if planID == "" {
		return c.WriteError(serr.New("plan ID required"), 400)
	}
	
	taskDB := db.GetTaskPlanDB()
	plan, err := taskDB.GetPlan(planID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get plan"), 404)
	}
	
	// Get executions
	executions, err := taskDB.GetExecutions(planID)
	if err != nil {
		logger.LogErr(err, "failed to get executions", "plan_id", planID)
		executions = []*db.TaskExecution{}
	}
	
	// Get metrics
	metrics, err := taskDB.GetMetrics(planID)
	if err != nil {
		logger.LogErr(err, "failed to get metrics", "plan_id", planID)
	}
	
	response := map[string]interface{}{
		"plan_id":     plan.ID,
		"status":      plan.Status,
		"description": plan.Description,
		"created_at":  plan.CreatedAt,
		"updated_at":  plan.UpdatedAt,
		"completed_at": plan.CompletedAt,
		"executions":  executions,
		"metrics":     metrics,
	}
	
	return c.WriteJSON(response)
}

// rollbackPlanHandler rolls back a plan to a checkpoint
func rollbackPlanHandler(c rweb.Context) error {
	planID := c.Request().Param("id")
	if planID == "" {
		return c.WriteError(serr.New("plan ID required"), 400)
	}
	
	var req struct {
		CheckpointID string `json:"checkpoint_id"`
	}
	if err := json.Unmarshal(c.Request().Body(), &req); err != nil {
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}
	
	if req.CheckpointID == "" {
		return c.WriteError(serr.New("checkpoint_id required"), 400)
	}
	
	// Get plan from database
	taskDB := db.GetTaskPlanDB()
	dbPlan, err := taskDB.GetPlan(planID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get plan"), 404)
	}
	
	// Create planner instance using factory
	contextMgr := context.NewManager()
	plannerOpts := planner.PlannerOptions{
		MaxConcurrentSteps: 3,
		EnableCheckpoints:  true,
		CheckpointInterval: 5,
		ContextManager:     contextMgr,
	}
	factory := planner.NewPlannerFactory()
	taskPlanner := factory.CreatePlanner(plannerOpts)
	
	// Convert DB plan to planner.TaskPlanner and load it
	var steps []planner.TaskStep
	if err := json.Unmarshal(dbPlan.Steps, &steps); err != nil {
		return c.WriteError(serr.Wrap(err, "failed to unmarshal steps"), 500)
	}
	
	var checkpoints []planner.Checkpoint
	if dbPlan.Checkpoints != nil {
		if err := json.Unmarshal(dbPlan.Checkpoints, &checkpoints); err != nil {
			return c.WriteError(serr.Wrap(err, "failed to unmarshal checkpoints"), 500)
		}
	}
	
	var ctx *planner.TaskContext
	if dbPlan.Context != nil {
		if err := json.Unmarshal(dbPlan.Context, &ctx); err != nil {
			ctx = &planner.TaskContext{
				Variables:     make(map[string]interface{}),
				Environment:   make(map[string]string),
				Files:         make([]string, 0),
				ModifiedFiles: make([]string, 0),
			}
		}
	} else {
		ctx = &planner.TaskContext{
			Variables:     make(map[string]interface{}),
			Environment:   make(map[string]string),
			Files:         make([]string, 0),
			ModifiedFiles: make([]string, 0),
		}
	}
	
	// Create planner.TaskPlanner from DB data
	plan := &planner.TaskPlanner{
		ID:          dbPlan.ID,
		SessionID:   dbPlan.SessionID,
		Description: dbPlan.Description,
		Status:      planner.TaskStatus(dbPlan.Status),
		Steps:       steps,
		CurrentStep: 0,
		Checkpoints: checkpoints,
		Context:     ctx,
		StartTime:   dbPlan.CreatedAt, // Use CreatedAt as StartTime
		CreatedAt:   dbPlan.CreatedAt,
		UpdatedAt:   dbPlan.UpdatedAt,
		CompletedAt: dbPlan.CompletedAt,
	}
	
	// Load the plan into the planner's memory
	if err := taskPlanner.LoadPlan(plan); err != nil {
		return c.WriteError(serr.Wrap(err, "failed to load plan"), 500)
	}
	
	// Perform rollback
	if err := taskPlanner.RollbackToCheckpoint(planID, req.CheckpointID); err != nil {
		return c.WriteError(serr.Wrap(err, "rollback failed"), 500)
	}
	
	// Broadcast rollback event
	broadcastPlanEvent("plan_rollback", dbPlan.SessionID, planID, map[string]interface{}{
		"checkpoint_id": req.CheckpointID,
	})
	
	return c.WriteJSON(map[string]string{
		"status": "rollback_completed",
		"plan_id": planID,
		"checkpoint_id": req.CheckpointID,
	})
}

// listCheckpointsHandler lists checkpoints for a plan
func listCheckpointsHandler(c rweb.Context) error {
	planID := c.Request().Param("id")
	if planID == "" {
		return c.WriteError(serr.New("plan ID required"), 400)
	}
	
	taskDB := db.GetTaskPlanDB()
	plan, err := taskDB.GetPlan(planID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get plan"), 404)
	}
	
	var checkpoints []planner.Checkpoint
	if err := json.Unmarshal(plan.Checkpoints, &checkpoints); err != nil {
		return c.WriteError(serr.Wrap(err, "failed to unmarshal checkpoints"), 500)
	}
	
	return c.WriteJSON(checkpoints)
}

// broadcastPlanEvent broadcasts a plan-related event via SSE
func broadcastPlanEvent(eventType, sessionID, planID string, data interface{}) {
	event := map[string]interface{}{
		"type":       eventType,
		"session_id": sessionID,
		"plan_id":    planID,
		"timestamp":  time.Now().Format(time.RFC3339),
	}
	
	if data != nil {
		event["data"] = data
	}
	
	// Use existing SSE broadcast function
	broadcastJSON(eventType, event)
}

// analyzePlanHandler analyzes the parallelizability of a plan
func analyzePlanHandler(c rweb.Context) error {
	planID := c.Request().Param("id")
	if planID == "" {
		return c.WriteError(serr.New("plan ID required"), 400)
	}
	
	// Get plan from database
	taskDB := db.GetTaskPlanDB()
	dbPlan, err := taskDB.GetPlan(planID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get plan"), 404)
	}
	
	// Create planner instance using factory
	contextMgr := context.NewManager()
	plannerOpts := planner.PlannerOptions{
		MaxConcurrentSteps: 5, // Enable parallel analysis
		EnableCheckpoints:  true,
		CheckpointInterval: 5,
		ContextManager:     contextMgr,
	}
	factory := planner.NewPlannerFactory()
	taskPlanner := factory.CreatePlanner(plannerOpts)
	
	// Convert DB plan to planner.TaskPlanner
	var steps []planner.TaskStep
	if err := json.Unmarshal(dbPlan.Steps, &steps); err != nil {
		return c.WriteError(serr.Wrap(err, "failed to unmarshal steps"), 500)
	}
	
	var checkpoints []planner.Checkpoint
	if dbPlan.Checkpoints != nil {
		if err := json.Unmarshal(dbPlan.Checkpoints, &checkpoints); err != nil {
			logger.LogErr(err, "failed to unmarshal checkpoints", "plan_id", planID)
		}
	}
	
	var ctx *planner.TaskContext
	if dbPlan.Context != nil {
		if err := json.Unmarshal(dbPlan.Context, &ctx); err != nil {
			ctx = &planner.TaskContext{
				Variables:     make(map[string]interface{}),
				Environment:   make(map[string]string),
				Files:         make([]string, 0),
				ModifiedFiles: make([]string, 0),
			}
		}
	} else {
		ctx = &planner.TaskContext{
			Variables:     make(map[string]interface{}),
			Environment:   make(map[string]string),
			Files:         make([]string, 0),
			ModifiedFiles: make([]string, 0),
		}
	}
	
	// Create planner.TaskPlanner from DB data
	plan := &planner.TaskPlanner{
		ID:          dbPlan.ID,
		SessionID:   dbPlan.SessionID,
		Description: dbPlan.Description,
		Status:      planner.TaskStatus(dbPlan.Status),
		Steps:       steps,
		CurrentStep: 0,
		Checkpoints: checkpoints,
		Context:     ctx,
		StartTime:   dbPlan.CreatedAt,
		CreatedAt:   dbPlan.CreatedAt,
		UpdatedAt:   dbPlan.UpdatedAt,
		CompletedAt: dbPlan.CompletedAt,
	}
	
	// Load the plan and analyze
	if err := taskPlanner.LoadPlan(plan); err != nil {
		return c.WriteError(serr.Wrap(err, "failed to load plan"), 500)
	}
	
	analysis, err := taskPlanner.AnalyzeParallelizability(planID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to analyze plan"), 500)
	}
	
	return c.WriteJSON(analysis)
}

// getGitOperationsHandler returns Git operations for a plan
func getGitOperationsHandler(c rweb.Context) error {
	planID := c.Request().Param("id")
	if planID == "" {
		return c.WriteError(serr.New("plan ID required"), 400)
	}
	
	// Get plan from database to verify it exists
	taskDB := db.GetTaskPlanDB()
	_, err := taskDB.GetPlan(planID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "plan not found"), 404)
	}
	
	// Create planner instance using factory
	contextMgr := context.NewManager()
	plannerOpts := planner.PlannerOptions{
		MaxConcurrentSteps: 3,
		EnableCheckpoints:  true,
		CheckpointInterval: 5,
		ContextManager:     contextMgr,
	}
	factory := planner.NewPlannerFactory()
	taskPlanner := factory.CreatePlanner(plannerOpts)
	
	// Get Git operations
	gitOps, err := taskPlanner.GetGitOperations(planID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get git operations"), 500)
	}
	
	return c.WriteJSON(gitOps)
}

// planManagementUI renders the plan management UI
func planManagementUI(b *element.Builder) {
	b.Div("id", "plan-management", "class", "plan-container").R(
		b.H3().T("Task Plans"),
		b.Div("class", "plan-controls").R(
			b.Button("id", "create-plan-btn", "class", "btn btn-primary").T("Create Plan"),
			b.Button("id", "refresh-plans-btn", "class", "btn btn-secondary").T("Refresh"),
		),
		b.Div("id", "plans-list", "class", "plans-list").T("Loading plans..."),
		b.Div("id", "plan-details", "class", "plan-details hidden"),
	)
}

// listPlanHistoryHandler returns paginated plan history with search and filtering
func listPlanHistoryHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")
	if sessionID == "" {
		return c.WriteError(serr.New("session ID required"), 400)
	}
	
	// Parse query parameters
	page := 1
	if pageStr := c.Request().Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	
	limit := 20
	if limitStr := c.Request().Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}
	
	status := c.Request().Query("status")
	search := c.Request().Query("search")
	
	// Get plans from database with pagination
	taskDB := db.GetTaskPlanDB()
	offset := (page - 1) * limit
	
	// Get filtered plans
	plans, total, err := taskDB.GetSessionPlansWithFilter(sessionID, status, search, limit, offset)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get plans"), 500)
	}
	
	// Convert to response format with basic info only
	responses := make([]map[string]interface{}, len(plans))
	for i, plan := range plans {
		// Count steps
		var steps []planner.TaskStep
		stepCount := 0
		if err := json.Unmarshal(plan.Steps, &steps); err == nil {
			stepCount = len(steps)
		}
		
		// Calculate duration if completed
		var duration *time.Duration
		if plan.CompletedAt != nil {
			d := plan.CompletedAt.Sub(plan.CreatedAt)
			duration = &d
		}
		
		responses[i] = map[string]interface{}{
			"id":          plan.ID,
			"description": plan.Description,
			"status":      plan.Status,
			"created_at":  plan.CreatedAt,
			"step_count":  stepCount,
			"duration":    duration,
		}
	}
	
	// Return paginated response
	return c.WriteJSON(map[string]interface{}{
		"plans":       responses,
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": (total + limit - 1) / limit,
	})
}

// getPlanFullDetailsHandler returns complete plan details including all steps
func getPlanFullDetailsHandler(c rweb.Context) error {
	planID := c.Request().Param("id")
	if planID == "" {
		return c.WriteError(serr.New("plan ID required"), 400)
	}
	
	taskDB := db.GetTaskPlanDB()
	plan, err := taskDB.GetPlan(planID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get plan"), 404)
	}
	
	// Get executions
	executions, err := taskDB.GetExecutions(planID)
	if err != nil {
		logger.LogErr(err, "failed to get executions", "plan_id", planID)
		executions = []*db.TaskExecution{}
	}
	
	// Get metrics
	metrics, err := taskDB.GetMetrics(planID)
	if err != nil {
		logger.LogErr(err, "failed to get metrics", "plan_id", planID)
	}
	
	// Unmarshal steps
	var steps []planner.TaskStep
	if err := json.Unmarshal(plan.Steps, &steps); err != nil {
		logger.LogErr(err, "failed to unmarshal steps", "plan_id", planID)
		steps = []planner.TaskStep{}
	}
	
	// Unmarshal checkpoints
	var checkpoints []planner.Checkpoint
	if plan.Checkpoints != nil {
		if err := json.Unmarshal(plan.Checkpoints, &checkpoints); err != nil {
			logger.LogErr(err, "failed to unmarshal checkpoints", "plan_id", planID)
		}
	}
	
	// Calculate execution stats
	var totalDuration time.Duration
	successCount := 0
	for _, exec := range executions {
		if exec.EndTime != nil {
			totalDuration += exec.EndTime.Sub(exec.StartTime)
			if exec.Status == "completed" {
				successCount++
			}
		}
	}
	
	successRate := 0.0
	if len(executions) > 0 {
		successRate = float64(successCount) / float64(len(executions)) * 100
	}
	
	// Get modified files from context
	var ctx *planner.TaskContext
	modifiedFiles := []string{}
	if plan.Context != nil {
		if err := json.Unmarshal(plan.Context, &ctx); err == nil {
			modifiedFiles = ctx.ModifiedFiles
		}
	}
	
	// Get git operations from steps
	gitOps := []map[string]interface{}{}
	for _, step := range steps {
		if step.Tool == "git_add" || step.Tool == "git_commit" || step.Tool == "git_push" || 
		   step.Tool == "git_pull" || step.Tool == "git_checkout" || step.Tool == "git_merge" {
			gitOps = append(gitOps, map[string]interface{}{
				"tool":       step.Tool,
				"parameters": step.Parameters,
				"status":     step.Status,
			})
		}
	}
	
	response := map[string]interface{}{
		"plan": PlanResponse{
			ID:          plan.ID,
			SessionID:   plan.SessionID,
			Description: plan.Description,
			Status:      string(plan.Status),
			Steps:       steps,
			CreatedAt:   plan.CreatedAt,
			UpdatedAt:   plan.UpdatedAt,
			CompletedAt: plan.CompletedAt,
		},
		"executions":     executions,
		"metrics":        metrics,
		"checkpoints":    checkpoints,
		"stats": map[string]interface{}{
			"total_duration": totalDuration.Seconds(),
			"success_rate":   successRate,
			"execution_count": len(executions),
		},
		"modified_files": modifiedFiles,
		"git_operations": gitOps,
	}
	
	return c.WriteJSON(response)
}

// clonePlanHandler creates a copy of an existing plan for re-execution
func clonePlanHandler(c rweb.Context) error {
	planID := c.Request().Param("id")
	if planID == "" {
		return c.WriteError(serr.New("plan ID required"), 400)
	}
	
	// Get original plan
	taskDB := db.GetTaskPlanDB()
	originalPlan, err := taskDB.GetPlan(planID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get plan"), 404)
	}
	
	// Unmarshal steps
	var steps []planner.TaskStep
	if err := json.Unmarshal(originalPlan.Steps, &steps); err != nil {
		return c.WriteError(serr.Wrap(err, "failed to unmarshal steps"), 500)
	}
	
	// Reset step statuses
	for i := range steps {
		steps[i].Status = planner.TaskStatusPending
		steps[i].Error = ""
		steps[i].Result = nil
		steps[i].StartTime = nil
		steps[i].EndTime = nil
	}
	
	// Create new plan with same steps
	newPlan := &db.TaskPlan{
		ID:          generateID(),
		SessionID:   originalPlan.SessionID,
		Description: originalPlan.Description + " (cloned)",
		Status:      db.PlanStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// Marshal steps
	stepsJSON, err := json.Marshal(steps)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to marshal steps"), 500)
	}
	newPlan.Steps = stepsJSON
	
	// Initialize empty context and checkpoints
	ctx := &planner.TaskContext{
		Variables:     make(map[string]interface{}),
		Environment:   make(map[string]string),
		Files:         make([]string, 0),
		ModifiedFiles: make([]string, 0),
	}
	contextJSON, _ := json.Marshal(ctx)
	newPlan.Context = contextJSON
	
	checkpointsJSON, _ := json.Marshal([]planner.Checkpoint{})
	newPlan.Checkpoints = checkpointsJSON
	
	// Save new plan
	if err := taskDB.SavePlan(newPlan); err != nil {
		return c.WriteError(serr.Wrap(err, "failed to save cloned plan"), 500)
	}
	
	// Broadcast plan creation event
	broadcastPlanEvent("plan_cloned", newPlan.SessionID, newPlan.ID, map[string]interface{}{
		"original_id": planID,
		"description": newPlan.Description,
		"steps":       len(steps),
	})
	
	// Return new plan details
	response := PlanResponse{
		ID:          newPlan.ID,
		SessionID:   newPlan.SessionID,
		Description: newPlan.Description,
		Status:      string(newPlan.Status),
		Steps:       steps,
		CreatedAt:   newPlan.CreatedAt,
		UpdatedAt:   newPlan.UpdatedAt,
	}
	
	return c.WriteJSON(response)
}

// deletePlanHandler deletes a plan from history
func deletePlanHandler(c rweb.Context) error {
	planID := c.Request().Param("id")
	if planID == "" {
		return c.WriteError(serr.New("plan ID required"), 400)
	}
	
	taskDB := db.GetTaskPlanDB()
	
	// Get plan to get session ID for event
	plan, err := taskDB.GetPlan(planID)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get plan"), 404)
	}
	
	// Delete the plan
	if err := taskDB.DeletePlan(planID); err != nil {
		return c.WriteError(serr.Wrap(err, "failed to delete plan"), 500)
	}
	
	// Broadcast deletion event
	broadcastPlanEvent("plan_deleted", plan.SessionID, planID, nil)
	
	return c.WriteJSON(map[string]string{
		"status": "deleted",
		"plan_id": planID,
	})
}

// generateID generates a unique ID for plans
func generateID() string {
	// Simple implementation - in production, use UUID or similar
	return fmt.Sprintf("plan_%d_%d", time.Now().Unix(), rand.Intn(10000))
}