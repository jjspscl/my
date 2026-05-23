package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func Migrate(db *sql.DB, migrationsDir string) error {
	if err := ensureMigrationsTable(db); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}

	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("list migration files: %w", err)
	}

	if len(files) == 0 {
		log.Println("no migration files found")
		return nil
	}

	sort.Strings(files)

	applied, err := getApplied(db)
	if err != nil {
		return fmt.Errorf("get applied migrations: %w", err)
	}

	for _, f := range files {
		name := filepath.Base(f)
		if applied[name] {
			log.Printf("migration %s already applied, skipping", name)
			continue
		}

		content, err := os.ReadFile(f)
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

		log.Printf("migration %s applied", name)
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