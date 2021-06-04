package ntrip

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	defaultHeader http.Header = map[string][]string{
		"User-Agent": {"NTRIP go-gnss/ntrip"},
	}
)

type Client struct {
	addr    string
	Timeout time.Duration
	// TODO: Setting Header at client level doesn't necessarily make sense, as you may want simultaneous GET and POST requests w/ different headers
	//  that said, reusing a client in NTRIP doesn't make a lot of sense anyway
	Header http.Header
}

// TODO: OR separate types for version (I think Client as interface is never good)
// type ClientV1 struct
// type ClientV2 struct

func NewClientV1(host string, port int) Client {
	return Client{
		addr:    fmt.Sprintf("tcp://%s:%d", host, port),
		Timeout: 0,
		Header:  defaultHeader,
	}
}

func NewClientV2(addr string) Client {
	header := defaultHeader
	header.Set(NTRIPVersionHeaderKey, NTRIPVersionHeaderValueV2)
	return Client{
		addr:    addr,
		Timeout: 0,
		Header:  header,
	}
}

func (client Client) SetBasicAuth(username, password string) {
	auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
	client.Header.Set("Authorization", "Basic "+auth)
}

func (client Client) Get(ctx context.Context, mount string) (io.Reader, error) {
	switch client.Header.Get(NTRIPVersionHeaderKey) {
	case NTRIPVersionHeaderValueV2:
		return client.getV2(ctx, mount)
	default:
		return client.getV1(ctx, mount)
	}
}

func (client Client) getV1(ctx context.Context, mount string) (io.Reader, error) {
	return nil, nil
}

func (client Client) getV2(ctx context.Context, mount string) (io.Reader, error) {
	return client.v2Request(ctx, http.MethodGet, client.addr+"/"+mount, nil)
}

func (client Client) Serve(ctx context.Context, mount string, body io.Reader) error {
	switch client.Header.Get(NTRIPVersionHeaderKey) {
	case NTRIPVersionHeaderValueV2:
		return client.serveV2(ctx, mount, body)
	default:
		return client.serveV1(ctx, mount, body)
	}
}

func (client Client) serveV1(ctx context.Context, mount string, body io.Reader) error {
	return nil
}

func (client Client) serveV2(ctx context.Context, mount string, body io.Reader) error {
	// v2Request(ctx, http.MethodGet, client.addr+"/"+mount, body)
	_, err := client.v2Request(ctx, http.MethodPost, client.addr+"/"+mount, body)
	return err
}

func (client Client) v2Request(ctx context.Context, method, url string, body io.Reader) (io.Reader, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, body)
	if err != nil {
		return nil, err
	}
	req.Header = client.Header

	c := http.Client{Timeout: client.Timeout}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("received non-200 response from caster: %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// NewClientV1
// TODO: Consider making the v1 and v2 API more similar. I like that the v2 client returns a
//  http.Request object, as it allows the caller to modify request headers etc.
func _NewClientV1(host string, path, username, password string) (io.ReadCloser, error) {
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
