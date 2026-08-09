package infrastructure

import (
	"context"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/timeutil"
)

func manilaClock(t *testing.T) *timeutil.Clock {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Manila")
	if err != nil {
		t.Fatalf("load Manila: %v", err)
	}
	return timeutil.New(loc)
}

func mustCategory(t *testing.T, repo *CategoryRepoLibSQL, name string, classification domain.Classification, essential bool) {
	t.Helper()
	ctx := context.Background()
	if _, err := repo.db.ExecContext(ctx, `
		INSERT INTO finance_categories (name, classification, essential, active) VALUES (?, ?, ?, 1)
		ON CONFLICT(name) DO UPDATE SET classification = excluded.classification, essential = excluded.essential
	`, name, classification, boolToInt(essential)); err != nil {
		t.Fatalf("insert category %s: %v", name, err)
	}
}

func TestAnalyticsGetCashFlowGroupsByCurrency(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	walletRepo := NewWalletRepoLibSQL(db)
	txRepo := NewTransactionRepoLibSQL(db)
	analytics := NewAnalyticsRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	mustWallet(t, walletRepo, "w-usd", "USD", 0)

	clock := manilaClock(t)
	jan, _ := clock.ParseDate("2026-01-05")
	feb, _ := clock.ParseDate("2026-02-10")

	mustTransaction(t, txRepo, "t1", "PHP", "Food", 3000, domain.TransactionExpense, jan, "w-php")
	mustTransaction(t, txRepo, "t2", "PHP", "Salary", 50000, domain.TransactionIncome, jan, "w-php")
	mustTransaction(t, txRepo, "t3", "USD", "Food", 1000, domain.TransactionExpense, feb, "w-usd")

	// Half-open range Jan 1..Feb 1: only January transactions qualify.
	from, _ := clock.ParseDate("2026-01-01")
	to, _ := clock.ParseDate("2026-02-01")

	totals, err := analytics.GetCashFlow(ctx, testUser, from, to)
	if err != nil {
		t.Fatalf("GetCashFlow: %v", err)
	}
	if len(totals) != 1 {
		t.Fatalf("expected 1 currency in January, got %d", len(totals))
	}
	if totals[0].Currency != "PHP" {
		t.Errorf("currency = %s, want PHP", totals[0].Currency)
	}
	if totals[0].ExpenseCents != 3000 || totals[0].IncomeCents != 50000 || totals[0].TotalCents != 47000 {
		t.Errorf("got expense=%d income=%d net=%d, want 3000/50000/47000",
			totals[0].ExpenseCents, totals[0].IncomeCents, totals[0].TotalCents)
	}
}

func TestAnalyticsGetMonthlyCashFlow(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	walletRepo := NewWalletRepoLibSQL(db)
	txRepo := NewTransactionRepoLibSQL(db)
	analytics := NewAnalyticsRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	clock := manilaClock(t)
	jan, _ := clock.ParseDate("2026-01-05")
	feb, _ := clock.ParseDate("2026-02-05")

	mustTransaction(t, txRepo, "t1", "PHP", "Food", 1000, domain.TransactionExpense, jan, "w-php")
	mustTransaction(t, txRepo, "t2", "PHP", "Food", 2000, domain.TransactionExpense, feb, "w-php")

	from, _ := clock.ParseDate("2026-01-01")
	to, _ := clock.ParseDate("2026-03-01")

	flows, err := analytics.GetMonthlyCashFlow(ctx, testUser, from, to)
	if err != nil {
		t.Fatalf("GetMonthlyCashFlow: %v", err)
	}
	if len(flows) != 2 {
		t.Fatalf("expected 2 monthly rows, got %d", len(flows))
	}
	if flows[0].Month != "2026-01" || flows[0].ExpenseCents != 1000 {
		t.Errorf("first row = %+v, want 2026-01 expense 1000", flows[0])
	}
	if flows[1].Month != "2026-02" || flows[1].ExpenseCents != 2000 {
		t.Errorf("second row = %+v, want 2026-02 expense 2000", flows[1])
	}
}

