package ntrip

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// SourceService represents a provider of stream data
type SourceService interface {
	GetSourcetable() Sourcetable
	// TODO: Specifying username and password may be limiting, could instead take the content of
	//  the auth header
	// TODO: A SourceService implementation can't support nearest base functionality because it
	//  wouldn't have access to NMEA headers - in general, it may be arbitrarily limiting to not
	//  pass the http.Request object (leaving it up to the implementation to parse headers etc.)
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
		if strings.ToUpper(r.Header.Get(NTRIPVersionHeaderKey)) == strings.ToUpper(NTRIPVersionHeaderValueV2) {
			requestVersion = 2
		}

		requestID := uuid.New().String()
		ctx := context.WithValue(r.Context(), RequestIDContextKey, requestID)

		username, _, _ := r.BasicAuth()

		l := logger.WithFields(logrus.Fields{
			"request_id":      requestID,
			"request_version": requestVersion,
			"path":            r.URL.Path,
			"method":          r.Method,
			"source_ip":       r.RemoteAddr,
			"username":        username,
			"user_agent":      r.UserAgent(),
		})

		h := &handler{svc, l}
		h.handleRequest(w, r.WithContext(ctx))
	})
}
