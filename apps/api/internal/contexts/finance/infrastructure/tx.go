package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// execer is the common surface shared by *sql.DB and *sql.Tx, letting repo
// methods transparently run against either.
type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type txKey struct{}

// Coordinator runs a function inside a single database transaction. Participating
// repos pick the *sql.Tx up from the context via executor(), so multi-repo writes
// (e.g. a goal contribution plus its backing transfer) become atomic.
type Coordinator struct {
	db *sql.DB
}

func NewCoordinator(db *sql.DB) *Coordinator {
	return &Coordinator{db: db}
}

// WithTx runs fn inside a transaction. If fn returns an error, the transaction
// is rolled back. The fn context carries the transaction.
func (c *Coordinator) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if err := fn(context.WithValue(ctx, txKey{}, tx)); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

// executor returns the transaction carried by ctx when one is present,
// otherwise the fallback database handle.
func executor(ctx context.Context, fallback *sql.DB) execer {
	if tx, ok := ctx.Value(txKey{}).(*sql.Tx); ok {
		return tx
	}
	return fallback
}

// isUniqueViolation reports whether err is a SQLite UNIQUE constraint failure.
// Used to detect idempotent replay races after a concurrent insert won the
// unique-index race.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}

// optionalString converts an empty string to nil so it is stored as SQL NULL.
func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
