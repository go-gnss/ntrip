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
		return nil, fmt.Errorf("error in authorisation: %s", err)
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

	// TODO: Read from r, and write to ss.mounts[mount]
	go func() {
		for {
			// Read
			buf := make([]byte, 1024)
			br, err := r.Read(buf)
			if err != nil {
				// Remove self from mounts map if Reader closes
				delete(ss.mounts, mount)
				return
			}
			// Write
			ss.Lock()
			for i, w := range ss.mounts[mount] {
				if _, err := w.Write(buf[:br]); err != nil {
					// Re-slice to remove closed Writer
					ss.mounts[mount] = append(ss.mounts[mount][:i], ss.mounts[mount][i+1:]...)
				}
			}
			ss.Unlock()
		}
	}()

	return w, nil
}

func (ss *SourceService) Subscriber(ctx context.Context, mount, username, password string) (chan []byte, error) {
	if auth, err := ss.auth.Authorise(SubscribeAction, mount, username, password); err != nil {
		return nil, fmt.Errorf("error in authorisation: %s", err)
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

	// Cleanup when client closes connection
	go func() {
		<-ctx.Done()
		w.Close()
	}()

	data := make(chan []byte, 1)
	// Read from r and write to data channel
	go func() {
		for {
			buf := make([]byte, 1024)
			br, err := r.Read(buf)
			if err != nil {
				// Server closed connection
				return
			}
			data <- buf[:br]
		}
	}()

	return data, nil
}
