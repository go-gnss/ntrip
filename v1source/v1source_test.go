package v1source

import (
	"bufio"
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/go-gnss/ntrip"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSourceService implements the ntrip.SourceService interface for testing
type MockSourceService struct {
	Sourcetable ntrip.Sourcetable
	Publishers  map[string]io.WriteCloser
}

func NewMockSourceService() *MockSourceService {
	return &MockSourceService{
		Sourcetable: ntrip.Sourcetable{
			Mounts: []ntrip.StreamEntry{
				{
					Name:           "TEST00AUS0",
					Format:         "RTCM 3.2",
					Authentication: "N",
				},
			},
		},
		Publishers: make(map[string]io.WriteCloser),
	}
}

func (m *MockSourceService) GetSourcetable() ntrip.Sourcetable {
	return m.Sourcetable
}

func (m *MockSourceService) Publisher(ctx context.Context, mount, username, password string) (io.WriteCloser, error) {
	if mount != "TEST00AUS0" {
		return nil, ntrip.ErrorNotFound
	}

	if username != "" && (username != "username" || password != "password") {
		return nil, ntrip.ErrorNotAuthorized
	}

	if _, exists := m.Publishers[mount]; exists {
		return nil, ntrip.ErrorConflict
	}

	r, w := io.Pipe()
	m.Publishers[mount] = w

	// Read from the pipe in a goroutine
	go func() {
		io.Copy(io.Discard, r)
	}()

	return w, nil
}

func (m *MockSourceService) Subscriber(ctx context.Context, mount, username, password string) (chan []byte, error) {
	return nil, ntrip.ErrorNotFound
}

// MockConn is a mock implementation of net.Conn for testing
type MockConn struct {
	ReadData  string
	WriteData strings.Builder
	Closed    bool
}

func NewMockConn(readData string) *MockConn {
	return &MockConn{
		ReadData: readData,
	}
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	if m.ReadData == "" {
		return 0, io.EOF
	}

	n = copy(b, m.ReadData)
	m.ReadData = m.ReadData[n:]
	return n, nil
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	return m.WriteData.Write(b)
}

func (m *MockConn) Close() error {
	m.Closed = true
	return nil
}

func (m *MockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234}
}

func (m *MockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5678}
}

func (m *MockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *MockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func TestHandleConnection(t *testing.T) {
	// Create a logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a mock source service
	svc := NewMockSourceService()

	// Create the server
	server := NewServer("", svc, logger)

	// Test cases
	tests := []struct {
		name           string
		input          string
		expectedOutput string
		expectedError  bool
	}{
		{
			name:           "Valid SOURCE request",
			input:          "SOURCE password TEST00AUS0\r\n\r\n",
			expectedOutput: "OK\r\n",
			expectedError:  false,
		},
		{
			name:           "Invalid request format",
			input:          "INVALID request\r\n\r\n",
			expectedOutput: "ERROR - Bad Request\r\n",
			expectedError:  true,
		},
		{
			name:           "Mount point not found",
			input:          "SOURCE password NONEXISTENT\r\n\r\n",
			expectedOutput: "ERROR - Mount Point Does Not Exist\r\n",
			expectedError:  true,
		},
		{
			name:           "Mount point in use",
			input:          "SOURCE password TEST00AUS0\r\n\r\n",
			expectedOutput: "OK\r\n",
			expectedError:  false,
		},
		{
			name:           "With authentication",
			input:          "SOURCE password TEST00AUS0\r\nAuthorization: Basic dXNlcm5hbWU6cGFzc3dvcmQ=\r\n\r\n",
			expectedOutput: "OK\r\n",
			expectedError:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Reset the mock service for each test
			svc = NewMockSourceService()
			server.svc = svc

			// Create a mock connection
			conn := NewMockConn(tc.input)

			// Handle the connection
			server.handleConnection(conn)

			// Check the output
			assert.Contains(t, conn.WriteData.String(), tc.expectedOutput)

			// Check if the connection was closed
			assert.True(t, conn.Closed)
		})
	}
}

func TestServer(t *testing.T) {
	// Create a logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a mock source service
	svc := NewMockSourceService()

	// Create the server with a random port
	server := NewServer("127.0.0.1:0", svc, logger)

	// Start the server in a goroutine
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for the server to start
	time.Sleep(100 * time.Millisecond)

	// Get the server address
	addr := server.listener.Addr().String()

	// Clean up
	defer server.Close()

	// Connect to the server
	conn, err := net.Dial("tcp", addr)
	require.NoError(t, err)
	defer conn.Close()

	// Send a SOURCE request
	_, err = conn.Write([]byte("SOURCE password TEST00AUS0\r\n\r\n"))
	require.NoError(t, err)

	// Read the response
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	require.NoError(t, err)

	// Check the response
	assert.Equal(t, "OK\r\n", response)

	// Send some data
	_, err = conn.Write([]byte("test data"))
	require.NoError(t, err)

	// Close the connection
	conn.Close()
}
