package auth

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-gnss/ntrip/admin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDBInterface is a mock implementation of the admin.DBInterface
type MockDBInterface struct {
	mock.Mock
}

// ValidateUserCredentials mocks the ValidateUserCredentials method
func (m *MockDBInterface) ValidateUserCredentials(username, password string) (bool, error) {
	args := m.Called(username, password)
	return args.Bool(0), args.Error(1)
}

// GetUser mocks the GetUser method
func (m *MockDBInterface) GetUser(username string) (*admin.User, error) {
	args := m.Called(username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*admin.User), args.Error(1)
}

// ValidateMountpointCredentials mocks the ValidateMountpointCredentials method
func (m *MockDBInterface) ValidateMountpointCredentials(name, password string) (bool, error) {
	args := m.Called(name, password)
	return args.Bool(0), args.Error(1)
}

// Implement other required methods of the DBInterface
func (m *MockDBInterface) CreateAPIKey(name, permissions string, expires time.Time) (*admin.APIKey, error) {
	args := m.Called(name, permissions, expires)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*admin.APIKey), args.Error(1)
}

func (m *MockDBInterface) GetAPIKey(key string) (*admin.APIKey, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*admin.APIKey), args.Error(1)
}

func (m *MockDBInterface) ListAPIKeys() ([]admin.APIKey, error) {
	args := m.Called()
	return args.Get(0).([]admin.APIKey), args.Error(1)
}

func (m *MockDBInterface) DeleteAPIKey(id int64) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockDBInterface) ValidateAPIKey(key, requiredPermission string) (bool, error) {
	args := m.Called(key, requiredPermission)
	return args.Bool(0), args.Error(1)
}

func (m *MockDBInterface) CreateUser(username, password, mountsAllowed string, maxConnections int) (*admin.User, error) {
	args := m.Called(username, password, mountsAllowed, maxConnections)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*admin.User), args.Error(1)
}

func (m *MockDBInterface) ListUsers() ([]admin.User, error) {
	args := m.Called()
	return args.Get(0).([]admin.User), args.Error(1)
}

func (m *MockDBInterface) UpdateUser(username string, mountsAllowed *string, maxConnections *int, password *string) (*admin.User, error) {
	args := m.Called(username, mountsAllowed, maxConnections, password)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*admin.User), args.Error(1)
}

func (m *MockDBInterface) DeleteUser(username string) error {
	args := m.Called(username)
	return args.Error(0)
}

func (m *MockDBInterface) CreateMountpoint(name, password, protocol string) (*admin.Mountpoint, error) {
	args := m.Called(name, password, protocol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*admin.Mountpoint), args.Error(1)
}

func (m *MockDBInterface) GetMountpoint(name string) (*admin.Mountpoint, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*admin.Mountpoint), args.Error(1)
}

func (m *MockDBInterface) ListMountpoints() ([]admin.Mountpoint, error) {
	args := m.Called()
	return args.Get(0).([]admin.Mountpoint), args.Error(1)
}

func (m *MockDBInterface) UpdateMountpointStatus(name, status string) (*admin.Mountpoint, error) {
	args := m.Called(name, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*admin.Mountpoint), args.Error(1)
}

func (m *MockDBInterface) UpdateMountpointLastActive(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockDBInterface) DeleteMountpoint(name string) error {
	args := m.Called(name)
	return args.Error(0)
}

func (m *MockDBInterface) MarkOfflineMountpoints(inactiveThreshold time.Duration) error {
	args := m.Called(inactiveThreshold)
	return args.Error(0)
}

