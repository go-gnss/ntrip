package rtsp

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
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
	DataChannel chan []byte
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
		DataChannel: make(chan []byte, 10),
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

	if m.DataChannel != nil {
		return nil, ntrip.ErrorConflict
	}

	m.DataChannel = make(chan []byte, 10)
	return channelWriter{m.DataChannel}, nil
}

func (m *MockSourceService) Subscriber(ctx context.Context, mount, username, password string) (chan []byte, error) {
	if mount != "TEST00AUS0" {
		return nil, ntrip.ErrorNotFound
	}

	if username != "" && (username != "username" || password != "password") {
		return nil, ntrip.ErrorNotAuthorized
	}

	if m.DataChannel == nil {
		m.DataChannel = make(chan []byte, 10)
		// Add some test data
		m.DataChannel <- []byte("test data")
	}

	return m.DataChannel, nil
}

// channelWriter is a helper type that implements io.WriteCloser
type channelWriter struct {
	ch chan []byte
}

func (c channelWriter) Write(p []byte) (n int, err error) {
	c.ch <- p
	return len(p), nil
}

func (c channelWriter) Close() error {
	close(c.ch)
	return nil
}

// MockConn implements net.Conn for testing
type MockConn struct {
	ReadBuffer  *bytes.Buffer
	WriteBuffer *bytes.Buffer
}

func NewMockConn() *MockConn {
	return &MockConn{
		ReadBuffer:  bytes.NewBuffer(nil),
		WriteBuffer: bytes.NewBuffer(nil),
	}
}

func (m *MockConn) Read(b []byte) (n int, err error) {
	return m.ReadBuffer.Read(b)
}

func (m *MockConn) Write(b []byte) (n int, err error) {
	return m.WriteBuffer.Write(b)
}

func (m *MockConn) Close() error {
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

func TestRTSPHandler(t *testing.T) {
	// Create a mock source service
	svc := NewMockSourceService()

	// Create a logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create the RTSP handler
	handler := RTSPHandler(svc, logger)

	// Test cases
	tests := []struct {
		name           string
		method         Method
		path           string
		headers        map[string]string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "OPTIONS request",
			method:         OPTIONS,
			path:           "/TEST00AUS0",
			headers:        map[string]string{"CSeq": "1"},
			expectedStatus: StatusOK,
			expectedBody:   "",
		},
		{
			name:           "DESCRIBE request",
			method:         DESCRIBE,
			path:           "/TEST00AUS0",
			headers:        map[string]string{"CSeq": "2"},
			expectedStatus: StatusOK,
			expectedBody:   "v=0",
		},
		{
			name:           "DESCRIBE request for non-existent mount",
			method:         DESCRIBE,
			path:           "/NONEXISTENT",
			headers:        map[string]string{"CSeq": "3"},
			expectedStatus: StatusNotFound,
			expectedBody:   "",
		},
		{
			name:           "SETUP request",
			method:         SETUP,
			path:           "/TEST00AUS0",
			headers:        map[string]string{"CSeq": "4", "Transport": "RTP/AVP;unicast;client_port=5000-5001"},
			expectedStatus: StatusOK,
			expectedBody:   "",
		},
		{
			name:           "SETUP request without Transport header",
			method:         SETUP,
			path:           "/TEST00AUS0",
			headers:        map[string]string{"CSeq": "5"},
			expectedStatus: StatusBadRequest,
			expectedBody:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock connection
			mockConn := NewMockConn()

			// Create a request
			reqStr := fmt.Sprintf("%s %s RTSP/1.0\r\n", tc.method, tc.path)
			for k, v := range tc.headers {
				reqStr += fmt.Sprintf("%s: %s\r\n", k, v)
			}
			reqStr += "\r\n"

			// Write the request to the mock connection
			mockConn.ReadBuffer.WriteString(reqStr)

			// Create a Conn
			conn := &Conn{
				conn:   mockConn,
				reader: bufio.NewReader(mockConn),
				writer: bufio.NewWriter(mockConn),
				logger: logger,
			}

			// Read the request
			req, err := conn.readRequest()
			require.NoError(t, err)
			conn.Request = req

			// Handle the request
			handler(conn)

			// Check the response
			response := mockConn.WriteBuffer.String()

			// Check status code
			statusLine := strings.Split(response, "\r\n")[0]
			assert.Contains(t, statusLine, fmt.Sprintf("%d", tc.expectedStatus))

			// Check body if expected
			if tc.expectedBody != "" {
				assert.Contains(t, response, tc.expectedBody)
			}
		})
	}
}

func TestGenerateSDPForNTRIP(t *testing.T) {
	// Test cases
	tests := []struct {
		name           string
		mount          string
		format         string
		expectedFields []string
	}{
		{
			name:   "RTCM 3.2 format",
			mount:  "TEST00AUS0",
			format: "RTCM 3.2",
			expectedFields: []string{
				"v=0",
				"s=NTRIP Stream",
				"i=TEST00AUS0",
				"m=application 0 RTP/AVP 96",
				"a=rtpmap:96 RTCM/2000",
				"a=control:rtsp://*/TEST00AUS0",
			},
		},
		{
			name:   "Unknown format",
			mount:  "TEST00AUS0",
			format: "UNKNOWN",
			expectedFields: []string{
				"v=0",
				"s=NTRIP Stream",
				"i=TEST00AUS0",
				"m=application 0 RTP/AVP 96",
				"a=rtpmap:96 GNSS/2000",
				"a=control:rtsp://*/TEST00AUS0",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sdp := GenerateSDPForNTRIP(tc.mount, tc.format)
			sdpStr := string(sdp)

			for _, field := range tc.expectedFields {
				assert.Contains(t, sdpStr, field)
			}
		})
	}
}

func TestRTSPServer(t *testing.T) {
	// Create a mock source service
	svc := NewMockSourceService()

	// Create a logger
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create the RTSP handler
	handler := RTSPHandler(svc, logger)

	// Create the server
	server := NewServer("127.0.0.1:0", handler, logger)

	// Start the server in a goroutine
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Wait for the server to start
	time.Sleep(100 * time.Millisecond)

	// Clean up
	defer server.Close()

	// The server is running, but we can't easily test it without a real RTSP client
	// This is more of an integration test that would require a real RTSP client library
	// For now, we'll just check that the server started successfully
	assert.NotNil(t, server.listener)
}
