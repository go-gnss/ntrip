package admin

import (
	"database/sql"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user
type User struct {
	ID             int64     `json:"id"`
	Username       string    `json:"username"`
	Password       string    `json:"password,omitempty"` // Only used for input
	MountsAllowed  string    `json:"mounts_allowed"`
	MaxConnections int       `json:"max_connections"`
	Created        time.Time `json:"created"`
}

// CreateUser creates a new user
func (db *DB) CreateUser(username, password, mountsAllowed string, maxConnections int) (*User, error) {
	// Hash the password
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Insert the user into the database
	result, err := db.Exec(
		"INSERT INTO users (username, password_hash, mounts_allowed, max_connections) VALUES (?, ?, ?, ?)",
		username, string(passwordHash), mountsAllowed, maxConnections,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	// Get the ID of the inserted user
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	// Return the user
	return &User{
		ID:             id,
		Username:       username,
		MountsAllowed:  mountsAllowed,
		MaxConnections: maxConnections,
		Created:        time.Now(),
	}, nil
}

// GetUser gets a user by username
func (db *DB) GetUser(username string) (*User, error) {
	var user User
	err := db.QueryRow(
		"SELECT id, username, mounts_allowed, max_connections, created FROM users WHERE username = ?",
		username,
	).Scan(&user.ID, &user.Username, &user.MountsAllowed, &user.MaxConnections, &user.Created)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// ListUsers lists all users
func (db *DB) ListUsers() ([]User, error) {
	rows, err := db.Query(
		"SELECT id, username, mounts_allowed, max_connections, created FROM users",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		err := rows.Scan(&user.ID, &user.Username, &user.MountsAllowed, &user.MaxConnections, &user.Created)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating users: %w", err)
	}

	return users, nil
}

// UpdateUser updates a user
func (db *DB) UpdateUser(username string, mountsAllowed *string, maxConnections *int, password *string) (*User, error) {
	// Check if the user exists
	_, err := db.GetUser(username)
	if err != nil {
		return nil, err
	}

	// Build the update query
	query := "UPDATE users SET"
	args := []interface{}{}

	// Add the fields to update
	if mountsAllowed != nil {
		query += " mounts_allowed = ?,"
		args = append(args, *mountsAllowed)
	}

	if maxConnections != nil {
		query += " max_connections = ?,"
		args = append(args, *maxConnections)
	}

	if password != nil {
		// Hash the password
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(*password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %w", err)
		}

		query += " password_hash = ?,"
		args = append(args, string(passwordHash))
	}

	// Remove the trailing comma
	query = query[:len(query)-1]

	// Add the WHERE clause
	query += " WHERE username = ?"
	args = append(args, username)

	// Execute the update
	_, err = db.Exec(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Get the updated user
	return db.GetUser(username)
}

// DeleteUser deletes a user by username
func (db *DB) DeleteUser(username string) error {
	result, err := db.Exec("DELETE FROM users WHERE username = ?", username)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// ValidateUserCredentials validates a user's credentials
func (db *DB) ValidateUserCredentials(username, password string) (bool, error) {
	var passwordHash string
	err := db.QueryRow(
		"SELECT password_hash FROM users WHERE username = ?",
		username,
	).Scan(&passwordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to validate user credentials: %w", err)
	}

	// Compare the password hash
	err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
	if err != nil {
		return false, nil
	}

	return true, nil
}
