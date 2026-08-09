package database

import (
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"time"
)

// Migrate applies every *.sql file in fsys that has not already been applied,
// in filename order. A missing or empty migration set is an error — silently
// booting with no schema produces a "healthy" app whose every query fails.
func Migrate(db *sql.DB, fsys fs.FS, log *slog.Logger) error {
	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}

	files, err := fs.Glob(fsys, "*.sql")
	if err != nil {
		return fmt.Errorf("list migration files: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no migration files found in embedded filesystem")
	}

	sort.Strings(files)

	applied, err := getApplied(db)
	if err != nil {
		return fmt.Errorf("get applied migrations: %w", err)
	}

	for _, name := range files {
		if applied[name] {
			log.Info("migration already applied, skipping", slog.String("file", name))
			continue
		}

		content, err := fs.ReadFile(fsys, name)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", name, err)
		}

		if _, err := tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("execute migration %s: %w", name, err)
		}

		if _, err := tx.Exec(
			"INSERT INTO _migrations (filename, applied_at) VALUES (?, ?)",
			name, time.Now().UTC(),
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", name, err)
		}

		log.Info("migration applied", slog.String("file", name))
	}

	return nil
}

func VerifySchema(db *sql.DB) error {
	var table string
	err := db.QueryRow("SELECT name FROM sqlite_master WHERE type = 'table' AND name = '_migrations'").Scan(&table)
	if err == sql.ErrNoRows {
		return fmt.Errorf("database schema is not initialized; run mise run migrate before starting standalone my-mcp")
	}
	if err != nil {
		return fmt.Errorf("verify database schema: %w", err)
	}
	return nil
}

func ensureMigrationsTable(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS _migrations (
			filename TEXT PRIMARY KEY,
			applied_at DATETIME NOT NULL
		)
	`)
	return err
}

func getApplied(db *sql.DB) (map[string]bool, error) {
	rows, err := db.Query("SELECT filename FROM _migrations ORDER BY filename")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		applied[name] = true
	}

	return applied, rows.Err()
}
