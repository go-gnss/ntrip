# Docker Deployment for NTRIP Caster and Server

This directory contains Docker configuration files for deploying the NTRIP caster and server.

## Prerequisites

- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)

## Quick Start

1. Clone the repository:
   ```bash
   git clone https://github.com/go-gnss/ntrip.git
   cd ntrip
   ```

2. Create a `.env` file with your admin API key:
   ```bash
   cp .env.example .env
   # Edit .env and set a secure ADMIN_API_KEY
   ```

3. Start the NTRIP server:
   ```bash
   docker-compose up -d
   ```

4. Check the logs:
   ```bash
   docker-compose logs -f
   ```

5. Access the admin API at `http://localhost:8080`

## Configuration

### Environment Variables

You can configure the server by setting environment variables in the `.env` file or directly in the `docker-compose.yml` file:

- `ADMIN_API_KEY`: **Required** - Set a secure random key for admin API authentication
- `LOG_LEVEL`: Set the logging level (debug, info, warn, error)

### Ports

The following ports are exposed:

- `2101`: NTRIP HTTP port (v1/v2)
- `554`: RTSP port
- `2102`: NTRIP v1 SOURCE port
- `8080`: Admin API port

### Volumes

- `ntrip-data`: Persists the SQLite database
- `./certs:/app/certs:ro`: Mount TLS certificates (optional)

## TLS Configuration

To enable TLS for the admin API:

1. Create a `certs` directory and place your certificate and key files there:
   ```bash
   mkdir -p certs
   # Copy your certificate and key files to the certs directory
   ```

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

## Custom Configuration

To use a custom configuration:

1. Create a `docker-compose.override.yml` file:
   ```yaml
   version: '3.8'

   services:
     ntrip-server:
       command: >
         --http-port=2101
         --rtsp-port=554
         --v1source-port=2102
         --admin-port=8080
         --db-path=/app/data/ntrip.db
         --log-level=debug
         # Add any other custom flags here
   ```

2. Run Docker Compose:
   ```bash
   docker-compose up -d
   ```

## Building a Custom Image

To build a custom Docker image:

```bash
docker build -t your-registry/ntrip-server:tag .
```

## Running the Client

To run the NTRIP client using Docker:

```bash
docker run --rm -it your-registry/ntrip-server:tag /app/ntrip-client --help
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

## Advanced Usage

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
