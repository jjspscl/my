package infrastructure

import (
	"database/sql"
	"io"
	"log"
	"path/filepath"
	"testing"

	"github.com/jjspscl/my/internal/platform/database"
)

// migrationsDir resolves to apps/api/migrations relative to this package
// (tests run with the package directory as the working directory).
const migrationsDir = "../../../../migrations"

// newTestDB opens a real temporary libSQL/SQLite file database and applies the
// full migration set, mirroring production bootstrap. Repos get no special
// treatment: every query runs against real SQL, real indexes, and real
// constraints.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	silenceMigrationLogs(t)
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := database.Open("file:" + path)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.Migrate(db, migrationsDir); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}

// silenceMigrationLogs quiets the per-migration log lines during tests.
func silenceMigrationLogs(t *testing.T) {
	t.Helper()
	original := log.Writer()
	log.SetOutput(io.Discard)
	t.Cleanup(func() { log.SetOutput(original) })
}
