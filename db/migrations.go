package db

import (
	"database/sql"
	"fmt"

	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	SQL         string
}

// migrations list all database migrations in order
var migrations = []Migration{
	{
		Version:     1,
		Description: "Create initial schema",
		SQL: `
			-- Create sessions table
			CREATE TABLE IF NOT EXISTS sessions (
				id TEXT PRIMARY KEY,
				title TEXT NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				initial_prompts TEXT[],
				model_preference TEXT,
				metadata JSON
			);

			-- Create messages table
			CREATE TABLE IF NOT EXISTS messages (
				id INTEGER PRIMARY KEY,
				session_id TEXT NOT NULL,
				role TEXT NOT NULL,
				content JSON NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				model TEXT,
				token_usage JSON,
				FOREIGN KEY (session_id) REFERENCES sessions(id)
			);
			CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id);

			-- Create tool_permissions table
			CREATE TABLE IF NOT EXISTS tool_permissions (
				id INTEGER PRIMARY KEY,
				session_id TEXT NOT NULL,
				tool_name TEXT NOT NULL,
				permission_type TEXT NOT NULL CHECK (permission_type IN ('allowed', 'denied', 'ask')),
				granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				expires_at TIMESTAMP,
				scope JSON,
				FOREIGN KEY (session_id) REFERENCES sessions(id)
			);
			CREATE UNIQUE INDEX IF NOT EXISTS idx_tool_permissions_session_tool ON tool_permissions(session_id, tool_name);

			-- Create tool_usage table
			CREATE TABLE IF NOT EXISTS tool_usage (
				id INTEGER PRIMARY KEY,
				session_id TEXT NOT NULL,
				tool_name TEXT NOT NULL,
				input JSON NOT NULL,
				output TEXT,
				executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				duration_ms INTEGER,
				error TEXT,
				FOREIGN KEY (session_id) REFERENCES sessions(id)
			);
			CREATE INDEX IF NOT EXISTS idx_tool_usage_session ON tool_usage(session_id);

			-- Create migrations table
			CREATE TABLE IF NOT EXISTS migrations (
				version INTEGER PRIMARY KEY,
				description TEXT NOT NULL,
				applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
		`,
	},
	{
		Version:     2,
		Description: "Fix auto-increment for message IDs",
		SQL: `
			-- Drop and recreate messages table with proper auto-increment
			DROP TABLE IF EXISTS messages;
			
			-- Create sequence for messages
			CREATE SEQUENCE IF NOT EXISTS messages_id_seq;
			
			-- Recreate messages table with sequence
			CREATE TABLE messages (
				id INTEGER PRIMARY KEY DEFAULT nextval('messages_id_seq'),
				session_id TEXT NOT NULL,
				role TEXT NOT NULL,
				content JSON NOT NULL,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				model TEXT,
				token_usage JSON,
				FOREIGN KEY (session_id) REFERENCES sessions(id)
			);
			CREATE INDEX idx_messages_session ON messages(session_id);
			
			-- Also fix tool_permissions and tool_usage tables
			DROP TABLE IF EXISTS tool_permissions;
			DROP TABLE IF EXISTS tool_usage;
			
			CREATE SEQUENCE IF NOT EXISTS tool_permissions_id_seq;
			CREATE SEQUENCE IF NOT EXISTS tool_usage_id_seq;
			
			CREATE TABLE tool_permissions (
				id INTEGER PRIMARY KEY DEFAULT nextval('tool_permissions_id_seq'),
				session_id TEXT NOT NULL,
				tool_name TEXT NOT NULL,
				permission_type TEXT NOT NULL CHECK (permission_type IN ('allowed', 'denied', 'ask')),
				granted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				expires_at TIMESTAMP,
				scope JSON,
				FOREIGN KEY (session_id) REFERENCES sessions(id)
			);
			CREATE UNIQUE INDEX idx_tool_permissions_session_tool ON tool_permissions(session_id, tool_name);
			
			CREATE TABLE tool_usage (
				id INTEGER PRIMARY KEY DEFAULT nextval('tool_usage_id_seq'),
				session_id TEXT NOT NULL,
				tool_name TEXT NOT NULL,
				input JSON NOT NULL,
				output TEXT,
				executed_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				duration_ms INTEGER,
				error TEXT,
				FOREIGN KEY (session_id) REFERENCES sessions(id)
			);
			CREATE INDEX idx_tool_usage_session ON tool_usage(session_id);
		`,
	},
	{
		Version:     3,
		Description: "Create initial prompts management table",
		SQL: `
			-- Create initial_prompts table for managing reusable prompts
			CREATE SEQUENCE IF NOT EXISTS initial_prompts_id_seq;
			
			CREATE TABLE IF NOT EXISTS initial_prompts (
				id INTEGER PRIMARY KEY DEFAULT nextval('initial_prompts_id_seq'),
				name TEXT NOT NULL UNIQUE,
				description TEXT,
				content TEXT NOT NULL,
				includes_permissions BOOLEAN DEFAULT false,
				permission_template JSON,
				is_active BOOLEAN DEFAULT true,
				is_default BOOLEAN DEFAULT false,
				created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
			);
			
			-- Create session_initial_prompts join table to link sessions with prompts
			CREATE TABLE IF NOT EXISTS session_initial_prompts (
				session_id TEXT NOT NULL,
				prompt_id INTEGER NOT NULL,
				applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (session_id) REFERENCES sessions(id),
				FOREIGN KEY (prompt_id) REFERENCES initial_prompts(id),
				PRIMARY KEY (session_id, prompt_id)
			);
			
			-- Insert default prompts
			INSERT INTO initial_prompts (name, description, content, includes_permissions, is_default) VALUES
			('permission_prompt', 'Default permission prompt', 'Always ask before creating or writing files or using any tools', true, true),
			('go_language_prompt', 'Prefer Go language', 'Use the Go language as much as possible', false, false);
		`,
	},
	{
		Version:     4,
		Description: "Add task planning tables",
		SQL: `
			-- Create task_plans table for storing AI task plans
			CREATE TABLE IF NOT EXISTS task_plans (
				id TEXT PRIMARY KEY,
				session_id TEXT NOT NULL,
				description TEXT NOT NULL,
				status TEXT NOT NULL CHECK (status IN ('pending', 'planning', 'executing', 'paused', 'completed', 'failed', 'cancelled')),
				steps JSON NOT NULL,
				context JSON,
				checkpoints JSON,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				completed_at TIMESTAMP,
				FOREIGN KEY (session_id) REFERENCES sessions(id)
			);
			CREATE INDEX idx_task_plans_session ON task_plans(session_id);
			CREATE INDEX idx_task_plans_status ON task_plans(status);

			-- Create task_executions table for tracking step executions
			CREATE SEQUENCE IF NOT EXISTS task_executions_id_seq;
			CREATE TABLE IF NOT EXISTS task_executions (
				id INTEGER PRIMARY KEY DEFAULT nextval('task_executions_id_seq'),
				plan_id TEXT NOT NULL,
				step_id TEXT NOT NULL,
				status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'success', 'failed', 'skipped')),
				result JSON,
				started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				completed_at TIMESTAMP,
				duration_ms INTEGER,
				retries INTEGER DEFAULT 0,
				error_message TEXT,
				FOREIGN KEY (plan_id) REFERENCES task_plans(id)
			);
			CREATE INDEX idx_task_executions_plan ON task_executions(plan_id);
			CREATE INDEX idx_task_executions_step ON task_executions(plan_id, step_id);

			-- Create file_snapshots table for rollback support
			CREATE SEQUENCE IF NOT EXISTS file_snapshots_id_seq;
			CREATE TABLE IF NOT EXISTS file_snapshots (
				id INTEGER PRIMARY KEY DEFAULT nextval('file_snapshots_id_seq'),
				snapshot_id TEXT NOT NULL UNIQUE,
				plan_id TEXT NOT NULL,
				checkpoint_id TEXT,
				file_path TEXT NOT NULL,
				content TEXT NOT NULL,
				hash TEXT NOT NULL,
				file_mode INTEGER,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (plan_id) REFERENCES task_plans(id)
			);
			CREATE INDEX idx_file_snapshots_plan ON file_snapshots(plan_id);
			CREATE INDEX idx_file_snapshots_checkpoint ON file_snapshots(checkpoint_id);
			CREATE INDEX idx_file_snapshots_hash ON file_snapshots(hash);

			-- Create task_metrics table for performance tracking
			CREATE TABLE IF NOT EXISTS task_metrics (
				plan_id TEXT PRIMARY KEY,
				total_steps INTEGER NOT NULL DEFAULT 0,
				completed_steps INTEGER NOT NULL DEFAULT 0,
				failed_steps INTEGER NOT NULL DEFAULT 0,
				skipped_steps INTEGER NOT NULL DEFAULT 0,
				total_duration_ms INTEGER,
				avg_step_duration_ms INTEGER,
				total_retries INTEGER DEFAULT 0,
				context_files_used INTEGER DEFAULT 0,
				tools_used JSON,
				updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (plan_id) REFERENCES task_plans(id)
			);

			-- Create task_logs table for detailed execution logging
			CREATE SEQUENCE IF NOT EXISTS task_logs_id_seq;
			CREATE TABLE IF NOT EXISTS task_logs (
				id INTEGER PRIMARY KEY DEFAULT nextval('task_logs_id_seq'),
				plan_id TEXT NOT NULL,
				step_id TEXT,
				level TEXT NOT NULL CHECK (level IN ('info', 'warning', 'error', 'debug')),
				message TEXT NOT NULL,
				metadata JSON,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY (plan_id) REFERENCES task_plans(id)
			);
			CREATE INDEX idx_task_logs_plan ON task_logs(plan_id);
			CREATE INDEX idx_task_logs_level ON task_logs(level);
		`,
	},
}

