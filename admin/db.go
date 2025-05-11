package admin

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
)

// DB represents the database connection
type DB struct {
	*sql.DB
	logger logrus.FieldLogger
}

// NewDB creates a new database connection
func NewDB(dbPath string, logger logrus.FieldLogger) (*DB, error) {
	// Ensure directory exists
	if dbPath != ":memory:" {
		dir := filepath.Dir(dbPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath+"?_journal=WAL&_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection parameters
	db.SetMaxOpenConns(1) // SQLite only supports one writer at a time
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	// Create a new DB instance
	database := &DB{
		DB:     db,
		logger: logger,
	}

	// Initialize the database schema
	if err := database.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize database schema: %w", err)
	}

	return database, nil
}

// initSchema creates the database tables if they don't exist
func (db *DB) initSchema() error {
	// Create API keys table
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			permissions TEXT NOT NULL,
			expires DATETIME,
			created DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create api_keys table: %w", err)
	}

	// Create users table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			mounts_allowed TEXT,
			max_connections INTEGER DEFAULT 1,
			created DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Create mountpoints table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS mountpoints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			protocol TEXT DEFAULT 'NTRIP/2.0',
			status TEXT CHECK(status IN ('online','offline','maintenance')) DEFAULT 'online',
			last_active DATETIME
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create mountpoints table: %w", err)
	}

	return nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.DB.Close()
}
