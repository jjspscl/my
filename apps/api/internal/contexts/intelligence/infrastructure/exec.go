package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
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
// repos pick the *sql.Tx up from the context via executor().
type Coordinator struct {
	db *sql.DB
}

func NewCoordinator(db *sql.DB) *Coordinator {
	return &Coordinator{db: db}
}

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
