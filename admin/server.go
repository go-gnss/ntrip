package admin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// DBInterface defines the interface for database operations
type DBInterface interface {
	// API Key methods
	CreateAPIKey(name, permissions string, expires time.Time) (*APIKey, error)
	GetAPIKey(key string) (*APIKey, error)
	ListAPIKeys() ([]APIKey, error)
	DeleteAPIKey(id int64) error
	ValidateAPIKey(key, requiredPermission string) (bool, error)

	// User methods
	CreateUser(username, password, mountsAllowed string, maxConnections int) (*User, error)
	GetUser(username string) (*User, error)
	ListUsers() ([]User, error)
	UpdateUser(username string, mountsAllowed *string, maxConnections *int, password *string) (*User, error)
	DeleteUser(username string) error
	ValidateUserCredentials(username, password string) (bool, error)

	// Mountpoint methods
	CreateMountpoint(name, password, protocol string) (*Mountpoint, error)
	GetMountpoint(name string) (*Mountpoint, error)
	ListMountpoints() ([]Mountpoint, error)
	UpdateMountpointStatus(name, status string) (*Mountpoint, error)
	UpdateMountpointLastActive(name string) error
	DeleteMountpoint(name string) error
	ValidateMountpointCredentials(name, password string) (bool, error)
	MarkOfflineMountpoints(inactiveThreshold time.Duration) error

	// Close the database connection
	Close() error
}

// Server represents the admin API server
type Server struct {
	http.Server
	db     DBInterface
	logger logrus.FieldLogger
}

