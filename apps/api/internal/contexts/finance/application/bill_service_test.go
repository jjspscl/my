package application

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
)

// --- Mock BillRepository ---

type mockBillRepo struct {
	bills    map[string]*domain.RecurringBill
	payments map[string]*domain.BillPayment // key: billID:dueDate
}

func newMockBillRepo() *mockBillRepo {
	return &mockBillRepo{
		bills:    make(map[string]*domain.RecurringBill),
		payments: make(map[string]*domain.BillPayment),
	}
}

func (m *mockBillRepo) SaveBill(_ context.Context, bill *domain.RecurringBill) error {
	m.bills[bill.ID] = bill
	return nil
}

func (m *mockBillRepo) UpdateBill(_ context.Context, bill *domain.RecurringBill) error {
	m.bills[bill.ID] = bill
	return nil
}

func (m *mockBillRepo) DeleteBill(_ context.Context, id, userEmail string) error {
	b, ok := m.bills[id]
	if !ok || b.UserEmail != userEmail {
		return fmt.Errorf("bill not found")
	}
	delete(m.bills, id)
	return nil
}

func (m *mockBillRepo) FindBillByID(_ context.Context, id string) (*domain.RecurringBill, error) {
	b, ok := m.bills[id]
	if !ok {
		return nil, fmt.Errorf("bill not found")
	}
	return b, nil
}

func (m *mockBillRepo) ListBills(_ context.Context, userEmail string) ([]*domain.RecurringBill, error) {
	var result []*domain.RecurringBill
	for _, b := range m.bills {
		if b.UserEmail == userEmail {
			result = append(result, b)
		}
	}
	return result, nil
}

func (m *mockBillRepo) SavePayment(_ context.Context, payment *domain.BillPayment) error {
	key := payment.BillID + ":" + payment.DueDate.Format("2006-01-02")
	m.payments[key] = payment
	return nil
}

func (m *mockBillRepo) FindPayment(_ context.Context, billID, dueDate string) (*domain.BillPayment, error) {
	key := billID + ":" + dueDate
	p, ok := m.payments[key]
	if !ok {
		return nil, fmt.Errorf("payment not found")
	}
	return p, nil
}

