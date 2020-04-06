package ntrip

import (
	"errors"
	"github.com/benburkert/http"
	"io"
	"net/url"
)

// Server wraps http.Request for NTRIP server requests. Effectively a chunked
// encoding POST request which body is not expected to close.
type Server struct {
	*http.Request
	writer *io.PipeWriter
}

// NewServer constructs a Server given the URL to which to stream data
func NewServer(ntripCasterURL string) (server *Server, err error) {
	u, err := url.Parse(ntripCasterURL)
	server = &Server{
		Request: &http.Request{
			URL:              u,
			Method:           "POST",
			ProtoMajor:       1,
			ProtoMinor:       1,
			TransferEncoding: []string{"chunked"},
			Header:           make(map[string][]string),
		},
	}
	server.Header.Set("User-Agent", "NTRIP GoClient")
	server.Header.Set("Ntrip-Version", "Ntrip/2.0")
	return server, err
}

// Connect uses the http libraries Default Client to send the request. Uses an io PipeReader object as the body.
func (server *Server) Connect() (resp *http.Response, err error) {
	reader, writer := io.Pipe()
	server.Request.Body = reader
	server.writer = writer
	return http.DefaultClient.Do(server.Request)
}

// Write attempts to write data into an io PipeWriter. Returns an error if server is not connected.
func (server *Server) Write(data []byte) (n int, err error) {
	if server.writer == nil {
		return 0, errors.New("not connected")
	}
	// TODO: Check if connection is still open somehow
	return server.writer.Write(data)
}
