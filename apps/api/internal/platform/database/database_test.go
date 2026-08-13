package database

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_AppliesPragmas(t *testing.T) {
	db, err := Open("file:" + filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var fk int
	require.NoError(t, db.QueryRow("PRAGMA foreign_keys").Scan(&fk))
	assert.Equal(t, 1, fk, "foreign_keys must be ON")

	var busy int
	require.NoError(t, db.QueryRow("PRAGMA busy_timeout").Scan(&busy))
	assert.Equal(t, 5000, busy)

	var journal string
	require.NoError(t, db.QueryRow("PRAGMA journal_mode").Scan(&journal))
	assert.Equal(t, "wal", journal, "journal_mode must be WAL")
}

func TestWithPragmas_PreservesExistingQuery(t *testing.T) {
	got := withPragmas("file:/data/my.db?mode=rwc")
	assert.Contains(t, got, "mode=rwc&")
	assert.Contains(t, got, "_pragma=journal_mode(WAL)")
	assert.Equal(t, "file:/data/my.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)", withPragmas("file:/data/my.db"))
}

func TestOpen_SingleWriterPool(t *testing.T) {
	db, err := Open("file:" + filepath.Join(t.TempDir(), "test.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	assert.Equal(t, 1, db.Stats().MaxOpenConnections)
}
