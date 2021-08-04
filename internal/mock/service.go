package mock

import (
	"context"
	"io"
	"net/http"

	"github.com/go-gnss/ntrip"
)

const (
	MountName string = "TEST00AUS0"
	MountPath string = "/" + MountName
	Username  string = "username"
	Password  string = "password"
)

var (
	// Ensure mock meets interface requirements
	_ ntrip.SourceService = &MockSourceService{}
)

// MockSourceService implements ntrip.SourceService, copying data from a single connected server
// (mount name TEST00AUS0) into a channel
type MockSourceService struct {
	Reader      io.ReadCloser
	Sourcetable ntrip.Sourcetable
}

func NewMockSourceService() *MockSourceService {
	return &MockSourceService{
		Sourcetable: ntrip.Sourcetable{
			Casters: []ntrip.CasterEntry{
				{
					Host:       "localhost",
					Port:       2101,
					Identifier: "local",
					Country:    "AUS",
					Latitude:   -1.0,
					Longitude:  1.0,
				},
			},
		},
	}
}

func (m *MockSourceService) GetSourcetable() ntrip.Sourcetable {
	return m.Sourcetable
}

func (m *MockSourceService) Subscriber(ctx context.Context, r *http.Request) (io.ReadCloser, error) {
	username, password, _ := r.BasicAuth()
	if username != Username || password != Password {
		return nil, ntrip.ErrorNotAuthorized
	}

	if r.URL.Path != MountPath {
		return nil, ntrip.ErrorNotFound
	}

	if m.Reader == nil {
		return nil, ntrip.ErrorNotFound
	}

	return m.Reader, nil
}

func (m *MockSourceService) Publisher(ctx context.Context, r *http.Request) (io.WriteCloser, error) {
	username, password, _ := r.BasicAuth()
	if username != Username || password != Password {
		return nil, ntrip.ErrorNotAuthorized
	}

	if r.URL.Path != MountPath {
		return nil, ntrip.ErrorNotFound
	}

	if m.Reader != nil {
		return nil, ntrip.ErrorConflict
	}

	reader, writer := io.Pipe()
	m.Reader = reader

	go func() {
		<-ctx.Done()
		m.Reader = nil
		reader.Close()
	}()

	return writer, nil
}
