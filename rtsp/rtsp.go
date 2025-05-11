package rtsp

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-gnss/ntrip"
	"github.com/sirupsen/logrus"
)

// Method represents RTSP methods
type Method string

const (
	// OPTIONS is the RTSP OPTIONS method
	OPTIONS Method = "OPTIONS"
	// DESCRIBE is the RTSP DESCRIBE method
	DESCRIBE Method = "DESCRIBE"
	// SETUP is the RTSP SETUP method
	SETUP Method = "SETUP"
	// PLAY is the RTSP PLAY method
	PLAY Method = "PLAY"
	// PAUSE is the RTSP PAUSE method
	PAUSE Method = "PAUSE"
	// TEARDOWN is the RTSP TEARDOWN method
	TEARDOWN Method = "TEARDOWN"
)

// Status codes
const (
	StatusOK                  = 200
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusNotFound            = 404
	StatusMethodNotAllowed    = 405
	StatusUnsupportedMedia    = 415
	StatusInternalServerError = 500
)

// RTSPHeader represents RTSP headers
type RTSPHeader map[string][]string

// Add adds a key-value pair to the header
func (h RTSPHeader) Add(key, value string) {
	key = http.CanonicalHeaderKey(key)
	h[key] = append(h[key], value)
}

// Set sets a key-value pair in the header, replacing any existing values
func (h RTSPHeader) Set(key, value string) {
	key = http.CanonicalHeaderKey(key)
	h[key] = []string{value}
}

// Get gets the first value associated with the given key
func (h RTSPHeader) Get(key string) string {
	key = http.CanonicalHeaderKey(key)
	if v := h[key]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// Request represents an RTSP request
type Request struct {
	Method  Method
	URL     *url.URL
	Proto   string
	Header  RTSPHeader
	Body    io.ReadCloser
	Context context.Context
}

// Response represents an RTSP response
type Response struct {
	StatusCode int
	Proto      string
	Header     RTSPHeader
	Body       []byte
}

// Conn represents an RTSP connection
type Conn struct {
	conn      net.Conn
	reader    *bufio.Reader
	writer    *bufio.Writer
	Request   *Request
	sessionID string
	mu        sync.Mutex
	logger    logrus.FieldLogger
}

// Server represents an RTSP server
type Server struct {
	Addr     string
	Handler  HandlerFunc
	listener net.Listener
	logger   logrus.FieldLogger
}

// HandlerFunc is a function that handles RTSP requests
type HandlerFunc func(conn *Conn)

// NewServer creates a new RTSP server
func NewServer(addr string, handler HandlerFunc, logger logrus.FieldLogger) *Server {
	return &Server{
		Addr:    addr,
		Handler: handler,
		logger:  logger,
	}
}

// ListenAndServe starts the RTSP server
func (s *Server) ListenAndServe() error {
	addr := s.Addr
	if addr == "" {
		addr = ":554" // Default RTSP port
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	s.listener = listener

	s.logger.Infof("RTSP server listening on %s", addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			s.logger.Errorf("Error accepting connection: %v", err)
			continue
		}

		go s.handleConnection(conn)
	}
}

// Close closes the server
func (s *Server) Close() error {
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// handleConnection handles a new connection
func (s *Server) handleConnection(conn net.Conn) {
	rtspConn := &Conn{
		conn:   conn,
		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
		logger: s.logger,
	}

	defer conn.Close()

	// Read the request
	req, err := rtspConn.readRequest()
	if err != nil {
		s.logger.Errorf("Error reading request: %v", err)
		return
	}

	rtspConn.Request = req

	// Handle the request
	s.Handler(rtspConn)
}

// readRequest reads an RTSP request
func (c *Conn) readRequest() (*Request, error) {
	// Read the request line
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	line = strings.TrimSpace(line)
	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid request line: %s", line)
	}

	method := Method(parts[0])
	urlStr := parts[1]
	proto := parts[2]

	// Parse the URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	// Read headers
	header := make(RTSPHeader)
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		header.Add(key, value)
	}

	// Read body if Content-Length is set
	var body io.ReadCloser
	contentLength := header.Get("Content-Length")
	if contentLength != "" {
		length, err := strconv.Atoi(contentLength)
		if err != nil {
			return nil, err
		}

		if length > 0 {
			bodyBytes := make([]byte, length)
			_, err = io.ReadFull(c.reader, bodyBytes)
			if err != nil {
				return nil, err
			}

			body = io.NopCloser(strings.NewReader(string(bodyBytes)))
		}
	}

	return &Request{
		Method:  method,
		URL:     u,
		Proto:   proto,
		Header:  header,
		Body:    body,
		Context: context.Background(),
	}, nil
}

