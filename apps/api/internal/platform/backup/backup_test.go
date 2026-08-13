package backup

import (
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/jjspscl/my/internal/platform/database"
	"github.com/jjspscl/my/migrations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

func newDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := database.Open("file:" + filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })
	require.NoError(t, database.Migrate(db, migrations.FS, quiet))
	return db
}

func TestSnapshotTo_CreatesRestorableFile(t *testing.T) {
	db := newDB(t)

	// Seed a wallet so the snapshot has content.
	_, err := db.Exec("INSERT INTO wallets (id, user_email, name, kind, currency, opening_balance_cents, is_default, archived_at) VALUES ('w1', 'u@example.com', 'Main', 'bank', 'PHP', 100000, 1, NULL)")
	require.NoError(t, err)

	dest := filepath.Join(t.TempDir(), "snapshot.db")
	require.NoError(t, SnapshotTo(db, dest))

	// Open the snapshot as a fresh database and verify the data survived.
	snap, err := database.Open("file:" + dest)
	require.NoError(t, err)
	defer snap.Close()

	var name, currency string
	require.NoError(t, snap.QueryRow("SELECT name, currency FROM wallets WHERE id = 'w1'").Scan(&name, &currency))
	assert.Equal(t, "Main", name)
	assert.Equal(t, "PHP", currency)
}

func TestExportTo_CoversUserTablesExcludesTokens(t *testing.T) {
	db := newDB(t)

	export, err := ExportTo(db)
	require.NoError(t, err)
	require.NotNil(t, export)

	assert.Equal(t, 12, len(export.Tables), "all user-data tables exported")
	for _, table := range exportTables {
		assert.NotNil(t, export.Tables[table], "table %s present", table)
	}
	_, hasTokens := export.Tables["magic_tokens"]
	assert.False(t, hasTokens, "magic_tokens must never be exported")
	assert.NotEmpty(t, export.ExportedAt)
	assert.NotEmpty(t, export.AppVersion)

	// Round-trip a row: habit with a completion.
	_, err = db.Exec("INSERT INTO habits (id, user_email, name, color, frequency, target_per_week, archived) VALUES ('h1', 'u@example.com', 'Run', 'green', 'daily', 1, 0)")
	require.NoError(t, err)
	export, err = ExportTo(db)
	require.NoError(t, err)
	require.Len(t, export.Tables["habits"], 1)
	assert.Equal(t, "Run", export.Tables["habits"][0]["name"])
}
