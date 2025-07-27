package db

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/rohanthewiz/serr"
)

// TaskPlanDB handles task plan persistence
type TaskPlanDB struct {
	db *DB
}

// NewTaskPlanDB creates a new TaskPlanDB instance
func NewTaskPlanDB(db *DB) *TaskPlanDB {
	return &TaskPlanDB{db: db}
}

// SavePlan saves a task plan to the database
func (t *TaskPlanDB) SavePlan(plan *TaskPlan) error {
	stepsJSON, err := json.Marshal(plan.Steps)
	if err != nil {
		return serr.Wrap(err, "failed to marshal steps")
	}
	
	contextJSON, err := json.Marshal(plan.Context)
	if err != nil {
		return serr.Wrap(err, "failed to marshal context")
	}
	
	checkpointsJSON, err := json.Marshal(plan.Checkpoints)
	if err != nil {
		return serr.Wrap(err, "failed to marshal checkpoints")
	}
	
	query := `
		INSERT INTO task_plans (id, session_id, description, status, steps, context, checkpoints)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			status = excluded.status,
			steps = excluded.steps,
			context = excluded.context,
			checkpoints = excluded.checkpoints,
			updated_at = CURRENT_TIMESTAMP,
			completed_at = CASE WHEN excluded.status IN ('completed', 'failed', 'cancelled') THEN CURRENT_TIMESTAMP ELSE completed_at END
	`
	
	_, err = t.db.Exec(query, plan.ID, plan.SessionID, plan.Description, string(plan.Status),
		string(stepsJSON), string(contextJSON), string(checkpointsJSON))
	
	return serr.Wrap(err, "failed to save plan")
}

// GetPlan retrieves a task plan by ID
func (t *TaskPlanDB) GetPlan(planID string) (*TaskPlan, error) {
	var plan TaskPlan
	var stepsJSON, contextJSON, checkpointsJSON string
	var completedAt sql.NullTime
	var status string
	
	query := `
		SELECT id, session_id, description, status, steps, context, checkpoints, 
		       created_at, updated_at, completed_at
		FROM task_plans
		WHERE id = ?
	`
	
	err := t.db.QueryRow(query, planID).Scan(
		&plan.ID, &plan.SessionID, &plan.Description, &status,
		&stepsJSON, &contextJSON, &checkpointsJSON,
		&plan.CreatedAt, &plan.UpdatedAt, &completedAt,
	)
	plan.Status = PlanStatus(status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, serr.New("plan not found")
		}
		return nil, serr.Wrap(err, "failed to get plan")
	}
	
	if completedAt.Valid {
		plan.CompletedAt = &completedAt.Time
	}
	
	// Store raw JSON
	plan.Steps = json.RawMessage(stepsJSON)
	plan.Context = json.RawMessage(contextJSON)
	plan.Checkpoints = json.RawMessage(checkpointsJSON)
	
	return &plan, nil
}

// GetSessionPlans retrieves all plans for a session
func (t *TaskPlanDB) GetSessionPlans(sessionID string) ([]*TaskPlan, error) {
	query := `
		SELECT id, session_id, description, status, steps, context, checkpoints,
		       created_at, updated_at, completed_at
		FROM task_plans
		WHERE session_id = ?
		ORDER BY created_at DESC
	`
	
	rows, err := t.db.Query(query, sessionID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to query plans")
	}
	defer rows.Close()
	
	var plans []*TaskPlan
	for rows.Next() {
		var plan TaskPlan
		var stepsJSON, contextJSON, checkpointsJSON string
		var completedAt sql.NullTime
		var status string
		
		err := rows.Scan(
			&plan.ID, &plan.SessionID, &plan.Description, &status,
			&stepsJSON, &contextJSON, &checkpointsJSON,
			&plan.CreatedAt, &plan.UpdatedAt, &completedAt,
		)
		plan.Status = PlanStatus(status)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan plan")
		}
		
		if completedAt.Valid {
			plan.CompletedAt = &completedAt.Time
		}
		
		// Store raw JSON
		plan.Steps = json.RawMessage(stepsJSON)
		plan.Context = json.RawMessage(contextJSON)
		plan.Checkpoints = json.RawMessage(checkpointsJSON)
		
		plans = append(plans, &plan)
	}
	
	return plans, nil
}

