package infrastructure

import (
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/jjspscl/my/internal/platform/database"
	"github.com/jjspscl/my/migrations"
)

// newTestDB opens a real temporary libSQL/SQLite file database and applies the
// full embedded migration set, mirroring production bootstrap. Repos get no
// special treatment: every query runs against real SQL, real indexes, and real
// constraints.
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := database.Open("file:" + path)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	quiet := slog.New(slog.NewTextHandler(io.Discard, nil))
	if err := database.Migrate(db, migrations.FS, quiet); err != nil {
		t.Fatalf("migrate test database: %v", err)
	}
	return db
}
