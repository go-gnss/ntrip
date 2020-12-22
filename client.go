package ntrip

import (
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
)

// TODO: Consider whether it would be better for the Client / Server API to hang of a type
//  much like the paho mqtt clients

// NewClientRequestV2 constructs an http.Request which can be used as an NTRIP v2 Client
func NewClientRequestV2(url string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("User-Agent", "NTRIP go-gnss/ntrip/client")
	req.Header.Set(NTRIPVersionHeaderKey, NTRIPVersionHeaderValueV2)
	return req, err
}

// TODO: Consider making the v1 and v2 API more similar. I like that the v2 client returns a http.Request
//  object, as it allows the caller to modify request headers etc.
func NewClientV1(host string, path, username, password string) (io.ReadCloser, error) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	authString := fmt.Sprintf("%s:%s", username, password)
	b64Auth := make([]byte, base64.StdEncoding.EncodedLen(len(authString)))
	base64.StdEncoding.Encode(b64Auth, []byte(authString))

	_, err = fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nUser-Agent: NTRIP go-gnss/ntrip/client\r\nAuthorization: Basic %s\r\n\r\n", path, host, b64Auth)

	// TODO: Read response headers
	return conn, err
}
