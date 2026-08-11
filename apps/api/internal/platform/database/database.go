package database

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

// sqlitePragmas are appended to local SQLite DSNs. WAL makes concurrent
// readers safe and is what allows the API server and a standalone my-mcp to
// share one database file; busy_timeout makes writers wait instead of failing
// with SQLITE_BUSY; foreign_keys activates the ON DELETE CASCADE clauses the
// schema already declares (they are inert with the default OFF).
const sqlitePragmas = "_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"

func Open(databaseURL string) (*sql.DB, error) {
	driver := "sqlite"
	if strings.HasPrefix(databaseURL, "libsql://") || strings.HasPrefix(databaseURL, "https://") {
		driver = "libsql"
	} else {
		databaseURL = withPragmas(databaseURL)
	}

	db, err := sql.Open(driver, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("database open: %w", err)
	}
	// One writer at a time: the app is single-user, and bounding the pool
	// eliminates SQLITE_BUSY between the API and MCP processes on one file.
	// WAL still allows concurrent readers.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database ping: %w", err)
	}

	return db, nil
}

// withPragmas appends the pragma query parameters to a file DSN, preserving
// any existing query string.
func withPragmas(dsn string) string {
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + sqlitePragmas
}
