package database

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

func Open(databaseURL string) (*sql.DB, error) {
	driver := "sqlite"
	if strings.HasPrefix(databaseURL, "libsql://") || strings.HasPrefix(databaseURL, "https://") {
		driver = "libsql"
	}

	db, err := sql.Open(driver, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("database open: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("database ping: %w", err)
	}

	return db, nil
}