func TestAnalyticsSpendingByClassificationJoin(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	walletRepo := NewWalletRepoLibSQL(db)
	txRepo := NewTransactionRepoLibSQL(db)
	categoryRepo := NewCategoryRepoLibSQL(db)
	analytics := NewAnalyticsRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	clock := manilaClock(t)
	day, _ := clock.ParseDate("2026-01-05")

	mustCategory(t, categoryRepo, "Food", domain.ClassificationNeeds, true)
	mustTransaction(t, txRepo, "t1", "PHP", "Food", 4000, domain.TransactionExpense, day, "w-php")
	// "Shopping" has no finance_categories row: must join as unclassified.
	mustTransaction(t, txRepo, "t2", "PHP", "Shopping", 2000, domain.TransactionExpense, day, "w-php")

	from, _ := clock.ParseDate("2026-01-01")
	to, _ := clock.ParseDate("2026-02-01")

	spends, err := analytics.GetSpendingByClassification(ctx, testUser, from, to)
	if err != nil {
		t.Fatalf("GetSpendingByClassification: %v", err)
	}
	if len(spends) != 2 {
		t.Fatalf("expected 2 classification rows, got %d", len(spends))
	}
	got := map[domain.Classification]int64{}
	for _, s := range spends {
		got[s.Classification] = s.AmountCents
	}
	if got[domain.ClassificationNeeds] != 4000 {
		t.Errorf("needs = %d, want 4000", got[domain.ClassificationNeeds])
	}
	if got[domain.ClassificationUnclassified] != 2000 {
		t.Errorf("unclassified = %d, want 2000", got[domain.ClassificationUnclassified])
	}

	unclassified, err := analytics.GetUnclassifiedSpending(ctx, testUser, from, to)
	if err != nil {
		t.Fatalf("GetUnclassifiedSpending: %v", err)
	}
	if len(unclassified) != 1 {
		t.Fatalf("expected 1 currency split, got %d", len(unclassified))
	}
	if unclassified[0].UnclassifiedCents != 2000 || unclassified[0].TotalCents != 6000 {
		t.Errorf("split = %+v, want unclassified 2000 / total 6000", unclassified[0])
	}

	top, err := analytics.GetTopUnclassifiedCategories(ctx, testUser, from, to, 5)
	if err != nil {
		t.Fatalf("GetTopUnclassifiedCategories: %v", err)
	}
	if len(top) != 1 || top[0].Category != "Shopping" || top[0].AmountCents != 2000 {
		t.Errorf("top = %+v, want Shopping 2000", top)
	}
}

func TestAnalyticsGetCategoryMonthlySpend(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	walletRepo := NewWalletRepoLibSQL(db)
	txRepo := NewTransactionRepoLibSQL(db)
	analytics := NewAnalyticsRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	clock := manilaClock(t)
	jan, _ := clock.ParseDate("2026-01-05")
	feb, _ := clock.ParseDate("2026-02-05")

	mustTransaction(t, txRepo, "t1", "PHP", "Food", 1000, domain.TransactionExpense, jan, "w-php")
	mustTransaction(t, txRepo, "t2", "PHP", "Food", 2500, domain.TransactionExpense, feb, "w-php")
	mustTransaction(t, txRepo, "t3", "PHP", "Transport", 900, domain.TransactionExpense, jan, "w-php")

	from, _ := clock.ParseDate("2026-01-01")
	to, _ := clock.ParseDate("2026-03-01")

	amounts, err := analytics.GetCategoryMonthlySpend(ctx, testUser, "Food", "PHP", from, to)
	if err != nil {
		t.Fatalf("GetCategoryMonthlySpend: %v", err)
	}
	if len(amounts) != 2 {
		t.Fatalf("expected 2 months, got %d", len(amounts))
	}
	if amounts[0].Month != "2026-01" || amounts[0].AmountCents != 1000 {
		t.Errorf("first = %+v, want 2026-01 1000", amounts[0])
	}
	if amounts[1].Month != "2026-02" || amounts[1].AmountCents != 2500 {
		t.Errorf("second = %+v, want 2026-02 2500", amounts[1])
	}
}

