package ntrip_test

import (
	"bufio"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-gnss/ntrip"
	"github.com/go-gnss/ntrip/internal/mock"
	"github.com/sirupsen/logrus"
)

var (
	logger *logrus.Logger = logrus.StandardLogger()
)

func init() {
	logger.Level = logrus.DebugLevel
}

// HijackableResponseRecorder wraps httptest.ResponseRecorder to implement the http.Hijacker
// interface which is needed to test NTRIP v1 requests
// TODO: Move to another package?
// TODO: This doesn't prevent the server from writing to the original response Body, which
//  http.Server would do for a real request - this case is tested by caster_test.go
type HijackableResponseRecorder struct {
	*httptest.ResponseRecorder
}

func (h *HijackableResponseRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	_, conn := net.Pipe()
	rw := bufio.NewReadWriter(bufio.NewReader(h.Body), bufio.NewWriter(h.Body))
	return conn, rw, nil
}

func TestCasterHandlers(t *testing.T) {
	v2Sourcetable := mock.NewMockSourceService().Sourcetable.String()
	v1Sourcetable := fmt.Sprintf("SOURCETABLE 200 OK\r\nConnection: close\r\nContent-Type: text/plain\r\nContent-Length: %d\r\n\r\n%s", len(v2Sourcetable), v2Sourcetable)

	// TODO: Consider making request headers an attribute
	cases := []struct {
		TestName string

		// Inputs
		ChannelData        string // for GET requests, written to mockSourceService channel before connecting
		RequestMethod      string
		RequestURL         string
		RequestBody        string
		Username, Password string
		NTRIPVersion       int

		// Outputs
		ResponseCode int
		ResponseBody string
	}{
		{"v2 Sourcetable Success", "N/A", http.MethodGet, "/", "", "", "", 2, 200, v2Sourcetable},
		{"v2 POST Success", "N/A", http.MethodPost, mock.MountPath, "wow", mock.Username, mock.Password, 2, 200, ""},
		{"v2 GET Success", "v2 GET Success", http.MethodGet, mock.MountPath, "", mock.Username, mock.Password, 2, 200, "v2 GET Success"},
		{"v2 GET Unauthorized", "N/A", http.MethodGet, mock.MountPath, "", "", "", 2, 401, ""},
		{"v2 GET Not Found", "N/A", http.MethodGet, "/NotFound", "", mock.Username, mock.Password, 2, 404, ""},
		{"v2 PUT Not Implemented", "N/A", http.MethodPut, "/any", "", "", "", 2, 501, ""},
		{"v2 POST Unauthorized", "N/A", http.MethodPost, "/any", "", "", "", 2, 401, ""},
		{"v2 POST Not Found", "N/A", http.MethodPost, "/NotFound/longer/path", "", mock.Username, mock.Password, 2, 404, ""},
		{"v1 Sourcetable Success", "N/A", http.MethodGet, "/", "", "", "", 1, 0, v1Sourcetable},
		{"v1 GET Success", "v1 GET Success", http.MethodGet, mock.MountPath, "", mock.Username, mock.Password, 1, 0, "ICY 200 OK\r\nv1 GET Success"},
		// Response recorder headers aren't correctly set when HTTP headers are written to the body, as happens with v1 unauthorized
		{"v1 GET Unauthorized", "N/A", http.MethodGet, mock.MountPath, "", "", "", 1, 0, "HTTP/1.1 401 Unauthorized\r\nConnection: close\r\nWWW-Authenticate: Basic realm=\"/TEST00AUS0\"\r\nContent-Length: 0\r\n\r\n"},
		{"v1 GET Not Found", "N/A", http.MethodGet, "/NotFound", "", mock.Username, mock.Password, 1, 0, "HTTP/1.1 404 Not Found\r\nConnection: close\r\nWWW-Authenticate: Basic realm=\"/NotFound\"\r\nContent-Length: 0\r\n\r\n"},
		// 501 happens before the response is hijacked
		{"v1 POST Not Implemented", "N/A", http.MethodPost, "/any", "", mock.Username, mock.Password, 1, 501, ""},
	}

	for _, tc := range cases {
		req, _ := http.NewRequest(tc.RequestMethod, tc.RequestURL, strings.NewReader(tc.RequestBody))
		if tc.NTRIPVersion == 2 {
			req.Header.Add(ntrip.NTRIPVersionHeaderKey, ntrip.NTRIPVersionHeaderValueV2)
		}
		req.SetBasicAuth(tc.Username, tc.Password)

		rr := &HijackableResponseRecorder{httptest.NewRecorder()}
		// v1 responses don't actually return a code, but the httptest.ResponseRecorder default is
		// 200 which would lead to false positives without setting rr.Code to something else
		rr.Code = 0

		ms := mock.NewMockSourceService()

		// Write tc.ChannelData to ms.DataChannel for GET requests so they receive data in the
		// response Body
		if tc.RequestMethod == http.MethodGet {
			ms.DataChannel = make(chan []byte, 1)
			ms.DataChannel <- []byte(tc.ChannelData)
			// Close channel once client reads from it so we don't have to wait for client timeouts
			// TODO: These will only be closed by successful GET test cases, does this matter?
			go func() {
				// The channel is size 1, so this will block until the GET request client reads
				ms.DataChannel <- []byte{}
				close(ms.DataChannel)
			}()
		}

		caster := ntrip.NewCaster("N/A", ms, logger)
		caster.Handler.ServeHTTP(rr, req)

		if rr.Code != tc.ResponseCode {
			t.Errorf("error in %s: expected response code %d, but received %d", tc.TestName, tc.ResponseCode, rr.Code)
		}

		if rr.Body.String() != tc.ResponseBody {
			t.Errorf("error in %s: expected response body %q, received %q", tc.TestName, tc.ResponseBody, rr.Body.String())
		}
	}
}

