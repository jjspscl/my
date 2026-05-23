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

// ---- mocks ----

type mockTransactionRepo struct {
	transactions []*domain.Transaction
	saveFn       func(ctx context.Context, tx *domain.Transaction) error
	findByIDFn   func(ctx context.Context, id string) (*domain.Transaction, error)
	listFn       func(ctx context.Context, userEmail string, from, to time.Time, limit, offset int) ([]*domain.Transaction, error)
	deleteFn     func(ctx context.Context, id, userEmail string) error
	todayTotalFn func(ctx context.Context, userEmail string, date time.Time) (*domain.DailyTotal, error)
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

func (m *mockTransactionRepo) GetTodayTotal(ctx context.Context, userEmail string, date time.Time) (*domain.DailyTotal, error) {
	if m.todayTotalFn != nil {
		return m.todayTotalFn(ctx, userEmail, date)
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
	return &domain.DailyTotal{
		Date:         dateStr,
		TotalCents:   incomeCents - expenseCents,
		ExpenseCents: expenseCents,
		IncomeCents:  incomeCents,
		Currency:     "PHP",
	}, nil
}

func newTestTransactionService() *TransactionService {
	repo := &mockTransactionRepo{}
	return NewTransactionService(repo, "PHP")
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
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "type")
}

func TestGetTodayTotal_NoTransactions_ReturnsZeros(t *testing.T) {
	svc := newTestTransactionService()
	ctx := context.Background()

	total, err := svc.GetTodayTotal(ctx, "user@test.com")
	require.NoError(t, err)
	assert.Equal(t, int64(0), total.ExpenseCents)
	assert.Equal(t, int64(0), total.IncomeCents)
	assert.Equal(t, int64(0), total.TotalCents)
}

func TestGetTodayTotal_MixedTransactions_ReturnsCorrectAggregation(t *testing.T) {
	repo := &mockTransactionRepo{}
	svc := NewTransactionService(repo, "PHP")
	ctx := context.Background()
	now := time.Now()

	// Add some transactions
	repo.transactions = []*domain.Transaction{
		{ID: "tx-1", UserEmail: "user@test.com", AmountCents: 100000, Type: domain.TransactionExpense, TransactionDate: now},
		{ID: "tx-2", UserEmail: "user@test.com", AmountCents: 50000, Type: domain.TransactionExpense, TransactionDate: now},
		{ID: "tx-3", UserEmail: "user@test.com", AmountCents: 5000000, Type: domain.TransactionIncome, TransactionDate: now},
	}

	total, err := svc.GetTodayTotal(ctx, "user@test.com")
	require.NoError(t, err)
	assert.Equal(t, int64(150000), total.ExpenseCents)
	assert.Equal(t, int64(5000000), total.IncomeCents)
	assert.Equal(t, int64(4850000), total.TotalCents)
}

func TestList_DefaultLimit(t *testing.T) {
	repo := &mockTransactionRepo{}
	svc := NewTransactionService(repo, "PHP")
	ctx := context.Background()

	// Add 200 transactions
	for i := 0; i < 200; i++ {
		repo.transactions = append(repo.transactions, &domain.Transaction{
			ID:              "tx-" + string(rune(i)),
			UserEmail:       "user@test.com",
			TransactionDate: time.Now(),
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
	repo := &mockTransactionRepo{}
	svc := NewTransactionService(repo, "PHP")
	ctx := context.Background()

	repo.transactions = []*domain.Transaction{
		{ID: "tx-1", UserEmail: "user@test.com"},
	}

	err := svc.Delete(ctx, "tx-1", "user@test.com")
	assert.NoError(t, err)
}

func TestCreate_RepoFailure_ReturnsError(t *testing.T) {
	repo := &mockTransactionRepo{
		saveFn: func(ctx context.Context, tx *domain.Transaction) error {
			return errors.New("database error")
		},
	}
	svc := NewTransactionService(repo, "PHP")
	ctx := context.Background()

	_, err := svc.Create(ctx, "user@test.com", CreateTransactionInput{
		AmountCents: 100,
		Category:    "food",
		Type:        domain.TransactionExpense,
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "save transaction")
}

func TestGetTodayTotal_EmptyDateRange_ReturnsNilDefault(t *testing.T) {
	repo := &mockTransactionRepo{
		todayTotalFn: func(ctx context.Context, userEmail string, date time.Time) (*domain.DailyTotal, error) {
			return nil, nil
		},
	}
	svc := NewTransactionService(repo, "PHP")
	ctx := context.Background()

	total, err := svc.GetTodayTotal(ctx, "user@test.com")
	require.NoError(t, err)
	assert.NotNil(t, total)
	assert.Equal(t, int64(0), total.TotalCents)
	assert.Equal(t, "PHP", total.Currency)
}
