package db

import (
	"database/sql"
	"encoding/json"
	"time"

	duckdb "github.com/marcboeker/go-duckdb/v2"
	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// InitialPrompt represents a reusable initial prompt
type InitialPrompt struct {
	ID                  int                    `json:"id"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	Content             string                 `json:"content"`
	IncludesPermissions bool                   `json:"includes_permissions"`
	PermissionTemplate  map[string]interface{} `json:"permission_template,omitempty"`
	IsActive            bool                   `json:"is_active"`
	IsDefault           bool                   `json:"is_default"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// CreateInitialPrompt creates a new initial prompt
func (db *DB) CreateInitialPrompt(prompt *InitialPrompt) error {
	// Serialize permission template to JSON
	var permTemplateJSON []byte
	if prompt.PermissionTemplate != nil && len(prompt.PermissionTemplate) > 0 {
		var err error
		permTemplateJSON, err = json.Marshal(prompt.PermissionTemplate)
		if err != nil {
			return serr.Wrap(err, "failed to serialize permission template")
		}
	} else {
		// Use empty JSON object for empty permission template
		permTemplateJSON = []byte("{}")
	}

	// Insert the prompt
	err := db.QueryRow(`
		INSERT INTO initial_prompts (name, description, content, includes_permissions, permission_template, is_active, is_default)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, created_at, updated_at
	`, prompt.Name, prompt.Description, prompt.Content, prompt.IncludesPermissions,
		permTemplateJSON, prompt.IsActive, prompt.IsDefault).
		Scan(&prompt.ID, &prompt.CreatedAt, &prompt.UpdatedAt)

	if err != nil {
		return serr.Wrap(err, "failed to create initial prompt")
	}

	return nil
}

// GetInitialPrompt retrieves a single initial prompt by ID
func (db *DB) GetInitialPrompt(id int) (*InitialPrompt, error) {
	prompt := &InitialPrompt{}
	var permTemplateJSON duckdb.Composite[map[string]interface{}]

	err := db.QueryRow(`
		SELECT id, name, description, content, includes_permissions, permission_template, 
		       is_active, is_default, created_at, updated_at
		FROM initial_prompts
		WHERE id = ?
	`, id).Scan(&prompt.ID, &prompt.Name, &prompt.Description, &prompt.Content,
		&prompt.IncludesPermissions, &permTemplateJSON, &prompt.IsActive,
		&prompt.IsDefault, &prompt.CreatedAt, &prompt.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, serr.New("initial prompt not found")
		}
		return nil, serr.Wrap(err, "failed to get initial prompt")
	}

	// Get the permission template from the composite type
	if permTemplateJSON.Get() != nil {
		prompt.PermissionTemplate = permTemplateJSON.Get()
	}

	return prompt, nil
}

// GetAllInitialPrompts retrieves all initial prompts
func (db *DB) GetAllInitialPrompts(activeOnly bool) ([]*InitialPrompt, error) {
	query := `
		SELECT id, name, description, content, includes_permissions, permission_template, 
		       is_active, is_default, created_at, updated_at
		FROM initial_prompts
	`
	if activeOnly {
		query += " WHERE is_active = true"
	}
	query += " ORDER BY is_default DESC, name ASC"

	rows, err := db.Query(query)
	if err != nil {
		return nil, serr.Wrap(err, "failed to query initial prompts")
	}
	defer rows.Close()

	var prompts []*InitialPrompt
	for rows.Next() {
		prompt := &InitialPrompt{}
		var permTemplateJSON duckdb.Composite[map[string]interface{}]

		err := rows.Scan(&prompt.ID, &prompt.Name, &prompt.Description, &prompt.Content,
			&prompt.IncludesPermissions, &permTemplateJSON, &prompt.IsActive,
			&prompt.IsDefault, &prompt.CreatedAt, &prompt.UpdatedAt)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan initial prompt")
		}

		// Get the permission template from the composite type
		if permTemplateJSON.Get() != nil {
			prompt.PermissionTemplate = permTemplateJSON.Get()
		}

		prompts = append(prompts, prompt)
	}

	return prompts, nil
}

// UpdateInitialPrompt updates an existing initial prompt
func (db *DB) UpdateInitialPrompt(prompt *InitialPrompt) error {
	// Debug
	// prompts, err := db.GetAllInitialPrompts(false) // Debug
	// if err != nil {
	// 	return serr.Wrap(err, "failed to get all initial prompts")
	// }
	//
	// fmt.Println("***prompts***")
	// for _, prompt := range prompts {
	// 	fmt.Println(prompt)
	// }

	// Serialize permission template to JSON
	var permTemplateJSON []byte
	if prompt.PermissionTemplate != nil && len(prompt.PermissionTemplate) > 0 {
		var err error
		permTemplateJSON, err = json.Marshal(prompt.PermissionTemplate)
		if err != nil {
			return serr.Wrap(err, "failed to serialize permission template")
		}
	} else {
		// Use empty JSON object for empty permission template
		permTemplateJSON = []byte("{}")
	}

	// Update the prompt using UPDATE statement
	result, err := db.Exec(`
		UPDATE initial_prompts 
		SET name = ?, description = ?, content = ?, includes_permissions = ?,
		    permission_template = ?, is_active = ?, is_default = ?, updated_at = ?
		WHERE id = ?
	`, prompt.Name, prompt.Description, prompt.Content, prompt.IncludesPermissions,
		permTemplateJSON, prompt.IsActive, prompt.IsDefault, time.Now(), prompt.ID)

	if err != nil {
		return serr.Wrap(err, "failed to update initial prompt", "table", "initial_prompts")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return serr.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return serr.New("initial prompt not found")
	}

	// Update the prompt's updated_at field to reflect the change
	prompt.UpdatedAt = time.Now()

	logger.Debug("Updated initial prompt", "id", prompt.ID, "name", prompt.Name)

	return nil
}

