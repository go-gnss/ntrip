#!/bin/bash

# Start script for NTRIP Docker deployment

# Function to display help
show_help() {
    echo "NTRIP Docker Deployment Script"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  server      Start the NTRIP server"
    echo "  relay       Start the NTRIP relay"
    echo "  client      Start the NTRIP client"
    echo "  stop        Stop all running containers"
    echo "  logs        Show logs for running containers"
    echo "  genkey      Generate a secure random admin API key"
    echo "  help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 server   # Start the NTRIP server"
    echo "  $0 logs     # Show logs for running containers"
    echo "  $0 genkey   # Generate a secure random admin API key"
}

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo "Error: Docker is not installed or not in PATH"
    exit 1
fi

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo "Error: Docker Compose is not installed or not in PATH"
    exit 1
fi

# Get the directory where this script is located
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )"

# Change to the script directory
cd "$SCRIPT_DIR" || exit 1

# Process command
case "$1" in
    server)
        echo "Starting NTRIP server..."
        # Check if .env file exists
        if [ ! -f "../.env" ]; then
            echo "Warning: .env file not found. Creating from example..."
            cp "../.env.example" "../.env"
            echo "Please edit ../.env to set a secure ADMIN_API_KEY"
        fi
        docker-compose -f ../docker-compose.yml up -d
        ;;
    relay)
        echo "Starting NTRIP relay..."
        docker-compose -f docker-compose.relay.yml up -d
        ;;
    client)
        echo "Starting NTRIP client..."
        docker-compose -f docker-compose.client.yml up -d
        ;;
    stop)
        echo "Stopping all containers..."
        docker-compose -f ../docker-compose.yml down
        docker-compose -f docker-compose.relay.yml down
        docker-compose -f docker-compose.client.yml down
        ;;
    logs)
        echo "Showing logs..."
        docker-compose -f ../docker-compose.yml logs -f
        ;;
    genkey)
        echo "Generating secure random admin API key..."
        # Generate a random 32-character key
        KEY=$(openssl rand -base64 24 | tr -d '/+=')
        echo "Generated key: $KEY"

        # Check if .env file exists
        if [ -f "../.env" ]; then
            # Update existing .env file
            sed -i "s/^ADMIN_API_KEY=.*/ADMIN_API_KEY=$KEY/" "../.env"
            echo "Updated ADMIN_API_KEY in ../.env"
        else
            # Create new .env file
            echo "ADMIN_API_KEY=$KEY" > "../.env"
            echo "LOG_LEVEL=info" >> "../.env"
            echo "Created new ../.env file with ADMIN_API_KEY"
        fi

        echo "Done! Your admin API key has been set."
        ;;
    help|*)
        show_help
        ;;
esac

exit 0
