# NTRIP Caster Admin API

This package implements an admin API for the NTRIP caster, providing endpoints for managing API keys, users, and mountpoints.

## Features

- **API Key Management**: Create, list, and revoke API keys with specific permissions
- **User Management**: Manage NTRIP client users, their credentials, and mount access
- **Mountpoint Management**: Configure and monitor mountpoints
- **SQLite Storage**: Persistent storage using SQLite database
- **Secure Authentication**: API key-based authentication with permission control

## Usage

### Importing the Package

```go
import "github.com/go-gnss/ntrip/admin"
```

### Creating and Starting the Server

```go
// Create a logger
logger := logrus.New()

// Create the admin server
adminServer, err := admin.NewServer(":8080", "data/ntrip.db", logger)
if err != nil {
    logger.Fatalf("Failed to create admin server: %v", err)
}

// Start the server
go func() {
    logger.Infof("Starting admin API server on port 8080")
    if err := adminServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        logger.Fatalf("Admin API server error: %v", err)
    }
}()

// Graceful shutdown
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()
if err := adminServer.Shutdown(ctx); err != nil {
    logger.Errorf("Error closing admin API server: %v", err)
}

// Close the admin database
if err := adminServer.Close(); err != nil {
    logger.Errorf("Error closing admin database: %v", err)
}
```

### Authentication

The admin API uses API keys for authentication. The admin API key is set via the `ADMIN_API_KEY` environment variable and has full access to all endpoints.

### Database Schema

The package uses SQLite for data storage with the following schema:

#### API Keys Table

```sql
CREATE TABLE api_keys (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  key TEXT UNIQUE NOT NULL,
  name TEXT NOT NULL,
  permissions TEXT NOT NULL,
  expires DATETIME,
  created DATETIME DEFAULT CURRENT_TIMESTAMP
)
```

#### Users Table

```sql
CREATE TABLE users (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  username TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  mounts_allowed TEXT,
  max_connections INTEGER DEFAULT 1,
  created DATETIME DEFAULT CURRENT_TIMESTAMP
)
```

#### Mountpoints Table

```sql
CREATE TABLE mountpoints (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  protocol TEXT DEFAULT 'NTRIP/2.0',
  status TEXT CHECK(status IN ('online','offline','maintenance')) DEFAULT 'online',
  last_active DATETIME
)
```

## API Endpoints

See the [Admin API Documentation](../docs/admin.md) for detailed information about the available endpoints.

## Testing

The package includes unit tests that can be run with:

```bash
go test -v ./admin
```

Note that the tests require CGO to be enabled for SQLite support.

## Dependencies

- `github.com/mattn/go-sqlite3`: SQLite driver for Go
- `golang.org/x/crypto/bcrypt`: Password hashing
- `github.com/sirupsen/logrus`: Logging