// DeleteInitialPrompt deletes an initial prompt
func (db *DB) DeleteInitialPrompt(id int) error {
	result, err := db.Exec("DELETE FROM initial_prompts WHERE id = ?", id)
	if err != nil {
		return serr.Wrap(err, "failed to delete initial prompt")
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return serr.Wrap(err, "failed to get rows affected")
	}

	if rowsAffected == 0 {
		return serr.New("initial prompt not found")
	}

	return nil
}

// GetDefaultInitialPrompts returns all default prompts
func (db *DB) GetDefaultInitialPrompts() ([]*InitialPrompt, error) {
	return db.getInitialPromptsByCondition("WHERE is_default = true AND is_active = true")
}

// GetSessionInitialPrompts returns the initial prompts associated with a session
func (db *DB) GetSessionInitialPrompts(sessionID string) ([]*InitialPrompt, error) {
	query := `
		SELECT ip.id, ip.name, ip.description, ip.content, ip.includes_permissions, 
		       ip.permission_template, ip.is_active, ip.is_default, ip.created_at, ip.updated_at
		FROM initial_prompts ip
		JOIN session_initial_prompts sip ON ip.id = sip.prompt_id
		WHERE sip.session_id = ?
		ORDER BY sip.applied_at
	`

	rows, err := db.Query(query, sessionID)
	if err != nil {
		return nil, serr.Wrap(err, "failed to query session initial prompts")
	}
	defer rows.Close()

	var prompts []*InitialPrompt
	for rows.Next() {
		prompt := &InitialPrompt{}
		var permTemplateJSON duckdb.Composite[map[string]interface{}]

		err := rows.Scan(&prompt.ID, &prompt.Name, &prompt.Description, &prompt.Content,
			&prompt.IncludesPermissions, &permTemplateJSON, &prompt.IsActive,
			&prompt.IsDefault, &prompt.CreatedAt, &prompt.UpdatedAt)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan initial prompt")
		}

		// Get the permission template from the composite type
		if permTemplateJSON.Get() != nil {
			prompt.PermissionTemplate = permTemplateJSON.Get()
		}

		prompts = append(prompts, prompt)
	}

	return prompts, nil
}

// AssociatePromptsWithSession links initial prompts to a session
func (db *DB) AssociatePromptsWithSession(sessionID string, promptIDs []int) error {
	return db.Transaction(func(tx *sql.Tx) error {
		// First, remove any existing associations
		_, err := tx.Exec("DELETE FROM session_initial_prompts WHERE session_id = ?", sessionID)
		if err != nil {
			return serr.Wrap(err, "failed to remove existing prompt associations")
		}

		// Insert new associations
		for _, promptID := range promptIDs {
			_, err := tx.Exec(`
				INSERT INTO session_initial_prompts (session_id, prompt_id)
				VALUES (?, ?)
			`, sessionID, promptID)
			if err != nil {
				return serr.Wrap(err, "failed to associate prompt with session")
			}
		}

		return nil
	})
}

// Helper function to get prompts by condition
func (db *DB) getInitialPromptsByCondition(condition string) ([]*InitialPrompt, error) {
	query := `
		SELECT id, name, description, content, includes_permissions, permission_template, 
		       is_active, is_default, created_at, updated_at
		FROM initial_prompts
	` + condition + " ORDER BY name ASC"

	rows, err := db.Query(query)
	if err != nil {
		return nil, serr.Wrap(err, "failed to query initial prompts")
	}
	defer rows.Close()

	var prompts []*InitialPrompt
	for rows.Next() {
		prompt := &InitialPrompt{}
		var permTemplateJSON duckdb.Composite[map[string]interface{}]

		err := rows.Scan(&prompt.ID, &prompt.Name, &prompt.Description, &prompt.Content,
			&prompt.IncludesPermissions, &permTemplateJSON, &prompt.IsActive,
			&prompt.IsDefault, &prompt.CreatedAt, &prompt.UpdatedAt)
		if err != nil {
			return nil, serr.Wrap(err, "failed to scan initial prompt")
		}

		// Get the permission template from the composite type
		if permTemplateJSON.Get() != nil {
			prompt.PermissionTemplate = permTemplateJSON.Get()
		}

		prompts = append(prompts, prompt)
	}

	return prompts, nil
}
