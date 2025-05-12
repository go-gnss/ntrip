package auth

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Method represents different authentication methods supported by NTRIP
type Method int

const (
	// None represents no authentication required
	None Method = iota
	// Basic represents HTTP Basic authentication
	Basic
	// Digest represents HTTP Digest authentication
	Digest
	// Bearer represents token-based authentication
	Bearer
)

// String returns the string representation of the authentication method
func (m Method) String() string {
	switch m {
	case None:
		return "N"
	case Basic:
		return "B"
	case Digest:
		return "D"
	case Bearer:
		return "T" // Token-based
	default:
		return "U" // Unknown
	}
}

// ParseMethod converts a string to an authentication Method
func ParseMethod(s string) Method {
	switch strings.ToUpper(s) {
	case "N":
		return None
	case "B":
		return Basic
	case "D":
		return Digest
	case "T":
		return Bearer
	default:
		return None
	}
}

// Authenticator defines the interface for authentication providers
type Authenticator interface {
	// Authenticate checks if the request is authenticated for the given mount point
	Authenticate(r *http.Request, mount string) (bool, error)

	// Challenge generates the appropriate authentication challenge header
	Challenge(mount string) string

	// Method returns the authentication method used by this authenticator
	Method() Method
}

// BasicAuth implements basic authentication
type BasicAuth struct {
	credentials map[string]string // username -> password
	mu          sync.RWMutex
}

// NewBasicAuth creates a new basic authenticator
func NewBasicAuth() *BasicAuth {
	return &BasicAuth{
		credentials: make(map[string]string),
	}
}

// AddCredential adds a username/password pair
func (b *BasicAuth) AddCredential(username, password string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.credentials[username] = password
}

// Authenticate implements the Authenticator interface
func (b *BasicAuth) Authenticate(r *http.Request, mount string) (bool, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false, nil
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	storedPassword, exists := b.credentials[username]
	if !exists {
		return false, nil
	}

	return password == storedPassword, nil
}

// Challenge implements the Authenticator interface
func (b *BasicAuth) Challenge(mount string) string {
	return fmt.Sprintf(`Basic realm="%s"`, mount)
}

// Method implements the Authenticator interface
func (b *BasicAuth) Method() Method {
	return Basic
}

// DigestAuth implements digest authentication according to RFC 2617
type DigestAuth struct {
	credentials map[string]string    // username -> password
	nonces      map[string]time.Time // nonce -> expiry time
	mu          sync.RWMutex
	nonceExpiry time.Duration
}

// NewDigestAuth creates a new digest authenticator
func NewDigestAuth() *DigestAuth {
	return &DigestAuth{
		credentials: make(map[string]string),
		nonces:      make(map[string]time.Time),
		nonceExpiry: 5 * time.Minute,
	}
}

// AddCredential adds a username/password pair
func (d *DigestAuth) AddCredential(username, password string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.credentials[username] = password
}

// generateNonce creates a new nonce value
func (d *DigestAuth) generateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	nonce := hex.EncodeToString(b)

	d.mu.Lock()
	defer d.mu.Unlock()
	d.nonces[nonce] = time.Now().Add(d.nonceExpiry)

	return nonce
}

// cleanupExpiredNonces removes expired nonces
func (d *DigestAuth) cleanupExpiredNonces() {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := time.Now()
	for nonce, expiry := range d.nonces {
		if now.After(expiry) {
			delete(d.nonces, nonce)
		}
	}
}

// Authenticate implements the Authenticator interface
func (d *DigestAuth) Authenticate(r *http.Request, mount string) (bool, error) {
	// Clean up expired nonces periodically
	d.cleanupExpiredNonces()

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Digest ") {
		return false, nil
	}

	// Parse the digest auth header
	parts := strings.TrimPrefix(authHeader, "Digest ")
	params := make(map[string]string)

	for _, part := range strings.Split(parts, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}

		key := kv[0]
		value := strings.Trim(kv[1], "\"")
		params[key] = value
	}

	// Check required parameters
	username, ok1 := params["username"]
	realm, ok2 := params["realm"]
	nonce, ok3 := params["nonce"]
	uri, ok4 := params["uri"]
	response, ok5 := params["response"]

	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
		return false, nil
	}

	// Verify the nonce is valid
	d.mu.RLock()
	expiry, nonceValid := d.nonces[nonce]
	d.mu.RUnlock()

	if !nonceValid || time.Now().After(expiry) {
		return false, nil
	}

	// Get the password
	d.mu.RLock()
	password, exists := d.credentials[username]
	d.mu.RUnlock()

	if !exists {
		return false, nil
	}

	// Calculate expected response
	ha1 := md5sum(fmt.Sprintf("%s:%s:%s", username, realm, password))
	ha2 := md5sum(fmt.Sprintf("%s:%s", r.Method, uri))
	expected := md5sum(fmt.Sprintf("%s:%s:%s", ha1, nonce, ha2))

	return expected == response, nil
}

// Challenge implements the Authenticator interface
func (d *DigestAuth) Challenge(mount string) string {
	nonce := d.generateNonce()
	return fmt.Sprintf(`Digest realm="%s", nonce="%s", algorithm=MD5, qop="auth"`, mount, nonce)
}

// Method implements the Authenticator interface
func (d *DigestAuth) Method() Method {
	return Digest
}

// md5sum calculates the MD5 hash of a string
func md5sum(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// NoAuth implements a pass-through authenticator
type NoAuth struct{}

// NewNoAuth creates a new no-auth authenticator
func NewNoAuth() *NoAuth {
	return &NoAuth{}
}

// Authenticate implements the Authenticator interface
func (n *NoAuth) Authenticate(r *http.Request, mount string) (bool, error) {
	return true, nil
}

// Challenge implements the Authenticator interface
func (n *NoAuth) Challenge(mount string) string {
	return ""
}

// Method implements the Authenticator interface
func (n *NoAuth) Method() Method {
	return None
}

// AuthManager manages multiple authentication methods
type AuthManager struct {
	mountAuth   map[string]Authenticator
	defaultAuth Authenticator
	mu          sync.RWMutex
}

// NewAuthManager creates a new authentication manager
func NewAuthManager() *AuthManager {
	return &AuthManager{
		mountAuth:   make(map[string]Authenticator),
		defaultAuth: NewNoAuth(),
	}
}

// SetMountAuthenticator sets the authenticator for a specific mount point
func (am *AuthManager) SetMountAuthenticator(mount string, auth Authenticator) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.mountAuth[mount] = auth
}

// SetDefaultAuthenticator sets the default authenticator
func (am *AuthManager) SetDefaultAuthenticator(auth Authenticator) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.defaultAuth = auth
}

// GetAuthenticator returns the authenticator for a mount point
func (am *AuthManager) GetAuthenticator(mount string) Authenticator {
	am.mu.RLock()
	defer am.mu.RUnlock()

	auth, exists := am.mountAuth[mount]
	if exists {
		return auth
	}

	return am.defaultAuth
}

// Authenticate authenticates a request for a mount point
func (am *AuthManager) Authenticate(r *http.Request, mount string) (bool, error) {
	auth := am.GetAuthenticator(mount)
	return auth.Authenticate(r, mount)
}

// Challenge returns the authentication challenge for a mount point
func (am *AuthManager) Challenge(mount string) string {
	auth := am.GetAuthenticator(mount)
	return auth.Challenge(mount)
}
