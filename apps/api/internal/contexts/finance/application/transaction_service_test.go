package application

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- mocks ----

type mockWalletRepo struct {
	wallets  []*domain.Wallet
	balances []*domain.WalletBalance
}

func (m *mockWalletRepo) Save(ctx context.Context, wallet *domain.Wallet) error {
	return nil
}

func (m *mockWalletRepo) FindByID(ctx context.Context, id string) (*domain.Wallet, error) {
	for _, w := range m.wallets {
		if w.ID == id {
			return w, nil
		}
	}
	return nil, errors.New("wallet not found")
}

func (m *mockWalletRepo) ListByUser(ctx context.Context, userEmail string) ([]*domain.Wallet, error) {
	return m.wallets, nil
}

func (m *mockWalletRepo) Update(ctx context.Context, wallet *domain.Wallet) error {
	return nil
}

func (m *mockWalletRepo) Archive(ctx context.Context, id, userEmail string) error {
	return nil
}

func (m *mockWalletRepo) FindDefault(ctx context.Context, userEmail string) (*domain.Wallet, error) {
	for _, w := range m.wallets {
		if w.UserEmail == userEmail && w.IsDefault {
			return w, nil
		}
	}
	if len(m.wallets) > 0 {
		return m.wallets[0], nil
	}
	return nil, errors.New("no wallets found")
}

func (m *mockWalletRepo) GetBalancesByUser(ctx context.Context, userEmail string) ([]*domain.WalletBalance, error) {
	return m.balances, nil
}

type mockTransactionRepo struct {
	transactions  []*domain.Transaction
	saveFn        func(ctx context.Context, tx *domain.Transaction) error
	findByIDFn    func(ctx context.Context, id string) (*domain.Transaction, error)
	listFn        func(ctx context.Context, userEmail string, from, to time.Time, limit, offset int) ([]*domain.Transaction, error)
	deleteFn      func(ctx context.Context, id, userEmail string) error
	updateFn      func(ctx context.Context, tx *domain.Transaction, expectedRevision int) error
	todayTotalsFn func(ctx context.Context, userEmail string, date time.Time) ([]domain.CurrencyTotal, error)
}

func (m *mockTransactionRepo) Save(ctx context.Context, tx *domain.Transaction) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, tx)
	}
	m.transactions = append(m.transactions, tx)
	return nil
}

func (m *mockTransactionRepo) FindByID(ctx context.Context, id string) (*domain.Transaction, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	for _, tx := range m.transactions {
		if tx.ID == id {
			return tx, nil
		}
	}
	return nil, errors.New("transaction not found")
}

func (m *mockTransactionRepo) FindByIdempotencyKey(ctx context.Context, userEmail, key string) (*domain.Transaction, error) {
	for _, tx := range m.transactions {
		if tx.UserEmail == userEmail && tx.IdempotencyKey == key {
			return tx, nil
		}
	}
	return nil, nil
}

func (m *mockTransactionRepo) ListByUserAndDateRange(ctx context.Context, userEmail string, from, to time.Time, limit, offset int) ([]*domain.Transaction, error) {
	if m.listFn != nil {
		return m.listFn(ctx, userEmail, from, to, limit, offset)
	}
	var result []*domain.Transaction
	for _, tx := range m.transactions {
		if tx.UserEmail == userEmail && !tx.TransactionDate.Before(from) && !tx.TransactionDate.After(to) {
			result = append(result, tx)
		}
	}
	// Apply limit
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (m *mockTransactionRepo) Update(ctx context.Context, tx *domain.Transaction, expectedRevision int) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, tx, expectedRevision)
	}
	for i, stored := range m.transactions {
		if stored.ID == tx.ID && stored.UserEmail == tx.UserEmail {
			if stored.Revision != expectedRevision {
				return domain.ErrStaleRevision
			}
			// The real repo bumps the stored revision in SQL and leaves the
			// caller's pointer at the old value; mirror that so the service's
			// own increment produces the correct new revision.
			next := *tx
			next.Revision = stored.Revision + 1
			m.transactions[i] = &next
			return nil
		}
	}
	return errors.New("transaction not found")
}