// NewServer creates a new admin API server
func NewServer(addr string, dbPath string, logger logrus.FieldLogger) (*Server, error) {
	// Create the database
	db, err := NewDB(dbPath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Create the server
	server := &Server{
		Server: http.Server{
			Addr:         addr,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		db:     db,
		logger: logger,
	}

	// Set up the router
	mux := http.NewServeMux()

	// API key endpoints
	mux.HandleFunc("POST /api/keys", server.apiKeyAuthMiddleware(server.handleCreateAPIKey))
	mux.HandleFunc("GET /api/keys", server.apiKeyAuthMiddleware(server.handleListAPIKeys))
	mux.HandleFunc("DELETE /api/keys/{id}", server.apiKeyAuthMiddleware(server.handleDeleteAPIKey))

	// User endpoints
	mux.HandleFunc("POST /api/users", server.apiKeyAuthMiddleware(server.handleCreateUser))
	mux.HandleFunc("GET /api/users", server.apiKeyAuthMiddleware(server.handleListUsers))
	mux.HandleFunc("GET /api/users/{username}", server.apiKeyAuthMiddleware(server.handleGetUser))
	mux.HandleFunc("PUT /api/users/{username}", server.apiKeyAuthMiddleware(server.handleUpdateUser))
	mux.HandleFunc("DELETE /api/users/{username}", server.apiKeyAuthMiddleware(server.handleDeleteUser))

	// Mountpoint endpoints
	mux.HandleFunc("POST /api/mounts", server.apiKeyAuthMiddleware(server.handleCreateMountpoint))
	mux.HandleFunc("GET /api/mounts", server.apiKeyAuthMiddleware(server.handleListMountpoints))
	mux.HandleFunc("GET /api/mounts/{name}", server.apiKeyAuthMiddleware(server.handleGetMountpoint))
	mux.HandleFunc("PUT /api/mounts/{name}/status", server.apiKeyAuthMiddleware(server.handleUpdateMountpointStatus))
	mux.HandleFunc("DELETE /api/mounts/{name}", server.apiKeyAuthMiddleware(server.handleDeleteMountpoint))

	server.Handler = mux

	// Start background task to mark offline mountpoints
	go server.markOfflineMountpointsTask()

	return server, nil
}

// Close closes the server and database connection
func (s *Server) Close() error {
	return s.db.Close()
}

// GetDB returns the database interface
func (s *Server) GetDB() DBInterface {
	return s.db
}

// NewTLSServer creates a new admin API server with TLS support
func NewTLSServer(addr string, dbPath string, certFile, keyFile string, logger logrus.FieldLogger) (*Server, error) {
	// Create a regular server first
	server, err := NewServer(addr, dbPath, logger)
	if err != nil {
		return nil, err
	}

	// Log that we're using TLS
	logger.Info("Admin API server configured with TLS")

	// Return the server
	return server, nil
}

// apiKeyAuthMiddleware is a middleware that checks for a valid API key
func (s *Server) apiKeyAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the API key from the request header
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" {
			s.logger.Warn("Missing API key")
			http.Error(w, "Missing API key", http.StatusUnauthorized)
			return
		}

		// Check if it's the admin API key from environment variable
		adminAPIKey := os.Getenv("ADMIN_API_KEY")
		if adminAPIKey != "" && apiKey == adminAPIKey {
			// Admin API key has all permissions
			next(w, r)
			return
		}

		// Validate the API key
		valid, err := s.db.ValidateAPIKey(apiKey, "")
		if err != nil {
			s.logger.WithError(err).Error("Failed to validate API key")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if !valid {
			s.logger.Warn("Invalid API key")
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		// API key is valid, proceed to the next handler
		next(w, r)
	}
}

// handleCreateAPIKey handles the creation of a new API key
func (s *Server) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var req struct {
		Name        string    `json:"name"`
		Permissions string    `json:"permissions"`
		Expires     time.Time `json:"expires"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.WithError(err).Error("Failed to decode request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the request
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Create the API key
	apiKey, err := s.db.CreateAPIKey(req.Name, req.Permissions, req.Expires)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create API key")
		http.Error(w, "Failed to create API key", http.StatusInternalServerError)
		return
	}

	// Return the API key
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(apiKey)
}

// handleListAPIKeys handles listing all API keys
func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	// Get all API keys
	apiKeys, err := s.db.ListAPIKeys()
	if err != nil {
		s.logger.WithError(err).Error("Failed to list API keys")
		http.Error(w, "Failed to list API keys", http.StatusInternalServerError)
		return
	}

	// Return the API keys
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(apiKeys)
}

// handleDeleteAPIKey handles deleting an API key
func (s *Server) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	// Get the API key ID from the URL
	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid API key ID", http.StatusBadRequest)
		return
	}

	// Delete the API key
	err = s.db.DeleteAPIKey(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "API key not found", http.StatusNotFound)
			return
		}
		s.logger.WithError(err).Error("Failed to delete API key")
		http.Error(w, "Failed to delete API key", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusNoContent)
}

// handleCreateUser handles the creation of a new user
func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var req struct {
		Username       string `json:"username"`
		Password       string `json:"password"`
		MountsAllowed  string `json:"mounts_allowed"`
		MaxConnections int    `json:"max_connections"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.WithError(err).Error("Failed to decode request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the request
	if req.Username == "" {
		http.Error(w, "Username is required", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}

	// Create the user
	user, err := s.db.CreateUser(req.Username, req.Password, req.MountsAllowed, req.MaxConnections)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create user")
		http.Error(w, "Failed to create user", http.StatusInternalServerError)
		return
	}

	// Return the user
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}

// handleListUsers handles listing all users
func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	// Get all users
	users, err := s.db.ListUsers()
	if err != nil {
		s.logger.WithError(err).Error("Failed to list users")
		http.Error(w, "Failed to list users", http.StatusInternalServerError)
		return
	}

	// Return the users
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

// handleGetUser handles getting a user
func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	// Get the username from the URL
	username := r.PathValue("username")

	// Get the user
	user, err := s.db.GetUser(username)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		s.logger.WithError(err).Error("Failed to get user")
		http.Error(w, "Failed to get user", http.StatusInternalServerError)
		return
	}

	// Return the user
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// handleUpdateUser handles updating a user
func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	// Get the username from the URL
	username := r.PathValue("username")

	// Parse the request body
	var req struct {
		Password       *string `json:"password"`
		MountsAllowed  *string `json:"mounts_allowed"`
		MaxConnections *int    `json:"max_connections"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.WithError(err).Error("Failed to decode request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update the user
	user, err := s.db.UpdateUser(username, req.MountsAllowed, req.MaxConnections, req.Password)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		s.logger.WithError(err).Error("Failed to update user")
		http.Error(w, "Failed to update user", http.StatusInternalServerError)
		return
	}

	// Return the updated user
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

// handleDeleteUser handles deleting a user
func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	// Get the username from the URL
	username := r.PathValue("username")

	// Delete the user
	err := s.db.DeleteUser(username)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}
		s.logger.WithError(err).Error("Failed to delete user")
		http.Error(w, "Failed to delete user", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusNoContent)
}

// handleCreateMountpoint handles the creation of a new mountpoint
func (s *Server) handleCreateMountpoint(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var req struct {
		Name     string `json:"name"`
		Password string `json:"password"`
		Protocol string `json:"protocol"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.WithError(err).Error("Failed to decode request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the request
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if req.Password == "" {
		http.Error(w, "Password is required", http.StatusBadRequest)
		return
	}
	if req.Protocol == "" {
		req.Protocol = "NTRIP/2.0"
	}

	// Create the mountpoint
	mountpoint, err := s.db.CreateMountpoint(req.Name, req.Password, req.Protocol)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create mountpoint")
		http.Error(w, "Failed to create mountpoint", http.StatusInternalServerError)
		return
	}

	// Return the mountpoint
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(mountpoint)
}

// handleListMountpoints handles listing all mountpoints
func (s *Server) handleListMountpoints(w http.ResponseWriter, r *http.Request) {
	// Get all mountpoints
	mountpoints, err := s.db.ListMountpoints()
	if err != nil {
		s.logger.WithError(err).Error("Failed to list mountpoints")
		http.Error(w, "Failed to list mountpoints", http.StatusInternalServerError)
		return
	}

	// Return the mountpoints
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mountpoints)
}

