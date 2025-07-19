package planner

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/rohanthewiz/serr"
)

// ExecutionMetrics tracks metrics for task plan execution
type ExecutionMetrics struct {
	PlanID         string                 `json:"plan_id"`
	TotalSteps     int                    `json:"total_steps"`
	CompletedSteps int                    `json:"completed_steps"`
	FailedSteps    int                    `json:"failed_steps"`
	SkippedSteps   int                    `json:"skipped_steps"`
	TotalDuration  time.Duration          `json:"total_duration"`
	StepMetrics    map[string]*StepMetric `json:"step_metrics"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        *time.Time             `json:"end_time,omitempty"`
	ParallelInfo   *ParallelExecutionInfo `json:"parallel_info,omitempty"`
	mu             sync.RWMutex
}

// StepMetric tracks metrics for a single step execution
type StepMetric struct {
	StepID        string        `json:"step_id"`
	Tool          string        `json:"tool"`
	Duration      time.Duration `json:"duration"`
	RetryCount    int           `json:"retry_count"`
	MemoryBefore  uint64        `json:"memory_before"`
	MemoryAfter   uint64        `json:"memory_after"`
	MemoryDelta   int64         `json:"memory_delta"`
	Success       bool          `json:"success"`
	Error         string        `json:"error,omitempty"`
	FilesModified []string      `json:"files_modified,omitempty"`
	BytesWritten  int64         `json:"bytes_written"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       *time.Time    `json:"end_time,omitempty"`
}

// ParallelExecutionInfo contains information about parallel execution
type ParallelExecutionInfo struct {
	MaxConcurrency   int        `json:"max_concurrency"`
	ActualSpeedup    float64    `json:"actual_speedup"`
	EstimatedSpeedup float64    `json:"estimated_speedup"`
	ParallelGroups   [][]string `json:"parallel_groups"`
	CriticalPath     []string   `json:"critical_path"`
}

// MetricsCollector collects execution metrics
type MetricsCollector struct {
	metrics map[string]*ExecutionMetrics
	mu      sync.RWMutex
	dbStore interface{} // Will be *db.TaskPlanDB but avoid import cycle
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*ExecutionMetrics),
	}
}

// StartPlanExecution starts tracking metrics for a plan
func (mc *MetricsCollector) StartPlanExecution(planID string, totalSteps int) *ExecutionMetrics {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics := &ExecutionMetrics{
		PlanID:      planID,
		TotalSteps:  totalSteps,
		StepMetrics: make(map[string]*StepMetric),
		StartTime:   time.Now(),
	}

	mc.metrics[planID] = metrics
	return metrics
}

// StartStepExecution starts tracking metrics for a step
func (mc *MetricsCollector) StartStepExecution(planID, stepID, tool string) (*StepMetric, error) {
	mc.mu.RLock()
	metrics, exists := mc.metrics[planID]
	mc.mu.RUnlock()

	if !exists {
		return nil, serr.New("metrics not found for plan")
	}

	// Get current memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	stepMetric := &StepMetric{
		StepID:       stepID,
		Tool:         tool,
		StartTime:    time.Now(),
		MemoryBefore: memStats.Alloc,
	}

	metrics.mu.Lock()
	metrics.StepMetrics[stepID] = stepMetric
	metrics.mu.Unlock()

	return stepMetric, nil
}