// WriteResponse writes an RTSP response
func (c *Conn) WriteResponse(resp Response) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Write status line
	statusLine := fmt.Sprintf("RTSP/1.0 %d %s\r\n", resp.StatusCode, statusText(resp.StatusCode))
	if _, err := c.writer.WriteString(statusLine); err != nil {
		return err
	}

	// Write headers
	for key, values := range resp.Header {
		for _, value := range values {
			headerLine := fmt.Sprintf("%s: %s\r\n", key, value)
			if _, err := c.writer.WriteString(headerLine); err != nil {
				return err
			}
		}
	}

	// Add Content-Length header if body is present
	if len(resp.Body) > 0 {
		contentLengthLine := fmt.Sprintf("Content-Length: %d\r\n", len(resp.Body))
		if _, err := c.writer.WriteString(contentLengthLine); err != nil {
			return err
		}
	}

	// End of headers
	if _, err := c.writer.WriteString("\r\n"); err != nil {
		return err
	}

	// Write body if present
	if len(resp.Body) > 0 {
		if _, err := c.writer.Write(resp.Body); err != nil {
			return err
		}
	}

	return c.writer.Flush()
}

// WritePacketRTP writes an RTP packet
func (c *Conn) WritePacketRTP(packet *PacketRTP) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// For now, just write the payload directly
	// In a real implementation, we would properly encode the RTP header
	if _, err := c.writer.Write(packet.Payload); err != nil {
		return err
	}

	return c.writer.Flush()
}

// statusText returns a text for the HTTP status code
func statusText(code int) string {
	switch code {
	case StatusOK:
		return "OK"
	case StatusBadRequest:
		return "Bad Request"
	case StatusUnauthorized:
		return "Unauthorized"
	case StatusNotFound:
		return "Not Found"
	case StatusMethodNotAllowed:
		return "Method Not Allowed"
	case StatusUnsupportedMedia:
		return "Unsupported Media Type"
	case StatusInternalServerError:
		return "Internal Server Error"
	default:
		return "Unknown"
	}
}

// PacketRTP represents an RTP packet
type PacketRTP struct {
	Header  RTPHeader
	Payload []byte
}

// RTPHeader represents an RTP header
type RTPHeader struct {
	Version          uint8
	Padding          bool
	Extension        bool
	CSRCCount        uint8
	Marker           bool
	PayloadType      uint8
	SequenceNumber   uint16
	Timestamp        uint32
	SSRC             uint32
	CSRC             []uint32
	ExtensionProfile uint16
	ExtensionLength  uint16
	ExtensionData    []byte
}

// GenerateSDPForNTRIP generates an SDP description for NTRIP streams
func GenerateSDPForNTRIP(mount string, format string) []byte {
	sdp := []string{
		"v=0",
		"o=- 0 0 IN IP4 0.0.0.0",
		"s=NTRIP Stream",
		fmt.Sprintf("i=%s", mount),
		"c=IN IP4 0.0.0.0",
		"t=0 0",
	}

	// Add media description based on format
	switch format {
	case "RTCM 3.0", "RTCM 3.1", "RTCM 3.2", "RTCM 3.3", "RTCM 3":
		sdp = append(sdp, "m=application 0 RTP/AVP 96")
		sdp = append(sdp, "a=rtpmap:96 RTCM/2000")
		sdp = append(sdp, "a=control:rtsp://*/"+mount)
	default:
		sdp = append(sdp, "m=application 0 RTP/AVP 96")
		sdp = append(sdp, "a=rtpmap:96 GNSS/2000")
		sdp = append(sdp, "a=control:rtsp://*/"+mount)
	}

	return []byte(strings.Join(sdp, "\r\n") + "\r\n")
}