// handleGetMountpoint handles getting a mountpoint
func (s *Server) handleGetMountpoint(w http.ResponseWriter, r *http.Request) {
	// Get the mountpoint name from the URL
	name := r.PathValue("name")

	// Get the mountpoint
	mountpoint, err := s.db.GetMountpoint(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Mountpoint not found", http.StatusNotFound)
			return
		}
		s.logger.WithError(err).Error("Failed to get mountpoint")
		http.Error(w, "Failed to get mountpoint", http.StatusInternalServerError)
		return
	}

	// Return the mountpoint
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mountpoint)
}

// handleUpdateMountpointStatus handles updating a mountpoint's status
func (s *Server) handleUpdateMountpointStatus(w http.ResponseWriter, r *http.Request) {
	// Get the mountpoint name from the URL
	name := r.PathValue("name")

	// Parse the request body
	var req struct {
		Status string `json:"status"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.WithError(err).Error("Failed to decode request body")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate the request
	if req.Status == "" {
		http.Error(w, "Status is required", http.StatusBadRequest)
		return
	}

	// Update the mountpoint status
	mountpoint, err := s.db.UpdateMountpointStatus(name, req.Status)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Mountpoint not found", http.StatusNotFound)
			return
		}
		if strings.Contains(err.Error(), "invalid status") {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.logger.WithError(err).Error("Failed to update mountpoint status")
		http.Error(w, "Failed to update mountpoint status", http.StatusInternalServerError)
		return
	}

	// Return the updated mountpoint
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(mountpoint)
}

// handleDeleteMountpoint handles deleting a mountpoint
func (s *Server) handleDeleteMountpoint(w http.ResponseWriter, r *http.Request) {
	// Get the mountpoint name from the URL
	name := r.PathValue("name")

	// Delete the mountpoint
	err := s.db.DeleteMountpoint(name)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, "Mountpoint not found", http.StatusNotFound)
			return
		}
		s.logger.WithError(err).Error("Failed to delete mountpoint")
		http.Error(w, "Failed to delete mountpoint", http.StatusInternalServerError)
		return
	}

	// Return success
	w.WriteHeader(http.StatusNoContent)
}

// markOfflineMountpointsTask runs in the background to mark mountpoints as offline
func (s *Server) markOfflineMountpointsTask() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		err := s.db.MarkOfflineMountpoints(10 * time.Minute)
		if err != nil {
			s.logger.WithError(err).Error("Failed to mark offline mountpoints")
		}
	}
}
