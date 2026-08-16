package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- fakes ----

type fakeImportRepo struct {
	batches  []*domain.ImportBatch
	entries  map[string][]*domain.ImportEntry
	deleted  []string // entityType:entityID removed during rollback
	wallets  []string // wallet IDs deleted
	revoked  []string // batch IDs marked rolled back
	walletTx int
	walletTr int
}

func (f *fakeImportRepo) SaveBatch(ctx context.Context, b *domain.ImportBatch) error {
	f.batches = append(f.batches, b)
	return nil
}

func (f *fakeImportRepo) SaveEntries(ctx context.Context, es []*domain.ImportEntry) error {
	f.entries[batchID(es)] = append(f.entries[batchID(es)], es...)
	return nil
}

func (f *fakeImportRepo) FindByFingerprint(ctx context.Context, email, fp string) (*domain.ImportBatch, error) {
	for _, b := range f.batches {
		if b.UserEmail == email && b.FileFingerprint == fp {
			return b, nil
		}
	}
	return nil, nil
}

func (f *fakeImportRepo) FindBatchByID(ctx context.Context, id, email string) (*domain.ImportBatch, error) {
	for _, b := range f.batches {
		if b.ID == id && b.UserEmail == email {
			return b, nil
		}
	}
	return nil, errors.New("import batch not found")
}

func (f *fakeImportRepo) ListByUser(ctx context.Context, email string, limit, offset int) ([]*domain.ImportBatch, error) {
	return f.batches, nil
}

func (f *fakeImportRepo) ListEntries(ctx context.Context, importID string) ([]*domain.ImportEntry, error) {
	for id, es := range f.entries {
		if id == importID {
			return es, nil
		}
	}
	return nil, nil
}

func (f *fakeImportRepo) MarkRolledBack(ctx context.Context, id string, t time.Time) error {
	f.revoked = append(f.revoked, id)
	for _, b := range f.batches {
		if b.ID == id {
			b.Status = domain.ImportStatusRolledBack
			b.RolledBackAt = &t
		}
	}
	return nil
}

func (f *fakeImportRepo) MarkTransactionProvenance(ctx context.Context, txID, status string, at time.Time) error {
	return nil
}

func (f *fakeImportRepo) DeleteTransactionEntity(ctx context.Context, entityType, entityID, email string) (bool, error) {
	f.deleted = append(f.deleted, entityType+":"+entityID)
	return true, nil
}

func (f *fakeImportRepo) DeleteWallet(ctx context.Context, id, email string) error {
	f.wallets = append(f.wallets, id)
	return nil
}

func (f *fakeImportRepo) CountTransactionsForWallet(ctx context.Context, walletID string) (int, error) {
	return f.walletTx, nil
}

func (f *fakeImportRepo) CountTransfersForWallet(ctx context.Context, walletID string) (int, error) {
	return f.walletTr, nil
}

func batchID(es []*domain.ImportEntry) string {
	if len(es) == 0 {
		return ""
	}
	return es[0].ImportID
}

type fakeTxRepo struct {
	saved []*domain.Transaction
}

func (f *fakeTxRepo) Save(ctx context.Context, tx *domain.Transaction) error {
	f.saved = append(f.saved, tx)
	return nil
}

func (f *fakeTxRepo) FindByID(ctx context.Context, id string) (*domain.Transaction, error) {
	for _, t := range f.saved {
		if t.ID == id {
			return t, nil
		}
	}
	return nil, errors.New("transaction not found")
}

func (f *fakeTxRepo) FindByIdempotencyKey(ctx context.Context, email, key string) (*domain.Transaction, error) {
	for _, t := range f.saved {
		if t.UserEmail == email && t.IdempotencyKey == key {
			return t, nil
		}
	}
	return nil, nil
}

func (f *fakeTxRepo) ListByUserAndDateRange(ctx context.Context, email string, from, to time.Time, limit, offset int) ([]*domain.Transaction, error) {
	return f.saved, nil
}

func (f *fakeTxRepo) Delete(ctx context.Context, id, email string) error { return nil }
func (f *fakeTxRepo) DeleteAtRevision(ctx context.Context, id, email string, rev int) error {
	return nil
}
func (f *fakeTxRepo) Update(ctx context.Context, tx *domain.Transaction, rev int) error {
	return nil
}

func (f *fakeTxRepo) GetTodayTotals(ctx context.Context, email string, date time.Time) ([]domain.CurrencyTotal, error) {
	return nil, nil
}

type fakeTransferRepo struct {
	saved []*domain.WalletTransfer
}

func (f *fakeTransferRepo) Save(ctx context.Context, tr *domain.WalletTransfer) error {
	f.saved = append(f.saved, tr)
	return nil
}

