# NTRIP Caster Admin API Quick Start Guide

This quick start guide will help you get up and running with the NTRIP Caster Admin API.

## Prerequisites

- Go 1.22 or later
- CGO enabled (required for SQLite)

## Starting the Server

1. **Set the admin API key**

   ```bash
   # Linux/Mac
   export ADMIN_API_KEY=your-secure-api-key-here

   # Windows
   set ADMIN_API_KEY=your-secure-api-key-here
   ```

   Or create a `.env` file in your project directory:

   ```
   ADMIN_API_KEY=your-secure-api-key-here
   ```

2. **Start the server**

   ```bash
   go run cmd/ntrip-server/main.go --admin-port 8080 --db-path data/ntrip.db
   ```

   The server will create the database file if it doesn't exist.

## Basic Operations

### Creating an API Key

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-admin-api-key" \
  -d '{
    "name": "Client API Key",
    "permissions": "users:read,mounts:read",
    "expires": "2025-01-01T00:00:00Z"
  }' \
  http://localhost:8080/api/keys
```

### Creating a User

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-admin-api-key" \
  -d '{
    "username": "survey_team",
    "password": "secure_password",
    "mounts_allowed": "RTCM3_MOUNT,CMR_MOUNT",
    "max_connections": 3
  }' \
  http://localhost:8080/api/users
```

### Creating a Mountpoint

```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: your-admin-api-key" \
  -d '{
    "name": "RTCM3_MOUNT",
    "password": "mount_password",
    "protocol": "NTRIP/2.0"
  }' \
  http://localhost:8080/api/mounts
```

## Testing Scripts

We've included test scripts to help you verify your setup:

### Bash Script (Linux/Mac)

```bash
# Make the script executable
chmod +x scripts/test_admin_api.sh

# Edit the script to set your admin API key
nano scripts/test_admin_api.sh

# Run the script
./scripts/test_admin_api.sh
```

### PowerShell Script (Windows)

```powershell
# Edit the script to set your admin API key
notepad scripts/test_admin_api.ps1

# Run the script
.\scripts\test_admin_api.ps1
```

## Next Steps

For more detailed information, see the full [Admin API Documentation](admin.md).

## Troubleshooting

### Common Issues

1. **SQLite Errors**

   If you see errors related to SQLite, ensure CGO is enabled:

   ```bash
   # Check if CGO is enabled
   go env CGO_ENABLED
   
   # Enable CGO if needed
   set CGO_ENABLED=1  # Windows
   export CGO_ENABLED=1  # Linux/Mac
   ```

2. **Authentication Errors**

   Ensure your admin API key is correctly set in the environment or `.env` file.

3. **Port Conflicts**

   If the admin port is already in use, change it with the `--admin-port` flag.
