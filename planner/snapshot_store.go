package planner

import (
	"rcode/db"
)

// snapshotStoreAdapter adapts db.TaskPlanDB to implement SnapshotStore interface
type snapshotStoreAdapter struct {
	taskDB *db.TaskPlanDB
}

// NewSnapshotStoreAdapter creates a new adapter
func NewSnapshotStoreAdapter(taskDB *db.TaskPlanDB) SnapshotStore {
	return &snapshotStoreAdapter{taskDB: taskDB}
}

// SaveSnapshot implements SnapshotStore interface
func (s *snapshotStoreAdapter) SaveSnapshot(snapshot *FileSnapshot) error {
	// Convert to db.FileSnapshot
	dbSnapshot := &db.FileSnapshot{
		ID:           snapshot.ID,
		SnapshotID:   snapshot.SnapshotID,
		PlanID:       snapshot.PlanID,
		CheckpointID: snapshot.CheckpointID,
		FilePath:     snapshot.FilePath,
		Content:      snapshot.Content,
		Hash:         snapshot.Hash,
		FileMode:     snapshot.FileMode,
		CreatedAt:    snapshot.CreatedAt,
	}
	return s.taskDB.SaveSnapshot(dbSnapshot)
}

// GetSnapshots implements SnapshotStore interface
func (s *snapshotStoreAdapter) GetSnapshots(checkpointID string) ([]*FileSnapshot, error) {
	dbSnapshots, err := s.taskDB.GetSnapshots(checkpointID)
	if err != nil {
		return nil, err
	}
	
	// Convert from db.FileSnapshot to planner.FileSnapshot
	snapshots := make([]*FileSnapshot, len(dbSnapshots))
	for i, dbSnap := range dbSnapshots {
		snapshots[i] = &FileSnapshot{
			ID:           dbSnap.ID,
			SnapshotID:   dbSnap.SnapshotID,
			PlanID:       dbSnap.PlanID,
			CheckpointID: dbSnap.CheckpointID,
			FilePath:     dbSnap.FilePath,
			Content:      dbSnap.Content,
			Hash:         dbSnap.Hash,
			FileMode:     dbSnap.FileMode,
			CreatedAt:    dbSnap.CreatedAt,
		}
	}
	
	return snapshots, nil
}

// GetSnapshotByHash implements SnapshotStore interface
func (s *snapshotStoreAdapter) GetSnapshotByHash(hash string) (*FileSnapshot, error) {
	dbSnapshot, err := s.taskDB.GetSnapshotByHash(hash)
	if err != nil {
		return nil, err
	}
	
	if dbSnapshot == nil {
		return nil, nil
	}
	
	// Convert from db.FileSnapshot to planner.FileSnapshot
	return &FileSnapshot{
		ID:           dbSnapshot.ID,
		SnapshotID:   dbSnapshot.SnapshotID,
		PlanID:       dbSnapshot.PlanID,
		CheckpointID: dbSnapshot.CheckpointID,
		FilePath:     dbSnapshot.FilePath,
		Content:      dbSnapshot.Content,
		Hash:         dbSnapshot.Hash,
		FileMode:     dbSnapshot.FileMode,
		CreatedAt:    dbSnapshot.CreatedAt,
	}, nil
}