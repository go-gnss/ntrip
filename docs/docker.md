# Docker Deployment Guide

This guide explains how to deploy the NTRIP caster, server, relay, and client using Docker.

## Overview

The Docker deployment provides a containerized environment for running the NTRIP components:

- **NTRIP Server**: The main server that handles NTRIP v1/v2 requests, RTSP, and admin API
- **NTRIP Relay**: A component that relays NTRIP data between two casters
- **NTRIP Client**: A simple client for connecting to NTRIP casters

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)

## Quick Start

### Setting Up Environment Variables

Before starting the server, you need to set up your environment variables, especially the admin API key:

```bash
# Copy the example environment file
cp .env.example .env

# Edit the .env file to set a secure admin API key
# This key is required for authenticating with the admin API
```

### Starting the NTRIP Server

```bash
# Linux/macOS
./docker/start.sh server

# Windows
docker\start.bat server
```

Or manually:

```bash
docker-compose up -d
```

### Viewing Logs

```bash
# Linux/macOS
./docker/start.sh logs

# Windows
docker\start.bat logs
```

Or manually:

```bash
docker-compose logs -f
```

### Stopping the Server

```bash
# Linux/macOS
./docker/start.sh stop

# Windows
docker\start.bat stop
```

Or manually:

```bash
docker-compose down
```

## Configuration

### NTRIP Server Configuration

The NTRIP server can be configured using environment variables and command-line arguments in the `docker-compose.yml` file:

```yaml
services:
  ntrip-server:
    environment:
      - ADMIN_API_KEY=${ADMIN_API_KEY:-change_this_to_a_secure_random_key}
      - LOG_LEVEL=info
    command: >
      --http-port=2101
      --rtsp-port=554
      --v1source-port=2102
      --admin-port=8080
      --db-path=/app/data/ntrip.db
      --log-level=${LOG_LEVEL:-info}
```

### NTRIP Relay Configuration

The NTRIP relay can be configured using environment variables in the `docker/docker-compose.relay.yml` file:

```yaml
services:
  ntrip-relay:
    environment:
      - SOURCE_URL=http://source-caster:2101/MOUNTPOINT
      - SOURCE_USER=username
      - SOURCE_PASS=password
      - DEST_URL=http://destination-caster:2101/MOUNTPOINT
      - DEST_USER=username
      - DEST_PASS=password
      - TIMEOUT=2
```

### NTRIP Client Configuration

The NTRIP client can be configured using environment variables in the `docker/docker-compose.client.yml` file:

```yaml
services:
  ntrip-client:
    environment:
      - NTRIP_URL=http://ntrip-server:2101/MOUNTPOINT
      - NTRIP_USER=username
      - NTRIP_PASS=password
      - OUTPUT_FILE=/data/output.rtcm
```

## Data Persistence

The NTRIP server uses a Docker volume to persist the SQLite database:

```yaml
volumes:
  ntrip-data:
    # This volume persists the SQLite database
```

## TLS Configuration

To enable TLS for the admin API:

1. Create a `certs` directory and place your certificate and key files there.
2. Uncomment the TLS configuration lines in the `docker-compose.yml` file:

```yaml
command: >
  --http-port=2101
  --rtsp-port=554
  --v1source-port=2102
  --admin-port=8080
  --db-path=/app/data/ntrip.db
  --log-level=${LOG_LEVEL:-info}
  --tls-cert=/app/certs/cert.pem
  --tls-key=/app/certs/key.pem
```

## Advanced Configuration

### Custom Networks

To connect the NTRIP components to a custom Docker network:

```yaml
services:
  ntrip-server:
    networks:
      - ntrip-network

networks:
  ntrip-network:
    external: true
```

### Running Behind a Reverse Proxy

When running behind a reverse proxy like Nginx or Traefik, configure the proxy to forward the appropriate ports to the NTRIP server.

Example Nginx configuration:

```nginx
server {
    listen 80;
    server_name ntrip.example.com;

    location / {
        proxy_pass http://ntrip-server:2101;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}

server {
    listen 80;
    server_name admin.ntrip.example.com;

    location / {
        proxy_pass http://ntrip-server:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## Troubleshooting

### Database Permissions

If you encounter database permission issues, ensure the data directory has the correct permissions:

```bash
docker-compose down
docker volume rm ntrip_ntrip-data
docker-compose up -d
```

### Connection Issues

If clients cannot connect to the server, check that the ports are correctly exposed and not blocked by a firewall.

### Checking Container Status

To check the status of running containers:

```bash
docker-compose ps
```

### Accessing Container Shell

To access a shell in a running container:

```bash
docker-compose exec ntrip-server /bin/sh
```

## Building Custom Images

To build custom Docker images:

```bash
# Build the server image
docker build -t your-registry/ntrip-server:tag .

# Build the relay image
docker build -t your-registry/ntrip-relay:tag -f docker/Dockerfile.relay .

# Build the client image
docker build -t your-registry/ntrip-client:tag -f docker/Dockerfile.client .
```

## Deployment Examples

### Example: Running Multiple NTRIP Servers

To run multiple NTRIP servers on different ports:

```yaml
version: '3.8'

services:
  ntrip-server-1:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "2101:2101"
    volumes:
      - ntrip-data-1:/app/data
    command: >
      --http-port=2101
      --db-path=/app/data/ntrip.db

  ntrip-server-2:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "2102:2101"
    volumes:
      - ntrip-data-2:/app/data
    command: >
      --http-port=2101
      --db-path=/app/data/ntrip.db

volumes:
  ntrip-data-1:
  ntrip-data-2:
```

### Example: Setting Up a Relay Network

To set up a relay network with multiple relays:

```yaml
version: '3.8'

services:
  ntrip-server:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "2101:2101"
    volumes:
      - ntrip-data:/app/data

  relay-1:
    build:
      context: .
      dockerfile: docker/Dockerfile.relay
    environment:
      - SOURCE_URL=http://external-source:2101/MOUNT1
      - DEST_URL=http://ntrip-server:2101/RELAY1

  relay-2:
    build:
      context: .
      dockerfile: docker/Dockerfile.relay
    environment:
      - SOURCE_URL=http://external-source:2101/MOUNT2
      - DEST_URL=http://ntrip-server:2101/RELAY2

volumes:
  ntrip-data:
```
