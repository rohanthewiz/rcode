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
	planner := &Planner{
		tasks:          make(map[string]*TaskPlanner),
		executor:       NewStepExecutor(),
		analyzer:       NewTaskAnalyzer(),
		templates:      make(map[string]*TaskTemplate),
		logs:           make(map[string][]ExecutionLog),
		options:        options,
		contextManager: options.ContextManager,
	}
	
	// Initialize snapshot manager with the database
	if f.taskDB != nil {
		store := NewSnapshotStoreAdapter(f.taskDB)
		planner.snapshotManager = NewSnapshotManager(store)
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