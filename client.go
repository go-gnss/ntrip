package ntrip

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"
)

type Client interface {
	GetSourcetable() (Sourcetable, error)
	GetMount(mount string) (io.ReadCloser, error)
	// TODO: PostMount as part of ntrip.Client doesn't really fit the NTRIP naming, but would likely be a better interface for this library
}

type ClientV2 struct {
	http.Client
	url      string
	Username string
	Password string
}

func NewClientV2(url string) *ClientV2 {
	return &ClientV2{
		Client: http.Client{
			Timeout: 5 * time.Second,
		},
		url: url,
	}
}

func (c *ClientV2) connect(path string) (*http.Response, error) {
	req, _ := http.NewRequest(http.MethodGet, c.url+path, nil)
	req.Header.Set(NTRIPVersionHeaderKey, NTRIPVersionHeaderValueV2)
	req.Header.Set("User-Agent", "NTRIP go-gnss/ntrip/client")
	req.SetBasicAuth(c.Username, c.Password)

	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close() // TODO: Is this needed?
		return nil, fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
	}

	return resp, nil
}

func (c *ClientV2) GetSourcetable() (Sourcetable, error) {
	// TODO:
	resp, err := c.connect("/")
	if err != nil {
		return Sourcetable{}, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Sourcetable{}, fmt.Errorf("error reading caster response: %s", err)
	}

	st, errs := ParseSourcetable(string(body))
	if len(errs) > 0 {
		// TODO: How best to return parsing errors?
		err = fmt.Errorf("errors parsing sourcetable")
	}

	return st, nil
}

func (c *ClientV2) GetMount(mount string) (io.ReadCloser, error) {
	resp, err := c.connect("/" + mount)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

// TODO: Implement ClientV1
//func (c *ClientV1) ConnectV1(mount string) (r io.ReadCloser, err error) {
//	addr := fmt.Sprintf("%s:%d", c.host, c.port)
//
//	var conn net.Conn
//	if c.tls {
//		conn, err = tls.Dial("tcp", addr, &tls.Config{})
//	} else {
//		conn, err = net.Dial("tcp", addr)
//	}
//	if err != nil {
//		return nil, err
//	}
//
//	// V1 requests are valid HTTP, but the response may not be
//	url := fmt.Sprintf("tcp://%s:%d/%s", c.host, c.port, mount)
//	req, err := http.NewRequest(http.MethodGet, url, strings.NewReader(""))
//	if err != nil {
//		return nil, err
//	}
//	req.SetBasicAuth(c.Username, c.Password)
//	req.Header.Add("User-Agent", "NTRIP go-gnss/ntrip/client")
//
//	err = req.Write(conn)
//	if err != nil {
//		conn.Close()
//		return nil, fmt.Errorf("error making request: %s", err)
//	}
//
//	// TODO: Maybe read until \r\n instead of this - would make the error clearer
//	expected := "ICY 200 OK\r\n"
//	resp := make([]byte, len(expected))
//	_, err = conn.Read(resp)
//	if err != nil {
//		conn.Close()
//		return nil, fmt.Errorf("error reading response: %s", err)
//	}
//
//	if string(resp) != expected {
//		return nil, fmt.Errorf("received non-200 response: %q", string(resp))
//	}
//
//	return conn, nil
//}
