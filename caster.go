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
	Publisher(ctx context.Context, request *http.Request) (io.WriteCloser, error)
	Subscriber(ctx context.Context, request *http.Request) (io.ReadCloser, error)
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

		requestID := uuid.New().String()
		ctx := context.WithValue(r.Context(), RequestIDContextKey, requestID)

		l := logger.WithField("request_id", requestID).
			WithField("request_version", requestVersion).
			WithField("path", r.URL.Path).
			WithField("method", r.Method).
			WithField("source_ip", r.RemoteAddr).
			WithField("user_agent", r.Header.Get("User-Agent"))

		h := &handler{svc, l}
		h.handleRequest(w, r.WithContext(ctx))
	})
}
