package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/rohanthewiz/logger"
	"github.com/rohanthewiz/serr"
)

// DB represents the database connection
type DB struct {
	conn *sql.DB
	path string
}

// singleton instance
var instance *DB

// GetDB returns the database instance, creating it if necessary
func GetDB() (*DB, error) {
	if instance != nil {
		return instance, nil
	}

	// Get database path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, serr.Wrap(err, "failed to get home directory")
	}

	dataDir := filepath.Join(homeDir, ".local", "share", "rcode")
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, serr.Wrap(err, "failed to create data directory")
	}

	dbPath := filepath.Join(dataDir, "rcode.db")
	
	// Open database connection
	conn, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, serr.Wrap(err, "failed to open database")
	}

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, serr.Wrap(err, "failed to ping database")
	}

	instance = &DB{
		conn: conn,
		path: dbPath,
	}

	logger.Info("Database connected", "path", dbPath)

	// Run migrations
	if err := instance.Migrate(); err != nil {
		return nil, serr.Wrap(err, "failed to run migrations")
	}

	return instance, nil
}

// Conn returns the underlying database connection
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// Close closes the database connection
func (db *DB) Close() error {
	if db.conn != nil {
		return db.conn.Close()
	}
	return nil
}

// Transaction executes a function within a database transaction
func (db *DB) Transaction(fn func(*sql.Tx) error) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return serr.Wrap(err, "failed to begin transaction")
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return serr.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// Query executes a query that returns rows
func (db *DB) Query(query string, args ...interface{}) (*sql.Rows, error) {
	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, serr.Wrap(err, fmt.Sprintf("query failed: %s", query))
	}
	return rows, nil
}

// QueryRow executes a query that returns a single row
func (db *DB) QueryRow(query string, args ...interface{}) *sql.Row {
	return db.conn.QueryRow(query, args...)
}

// Exec executes a query that doesn't return rows
func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	result, err := db.conn.Exec(query, args...)
	if err != nil {
		return nil, serr.Wrap(err, fmt.Sprintf("exec failed: %s", query))
	}
	return result, nil
}