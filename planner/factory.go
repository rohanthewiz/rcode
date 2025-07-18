package planner

import (
	"rcode/db"
)

// Factory for creating planners with proper dependencies
type PlannerFactory struct {
	taskDB *db.TaskPlanDB
}

// NewPlannerFactory creates a new planner factory
func NewPlannerFactory() *PlannerFactory {
	return &PlannerFactory{
		taskDB: db.GetTaskPlanDB(),
	}
}

// CreatePlanner creates a new planner instance with proper initialization
func (f *PlannerFactory) CreatePlanner(options PlannerOptions) *Planner {
	// Create analyzer with context support if available
	var analyzer *TaskAnalyzer
	if options.ContextManager != nil {
		analyzer = NewTaskAnalyzerWithContext(options.ContextManager)
	} else {
		analyzer = NewTaskAnalyzer()
	}

	stepExecutor := NewStepExecutor()
	metricsCollector := NewMetricsCollector()

	planner := &Planner{
		tasks:            make(map[string]*TaskPlanner),
		executor:         stepExecutor,
		analyzer:         analyzer,
		templates:        make(map[string]*TaskTemplate),
		logs:             make(map[string][]ExecutionLog),
		options:          options,
		contextManager:   options.ContextManager,
		metricsCollector: metricsCollector,
	}

	// Initialize parallel executor if concurrent steps are enabled
	if options.MaxConcurrentSteps > 1 {
		planner.parallelExecutor = NewParallelExecutor(stepExecutor, options.MaxConcurrentSteps)
	}

	// Initialize snapshot manager with the database
	if f.taskDB != nil {
		store := NewSnapshotStoreAdapter(f.taskDB)
		planner.snapshotManager = NewSnapshotManager(store)
		// Also set the database store for saving progress
		planner.SetDatabaseStore(f.taskDB)
		// Also set database store for metrics
		metricsCollector.SetDatabaseStore(f.taskDB)
	}

	return planner
}

// SetSnapshotStore allows external initialization of the snapshot store
func (p *Planner) SetSnapshotStore(store SnapshotStore) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if store != nil {
		p.snapshotManager = NewSnapshotManager(store)
	}
}
