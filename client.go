package ntrip

import (
	"io"
	"net"
	"net/http"
	"strings"
)

// TODO: Consider whether it would be better for the Client / Server API to hang of a type much like
//  the paho mqtt clients

// NewClientRequestV2 constructs an http.Request which can be used as an NTRIP v2 Client
func NewClientRequestV2(url string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "NTRIP go-gnss/ntrip/client")
	req.Header.Set(NTRIPVersionHeaderKey, NTRIPVersionHeaderValueV2)
	return req, err
}

// NewClientV1
// TODO: Consider making the v1 and v2 API more similar. I like that the v2 client returns a
//  http.Request object, as it allows the caller to modify request headers etc.
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