func TestAnalyticsGetCategoryMonthlySpendAll(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	walletRepo := NewWalletRepoLibSQL(db)
	txRepo := NewTransactionRepoLibSQL(db)
	analytics := NewAnalyticsRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	clock := manilaClock(t)
	jan, _ := clock.ParseDate("2026-01-05")
	feb, _ := clock.ParseDate("2026-02-05")

	mustTransaction(t, txRepo, "t1", "PHP", "Food", 1000, domain.TransactionExpense, jan, "w-php")
	mustTransaction(t, txRepo, "t2", "PHP", "Food", 2500, domain.TransactionExpense, feb, "w-php")
	mustTransaction(t, txRepo, "t3", "PHP", "Transport", 900, domain.TransactionExpense, jan, "w-php")

	from, _ := clock.ParseDate("2026-01-01")
	to, _ := clock.ParseDate("2026-03-01")

	rows, err := analytics.GetCategoryMonthlySpendAll(ctx, testUser, from, to)
	if err != nil {
		t.Fatalf("GetCategoryMonthlySpendAll: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	got := map[string]int64{}
	for _, r := range rows {
		got[r.Category+"|"+r.Month] = r.AmountCents
	}
	if got["Food|2026-01"] != 1000 || got["Food|2026-02"] != 2500 || got["Transport|2026-01"] != 900 {
		t.Errorf("rows = %+v, want Food 2026-01=1000, Food 2026-02=2500, Transport 2026-01=900", rows)
	}
}

func TestAnalyticsGetExpenseAmounts(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	walletRepo := NewWalletRepoLibSQL(db)
	txRepo := NewTransactionRepoLibSQL(db)
	analytics := NewAnalyticsRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	clock := manilaClock(t)
	jan, _ := clock.ParseDate("2026-01-05")
	feb, _ := clock.ParseDate("2026-02-05")

	mustTransaction(t, txRepo, "t1", "PHP", "Food", 1000, domain.TransactionExpense, jan, "w-php")
	mustTransaction(t, txRepo, "t2", "PHP", "Food", 2500, domain.TransactionExpense, feb, "w-php")
	mustTransaction(t, txRepo, "t3", "PHP", "Salary", 50000, domain.TransactionIncome, jan, "w-php")

	from, _ := clock.ParseDate("2026-01-01")
	to, _ := clock.ParseDate("2026-03-01")

	amounts, err := analytics.GetExpenseAmounts(ctx, testUser, from, to)
	if err != nil {
		t.Fatalf("GetExpenseAmounts: %v", err)
	}
	// Income must be excluded; only the two Food expenses qualify.
	if len(amounts) != 2 {
		t.Fatalf("expected 2 expense rows, got %d", len(amounts))
	}
	if amounts[0].Category != "Food" || amounts[0].AmountCents != 1000 {
		t.Errorf("first = %+v, want Food 1000", amounts[0])
	}
	if amounts[1].Month != "2026-02" || amounts[1].AmountCents != 2500 {
		t.Errorf("second = %+v, want 2026-02 2500", amounts[1])
	}
}

func TestAnalyticsGetBillReconciliation(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	billRepo := NewBillRepoLibSQL(db)
	analytics := NewAnalyticsRepoLibSQL(db)

	clock := manilaClock(t)
	start, _ := clock.ParseDate("2026-01-01")
	bill, err := domain.NewRecurringBill("b1", testUser, "Rent", "Housing", 10000, "PHP", domain.FrequencyMonthly, 1, start, nil, false, nil)
	if err != nil {
		t.Fatalf("new bill: %v", err)
	}
	if err := billRepo.SaveBill(ctx, bill); err != nil {
		t.Fatalf("save bill: %v", err)
	}

	due, _ := clock.ParseDate("2026-07-01")
	payment, err := domain.NewBillPayment("p1", "b1", due, 10000)
	if err != nil {
		t.Fatalf("new payment: %v", err)
	}
	payment.Status = domain.OccurrencePaid
	if err := billRepo.SavePayment(ctx, payment); err != nil {
		t.Fatalf("save payment: %v", err)
	}

	from, _ := clock.ParseDate("2026-07-01")
	to, _ := clock.ParseDate("2026-08-01")

	rows, err := analytics.GetBillReconciliation(ctx, testUser, from, to)
	if err != nil {
		t.Fatalf("GetBillReconciliation: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 row, got %d", len(rows))
	}
	row := rows[0]
	if row.Name != "Rent" || row.PaidCents != 10000 || row.PaidCount != 1 {
		t.Errorf("row = %+v, want Rent paid 10000 count 1", row)
	}
	if row.PaidWithoutTransactionCount != 1 {
		t.Errorf("paid without transaction = %d, want 1 (payment has no transaction_id)", row.PaidWithoutTransactionCount)
	}
}

func TestAnalyticsGetUnbudgetedSpend(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	walletRepo := NewWalletRepoLibSQL(db)
	txRepo := NewTransactionRepoLibSQL(db)
	budgetRepo := NewBudgetRepoLibSQL(db)
	analytics := NewAnalyticsRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	clock := manilaClock(t)
	day, _ := clock.ParseDate("2026-01-05")

	mustTransaction(t, txRepo, "t1", "PHP", "Food", 4000, domain.TransactionExpense, day, "w-php")
	mustTransaction(t, txRepo, "t2", "PHP", "Shopping", 1500, domain.TransactionExpense, day, "w-php")

	budget, err := domain.NewBudget("b1", testUser, "2026-01", "PHP")
	if err != nil {
		t.Fatalf("new budget: %v", err)
	}
	if err := budgetRepo.UpsertBudget(ctx, budget); err != nil {
		t.Fatalf("upsert budget: %v", err)
	}
	bc, err := domain.NewBudgetCategory("bc1", "b1", "Food", 10000, false)
	if err != nil {
		t.Fatalf("new budget category: %v", err)
	}
	if err := budgetRepo.UpsertBudgetCategories(ctx, "b1", []*domain.BudgetCategory{bc}); err != nil {
		t.Fatalf("upsert budget categories: %v", err)
	}

	from, _ := clock.ParseDate("2026-01-01")
	to, _ := clock.ParseDate("2026-02-01")

	unbudgeted, err := analytics.GetUnbudgetedSpend(ctx, testUser, "PHP", "2026-01", from, to)
	if err != nil {
		t.Fatalf("GetUnbudgetedSpend: %v", err)
	}
	if unbudgeted != 1500 {
		t.Errorf("unbudgeted = %d, want 1500 (Shopping only; Food is budgeted)", unbudgeted)
	}
}

func TestAnalyticsGetEssentialMonthlySpend(t *testing.T) {
	db := newTestDB(t)
	ctx := context.Background()
	walletRepo := NewWalletRepoLibSQL(db)
	txRepo := NewTransactionRepoLibSQL(db)
	catRepo := NewCategoryRepoLibSQL(db)
	analytics := NewAnalyticsRepoLibSQL(db)

	mustWallet(t, walletRepo, "w-php", "PHP", 0)
	clock := manilaClock(t)
	jan, _ := clock.ParseDate("2026-01-05")
	feb, _ := clock.ParseDate("2026-02-05")

	// Food is essential; Shopping has no category row (not essential).
	mustCategory(t, catRepo, "Food", domain.ClassificationNeeds, true)
	mustCategory(t, catRepo, "Transport", domain.ClassificationNeeds, false)

	mustTransaction(t, txRepo, "t1", "PHP", "Food", 1000, domain.TransactionExpense, jan, "w-php")
	mustTransaction(t, txRepo, "t2", "PHP", "Food", 2500, domain.TransactionExpense, feb, "w-php")
	mustTransaction(t, txRepo, "t3", "PHP", "Transport", 900, domain.TransactionExpense, jan, "w-php")
	mustTransaction(t, txRepo, "t4", "PHP", "Salary", 50000, domain.TransactionIncome, jan, "w-php")

	from, _ := clock.ParseDate("2026-01-01")
	to, _ := clock.ParseDate("2026-03-01")

	rows, err := analytics.GetEssentialMonthlySpend(ctx, testUser, from, to)
	if err != nil {
		t.Fatalf("GetEssentialMonthlySpend: %v", err)
	}
	// Only Food (essential) qualifies; income and non-essential Transport excluded.
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	got := map[string]int64{}
	for _, r := range rows {
		got[r.Month] = r.AmountCents
	}
	if got["2026-01"] != 1000 || got["2026-02"] != 2500 {
		t.Errorf("rows = %+v, want 2026-01=1000, 2026-02=2500", rows)
	}
}