func (m *mockTransactionRepo) DeleteAtRevision(ctx context.Context, id, userEmail string, expectedRevision int) error {
	for i, tx := range m.transactions {
		if tx.ID == id && tx.UserEmail == userEmail {
			if tx.Revision != expectedRevision {
				return domain.ErrStaleRevision
			}
			m.transactions = append(m.transactions[:i], m.transactions[i+1:]...)
			return nil
		}
	}
	return errors.New("transaction not found")
}

func (m *mockTransactionRepo) Delete(ctx context.Context, id, userEmail string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id, userEmail)
	}
	for i, tx := range m.transactions {
		if tx.ID == id && tx.UserEmail == userEmail {
			m.transactions = append(m.transactions[:i], m.transactions[i+1:]...)
			return nil
		}
	}
	return errors.New("transaction not found")
}

func (m *mockTransactionRepo) GetTodayTotals(ctx context.Context, userEmail string, date time.Time) ([]domain.CurrencyTotal, error) {
	if m.todayTotalsFn != nil {
		return m.todayTotalsFn(ctx, userEmail, date)
	}
	var expenseCents, incomeCents int64
	dateStr := date.Format("2006-01-02")
	for _, tx := range m.transactions {
		if tx.UserEmail == userEmail && tx.TransactionDate.Format("2006-01-02") == dateStr {
			if tx.Type == domain.TransactionExpense {
				expenseCents += tx.AmountCents
			} else {
				incomeCents += tx.AmountCents
			}
		}
	}
	return []domain.CurrencyTotal{{
		Currency:     "PHP",
		TotalCents:   incomeCents - expenseCents,
		ExpenseCents: expenseCents,
		IncomeCents:  incomeCents,
	}}, nil
}

func newTestTransactionService() *TransactionService {
	repo := &mockTransactionRepo{}
	walletRepo := &mockWalletRepo{
		wallets: []*domain.Wallet{
			{ID: "w-default", UserEmail: "user@test.com", Name: "Cash", Currency: "PHP", IsDefault: true},
		},
	}
	return NewTransactionService(repo, walletRepo, timeutil.New(time.UTC))
}

// ---- tests ----

func TestCreate_ValidExpense_SavesAndReturns(t *testing.T) {
	svc := newTestTransactionService()
	ctx := context.Background()

	tx, err := svc.Create(ctx, "user@test.com", CreateTransactionInput{
		AmountCents:     150000,
		Category:        "food",
		Description:     "Lunch",
		Type:            domain.TransactionExpense,
		WalletID:        "w-default",
		TransactionDate: time.Now(),
	})
	require.NoError(t, err)
	assert.NotEmpty(t, tx.ID)
	assert.Equal(t, "user@test.com", tx.UserEmail)
	assert.Equal(t, int64(150000), tx.AmountCents)
	assert.Equal(t, domain.TransactionExpense, tx.Type)
	assert.Equal(t, "PHP", tx.Currency)
}

func TestCreate_ValidIncome_SavesAndReturns(t *testing.T) {
	svc := newTestTransactionService()
	ctx := context.Background()

	tx, err := svc.Create(ctx, "user@test.com", CreateTransactionInput{
		AmountCents:     5000000,
		Category:        "salary",
		Description:     "Monthly salary",
		Type:            domain.TransactionIncome,
		WalletID:        "w-default",
		TransactionDate: time.Now(),
	})
	require.NoError(t, err)
	assert.Equal(t, domain.TransactionIncome, tx.Type)
	assert.Equal(t, int64(5000000), tx.AmountCents)
}

