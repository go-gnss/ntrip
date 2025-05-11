# Set the admin API key
$ADMIN_API_KEY = "your-admin-api-key-here"

# Set the base URL
$BASE_URL = "http://localhost:8080"

# Create a new API key
Write-Host "Creating a new API key..."
$apiKeyResponse = Invoke-RestMethod -Method POST `
  -Headers @{
    "Content-Type" = "application/json"
    "X-API-Key" = $ADMIN_API_KEY
  } `
  -Body '{
    "name": "Test API Key",
    "permissions": "users:read,mounts:read",
    "expires": "2025-01-01T00:00:00Z"
  }' `
  -Uri "$BASE_URL/api/keys"

Write-Host "API Key Response: $($apiKeyResponse | ConvertTo-Json -Depth 10)"
$NEW_API_KEY = $apiKeyResponse.key
Write-Host "New API Key: $NEW_API_KEY"

# Create a new user
Write-Host "Creating a new user..."
Invoke-RestMethod -Method POST `
  -Headers @{
    "Content-Type" = "application/json"
    "X-API-Key" = $ADMIN_API_KEY
  } `
  -Body '{
    "username": "testuser",
    "password": "testpassword",
    "mounts_allowed": "MOUNT1,MOUNT2",
    "max_connections": 5
  }' `
  -Uri "$BASE_URL/api/users"

# Create a new mountpoint
Write-Host "Creating a new mountpoint..."
Invoke-RestMethod -Method POST `
  -Headers @{
    "Content-Type" = "application/json"
    "X-API-Key" = $ADMIN_API_KEY
  } `
  -Body '{
    "name": "TESTMOUNT",
    "password": "testpassword",
    "protocol": "NTRIP/2.0"
  }' `
  -Uri "$BASE_URL/api/mounts"

# List users
Write-Host "Listing users..."
$users = Invoke-RestMethod -Method GET `
  -Headers @{
    "X-API-Key" = $ADMIN_API_KEY
  } `
  -Uri "$BASE_URL/api/users"
Write-Host "Users: $($users | ConvertTo-Json -Depth 10)"

# List mountpoints
Write-Host "Listing mountpoints..."
$mounts = Invoke-RestMethod -Method GET `
  -Headers @{
    "X-API-Key" = $ADMIN_API_KEY
  } `
  -Uri "$BASE_URL/api/mounts"
Write-Host "Mountpoints: $($mounts | ConvertTo-Json -Depth 10)"

# Update mountpoint status
Write-Host "Updating mountpoint status..."
$updatedMount = Invoke-RestMethod -Method PUT `
  -Headers @{
    "Content-Type" = "application/json"
    "X-API-Key" = $ADMIN_API_KEY
  } `
  -Body '{
    "status": "maintenance"
  }' `
  -Uri "$BASE_URL/api/mounts/TESTMOUNT/status"
Write-Host "Updated Mountpoint: $($updatedMount | ConvertTo-Json -Depth 10)"

# Get mountpoint
Write-Host "Getting mountpoint..."
$mount = Invoke-RestMethod -Method GET `
  -Headers @{
    "X-API-Key" = $ADMIN_API_KEY
  } `
  -Uri "$BASE_URL/api/mounts/TESTMOUNT"
Write-Host "Mountpoint: $($mount | ConvertTo-Json -Depth 10)"

# Delete mountpoint
Write-Host "Deleting mountpoint..."
Invoke-RestMethod -Method DELETE `
  -Headers @{
    "X-API-Key" = $ADMIN_API_KEY
  } `
  -Uri "$BASE_URL/api/mounts/TESTMOUNT"

# Delete user
Write-Host "Deleting user..."
Invoke-RestMethod -Method DELETE `
  -Headers @{
    "X-API-Key" = $ADMIN_API_KEY
  } `
  -Uri "$BASE_URL/api/users/testuser"
