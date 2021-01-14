package inmemory

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/go-gnss/ntrip"
)

type OnlineMount struct {
	ctx    context.Context
	cancel context.CancelFunc
	// Subscribers register themselves by adding their writer to the slice
	clients []io.WriteCloser
}

// Mount exists regardless of whether or not it is currently being streamed to.
// OnlineMount will be nil if the mount is offline.
type Mount struct {
	sync.RWMutex
	ntrip.MountEntry
	OnlineMount *OnlineMount
}

// Sources contains Sourcetable information as well as a Mount collection
type Sources struct {
	sync.RWMutex
	Casters  []ntrip.CasterEntry
	Networks []ntrip.NetworkEntry
	Mounts   map[string]*Mount
}

// SourceService is an in-memory implementation of ntrip.SourceService
type SourceService struct {
	sources *Sources
	auth    Authoriser
}

func NewSourceService(st ntrip.Sourcetable, auth Authoriser) *SourceService {
	ss := &SourceService{
		sources: &Sources{Mounts: map[string]*Mount{}},
		auth:    auth,
	}
	ss.UpdateSourcetable(st)
	return ss
}

func (ss *SourceService) UpdateSourcetable(st ntrip.Sourcetable) {
	ss.sources.Lock()
	defer ss.sources.Unlock()

	ss.sources.Casters = st.Casters
	ss.sources.Networks = st.Networks

	// Check for new mounts from config
	for _, mount := range st.Mounts {
		if m, ok := ss.sources.Mounts[mount.Name]; !ok {
			ss.sources.Mounts[mount.Name] = &Mount{MountEntry: mount}
		} else {
			m.MountEntry = mount
		}
	}

	// Remove mounts which have been deleted from the config file
	// TODO: Efficient?
OUTER:
	for name, mount := range ss.sources.Mounts {
		for _, mountEntry := range st.Mounts {
			if mount.MountEntry == mountEntry {
				continue OUTER
			}
		}
		// Cancel mount and remove from list of Mounts
		// TODO: Log that mount was removed
		mount.Lock()
		if mount.OnlineMount != nil {
			mount.OnlineMount.cancel()
		}
		mount.Unlock()
		delete(ss.sources.Mounts, name)
	}
}

func (ss *SourceService) GetSourcetable() ntrip.Sourcetable {
	ss.sources.RLock()
	defer ss.sources.RUnlock()
	st := ntrip.Sourcetable{
		Casters:  ss.sources.Casters,
		Networks: ss.sources.Networks,
		Mounts:   []ntrip.MountEntry{},
	}

	// Include only online mounts in returned Sourcetable
	for _, mount := range ss.sources.Mounts {
		if mount.OnlineMount != nil {
			st.Mounts = append(st.Mounts, mount.MountEntry)
		}
	}
	return st
}

func (ss *SourceService) Publisher(ctx context.Context, mountName, username, password string) (io.WriteCloser, error) {
	if auth, err := ss.auth.Authorise(PublishAction, mountName, username, password); err != nil {
		return nil, fmt.Errorf("error in authorisation: %s", err)
	} else if !auth {
		return nil, ntrip.ErrorNotAuthorized
	}

	ss.sources.Lock()
	defer ss.sources.Unlock()

	m, ok := ss.sources.Mounts[mountName]
	if ok && m.OnlineMount != nil {
		return nil, ntrip.ErrorConflict
	}

	if !ok {
		// TODO: Should this be NotAuthorized or NotFound?
		return nil, ntrip.ErrorNotAuthorized
	}

	ctx, cancel := context.WithCancel(ctx)
	m.Lock()
	m.OnlineMount = &OnlineMount{ctx: ctx, cancel: cancel, clients: []io.WriteCloser{}}
	m.Unlock()

	r, w := io.Pipe()
	go serve(r, m)

	return w, nil
}

// Read from r and write to m.mount.clients
func serve(r io.ReadCloser, m *Mount) {
	defer func() { // Cleanup
		// Close client writers
		m.Lock()
		defer m.Unlock()
		for _, client := range m.OnlineMount.clients {
			client.Close()
		}

		// Make Mount "offline"
		m.OnlineMount = nil

		r.Close()
	}()

	for {
		// Read
		var data []byte
		select {
		case <-m.OnlineMount.ctx.Done():
			return
		case result := <-readChannel(r):
			if result.err != nil {
				return
			}
			data = result.data
		}

		// Write
		m.Lock()
		for i, w := range m.OnlineMount.clients {
			if _, err := w.Write(data); err != nil {
				// Re-slice to remove closed Writer
				m.OnlineMount.clients = append(m.OnlineMount.clients[:i], m.OnlineMount.clients[i+1:]...)
			}
		}
		m.Unlock()
	}
}

type readResult struct {
	data []byte
	err  error
}

func readChannel(r io.Reader) chan readResult {
	result := make(chan readResult, 1)
	go func() {
		buf := make([]byte, 1024)
		br, err := r.Read(buf)
		result <- readResult{buf[:br], err}
	}()
	return result
}

func (ss *SourceService) Subscriber(ctx context.Context, mountName, username, password string) (chan []byte, error) {
	if auth, err := ss.auth.Authorise(SubscribeAction, mountName, username, password); err != nil {
		return nil, fmt.Errorf("error in authorisation: %s", err)
	} else if !auth {
		return nil, ntrip.ErrorNotAuthorized
	}

	ss.sources.RLock()
	m, ok := ss.sources.Mounts[mountName]
	ss.sources.RUnlock()
	if !ok || m.OnlineMount == nil {
		return nil, ntrip.ErrorNotFound
	}

	r, w := io.Pipe()
	m.Lock()
	m.OnlineMount.clients = append(m.OnlineMount.clients, w)
	m.Unlock()

	// Cleanup when client closes connection
	go func() {
		<-ctx.Done()
		// TODO: Check for mount.ctx.Done()?
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
				close(data)
				return
			}
			data <- buf[:br]
		}
	}()

	return data, nil
}
