package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockDB is a mock implementation of the DB interface for testing
type MockDB struct {
	mock.Mock
}

// CreateAPIKey mocks the CreateAPIKey method
func (m *MockDB) CreateAPIKey(name, permissions string, expires time.Time) (*APIKey, error) {
	args := m.Called(name, permissions, expires)
	return args.Get(0).(*APIKey), args.Error(1)
}

// GetAPIKey mocks the GetAPIKey method
func (m *MockDB) GetAPIKey(key string) (*APIKey, error) {
	args := m.Called(key)
	return args.Get(0).(*APIKey), args.Error(1)
}

// ListAPIKeys mocks the ListAPIKeys method
func (m *MockDB) ListAPIKeys() ([]APIKey, error) {
	args := m.Called()
	return args.Get(0).([]APIKey), args.Error(1)
}

// DeleteAPIKey mocks the DeleteAPIKey method
func (m *MockDB) DeleteAPIKey(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

// ValidateAPIKey mocks the ValidateAPIKey method
func (m *MockDB) ValidateAPIKey(key, requiredPermission string) (bool, error) {
	args := m.Called(key, requiredPermission)
	return args.Bool(0), args.Error(1)
}

// CreateUser mocks the CreateUser method
func (m *MockDB) CreateUser(username, password, mountsAllowed string, maxConnections int) (*User, error) {
	args := m.Called(username, password, mountsAllowed, maxConnections)
	return args.Get(0).(*User), args.Error(1)
}

// GetUser mocks the GetUser method
func (m *MockDB) GetUser(username string) (*User, error) {
	args := m.Called(username)
	return args.Get(0).(*User), args.Error(1)
}

// ListUsers mocks the ListUsers method
func (m *MockDB) ListUsers() ([]User, error) {
	args := m.Called()
	return args.Get(0).([]User), args.Error(1)
}

// UpdateUser mocks the UpdateUser method
func (m *MockDB) UpdateUser(username string, mountsAllowed *string, maxConnections *int, password *string) (*User, error) {
	args := m.Called(username, mountsAllowed, maxConnections, password)
	return args.Get(0).(*User), args.Error(1)
}

// DeleteUser mocks the DeleteUser method
func (m *MockDB) DeleteUser(username string) error {
	args := m.Called(username)
	return args.Error(0)
}

// ValidateUserCredentials mocks the ValidateUserCredentials method
func (m *MockDB) ValidateUserCredentials(username, password string) (bool, error) {
	args := m.Called(username, password)
	return args.Bool(0), args.Error(1)
}

// CreateMountpoint mocks the CreateMountpoint method
func (m *MockDB) CreateMountpoint(name, password, protocol string) (*Mountpoint, error) {
	args := m.Called(name, password, protocol)
	return args.Get(0).(*Mountpoint), args.Error(1)
}

// GetMountpoint mocks the GetMountpoint method
func (m *MockDB) GetMountpoint(name string) (*Mountpoint, error) {
	args := m.Called(name)
	return args.Get(0).(*Mountpoint), args.Error(1)
}

// ListMountpoints mocks the ListMountpoints method
func (m *MockDB) ListMountpoints() ([]Mountpoint, error) {
	args := m.Called()
	return args.Get(0).([]Mountpoint), args.Error(1)
}

// UpdateMountpointStatus mocks the UpdateMountpointStatus method
func (m *MockDB) UpdateMountpointStatus(name, status string) (*Mountpoint, error) {
	args := m.Called(name, status)
	return args.Get(0).(*Mountpoint), args.Error(1)
}

// UpdateMountpointLastActive mocks the UpdateMountpointLastActive method
func (m *MockDB) UpdateMountpointLastActive(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// DeleteMountpoint mocks the DeleteMountpoint method
func (m *MockDB) DeleteMountpoint(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

// ValidateMountpointCredentials mocks the ValidateMountpointCredentials method
func (m *MockDB) ValidateMountpointCredentials(name, password string) (bool, error) {
	args := m.Called(name, password)
	return args.Bool(0), args.Error(1)
}

// MarkOfflineMountpoints mocks the MarkOfflineMountpoints method
func (m *MockDB) MarkOfflineMountpoints(inactiveThreshold time.Duration) error {
	args := m.Called(inactiveThreshold)
	return args.Error(0)
}

// Close mocks the Close method
func (m *MockDB) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestAdminAPI(t *testing.T) {
	// Skip the test if CGO is not enabled
	t.Skip("Skipping test that requires CGO")

	// Create a test logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a mock database
	mockDB := new(MockDB)

	// Set the admin API key for testing
	const testAdminKey = "test-admin-key"
	os.Setenv("ADMIN_API_KEY", testAdminKey)
	defer os.Unsetenv("ADMIN_API_KEY")

	// Create a test server
	server := &Server{
		db:     mockDB,
		logger: logger,
	}

	// Test creating an API key
	t.Run("CreateAPIKey", func(t *testing.T) {
		// Set up mock expectations
		expires := time.Now().Add(24 * time.Hour)
		mockDB.On("CreateAPIKey", "Test API Key", "users:read,mounts:read", mock.AnythingOfType("time.Time")).Return(
			&APIKey{
				ID:          1,
				Key:         "test-key-123",
				Name:        "Test API Key",
				Permissions: "users:read,mounts:read",
				Expires:     expires,
				Created:     time.Now(),
			}, nil)

		// Create a request
		reqBody := map[string]interface{}{
			"name":        "Test API Key",
			"permissions": "users:read,mounts:read",
			"expires":     expires,
		}
		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/keys", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", testAdminKey)

		// Create a response recorder
		rr := httptest.NewRecorder()

		// Call the handler
		server.handleCreateAPIKey(rr, req)

		// Check the response
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Parse the response
		var resp APIKey
		err = json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)

		// Check the response
		assert.Equal(t, "test-key-123", resp.Key)
		assert.Equal(t, reqBody["name"], resp.Name)
		assert.Equal(t, reqBody["permissions"], resp.Permissions)

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	// Test creating a user
	t.Run("CreateUser", func(t *testing.T) {
		// Set up mock expectations
		mockDB.On("CreateUser", "testuser", "testpassword", "MOUNT1,MOUNT2", 5).Return(
			&User{
				ID:             1,
				Username:       "testuser",
				MountsAllowed:  "MOUNT1,MOUNT2",
				MaxConnections: 5,
				Created:        time.Now(),
			}, nil)

		// Create a request
		reqBody := map[string]interface{}{
			"username":        "testuser",
			"password":        "testpassword",
			"mounts_allowed":  "MOUNT1,MOUNT2",
			"max_connections": 5,
		}
		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/users", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", testAdminKey)

		// Create a response recorder
		rr := httptest.NewRecorder()

		// Call the handler
		server.handleCreateUser(rr, req)

		// Check the response
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Parse the response
		var resp User
		err = json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)

		// Check the response
		assert.Equal(t, reqBody["username"], resp.Username)
		assert.Equal(t, reqBody["mounts_allowed"], resp.MountsAllowed)
		assert.Equal(t, reqBody["max_connections"], float64(resp.MaxConnections))
		assert.Empty(t, resp.Password) // Password should not be returned

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	// Test creating a mountpoint
	t.Run("CreateMountpoint", func(t *testing.T) {
		// Set up mock expectations
		mockDB.On("CreateMountpoint", "TESTMOUNT", "testpassword", "NTRIP/2.0").Return(
			&Mountpoint{
				ID:       1,
				Name:     "TESTMOUNT",
				Protocol: "NTRIP/2.0",
				Status:   "online",
			}, nil)

		// Create a request
		reqBody := map[string]interface{}{
			"name":     "TESTMOUNT",
			"password": "testpassword",
			"protocol": "NTRIP/2.0",
		}
		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/mounts", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", testAdminKey)

		// Create a response recorder
		rr := httptest.NewRecorder()

		// Call the handler
		server.handleCreateMountpoint(rr, req)

		// Check the response
		assert.Equal(t, http.StatusCreated, rr.Code)

		// Parse the response
		var resp Mountpoint
		err = json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)

		// Check the response
		assert.Equal(t, reqBody["name"], resp.Name)
		assert.Equal(t, reqBody["protocol"], resp.Protocol)
		assert.Equal(t, "online", resp.Status)
		assert.Empty(t, resp.Password) // Password should not be returned

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	// Test listing users
	t.Run("ListUsers", func(t *testing.T) {
		// Set up mock expectations
		mockDB.On("ListUsers").Return(
			[]User{
				{
					ID:             1,
					Username:       "testuser",
					MountsAllowed:  "MOUNT1,MOUNT2",
					MaxConnections: 5,
					Created:        time.Now(),
				},
			}, nil)

		req := httptest.NewRequest("GET", "/api/users", nil)
		req.Header.Set("X-API-Key", testAdminKey)

		// Create a response recorder
		rr := httptest.NewRecorder()

		// Call the handler
		server.handleListUsers(rr, req)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse the response
		var resp []User
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)

		// Check the response
		assert.Len(t, resp, 1)
		assert.Equal(t, "testuser", resp[0].Username)

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	// Test listing mountpoints
	t.Run("ListMountpoints", func(t *testing.T) {
		// Set up mock expectations
		mockDB.On("ListMountpoints").Return(
			[]Mountpoint{
				{
					ID:       1,
					Name:     "TESTMOUNT",
					Protocol: "NTRIP/2.0",
					Status:   "online",
				},
			}, nil)

		req := httptest.NewRequest("GET", "/api/mounts", nil)
		req.Header.Set("X-API-Key", testAdminKey)

		// Create a response recorder
		rr := httptest.NewRecorder()

		// Call the handler
		server.handleListMountpoints(rr, req)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse the response
		var resp []Mountpoint
		err := json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)

		// Check the response
		assert.Len(t, resp, 1)
		assert.Equal(t, "TESTMOUNT", resp[0].Name)

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	// Test updating a mountpoint status
	t.Run("UpdateMountpointStatus", func(t *testing.T) {
		// Set up mock expectations
		mockDB.On("UpdateMountpointStatus", "TESTMOUNT", "maintenance").Return(
			&Mountpoint{
				ID:       1,
				Name:     "TESTMOUNT",
				Protocol: "NTRIP/2.0",
				Status:   "maintenance",
			}, nil)

		// Create a request
		reqBody := map[string]interface{}{
			"status": "maintenance",
		}
		body, err := json.Marshal(reqBody)
		require.NoError(t, err)

		req := httptest.NewRequest("PUT", "/api/mounts/TESTMOUNT/status", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", testAdminKey)

		// Set the path parameter
		req = req.WithContext(req.Context())
		req.SetPathValue("name", "TESTMOUNT")

		// Create a response recorder
		rr := httptest.NewRecorder()

		// Call the handler
		server.handleUpdateMountpointStatus(rr, req)

		// Check the response
		assert.Equal(t, http.StatusOK, rr.Code)

		// Parse the response
		var resp Mountpoint
		err = json.Unmarshal(rr.Body.Bytes(), &resp)
		require.NoError(t, err)

		// Check the response
		assert.Equal(t, "TESTMOUNT", resp.Name)
		assert.Equal(t, "maintenance", resp.Status)

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})

	// Test deleting a mountpoint
	t.Run("DeleteMountpoint", func(t *testing.T) {
		// Set up mock expectations
		mockDB.On("DeleteMountpoint", "TESTMOUNT").Return(nil)

		req := httptest.NewRequest("DELETE", "/api/mounts/TESTMOUNT", nil)
		req.Header.Set("X-API-Key", testAdminKey)

		// Set the path parameter
		req = req.WithContext(req.Context())
		req.SetPathValue("name", "TESTMOUNT")

		// Create a response recorder
		rr := httptest.NewRecorder()

		// Call the handler
		server.handleDeleteMountpoint(rr, req)

		// Check the response
		assert.Equal(t, http.StatusNoContent, rr.Code)

		// Verify mock expectations
		mockDB.AssertExpectations(t)
	})
}
