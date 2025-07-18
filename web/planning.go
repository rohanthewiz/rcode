package web

import (
	"encoding/json"
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