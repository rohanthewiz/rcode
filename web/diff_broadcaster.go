package web

import "rcode/diff"

// diffBroadcaster implements the diff.EventBroadcaster interface
type diffBroadcaster struct{}

// BroadcastDiffAvailable implements the EventBroadcaster interface
func (db *diffBroadcaster) BroadcastDiffAvailable(sessionID string, diffID int64, filePath string, stats interface{}, toolName string) {
	BroadcastDiffAvailable(sessionID, diffID, filePath, stats, toolName)
}

// InitDiffBroadcaster sets up the diff event broadcaster
func InitDiffBroadcaster() {
	diff.SetEventBroadcaster(&diffBroadcaster{})
}