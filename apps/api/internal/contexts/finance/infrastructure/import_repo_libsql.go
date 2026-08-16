package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

type ImportRepoLibSQL struct {
	db *sql.DB
}

func NewImportRepoLibSQL(db *sql.DB) *ImportRepoLibSQL {
	return &ImportRepoLibSQL{db: db}
}

func (r *ImportRepoLibSQL) SaveBatch(ctx context.Context, batch *domain.ImportBatch) error {
	summary, err := json.Marshal(batch.Summary)
	if err != nil {
		return fmt.Errorf("marshal summary: %w", err)
	}
	_, err = executor(ctx, r.db).ExecContext(ctx,
		`INSERT INTO finance_imports
			(id, user_email, provider, file_fingerprint, statement_from, statement_to,
			 wallet_id, created_wallet_id, opening_balance_cents, ending_balance_cents,
			 reconciliation, status, integrity_status, summary_json, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		batch.ID, batch.UserEmail, batch.Provider, batch.FileFingerprint,
		batch.StatementFrom.Format("2006-01-02"), batch.StatementTo.Format("2006-01-02"),
		batch.WalletID, nullableString(optionalString(batch.CreatedWalletID)),
		batch.OpeningBalanceCents, batch.EndingBalanceCents,
		batch.Reconciliation, batch.Status, batch.Integrity, string(summary),
		batch.CreatedAt.Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("save import batch: %w", err)
	}
	return nil
}

// SaveEntries persists every entry of a batch.
func (r *ImportRepoLibSQL) SaveEntries(ctx context.Context, entries []*domain.ImportEntry) error {
	for _, e := range entries {
		_, err := executor(ctx, r.db).ExecContext(ctx,
			`INSERT INTO finance_import_entries
				(id, import_id, source_reference, occurred_at, amount_cents, kind,
				 category, description, counterparty, counter_wallet_id, outcome,
				 entity_type, entity_id, entity_status)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			e.ID, e.ImportID, e.SourceReference, e.OccurredAt.Format(time.RFC3339),
			e.AmountCents, e.Kind, e.Category, e.Description,
			nullableString(optionalString(e.Counterparty)),
			nullableString(optionalString(e.CounterWalletID)),
			e.Outcome, nullableString(optionalString(e.EntityType)),
			nullableString(optionalString(e.EntityID)),
			entityStatusOrActive(e.EntityStatus),
		)
		if err != nil {
			return fmt.Errorf("save import entry: %w", err)
		}
	}
	return nil
}

func (r *ImportRepoLibSQL) FindByFingerprint(ctx context.Context, userEmail, fingerprint string) (*domain.ImportBatch, error) {
	row := executor(ctx, r.db).QueryRowContext(ctx,
		`SELECT id, user_email, provider, file_fingerprint, statement_from, statement_to,
		        wallet_id, created_wallet_id, opening_balance_cents, ending_balance_cents,
		        reconciliation, status, integrity_status, summary_json, created_at, rolled_back_at
		 FROM finance_imports
		 WHERE user_email = ? AND file_fingerprint = ?`, userEmail, fingerprint)
	batch, err := scanImportBatch(row)
	if err != nil && err.Error() == "import batch not found" {
		return nil, nil
	}
	return batch, err
}

func (r *ImportRepoLibSQL) FindBatchByID(ctx context.Context, id, userEmail string) (*domain.ImportBatch, error) {
	row := executor(ctx, r.db).QueryRowContext(ctx,
		`SELECT id, user_email, provider, file_fingerprint, statement_from, statement_to,
		        wallet_id, created_wallet_id, opening_balance_cents, ending_balance_cents,
		        reconciliation, status, integrity_status, summary_json, created_at, rolled_back_at
		 FROM finance_imports
		 WHERE id = ? AND user_email = ?`, id, userEmail)
	return scanImportBatch(row)
}

func (r *ImportRepoLibSQL) ListByUser(ctx context.Context, userEmail string, limit, offset int) ([]*domain.ImportBatch, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx,
		`SELECT id, user_email, provider, file_fingerprint, statement_from, statement_to,
		        wallet_id, created_wallet_id, opening_balance_cents, ending_balance_cents,
		        reconciliation, status, integrity_status, summary_json, created_at, rolled_back_at
		 FROM finance_imports
		 WHERE user_email = ?
		 ORDER BY created_at DESC
		 LIMIT ? OFFSET ?`, userEmail, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list import batches: %w", err)
	}
	defer rows.Close()

	var batches []*domain.ImportBatch
	for rows.Next() {
		b, err := scanImportBatch(rows)
		if err != nil {
			return nil, err
		}
		batches = append(batches, b)
	}
	return batches, rows.Err()
}

