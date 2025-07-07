package web

import (
	"encoding/json"
	"net/url"
	"strconv"

	"rcode/db"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/rweb"
	"github.com/rohanthewiz/serr"
)

// listPromptsHandler returns all initial prompts
func listPromptsHandler(c rweb.Context) error {
	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		logger.LogErr(err, "Failed to get database")
		return c.WriteError(err, 500)
	}

	// Check if we should only return active prompts
	queryString := c.Request().Query()
	params, _ := url.ParseQuery(queryString)
	activeOnly := params.Get("active") == "true"

	// List prompts from database
	prompts, err := database.GetAllInitialPrompts(activeOnly)
	if err != nil {
		logger.LogErr(err, "Failed to list prompts")
		return c.WriteError(serr.Wrap(err, "failed to list prompts"), 500)
	}

	return c.WriteJSON(prompts)
}

// getPromptHandler returns a single prompt by ID
func getPromptHandler(c rweb.Context) error {
	// Get prompt ID from URL
	idStr := c.Request().Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.LogErr(err, "Failed to get prompt ID from request")
		return c.WriteError(serr.Wrap(err, "invalid prompt ID"), 400)
	}

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		logger.LogErr(err, "Failed to get database")
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Get prompt from database
	prompt, err := database.GetInitialPrompt(id)
	if err != nil {
		err = serr.Wrap(err, "Failed to get Initial Prompt")
		logger.LogErr(err)
		return c.WriteError(err, 500)
	}

	return c.WriteJSON(prompt)
}

// createPromptHandler creates a new initial prompt
func createPromptHandler(c rweb.Context) error {
	// Parse request body
	body := c.Request().Body()
	var prompt db.InitialPrompt
	if err := json.Unmarshal(body, &prompt); err != nil {
		logger.LogErr(err, "Failed to unmarshal prompt from request")
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		logger.LogErr(err, "Failed to get database")
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Create prompt in database
	err = database.CreateInitialPrompt(&prompt)
	if err != nil {
		logger.LogErr(err, "Failed to create prompt", "name", prompt.Name)
		return c.WriteError(serr.Wrap(err, "failed to create prompt"), 500)
	}

	logger.Info("Created new prompt", "id", prompt.ID, "name", prompt.Name)

	return c.WriteJSON(prompt)
}

// updatePromptHandler updates an existing initial prompt
func updatePromptHandler(c rweb.Context) error {
	// Get prompt ID from URL
	idStr := c.Request().Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return c.WriteError(serr.Wrap(err, "invalid prompt ID"), 400)
	}

	// Parse request body into a map first to avoid ID conflicts
	body := c.Request().Body()
	var requestData map[string]interface{}
	if err := json.Unmarshal(body, &requestData); err != nil {
		logger.LogErr(err, "Failed to unmarshal prompt from request")
		return c.WriteError(serr.Wrap(err, "invalid request body"), 400)
	}

	// Build the prompt object from the request data
	prompt := db.InitialPrompt{
		ID:                  id, // Use ID from URL, not from body
		Name:                getStringFromMap(requestData, "name"),
		Description:         getStringFromMap(requestData, "description"),
		Content:             getStringFromMap(requestData, "content"),
		IncludesPermissions: getBoolFromMap(requestData, "includes_permissions"),
		IsActive:            getBoolFromMap(requestData, "is_active"),
		IsDefault:           getBoolFromMap(requestData, "is_default"),
	}

	// Handle permission template if provided
	if permTemplate, ok := requestData["permission_template"].(map[string]interface{}); ok {
		prompt.PermissionTemplate = permTemplate
	}

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		logger.LogErr(err, "Failed to get database")
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Update prompt in database
	err = database.UpdateInitialPrompt(&prompt)
	if err != nil {
		err = serr.Wrap(err, "Failed to update prompt")
		logger.LogErr(err)
		return c.WriteError(err, 500)
	}

	// Fetch the updated prompt from database to ensure we have all fields including timestamps
	updatedPrompt, err := database.GetInitialPrompt(id)
	if err != nil {
		logger.LogErr(err, "failed to get updated prompt")
		return c.WriteError(serr.Wrap(err, "failed to get updated prompt"), 500)
	}

	logger.Info("Updated prompt", "id", updatedPrompt.ID, "name", updatedPrompt.Name)

	// Return the updated prompt object instead of just success flag
	return c.WriteJSON(updatedPrompt)
}

// Helper functions to safely extract values from map
func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBoolFromMap(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

// deletePromptHandler deletes an initial prompt
func deletePromptHandler(c rweb.Context) error {
	// Get prompt ID from URL
	idStr := c.Request().Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		logger.LogErr(err, "Failed to get prompt ID from request")
		return c.WriteError(serr.Wrap(err, "invalid prompt ID"), 400)
	}

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		logger.LogErr(err, "Failed to get database")
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Delete prompt from database
	err = database.DeleteInitialPrompt(id)
	if err != nil {
		logger.LogErr(err, "Failed to delete Initial prompt")
		return c.WriteError(serr.Wrap(err, "failed to delete prompt"), 500)
	}

	logger.Info("Deleted prompt", "id", id)

	return c.WriteJSON(map[string]bool{"success": true})
}

// getSessionPromptsHandler returns the prompts associated with a session
func getSessionPromptsHandler(c rweb.Context) error {
	sessionID := c.Request().Param("id")

	// Get database instance
	database, err := db.GetDB()
	if err != nil {
		logger.LogErr(err, "Failed to get database")
		return c.WriteError(serr.Wrap(err, "failed to get database"), 500)
	}

	// Get prompts for session
	prompts, err := database.GetSessionInitialPrompts(sessionID)
	if err != nil {
		logger.LogErr(err, "Failed to get session prompts")
		return c.WriteError(serr.Wrap(err, "failed to get session prompts"), 500)
	}

	return c.WriteJSON(prompts)
}