func (f *fakeTransferRepo) FindByID(ctx context.Context, id string) (*domain.WalletTransfer, error) {
	return nil, nil
}

func (f *fakeTransferRepo) FindByIdempotencyKey(ctx context.Context, email, key string) (*domain.WalletTransfer, error) {
	for _, t := range f.saved {
		if t.UserEmail == email && t.IdempotencyKey == key {
			return t, nil
		}
	}
	return nil, nil
}

func (f *fakeTransferRepo) ListByUser(ctx context.Context, email string, limit, offset int) ([]*domain.WalletTransfer, error) {
	return f.saved, nil
}

// fakeCoordinator runs fn inline; the repo fakes are in-memory so atomicity is
// not observable here (covered by the integration test).
type fakeCoordinator struct{}

func (fakeCoordinator) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

// ---- helpers ----

func newImportService() (*ImportService, *fakeImportRepo, *fakeTxRepo, *fakeTransferRepo, *mockWalletRepo) {
	importRepo := &fakeImportRepo{entries: map[string][]*domain.ImportEntry{}}
	txRepo := &fakeTxRepo{}
	transferRepo := &fakeTransferRepo{}
	walletRepo := &mockWalletRepo{}
	svc := NewImportService(importRepo, txRepo, transferRepo, walletRepo, fakeCoordinator{})
	return svc, importRepo, txRepo, transferRepo, walletRepo
}

func gcashWallet(id string, currency string) *domain.Wallet {
	w, err := domain.NewWallet(id, "you@example.com", "GCash", domain.WalletEwallet, currency, 0, false)
	if err != nil {
		panic(err)
	}
	return w
}

func sampleRows() []ImportRowInput {
	return []ImportRowInput{
		{
			SourceReference: "REF000000001",
			OccurredAt:      time.Date(2026, 7, 1, 9, 30, 0, 0, time.UTC),
			AmountCents:     150000,
			Kind:            domain.EntryExpense,
			Category:        "Food",
			Description:     "Jollibee",
		},
		{
			SourceReference: "REF000000002",
			OccurredAt:      time.Date(2026, 7, 2, 18, 0, 0, 0, time.UTC),
			AmountCents:     500000,
			Kind:            domain.EntryIncome,
			Category:        "Salary",
			Description:     "Payroll",
		},
	}
}

func baseImportInput() CreateImportInput {
	return CreateImportInput{
		Provider:            domain.ImportProviderGCashPDF,
		FileFingerprint:     "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		StatementFrom:       time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		StatementTo:         time.Date(2026, 7, 31, 0, 0, 0, 0, time.UTC),
		WalletID:            "wallet-1",
		OpeningBalanceCents: 10000,
		EndingBalanceCents:  360000,
		Reconciliation:      domain.ReconciliationOK,
		Rows:                sampleRows(),
	}
}

// ---- tests ----

func TestImportCreateBooksTransactions(t *testing.T) {
	svc, importRepo, txRepo, _, walletRepo := newImportService()
	walletRepo.wallets = []*domain.Wallet{gcashWallet("wallet-1", "PHP")}

	batch, err := svc.Create(context.Background(), "you@example.com", baseImportInput())
	require.NoError(t, err)
	require.Equal(t, domain.ImportStatusCompleted, batch.Status)
	require.Equal(t, "wallet-1", batch.WalletID)

	require.Len(t, txRepo.saved, 2)
	expense := txRepo.saved[0]
	income := txRepo.saved[1]
	assert.Equal(t, domain.TransactionExpense, expense.Type)
	assert.Equal(t, "PHP", expense.Currency)
	assert.Equal(t, "wallet-1", expense.WalletID)
	assert.Equal(t, int64(150000), expense.AmountCents)
	assert.Equal(t, "imp:"+batch.ID+":REF000000001", expense.IdempotencyKey)
	assert.Equal(t, domain.TransactionIncome, income.Type)

	assert.Equal(t, 2, batch.Summary.Transactions)
	assert.Equal(t, int64(150000), batch.Summary.ExpenseCents)
	assert.Equal(t, int64(500000), batch.Summary.IncomeCents)
	require.Len(t, importRepo.batches, 1)
}

