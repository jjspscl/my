package infrastructure

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

type WalletRepoLibSQL struct {
	db *sql.DB
}

func NewWalletRepoLibSQL(db *sql.DB) *WalletRepoLibSQL {
	return &WalletRepoLibSQL{db: db}
}

func (r *WalletRepoLibSQL) Save(ctx context.Context, wallet *domain.Wallet) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO wallets (id, user_email, name, kind, currency, opening_balance_cents, is_default, archived_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, wallet.ID, wallet.UserEmail, wallet.Name, wallet.Kind, wallet.Currency,
		wallet.OpeningBalanceCents, boolToInt(wallet.IsDefault), nullableTime(wallet.ArchivedAt),
		wallet.CreatedAt.Format(time.RFC3339), wallet.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save wallet: %w", err)
	}
	return nil
}

func (r *WalletRepoLibSQL) FindByID(ctx context.Context, id string) (*domain.Wallet, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_email, name, kind, currency, opening_balance_cents, is_default, archived_at, created_at, updated_at
		FROM wallets WHERE id = ?
	`, id)
	return scanWallet(row)
}

func (r *WalletRepoLibSQL) ListByUser(ctx context.Context, userEmail string) ([]*domain.Wallet, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_email, name, kind, currency, opening_balance_cents, is_default, archived_at, created_at, updated_at
		FROM wallets WHERE user_email = ? AND archived_at IS NULL
		ORDER BY is_default DESC, name ASC
	`, userEmail)
	if err != nil {
		return nil, fmt.Errorf("list wallets: %w", err)
	}
	defer rows.Close()

	var wallets []*domain.Wallet
	for rows.Next() {
		w, err := scanWallet(rows)
		if err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}
	return wallets, rows.Err()
}

func (r *WalletRepoLibSQL) Update(ctx context.Context, wallet *domain.Wallet) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE wallets SET name = ?, kind = ?, opening_balance_cents = ?, is_default = ?, updated_at = ?
		WHERE id = ? AND user_email = ?
	`, wallet.Name, wallet.Kind, wallet.OpeningBalanceCents, boolToInt(wallet.IsDefault),
		wallet.UpdatedAt.Format(time.RFC3339), wallet.ID, wallet.UserEmail)
	if err != nil {
		return fmt.Errorf("update wallet: %w", err)
	}
	return nil
}

func (r *WalletRepoLibSQL) Archive(ctx context.Context, id, userEmail string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE wallets SET archived_at = datetime('now'), updated_at = datetime('now')
		WHERE id = ? AND user_email = ?
	`, id, userEmail)
	if err != nil {
		return fmt.Errorf("archive wallet: %w", err)
	}
	return nil
}

func (r *WalletRepoLibSQL) FindDefault(ctx context.Context, userEmail string) (*domain.Wallet, error) {
	// First try to find the explicit default
	row := r.db.QueryRowContext(ctx, `
		SELECT id, user_email, name, kind, currency, opening_balance_cents, is_default, archived_at, created_at, updated_at
		FROM wallets WHERE user_email = ? AND is_default = 1 AND archived_at IS NULL
	`, userEmail)
	w, err := scanWallet(row)
	if err == nil {
		return w, nil
	}

	// Fallback: get the first active wallet
	row = r.db.QueryRowContext(ctx, `
		SELECT id, user_email, name, kind, currency, opening_balance_cents, is_default, archived_at, created_at, updated_at
		FROM wallets WHERE user_email = ? AND archived_at IS NULL
		ORDER BY created_at ASC LIMIT 1
	`, userEmail)
	return scanWallet(row)
}

func (r *WalletRepoLibSQL) GetBalancesByUser(ctx context.Context, userEmail string) ([]*domain.WalletBalance, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			w.id,
			w.opening_balance_cents,
			COALESCE((SELECT SUM(amount_cents) FROM transactions t WHERE t.wallet_id = w.id AND t.type = 'income'), 0) as income_cents,
			COALESCE((SELECT SUM(amount_cents) FROM transactions t WHERE t.wallet_id = w.id AND t.type = 'expense'), 0) as expense_cents,
			COALESCE((SELECT SUM(amount_cents) FROM wallet_transfers wt WHERE wt.to_wallet_id = w.id), 0) as incoming_transfer_cents,
			COALESCE((SELECT SUM(amount_cents) FROM wallet_transfers wt WHERE wt.from_wallet_id = w.id), 0) as outgoing_transfer_cents
		FROM wallets w
		WHERE w.user_email = ? AND w.archived_at IS NULL
		ORDER BY w.is_default DESC, w.name ASC
	`, userEmail)
	if err != nil {
		return nil, fmt.Errorf("get balances: %w", err)
	}
	defer rows.Close()

	var balances []*domain.WalletBalance
	for rows.Next() {
		var b domain.WalletBalance
		var walletID string
		if err := rows.Scan(&walletID, &b.Wallet.OpeningBalanceCents, &b.IncomeCents, &b.ExpenseCents, &b.IncomingTransferCents, &b.OutgoingTransferCents); err != nil {
			return nil, fmt.Errorf("scan balance: %w", err)
		}
		b.Wallet.ID = walletID
		b.BalanceCents = b.Wallet.OpeningBalanceCents + b.IncomeCents - b.ExpenseCents + b.IncomingTransferCents - b.OutgoingTransferCents
		balances = append(balances, &b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Build a map of wallet balances keyed by ID, then merge with wallet data
	balanceMap := make(map[string]*domain.WalletBalance, len(balances))
	for _, b := range balances {
		balanceMap[b.Wallet.ID] = b
	}

	wallets, err := r.ListByUser(ctx, userEmail)
	if err != nil {
		return nil, fmt.Errorf("list wallets for balances: %w", err)
	}

	result := make([]*domain.WalletBalance, 0, len(wallets))
	for _, w := range wallets {
		if b, ok := balanceMap[w.ID]; ok {
			b.Wallet = *w
			result = append(result, b)
		} else {
			// Wallet has no transactions or transfers yet
			result = append(result, &domain.WalletBalance{
				Wallet:       *w,
				BalanceCents: w.OpeningBalanceCents,
			})
		}
	}

	return result, nil
}

func scanWallet(row scannable) (*domain.Wallet, error) {
	var w domain.Wallet
	var archivedAt *string
	var createdAt, updatedAt string
	var isDefault int

	err := row.Scan(&w.ID, &w.UserEmail, &w.Name, &w.Kind, &w.Currency,
		&w.OpeningBalanceCents, &isDefault, &archivedAt, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("wallet not found")
	}
	if err != nil {
		return nil, fmt.Errorf("scan wallet: %w", err)
	}

	w.IsDefault = isDefault == 1
	if archivedAt != nil && *archivedAt != "" {
		parsed, _ := time.Parse(time.RFC3339, *archivedAt)
		w.ArchivedAt = &parsed
	}
	w.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	w.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

	return &w, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
