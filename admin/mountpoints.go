package admin

import (
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Mountpoint represents a mountpoint
type Mountpoint struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Password   string    `json:"password,omitempty"` // Only used for input
	Protocol   string    `json:"protocol"`
	Status     string    `json:"status"`
	LastActive time.Time `json:"last_active,omitempty"`
}

// CreateMountpoint creates a new mountpoint
func (db *DB) CreateMountpoint(name, password, protocol string) (*Mountpoint, error) {
	// Hash the password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert the mountpoint into the database
	result, err := db.Exec(
		"INSERT INTO mountpoints (name, password_hash, protocol, status) VALUES (?, ?, ?, 'online')",
		name, string(passwordHash), protocol,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert mountpoint: %w", err)
	}

	// Get the ID of the inserted mountpoint
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get mountpoint ID: %w", err)
	}

	// Return the mountpoint
	return &Mountpoint{
		ID:       id,
		Name:     name,
		Protocol: protocol,
		Status:   "online",
	}, nil
}

// GetMountpoint gets a mountpoint by name
func (db *DB) GetMountpoint(name string) (*Mountpoint, error) {
	var mountpoint Mountpoint
	err := db.QueryRow(
		"SELECT id, name, protocol, status, last_active FROM mountpoints WHERE name = ?",
		name,
	).Scan(&mountpoint.ID, &mountpoint.Name, &mountpoint.Protocol, &mountpoint.Status, &mountpoint.LastActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("mountpoint not found")
		}
		return nil, fmt.Errorf("failed to get mountpoint: %w", err)
	}

	return &mountpoint, nil
}

// ListMountpoints lists all mountpoints
func (db *DB) ListMountpoints() ([]Mountpoint, error) {
	rows, err := db.Query(
		"SELECT id, name, protocol, status, last_active FROM mountpoints",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list mountpoints: %w", err)
	}
	defer rows.Close()

	var mountpoints []Mountpoint
	for rows.Next() {
		var mountpoint Mountpoint
		err := rows.Scan(&mountpoint.ID, &mountpoint.Name, &mountpoint.Protocol, &mountpoint.Status, &mountpoint.LastActive)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mountpoint: %w", err)
		}
		mountpoints = append(mountpoints, mountpoint)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating mountpoints: %w", err)
	}

	return mountpoints, nil
}

// UpdateMountpointStatus updates a mountpoint's status
func (db *DB) UpdateMountpointStatus(name, status string) (*Mountpoint, error) {
	// Validate status
	if status != "online" && status != "offline" && status != "maintenance" {
		return nil, fmt.Errorf("invalid status: %s", status)
	}

	// Update the mountpoint
	result, err := db.Exec(
		"UPDATE mountpoints SET status = ? WHERE name = ?",
		status, name,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update mountpoint status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("mountpoint not found")
	}

	// Get the updated mountpoint
	return db.GetMountpoint(name)
}

// UpdateMountpointLastActive updates a mountpoint's last active timestamp
func (db *DB) UpdateMountpointLastActive(name string) error {
	_, err := db.Exec(
		"UPDATE mountpoints SET last_active = CURRENT_TIMESTAMP, status = CASE WHEN status = 'offline' THEN 'online' ELSE status END WHERE name = ?",
		name,
	)
	if err != nil {
		return fmt.Errorf("failed to update mountpoint last active: %w", err)
	}

	return nil
}

// DeleteMountpoint deletes a mountpoint by name
func (db *DB) DeleteMountpoint(name string) error {
	result, err := db.Exec("DELETE FROM mountpoints WHERE name = ?", name)
	if err != nil {
		return fmt.Errorf("failed to delete mountpoint: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("mountpoint not found")
	}

	return nil
}

// ValidateMountpointCredentials validates a mountpoint's credentials
func (db *DB) ValidateMountpointCredentials(name, password string) (bool, error) {
	var passwordHash string
	err := db.QueryRow(
		"SELECT password_hash FROM mountpoints WHERE name = ?",
		name,
	).Scan(&passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to validate mountpoint credentials: %w", err)
	}

	// Compare the password hash
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		return false, nil
	}

	return true, nil
}

// MarkOfflineMountpoints marks mountpoints as offline if they haven't been active for a while
func (db *DB) MarkOfflineMountpoints(inactiveThreshold time.Duration) error {
	_, err := db.Exec(
		"UPDATE mountpoints SET status = 'offline' WHERE status = 'online' AND last_active < datetime('now', ?)",
		fmt.Sprintf("-%d minutes", int(inactiveThreshold.Minutes())),
	)
	if err != nil {
		return fmt.Errorf("failed to mark offline mountpoints: %w", err)
	}

	return nil
}
