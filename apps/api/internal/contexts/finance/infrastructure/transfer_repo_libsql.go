package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

type TransferRepoLibSQL struct {
	db *sql.DB
}

func NewTransferRepoLibSQL(db *sql.DB) *TransferRepoLibSQL {
	return &TransferRepoLibSQL{db: db}
}

func (r *TransferRepoLibSQL) Save(ctx context.Context, t *domain.WalletTransfer) error {
	_, err := executor(ctx, r.db).ExecContext(ctx, `
		INSERT INTO wallet_transfers (id, user_email, from_wallet_id, to_wallet_id, amount_cents, from_amount_cents, to_amount_cents, description, transfer_date, idempotency_key, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.UserEmail, t.FromWalletID, t.ToWalletID, t.FromAmountCents, t.FromAmountCents, t.ToAmountCents, t.Description,
		t.TransferDate.Format("2006-01-02"), optionalString(t.IdempotencyKey), t.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save transfer: %w", err)
	}
	return nil
}

func (r *TransferRepoLibSQL) FindByID(ctx context.Context, id string) (*domain.WalletTransfer, error) {
	row := executor(ctx, r.db).QueryRowContext(ctx, `
		SELECT id, user_email, from_wallet_id, to_wallet_id, amount_cents, from_amount_cents, to_amount_cents, description, transfer_date, idempotency_key, created_at
		FROM wallet_transfers WHERE id = ?
	`, id)
	return scanTransfer(row)
}

func (r *TransferRepoLibSQL) FindByIdempotencyKey(ctx context.Context, userEmail, key string) (*domain.WalletTransfer, error) {
	row := executor(ctx, r.db).QueryRowContext(ctx, `
		SELECT id, user_email, from_wallet_id, to_wallet_id, amount_cents, from_amount_cents, to_amount_cents, description, transfer_date, idempotency_key, created_at
		FROM wallet_transfers WHERE user_email = ? AND idempotency_key = ?
	`, userEmail, key)
	t, err := scanTransfer(row)
	if err != nil && err.Error() == "transfer not found" {
		return nil, nil
	}
	return t, err
}

func (r *TransferRepoLibSQL) ListByUser(ctx context.Context, userEmail string, limit, offset int) ([]*domain.WalletTransfer, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := executor(ctx, r.db).QueryContext(ctx, `
		SELECT id, user_email, from_wallet_id, to_wallet_id, amount_cents, from_amount_cents, to_amount_cents, description, transfer_date, idempotency_key, created_at
		FROM wallet_transfers WHERE user_email = ?
		ORDER BY transfer_date DESC, created_at DESC
		LIMIT ? OFFSET ?
	`, userEmail, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list transfers: %w", err)
	}
	defer rows.Close()

	var transfers []*domain.WalletTransfer
	for rows.Next() {
		t, err := scanTransfer(rows)
		if err != nil {
			return nil, err
		}
		transfers = append(transfers, t)
	}
	return transfers, rows.Err()
}

func scanTransfer(row scannable) (*domain.WalletTransfer, error) {
	var t domain.WalletTransfer
	var transferDate, createdAt string
	var amountCents, fromAmountCents, toAmountCents sql.NullInt64
	var idempotencyKey *string

	err := row.Scan(&t.ID, &t.UserEmail, &t.FromWalletID, &t.ToWalletID, &amountCents, &fromAmountCents, &toAmountCents, &t.Description, &transferDate, &idempotencyKey, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transfer not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scan transfer: %w", err)
	}

	t.FromAmountCents = fromAmountCents.Int64
	if !fromAmountCents.Valid {
		t.FromAmountCents = amountCents.Int64
	}
	t.ToAmountCents = toAmountCents.Int64
	if !toAmountCents.Valid {
		t.ToAmountCents = amountCents.Int64
	}

	t.TransferDate, _ = time.Parse("2006-01-02", transferDate)
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	if idempotencyKey != nil {
		t.IdempotencyKey = *idempotencyKey
	}

	return &t, nil
}
