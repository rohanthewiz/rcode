package web

import (
	"encoding/json"
	
	"rcode/db"

	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

// getSessionOpenFilesHandler returns currently open files for a session
func getSessionOpenFilesHandler(c rweb.Context) error {
	sessionId := c.Request().Param("id")
	if sessionId == "" {
		return c.WriteError(serr.New("session ID required"), 400)
	}

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Get open files from database (active only)
	sessionFiles, err := database.GetSessionFiles(sessionId, true)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get session files"), 500)
	}

	// Convert to response format
	var files []map[string]interface{}
	for _, sf := range sessionFiles {
		file := map[string]interface{}{
			"path":         sf.FilePath,
			"openedAt":     sf.OpenedAt,
			"lastViewedAt": sf.LastViewedAt,
			"isActive":     sf.IsActive,
		}
		files = append(files, file)
	}

	return c.WriteJSON(map[string]interface{}{
		"files": files,
		"count": len(files),
	})
}

// closeFileInSessionHandler marks a file as closed in a session
func closeFileInSessionHandler(c rweb.Context) error {
	sessionId := c.Request().Param("id")
	if sessionId == "" {
		return c.WriteError(serr.New("session ID required"), 400)
	}

	var req struct {
		Path string `json:"path"`
	}

	body := c.Request().Body()
	if err := json.Unmarshal(body, &req); err != nil {
		return c.WriteError(serr.New("invalid request body"), 400)
	}

	if req.Path == "" {
		return c.WriteError(serr.New("file path required"), 400)
	}

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Close file in database
	if err := database.CloseFileInSession(sessionId, req.Path); err != nil {
		return c.WriteError(serr.Wrap(err, "failed to close file"), 500)
	}

	return c.WriteJSON(map[string]interface{}{
		"status": "ok",
		"path":   req.Path,
	})
}