// EndStepExecution ends tracking for a step and updates metrics
func (mc *MetricsCollector) EndStepExecution(planID, stepID string, success bool, err error) error {
	mc.mu.RLock()
	metrics, exists := mc.metrics[planID]
	mc.mu.RUnlock()

	if !exists {
		return serr.New("metrics not found for plan")
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	stepMetric, exists := metrics.StepMetrics[stepID]
	if !exists {
		return serr.New("step metric not found")
	}

	// Update end time
	endTime := time.Now()
	stepMetric.EndTime = &endTime
	stepMetric.Duration = endTime.Sub(stepMetric.StartTime)
	stepMetric.Success = success

	// Get final memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	stepMetric.MemoryAfter = memStats.Alloc
	stepMetric.MemoryDelta = int64(stepMetric.MemoryAfter) - int64(stepMetric.MemoryBefore)

	// Update error if failed
	if err != nil {
		stepMetric.Error = err.Error()
	}

	// Update plan-level metrics
	if success {
		metrics.CompletedSteps++
	} else {
		metrics.FailedSteps++
	}

	return nil
}

// RecordRetry records a retry attempt for a step
func (mc *MetricsCollector) RecordRetry(planID, stepID string) error {
	mc.mu.RLock()
	metrics, exists := mc.metrics[planID]
	mc.mu.RUnlock()

	if !exists {
		return serr.New("metrics not found for plan")
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	stepMetric, exists := metrics.StepMetrics[stepID]
	if !exists {
		return serr.New("step metric not found")
	}

	stepMetric.RetryCount++
	return nil
}

// RecordStepSkipped records that a step was skipped
func (mc *MetricsCollector) RecordStepSkipped(planID, stepID string) error {
	mc.mu.RLock()
	metrics, exists := mc.metrics[planID]
	mc.mu.RUnlock()

	if !exists {
		return serr.New("metrics not found for plan")
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	metrics.SkippedSteps++
	return nil
}

// RecordFileModification records that a step modified files
func (mc *MetricsCollector) RecordFileModification(planID, stepID string, files []string, bytesWritten int64) error {
	mc.mu.RLock()
	metrics, exists := mc.metrics[planID]
	mc.mu.RUnlock()

	if !exists {
		return serr.New("metrics not found for plan")
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	stepMetric, exists := metrics.StepMetrics[stepID]
	if !exists {
		return serr.New("step metric not found")
	}

	stepMetric.FilesModified = files
	stepMetric.BytesWritten = bytesWritten
	return nil
}

// EndPlanExecution ends tracking for a plan and calculates final metrics
func (mc *MetricsCollector) EndPlanExecution(planID string) (*ExecutionMetrics, error) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics, exists := mc.metrics[planID]
	if !exists {
		return nil, serr.New("metrics not found for plan")
	}

	// Update end time and duration
	endTime := time.Now()
	metrics.EndTime = &endTime
	metrics.TotalDuration = endTime.Sub(metrics.StartTime)

	// Save to database if available
	if mc.dbStore != nil {
		if err := mc.saveMetrics(planID, metrics); err != nil {
			// Log error but don't fail
			fmt.Printf("Failed to save metrics: %v\n", err)
		}
	}

	// Remove from active metrics
	delete(mc.metrics, planID)

	return metrics, nil
}

// SetParallelExecutionInfo sets information about parallel execution
func (mc *MetricsCollector) SetParallelExecutionInfo(planID string, info *ParallelExecutionInfo) error {
	mc.mu.RLock()
	metrics, exists := mc.metrics[planID]
	mc.mu.RUnlock()

	if !exists {
		return serr.New("metrics not found for plan")
	}

	metrics.mu.Lock()
	defer metrics.mu.Unlock()

	metrics.ParallelInfo = info

	// Calculate actual speedup if execution is complete
	if metrics.EndTime != nil && info != nil && len(info.CriticalPath) > 0 {
		// Sum durations of critical path steps
		var criticalPathDuration time.Duration
		for _, stepID := range info.CriticalPath {
			if stepMetric, exists := metrics.StepMetrics[stepID]; exists && stepMetric.Duration > 0 {
				criticalPathDuration += stepMetric.Duration
			}
		}

		if criticalPathDuration > 0 {
			info.ActualSpeedup = float64(metrics.TotalDuration) / float64(criticalPathDuration)
		}
	}

	return nil
}

// GetMetrics retrieves metrics for a plan
func (mc *MetricsCollector) GetMetrics(planID string) (*ExecutionMetrics, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics, exists := mc.metrics[planID]
	if !exists {
		return nil, serr.New("metrics not found for plan")
	}

	// Return a copy to avoid race conditions
	metricsCopy := *metrics
	metricsCopy.StepMetrics = make(map[string]*StepMetric)

	metrics.mu.RLock()
	for k, v := range metrics.StepMetrics {
		stepCopy := *v
		metricsCopy.StepMetrics[k] = &stepCopy
	}
	metrics.mu.RUnlock()

	return &metricsCopy, nil
}

// SetDatabaseStore sets the database store for persisting metrics
func (mc *MetricsCollector) SetDatabaseStore(store interface{}) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.dbStore = store
}

// saveMetrics saves metrics to the database
func (mc *MetricsCollector) saveMetrics(planID string, metrics *ExecutionMetrics) error {
	if mc.dbStore == nil {
		return nil
	}

	// Use type assertion to call SaveMetrics method
	type dbInterface interface {
		SaveMetrics(planID string, metrics interface{}) error
	}

	db, ok := mc.dbStore.(dbInterface)
	if !ok {
		return nil // Database doesn't support SaveMetrics
	}

	return db.SaveMetrics(planID, metrics)
}

// GenerateMetricsReport generates a human-readable metrics report
func GenerateMetricsReport(metrics *ExecutionMetrics) string {
	report := fmt.Sprintf("=== Execution Metrics Report ===\n")
	report += fmt.Sprintf("Plan ID: %s\n", metrics.PlanID)
	report += fmt.Sprintf("Duration: %s\n", metrics.TotalDuration)
	report += fmt.Sprintf("Steps: %d total, %d completed, %d failed, %d skipped\n",
		metrics.TotalSteps, metrics.CompletedSteps, metrics.FailedSteps, metrics.SkippedSteps)

	if metrics.ParallelInfo != nil {
		report += fmt.Sprintf("\n=== Parallel Execution ===\n")
		report += fmt.Sprintf("Max Concurrency: %d\n", metrics.ParallelInfo.MaxConcurrency)
		report += fmt.Sprintf("Estimated Speedup: %.2fx\n", metrics.ParallelInfo.EstimatedSpeedup)
		if metrics.ParallelInfo.ActualSpeedup > 0 {
			report += fmt.Sprintf("Actual Speedup: %.2fx\n", metrics.ParallelInfo.ActualSpeedup)
		}
		report += fmt.Sprintf("Critical Path: %v\n", metrics.ParallelInfo.CriticalPath)
	}

	report += fmt.Sprintf("\n=== Step Details ===\n")
	var totalMemoryDelta int64
	var totalBytesWritten int64

	for _, step := range metrics.StepMetrics {
		status := "✓"
		if !step.Success {
			status = "✗"
		}
		report += fmt.Sprintf("\n%s Step: %s (Tool: %s)\n", status, step.StepID, step.Tool)
		report += fmt.Sprintf("  Duration: %s\n", step.Duration)
		if step.RetryCount > 0 {
			report += fmt.Sprintf("  Retries: %d\n", step.RetryCount)
		}
		if step.MemoryDelta != 0 {
			report += fmt.Sprintf("  Memory: %+d bytes\n", step.MemoryDelta)
			totalMemoryDelta += step.MemoryDelta
		}
		if len(step.FilesModified) > 0 {
			report += fmt.Sprintf("  Files Modified: %v\n", step.FilesModified)
			report += fmt.Sprintf("  Bytes Written: %d\n", step.BytesWritten)
			totalBytesWritten += step.BytesWritten
		}
		if step.Error != "" {
			report += fmt.Sprintf("  Error: %s\n", step.Error)
		}
	}

	report += fmt.Sprintf("\n=== Summary ===\n")
	report += fmt.Sprintf("Total Memory Delta: %+d bytes\n", totalMemoryDelta)
	report += fmt.Sprintf("Total Bytes Written: %d\n", totalBytesWritten)

	return report
}
