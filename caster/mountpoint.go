package caster

import (
	"errors"
	"sync"
	"time"
)

// Subscriber represents an object which can subscribe to a Mountpoint
type Subscriber interface {
	ID() string
	Channel() chan []byte
}

// Mountpoint represents POST requests to a Caster through which a constant
// stream of data is expected. Mountpoints can be subscribed to, Subscribers
// implement a Channel to which POSTed data is written.
type Mountpoint struct {
	sync.RWMutex
	// Connection from which data is received
	Source *Connection
	// A collection of Subscribers to send data to
	Subscribers map[string]Subscriber
}

// ReadSourceData reads data from Source Request Body and writes to Source.Channel
func (mount *Mountpoint) Read() chan []byte {
	data := make(chan []byte)
	go func() {
		buf := make([]byte, 4096)
		nbytes, err := mount.Source.Request.Body.Read(buf)
		if err != nil {
			data <- nil
			return
		}
		data <- buf[:nbytes]
	}()
	return data
}

// Broadcast reads data from Source.Channel and writes to all registered Subscriber Channels
func (mount *Mountpoint) Broadcast(timeout time.Duration) error {
	for {
		select {
		case data := <-mount.Read():
			if data == nil {
				return errors.New("Source closed connection")
			}

			mount.RLock()
			for _, subscriber := range mount.Subscribers {
				select {
				case subscriber.Channel() <- data:
					continue
				default:
					continue // The default case should not occur now that subscriber can be deregistered
				}
			}
			mount.RUnlock()

		case <-time.After(timeout):
			return errors.New("Timeout reading from source")

		case <-mount.Source.Request.Context().Done():
			return errors.New("Source closed connection")
		}
	}
}

// RegisterSubscriber adds a Subscriber to mount.Subscribers
func (mount *Mountpoint) RegisterSubscriber(subscriber Subscriber) {
	mount.Lock()
	defer mount.Unlock()
	mount.Subscribers[subscriber.ID()] = subscriber
}

// DeregisterSubscriber removes a Subscriber from mount.Subscribers
func (mount *Mountpoint) DeregisterSubscriber(subscriber Subscriber) {
	mount.Lock()
	defer mount.Unlock()
	delete(mount.Subscribers, subscriber.ID())
}
