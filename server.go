package ntrip

import (
	"io"
	"net/http"
)

// NewServerRequestV2 constructs an http.Request which can be used as an NTRIP v2 Server
// Effectively a chunked encoding POST request which is not expected to close
func NewServerRequestV2(url string, r io.ReadCloser) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodPost, url, r)
	req.TransferEncoding = []string{"chunked"}
	req.Header.Set("User-Agent", "NTRIP go-gnss/ntrip/server")
	req.Header.Set(NTRIPVersionHeaderKey, NTRIPVersionHeaderValueV2)
	return req, err
}
