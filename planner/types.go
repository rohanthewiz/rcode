package planner

import (
	"time"
)

// TaskPlanner manages multi-step task execution
type TaskPlanner struct {
	ID          string            `json:"id"`
	Description string            `json:"description"`
	Steps       []TaskStep        `json:"steps"`
	CurrentStep int               `json:"current_step"`
	Checkpoints []Checkpoint      `json:"checkpoints"`
	Context     *TaskContext      `json:"context"`
	Status      TaskStatus        `json:"status"`
	StartTime   time.Time         `json:"start_time"`
	EndTime     *time.Time        `json:"end_time,omitempty"`
}

// TaskStep represents a single step in a task plan
type TaskStep struct {
	ID           string                 `json:"id"`
	Description  string                 `json:"description"`
	Tool         string                 `json:"tool"`
	Params       map[string]interface{} `json:"params"`
	Dependencies []string               `json:"dependencies"`
	Retryable    bool                   `json:"retryable"`
	MaxRetries   int                    `json:"max_retries"`
	Status       StepStatus             `json:"status"`
	Result       *StepResult            `json:"result,omitempty"`
	StartTime    *time.Time             `json:"start_time,omitempty"`
	EndTime      *time.Time             `json:"end_time,omitempty"`
}

// StepResult contains the result of executing a step
type StepResult struct {
	Success  bool        `json:"success"`
	Output   interface{} `json:"output"`
	Error    string      `json:"error,omitempty"`
	Retries  int         `json:"retries"`
	Duration time.Duration `json:"duration"`
}

// Checkpoint represents a saved state in the task execution
type Checkpoint struct {
	ID          string    `json:"id"`
	StepID      string    `json:"step_id"`
	Timestamp   time.Time `json:"timestamp"`
	State       TaskState `json:"state"`
	Description string    `json:"description"`
}

// TaskContext contains context for task execution
type TaskContext struct {
	WorkingDirectory string                 `json:"working_directory"`
	Environment      map[string]string      `json:"environment"`
	Variables        map[string]interface{} `json:"variables"`
	Files            []string               `json:"files"`
	ModifiedFiles    []string               `json:"modified_files"`
}

// TaskState represents the state of a task at a checkpoint
type TaskState struct {
	CompletedSteps []string               `json:"completed_steps"`
	Variables      map[string]interface{} `json:"variables"`
	FileSnapshots  map[string]string      `json:"file_snapshots"`
}

// TaskStatus represents the overall status of a task
type TaskStatus string

const (
	TaskStatusPending    TaskStatus = "pending"
	TaskStatusPlanning   TaskStatus = "planning"
	TaskStatusExecuting  TaskStatus = "executing"
	TaskStatusPaused     TaskStatus = "paused"
	TaskStatusCompleted  TaskStatus = "completed"
	TaskStatusFailed     TaskStatus = "failed"
	TaskStatusCancelled  TaskStatus = "cancelled"
)

// StepStatus represents the status of a single step
type StepStatus string

const (
	StepStatusPending   StepStatus = "pending"
	StepStatusRunning   StepStatus = "running"
	StepStatusCompleted StepStatus = "completed"
	StepStatusFailed    StepStatus = "failed"
	StepStatusSkipped   StepStatus = "skipped"
	StepStatusRetrying  StepStatus = "retrying"
)

// PlannerOptions contains options for creating a task planner
type PlannerOptions struct {
	MaxSteps          int
	MaxRetries        int
	TimeoutPerStep    time.Duration
	EnableCheckpoints bool
	CheckpointEvery   int // Create checkpoint every N steps
}

// DefaultPlannerOptions returns default planner options
func DefaultPlannerOptions() PlannerOptions {
	return PlannerOptions{
		MaxSteps:          50,
		MaxRetries:        3,
		TimeoutPerStep:    5 * time.Minute,
		EnableCheckpoints: true,
		CheckpointEvery:   5,
	}
}

// TaskTemplate represents a reusable task template
type TaskTemplate struct {
	Name        string              `json:"name"`
	Description string              `json:"description"`
	Category    string              `json:"category"`
	Steps       []TaskStepTemplate  `json:"steps"`
	Variables   []VariableDefinition `json:"variables"`
}

// TaskStepTemplate represents a template for a task step
type TaskStepTemplate struct {
	ID           string                 `json:"id"`
	Description  string                 `json:"description"`
	Tool         string                 `json:"tool"`
	ParamMapping map[string]string      `json:"param_mapping"` // Maps template vars to tool params
	Conditions   []StepCondition        `json:"conditions"`
	OnSuccess    []string               `json:"on_success"` // Next steps on success
	OnFailure    []string               `json:"on_failure"` // Next steps on failure
}

// StepCondition represents a condition for executing a step
type StepCondition struct {
	Type     string `json:"type"` // "variable", "file_exists", "previous_step"
	Variable string `json:"variable,omitempty"`
	Operator string `json:"operator,omitempty"` // "equals", "not_equals", "exists"
	Value    interface{} `json:"value,omitempty"`
}

// VariableDefinition defines a variable in a template
type VariableDefinition struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Description  string      `json:"description"`
	DefaultValue interface{} `json:"default_value,omitempty"`
	Required     bool        `json:"required"`
}

// ExecutionLog represents a log entry during task execution
type ExecutionLog struct {
	Timestamp time.Time   `json:"timestamp"`
	Level     string      `json:"level"` // "info", "warning", "error"
	StepID    string      `json:"step_id,omitempty"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
}

// TaskReport represents a summary report of task execution
type TaskReport struct {
	TaskID          string        `json:"task_id"`
	Description     string        `json:"description"`
	Status          TaskStatus    `json:"status"`
	TotalSteps      int           `json:"total_steps"`
	CompletedSteps  int           `json:"completed_steps"`
	FailedSteps     int           `json:"failed_steps"`
	Duration        time.Duration `json:"duration"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         *time.Time    `json:"end_time,omitempty"`
	ModifiedFiles   []string      `json:"modified_files"`
	Errors          []string      `json:"errors"`
	Checkpoints     int           `json:"checkpoints"`
	LastCheckpoint  *Checkpoint   `json:"last_checkpoint,omitempty"`
}