func TestImportCreateBooksTransfersWithDirection(t *testing.T) {
	svc, _, _, transferRepo, walletRepo := newImportService()
	walletRepo.wallets = []*domain.Wallet{
		gcashWallet("gcash-1", "PHP"),
		gcashWallet("bank-1", "PHP"),
	}

	input := baseImportInput()
	input.WalletID = "gcash-1"
	input.Rows = []ImportRowInput{
		{
			SourceReference: "REF000000010",
			OccurredAt:      time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
			AmountCents:     100000,
			Kind:            domain.EntryTransferOut,
			Category:        "Transfer",
			Description:     "BDO transfer",
			CounterWalletID: "bank-1",
		},
		{
			SourceReference: "REF000000011",
			OccurredAt:      time.Date(2026, 7, 4, 12, 0, 0, 0, time.UTC),
			AmountCents:     50000,
			Kind:            domain.EntryTransferIn,
			Category:        "Transfer",
			Description:     "BDO top-up",
			CounterWalletID: "bank-1",
		},
	}

	batch, err := svc.Create(context.Background(), "you@example.com", input)
	require.NoError(t, err)
	require.Len(t, transferRepo.saved, 2)

	out := transferRepo.saved[0]
	assert.Equal(t, "gcash-1", out.FromWalletID)
	assert.Equal(t, "bank-1", out.ToWalletID)

	in := transferRepo.saved[1]
	assert.Equal(t, "bank-1", in.FromWalletID)
	assert.Equal(t, "gcash-1", in.ToWalletID)

	assert.Equal(t, 2, batch.Summary.Transfers)
	assert.Zero(t, batch.Summary.Transactions)
}

func TestImportReplaySameFingerprint(t *testing.T) {
	svc, importRepo, txRepo, _, walletRepo := newImportService()
	walletRepo.wallets = []*domain.Wallet{gcashWallet("wallet-1", "PHP")}

	first, err := svc.Create(context.Background(), "you@example.com", baseImportInput())
	require.NoError(t, err)

	second, err := svc.Create(context.Background(), "you@example.com", baseImportInput())
	require.NoError(t, err)

	assert.Equal(t, first.ID, second.ID)
	assert.True(t, second.Summary.Replay)
	// No duplicate rows: batch row count unchanged, no new transactions.
	assert.Len(t, txRepo.saved, 2)
	assert.Len(t, importRepo.batches, 1)
}

func TestImportRejectsWalletCurrencyMismatch(t *testing.T) {
	svc, _, _, _, walletRepo := newImportService()
	walletRepo.wallets = []*domain.Wallet{gcashWallet("wallet-1", "USD")}

	_, err := svc.Create(context.Background(), "you@example.com", baseImportInput())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "currency")
}

func TestImportCreatesWalletWhenRequested(t *testing.T) {
	svc, importRepo, txRepo, _, walletRepo := newImportService()
	walletRepo.wallets = nil // no wallets yet -> new one becomes default

	input := baseImportInput()
	input.WalletID = ""
	input.CreateWallet = &CreateWalletForImport{Name: "GCash", OpeningBalanceCents: 10000}

	batch, err := svc.Create(context.Background(), "you@example.com", input)
	require.NoError(t, err)

	assert.NotEmpty(t, batch.CreatedWalletID)
	assert.NotEmpty(t, batch.WalletID)
	assert.Equal(t, batch.CreatedWalletID, batch.WalletID)
	require.Len(t, txRepo.saved, 2)
	assert.Equal(t, "PHP", txRepo.saved[0].Currency)
	require.Len(t, importRepo.batches, 1)
}

func TestImportDuplicateReferenceWithinPayload(t *testing.T) {
	svc, _, txRepo, _, walletRepo := newImportService()
	walletRepo.wallets = []*domain.Wallet{gcashWallet("wallet-1", "PHP")}

	input := baseImportInput()
	dup := input.Rows[0]
	dup.SourceReference = input.Rows[0].SourceReference // same ref, second row
	input.Rows = append(input.Rows, dup)

	batch, err := svc.Create(context.Background(), "you@example.com", input)
	require.NoError(t, err)
	assert.Equal(t, 1, batch.Summary.Duplicates)
	// Only the first occurrence was booked.
	assert.Len(t, txRepo.saved, 2)
}