func (m *MockDBInterface) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestDBAuth_Authenticate(t *testing.T) {
	// Create a mock DB
	mockDB := new(MockDBInterface)

	// Create a DBAuth instance with the mock DB
	dbAuth := NewDBAuth(mockDB)

	// Test case 1: Valid credentials with access to the mount
	mockDB.On("ValidateUserCredentials", "user1", "pass1").Return(true, nil)
	mockDB.On("GetUser", "user1").Return(&admin.User{
		Username:      "user1",
		MountsAllowed: "mount1,mount2",
	}, nil)

	// Create a request with basic auth
	req1, _ := http.NewRequest("GET", "http://example.com", nil)
	req1.SetBasicAuth("user1", "pass1")

	// Test authentication
	result1, err1 := dbAuth.Authenticate(req1, "mount1")
	assert.NoError(t, err1)
	assert.True(t, result1)

	// Test case 2: Valid credentials but no access to the mount
	mockDB.On("ValidateUserCredentials", "user2", "pass2").Return(true, nil)
	mockDB.On("GetUser", "user2").Return(&admin.User{
		Username:      "user2",
		MountsAllowed: "mount3,mount4",
	}, nil)

	// Create a request with basic auth
	req2, _ := http.NewRequest("GET", "http://example.com", nil)
	req2.SetBasicAuth("user2", "pass2")

	// Test authentication
	result2, err2 := dbAuth.Authenticate(req2, "mount1")
	assert.NoError(t, err2)
	assert.False(t, result2)

	// Test case 3: Invalid credentials
	mockDB.On("ValidateUserCredentials", "user3", "pass3").Return(false, nil)

	// Create a request with basic auth
	req3, _ := http.NewRequest("GET", "http://example.com", nil)
	req3.SetBasicAuth("user3", "pass3")

	// Test authentication
	result3, err3 := dbAuth.Authenticate(req3, "mount1")
	assert.NoError(t, err3)
	assert.False(t, result3)

	// Test case 4: No auth in request
	req4, _ := http.NewRequest("GET", "http://example.com", nil)

	// Test authentication
	result4, err4 := dbAuth.Authenticate(req4, "mount1")
	assert.NoError(t, err4)
	assert.False(t, result4)

	// Test case 5: User with empty mounts_allowed (access to all mounts)
	mockDB.On("ValidateUserCredentials", "user5", "pass5").Return(true, nil)
	mockDB.On("GetUser", "user5").Return(&admin.User{
		Username:      "user5",
		MountsAllowed: "",
	}, nil)

	// Create a request with basic auth
	req5, _ := http.NewRequest("GET", "http://example.com", nil)
	req5.SetBasicAuth("user5", "pass5")

	// Test authentication
	result5, err5 := dbAuth.Authenticate(req5, "mount1")
	assert.NoError(t, err5)
	assert.True(t, result5)

	// Test case 6: Error getting user
	mockDB.On("ValidateUserCredentials", "user6", "pass6").Return(true, nil)
	mockDB.On("GetUser", "user6").Return(nil, assert.AnError)

	// Create a request with basic auth
	req6, _ := http.NewRequest("GET", "http://example.com", nil)
	req6.SetBasicAuth("user6", "pass6")

	// Test authentication
	result6, err6 := dbAuth.Authenticate(req6, "mount1")
	assert.Error(t, err6)
	assert.False(t, result6)

	// Verify all expectations were met
	mockDB.AssertExpectations(t)
}

func TestDBAuth_AuthenticateMountpoint(t *testing.T) {
	// Create a mock DB
	mockDB := new(MockDBInterface)

	// Create a DBAuth instance with the mock DB
	dbAuth := NewDBAuth(mockDB)

	// Test case 1: Valid mountpoint credentials
	mockDB.On("ValidateMountpointCredentials", "mount1", "pass1").Return(true, nil)

	// Test authentication
	result1, err1 := dbAuth.AuthenticateMountpoint("mount1", "pass1")
	assert.NoError(t, err1)
	assert.True(t, result1)

	// Test case 2: Invalid mountpoint credentials
	mockDB.On("ValidateMountpointCredentials", "mount2", "pass2").Return(false, nil)

	// Test authentication
	result2, err2 := dbAuth.AuthenticateMountpoint("mount2", "pass2")
	assert.NoError(t, err2)
	assert.False(t, result2)

	// Test case 3: Error validating mountpoint credentials
	mockDB.On("ValidateMountpointCredentials", "mount3", "pass3").Return(false, assert.AnError)

	// Test authentication
	result3, err3 := dbAuth.AuthenticateMountpoint("mount3", "pass3")
	assert.Error(t, err3)
	assert.False(t, result3)

	// Verify all expectations were met
	mockDB.AssertExpectations(t)
}

func TestDBAuth_Method(t *testing.T) {
	// Create a mock DB
	mockDB := new(MockDBInterface)

	// Create a DBAuth instance with the mock DB
	dbAuth := NewDBAuth(mockDB)

	// Test the method
	assert.Equal(t, Basic, dbAuth.Method())
}

func TestDBAuth_Challenge(t *testing.T) {
	// Create a mock DB
	mockDB := new(MockDBInterface)

	// Create a DBAuth instance with the mock DB
	dbAuth := NewDBAuth(mockDB)

	// Test the challenge
	challenge := dbAuth.Challenge("mount1")
	assert.Equal(t, `Basic realm="mount1"`, challenge)
}
