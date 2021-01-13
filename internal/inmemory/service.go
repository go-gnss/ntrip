package inmemory

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/go-gnss/ntrip"
)

// TODO: Better names
type mount struct {
	ctx    context.Context
	cancel context.CancelFunc
	// Subscribers register themselves by adding their writer to the slice
	clients []io.WriteCloser
}

type Mount struct {
	sync.RWMutex
	ntrip.MountEntry
	mount *mount
}

type Sources struct {
	sync.RWMutex
	Casters  []ntrip.CasterEntry
	Networks []ntrip.NetworkEntry
	Mounts   map[string]*Mount
}

// SourceService is an in-memory implementation of ntrip.SourceService
type SourceService struct {
	sourcetable *Sources
	auth        Authoriser
}

func NewSourceService(st ntrip.Sourcetable, auth Authoriser) *SourceService {
	ss := &SourceService{
		sourcetable: &Sources{Mounts: map[string]*Mount{}},
		auth:        auth,
	}
	ss.UpdateSourcetable(st)
	return ss
}

func (ss *SourceService) UpdateSourcetable(st ntrip.Sourcetable) {
	ss.sourcetable.Lock()
	defer ss.sourcetable.Unlock()

	ss.sourcetable.Casters = st.Casters
	ss.sourcetable.Networks = st.Networks

	// Check for new mounts from config
	for _, mount := range st.Mounts {
		if m, ok := ss.sourcetable.Mounts[mount.Name]; !ok {
			ss.sourcetable.Mounts[mount.Name] = &Mount{MountEntry: mount}
		} else {
			m.MountEntry = mount
		}
	}

	// Remove mounts which have been deleted from the config file
	// TODO: Efficient?
OUTER:
	for name, mount := range ss.sourcetable.Mounts {
		for _, mountEntry := range st.Mounts {
			if mount.MountEntry == mountEntry {
				continue OUTER
			}
		}
		// Cancel mount and remove from list of Mounts
		// TODO: Log that mount was removed
		mount.Lock()
		if mount.mount != nil {
			mount.mount.cancel()
		}
		mount.Unlock()
		delete(ss.sourcetable.Mounts, name)
	}
}

func (ss *SourceService) GetSourcetable() ntrip.Sourcetable {
	ss.sourcetable.RLock()
	defer ss.sourcetable.RUnlock()
	st := ntrip.Sourcetable{
		Casters:  ss.sourcetable.Casters,
		Networks: ss.sourcetable.Networks,
		Mounts:   []ntrip.MountEntry{},
	}
	for _, mount := range ss.sourcetable.Mounts {
		if mount.mount != nil {
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

	ss.sourcetable.Lock()
	defer ss.sourcetable.Unlock()

	m, ok := ss.sourcetable.Mounts[mountName]
	if ok && m.mount != nil {
		return nil, ntrip.ErrorConflict
	}

	if !ok {
		// TODO: Should this be NotAuthorized or NotFound?
		return nil, ntrip.ErrorNotAuthorized
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.Lock()
	m.mount = &mount{ctx: ctx, cancel: cancel, clients: []io.WriteCloser{}}
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
		for _, client := range m.mount.clients {
			client.Close()
		}

		// Make Mount "offline"
		m.mount = nil

		r.Close()
	}()

	for {
		// Read
		var data []byte
		select {
		case <-m.mount.ctx.Done():
			return
		case result := <-readChannel(r):
			if result.err != nil {
				return
			}
			data = result.data
		}

		// Write
		m.Lock()
		for i, w := range m.mount.clients {
			if _, err := w.Write(data); err != nil {
				// Re-slice to remove closed Writer
				m.mount.clients = append(m.mount.clients[:i], m.mount.clients[i+1:]...)
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

	ss.sourcetable.RLock()
	m, ok := ss.sourcetable.Mounts[mountName]
	ss.sourcetable.RUnlock()
	if !ok || m.mount == nil {
		return nil, ntrip.ErrorNotFound
	}

	r, w := io.Pipe()
	m.Lock()
	m.mount.clients = append(m.mount.clients, w)
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
