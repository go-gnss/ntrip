package caster

import (
	"github.com/google/uuid"
	"net/http"
)

// Connection represents a client HTTP(S) request, implements Subscriber interface
type Connection struct {
	id      string
	channel chan []byte
	Writer  http.ResponseWriter
	Request *http.Request
}

// NewConnection constructs a Connection object from a http Request and ResponseWriter
func NewConnection(w http.ResponseWriter, r *http.Request) (conn *Connection) {
	requestID := uuid.New().String()
	return &Connection{requestID, make(chan []byte, 10), w, r}
}

// ID returns the unexported id field of the Connection object which is generated on construction
func (conn *Connection) ID() string {
	return conn.id
}

// Channel returns the unexported channel field of the Connection object which is generated on construction.
// Mountpoints use this to collect incomming data from a Request. Subscribers use this to receive data from
// the Mountpoint to which they subscribe.
func (conn *Connection) Channel() chan []byte {
	return conn.channel
}
