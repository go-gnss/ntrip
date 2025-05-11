# Using the Admin API with Docker

This guide explains how to use the Admin API when running the NTRIP server in Docker.

## Prerequisites

- NTRIP server running in Docker (see [Docker Deployment Guide](docker.md))
- A secure admin API key set in your `.env` file

## Authentication

All Admin API endpoints require authentication using the `X-API-Key` header. The admin API key is set via the `ADMIN_API_KEY` environment variable in your `.env` file or directly in the `docker-compose.yml` file.

## Accessing the Admin API

The Admin API is available at `http://localhost:8080` by default. You can change the port by modifying the `--admin-port` flag in the `docker-compose.yml` file.

## Example API Calls

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
    "protocol": "NTRIP/2.0",
    "status": "online"
  }' \
  http://localhost:8080/api/mounts
```

## Generating a Secure Admin API Key

For production use, you should generate a secure random API key. You can use the following commands:

### Linux/macOS

```bash
# Generate a random 32-character key
openssl rand -base64 24 | tr -d '/+=' > .env
echo "ADMIN_API_KEY=$(cat .env)" > .env
echo "LOG_LEVEL=info" >> .env
```

### Windows PowerShell

```powershell
# Generate a random 32-character key
$key = -join ((65..90) + (97..122) + (48..57) | Get-Random -Count 32 | ForEach-Object {[char]$_})
Set-Content -Path .env -Value "ADMIN_API_KEY=$key`nLOG_LEVEL=info"
```

## Security Best Practices

1. **Use HTTPS**: For production environments, enable TLS for the admin API by providing certificate files and uncommenting the TLS configuration in the `docker-compose.yml` file.

2. **Restrict Access**: Use a firewall or network configuration to restrict access to the admin API port (8080 by default).

3. **Regular Key Rotation**: Regularly rotate your admin API key and any other API keys created through the admin API.

4. **Least Privilege**: When creating API keys for other applications, assign only the necessary permissions.

## Troubleshooting

### Authentication Failures

If you receive a 401 Unauthorized response, check that:

1. The `ADMIN_API_KEY` environment variable is correctly set in your `.env` file or `docker-compose.yml`.
2. You're using the correct API key in the `X-API-Key` header.
3. The API key has not expired.

### Database Issues

If you encounter database-related errors:

1. Check that the SQLite database file is being correctly persisted in the Docker volume.
2. Ensure the container has write permissions to the database directory.

```bash
# Check the database file
docker-compose exec ntrip-server ls -la /app/data

# If needed, fix permissions
docker-compose down
docker volume rm ntrip_ntrip-data
docker-compose up -d
```

## Further Reading

- [Admin API Documentation](admin.md) - Complete documentation of all Admin API endpoints
- [Docker Deployment Guide](docker.md) - Comprehensive guide to deploying the NTRIP server with Docker
