package ntrip

import (
	"io"
	"net"
	"net/http"
	"strings"
)

// NewClientRequest constructs an http.Request which can be used as an NTRIP v2 Client
func NewClientRequest(url string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
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
	req, err := http.NewRequest(http.MethodPost, url, r)
	req.TransferEncoding = []string{"chunked"}
	req.Header.Set("User-Agent", "NTRIP go-gnss/ntrip/server")
	req.Header.Set(NTRIPVersionHeaderKey, NTRIPVersionHeaderValueV2)
	return req, err
}

// TODO: Remove v1 client
func NewClientV1(host string, path, username, password string) (io.ReadCloser, error) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	// V1 requests are valid HTTP, but the response may not be
	req, err := http.NewRequest(http.MethodGet, "tcp://"+host+path, strings.NewReader(""))
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(username, password)
	req.Header.Add("User-Agent", "NTRIP go-gnss/ntrip/client")

	// TODO: Read response headers
	return conn, req.Write(conn)
}
