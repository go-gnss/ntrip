package inmemory

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/go-gnss/ntrip"
)

// SourceService is a simple in-memory implementation of ntrip.SourceService
type SourceService struct {
	sync.Mutex
	Sourcetable ntrip.Sourcetable
	mounts      map[string][]io.Writer
	auth        Authoriser
}

func NewSourceService(auth Authoriser) *SourceService {
	return &SourceService{
		mounts: map[string][]io.Writer{},
		auth:   auth,
	}
}

func (ss *SourceService) GetSourcetable() ntrip.Sourcetable {
	// TODO: Only include online Mounts in output
	return ss.Sourcetable
}

func (ss *SourceService) Publisher(ctx context.Context, mount, username, password string) (io.WriteCloser, error) {
	if auth, err := ss.auth.Authorise(PublishAction, mount, username, password); err != nil {
		return nil, fmt.Errorf("authorization failed: %w", err)
	} else if !auth {
		return nil, ntrip.ErrorNotAuthorized
	}

	ss.Lock()
	defer ss.Unlock()

	_, ok := ss.mounts[mount]
	if ok {
		return nil, ntrip.ErrorConflict
	}

	// Subscribers register themselves by adding their writer to this slice
	ss.mounts[mount] = []io.Writer{}

	r, w := io.Pipe()

	// Create a buffer pool for efficient memory reuse
	bufPool := sync.Pool{
		New: func() any { return make([]byte, 4096) },
	}

	// Read from r, and write to ss.mounts[mount]
	go func() {
		defer func() {
			// Clean up when goroutine exits
			ss.Lock()
			delete(ss.mounts, mount)
			ss.Unlock()
		}()

		for {
			select {
			case <-ctx.Done():
				// Context cancelled, exit goroutine
				return
			default:
				// Get a buffer from the pool
				buf := bufPool.Get().([]byte)
				br, err := r.Read(buf)
				if err != nil {
					// Return buffer to pool and exit if reader is closed
					bufPool.Put(buf)
					return
				}

				// Write to all subscribers
				ss.Lock()
				var activeWriters []io.Writer
				for _, writer := range ss.mounts[mount] {
					if _, err := writer.Write(buf[:br]); err == nil {
						// Keep only active writers
						activeWriters = append(activeWriters, writer)
					}
				}
				// Replace with only active writers
				ss.mounts[mount] = activeWriters
				ss.Unlock()

				// Return buffer to pool
				bufPool.Put(buf)
			}
		}
	}()

	return w, nil
}

func (ss *SourceService) Subscriber(ctx context.Context, mount, username, password string) (chan []byte, error) {
	if auth, err := ss.auth.Authorise(SubscribeAction, mount, username, password); err != nil {
		return nil, fmt.Errorf("authorization failed: %w", err)
	} else if !auth {
		return nil, ntrip.ErrorNotAuthorized
	}

	ss.Lock()
	defer ss.Unlock()

	mw, ok := ss.mounts[mount]
	if !ok {
		return nil, ntrip.ErrorNotFound
	}

	r, w := io.Pipe()
	ss.mounts[mount] = append(mw, w)

	// Create a buffer pool for efficient memory reuse
	bufPool := sync.Pool{
		New: func() any { return make([]byte, 4096) },
	}

	// Create a buffered channel for data
	data := make(chan []byte, 8)

	// Cleanup when client closes connection
	go func() {
		<-ctx.Done()
		w.Close()
	}()

	// Read from r and write to data channel
	go func() {
		defer close(data) // Close channel when done

		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Get buffer from pool
				buf := bufPool.Get().([]byte)
				br, err := r.Read(buf)
				if err != nil {
					// Return buffer to pool and exit if reader is closed
					bufPool.Put(buf)
					return
				}

				// Create a copy of the data to send through the channel
				// This is necessary because we're returning the buffer to the pool
				dataCopy := make([]byte, br)
				copy(dataCopy, buf[:br])

				// Return buffer to pool
				bufPool.Put(buf)

				// Send data to channel, with context cancellation support
				select {
				case data <- dataCopy:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return data, nil
}