// Migrate runs all pending database migrations
func (db *DB) Migrate() error {
	// First, ensure migrations table exists
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS migrations (
			version INTEGER PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return serr.Wrap(err, "failed to create migrations table")
	}

	// Get current version
	var currentVersion int
	err = db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM migrations").Scan(&currentVersion)
	if err != nil {
		return serr.Wrap(err, "failed to get current migration version")
	}

	logger.Info("Current migration version", "version", currentVersion)

	// Apply pending migrations
	for _, migration := range migrations {
		if migration.Version <= currentVersion {
			continue
		}

		logger.Info("Applying migration", "version", migration.Version, "description", migration.Description)

		// Execute migration in a transaction
		err := db.Transaction(func(tx *sql.Tx) error {
			// Execute migration SQL
			if _, err := tx.Exec(migration.SQL); err != nil {
				return serr.Wrap(err, fmt.Sprintf("failed to execute migration %d", migration.Version))
			}

			// Record migration
			_, err := tx.Exec(
				"INSERT INTO migrations (version, description) VALUES (?, ?)",
				migration.Version, migration.Description,
			)
			if err != nil {
				return serr.Wrap(err, "failed to record migration")
			}

			return nil
		})

		if err != nil {
			return err
		}

		logger.Info("Migration applied successfully", "version", migration.Version)
	}

	return nil
}