func TestCreate_NegativeAmount_ReturnsError(t *testing.T) {
	svc := newTestTransactionService()
	ctx := context.Background()

	_, err := svc.Create(ctx, "user@test.com", CreateTransactionInput{
		AmountCents: -100,
		Category:    "food",
		Type:        domain.TransactionExpense,
		WalletID:    "w-default",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "positive")
}

func TestCreate_ZeroAmount_ReturnsError(t *testing.T) {
	svc := newTestTransactionService()
	ctx := context.Background()

	_, err := svc.Create(ctx, "user@test.com", CreateTransactionInput{
		AmountCents: 0,
		Category:    "food",
		Type:        domain.TransactionExpense,
		WalletID:    "w-default",
	})
	assert.Error(t, err)
}

func TestCreate_MissingCategory_ReturnsError(t *testing.T) {
	svc := newTestTransactionService()
	ctx := context.Background()

	_, err := svc.Create(ctx, "user@test.com", CreateTransactionInput{
		AmountCents: 100,
		Category:    "",
		Type:        domain.TransactionExpense,
		WalletID:    "w-default",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "category")
}

func TestCreate_InvalidType_ReturnsError(t *testing.T) {
	svc := newTestTransactionService()
	ctx := context.Background()

	_, err := svc.Create(ctx, "user@test.com", CreateTransactionInput{
		AmountCents: 100,
		Category:    "food",
		Type:        "invalid",
		WalletID:    "w-default",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "type")
}

func TestGetTodayTotal_NoTransactions_ReturnsZeros(t *testing.T) {
	svc := newTestTransactionService()
	ctx := context.Background()

	total, err := svc.GetTodayTotal(ctx, "user@test.com", "PHP")
	require.NoError(t, err)
	assert.Equal(t, int64(0), total.ExpenseCents)
	assert.Equal(t, int64(0), total.IncomeCents)
	assert.Equal(t, int64(0), total.TotalCents)
}

func TestGetTodayTotal_MixedTransactions_ReturnsCorrectAggregation(t *testing.T) {
	walletRepo := &mockWalletRepo{
		wallets: []*domain.Wallet{
			{ID: "w-default", UserEmail: "user@test.com", Name: "Cash", Currency: "PHP", IsDefault: true},
		},
	}
	repo := &mockTransactionRepo{}
	svc := NewTransactionService(repo, walletRepo, timeutil.New(time.UTC))
	ctx := context.Background()
	now := time.Now().UTC()

	// Add some transactions
	repo.transactions = []*domain.Transaction{
		{ID: "tx-1", UserEmail: "user@test.com", AmountCents: 100000, Type: domain.TransactionExpense, TransactionDate: now, WalletID: "w-default"},
		{ID: "tx-2", UserEmail: "user@test.com", AmountCents: 50000, Type: domain.TransactionExpense, TransactionDate: now, WalletID: "w-default"},
		{ID: "tx-3", UserEmail: "user@test.com", AmountCents: 5000000, Type: domain.TransactionIncome, TransactionDate: now, WalletID: "w-default"},
	}

	total, err := svc.GetTodayTotal(ctx, "user@test.com", "PHP")
	require.NoError(t, err)
	assert.Equal(t, int64(150000), total.ExpenseCents)
	assert.Equal(t, int64(5000000), total.IncomeCents)
	assert.Equal(t, int64(4850000), total.TotalCents)
}

func TestList_DefaultLimit(t *testing.T) {
	walletRepo := &mockWalletRepo{
		wallets: []*domain.Wallet{
			{ID: "w-default", UserEmail: "user@test.com", Name: "Cash", Currency: "PHP", IsDefault: true},
		},
	}
	repo := &mockTransactionRepo{}
	svc := NewTransactionService(repo, walletRepo, timeutil.New(time.UTC))
	ctx := context.Background()

	// Add 200 transactions
	for i := 0; i < 200; i++ {
		repo.transactions = append(repo.transactions, &domain.Transaction{
			ID:              "tx-" + string(rune(i)),
			UserEmail:       "user@test.com",
			TransactionDate: time.Now(),
			WalletID:        "w-default",
		})
	}

	txs, err := svc.List(ctx, "user@test.com", TransactionFilter{
		From: time.Now().Add(-24 * time.Hour),
		To:   time.Now().Add(24 * time.Hour),
	})
	require.NoError(t, err)
	assert.LessOrEqual(t, len(txs), 50)
}

func TestDelete_ExistingTransaction_Succeeds(t *testing.T) {
	walletRepo := &mockWalletRepo{
		wallets: []*domain.Wallet{
			{ID: "w-default", UserEmail: "user@test.com", Name: "Cash", Currency: "PHP", IsDefault: true},
		},
	}
	repo := &mockTransactionRepo{}
	svc := NewTransactionService(repo, walletRepo, timeutil.New(time.UTC))
	ctx := context.Background()

	repo.transactions = []*domain.Transaction{
		{ID: "tx-1", UserEmail: "user@test.com", WalletID: "w-default"},
	}

	err := svc.Delete(ctx, "tx-1", "user@test.com")
	assert.NoError(t, err)
}

func TestCreate_RepoFailure_ReturnsError(t *testing.T) {
	walletRepo := &mockWalletRepo{
		wallets: []*domain.Wallet{
			{ID: "w-default", UserEmail: "user@test.com", Name: "Cash", Currency: "PHP", IsDefault: true},
		},
	}
	repo := &mockTransactionRepo{
		saveFn: func(ctx context.Context, tx *domain.Transaction) error {
			return errors.New("database error")
		},
	}
	svc := NewTransactionService(repo, walletRepo, timeutil.New(time.UTC))
	ctx := context.Background()

	_, err := svc.Create(ctx, "user@test.com", CreateTransactionInput{
		AmountCents: 100,
		Category:    "food",
		Type:        domain.TransactionExpense,
		WalletID:    "w-default",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save transaction")
}

func TestGetTodayTotal_EmptyDateRange_ReturnsNilDefault(t *testing.T) {
	walletRepo := &mockWalletRepo{
		wallets: []*domain.Wallet{
			{ID: "w-default", UserEmail: "user@test.com", Name: "Cash", Currency: "PHP", IsDefault: true},
		},
	}
	repo := &mockTransactionRepo{
		todayTotalsFn: func(ctx context.Context, userEmail string, date time.Time) ([]domain.CurrencyTotal, error) {
			return nil, nil
		},
	}
	svc := NewTransactionService(repo, walletRepo, timeutil.New(time.UTC))
	ctx := context.Background()

	total, err := svc.GetTodayTotal(ctx, "user@test.com", "PHP")
	require.NoError(t, err)
	assert.NotNil(t, total)
	assert.Equal(t, int64(0), total.TotalCents)
	assert.Equal(t, "PHP", total.Currency)
}

// ---- edit/delete fakes ----

type fakeBillLinkRepo struct {
	payments []*domain.BillPayment
	deleted  []string
}

func (f *fakeBillLinkRepo) FindPaymentsByTransaction(ctx context.Context, txID string) ([]*domain.BillPayment, error) {
	var out []*domain.BillPayment
	for _, p := range f.payments {
		if p.TransactionID != nil && *p.TransactionID == txID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (f *fakeBillLinkRepo) DeletePayment(ctx context.Context, paymentID string) error {
	f.deleted = append(f.deleted, paymentID)
	return nil
}

type fakeProvenanceMarker struct {
	calls []struct {
		txID   string
		status string
	}
}

func (f *fakeProvenanceMarker) MarkTransactionProvenance(ctx context.Context, txID, status string, at time.Time) error {
	f.calls = append(f.calls, struct {
		txID   string
		status string
	}{txID, status})
	return nil
}

func editTestService() (*TransactionService, *mockTransactionRepo, *mockWalletRepo) {
	repo := &mockTransactionRepo{
		transactions: []*domain.Transaction{
			{
				ID: "tx-1", UserEmail: "user@test.com", AmountCents: 1000,
				Currency: "PHP", Category: "food", Description: "lunch",
				Type: domain.TransactionExpense, WalletID: "w-default",
				TransactionDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
				CreatedAt:       time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
				Revision:        1,
			},
		},
	}
	walletRepo := &mockWalletRepo{
		wallets: []*domain.Wallet{
			{ID: "w-default", UserEmail: "user@test.com", Name: "Cash", Currency: "PHP", IsDefault: true},
			{ID: "w-usd", UserEmail: "user@test.com", Name: "Dollar", Currency: "USD"},
		},
	}
	return NewTransactionService(repo, walletRepo, timeutil.New(time.UTC)), repo, walletRepo
}

// ---- edit/delete tests ----

func TestUpdate_PartialEdit_PreservesOmittedFields(t *testing.T) {
	svc, _, _ := editTestService()
	ctx := context.Background()

	cat := "groceries"
	tx, err := svc.Update(ctx, "user@test.com", "tx-1", UpdateTransactionInput{
		Category:         &cat,
		ExpectedRevision: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, "groceries", tx.Category)
	assert.Equal(t, int64(1000), tx.AmountCents)
	assert.Equal(t, "lunch", tx.Description)
	assert.Equal(t, domain.TransactionExpense, tx.Type)
	assert.Equal(t, "PHP", tx.Currency)
	assert.Equal(t, 2, tx.Revision)
}

func TestUpdate_EmptyPatch_ReturnsError(t *testing.T) {
	svc, _, _ := editTestService()

	_, err := svc.Update(context.Background(), "user@test.com", "tx-1", UpdateTransactionInput{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty update")
}

func TestUpdate_StaleRevision_ReturnsStaleError(t *testing.T) {
	svc, _, _ := editTestService()

	cat := "food"
	_, err := svc.Update(context.Background(), "user@test.com", "tx-1", UpdateTransactionInput{
		Category:         &cat,
		ExpectedRevision: 99,
	})
	assert.ErrorIs(t, err, domain.ErrStaleRevision)
}

func TestUpdate_InvalidValues_ReturnsError(t *testing.T) {
	svc, _, _ := editTestService()

	neg := int64(-5)
	_, err := svc.Update(context.Background(), "user@test.com", "tx-1", UpdateTransactionInput{
		AmountCents:      &neg,
		ExpectedRevision: 1,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "positive")
}

func TestUpdate_WalletChange_RederivesCurrency(t *testing.T) {
	svc, _, _ := editTestService()
	ctx := context.Background()

	walletID := "w-usd"
	tx, err := svc.Update(ctx, "user@test.com", "tx-1", UpdateTransactionInput{
		WalletID:         &walletID,
		ExpectedRevision: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, "w-usd", tx.WalletID)
	assert.Equal(t, "USD", tx.Currency)
}

func TestUpdate_ImportedTransaction_MarksProvenanceModified(t *testing.T) {
	svc, repo, _ := editTestService()
	repo.transactions[0].Imported = true
	marker := &fakeProvenanceMarker{}
	svc.WithImportProvenanceMarker(marker)
	ctx := context.Background()

	cat := "groceries"
	_, err := svc.Update(ctx, "user@test.com", "tx-1", UpdateTransactionInput{
		Category:         &cat,
		ExpectedRevision: 1,
	})
	require.NoError(t, err)
	require.Len(t, marker.calls, 1)
	assert.Equal(t, "tx-1", marker.calls[0].txID)
	assert.Equal(t, domain.EntityStatusModified, marker.calls[0].status)
}

func TestUpdate_ManualTransaction_SkipsProvenance(t *testing.T) {
	svc, _, _ := editTestService()
	marker := &fakeProvenanceMarker{}
	svc.WithImportProvenanceMarker(marker)
	ctx := context.Background()

	cat := "groceries"
	_, err := svc.Update(ctx, "user@test.com", "tx-1", UpdateTransactionInput{
		Category:         &cat,
		ExpectedRevision: 1,
	})
	require.NoError(t, err)
	assert.Empty(t, marker.calls)
}

func TestUpdate_AutoBillLinkRemoved_ManualKept(t *testing.T) {
	svc, _, _ := editTestService()
	autoID := "pay-auto"
	links := &fakeBillLinkRepo{payments: []*domain.BillPayment{
		{ID: autoID, TransactionID: strPtr("tx-1"), TransactionLinkSource: domain.PaymentLinkAuto},
		{ID: "pay-manual", TransactionID: strPtr("tx-1"), TransactionLinkSource: domain.PaymentLinkManual},
	}}
	svc.WithBillLinkRepo(links)
	ctx := context.Background()

	amount := int64(2000)
	_, err := svc.Update(ctx, "user@test.com", "tx-1", UpdateTransactionInput{
		AmountCents:      &amount,
		ExpectedRevision: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, []string{autoID}, links.deleted)
}

func TestUpdate_NonMatchAffectingChange_KeepsAutoLink(t *testing.T) {
	svc, _, _ := editTestService()
	links := &fakeBillLinkRepo{payments: []*domain.BillPayment{
		{ID: "pay-auto", TransactionID: strPtr("tx-1"), TransactionLinkSource: domain.PaymentLinkAuto},
	}}
	svc.WithBillLinkRepo(links)
	ctx := context.Background()

	desc := "edited note"
	_, err := svc.Update(ctx, "user@test.com", "tx-1", UpdateTransactionInput{
		Description:      &desc,
		ExpectedRevision: 1,
	})
	require.NoError(t, err)
	assert.Empty(t, links.deleted)
}

func TestDelete_ImportedTransaction_MarksProvenanceDeleted(t *testing.T) {
	svc, repo, _ := editTestService()
	repo.transactions[0].Imported = true
	marker := &fakeProvenanceMarker{}
	svc.WithImportProvenanceMarker(marker)

	err := svc.Delete(context.Background(), "tx-1", "user@test.com")
	require.NoError(t, err)
	require.Len(t, marker.calls, 1)
	assert.Equal(t, domain.EntityStatusDeleted, marker.calls[0].status)
}

func TestDelete_RemovesAutoBillLinks_KeepsManual(t *testing.T) {
	svc, _, _ := editTestService()
	autoID := "pay-auto"
	links := &fakeBillLinkRepo{payments: []*domain.BillPayment{
		{ID: autoID, TransactionID: strPtr("tx-1"), TransactionLinkSource: domain.PaymentLinkAuto},
		{ID: "pay-manual", TransactionID: strPtr("tx-1"), TransactionLinkSource: domain.PaymentLinkManual},
	}}
	svc.WithBillLinkRepo(links)

	err := svc.Delete(context.Background(), "tx-1", "user@test.com")
	require.NoError(t, err)
	assert.Equal(t, []string{autoID}, links.deleted)
}

func TestDeleteAtRevision_StaleRevision_ReturnsStaleError(t *testing.T) {
	svc, _, _ := editTestService()

	err := svc.DeleteAtRevision(context.Background(), "tx-1", "user@test.com", 99)
	assert.ErrorIs(t, err, domain.ErrStaleRevision)
}

func strPtr(s string) *string { return &s }

// ---- bulk edit/delete tests ----

// rollbackCoordinator mimics the infrastructure Coordinator: fn runs inside a
// logical transaction that restores the in-memory repo state when it fails.
type rollbackCoordinator struct{ repo *mockTransactionRepo }

func (c rollbackCoordinator) WithTx(ctx context.Context, fn func(ctx context.Context) error) error {
	snapshot := make([]*domain.Transaction, len(c.repo.transactions))
	copy(snapshot, c.repo.transactions)
	if err := fn(ctx); err != nil {
		c.repo.transactions = snapshot
		return err
	}
	return nil
}

func bulkService() (*TransactionService, *mockTransactionRepo, *fakeBillLinkRepo, *fakeProvenanceMarker) {
	repo := &mockTransactionRepo{}
	for i := 1; i <= 3; i++ {
		repo.transactions = append(repo.transactions, &domain.Transaction{
			ID: fmt.Sprintf("tx-%d", i), UserEmail: "user@test.com", AmountCents: int64(1000 * i),
			Currency: "PHP", Category: "food", Description: "item",
			Type: domain.TransactionExpense, WalletID: "w-default",
			TransactionDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			CreatedAt:       time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
			Revision:        1,
		})
	}
	walletRepo := &mockWalletRepo{
		wallets: []*domain.Wallet{
			{ID: "w-default", UserEmail: "user@test.com", Name: "Cash", Currency: "PHP", IsDefault: true},
		},
	}
	svc := NewTransactionService(repo, walletRepo, timeutil.New(time.UTC))
	links := &fakeBillLinkRepo{}
	marker := &fakeProvenanceMarker{}
	svc.WithBillLinkRepo(links).WithImportProvenanceMarker(marker)
	svc.WithCoordinator(rollbackCoordinator{repo: repo})
	return svc, repo, links, marker
}

func bulkItems() []BulkItem {
	return []BulkItem{{ID: "tx-1", Revision: 1}, {ID: "tx-2", Revision: 1}, {ID: "tx-3", Revision: 1}}
}

func TestUpdateMany_AppliesPatchToAllItems(t *testing.T) {
	svc, repo, _, _ := bulkService()
	ctx := context.Background()

	cat := "groceries"
	updated, err := svc.UpdateMany(ctx, "user@test.com", bulkItems(), BulkPatch{Category: &cat})
	require.NoError(t, err)
	require.Len(t, updated, 3)
	for i, tx := range updated {
		assert.Equal(t, "groceries", tx.Category)
		assert.Equal(t, 2, tx.Revision)
		// Stored rows moved on too.
		stored := repo.transactions[i]
		assert.Equal(t, "groceries", stored.Category)
		assert.Equal(t, 2, stored.Revision)
	}
}

func TestUpdateMany_AllOrNothingOnStaleRevision(t *testing.T) {
	svc, repo, _, _ := bulkService()
	ctx := context.Background()

	// tx-2 has moved on (revision 2).
	repo.transactions[1].Revision = 2
	cat := "groceries"
	_, err := svc.UpdateMany(ctx, "user@test.com", bulkItems(), BulkPatch{Category: &cat})
	assert.ErrorIs(t, err, domain.ErrStaleRevision)

	// Nothing applied: tx-1 and tx-3 still at revision 1 with original values.
	assert.Equal(t, "food", repo.transactions[0].Category)
	assert.Equal(t, 1, repo.transactions[0].Revision)
	assert.Equal(t, 1, repo.transactions[2].Revision)
}

func TestUpdateMany_RejectsEmptyPatchAndOversizeAndDuplicates(t *testing.T) {
	svc, _, _, _ := bulkService()
	ctx := context.Background()

	_, err := svc.UpdateMany(ctx, "user@test.com", bulkItems(), BulkPatch{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty bulk update")

	items := make([]BulkItem, 0, MaxBulkTransactionItems+1)
	for i := 0; i < MaxBulkTransactionItems+1; i++ {
		items = append(items, BulkItem{ID: fmt.Sprintf("tx-%d", i), Revision: 1})
	}
	_, err = svc.UpdateMany(ctx, "user@test.com", items, BulkPatch{Category: strPtr("x")})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "limit")

	_, err = svc.UpdateMany(ctx, "user@test.com",
		[]BulkItem{{ID: "tx-1", Revision: 1}, {ID: "tx-1", Revision: 1}},
		BulkPatch{Category: strPtr("x")})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestUpdateMany_MarksProvenanceForImportedItems(t *testing.T) {
	svc, repo, _, marker := bulkService()
	repo.transactions[0].Imported = true
	repo.transactions[2].Imported = true
	ctx := context.Background()

	cat := "groceries"
	_, err := svc.UpdateMany(ctx, "user@test.com", bulkItems(), BulkPatch{Category: &cat})
	require.NoError(t, err)
	require.Len(t, marker.calls, 2)
	assert.Equal(t, "tx-1", marker.calls[0].txID)
	assert.Equal(t, "tx-3", marker.calls[1].txID)
}

func TestUpdateMany_ReconcilesAutoBillLinks(t *testing.T) {
	svc, _, links, _ := bulkService()
	autoID := "pay-auto-1"
	links.payments = []*domain.BillPayment{
		{ID: autoID, TransactionID: strPtr("tx-1"), TransactionLinkSource: domain.PaymentLinkAuto},
		{ID: "pay-manual-1", TransactionID: strPtr("tx-1"), TransactionLinkSource: domain.PaymentLinkManual},
	}
	ctx := context.Background()

	cat := "groceries"
	_, err := svc.UpdateMany(ctx, "user@test.com", bulkItems(), BulkPatch{Category: &cat})
	require.NoError(t, err)
	assert.Equal(t, []string{autoID}, links.deleted)
}

func TestDeleteMany_RemovesAllAndCounts(t *testing.T) {
	svc, repo, _, marker := bulkService()
	repo.transactions[1].Imported = true

	deleted, err := svc.DeleteMany(context.Background(), "user@test.com", bulkItems())
	require.NoError(t, err)
	assert.Equal(t, 3, deleted)
	assert.Len(t, repo.transactions, 0)
	require.Len(t, marker.calls, 1)
	assert.Equal(t, domain.EntityStatusDeleted, marker.calls[0].status)
}

func TestDeleteMany_AllOrNothingOnStaleRevision(t *testing.T) {
	svc, repo, _, _ := bulkService()
	repo.transactions[2].Revision = 9

	_, err := svc.DeleteMany(context.Background(), "user@test.com", bulkItems())
	assert.ErrorIs(t, err, domain.ErrStaleRevision)
	assert.Len(t, repo.transactions, 3)
}

func TestDeleteMany_RejectsEmptyAndDuplicates(t *testing.T) {
	svc, _, _, _ := bulkService()

	_, err := svc.DeleteMany(context.Background(), "user@test.com", nil)
	assert.Error(t, err)

	_, err = svc.DeleteMany(context.Background(), "user@test.com",
		[]BulkItem{{ID: "tx-1", Revision: 1}, {ID: "tx-1", Revision: 1}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}
