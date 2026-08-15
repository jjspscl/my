package database

import (
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/jjspscl/my/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// newMigrateDB opens a fresh temp-file database with no schema.
func newMigrateDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := Open("file:" + filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestMigrate_AppliesInFilenameOrder(t *testing.T) {
	db := newMigrateDB(t)

	// Order matters: 002 inserts into the table created by 001. If files ran
	// out of order, 002 fails.
	fsys := fstest.MapFS{
		"002_seed.sql": &fstest.MapFile{Data: []byte("INSERT INTO widgets (name) VALUES ('b');")},
		"001_widgets.sql": &fstest.MapFile{Data: []byte(`
			CREATE TABLE widgets (id INTEGER PRIMARY KEY, name TEXT NOT NULL);
		`)},
	}

	err := Migrate(db, fsys, quiet)
	require.NoError(t, err)

	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM widgets").Scan(&count))
	assert.Equal(t, 1, count, "002 ran after 001, so the seeded row exists")

	var applied int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM _migrations").Scan(&applied))
	assert.Equal(t, 2, applied)
}

func TestMigrate_IsIdempotent(t *testing.T) {
	db := newMigrateDB(t)

	fsys := fstest.MapFS{
		"001_widgets.sql": &fstest.MapFile{Data: []byte("CREATE TABLE widgets (id INTEGER PRIMARY KEY);")},
	}

	require.NoError(t, Migrate(db, fsys, quiet))
	require.NoError(t, Migrate(db, fsys, quiet), "second run must skip applied migrations without error")

	var applied int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM _migrations").Scan(&applied))
	assert.Equal(t, 1, applied)
}

func TestMigrate_EmptyFilesystem_Errors(t *testing.T) {
	db := newMigrateDB(t)
	err := Migrate(db, fstest.MapFS{}, quiet)
	assert.ErrorContains(t, err, "no migration files found")
}

func TestMigrate_EmbeddedSet_AppliesFully(t *testing.T) {
	// Guards the //go:embed wiring: if the embed pattern ever stops matching
	// the SQL files, this fails loudly instead of booting an empty schema.
	db := newMigrateDB(t)

	require.NoError(t, Migrate(db, migrations.FS, quiet))

	var applied int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM _migrations").Scan(&applied))
	assert.Equal(t, 11, applied, "all eleven migration files applied")

	// Spot-check a real table and the finance categories seed.
	var tables int
	require.NoError(t, db.QueryRow(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name IN ('transactions','habits','finance_categories','wallets')",
	).Scan(&tables))
	assert.Equal(t, 4, tables)

	var categories int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM finance_categories").Scan(&categories))
	assert.Greater(t, categories, 0)
}
