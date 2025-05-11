package ntrip

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

// DefaultHTTPClient returns a properly configured HTTP client with appropriate timeouts
func DefaultHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			MaxIdleConns:        100,
			MaxConnsPerHost:     10,
		},
	}
}

// NewClientRequest constructs an http.Request which can be used as an NTRIP v2 Client
func NewClientRequest(url string) (*http.Request, error) {
	return NewClientRequestWithContext(context.Background(), url)
}

// NewClientRequestWithContext constructs an http.Request with context which can be used as an NTRIP v2 Client
func NewClientRequestWithContext(ctx context.Context, url string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return req, err
	}
	req.Header.Set("User-Agent", "NTRIP go-gnss/ntrip/client")
	req.Header.Set(NTRIPVersionHeaderKey, NTRIPVersionHeaderValueV2)
	return req, err
}

// NewServerRequest constructs an http.Request which can be used as an NTRIP v2 Server
// Effectively a chunked encoding POST request which is not expected to close
func NewServerRequest(url string, r io.ReadCloser) (*http.Request, error) {
	return NewServerRequestWithContext(context.Background(), url, r)
}

// NewServerRequestWithContext constructs an http.Request with context which can be used as an NTRIP v2 Server
// Effectively a chunked encoding POST request which is not expected to close
func NewServerRequestWithContext(ctx context.Context, url string, r io.ReadCloser) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, r)
	if err != nil {
		return nil, err
	}
	req.TransferEncoding = []string{"chunked"}
	req.Header.Set("User-Agent", "NTRIP go-gnss/ntrip/server")
	req.Header.Set(NTRIPVersionHeaderKey, NTRIPVersionHeaderValueV2)
	return req, err
}

// TODO: Remove v1 client
// Deprecated: Use NewClientRequest with DefaultHTTPClient instead
func NewClientV1(host string, path, username, password string) (io.ReadCloser, error) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	// V1 requests are valid HTTP, but the response may not be
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "tcp://"+host+path, strings.NewReader(""))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(username, password)
	req.Header.Add("User-Agent", "NTRIP go-gnss/ntrip/client")

	// TODO: Read response headers
	return conn, req.Write(conn)
}