func (m *mockBillRepo) ListPaymentsByBill(_ context.Context, billID string) ([]*domain.BillPayment, error) {
	var result []*domain.BillPayment
	for _, p := range m.payments {
		if p.BillID == billID {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockBillRepo) ListPaymentsByBills(_ context.Context, billIDs []string, from, to time.Time) ([]*domain.BillPayment, error) {
	var result []*domain.BillPayment
	for _, p := range m.payments {
		inIDs := false
		for _, id := range billIDs {
			if p.BillID == id {
				inIDs = true
				break
			}
		}
		if inIDs && !p.DueDate.Before(from) && p.DueDate.Before(to) {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockBillRepo) ListUpcomingBills(_ context.Context, _ string, _ int) ([]*domain.BillWithPayment, error) {
	return nil, nil
}

func (m *mockBillRepo) FindTransactionByMatch(_ context.Context, _, _ string, _ int64, _, _ string) (*domain.Transaction, error) {
	return nil, nil
}

// --- Tests ---

func TestCreateBill_Valid(t *testing.T) {
	repo := newMockBillRepo()
	svc := NewBillService(repo)

	bill, err := svc.Create(context.Background(), "user@test.com", CreateBillInput{
		Name:        "Netflix",
		Category:    "Subscription",
		AmountCents: 49900,
		Frequency:   domain.FrequencyMonthly,
		DayOfMonth:  15,
		StartDate:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	assert.Equal(t, "Netflix", bill.Name)
	assert.Equal(t, int64(49900), bill.AmountCents)
	assert.NotEmpty(t, bill.ID)
}

func TestCreateBill_Invalid(t *testing.T) {
	repo := newMockBillRepo()
	svc := NewBillService(repo)

	_, err := svc.Create(context.Background(), "user@test.com", CreateBillInput{
		Name:        "",
		Category:    "Subscription",
		AmountCents: 49900,
		Frequency:   domain.FrequencyMonthly,
		DayOfMonth:  15,
		StartDate:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name is required")
}

func TestListBills(t *testing.T) {
	repo := newMockBillRepo()
	svc := NewBillService(repo)

	_, _ = svc.Create(context.Background(), "user@test.com", CreateBillInput{
		Name: "Bill A", Category: "Cat", AmountCents: 100, Frequency: domain.FrequencyMonthly, DayOfMonth: 1, StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})
	_, _ = svc.Create(context.Background(), "user@test.com", CreateBillInput{
		Name: "Bill B", Category: "Cat", AmountCents: 200, Frequency: domain.FrequencyMonthly, DayOfMonth: 15, StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	bills, err := svc.List(context.Background(), "user@test.com")
	require.NoError(t, err)
	assert.Len(t, bills, 2)
}

func TestDeleteBill(t *testing.T) {
	repo := newMockBillRepo()
	svc := NewBillService(repo)

	bill, _ := svc.Create(context.Background(), "user@test.com", CreateBillInput{
		Name: "Test", Category: "Cat", AmountCents: 100, Frequency: domain.FrequencyMonthly, DayOfMonth: 1, StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	err := svc.Delete(context.Background(), bill.ID, "user@test.com")
	require.NoError(t, err)

	// Should have no bills
	bills, _ := svc.List(context.Background(), "user@test.com")
	assert.Empty(t, bills)
}

func TestMarkPaid(t *testing.T) {
	repo := newMockBillRepo()
	svc := NewBillService(repo)

	bill, _ := svc.Create(context.Background(), "user@test.com", CreateBillInput{
		Name: "Rent", Category: "Housing", AmountCents: 1500000, Frequency: domain.FrequencyMonthly, DayOfMonth: 1, StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	dueDate := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	payment, err := svc.MarkPaid(context.Background(), "user@test.com", MarkPaidInput{BillID: bill.ID, DueDate: dueDate})
	require.NoError(t, err)
	assert.Equal(t, domain.OccurrencePaid, payment.Status)
	assert.NotNil(t, payment.PaidDate)
}

func TestMarkPaid_WrongUser(t *testing.T) {
	repo := newMockBillRepo()
	svc := NewBillService(repo)

	bill, _ := svc.Create(context.Background(), "user@test.com", CreateBillInput{
		Name: "Rent", Category: "Housing", AmountCents: 1500000, Frequency: domain.FrequencyMonthly, DayOfMonth: 1, StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	_, err := svc.MarkPaid(context.Background(), "other@test.com", MarkPaidInput{BillID: bill.ID, DueDate: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)})
	assert.Error(t, err)
}

// TestMarkPaid_PreservesTransactionLink guards the regression where marking a
// bill paid without a transaction ID nulled an existing transaction link.
func TestMarkPaid_PreservesTransactionLink(t *testing.T) {
	repo := newMockBillRepo()
	svc := NewBillService(repo)

	bill, _ := svc.Create(context.Background(), "user@test.com", CreateBillInput{
		Name: "Rent", Category: "Housing", AmountCents: 1500000, Frequency: domain.FrequencyMonthly, DayOfMonth: 1, StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	dueDate := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	txID := "tx-123"
	first, err := svc.MarkPaid(context.Background(), "user@test.com", MarkPaidInput{BillID: bill.ID, DueDate: dueDate, TransactionID: &txID})
	require.NoError(t, err)
	require.NotNil(t, first.TransactionID)
	assert.Equal(t, txID, *first.TransactionID)

	// Re-marking without a transaction ID must keep the existing link.
	second, err := svc.MarkPaid(context.Background(), "user@test.com", MarkPaidInput{BillID: bill.ID, DueDate: dueDate})
	require.NoError(t, err)
	require.NotNil(t, second.TransactionID)
	assert.Equal(t, txID, *second.TransactionID)
}

// TestMarkPaid_CreateTransaction verifies the payment and its backing expense
// transaction are both written, and the payment links the new transaction.
func TestMarkPaid_CreateTransaction(t *testing.T) {
	repo := newMockBillRepo()
	txRepo := &mockTransactionRepo{}
	walletRepo := &mockWalletRepo{wallets: []*domain.Wallet{{
		ID: "wallet-1", UserEmail: "user@test.com", Name: "Cash", Currency: "PHP", IsDefault: true,
	}}}
	svc := NewBillService(repo).WithTransactionSupport(txRepo, walletRepo)

	bill, _ := svc.Create(context.Background(), "user@test.com", CreateBillInput{
		Name: "Rent", Category: "Housing", AmountCents: 1500000, Frequency: domain.FrequencyMonthly, DayOfMonth: 1, StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	dueDate := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	payment, err := svc.MarkPaid(context.Background(), "user@test.com", MarkPaidInput{BillID: bill.ID, DueDate: dueDate, CreateTransaction: true})
	require.NoError(t, err)

	require.Len(t, txRepo.transactions, 1)
	tx := txRepo.transactions[0]
	assert.Equal(t, bill.AmountCents, tx.AmountCents)
	assert.Equal(t, bill.Category, tx.Category)
	assert.Equal(t, domain.TransactionExpense, tx.Type)
	assert.Equal(t, "PHP", tx.Currency)
	assert.Equal(t, "wallet-1", tx.WalletID)
	assert.Equal(t, dueDate, tx.TransactionDate)

	require.NotNil(t, payment.TransactionID)
	assert.Equal(t, tx.ID, *payment.TransactionID)
}

// TestMarkPaid_CreateTransaction_ExplicitLinkWins verifies an explicit
// transaction ID takes precedence over create_transaction.
func TestMarkPaid_CreateTransaction_ExplicitLinkWins(t *testing.T) {
	repo := newMockBillRepo()
	txRepo := &mockTransactionRepo{}
	walletRepo := &mockWalletRepo{wallets: []*domain.Wallet{{
		ID: "wallet-1", UserEmail: "user@test.com", Name: "Cash", Currency: "PHP", IsDefault: true,
	}}}
	svc := NewBillService(repo).WithTransactionSupport(txRepo, walletRepo)

	bill, _ := svc.Create(context.Background(), "user@test.com", CreateBillInput{
		Name: "Rent", Category: "Housing", AmountCents: 1500000, Frequency: domain.FrequencyMonthly, DayOfMonth: 1, StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	dueDate := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	txID := "existing-tx"
	payment, err := svc.MarkPaid(context.Background(), "user@test.com", MarkPaidInput{BillID: bill.ID, DueDate: dueDate, TransactionID: &txID, CreateTransaction: true})
	require.NoError(t, err)
	assert.Empty(t, txRepo.transactions, "no transaction should be created when an explicit link is given")
	require.NotNil(t, payment.TransactionID)
	assert.Equal(t, txID, *payment.TransactionID)
}

// TestMarkPaid_CreateTransaction_CoordinatorError verifies a failing
// coordinator aborts the whole operation and no payment is returned.
func TestMarkPaid_CreateTransaction_CoordinatorError(t *testing.T) {
	repo := newMockBillRepo()
	txRepo := &mockTransactionRepo{}
	walletRepo := &mockWalletRepo{wallets: []*domain.Wallet{{
		ID: "wallet-1", UserEmail: "user@test.com", Name: "Cash", Currency: "PHP", IsDefault: true,
	}}}
	svc := NewBillService(repo).WithTransactionSupport(txRepo, walletRepo).WithCoordinator(failingCoordinator{})

	bill, _ := svc.Create(context.Background(), "user@test.com", CreateBillInput{
		Name: "Rent", Category: "Housing", AmountCents: 1500000, Frequency: domain.FrequencyMonthly, DayOfMonth: 1, StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	dueDate := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	payment, err := svc.MarkPaid(context.Background(), "user@test.com", MarkPaidInput{BillID: bill.ID, DueDate: dueDate, CreateTransaction: true})
	assert.Error(t, err)
	assert.Nil(t, payment)
}

// failingCoordinator simulates a database transaction that aborts before
// committing: the inner function never runs, so no writes are applied.
type failingCoordinator struct{}

func (failingCoordinator) WithTx(_ context.Context, _ func(ctx context.Context) error) error {
	return fmt.Errorf("begin tx: simulated failure")
}

func TestGetUpcoming(t *testing.T) {
	repo := newMockBillRepo()
	svc := NewBillService(repo)

	now := time.Now().UTC()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -2, 0)

	_, _ = svc.Create(context.Background(), "user@test.com", CreateBillInput{
		Name: "Monthly Bill", Category: "Cat", AmountCents: 1000, Frequency: domain.FrequencyMonthly, DayOfMonth: 15, StartDate: startDate,
	})

	upcoming, err := svc.GetUpcoming(context.Background(), "user@test.com", 60)
	require.NoError(t, err)
	assert.NotEmpty(t, upcoming)
}

func TestGenerateOccurrences_Monthly(t *testing.T) {
	bill := &domain.RecurringBill{
		Frequency:  domain.FrequencyMonthly,
		DayOfMonth: 15,
		StartDate:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)

	dates := generateOccurrences(bill, from, to)
	require.Len(t, dates, 3)
	assert.Equal(t, 15, dates[0].Day())
	assert.Equal(t, time.February, dates[0].Month())
	assert.Equal(t, time.March, dates[1].Month())
	assert.Equal(t, time.April, dates[2].Month())
}

func TestGenerateOccurrences_Weekly(t *testing.T) {
	bill := &domain.RecurringBill{
		Frequency: domain.FrequencyWeekly,
		StartDate: time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC), // Monday
	}
	from := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 19, 0, 0, 0, 0, time.UTC)

	dates := generateOccurrences(bill, from, to)
	require.Len(t, dates, 3) // 3 mondays
}

func TestGenerateOccurrences_Yearly(t *testing.T) {
	bill := &domain.RecurringBill{
		Frequency:  domain.FrequencyYearly,
		DayOfMonth: 1,
		StartDate:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2028, 12, 31, 0, 0, 0, 0, time.UTC)

	dates := generateOccurrences(bill, from, to)
	require.Len(t, dates, 3) // 2026, 2027, 2028
}

func TestDayInMonth(t *testing.T) {
	// Feb 2026 has 28 days
	d := dayInMonth(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), 30)
	assert.Equal(t, 28, d.Day())

	// Normal case
	d = dayInMonth(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), 15)
	assert.Equal(t, 15, d.Day())
}
