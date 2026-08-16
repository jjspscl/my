package infrastructure

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Edit/delete integration tests: revision concurrency, import provenance
// lifecycle, and bill payment link sources against the real schema.

func TestTransactionRepo_UpdateBumpsRevisionAndEnforcesPrecondition(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := NewTransactionRepoLibSQL(db)
	walletRepo := NewWalletRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	mustTransaction(t, repo, "tx-1", "PHP", "food", 1000, domain.TransactionExpense, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), "w-php")

	loaded, err := repo.FindByID(ctx, "tx-1")
	require.NoError(t, err)
	assert.Equal(t, 1, loaded.Revision)

	loaded.Category = "groceries"
	loaded.UpdatedAt = time.Now().UTC()
	require.NoError(t, repo.Update(ctx, loaded, 1))

	// Stale write must fail with the domain sentinel.
	stale := *loaded
	stale.Category = "other"
	err = repo.Update(ctx, &stale, 1)
	assert.ErrorIs(t, err, domain.ErrStaleRevision)

	// Current revision read back as 2.
	again, err := repo.FindByID(ctx, "tx-1")
	require.NoError(t, err)
	assert.Equal(t, 2, again.Revision)
	assert.Equal(t, "groceries", again.Category)
	assert.False(t, again.UpdatedAt.IsZero())
}

func TestTransactionRepo_DeleteAtRevisionPrecondition(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := NewTransactionRepoLibSQL(db)
	walletRepo := NewWalletRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	mustTransaction(t, repo, "tx-1", "PHP", "food", 1000, domain.TransactionExpense, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), "w-php")

	// Wrong revision: not deleted, stale error.
	err := repo.DeleteAtRevision(ctx, "tx-1", testUser, 5)
	assert.ErrorIs(t, err, domain.ErrStaleRevision)
	_, err = repo.FindByID(ctx, "tx-1")
	require.NoError(t, err)

	// Correct revision: deleted.
	require.NoError(t, repo.DeleteAtRevision(ctx, "tx-1", testUser, 1))
	_, err = repo.FindByID(ctx, "tx-1")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transaction not found")
}

func TestTransactionRepo_ImportedFlagFromProvenance(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := NewTransactionRepoLibSQL(db)
	walletRepo := NewWalletRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	mustTransaction(t, repo, "tx-imported", "PHP", "food", 1000, domain.TransactionExpense, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), "w-php")
	mustTransaction(t, repo, "tx-manual", "PHP", "food", 2000, domain.TransactionExpense, time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC), "w-php")

	seedImportEntry(t, db, "imp-1", "tx-imported")

	imported, err := repo.FindByID(ctx, "tx-imported")
	require.NoError(t, err)
	assert.True(t, imported.Imported)
	assert.Equal(t, domain.ImportProviderGCashPDF, imported.ImportProvider)

	manual, err := repo.FindByID(ctx, "tx-manual")
	require.NoError(t, err)
	assert.False(t, manual.Imported)
}

func TestImportRepo_MarkTransactionProvenanceUpdatesEntryAndBatch(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := NewImportRepoLibSQL(db)

	seedImportEntry(t, db, "imp-1", "tx-1")
	at := time.Date(2026, 8, 16, 10, 0, 0, 0, time.UTC)

	require.NoError(t, repo.MarkTransactionProvenance(ctx, "tx-1", domain.EntityStatusModified, at))

	batch, err := repo.FindBatchByID(ctx, "imp-1", testUser)
	require.NoError(t, err)
	assert.Equal(t, domain.BatchIntegrityModified, batch.Integrity)

	entries, err := repo.ListEntries(ctx, "imp-1")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, domain.EntityStatusModified, entries[0].EntityStatus)
	require.NotNil(t, entries[0].EntityModifiedAt)
	assert.Equal(t, at.Truncate(time.Second), entries[0].EntityModifiedAt.Truncate(time.Second))
	assert.Nil(t, entries[0].EntityDeletedAt)
}

func TestImportRepo_MarkTransactionProvenance_NoopForManualTransaction(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	repo := NewImportRepoLibSQL(db)

	require.NoError(t, repo.MarkTransactionProvenance(ctx, "tx-ghost", domain.EntityStatusDeleted, time.Now().UTC()))
}

func TestBillRepo_PaymentLinkSourceRoundTripAndDelete(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	billRepo := NewBillRepoLibSQL(db)
	txRepo := NewTransactionRepoLibSQL(db)
	walletRepo := NewWalletRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	mustTransaction(t, txRepo, "tx-1", "PHP", "Subscription", 49900, domain.TransactionExpense, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), "w-php")

	bill, err := domain.NewRecurringBill("b-1", testUser, "Netflix", "Subscription", 49900, "PHP", domain.FrequencyMonthly, 1, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil, true, nil)
	require.NoError(t, err)
	require.NoError(t, billRepo.SaveBill(ctx, bill))

	payment, err := domain.NewBillPayment("p-1", bill.ID, time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), 49900)
	require.NoError(t, err)
	txID := "tx-1"
	payment.TransactionID = &txID
	payment.TransactionLinkSource = domain.PaymentLinkAuto
	payment.Status = domain.OccurrencePaid
	paidAt := time.Now().UTC()
	payment.PaidDate = &paidAt
	require.NoError(t, billRepo.SavePayment(ctx, payment))

	found, err := billRepo.FindPayment(ctx, bill.ID, "2026-08-01")
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentLinkAuto, found.TransactionLinkSource)

	byTx, err := billRepo.FindPaymentsByTransaction(ctx, "tx-1")
	require.NoError(t, err)
	require.Len(t, byTx, 1)
	assert.Equal(t, "p-1", byTx[0].ID)
	assert.Equal(t, domain.PaymentLinkAuto, byTx[0].TransactionLinkSource)

	// Missing link source defaults to legacy (old rows).
	legacy, err := domain.NewBillPayment("p-2", bill.ID, time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC), 49900)
	require.NoError(t, err)
	legacy.Status = domain.OccurrencePaid
	require.NoError(t, billRepo.SavePayment(ctx, legacy))
	gotLegacy, err := billRepo.FindPayment(ctx, bill.ID, "2026-09-01")
	require.NoError(t, err)
	assert.Equal(t, domain.PaymentLinkLegacy, gotLegacy.TransactionLinkSource)

	require.NoError(t, billRepo.DeletePayment(ctx, "p-1"))
	byTx, err = billRepo.FindPaymentsByTransaction(ctx, "tx-1")
	require.NoError(t, err)
	assert.Empty(t, byTx)
}

// seedImportEntry inserts a minimal import batch + entry row so provenance
// lookups hit real rows.
func seedImportEntry(t *testing.T, db *sql.DB, importID, txID string) {
	t.Helper()
	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `
		INSERT INTO finance_imports
			(id, user_email, provider, file_fingerprint, statement_from, statement_to,
			 opening_balance_cents, ending_balance_cents, reconciliation,
			 status, summary_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, 0, 0, 'ok', 'completed', '{}', datetime('now'))
	`, importID, testUser, domain.ImportProviderGCashPDF,
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"2026-08-01", "2026-08-31"); err != nil {
		t.Fatalf("seed import batch: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		INSERT INTO finance_import_entries
			(id, import_id, source_reference, occurred_at, amount_cents, kind,
			 category, description, outcome, entity_type, entity_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'imported', 'transaction', ?)
	`, importID+"-e1", importID, "REF001", "2026-08-01T09:00:00Z", 1000,
		"expense", "food", "test", txID); err != nil {
		t.Fatalf("seed import entry: %v", err)
	}
}
