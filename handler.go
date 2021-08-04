package ntrip

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// handler is used by Caster, and is an instance of a request being handled with methods
// for handing v1 and v2 requests
// TODO: Better name - the http.Handler constructs this and uses it's methods for handling
//  requests (so the word "handle" is a bit overloaded)
// TODO: Separate package (in internal)?
type handler struct {
	svc    SourceService
	logger logrus.FieldLogger
}

func (h *handler) handleRequest(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("request received")
	switch strings.ToUpper(r.Header.Get(NTRIPVersionHeaderKey)) {
	case NTRIPVersionHeaderValueV2:
		h.handleRequestV2(w, r)
	default:
		h.handleRequestV1(w, r)
	}
}

// NTRIP v1 is not valid HTTP, so the underlying socket must be hijacked from the HTTP library
// Would need to use net.Listen instead of http.Server to support v1 SOURCE requests
func (h *handler) handleRequestV1(w http.ResponseWriter, r *http.Request) {
	// Can only support NTRIP v1 GET requests with http.Server
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	// Extract underlying net.Conn from ResponseWriter
	hj, ok := w.(http.Hijacker)
	if !ok {
		h.logger.Error("server does not implement hijackable response writers, cannot support NTRIP v1")
		// There is no NTRIP v1 response to signal failure, so this is probably the most useful
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	conn, rw, err := hj.Hijack()
	if err != nil {
		h.logger.Errorf("error hijacking HTTP response writer: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	if r.URL.Path == "/" {
		h.handleGetSourcetableV1(rw, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		h.handleGetMountV1(rw, r)
	default:
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func (h *handler) handleGetSourcetableV1(w *bufio.ReadWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, "SOURCETABLE 200 OK\r\n%s", h.svc.GetSourcetable())
	if err != nil {
		h.logger.Errorf("error writing sourcetable to client: %s", err)
		return
	}

	if err = w.Flush(); err != nil {
		h.logger.Warnf("error flushing data to client: %s", err)
		return
	}
}

func (h *handler) handleGetMountV1(w *bufio.ReadWriter, r *http.Request) {
	sub, err := h.svc.Subscriber(r.Context(), r)
	if err != nil {
		// NTRIP v1 says to return 401 for unauthorized, but sourcetable for any other error
		if err == ErrorNotAuthorized {
			// TODO: Check errors in writing and flushing
			writeUnauthorizedV1(w, r)
			w.Flush()
			return
		}

		h.handleGetSourcetableV1(w, r)
		return
	}

	// TODO: Check error in Write and Flush
	w.Write([]byte("ICY 200 OK\r\n")) // NTRIP v1 is ICECAST, this is the equivalent of HTTP 200 OK
	w.Flush()

	// TODO: Does this flush data to client?
	_, err = io.Copy(w, sub)
}

func (h *handler) handleRequestV2(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Connection", "close")
	if r.URL.Path == "/" {
		h.handleGetSourcetableV2(w, r)
		return
	}

	var err error

	switch r.Method {
	case http.MethodGet:
		err = h.handleGetMountV2(w, r)
	case http.MethodPost:
		err = h.handlePostMountV2(w, r)
	default:
		h.logger.Debugf("ignoring unsupported %s request", r.Method)
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	// TODO: Check errors in writes
	switch err {
	case nil:
	case ErrorNotAuthorized:
		w.Header().Add("WWW-Authenticate", fmt.Sprintf("Basic realm=%q", r.URL.Path))
		w.WriteHeader(http.StatusUnauthorized)
	case ErrorNotFound:
		w.WriteHeader(http.StatusNotFound)
	case ErrorConflict:
		w.WriteHeader(http.StatusConflict)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (h *handler) handleGetSourcetableV2(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement sourcetable filtering support
	_, err := fmt.Fprintf(w, h.svc.GetSourcetable().String())
	if err != nil {
		h.logger.Warnf("error writing sourcetable to client: %s", err)
		return
	}

}

func (h *handler) handlePostMountV2(w http.ResponseWriter, r *http.Request) error {
	pub, err := h.svc.Publisher(r.Context(), r)
	if err != nil {
		return err
	}
	defer pub.Close()

	// Write response headers in order for client to begin sending data
	// TODO: Check if type cast is successful
	w.(http.Flusher).Flush()

	_, err = io.Copy(pub, r.Body)
	if err == nil {
		// TODO: Also check for "unexpected EOF"
		err = fmt.Errorf("request body closed")
	}

	// Duplicating connection closed message here to avoid superfluous calls to WriteHeader
	return nil
}

func (h *handler) handleGetMountV2(w http.ResponseWriter, r *http.Request) error {
	sub, err := h.svc.Subscriber(r.Context(), r)
	if err != nil {
		return err
	}

	w.Header().Add("Content-Type", "gnss/data")
	// Flush response headers before sending data to client, default status code is 200
	// TODO: Don't necessarily need to do this, since the first data written to client will flush
	w.(http.Flusher).Flush()

	// TODO: This won't flush periodically?
	_, err = io.Copy(w, sub)
	// Duplicating connection closed message here to avoid superfluous calls to WriteHeader
	return nil
}

// Spec says that WWW-Authenticate header is required for casters
func writeUnauthorizedV1(w io.Writer, r *http.Request) error {
	// TODO: Not sure about setting the HTTP version
	// TODO: Check for errors writing and flushing
	resp := http.Response{
		StatusCode: http.StatusUnauthorized,
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header: map[string][]string{
			"WWW-Authenticate": {fmt.Sprintf("Basic realm=%q", r.URL.Path)},
		},
		Close: true,
	}
	return resp.Write(w)
}