func (r *ImportRepoLibSQL) ListEntries(ctx context.Context, importID string) ([]*domain.ImportEntry, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx,
		`SELECT id, import_id, source_reference, occurred_at, amount_cents, kind,
		        category, description, counterparty, counter_wallet_id, outcome,
		        entity_type, entity_id, entity_status, entity_modified_at, entity_deleted_at
		 FROM finance_import_entries
		 WHERE import_id = ?
		 ORDER BY occurred_at`, importID)
	if err != nil {
		return nil, fmt.Errorf("list import entries: %w", err)
	}
	defer rows.Close()

	var entries []*domain.ImportEntry
	for rows.Next() {
		e, err := scanImportEntry(rows)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func (r *ImportRepoLibSQL) MarkRolledBack(ctx context.Context, id string, rolledBackAt time.Time) error {
	_, err := executor(ctx, r.db).ExecContext(ctx,
		`UPDATE finance_imports SET status = ?, rolled_back_at = ? WHERE id = ?`,
		domain.ImportStatusRolledBack, rolledBackAt.Format(time.RFC3339), id)
	if err != nil {
		return fmt.Errorf("mark import rolled back: %w", err)
	}
	return nil
}

// MarkTransactionProvenance records an edit/delete of a transaction that an
// import created: the immutable statement entry gains the lifecycle status,
// and the owning batch is flagged as modified. No-ops when the transaction is
// not backed by an import entry.
func (r *ImportRepoLibSQL) MarkTransactionProvenance(ctx context.Context, txID, status string, at time.Time) error {
	var column string
	switch status {
	case domain.EntityStatusModified:
		column = "entity_modified_at"
	case domain.EntityStatusDeleted:
		column = "entity_deleted_at"
	default:
		return fmt.Errorf("invalid entity lifecycle status: %s", status)
	}
	atStr := at.Format(time.RFC3339)

	if _, err := executor(ctx, r.db).ExecContext(ctx,
		`UPDATE finance_import_entries
		 SET entity_status = ?, `+column+` = ?
		 WHERE entity_type = 'transaction' AND entity_id = ?`,
		status, atStr, txID); err != nil {
		return fmt.Errorf("mark import entry lifecycle: %w", err)
	}

	if _, err := executor(ctx, r.db).ExecContext(ctx,
		`UPDATE finance_imports
		 SET integrity_status = ?
		 WHERE id IN (
		   SELECT import_id FROM finance_import_entries
		   WHERE entity_type = 'transaction' AND entity_id = ?
		 )`,
		domain.BatchIntegrityModified, txID); err != nil {
		return fmt.Errorf("mark import batch modified: %w", err)
	}
	return nil
}

func (r *ImportRepoLibSQL) DeleteTransactionEntity(ctx context.Context, entityType, entityID, userEmail string) (bool, error) {
	if entityType == "" || entityID == "" {
		return false, nil
	}
	var result sql.Result
	var err error
	switch entityType {
	case "transaction":
		result, err = executor(ctx, r.db).ExecContext(ctx,
			"DELETE FROM transactions WHERE id = ? AND user_email = ?", entityID, userEmail)
	case "transfer":
		result, err = executor(ctx, r.db).ExecContext(ctx,
			"DELETE FROM wallet_transfers WHERE id = ? AND user_email = ?", entityID, userEmail)
	default:
		return false, fmt.Errorf("unknown entity type: %s", entityType)
	}
	if err != nil {
		return false, fmt.Errorf("delete import entity: %w", err)
	}
	n, _ := result.RowsAffected()
	return n > 0, nil
}

func (r *ImportRepoLibSQL) DeleteWallet(ctx context.Context, id, userEmail string) error {
	_, err := executor(ctx, r.db).ExecContext(ctx,
		"DELETE FROM wallets WHERE id = ? AND user_email = ?", id, userEmail)
	if err != nil {
		return fmt.Errorf("delete import wallet: %w", err)
	}
	return nil
}

func (r *ImportRepoLibSQL) CountTransactionsForWallet(ctx context.Context, walletID string) (int, error) {
	var n int
	err := executor(ctx, r.db).QueryRowContext(ctx,
		"SELECT COUNT(*) FROM transactions WHERE wallet_id = ?", walletID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count wallet transactions: %w", err)
	}
	return n, nil
}

func (r *ImportRepoLibSQL) CountTransfersForWallet(ctx context.Context, walletID string) (int, error) {
	var n int
	err := executor(ctx, r.db).QueryRowContext(ctx,
		`SELECT COUNT(*) FROM wallet_transfers WHERE from_wallet_id = ? OR to_wallet_id = ?`,
		walletID, walletID).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count wallet transfers: %w", err)
	}
	return n, nil
}

