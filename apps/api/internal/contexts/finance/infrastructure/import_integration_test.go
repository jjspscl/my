package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/application"
	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var importWide = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// countTransactions and countTransfers are thin wrappers over the real repo
// list methods used by assertions.
func countTransactions(t *testing.T, repo *TransactionRepoLibSQL, ctx context.Context) int {
	t.Helper()
	rows, err := repo.ListByUserAndDateRange(ctx, testUser, importWide, time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), 100, 0)
	require.NoError(t, err)
	return len(rows)
}

func countTransfers(t *testing.T, repo *TransferRepoLibSQL, ctx context.Context) int {
	t.Helper()
	rows, err := repo.ListByUser(ctx, testUser, 100, 0)
	require.NoError(t, err)
	return len(rows)
}

func newImportService(t *testing.T) (*application.ImportService, *ImportRepoLibSQL, *TransactionRepoLibSQL, *TransferRepoLibSQL, *WalletRepoLibSQL, *Coordinator) {
	t.Helper()
	db := newTestDB(t)
	importRepo := NewImportRepoLibSQL(db)
	txRepo := NewTransactionRepoLibSQL(db)
	transferRepo := NewTransferRepoLibSQL(db)
	walletRepo := NewWalletRepoLibSQL(db)
	coordinator := NewCoordinator(db)
	svc := application.NewImportService(importRepo, txRepo, transferRepo, walletRepo, coordinator)
	return svc, importRepo, txRepo, transferRepo, walletRepo, coordinator
}

func importRow(ref string, day int, amount int64, kind, category string) application.ImportRowInput {
	return application.ImportRowInput{
		SourceReference: ref,
		OccurredAt:      time.Date(2026, 7, day, 10, 0, 0, 0, time.UTC),
		AmountCents:     amount,
		Kind:            kind,
		Category:        category,
		Description:     "desc " + ref,
	}
}

func importInput(walletID string, rows ...application.ImportRowInput) application.CreateImportInput {
	return application.CreateImportInput{
		Provider:            domain.ImportProviderGCashPDF,
		FileFingerprint:     "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		StatementFrom:       time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		StatementTo:         time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		WalletID:            walletID,
		OpeningBalanceCents: 10000,
		EndingBalanceCents:  20000,
		Reconciliation:      domain.ReconciliationOK,
		Rows:                rows,
	}
}

func TestImportServiceEndToEnd(t *testing.T) {
	svc, importRepo, txRepo, transferRepo, walletRepo, _ := newImportService(t)
	ctx := context.Background()
	

	// Pre-existing PHP wallet + a counter wallet for transfers.
	mustWallet(t, walletRepo, "gcash-1", "PHP", 10000)
	mustWallet(t, walletRepo, "bank-1", "PHP", 0)

	rows := []application.ImportRowInput{
		importRow("R100000000001", 1, 150000, domain.EntryExpense, "Food"),
		importRow("R100000000002", 2, 500000, domain.EntryIncome, "Salary"),
		{SourceReference: "R100000000003", OccurredAt: time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
			AmountCents: 100000, Kind: domain.EntryTransferOut, Category: "Transfer",
			Description: "BDO", CounterWalletID: "bank-1"},
	}

	batch, err := svc.Create(ctx, testUser, importInput("gcash-1", rows...))
	require.NoError(t, err)
	require.Equal(t, "gcash-1", batch.WalletID)

	// Two transactions and one transfer landed.
	assert.Equal(t, 2, countTransactions(t, txRepo, ctx))
	assert.Equal(t, 1, countTransfers(t, transferRepo, ctx))

	// Entries round-trip with entity links.
	entries, err := importRepo.ListEntries(ctx, batch.ID)
	require.NoError(t, err)
	require.Len(t, entries, 3)
	linked := 0
	for _, e := range entries {
		if e.EntityID != "" {
			linked++
		}
	}
	assert.Equal(t, 3, linked)

	// Replay returns the same batch without new rows.
	replayed, err := svc.Create(ctx, testUser, importInput("gcash-1", rows...))
	require.NoError(t, err)
	assert.Equal(t, batch.ID, replayed.ID)
	assert.True(t, replayed.Summary.Replay)
	assert.Equal(t, 2, countTransactions(t, txRepo, ctx))
}

