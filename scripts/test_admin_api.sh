#!/bin/bash

# Set the admin API key
ADMIN_API_KEY="your-admin-api-key-here"

# Set the base URL
BASE_URL="http://localhost:8080"

# Create a new API key
echo "Creating a new API key..."
API_KEY_RESPONSE=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $ADMIN_API_KEY" \
  -d '{
    "name": "Test API Key",
    "permissions": "users:read,mounts:read",
    "expires": "2025-01-01T00:00:00Z"
  }' \
  $BASE_URL/api/keys)

echo "API Key Response: $API_KEY_RESPONSE"
NEW_API_KEY=$(echo $API_KEY_RESPONSE | grep -o '"key":"[^"]*' | cut -d'"' -f4)
echo "New API Key: $NEW_API_KEY"

# Create a new user
echo "Creating a new user..."
curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $ADMIN_API_KEY" \
  -d '{
    "username": "testuser",
    "password": "testpassword",
    "mounts_allowed": "MOUNT1,MOUNT2",
    "max_connections": 5
  }' \
  $BASE_URL/api/users

# Create a new mountpoint
echo "Creating a new mountpoint..."
curl -s -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $ADMIN_API_KEY" \
  -d '{
    "name": "TESTMOUNT",
    "password": "testpassword",
    "protocol": "NTRIP/2.0"
  }' \
  $BASE_URL/api/mounts

# List users
echo "Listing users..."
curl -s -X GET \
  -H "X-API-Key: $ADMIN_API_KEY" \
  $BASE_URL/api/users

# List mountpoints
echo "Listing mountpoints..."
curl -s -X GET \
  -H "X-API-Key: $ADMIN_API_KEY" \
  $BASE_URL/api/mounts

# Update mountpoint status
echo "Updating mountpoint status..."
curl -s -X PUT \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $ADMIN_API_KEY" \
  -d '{
    "status": "maintenance"
  }' \
  $BASE_URL/api/mounts/TESTMOUNT/status

# Get mountpoint
echo "Getting mountpoint..."
curl -s -X GET \
  -H "X-API-Key: $ADMIN_API_KEY" \
  $BASE_URL/api/mounts/TESTMOUNT

# Delete mountpoint
echo "Deleting mountpoint..."
curl -s -X DELETE \
  -H "X-API-Key: $ADMIN_API_KEY" \
  $BASE_URL/api/mounts/TESTMOUNT

# Delete user
echo "Deleting user..."
curl -s -X DELETE \
  -H "X-API-Key: $ADMIN_API_KEY" \
  $BASE_URL/api/users/testuser
