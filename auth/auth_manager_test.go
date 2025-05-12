package auth

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthManagerExtended(t *testing.T) {
	// Create an auth manager
	manager := NewAuthManager()

	// Create authenticators
	basicAuth := NewBasicAuth()
	basicAuth.AddCredential("username", "password")

	digestAuth := NewDigestAuth()
	digestAuth.AddCredential("username", "password")

	noAuth := NewNoAuth()

	// Test default authenticator
	assert.Equal(t, None, manager.GetAuthenticator("any").Method())

	// Set mount authenticators
	manager.SetMountAuthenticator("basic", basicAuth)
	manager.SetMountAuthenticator("digest", digestAuth)
	manager.SetMountAuthenticator("none", noAuth)

	// Set default authenticator
	manager.SetDefaultAuthenticator(basicAuth)

	// Test getting authenticators
	assert.Equal(t, basicAuth, manager.GetAuthenticator("basic"))
	assert.Equal(t, digestAuth, manager.GetAuthenticator("digest"))
	assert.Equal(t, noAuth, manager.GetAuthenticator("none"))
	assert.Equal(t, basicAuth, manager.GetAuthenticator("unknown"))

	// Test authentication with basic auth
	req1, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)
	req1.SetBasicAuth("username", "password")

	authenticated, err := manager.Authenticate(req1, "basic")
	require.NoError(t, err)
	assert.True(t, authenticated)

	// Test authentication with no auth
	req2, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	authenticated, err = manager.Authenticate(req2, "none")
	require.NoError(t, err)
	assert.True(t, authenticated)

	// Test authentication with wrong password
	req3, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)
	req3.SetBasicAuth("username", "wrong")

	authenticated, err = manager.Authenticate(req3, "basic")
	require.NoError(t, err)
	assert.False(t, authenticated)

	// Test challenge
	challenge := manager.Challenge("basic")
	assert.Equal(t, `Basic realm="basic"`, challenge)

	challenge = manager.Challenge("digest")
	assert.Contains(t, challenge, `Digest realm="digest"`)

	challenge = manager.Challenge("none")
	assert.Equal(t, "", challenge)

	challenge = manager.Challenge("unknown")
	assert.Equal(t, `Basic realm="unknown"`, challenge)
}

func TestAuthManagerConcurrency(t *testing.T) {
	// Create an auth manager
	manager := NewAuthManager()

	// Create authenticators
	basicAuth := NewBasicAuth()
	basicAuth.AddCredential("username", "password")

	digestAuth := NewDigestAuth()
	digestAuth.AddCredential("username", "password")

	// Set mount authenticators and default authenticator concurrently
	go func() {
		manager.SetMountAuthenticator("basic", basicAuth)
	}()

	go func() {
		manager.SetMountAuthenticator("digest", digestAuth)
	}()

	go func() {
		manager.SetDefaultAuthenticator(basicAuth)
	}()

	// Get authenticators concurrently
	go func() {
		auth := manager.GetAuthenticator("basic")
		if auth != nil {
			_ = auth.Method()
		}
	}()

	go func() {
		auth := manager.GetAuthenticator("digest")
		if auth != nil {
			_ = auth.Method()
		}
	}()

	go func() {
		auth := manager.GetAuthenticator("unknown")
		if auth != nil {
			_ = auth.Method()
		}
	}()

	// No assertions here, we're just testing that there are no race conditions
	// This test will fail if the race detector is enabled and there are race conditions
}