// Runs Publishing NTRIP Server client asynchronously and writes to chan when done
func asyncServer(t *testing.T, testName string, caster *ntrip.Caster, data string) chan bool {
	done := make(chan bool, 1)

	r, w := io.Pipe()

	// Write blocks until POST request is connected
	go func() {
		w.Write([]byte(data))
		time.Sleep(20 * time.Millisecond)
		w.Close()
	}()

	// ServeHTTP will block until the PipeWriter is closed
	go func() {
		postReq, _ := http.NewRequest(http.MethodPost, mock.MountPath, r)
		postReq.Header.Add(ntrip.NTRIPVersionHeaderKey, ntrip.NTRIPVersionHeaderValueV2)
		postReq.SetBasicAuth(mock.Username, mock.Password)

		postrr := httptest.NewRecorder()
		postrr.Code = 0 // Default response code is 200 which can lead to false positives
		caster.Handler.ServeHTTP(postrr, postReq)

		if postrr.Code != http.StatusOK {
			t.Errorf("error in %q: expected response code %d for POST request, received %d", testName, http.StatusOK, postrr.Code)
		}
		done <- true
	}()

	return done
}

func TestAsyncPublishSubscribe(t *testing.T) {
	randomLarge := make([]byte, 32768)
	rand.Read(randomLarge)

	cases := []struct {
		TestName string

		NTRIPVersion int
		WriteData    string

		ResponseCode int
		ResponseBody string
	}{
		{"v2 Success", 2, "read by v2 GET request", 200, "read by v2 GET request"},
		{"v2 Success Large Body", 2, string(randomLarge), 200, string(randomLarge)},
		{"v1 Success", 1, "read by v1 GET request", 0, "ICY 200 OK\r\nread by v1 GET request"},
		{"v1 Success Large Body", 1, string(randomLarge), 0, "ICY 200 OK\r\n" + string(randomLarge)},
	}

	for _, tc := range cases {
		ms := mock.NewMockSourceService()
		caster := ntrip.NewCaster("N/A", ms, logger)

		serverDone := asyncServer(t, tc.TestName, caster, tc.WriteData)
		// TODO: Better way to wait for POST request to connect - maybe just implement a retry
		time.Sleep(10 * time.Millisecond)

		getReq, _ := http.NewRequest(http.MethodGet, mock.MountPath, strings.NewReader(""))
		if tc.NTRIPVersion == 2 {
			getReq.Header.Add(ntrip.NTRIPVersionHeaderKey, ntrip.NTRIPVersionHeaderValueV2)
		}
		getReq.SetBasicAuth(mock.Username, mock.Password)

		getrr := &HijackableResponseRecorder{httptest.NewRecorder()}
		getrr.Code = 0
		caster.Handler.ServeHTTP(getrr, getReq)

		if getrr.Code != tc.ResponseCode {
			t.Errorf("error in %q: expected response code %d for GET request, received %d", tc.TestName, tc.ResponseCode, getrr.Code)
		}

		if getrr.Body.String() != tc.ResponseBody {
			t.Errorf("error in %q: response body did not match expected output", tc.TestName)
		}

		select {
		case <-serverDone:
		case <-time.After(1 * time.Second):
			t.Errorf("%s - timeout waiting for server to close", tc.TestName)
		}
	}
}

func TestMountInUse(t *testing.T) {
	ms := mock.NewMockSourceService()
	// MockSourceService returns ntrip.ErrorConflict if DataChannel is not nil
	ms.DataChannel = make(chan []byte, 1)

	req, _ := http.NewRequest(http.MethodPost, mock.MountPath, strings.NewReader("N/A"))
	req.Header.Add(ntrip.NTRIPVersionHeaderKey, ntrip.NTRIPVersionHeaderValueV2)
	req.SetBasicAuth(mock.Username, mock.Password)

	rr := httptest.NewRecorder()
	ntrip.NewCaster("N/A", ms, logger).Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Errorf("expected response status code %d, received %d", http.StatusConflict, rr.Code)
	}
}
