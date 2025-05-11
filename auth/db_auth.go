package auth

import (
	"net/http"
	"strings"

	"github.com/go-gnss/ntrip/admin"
)

// DBAuth implements authentication using the admin database
type DBAuth struct {
	db admin.DBInterface
}

// NewDBAuth creates a new database-backed authenticator
func NewDBAuth(db admin.DBInterface) *DBAuth {
	return &DBAuth{
		db: db,
	}
}

// Authenticate implements the Authenticator interface
func (d *DBAuth) Authenticate(r *http.Request, mount string) (bool, error) {
	username, password, ok := r.BasicAuth()
	if !ok {
		return false, nil
	}

	// Validate user credentials
	valid, err := d.db.ValidateUserCredentials(username, password)
	if err != nil || !valid {
		return false, err
	}

	// Check if user has access to this mount
	user, err := d.db.GetUser(username)
	if err != nil {
		return false, err
	}

	// If mounts_allowed is empty, user has access to all mounts
	if user.MountsAllowed == "" {
		return true, nil
	}

	// Check if the mount is in the allowed list
	allowedMounts := strings.Split(user.MountsAllowed, ",")
	for _, allowed := range allowedMounts {
		if strings.TrimSpace(allowed) == mount {
			return true, nil
		}
	}

	return false, nil
}

// Challenge implements the Authenticator interface
func (d *DBAuth) Challenge(mount string) string {
	return `Basic realm="` + mount + `"`
}

// Method implements the Authenticator interface
func (d *DBAuth) Method() Method {
	return Basic
}

// AuthenticateMountpoint authenticates a mountpoint
func (d *DBAuth) AuthenticateMountpoint(mount, password string) (bool, error) {
	return d.db.ValidateMountpointCredentials(mount, password)
}
