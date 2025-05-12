@echo off
REM Start script for NTRIP Docker deployment on Windows

REM Function to display help
:show_help
    echo NTRIP Docker Deployment Script
    echo.
    echo Usage: %0 [command]
    echo.
    echo Commands:
    echo   server      Start the NTRIP server
    echo   relay       Start the NTRIP relay
    echo   client      Start the NTRIP client
    echo   stop        Stop all running containers
    echo   logs        Show logs for running containers
    echo   genkey      Generate a secure random admin API key
    echo   help        Show this help message
    echo.
    echo Examples:
    echo   %0 server   # Start the NTRIP server
    echo   %0 logs     # Show logs for running containers
    echo   %0 genkey   # Generate a secure random admin API key
    goto :eof

REM Check if Docker is installed
where docker >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo Error: Docker is not installed or not in PATH
    exit /b 1
)

REM Check if Docker Compose is installed
where docker-compose >nul 2>nul
if %ERRORLEVEL% neq 0 (
    echo Error: Docker Compose is not installed or not in PATH
    exit /b 1
)

REM Get the directory where this script is located
set "SCRIPT_DIR=%~dp0"

REM Change to the script directory
cd /d "%SCRIPT_DIR%" || exit /b 1

REM Process command
if "%1"=="server" (
    echo Starting NTRIP server...
    REM Check if .env file exists
    if not exist "../.env" (
        echo Warning: .env file not found. Creating from example...
        copy "../.env.example" "../.env"
        echo Please edit ../.env to set a secure ADMIN_API_KEY
    )
    docker-compose -f ../docker-compose.yml up -d
) else if "%1"=="relay" (
    echo Starting NTRIP relay...
    docker-compose -f docker-compose.relay.yml up -d
) else if "%1"=="client" (
    echo Starting NTRIP client...
    docker-compose -f docker-compose.client.yml up -d
) else if "%1"=="stop" (
    echo Stopping all containers...
    docker-compose -f ../docker-compose.yml down
    docker-compose -f docker-compose.relay.yml down
    docker-compose -f docker-compose.client.yml down
) else if "%1"=="logs" (
    echo Showing logs...
    docker-compose -f ../docker-compose.yml logs -f
) else if "%1"=="genkey" (
    echo Generating secure random admin API key...
    REM Generate a random 32-character key using PowerShell
    for /f "delims=" %%a in ('powershell -Command "$key = -join ((65..90) + (97..122) + (48..57) | Get-Random -Count 32 | ForEach-Object {[char]$_}); Write-Output $key"') do set KEY=%%a
    echo Generated key: %KEY%

    REM Check if .env file exists
    if exist "../.env" (
        REM Update existing .env file using PowerShell
        powershell -Command "(Get-Content '../.env') -replace '^ADMIN_API_KEY=.*', 'ADMIN_API_KEY=%KEY%' | Set-Content '../.env'"
        echo Updated ADMIN_API_KEY in ../.env
    ) else (
        REM Create new .env file
        echo ADMIN_API_KEY=%KEY%> "../.env"
        echo LOG_LEVEL=info>> "../.env"
        echo Created new ../.env file with ADMIN_API_KEY
    )

    echo Done! Your admin API key has been set.
) else (
    call :show_help
)

exit /b 0
