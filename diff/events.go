package diff

// EventBroadcaster defines the interface for broadcasting diff-related events
type EventBroadcaster interface {
	BroadcastDiffAvailable(sessionID string, diffID int64, filePath string, stats interface{}, toolName string)
}

// globalBroadcaster holds the global event broadcaster instance
var globalBroadcaster EventBroadcaster

// SetEventBroadcaster sets the global event broadcaster
func SetEventBroadcaster(broadcaster EventBroadcaster) {
	globalBroadcaster = broadcaster
}

// BroadcastDiffAvailable broadcasts a diff available event using the global broadcaster
func BroadcastDiffAvailable(sessionID string, diffID int64, filePath string, stats interface{}, toolName string) {
	if globalBroadcaster != nil {
		globalBroadcaster.BroadcastDiffAvailable(sessionID, diffID, filePath, stats, toolName)
	}
}