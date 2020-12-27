package ntrip

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// It's expected that SourceService implementations will use these errors to signal specific
// failures.
// TODO: Could use some kind of response code enum type rather than errors?
var (
	ErrorNotAuthorized error = fmt.Errorf("request not authorized")
	ErrorNotFound      error = fmt.Errorf("mount not found")
	ErrorConflict      error = fmt.Errorf("mount in use")
)

// SourceService represents a provider of stream data
type SourceService interface {
	Sourcetable() string // TODO: return ntrip.Sourcetable so we can implement filtering to spec
	// TODO: Specifying username and password may be limiting, could instead take the content of
	//  the auth header
	Publisher(ctx context.Context, mount, username, password string) (io.WriteCloser, error)
	Subscriber(ctx context.Context, mount, username, password string) (chan []byte, error)
}

// Caster wraps http.Server, it provides nothing but timeouts and the Handler
type Caster struct {
	http.Server
}

// NewCaster constructs a Caster, setting up the Handler and timeouts - run using ListenAndServe()
// TODO: Consider not constructing the http.Server, and leaving Caster as a http.Handler
//  Then the caller can create other routes on the server, such as (for example) a /health endpoint,
//  or a /stats endpoint - Though those could instead be run on separate http.Server's
//  Also, middleware can be added to a Caster by doing `c.Handler = someMiddleware(c.Handler)`
func NewCaster(addr string, svc SourceService, logger logrus.FieldLogger) *Caster {
	return &Caster{
		http.Server{
			Addr:        addr,
			Handler:     getHandler(svc, logger),
			IdleTimeout: 10 * time.Second,
			// Read timeout kills publishing connections because they don't necessarily read from
			// the response body
			//ReadTimeout: 10 * time.Second,
			// Write timeout kills subscriber connections because they don't write to the request
			// body
			//WriteTimeout: 10 * time.Second,
		},
	}
}

// Wraps handler in a http.Handler - this is done instead of making handler implement the
// http.Handler interface so that a new handler can be constructed for each request
// TODO: See TODO on handler type about changing the name
func getHandler(svc SourceService, logger logrus.FieldLogger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestVersion := 1
		if strings.ToUpper(r.Header.Get(NTRIPVersionHeaderKey)) == NTRIPVersionHeaderValueV2 {
			requestVersion = 2
		}

		l := logger.WithFields(logrus.Fields{
			"request_id":      uuid.New().String(),
			"request_version": requestVersion,
			"path":            r.URL.Path,
			"method":          r.Method,
			"source_ip":       r.RemoteAddr,
		})

		h := &handler{svc, l}
		h.handleRequest(w, r)
	})
}
