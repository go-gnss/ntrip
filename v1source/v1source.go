package v1source

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"github.com/go-gnss/ntrip"
	"github.com/sirupsen/logrus"
)

// Server represents a server that handles NTRIP v1 SOURCE requests
type Server struct {
	Addr     string
	svc      ntrip.SourceService
	listener net.Listener
	logger   logrus.FieldLogger
	mu       sync.Mutex
	running  bool
}

// NewServer creates a new NTRIP v1 SOURCE server
func NewServer(addr string, svc ntrip.SourceService, logger logrus.FieldLogger) *Server {
	return &Server{
		Addr:    addr,
		svc:     svc,
		logger:  logger,
		running: false,
	}
}

// ListenAndServe starts the server
func (s *Server) ListenAndServe() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	addr := s.Addr
	if addr == "" {
		addr = ":2101" // Default NTRIP port
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
		return err
	}

	s.listener = listener
	s.logger.Infof("NTRIP v1 SOURCE server listening on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if !s.running {
				return nil // Server was closed
			}
			s.logger.Errorf("Error accepting connection: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

// Close closes the server
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// handleConnection handles a new connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	writer := bufio.NewWriter(conn)

	// Read the first line to determine if it's a SOURCE request
	line, err := reader.ReadString('\n')
	if err != nil {
		s.logger.Errorf("Error reading request: %v", err)
		return
	}

	line = strings.TrimSpace(line)
	parts := strings.Split(line, " ")

	// Check if it's a SOURCE request
	if len(parts) < 3 || parts[0] != "SOURCE" {
		s.logger.Errorf("Invalid request: %s", line)
		writer.WriteString("ERROR - Bad Request\r\n")
		writer.Flush()
		return
	}

	// Extract password and mount point
	password := parts[1]
	mount := parts[2]

	// Read headers until empty line
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			s.logger.Errorf("Error reading headers: %v", err)
			return
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		headerParts := strings.SplitN(line, ":", 2)
		if len(headerParts) == 2 {
			key := strings.TrimSpace(headerParts[0])
			value := strings.TrimSpace(headerParts[1])
			headers[key] = value
		}
	}

	// Extract username from headers if present
	username := ""
	if auth, ok := headers["Authorization"]; ok {
		username = extractUsername(auth)
	}

	// Try to get a publisher for the mount point
	ctx := context.Background()
	publisher, err := s.svc.Publisher(ctx, mount, username, password)
	if err != nil {
		s.logger.Errorf("Error getting publisher: %v", err)
		if err == ntrip.ErrorNotAuthorized {
			writer.WriteString("ERROR - Not Authorized\r\n")
		} else if err == ntrip.ErrorNotFound {
			writer.WriteString("ERROR - Mount Point Does Not Exist\r\n")
		} else if err == ntrip.ErrorConflict {
			writer.WriteString("ERROR - Mount Point Already In Use\r\n")
		} else {
			writer.WriteString("ERROR - Internal Server Error\r\n")
		}
		writer.Flush()
		return
	}

	// Send OK response
	writer.WriteString("OK\r\n")
	writer.Flush()

	s.logger.Infof("NTRIP v1 SOURCE connected to mount point %s", mount)

	// Copy data from the connection to the publisher
	_, err = io.Copy(publisher, reader)
	if err != nil {
		s.logger.Errorf("Error copying data: %v", err)
	}

	// Close the publisher
	publisher.Close()
	s.logger.Infof("NTRIP v1 SOURCE disconnected from mount point %s", mount)
}

// extractUsername extracts the username from an Authorization header
func extractUsername(auth string) string {
	if !strings.HasPrefix(auth, "Basic ") {
		return ""
	}

	auth = strings.TrimPrefix(auth, "Basic ")
	decoded, err := base64Decode(auth)
	if err != nil {
		return ""
	}

	parts := strings.SplitN(decoded, ":", 2)
	if len(parts) != 2 {
		return ""
	}

	return parts[0]
}

// base64Decode decodes a base64 string
func base64Decode(s string) (string, error) {
	bytes, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
