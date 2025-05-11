FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 go build -o /ntrip-server ./cmd/ntrip-server/main.go

# Create a minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata sqlite

# Create a non-root user to run the application
RUN adduser -D -h /app ntrip
USER ntrip
WORKDIR /app

# Create data directory for SQLite database
RUN mkdir -p /app/data

# Copy the binary from the builder stage
COPY --from=builder --chown=ntrip:ntrip /ntrip-server /app/ntrip-server

# Expose ports
EXPOSE 2101 554 2102 8080

# Set default command
ENTRYPOINT ["/app/ntrip-server"]
