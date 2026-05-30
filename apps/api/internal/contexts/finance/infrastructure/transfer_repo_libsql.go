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
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO wallet_transfers (id, user_email, from_wallet_id, to_wallet_id, amount_cents, description, transfer_date, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, t.ID, t.UserEmail, t.FromWalletID, t.ToWalletID, t.AmountCents, t.Description,
		t.TransferDate.Format("2006-01-02"), t.CreatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save transfer: %w", err)
	}
	return nil
}

func (r *TransferRepoLibSQL) FindByID(ctx context.Context, id string) (*domain.WalletTransfer, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_email, from_wallet_id, to_wallet_id, amount_cents, description, transfer_date, created_at
		FROM wallet_transfers WHERE id = ?
	`, id)
	return scanTransfer(row)
}

func (r *TransferRepoLibSQL) ListByUser(ctx context.Context, userEmail string, limit, offset int) ([]*domain.WalletTransfer, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_email, from_wallet_id, to_wallet_id, amount_cents, description, transfer_date, created_at
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

	err := row.Scan(&t.ID, &t.UserEmail, &t.FromWalletID, &t.ToWalletID, &t.AmountCents, &t.Description, &transferDate, &createdAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("transfer not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scan transfer: %w", err)
	}

	t.TransferDate, _ = time.Parse("2006-01-02", transferDate)
	t.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)

	return &t, nil
}
