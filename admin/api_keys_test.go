package admin

import (
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateAPIKey(t *testing.T) {
	// Generate a key
	key1, err := GenerateAPIKey()
	require.NoError(t, err)
	assert.NotEmpty(t, key1)

	// Generate another key and ensure it's different
	key2, err := GenerateAPIKey()
	require.NoError(t, err)
	assert.NotEmpty(t, key2)
	assert.NotEqual(t, key1, key2)
}

func TestAPIKeyDB(t *testing.T) {
	// Skip the test since it requires CGO
	t.Skip("Skipping test that requires CGO")

	// Create an in-memory database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Initialize the schema
	_, err = db.Exec(`
		CREATE TABLE api_keys (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			permissions TEXT NOT NULL,
			expires DATETIME,
			created DATETIME DEFAULT CURRENT_TIMESTAMP
		)
	`)
	require.NoError(t, err)

	// Create a DB instance
	dbInstance := &DB{DB: db}

	// Test creating an API key
	t.Run("CreateAPIKey", func(t *testing.T) {
		// Create an API key
		name := "Test API Key"
		permissions := "users:read,mounts:read"
		expires := time.Now().Add(24 * time.Hour)

		apiKey, err := dbInstance.CreateAPIKey(name, permissions, expires)
		require.NoError(t, err)
		assert.NotNil(t, apiKey)
		assert.NotEmpty(t, apiKey.Key)
		assert.Equal(t, name, apiKey.Name)
		assert.Equal(t, permissions, apiKey.Permissions)
		assert.WithinDuration(t, expires, apiKey.Expires, time.Second)
	})

	// Test getting an API key
	t.Run("GetAPIKey", func(t *testing.T) {
		// Create an API key first
		name := "Test Get API Key"
		permissions := "users:read"
		expires := time.Now().Add(24 * time.Hour)

		apiKey, err := dbInstance.CreateAPIKey(name, permissions, expires)
		require.NoError(t, err)

		// Get the API key
		retrievedKey, err := dbInstance.GetAPIKey(apiKey.Key)
		require.NoError(t, err)
		assert.NotNil(t, retrievedKey)
		assert.Equal(t, apiKey.ID, retrievedKey.ID)
		assert.Equal(t, name, retrievedKey.Name)
		assert.Equal(t, permissions, retrievedKey.Permissions)
		assert.WithinDuration(t, expires, retrievedKey.Expires, time.Second)
		assert.Empty(t, retrievedKey.Key) // Key should not be returned
	})

	// Test listing API keys
	t.Run("ListAPIKeys", func(t *testing.T) {
		// Create a few API keys
		_, err := dbInstance.CreateAPIKey("List Key 1", "users:read", time.Now().Add(24*time.Hour))
		require.NoError(t, err)
		_, err = dbInstance.CreateAPIKey("List Key 2", "mounts:read", time.Now().Add(48*time.Hour))
		require.NoError(t, err)

		// List the API keys
		keys, err := dbInstance.ListAPIKeys()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(keys), 2) // At least the two we just created
	})

	// Test validating API key permissions
	t.Run("ValidateAPIKey", func(t *testing.T) {
		// Create an API key with specific permissions
		name := "Validation Key"
		permissions := "users:read,mounts:write"
		expires := time.Now().Add(24 * time.Hour)

		apiKey, err := dbInstance.CreateAPIKey(name, permissions, expires)
		require.NoError(t, err)

		// Test with valid permission
		valid, err := dbInstance.ValidateAPIKey(apiKey.Key, "users:read")
		require.NoError(t, err)
		assert.True(t, valid)

		// Test with invalid permission
		valid, err = dbInstance.ValidateAPIKey(apiKey.Key, "users:write")
		require.NoError(t, err)
		assert.False(t, valid)

		// Test with no required permission
		valid, err = dbInstance.ValidateAPIKey(apiKey.Key, "")
		require.NoError(t, err)
		assert.True(t, valid)

		// Test with non-existent key
		valid, err = dbInstance.ValidateAPIKey("non-existent-key", "users:read")
		require.NoError(t, err)
		assert.False(t, valid)

		// Test with expired key
		expiredKey, err := dbInstance.CreateAPIKey("Expired Key", "users:read", time.Now().Add(-24*time.Hour))
		require.NoError(t, err)

		valid, err = dbInstance.ValidateAPIKey(expiredKey.Key, "users:read")
		require.NoError(t, err)
		assert.False(t, valid)
	})

	// Test deleting an API key
	t.Run("DeleteAPIKey", func(t *testing.T) {
		// Create an API key
		apiKey, err := dbInstance.CreateAPIKey("Delete Key", "users:read", time.Now().Add(24*time.Hour))
		require.NoError(t, err)

		// Delete the API key
		err = dbInstance.DeleteAPIKey(apiKey.ID)
		require.NoError(t, err)

		// Try to get the deleted key
		_, err = dbInstance.GetAPIKey(apiKey.Key)
		assert.Error(t, err)

		// Try to delete a non-existent key
		err = dbInstance.DeleteAPIKey(9999)
		assert.Error(t, err)
	})
}