func TestImportRollbackDeletesEntitiesAndWallet(t *testing.T) {
	svc, importRepo, txRepo, transferRepo, walletRepo, _ := newImportService(t)
	ctx := context.Background()

	// No wallets yet: the import creates the GCash wallet and it must be
	// removed on rollback when nothing else references it.
	input := importInput("", importRow("R200000000001", 1, 5000, domain.EntryExpense, "Food"))
	input.WalletID = ""
	input.CreateWallet = &application.CreateWalletForImport{Name: "GCash", OpeningBalanceCents: 10000}

	batch, err := svc.Create(ctx, testUser, input)
	require.NoError(t, err)
	require.NotEmpty(t, batch.CreatedWalletID)

	removed, err := svc.Rollback(ctx, batch.ID, testUser)
	require.NoError(t, err)
	assert.Equal(t, 1, removed)

	assert.Equal(t, 0, countTransactions(t, txRepo, ctx))
	assert.Equal(t, 0, countTransfers(t, transferRepo, ctx))

	rolledBack, err := importRepo.FindBatchByID(ctx, batch.ID, testUser)
	require.NoError(t, err)
	assert.Equal(t, domain.ImportStatusRolledBack, rolledBack.Status)
	assert.NotNil(t, rolledBack.RolledBackAt)

	// Wallet was deleted (no longer findable).
	_, err = walletRepo.FindByID(ctx, batch.CreatedWalletID)
	assert.Error(t, err)
}

func TestImportRollbackKeepsPreExistingWallet(t *testing.T) {
	svc, _, txRepo, _, walletRepo, _ := newImportService(t)
	ctx := context.Background()
	
	mustWallet(t, walletRepo, "gcash-1", "PHP", 0)

	batch, err := svc.Create(ctx, testUser, importInput("gcash-1", importRow("R300000000001", 1, 5000, domain.EntryExpense, "Food")))
	require.NoError(t, err)
	require.Empty(t, batch.CreatedWalletID)

	_, err = svc.Rollback(ctx, batch.ID, testUser)
	require.NoError(t, err)

	assert.Equal(t, 0, countTransactions(t, txRepo, ctx))
	// Pre-existing wallet survives.
	w, err := walletRepo.FindByID(ctx, "gcash-1")
	require.NoError(t, err)
	assert.Equal(t, "gcash-1", w.ID)
}

// TestImportRollbackOnFailure proves atomicity: when one row violates a
// constraint mid-insert, nothing from the batch persists.
func TestImportRollbackOnFailure(t *testing.T) {
	svc, importRepo, txRepo, _, walletRepo, _ := newImportService(t)
	ctx := context.Background()
	mustWallet(t, walletRepo, "gcash-1", "PHP", 0)

	// Two rows sharing a reference: the unique (import_id, source_reference)
	// constraint fails on the second insert, forcing the whole batch to roll
	// back — the wallet creation and first transaction must vanish too.
	rows := []application.ImportRowInput{
		importRow("R400000000001", 1, 5000, domain.EntryExpense, "Food"),
		importRow("R400000000001", 2, 7000, domain.EntryExpense, "Food"),
	}
	input := importInput("gcash-1", rows...)
	input.CreateWallet = &application.CreateWalletForImport{Name: "GCash"}

	_, err := svc.Create(ctx, testUser, input)
	require.Error(t, err)

	assert.Equal(t, 0, countTransactions(t, txRepo, ctx))
	batches, err := importRepo.ListByUser(ctx, testUser, 10, 0)
	require.NoError(t, err)
	assert.Empty(t, batches)
}
