package auth

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBasicAuth(t *testing.T) {
	// Create a basic authenticator
	auth := NewBasicAuth()

	// Add credentials
	auth.AddCredential("username", "password")

	// Create a request with basic auth
	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)
	req.SetBasicAuth("username", "password")

	// Test authentication
	authenticated, err := auth.Authenticate(req, "mount")
	require.NoError(t, err)
	assert.True(t, authenticated)

	// Test with wrong password
	req.SetBasicAuth("username", "wrong")
	authenticated, err = auth.Authenticate(req, "mount")
	require.NoError(t, err)
	assert.False(t, authenticated)

	// Test with wrong username
	req.SetBasicAuth("wrong", "password")
	authenticated, err = auth.Authenticate(req, "mount")
	require.NoError(t, err)
	assert.False(t, authenticated)

	// Test without auth
	req, err = http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)
	authenticated, err = auth.Authenticate(req, "mount")
	require.NoError(t, err)
	assert.False(t, authenticated)

	// Test challenge
	challenge := auth.Challenge("mount")
	assert.Equal(t, `Basic realm="mount"`, challenge)

	// Test method
	assert.Equal(t, Basic, auth.Method())
}

func TestDigestAuth(t *testing.T) {
	// Create a digest authenticator
	auth := NewDigestAuth()

	// Add credentials
	auth.AddCredential("username", "password")

	// Get a challenge
	challenge := auth.Challenge("mount")

	// Extract nonce from challenge
	parts := strings.Split(challenge, ", ")
	var nonce string
	for _, part := range parts {
		if strings.HasPrefix(part, "nonce=") {
			nonce = strings.Trim(strings.TrimPrefix(part, "nonce="), "\"")
			break
		}
	}
	require.NotEmpty(t, nonce)

	// Create a request with digest auth
	req, err := http.NewRequest("GET", "http://example.com/mount", nil)
	require.NoError(t, err)

	// Calculate digest response
	ha1 := md5sum("username:mount:password")
	ha2 := md5sum("GET:/mount")
	response := md5sum(ha1 + ":" + nonce + ":" + ha2)

	// Set the Authorization header
	authHeader := fmt.Sprintf(
		`Digest username="username", realm="mount", nonce="%s", uri="/mount", response="%s"`,
		nonce, response,
	)
	req.Header.Set("Authorization", authHeader)

	// Test authentication
	authenticated, err := auth.Authenticate(req, "mount")
	require.NoError(t, err)
	assert.True(t, authenticated)

	// Test with wrong password (wrong response)
	wrongResponse := md5sum(md5sum("username:mount:wrong") + ":" + nonce + ":" + ha2)
	authHeader = fmt.Sprintf(
		`Digest username="username", realm="mount", nonce="%s", uri="/mount", response="%s"`,
		nonce, wrongResponse,
	)
	req.Header.Set("Authorization", authHeader)
	authenticated, err = auth.Authenticate(req, "mount")
	require.NoError(t, err)
	assert.False(t, authenticated)

	// Test with wrong username
	authHeader = fmt.Sprintf(
		`Digest username="wrong", realm="mount", nonce="%s", uri="/mount", response="%s"`,
		nonce, response,
	)
	req.Header.Set("Authorization", authHeader)
	authenticated, err = auth.Authenticate(req, "mount")
	require.NoError(t, err)
	assert.False(t, authenticated)

	// Test with invalid nonce
	authHeader = fmt.Sprintf(
		`Digest username="username", realm="mount", nonce="invalid", uri="/mount", response="%s"`,
		response,
	)
	req.Header.Set("Authorization", authHeader)
	authenticated, err = auth.Authenticate(req, "mount")
	require.NoError(t, err)
	assert.False(t, authenticated)

	// Test without auth
	req, err = http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)
	authenticated, err = auth.Authenticate(req, "mount")
	require.NoError(t, err)
	assert.False(t, authenticated)

	// Test method
	assert.Equal(t, Digest, auth.Method())
}

func TestNoAuth(t *testing.T) {
	// Create a no-auth authenticator
	auth := NewNoAuth()

	// Create a request
	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)

	// Test authentication
	authenticated, err := auth.Authenticate(req, "mount")
	require.NoError(t, err)
	assert.True(t, authenticated)

	// Test challenge
	challenge := auth.Challenge("mount")
	assert.Equal(t, "", challenge)

	// Test method
	assert.Equal(t, None, auth.Method())
}

func TestAuthManager(t *testing.T) {
	// Create an auth manager
	manager := NewAuthManager()

	// Create authenticators
	basicAuth := NewBasicAuth()
	basicAuth.AddCredential("username", "password")

	digestAuth := NewDigestAuth()
	digestAuth.AddCredential("username", "password")

	noAuth := NewNoAuth()

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

	// Test authentication
	req, err := http.NewRequest("GET", "http://example.com", nil)
	require.NoError(t, err)
	req.SetBasicAuth("username", "password")

	authenticated, err := manager.Authenticate(req, "basic")
	require.NoError(t, err)
	assert.True(t, authenticated)

	authenticated, err = manager.Authenticate(req, "none")
	require.NoError(t, err)
	assert.True(t, authenticated)

	// Test challenge
	assert.Equal(t, `Basic realm="basic"`, manager.Challenge("basic"))
	assert.Equal(t, "", manager.Challenge("none"))
	assert.True(t, strings.HasPrefix(manager.Challenge("digest"), "Digest "))
}

func TestParseMethod(t *testing.T) {
	assert.Equal(t, None, ParseMethod("N"))
	assert.Equal(t, Basic, ParseMethod("B"))
	assert.Equal(t, Digest, ParseMethod("D"))
	assert.Equal(t, Bearer, ParseMethod("T"))
	assert.Equal(t, None, ParseMethod("X"))

	// Case insensitive
	assert.Equal(t, Basic, ParseMethod("b"))
	assert.Equal(t, Digest, ParseMethod("d"))
}

func TestMethodString(t *testing.T) {
	assert.Equal(t, "N", None.String())
	assert.Equal(t, "B", Basic.String())
	assert.Equal(t, "D", Digest.String())
	assert.Equal(t, "T", Bearer.String())
	assert.Equal(t, "U", Method(99).String())
}
