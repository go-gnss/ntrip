package admin

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"
)

// APIKey represents an API key
type APIKey struct {
	ID          int64     `json:"id"`
	Key         string    `json:"key,omitempty"` // Only included when creating a new key
	Name        string    `json:"name"`
	Permissions string    `json:"permissions"`
	Expires     time.Time `json:"expires"`
	Created     time.Time `json:"created"`
}

// GenerateAPIKey generates a new random API key
func GenerateAPIKey() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// CreateAPIKey creates a new API key
func (db *DB) CreateAPIKey(name, permissions string, expires time.Time) (*APIKey, error) {
	// Generate a new API key
	key, err := GenerateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Insert the API key into the database
	result, err := db.Exec(
		"INSERT INTO api_keys (key, name, permissions, expires) VALUES (?, ?, ?, ?)",
		key, name, permissions, expires,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert API key: %w", err)
	}

	// Get the ID of the inserted API key
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get API key ID: %w", err)
	}

	// Return the API key
	return &APIKey{
		ID:          id,
		Key:         key,
		Name:        name,
		Permissions: permissions,
		Expires:     expires,
		Created:     time.Now(),
	}, nil
}

// GetAPIKey gets an API key by its key
func (db *DB) GetAPIKey(key string) (*APIKey, error) {
	var apiKey APIKey
	err := db.QueryRow(
		"SELECT id, name, permissions, expires, created FROM api_keys WHERE key = ?",
		key,
	).Scan(&apiKey.ID, &apiKey.Name, &apiKey.Permissions, &apiKey.Expires, &apiKey.Created)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found")
		}
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	// Don't include the key in the response
	apiKey.Key = ""
	return &apiKey, nil
}

// ListAPIKeys lists all API keys
func (db *DB) ListAPIKeys() ([]APIKey, error) {
	rows, err := db.Query(
		"SELECT id, name, permissions, expires, created FROM api_keys",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []APIKey
	for rows.Next() {
		var apiKey APIKey
		err := rows.Scan(&apiKey.ID, &apiKey.Name, &apiKey.Permissions, &apiKey.Expires, &apiKey.Created)
		if err != nil {
			return nil, fmt.Errorf("failed to scan API key: %w", err)
		}
		apiKeys = append(apiKeys, apiKey)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating API keys: %w", err)
	}

	return apiKeys, nil
}

// DeleteAPIKey deletes an API key by its ID
func (db *DB) DeleteAPIKey(id int64) error {
	result, err := db.Exec("DELETE FROM api_keys WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found")
	}

	return nil
}

// ValidateAPIKey validates an API key and checks if it has the required permissions
func (db *DB) ValidateAPIKey(key, requiredPermission string) (bool, error) {
	var permissions string
	var expires time.Time

	err := db.QueryRow(
		"SELECT permissions, expires FROM api_keys WHERE key = ?",
		key,
	).Scan(&permissions, &expires)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to validate API key: %w", err)
	}

	// Check if the API key has expired
	if !expires.IsZero() && time.Now().After(expires) {
		return false, nil
	}

	// Check if the API key has the required permission
	if requiredPermission == "" {
		return true, nil
	}

	// Check if the API key has the required permission
	// Permissions are stored as a comma-separated list
	// The special permission "*" grants all permissions
	if permissions == "*" {
		return true, nil
	}

	// Check if the required permission is in the list
	for _, p := range parsePermissions(permissions) {
		if p == requiredPermission {
			return true, nil
		}
	}

	return false, nil
}

// parsePermissions parses a comma-separated list of permissions
func parsePermissions(permissions string) []string {
	if permissions == "" {
		return []string{}
	}
	return []string{permissions}
}
