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
	payment, err := svc.MarkPaid(context.Background(), bill.ID, "user@test.com", dueDate, nil)
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

	_, err := svc.MarkPaid(context.Background(), bill.ID, "other@test.com", time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), nil)
	assert.Error(t, err)
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
