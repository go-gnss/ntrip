package admin

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestMountpointsDB(t *testing.T) {
	// Skip the test since it requires CGO
	t.Skip("Skipping test that requires CGO")

	// Create an in-memory database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Initialize the schema
	_, err = db.Exec(`
		CREATE TABLE mountpoints (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			protocol TEXT DEFAULT 'NTRIP/2.0',
			status TEXT CHECK(status IN ('online','offline','maintenance')) DEFAULT 'online',
			last_active DATETIME
		)
	`)
	require.NoError(t, err)

	// Create a DB instance
	dbInstance := &DB{DB: db}

	// Test creating a mountpoint
	t.Run("CreateMountpoint", func(t *testing.T) {
		// Create a mountpoint
		name := "TESTMOUNT"
		password := "testpassword"
		protocol := "NTRIP/2.0"

		mountpoint, err := dbInstance.CreateMountpoint(name, password, protocol)
		require.NoError(t, err)
		assert.NotNil(t, mountpoint)
		assert.Equal(t, name, mountpoint.Name)
		assert.Equal(t, protocol, mountpoint.Protocol)
		assert.Equal(t, "online", mountpoint.Status) // Default status

		// Verify the password was hashed
		var passwordHash string
		err = db.QueryRow("SELECT password_hash FROM mountpoints WHERE name = ?", name).Scan(&passwordHash)
		require.NoError(t, err)
		assert.NotEqual(t, password, passwordHash)

		// Verify the hash is valid
		err = bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))
		assert.NoError(t, err)
	})

	// Test getting a mountpoint
	t.Run("GetMountpoint", func(t *testing.T) {
		// Create a mountpoint first
		name := "GETMOUNT"
		password := "getpassword"
		protocol := "NTRIP/2.0"

		_, err := dbInstance.CreateMountpoint(name, password, protocol)
		require.NoError(t, err)

		// Get the mountpoint
		mountpoint, err := dbInstance.GetMountpoint(name)
		require.NoError(t, err)
		assert.NotNil(t, mountpoint)
		assert.Equal(t, name, mountpoint.Name)
		assert.Equal(t, protocol, mountpoint.Protocol)
		assert.Equal(t, "online", mountpoint.Status)

		// Try to get a non-existent mountpoint
		_, err = dbInstance.GetMountpoint("NONEXISTENTMOUNT")
		assert.Error(t, err)
	})

	// Test listing mountpoints
	t.Run("ListMountpoints", func(t *testing.T) {
		// Create a few mountpoints
		_, err := dbInstance.CreateMountpoint("LISTMOUNT1", "password1", "NTRIP/2.0")
		require.NoError(t, err)
		_, err = dbInstance.CreateMountpoint("LISTMOUNT2", "password2", "NTRIP/1.0")
		require.NoError(t, err)

		// List the mountpoints
		mountpoints, err := dbInstance.ListMountpoints()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(mountpoints), 2) // At least the two we just created
	})

	// Test updating mountpoint status
	t.Run("UpdateMountpointStatus", func(t *testing.T) {
		// Create a mountpoint first
		name := "STATUSMOUNT"
		password := "statuspassword"
		protocol := "NTRIP/2.0"

		_, err := dbInstance.CreateMountpoint(name, password, protocol)
		require.NoError(t, err)

		// Update the mountpoint status
		mountpoint, err := dbInstance.UpdateMountpointStatus(name, "maintenance")
		require.NoError(t, err)
		assert.Equal(t, "maintenance", mountpoint.Status)

		// Update to offline
		mountpoint, err = dbInstance.UpdateMountpointStatus(name, "offline")
		require.NoError(t, err)
		assert.Equal(t, "offline", mountpoint.Status)

		// Update back to online
		mountpoint, err = dbInstance.UpdateMountpointStatus(name, "online")
		require.NoError(t, err)
		assert.Equal(t, "online", mountpoint.Status)

		// Try to update with an invalid status
		_, err = dbInstance.UpdateMountpointStatus(name, "invalid")
		assert.Error(t, err)

		// Try to update a non-existent mountpoint
		_, err = dbInstance.UpdateMountpointStatus("NONEXISTENTMOUNT", "online")
		assert.Error(t, err)
	})

	// Test updating mountpoint last active time
	t.Run("UpdateMountpointLastActive", func(t *testing.T) {
		// Create a mountpoint first
		name := "ACTIVEMOUNT"
		password := "activepassword"
		protocol := "NTRIP/2.0"

		_, err := dbInstance.CreateMountpoint(name, password, protocol)
		require.NoError(t, err)

		// Update the last active time
		err = dbInstance.UpdateMountpointLastActive(name)
		require.NoError(t, err)

		// Verify the last active time was updated
		var lastActive time.Time
		err = db.QueryRow("SELECT last_active FROM mountpoints WHERE name = ?", name).Scan(&lastActive)
		require.NoError(t, err)
		assert.WithinDuration(t, time.Now(), lastActive, 2*time.Second)

		// Try to update a non-existent mountpoint
		err = dbInstance.UpdateMountpointLastActive("NONEXISTENTMOUNT")
		assert.Error(t, err)
	})

	// Test validating mountpoint credentials
	t.Run("ValidateMountpointCredentials", func(t *testing.T) {
		// Create a mountpoint
		name := "VALIDATEMOUNT"
		password := "validatepassword"

		_, err := dbInstance.CreateMountpoint(name, password, "NTRIP/2.0")
		require.NoError(t, err)

		// Test with valid credentials
		valid, err := dbInstance.ValidateMountpointCredentials(name, password)
		require.NoError(t, err)
		assert.True(t, valid)

		// Test with invalid password
		valid, err = dbInstance.ValidateMountpointCredentials(name, "wrongpassword")
		require.NoError(t, err)
		assert.False(t, valid)

		// Test with non-existent mountpoint
		valid, err = dbInstance.ValidateMountpointCredentials("NONEXISTENTMOUNT", password)
		require.NoError(t, err)
		assert.False(t, valid)
	})

	// Test marking offline mountpoints
	t.Run("MarkOfflineMountpoints", func(t *testing.T) {
		// Create a mountpoint with an old last_active time
		name := "OLDMOUNT"
		_, err := dbInstance.CreateMountpoint(name, "password", "NTRIP/2.0")
		require.NoError(t, err)

		// Set the last_active time to 10 minutes ago
		tenMinutesAgo := time.Now().Add(-10 * time.Minute)
		_, err = db.Exec("UPDATE mountpoints SET last_active = ? WHERE name = ?", tenMinutesAgo, name)
		require.NoError(t, err)

		// Create a mountpoint with a recent last_active time
		recentName := "RECENTMOUNT"
		_, err = dbInstance.CreateMountpoint(recentName, "password", "NTRIP/2.0")
		require.NoError(t, err)

		// Set the last_active time to 1 minute ago
		oneMinuteAgo := time.Now().Add(-1 * time.Minute)
		_, err = db.Exec("UPDATE mountpoints SET last_active = ? WHERE name = ?", oneMinuteAgo, recentName)
		require.NoError(t, err)

		// Mark mountpoints as offline if they haven't been active for 5 minutes
		err = dbInstance.MarkOfflineMountpoints(5 * time.Minute)
		require.NoError(t, err)

		// Verify the old mountpoint is now offline
		oldMountpoint, err := dbInstance.GetMountpoint(name)
		require.NoError(t, err)
		assert.Equal(t, "offline", oldMountpoint.Status)

		// Verify the recent mountpoint is still online
		recentMountpoint, err := dbInstance.GetMountpoint(recentName)
		require.NoError(t, err)
		assert.Equal(t, "online", recentMountpoint.Status)
	})

	// Test deleting a mountpoint
	t.Run("DeleteMountpoint", func(t *testing.T) {
		// Create a mountpoint
		name := "DELETEMOUNT"

		_, err := dbInstance.CreateMountpoint(name, "deletepassword", "NTRIP/2.0")
		require.NoError(t, err)

		// Delete the mountpoint
		err = dbInstance.DeleteMountpoint(name)
		require.NoError(t, err)

		// Try to get the deleted mountpoint
		_, err = dbInstance.GetMountpoint(name)
		assert.Error(t, err)

		// Try to delete a non-existent mountpoint
		err = dbInstance.DeleteMountpoint("NONEXISTENTMOUNT")
		assert.Error(t, err)
	})
}