// GetSessionPlansWithFilter retrieves filtered plans for a session with pagination
func (t *TaskPlanDB) GetSessionPlansWithFilter(sessionID, status, search string, limit, offset int) ([]*TaskPlan, int, error) {
	// First, get the total count
	countQuery := `
		SELECT COUNT(*)
		FROM task_plans
		WHERE session_id = ?
	`
	args := []interface{}{sessionID}
	
	if status != "" {
		countQuery += " AND status = ?"
		args = append(args, status)
	}
	
	if search != "" {
		countQuery += " AND description LIKE ?"
		args = append(args, "%"+search+"%")
	}
	
	var total int
	err := t.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, serr.Wrap(err, "failed to count plans")
	}
	
	// Now get the paginated results
	query := `
		SELECT id, session_id, description, status, steps, context, checkpoints,
		       created_at, updated_at, completed_at
		FROM task_plans
		WHERE session_id = ?
	`
	
	// Reset args for the main query
	args = []interface{}{sessionID}
	
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	
	if search != "" {
		query += " AND description LIKE ?"
		args = append(args, "%"+search+"%")
	}
	
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)
	
	rows, err := t.db.Query(query, args...)
	if err != nil {
		return nil, 0, serr.Wrap(err, "failed to query plans")
	}
	defer rows.Close()
	
	var plans []*TaskPlan
	for rows.Next() {
		var plan TaskPlan
		var stepsJSON, contextJSON, checkpointsJSON string
		var completedAt sql.NullTime
		var status string
		
		err := rows.Scan(
			&plan.ID, &plan.SessionID, &plan.Description, &status,
			&stepsJSON, &contextJSON, &checkpointsJSON,
			&plan.CreatedAt, &plan.UpdatedAt, &completedAt,
		)
		plan.Status = PlanStatus(status)
		if err != nil {
			return nil, 0, serr.Wrap(err, "failed to scan plan")
		}
		
		if completedAt.Valid {
			plan.CompletedAt = &completedAt.Time
		}
		
		// Store raw JSON
		plan.Steps = json.RawMessage(stepsJSON)
		plan.Context = json.RawMessage(contextJSON)
		plan.Checkpoints = json.RawMessage(checkpointsJSON)
		
		plans = append(plans, &plan)
	}
	
	return plans, total, nil
}

// DeletePlan deletes a plan and all related data
func (t *TaskPlanDB) DeletePlan(planID string) error {
	// Use a transaction to ensure all related data is deleted
	tx, err := t.db.Conn().Begin()
	if err != nil {
		return serr.Wrap(err, "failed to start transaction")
	}
	defer tx.Rollback()
	
	// Delete in order to respect foreign key constraints
	// Delete logs
	_, err = tx.Exec("DELETE FROM task_logs WHERE plan_id = ?", planID)
	if err != nil {
		return serr.Wrap(err, "failed to delete logs")
	}
	
	// Delete metrics
	_, err = tx.Exec("DELETE FROM task_metrics WHERE plan_id = ?", planID)
	if err != nil {
		return serr.Wrap(err, "failed to delete metrics")
	}
	
	// Delete file snapshots
	_, err = tx.Exec("DELETE FROM file_snapshots WHERE plan_id = ?", planID)
	if err != nil {
		return serr.Wrap(err, "failed to delete snapshots")
	}
	
	// Delete executions
	_, err = tx.Exec("DELETE FROM task_executions WHERE plan_id = ?", planID)
	if err != nil {
		return serr.Wrap(err, "failed to delete executions")
	}
	
	// Finally, delete the plan itself
	_, err = tx.Exec("DELETE FROM task_plans WHERE id = ?", planID)
	if err != nil {
		return serr.Wrap(err, "failed to delete plan")
	}
	
	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return serr.Wrap(err, "failed to commit transaction")
	}
	
	return nil
}

