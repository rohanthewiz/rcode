package web

import "rcode/tools"

// FileEventNotifier implements the FileChangeNotifier interface
type FileEventNotifier struct{}

// NotifyFileChanged broadcasts when a file is changed
func (f *FileEventNotifier) NotifyFileChanged(path string, changeType string) {
	// TODO: This should ideally have the sessionID from the context
	// For now, broadcast to all sessions
	BroadcastFileChanged("", path, changeType)
}

// NotifyFileTreeUpdate broadcasts when the file tree needs refresh
func (f *FileEventNotifier) NotifyFileTreeUpdate(path string) {
	BroadcastFileTreeUpdate("", path)
}

// InitFileChangeNotifier initializes the file change notifier
func InitFileChangeNotifier() {
	notifier := &FileEventNotifier{}
	tools.SetFileChangeNotifier(notifier)
}
