package admin

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestUsersDB(t *testing.T) {
	// Skip the test since it requires CGO
	t.Skip("Skipping test that requires CGO")

	// Create an in-memory database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Initialize the schema
	_, err = db.Exec(`
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			mounts_allowed TEXT,
			max_connections INTEGER DEFAULT 1,
			created DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	// Create a DB instance
	dbInstance := &DB{DB: db}

	// Test creating a user
	t.Run("CreateUser", func(t *testing.T) {
		// Create a user
		username := "testuser"
		password := "testpassword"
		mountsAllowed := "MOUNT1,MOUNT2"
		maxConnections := 5

		user, err := dbInstance.CreateUser(username, password, mountsAllowed, maxConnections)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, username, user.Username)
		assert.Equal(t, mountsAllowed, user.MountsAllowed)
		assert.Equal(t, maxConnections, user.MaxConnections)

		// Verify the password was hashed
		var passwordHash string
		err = db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&passwordHash)
		require.NoError(t, err)
		assert.NotEqual(t, password, passwordHash)

		// Verify the hash is valid
		err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
		assert.NoError(t, err)
	})

	// Test getting a user
	t.Run("GetUser", func(t *testing.T) {
		// Create a user first
		username := "getuser"
		password := "getpassword"
		mountsAllowed := "MOUNT3,MOUNT4"
		maxConnections := 3

		_, err := dbInstance.CreateUser(username, password, mountsAllowed, maxConnections)
		require.NoError(t, err)

		// Get the user
		user, err := dbInstance.GetUser(username)
		require.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, username, user.Username)
		assert.Equal(t, mountsAllowed, user.MountsAllowed)
		assert.Equal(t, maxConnections, user.MaxConnections)

		// Try to get a non-existent user
		_, err = dbInstance.GetUser("nonexistentuser")
		assert.Error(t, err)
	})

	// Test listing users
	t.Run("ListUsers", func(t *testing.T) {
		// Create a few users
		_, err := dbInstance.CreateUser("listuser1", "password1", "MOUNT1", 1)
		require.NoError(t, err)
		_, err = dbInstance.CreateUser("listuser2", "password2", "MOUNT2", 2)
		require.NoError(t, err)

		// List the users
		users, err := dbInstance.ListUsers()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(users), 2) // At least the two we just created
	})

	// Test updating a user
	t.Run("UpdateUser", func(t *testing.T) {
		// Create a user first
		username := "updateuser"
		password := "updatepassword"
		mountsAllowed := "MOUNT5,MOUNT6"
		maxConnections := 4

		_, err := dbInstance.CreateUser(username, password, mountsAllowed, maxConnections)
		require.NoError(t, err)

		// Update the user's mounts
		newMountsAllowed := "MOUNT7,MOUNT8"
		user, err := dbInstance.UpdateUser(username, &newMountsAllowed, nil, nil)
		require.NoError(t, err)
		assert.Equal(t, newMountsAllowed, user.MountsAllowed)
		assert.Equal(t, maxConnections, user.MaxConnections)

		// Update the user's max connections
		newMaxConnections := 10
		user, err = dbInstance.UpdateUser(username, nil, &newMaxConnections, nil)
		require.NoError(t, err)
		assert.Equal(t, newMountsAllowed, user.MountsAllowed)
		assert.Equal(t, newMaxConnections, user.MaxConnections)

		// Update the user's password
		newPassword := "newpassword"
		_, err = dbInstance.UpdateUser(username, nil, nil, &newPassword)
		require.NoError(t, err)

		// Verify the password was updated
		var passwordHash string
		err = db.QueryRow("SELECT password_hash FROM users WHERE username = ?", username).Scan(&passwordHash)
		require.NoError(t, err)

		// Verify the new hash is valid
		err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(newPassword))
		assert.NoError(t, err)

		// Try to update a non-existent user
		_, err = dbInstance.UpdateUser("nonexistentuser", &newMountsAllowed, nil, nil)
		assert.Error(t, err)
	})

	// Test validating user credentials
	t.Run("ValidateUserCredentials", func(t *testing.T) {
		// Create a user
		username := "validateuser"
		password := "validatepassword"

		_, err := dbInstance.CreateUser(username, password, "", 1)
		require.NoError(t, err)

		// Test with valid credentials
		valid, err := dbInstance.ValidateUserCredentials(username, password)
		require.NoError(t, err)
		assert.True(t, valid)

		// Test with invalid password
		valid, err = dbInstance.ValidateUserCredentials(username, "wrongpassword")
		require.NoError(t, err)
		assert.False(t, valid)

		// Test with non-existent user
		valid, err = dbInstance.ValidateUserCredentials("nonexistentuser", password)
		require.NoError(t, err)
		assert.False(t, valid)
	})

	// Test deleting a user
	t.Run("DeleteUser", func(t *testing.T) {
		// Create a user
		username := "deleteuser"

		_, err := dbInstance.CreateUser(username, "deletepassword", "", 1)
		require.NoError(t, err)

		// Delete the user
		err = dbInstance.DeleteUser(username)
		require.NoError(t, err)

		// Try to get the deleted user
		_, err = dbInstance.GetUser(username)
		assert.Error(t, err)

		// Try to delete a non-existent user
		err = dbInstance.DeleteUser("nonexistentuser")
		assert.Error(t, err)
	})
}