// SaveExecution saves step execution result
func (t *TaskPlanDB) SaveExecution(planID, stepID string, result *StepResult) error {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return serr.Wrap(err, "failed to marshal result")
	}
	
	status := "success"
	if result.Error != "" {
		status = "failed"
	}
	
	query := `
		INSERT INTO task_executions (plan_id, step_id, status, result, duration_ms, retries, error_message, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	
	_, err = t.db.Exec(query, planID, stepID, status, string(resultJSON),
		result.Duration.Milliseconds(), result.Retries, result.Error)
	
	return serr.Wrap(err, "failed to save execution")
}

// GetExecutions retrieves all executions for a plan
func (t *TaskPlanDB) GetExecutions(planID string) ([]*TaskExecution, error) {
	query := `
		SELECT id, plan_id, step_id, status, result, started_at, completed_at,
		       duration_ms, retries, error_message
		FROM task_executions
		WHERE plan_id = ?
		ORDER BY started_at
	`
	
	rows, err := t.db.Query(query, planID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to query executions")
	}
	defer rows.Close()
	
	var executions []*TaskExecution
	for rows.Next() {
		var exec TaskExecution
		var resultJSON sql.NullString
		var completedAt sql.NullTime
		var durationMs, retries sql.NullInt64
		var errorMsg sql.NullString
		
		err := rows.Scan(
			&exec.ID, &exec.PlanID, &exec.StepID, &exec.Status,
			&resultJSON, &exec.StartedAt, &completedAt,
			&durationMs, &retries, &errorMsg,
		)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan execution")
		}
		
		// Set alias fields
		exec.StartTime = exec.StartedAt
		exec.EndTime = exec.CompletedAt
		
		if completedAt.Valid {
			exec.CompletedAt = &completedAt.Time
			exec.EndTime = &completedAt.Time
		}
		if durationMs.Valid {
			exec.DurationMs = int(durationMs.Int64)
		}
		if retries.Valid {
			exec.Retries = int(retries.Int64)
		}
		if errorMsg.Valid {
			exec.ErrorMessage = errorMsg.String
		}
		if resultJSON.Valid {
			exec.Result = json.RawMessage(resultJSON.String)
		}
		
		executions = append(executions, &exec)
	}
	
	return executions, nil
}

// SaveSnapshot saves a file snapshot
func (t *TaskPlanDB) SaveSnapshot(snapshot *FileSnapshot) error {
	if snapshot.SnapshotID == "" {
		snapshot.SnapshotID = uuid.New().String()
	}
	
	query := `
		INSERT INTO file_snapshots (snapshot_id, plan_id, checkpoint_id, file_path, content, hash, file_mode)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`
	
	_, err := t.db.Exec(query, snapshot.SnapshotID, snapshot.PlanID, snapshot.CheckpointID,
		snapshot.FilePath, snapshot.Content, snapshot.Hash, snapshot.FileMode)
	
	return serr.Wrap(err, "failed to save snapshot")
}

// GetSnapshots retrieves snapshots for a checkpoint
func (t *TaskPlanDB) GetSnapshots(checkpointID string) ([]*FileSnapshot, error) {
	query := `
		SELECT snapshot_id, plan_id, checkpoint_id, file_path, content, hash, file_mode, created_at
		FROM file_snapshots
		WHERE checkpoint_id = ?
	`
	
	rows, err := t.db.Query(query, checkpointID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to query snapshots")
	}
	defer rows.Close()
	
	var snapshots []*FileSnapshot
	for rows.Next() {
		var snap FileSnapshot
		var fileMode sql.NullInt64
		
		err := rows.Scan(
			&snap.SnapshotID, &snap.PlanID, &snap.CheckpointID,
			&snap.FilePath, &snap.Content, &snap.Hash,
			&fileMode, &snap.CreatedAt,
		)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan snapshot")
		}
		
		if fileMode.Valid {
			snap.FileMode = int(fileMode.Int64)
		}
		
		snapshots = append(snapshots, &snap)
	}
	
	return snapshots, nil
}

// GetSnapshotByHash retrieves a snapshot by its content hash
func (t *TaskPlanDB) GetSnapshotByHash(hash string) (*FileSnapshot, error) {
	var snap FileSnapshot
	var fileMode sql.NullInt64
	
	query := `
		SELECT snapshot_id, plan_id, checkpoint_id, file_path, content, hash, file_mode, created_at
		FROM file_snapshots
		WHERE hash = ?
		LIMIT 1
	`
	
	err := t.db.QueryRow(query, hash).Scan(
		&snap.SnapshotID, &snap.PlanID, &snap.CheckpointID,
		&snap.FilePath, &snap.Content, &snap.Hash,
		&fileMode, &snap.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, serr.New("snapshot not found")
		}
		return nil, serr.Wrap(err, "failed to get snapshot")
	}
	
	if fileMode.Valid {
		snap.FileMode = int(fileMode.Int64)
	}
	
	return &snap, nil
}

// SaveMetrics saves or updates task metrics
func (t *TaskPlanDB) SaveMetrics(metrics *TaskMetrics) error {
	toolsJSON, err := json.Marshal(metrics.ToolsUsed)
	if err != nil {
		return serr.Wrap(err, "failed to marshal tools used")
	}
	
	query := `
		INSERT INTO task_metrics (
			plan_id, total_steps, completed_steps, failed_steps, skipped_steps,
			total_duration_ms, avg_step_duration_ms, total_retries,
			context_files_used, tools_used
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(plan_id) DO UPDATE SET
			total_steps = excluded.total_steps,
			completed_steps = excluded.completed_steps,
			failed_steps = excluded.failed_steps,
			skipped_steps = excluded.skipped_steps,
			total_duration_ms = excluded.total_duration_ms,
			avg_step_duration_ms = excluded.avg_step_duration_ms,
			total_retries = excluded.total_retries,
			context_files_used = excluded.context_files_used,
			tools_used = excluded.tools_used,
			updated_at = CURRENT_TIMESTAMP
	`
	
	_, err = t.db.Exec(query, metrics.PlanID, metrics.TotalSteps, metrics.CompletedSteps,
		metrics.FailedSteps, metrics.SkippedSteps, metrics.TotalDurationMs,
		metrics.AvgStepDurationMs, metrics.TotalRetries, metrics.ContextFilesUsed,
		string(toolsJSON))
	
	return serr.Wrap(err, "failed to save metrics")
}

// GetMetrics retrieves metrics for a plan
func (t *TaskPlanDB) GetMetrics(planID string) (*TaskMetrics, error) {
	var metrics TaskMetrics
	var toolsJSON string
	
	query := `
		SELECT plan_id, total_steps, completed_steps, failed_steps, skipped_steps,
		       total_duration_ms, avg_step_duration_ms, total_retries,
		       context_files_used, tools_used, updated_at
		FROM task_metrics
		WHERE plan_id = ?
	`
	
	err := t.db.QueryRow(query, planID).Scan(
		&metrics.PlanID, &metrics.TotalSteps, &metrics.CompletedSteps,
		&metrics.FailedSteps, &metrics.SkippedSteps, &metrics.TotalDurationMs,
		&metrics.AvgStepDurationMs, &metrics.TotalRetries, &metrics.ContextFilesUsed,
		&toolsJSON, &metrics.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, serr.New("metrics not found")
		}
		return nil, serr.Wrap(err, "failed to get metrics")
	}
	
	if err := json.Unmarshal([]byte(toolsJSON), &metrics.ToolsUsed); err != nil {
		return nil, serr.Wrap(err, "failed to unmarshal tools used")
	}
	
	return &metrics, nil
}

// AddLog adds a log entry for a plan
func (t *TaskPlanDB) AddLog(log *TaskLog) error {
	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		return serr.Wrap(err, "failed to marshal metadata")
	}
	
	query := `
		INSERT INTO task_logs (plan_id, step_id, level, message, metadata)
		VALUES (?, ?, ?, ?, ?)
	`
	
	_, err = t.db.Exec(query, log.PlanID, log.StepID, log.Level, log.Message, string(metadataJSON))
	
	return serr.Wrap(err, "failed to add log")
}

// GetLogs retrieves logs for a plan
func (t *TaskPlanDB) GetLogs(planID string, level string) ([]*TaskLog, error) {
	query := `
		SELECT id, plan_id, step_id, level, message, metadata, created_at
		FROM task_logs
		WHERE plan_id = ?
	`
	args := []interface{}{planID}
	
	if level != "" {
		query += " AND level = ?"
		args = append(args, level)
	}
	
	query += " ORDER BY created_at"
	
	rows, err := t.db.Query(query, args...)
	if err != nil {
		return nil, serr.Wrap(err, "failed to query logs")
	}
	defer rows.Close()
	
	var logs []*TaskLog
	for rows.Next() {
		var log TaskLog
		var stepID sql.NullString
		var metadataJSON sql.NullString
		
		err := rows.Scan(
			&log.ID, &log.PlanID, &stepID, &log.Level,
			&log.Message, &metadataJSON, &log.CreatedAt,
		)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan log")
		}
		
		if stepID.Valid {
			log.StepID = stepID.String
		}
		if metadataJSON.Valid {
			if err := json.Unmarshal([]byte(metadataJSON.String), &log.Metadata); err != nil {
				return nil, serr.Wrap(err, "failed to unmarshal metadata")
			}
		}
		
		logs = append(logs, &log)
	}
	
	return logs, nil
}

// TaskExecution represents a step execution record
type TaskExecution struct {
	ID           int             `json:"id"`
	PlanID       string          `json:"plan_id"`
	StepID       string          `json:"step_id"`
	Status       string          `json:"status"`
	Result       json.RawMessage `json:"result,omitempty"`
	StartedAt    time.Time       `json:"started_at"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	StartTime    time.Time       `json:"start_time"`       // Alias for StartedAt
	EndTime      *time.Time      `json:"end_time,omitempty"` // Alias for CompletedAt
	DurationMs   int             `json:"duration_ms"`
	Retries      int             `json:"retries"`
	ErrorMessage string          `json:"error_message,omitempty"`
}