type importBatchScanner interface {
	Scan(dest ...any) error
}

func scanImportBatch(row importBatchScanner) (*domain.ImportBatch, error) {
	var b domain.ImportBatch
	var fromStr, toStr, summaryStr, createdAtStr string
	var walletID, createdWalletID, rolledBackAtStr *string
	if err := row.Scan(&b.ID, &b.UserEmail, &b.Provider, &b.FileFingerprint,
		&fromStr, &toStr, &walletID, &createdWalletID, &b.OpeningBalanceCents,
		&b.EndingBalanceCents, &b.Reconciliation, &b.Status, &b.Integrity, &summaryStr,
		&createdAtStr, &rolledBackAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("import batch not found")
		}
		return nil, fmt.Errorf("scan import batch: %w", err)
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		return nil, fmt.Errorf("parse statement_from: %w", err)
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		return nil, fmt.Errorf("parse statement_to: %w", err)
	}
	createdAt, err := parseDatetime(createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	if walletID != nil {
		b.WalletID = *walletID
	}
	if createdWalletID != nil {
		b.CreatedWalletID = *createdWalletID
	}
	if rolledBackAtStr != nil {
		t, err := parseDatetime(*rolledBackAtStr)
		if err != nil {
			return nil, fmt.Errorf("parse rolled_back_at: %w", err)
		}
		b.RolledBackAt = &t
	}
	if err := json.Unmarshal([]byte(summaryStr), &b.Summary); err != nil {
		return nil, fmt.Errorf("parse summary: %w", err)
	}
	b.StatementFrom, b.StatementTo, b.CreatedAt = from, to, createdAt
	return &b, nil
}

type importEntryScanner interface {
	Scan(dest ...any) error
}

func scanImportEntry(row importEntryScanner) (*domain.ImportEntry, error) {
	var e domain.ImportEntry
	var occurredAtStr string
	var counterparty, counterWalletID, entityType, entityID, modifiedAtStr, deletedAtStr *string
	if err := row.Scan(&e.ID, &e.ImportID, &e.SourceReference, &occurredAtStr,
		&e.AmountCents, &e.Kind, &e.Category, &e.Description, &counterparty,
		&counterWalletID, &e.Outcome, &entityType, &entityID,
		&e.EntityStatus, &modifiedAtStr, &deletedAtStr); err != nil {
		return nil, fmt.Errorf("scan import entry: %w", err)
	}
	occurredAt, err := parseDatetime(occurredAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse occurred_at: %w", err)
	}
	if counterparty != nil {
		e.Counterparty = *counterparty
	}
	if counterWalletID != nil {
		e.CounterWalletID = *counterWalletID
	}
	if entityType != nil {
		e.EntityType = *entityType
	}
	if entityID != nil {
		e.EntityID = *entityID
	}
	if e.EntityStatus == "" {
		e.EntityStatus = domain.EntityStatusActive
	}
	if modifiedAtStr != nil && *modifiedAtStr != "" {
		t, err := parseDatetime(*modifiedAtStr)
		if err != nil {
			return nil, fmt.Errorf("parse entity_modified_at: %w", err)
		}
		e.EntityModifiedAt = &t
	}
	if deletedAtStr != nil && *deletedAtStr != "" {
		t, err := parseDatetime(*deletedAtStr)
		if err != nil {
			return nil, fmt.Errorf("parse entity_deleted_at: %w", err)
		}
		e.EntityDeletedAt = &t
	}
	e.OccurredAt = occurredAt
	return &e, nil
}

// entityStatusOrActive normalizes an empty lifecycle status to active.
func entityStatusOrActive(s string) string {
	if s == "" {
		return domain.EntityStatusActive
	}
	return s
}
