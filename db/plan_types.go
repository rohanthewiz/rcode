package db

import (
	"encoding/json"
	"time"
)

// PlanStatus represents the status of a task plan
type PlanStatus string

const (
	PlanStatusPending    PlanStatus = "pending"
	PlanStatusPlanning   PlanStatus = "planning"
	PlanStatusExecuting  PlanStatus = "executing"
	PlanStatusPaused     PlanStatus = "paused"
	PlanStatusCompleted  PlanStatus = "completed"
	PlanStatusFailed     PlanStatus = "failed"
	PlanStatusCancelled  PlanStatus = "cancelled"
)

// TaskPlan represents a stored task plan
type TaskPlan struct {
	ID           string          `json:"id"`
	SessionID    string          `json:"session_id"`
	Description  string          `json:"description"`
	Status       PlanStatus      `json:"status"`
	Steps        json.RawMessage `json:"steps"`
	Context      json.RawMessage `json:"context,omitempty"`
	Checkpoints  json.RawMessage `json:"checkpoints,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
}

// StepResult represents the result of a step execution
type StepResult struct {
	Success    bool          `json:"success"`
	Output     interface{}   `json:"output,omitempty"`
	Error      string        `json:"error,omitempty"`
	Duration   time.Duration `json:"duration"`
	Retries    int           `json:"retries"`
	ToolResult interface{}   `json:"tool_result,omitempty"`
}