// FileSnapshot represents a file snapshot for rollback
type FileSnapshot struct {
	ID           int       `json:"id"`
	SnapshotID   string    `json:"snapshot_id"`
	PlanID       string    `json:"plan_id"`
	CheckpointID string    `json:"checkpoint_id"`
	FilePath     string    `json:"file_path"`
	Content      string    `json:"content"`
	Hash         string    `json:"hash"`
	FileMode     int       `json:"file_mode,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// TaskMetrics represents aggregated metrics for a task plan
type TaskMetrics struct {
	PlanID            string                 `json:"plan_id"`
	TotalSteps        int                    `json:"total_steps"`
	CompletedSteps    int                    `json:"completed_steps"`
	FailedSteps       int                    `json:"failed_steps"`
	SkippedSteps      int                    `json:"skipped_steps"`
	TotalDurationMs   int64                  `json:"total_duration_ms"`
	AvgStepDurationMs int64                  `json:"avg_step_duration_ms"`
	TotalRetries      int                    `json:"total_retries"`
	ContextFilesUsed  int                    `json:"context_files_used"`
	ToolsUsed         map[string]int         `json:"tools_used"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// TaskLog represents a log entry for a task plan
type TaskLog struct {
	ID        int                    `json:"id"`
	PlanID    string                 `json:"plan_id"`
	StepID    string                 `json:"step_id,omitempty"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}