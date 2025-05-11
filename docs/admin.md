# NTRIP Caster Admin API Documentation

This document provides comprehensive documentation for the NTRIP Caster Admin API, which allows you to manage API keys, users, and mountpoints through a RESTful interface.

## Table of Contents

- [Overview](#overview)
- [Getting Started](#getting-started)
  - [Configuration](#configuration)
  - [Security Considerations](#security-considerations)
- [API Reference](#api-reference)
  - [Authentication](#authentication)
  - [API Keys](#api-keys)
  - [Users](#users)
  - [Mountpoints](#mountpoints)
- [Tutorials](#tutorials)
  - [Setting Up the Admin API](#setting-up-the-admin-api)
  - [Managing API Keys](#managing-api-keys)
  - [Managing Users](#managing-users)
  - [Managing Mountpoints](#managing-mountpoints)
- [Troubleshooting](#troubleshooting)

## Overview

The Admin API provides a secure interface for managing your NTRIP caster. It uses SQLite for data storage and provides endpoints for:

- **API Key Management**: Create, list, and revoke API keys with specific permissions
- **User Management**: Manage NTRIP client users, their credentials, and mount access
- **Mountpoint Management**: Configure and monitor mountpoints
- **Authentication Integration**: Seamless integration with the NTRIP authentication system

## Getting Started

### Prerequisites

- Go 1.22 or later
- CGO enabled (required for SQLite)

```bash
# Check if CGO is enabled
go env CGO_ENABLED

# Enable CGO if needed
set CGO_ENABLED=1  # Windows
export CGO_ENABLED=1  # Linux/Mac
```

### Configuration

The Admin API is configured through command-line flags when starting the NTRIP server:

```bash
go run cmd/ntrip-server/main.go \
  --http-port 2101 \
  --admin-port 8080 \
  --db-path data/ntrip.db
```

Available flags:
- `--admin-port`: Port for the Admin API (default: 8080)
- `--db-path`: Path to the SQLite database file (default: data/ntrip.db)
- `--tls-cert`: Path to TLS certificate file for HTTPS support
- `--tls-key`: Path to TLS certificate key file for HTTPS support

### Security Considerations

1. **Admin API Key**

   Set a strong, random admin API key in your environment:

   ```bash
   # Linux/Mac
   export ADMIN_API_KEY=your-secure-api-key-here

   # Windows
   set ADMIN_API_KEY=your-secure-api-key-here
   ```

   Or use a `.env` file in your project directory:

   ```
   ADMIN_API_KEY=your-secure-api-key-here
   ```

2. **HTTPS**

   For production use, configure HTTPS for the Admin API to secure API key transmission:

   ```bash
   go run cmd/ntrip-server/main.go \
     --admin-port 8080 \
     --db-path data/ntrip.db \
     --tls-cert path/to/cert.pem \
     --tls-key path/to/key.pem
   ```

3. **API Key Rotation**

   Regularly rotate API keys and set appropriate expiration dates.

4. **Authentication Integration**

   The Admin API database is automatically integrated with the NTRIP authentication system. Users created through the Admin API can immediately connect to the NTRIP caster using their credentials, and mountpoints created through the Admin API are automatically available for data streaming.

## API Reference

### Authentication

All API endpoints require authentication using the `X-API-Key` header:

```
X-API-Key: your-api-key-here
```

The admin API key (set via environment variable) has full access to all endpoints. Other API keys can be created with specific permissions.

### API Keys

#### Create API Key

```
POST /api/keys
```

Request body:
```json
{
  "name": "Client API Key",
  "permissions": "users:read,mounts:read",
  "expires": "2025-01-01T00:00:00Z"
}
```

Response:
```json
{
  "id": 1,
  "key": "generated-api-key-value",
  "name": "Client API Key",
  "permissions": "users:read,mounts:read",
  "expires": "2025-01-01T00:00:00Z",
  "created": "2023-06-01T12:00:00Z"
}
```

**Note**: The API key value is only returned once upon creation.

#### List API Keys

```
GET /api/keys
```

Response:
```json
[
  {
    "id": 1,
    "name": "Client API Key",
    "permissions": "users:read,mounts:read",
    "expires": "2025-01-01T00:00:00Z",
    "created": "2023-06-01T12:00:00Z"
  }
]
```

#### Delete API Key

```
DELETE /api/keys/{id}
```

Response: 204 No Content

### Users

#### Create User

```
POST /api/users
```

Request body:
```json
{
  "username": "survey_team",
  "password": "secure_password",
  "mounts_allowed": "RTCM3_MOUNT,CMR_MOUNT",
  "max_connections": 3
}
```

Response:
```json
{
  "id": 1,
  "username": "survey_team",
  "mounts_allowed": "RTCM3_MOUNT,CMR_MOUNT",
  "max_connections": 3,
  "created": "2023-06-01T12:00:00Z"
}
```

#### List Users

```
GET /api/users
```

Response:
```json
[
  {
    "id": 1,
    "username": "survey_team",
    "mounts_allowed": "RTCM3_MOUNT,CMR_MOUNT",
    "max_connections": 3,
    "created": "2023-06-01T12:00:00Z"
  }
]
```

#### Get User

```
GET /api/users/{username}
```

Response:
```json
{
  "id": 1,
  "username": "survey_team",
  "mounts_allowed": "RTCM3_MOUNT,CMR_MOUNT",
  "max_connections": 3,
  "created": "2023-06-01T12:00:00Z"
}
```

#### Update User

```
PUT /api/users/{username}
```

Request body (all fields optional):
```json
{
  "password": "new_password",
  "mounts_allowed": "RTCM3_MOUNT,NEW_MOUNT",
  "max_connections": 5
}
```

Response:
```json
{
  "id": 1,
  "username": "survey_team",
  "mounts_allowed": "RTCM3_MOUNT,NEW_MOUNT",
  "max_connections": 5,
  "created": "2023-06-01T12:00:00Z"
}
```

#### Delete User

```
DELETE /api/users/{username}
```

Response: 204 No Content

### Mountpoints

#### Create Mountpoint

```
POST /api/mounts
```

Request body:
```json
{
  "name": "RTCM3_MOUNT",
  "password": "mount_password",
  "protocol": "NTRIP/2.0"
}
```

Response:
```json
{
  "id": 1,
  "name": "RTCM3_MOUNT",
  "protocol": "NTRIP/2.0",
  "status": "online"
}
```

#### List Mountpoints

```
GET /api/mounts
```

Response:
```json
[
  {
    "id": 1,
    "name": "RTCM3_MOUNT",
    "protocol": "NTRIP/2.0",
    "status": "online",
    "last_active": "2023-06-01T12:05:00Z"
  }
]
```

#### Get Mountpoint

```
GET /api/mounts/{name}
```

Response:
```json
{
  "id": 1,
  "name": "RTCM3_MOUNT",
  "protocol": "NTRIP/2.0",
  "status": "online",
  "last_active": "2023-06-01T12:05:00Z"
}
```

#### Update Mountpoint Status

```
PUT /api/mounts/{name}/status
```

Request body:
```json
{
  "status": "maintenance"
}
```

Valid status values:
- `online`: Mountpoint is available
- `offline`: Mountpoint is unavailable
- `maintenance`: Mountpoint is temporarily unavailable for maintenance

Response:
```json
{
  "id": 1,
  "name": "RTCM3_MOUNT",
  "protocol": "NTRIP/2.0",
  "status": "maintenance",
  "last_active": "2023-06-01T12:05:00Z"
}
```

#### Delete Mountpoint

```
DELETE /api/mounts/{name}
```

Response: 204 No Content

## Tutorials

### Setting Up the Admin API

1. **Start the NTRIP server with Admin API enabled**

   ```bash
   # Set the admin API key
   export ADMIN_API_KEY=your-secure-api-key-here

   # Start the server
   go run cmd/ntrip-server/main.go --admin-port 8080 --db-path data/ntrip.db
   ```

2. **Verify the API is running**

   ```bash
   curl -I -H "X-API-Key: your-secure-api-key-here" http://localhost:8080/api/keys
   ```

   You should receive a `200 OK` response.

### Managing API Keys

#### Creating an API Key for a Client Application

1. **Create the API key**

   ```bash
   curl -X POST \
     -H "Content-Type: application/json" \
     -H "X-API-Key: your-admin-api-key" \
     -d '{
       "name": "Mobile App Key",
       "permissions": "mounts:read",
       "expires": "2024-12-31T23:59:59Z"
     }' \
     http://localhost:8080/api/keys
   ```

2. **Save the returned API key value**

   The response will include a `key` field with the generated API key. Save this value securely, as it will not be displayed again.

3. **Use the API key in your application**

   ```
   X-API-Key: generated-api-key-value
   ```

#### Revoking an API Key

1. **List all API keys to find the ID**

   ```bash
   curl -H "X-API-Key: your-admin-api-key" http://localhost:8080/api/keys
   ```

2. **Delete the API key by ID**

   ```bash
   curl -X DELETE \
     -H "X-API-Key: your-admin-api-key" \
     http://localhost:8080/api/keys/1
   ```

### Managing Users

#### Creating a User for NTRIP Client Access

1. **Create the user**

   ```bash
   curl -X POST \
     -H "Content-Type: application/json" \
     -H "X-API-Key: your-admin-api-key" \
     -d '{
       "username": "field_team",
       "password": "secure_password",
       "mounts_allowed": "RTCM3_MOUNT,CMR_MOUNT",
       "max_connections": 3
     }' \
     http://localhost:8080/api/users
   ```

2. **Provide the credentials to the NTRIP client**

   The user can now connect to the allowed mountpoints using the provided username and password.

#### Updating User Permissions

1. **Update the user's allowed mountpoints**

   ```bash
   curl -X PUT \
     -H "Content-Type: application/json" \
     -H "X-API-Key: your-admin-api-key" \
     -d '{
       "mounts_allowed": "RTCM3_MOUNT,CMR_MOUNT,NEW_MOUNT"
     }' \
     http://localhost:8080/api/users/field_team
   ```

### Managing Mountpoints

#### Creating a New Mountpoint

1. **Create the mountpoint**

   ```bash
   curl -X POST \
     -H "Content-Type: application/json" \
     -H "X-API-Key: your-admin-api-key" \
     -d '{
       "name": "RTK_BASE1",
       "password": "mount_password",
       "protocol": "NTRIP/2.0"
     }' \
     http://localhost:8080/api/mounts
   ```

2. **Configure the NTRIP server**

   The NTRIP server can now connect to this mountpoint using the provided name and password.

#### Setting a Mountpoint to Maintenance Mode

1. **Update the mountpoint status**

   ```bash
   curl -X PUT \
     -H "Content-Type: application/json" \
     -H "X-API-Key: your-admin-api-key" \
     -d '{
       "status": "maintenance"
     }' \
     http://localhost:8080/api/mounts/RTK_BASE1/status
   ```

   This will temporarily make the mountpoint unavailable to clients with an appropriate status message.

## Troubleshooting

### Common Issues

1. **Authentication Errors**

   - Ensure the `X-API-Key` header is correctly set
   - Verify the API key has not expired
   - Check that the API key has the required permissions

2. **Database Errors**

   - Ensure the database directory exists and is writable
   - Check for disk space issues
   - Verify SQLite is properly installed with CGO enabled

3. **Connection Issues**

   - Verify the admin API port is not blocked by a firewall
   - Check that the server is running and listening on the correct port
   - Ensure there are no port conflicts with other services

### Logs

The server logs contain valuable information for troubleshooting. Set the log level to `debug` for more detailed information:

```bash
go run cmd/ntrip-server/main.go --log-level debug
```

### Getting Help

If you encounter issues not covered in this documentation, please:

1. Check the server logs for error messages
2. Review the API reference for correct endpoint usage
3. File an issue on the GitHub repository with detailed information about the problem