func TestImportRejectsTooManyRows(t *testing.T) {
	svc, _, _, _, walletRepo := newImportService()
	walletRepo.wallets = []*domain.Wallet{gcashWallet("wallet-1", "PHP")}

	input := baseImportInput()
	for i := 0; i < domain.MaxImportRows+1; i++ {
		input.Rows = append(input.Rows, ImportRowInput{
			SourceReference: "REF" + string(rune('0'+i%10)),
			OccurredAt:      time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
			AmountCents:     100,
			Kind:            domain.EntryExpense,
			Category:        "Other",
		})
	}

	_, err := svc.Create(context.Background(), "you@example.com", input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many rows")
}

func TestRollbackRemovesEntitiesAndCreatedWallet(t *testing.T) {
	svc, importRepo, _, _, walletRepo := newImportService()
	walletRepo.wallets = nil
	input := baseImportInput()
	input.WalletID = ""
	input.CreateWallet = &CreateWalletForImport{Name: "GCash"}

	batch, err := svc.Create(context.Background(), "you@example.com", input)
	require.NoError(t, err)

	// Created wallet has no other references.
	importRepo.walletTx = 0
	importRepo.walletTr = 0
	removed, err := svc.Rollback(context.Background(), batch.ID, "you@example.com")
	require.NoError(t, err)
	assert.Equal(t, 2, removed)
	assert.Len(t, importRepo.deleted, 2)
	assert.Equal(t, []string{batch.CreatedWalletID}, importRepo.wallets)
	assert.Equal(t, []string{batch.ID}, importRepo.revoked)

	// Idempotent: second rollback is a no-op.
	removed, err = svc.Rollback(context.Background(), batch.ID, "you@example.com")
	require.NoError(t, err)
	assert.Zero(t, removed)
}

func TestRollbackKeepsReferencedWallet(t *testing.T) {
	svc, importRepo, _, _, walletRepo := newImportService()
	walletRepo.wallets = []*domain.Wallet{gcashWallet("wallet-1", "PHP")}

	batch, err := svc.Create(context.Background(), "you@example.com", baseImportInput())
	require.NoError(t, err)
	require.Empty(t, batch.CreatedWalletID) // wallet pre-existed

	removed, err := svc.Rollback(context.Background(), batch.ID, "you@example.com")
	require.NoError(t, err)
	assert.Equal(t, 2, removed)
	assert.Empty(t, importRepo.wallets)
}

func TestImportRejectsTransferToSameWallet(t *testing.T) {
	svc, _, _, _, walletRepo := newImportService()
	walletRepo.wallets = []*domain.Wallet{gcashWallet("wallet-1", "PHP")}

	input := baseImportInput()
	input.Rows = []ImportRowInput{{
		SourceReference: "REF000000020",
		OccurredAt:      time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC),
		AmountCents:     1000,
		Kind:            domain.EntryTransferOut,
		Category:        "Transfer",
		CounterWalletID: "wallet-1",
	}}

	_, err := svc.Create(context.Background(), "you@example.com", input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "different counter wallet")
}

func transferRowInput(ref, counterID string) ImportRowInput {
	return ImportRowInput{
		SourceReference: ref,
		OccurredAt:      time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC),
		AmountCents:     1000,
		Kind:            domain.EntryTransferOut,
		Category:        "Transfer",
		CounterWalletID: counterID,
	}
}

func TestImportRejectsTransferToNonexistentCounterWallet(t *testing.T) {
	svc, _, _, _, walletRepo := newImportService()
	walletRepo.wallets = []*domain.Wallet{gcashWallet("wallet-1", "PHP")}

	input := baseImportInput()
	input.Rows = []ImportRowInput{transferRowInput("REF000000030", "ghost-wallet")}

	_, err := svc.Create(context.Background(), "you@example.com", input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "REF000000030")
	assert.Contains(t, err.Error(), "counter wallet")
	assert.Contains(t, err.Error(), "not found")
}

func TestImportRejectsTransferToForeignCounterWallet(t *testing.T) {
	svc, _, _, _, walletRepo := newImportService()
	foreign, err := domain.NewWallet("foreign-1", "other@example.com", "BDO", domain.WalletEwallet, "PHP", 0, false)
	require.NoError(t, err)
	walletRepo.wallets = []*domain.Wallet{gcashWallet("wallet-1", "PHP"), foreign}

	input := baseImportInput()
	input.Rows = []ImportRowInput{transferRowInput("REF000000031", "foreign-1")}

	_, err = svc.Create(context.Background(), "you@example.com", input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "counter wallet")
	assert.Contains(t, err.Error(), "not found")
}

func TestImportRejectsTransferToArchivedCounterWallet(t *testing.T) {
	svc, _, _, _, walletRepo := newImportService()
	archived := gcashWallet("archived-1", "PHP")
	now := time.Now()
	archived.ArchivedAt = &now
	walletRepo.wallets = []*domain.Wallet{gcashWallet("wallet-1", "PHP"), archived}

	input := baseImportInput()
	input.Rows = []ImportRowInput{transferRowInput("REF000000032", "archived-1")}

	_, err := svc.Create(context.Background(), "you@example.com", input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "archived")
}

func TestImportRejectsTransferToForeignCurrencyCounterWallet(t *testing.T) {
	svc, _, _, _, walletRepo := newImportService()
	walletRepo.wallets = []*domain.Wallet{gcashWallet("wallet-1", "PHP"), gcashWallet("usd-1", "USD")}

	input := baseImportInput()
	input.Rows = []ImportRowInput{transferRowInput("REF000000033", "usd-1")}

	_, err := svc.Create(context.Background(), "you@example.com", input)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "currency")
}
