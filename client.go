package ntrip

import (
	"github.com/benburkert/http"
	"net/url"
)

// Client wraps http.Request for NTRIP client requests
type Client struct {
	*http.Request
}

// NewClient constructs a GET request with some NTRIP specific requirements
func NewClient(casterURL string) (client *Client, err error) {
	u, err := url.Parse(casterURL)
	client = &Client{
		Request: &http.Request{
			URL:        u,
			Method:     "GET",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(map[string][]string),
		},
	}
	client.Header.Set("User-Agent", "NTRIP GoClient")
	client.Header.Set("Ntrip-Version", "Ntrip/2.0")
	return client, err
}

// Connect uses the http DefaultClient to send the constructed request. Data will
// be streamed into the Response object until the connection is closed.
func (client *Client) Connect() (resp *http.Response, err error) {
	return http.DefaultClient.Do(client.Request)
}
