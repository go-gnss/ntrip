package ntrip

import (
	"fmt"
)

const (
	NTRIPVersionHeaderKey     string = "Ntrip-Version"
	NTRIPVersionHeaderValueV2 string = "Ntrip/2.0"
)

// It's expected that SourceService implementations will use these errors to signal specific
// failures.
// TODO: Could use some kind of response code enum type rather than errors?
var (
	ErrorNotAuthorized error = fmt.Errorf("request not authorized")
	ErrorNotFound      error = fmt.Errorf("mount not found")
	ErrorConflict      error = fmt.Errorf("mount in use")
	ErrorBadRequest    error = fmt.Errorf("bad request")

	// TODO: Added this so a SourceService implementation can extract the Request ID, not sure that
	//  smuggling it in the context is the best approach
	RequestIDContextKey contextKey = contextKey("RequestID")
)

type contextKey string

func (c contextKey) String() string {
	return string(c)
}
