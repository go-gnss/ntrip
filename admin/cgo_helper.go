package admin

import (
	"database/sql"
	"strings"
)

// isCgoEnabled returns true if CGO is enabled
func isCgoEnabled() bool {
	// Try to open a SQLite database
	_, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		// If the error message contains "CGO_ENABLED=0", CGO is disabled
		return !strings.Contains(err.Error(), "CGO_ENABLED=0")
	}
	return true
}
