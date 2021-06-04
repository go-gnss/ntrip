package ntrip_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-gnss/ntrip"
	"github.com/go-gnss/ntrip/internal/mock"
	"github.com/sirupsen/logrus"
)

// TODO: Test failure cases with httptest.Server

// Test running Caster with mock service using httptest.Server, which is close to actually calling
//  caster.ListenAndServe(), write data with v2 server and read with v2 and v1 clients
func TestCasterServerClient(t *testing.T) {
	caster := ntrip.NewCaster("N/A", mock.NewMockSourceService(), logrus.StandardLogger())
	ts := httptest.NewServer(caster.Handler)
	defer ts.Close()

	r, w := io.Pipe()

	// Server
	{
		sreq, _ := ntrip.NewServerRequest(ts.URL+mock.MountPath, r)
		sreq.SetBasicAuth(mock.Username, mock.Password)
		sresp, err := http.DefaultClient.Do(sreq)
		if err != nil {
			t.Fatalf("server - error connecting to caster: %s", err)
		}
		defer sreq.Body.Close()

		if sresp.StatusCode != http.StatusOK {
			t.Fatalf("server - expected response code %d, received %d", http.StatusOK, sresp.StatusCode)
		}
	}

	testV2Client(t, ts.URL+mock.MountPath, w)

	// POST request's context may not get closed in the server before the next Write occurs,
	// resulting in the mock writing to the first connected client's Body
	// Nothing like a 10ms timeout to fix a bit of non-deterministic behaviour
	// TODO: Could fix this by rewriting the mock service, or using the inmemory SourceService
	time.Sleep(10 * time.Millisecond)

	testV1Client(t, ts.URL[7:], mock.MountPath, w)
}

func testV1Client(t *testing.T, host, path string, serverWriter io.Writer) {
	req, err := ntrip.NewClientV1(host, path, mock.Username, mock.Password)
	if err != nil {
		t.Fatalf("v1 client - error connecting to caster: %s", err)
	}
	defer req.Close()

	testString := "some test data"

	_, err = serverWriter.Write([]byte(testString))
	if err != nil {
		t.Fatalf("server - error during write for v1: %s", err)
	}

	responseHeaders := "ICY 200 OK\r\n"
	buf := make([]byte, len(responseHeaders))
	br, err := req.Read(buf)
	if err != nil {
		t.Fatalf("v1 client - error during read headers: %s", err)
	}

	if string(buf[:br]) != responseHeaders {
		t.Fatalf("v1 client - expected response headers %q, received %q", responseHeaders, string(buf[:br]))
	}

	buf = make([]byte, len(testString))
	br, err = req.Read(buf)
	if err != nil {
		t.Fatalf("v1 client - error during read: %s", err)
	}

	if string(buf[:br]) != testString {
		t.Fatalf("v1 client - expected response body %q, received %q", testString, string(buf[:br]))
	}
}

func testV2Client(t *testing.T, url string, serverWriter io.Writer) {
	req, _ := ntrip.NewClientRequest(url)
	req.SetBasicAuth(mock.Username, mock.Password)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("client - error connecting to caster: %s", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("v2 client - expected response code %d, received %d", http.StatusOK, resp.StatusCode)
	}

	testString := "some test data"

	_, err = serverWriter.Write([]byte(testString))
	if err != nil {
		t.Fatalf("server - error during write: %s", err)
	}

	buf := make([]byte, len(testString))
	_, err = resp.Body.Read(buf)
	if err != nil {
		t.Fatalf("v2 client - error during read: %s", err)
	}

	if string(buf) != testString {
		t.Fatalf("v2 client - expected response body %q, received %q", testString, string(buf))
	}

	resp.Body.Close()
}