// RTSPHandler creates an RTSP handler function for the NTRIP caster
func RTSPHandler(svc ntrip.SourceService, logger logrus.FieldLogger) HandlerFunc {
	return func(conn *Conn) {
		req := conn.Request
		mount := strings.TrimPrefix(req.URL.Path, "/")

		// Extract authentication info
		username, password, _ := extractAuth(req.Header.Get("Authorization"))

		switch req.Method {
		case OPTIONS:
			// Respond with supported methods
			conn.WriteResponse(Response{
				StatusCode: StatusOK,
				Header: RTSPHeader{
					"Public": []string{"OPTIONS, DESCRIBE, SETUP, PLAY, PAUSE, TEARDOWN"},
					"CSeq":   []string{req.Header.Get("CSeq")},
				},
			})

		case DESCRIBE:
			// Check if mount exists
			sourcetable := svc.GetSourcetable()
			var format string
			mountExists := false

			for _, m := range sourcetable.Mounts {
				if m.Name == mount {
					mountExists = true
					format = m.Format
					break
				}
			}

			if !mountExists {
				conn.WriteResponse(Response{
					StatusCode: StatusNotFound,
					Header: RTSPHeader{
						"CSeq": []string{req.Header.Get("CSeq")},
					},
				})
				return
			}

			// Generate SDP description
			sdp := GenerateSDPForNTRIP(mount, format)

			conn.WriteResponse(Response{
				StatusCode: StatusOK,
				Header: RTSPHeader{
					"Content-Type": []string{"application/sdp"},
					"CSeq":         []string{req.Header.Get("CSeq")},
				},
				Body: sdp,
			})

		case SETUP:
			// Parse Transport header
			transport := req.Header.Get("Transport")
			if transport == "" {
				conn.WriteResponse(Response{
					StatusCode: StatusBadRequest,
					Header: RTSPHeader{
						"CSeq": []string{req.Header.Get("CSeq")},
					},
				})
				return
			}

			// Generate session ID
			sessionID := generateSessionID()
			conn.sessionID = sessionID

			conn.WriteResponse(Response{
				StatusCode: StatusOK,
				Header: RTSPHeader{
					"Transport": []string{transport + ";server_port=5000-5001"},
					"Session":   []string{sessionID},
					"CSeq":      []string{req.Header.Get("CSeq")},
				},
			})

		case PLAY:
			// Check session ID
			if req.Header.Get("Session") != conn.sessionID {
				conn.WriteResponse(Response{
					StatusCode: StatusBadRequest,
					Header: RTSPHeader{
						"CSeq": []string{req.Header.Get("CSeq")},
					},
				})
				return
			}

			// Try to subscribe to the mount
			sub, err := svc.Subscriber(req.Context, mount, username, password)
			if err != nil {
				statusCode := StatusInternalServerError
				if err == ntrip.ErrorNotAuthorized {
					statusCode = StatusUnauthorized
				} else if err == ntrip.ErrorNotFound {
					statusCode = StatusNotFound
				}

				conn.WriteResponse(Response{
					StatusCode: statusCode,
					Header: RTSPHeader{
						"CSeq": []string{req.Header.Get("CSeq")},
					},
				})
				return
			}

			// Send OK response
			conn.WriteResponse(Response{
				StatusCode: StatusOK,
				Header: RTSPHeader{
					"Session":  []string{conn.sessionID},
					"CSeq":     []string{req.Header.Get("CSeq")},
					"RTP-Info": []string{fmt.Sprintf("url=%s;seq=0;rtptime=0", req.URL.String())},
				},
			})

			// Start streaming data
			seqNum := uint16(0)
			timestamp := uint32(0)
			ssrc := uint32(0x12345678) // Example SSRC

			for data := range sub {
				packet := &PacketRTP{
					Header: RTPHeader{
						Version:        2,
						Padding:        false,
						Extension:      false,
						CSRCCount:      0,
						Marker:         false,
						PayloadType:    96, // Dynamic payload type
						SequenceNumber: seqNum,
						Timestamp:      timestamp,
						SSRC:           ssrc,
					},
					Payload: data,
				}

				if err := conn.WritePacketRTP(packet); err != nil {
					logger.Errorf("Error writing RTP packet: %v", err)
					break
				}

				seqNum++
				timestamp += uint32(len(data))
			}

		case TEARDOWN:
			// Check session ID
			if req.Header.Get("Session") != conn.sessionID {
				conn.WriteResponse(Response{
					StatusCode: StatusBadRequest,
					Header: RTSPHeader{
						"CSeq": []string{req.Header.Get("CSeq")},
					},
				})
				return
			}

			conn.WriteResponse(Response{
				StatusCode: StatusOK,
				Header: RTSPHeader{
					"CSeq": []string{req.Header.Get("CSeq")},
				},
			})

			// Close the connection
			conn.conn.Close()

		default:
			conn.WriteResponse(Response{
				StatusCode: StatusMethodNotAllowed,
				Header: RTSPHeader{
					"CSeq": []string{req.Header.Get("CSeq")},
				},
			})
		}
	}
}

// extractAuth extracts username and password from Authorization header
func extractAuth(authHeader string) (string, string, bool) {
	if authHeader == "" {
		return "", "", false
	}

	if !strings.HasPrefix(authHeader, "Basic ") {
		return "", "", false
	}

	auth := strings.TrimPrefix(authHeader, "Basic ")
	decoded, err := base64.StdEncoding.DecodeString(auth)
	if err != nil {
		return "", "", false
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	return parts[0], parts[1], true
}

// generateSessionID generates a random session ID
func generateSessionID() string {
	b := make([]byte, 8)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("%x", b)
}
