package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/finance/application"
	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/bootstrap"
	"github.com/jjspscl/my/internal/platform/config"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Trimmed fake repositories local to this package (Go test files cannot see
// the fakes in finance/interfaces/http). They cover only what the exercised
// tools reach through their services.

type fakeTransactionRepo struct {
	txns  []*domain.Transaction
	err   error
	byKey map[string]*domain.Transaction
}

func (f *fakeTransactionRepo) fail() error {
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeTransactionRepo) Save(_ context.Context, tx *domain.Transaction) error {
	if f.err != nil {
		return f.err
	}
	f.txns = append(f.txns, tx)
	if f.byKey != nil && tx.IdempotencyKey != "" {
		f.byKey[tx.UserEmail+":"+tx.IdempotencyKey] = tx
	}
	return nil
}
func (f *fakeTransactionRepo) FindByID(_ context.Context, _ string) (*domain.Transaction, error) {
	return nil, f.fail()
}
func (f *fakeTransactionRepo) FindByIdempotencyKey(_ context.Context, userEmail, key string) (*domain.Transaction, error) {
	if f.byKey != nil {
		if tx, ok := f.byKey[userEmail+":"+key]; ok {
			return tx, nil
		}
	}
	return nil, f.fail()
}
func (f *fakeTransactionRepo) ListByUserAndDateRange(_ context.Context, _ string, _, _ time.Time, _, _ int) ([]*domain.Transaction, error) {
	return nil, f.fail()
}
func (f *fakeTransactionRepo) Update(_ context.Context, tx *domain.Transaction, _ int) error {
	if f.err != nil {
		return f.err
	}
	for i, stored := range f.txns {
		if stored.ID == tx.ID {
			f.txns[i] = tx
			return nil
		}
	}
	return errors.New("transaction not found")
}
func (f *fakeTransactionRepo) Delete(_ context.Context, _, _ string) error { return f.fail() }
func (f *fakeTransactionRepo) DeleteAtRevision(_ context.Context, _, _ string, _ int) error {
	return f.fail()
}
func (f *fakeTransactionRepo) GetTodayTotals(_ context.Context, _ string, _ time.Time) ([]domain.CurrencyTotal, error) {
	return nil, f.fail()
}

type fakeWalletRepo struct {
	wallets []*domain.Wallet
	err     error
}

func (f *fakeWalletRepo) fail() error {
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeWalletRepo) Save(_ context.Context, _ *domain.Wallet) error { return f.fail() }
func (f *fakeWalletRepo) FindByID(_ context.Context, id string) (*domain.Wallet, error) {
	for _, w := range f.wallets {
		if w.ID == id {
			return w, nil
		}
	}
	return nil, errors.New("wallet not found")
}
func (f *fakeWalletRepo) ListByUser(_ context.Context, _ string) ([]*domain.Wallet, error) {
	return f.wallets, f.fail()
}
func (f *fakeWalletRepo) Update(_ context.Context, _ *domain.Wallet) error { return f.fail() }
func (f *fakeWalletRepo) Archive(_ context.Context, _, _ string) error     { return f.fail() }
func (f *fakeWalletRepo) FindDefault(_ context.Context, _ string) (*domain.Wallet, error) {
	for _, w := range f.wallets {
		if w.IsDefault {
			return w, nil
		}
	}
	if len(f.wallets) > 0 {
		return f.wallets[0], nil
	}
	return nil, errors.New("no wallets found")
}
func (f *fakeWalletRepo) GetBalancesByUser(_ context.Context, _ string) ([]*domain.WalletBalance, error) {
	return nil, f.fail()
}

type fakeCategoryRepo struct {
	byName map[string]*domain.Category
	err    error
}

func (f *fakeCategoryRepo) fail() error {
	if f.err != nil {
		return f.err
	}
	return nil
}

func (f *fakeCategoryRepo) List(_ context.Context) ([]*domain.Category, error) {
	out := make([]*domain.Category, 0, len(f.byName))
	for _, c := range f.byName {
		out = append(out, c)
	}
	return out, f.fail()
}
func (f *fakeCategoryRepo) FindByName(_ context.Context, name string) (*domain.Category, error) {
	c, ok := f.byName[name]
	if !ok {
		return nil, errors.New("category not found")
	}
	return c, f.fail()
}
func (f *fakeCategoryRepo) Update(_ context.Context, category *domain.Category) error {
	if f.err != nil {
		return f.err
	}
	f.byName[category.Name] = category
	return nil
}

type fakeAnalyticsRepo struct {
	classification  []domain.ClassificationSpend
	unclassified    []domain.UnclassifiedSpending
	topUnclassified []domain.CategorySpend
}

func (f *fakeAnalyticsRepo) GetSpendingByClassification(_ context.Context, _ string, _, _ time.Time) ([]domain.ClassificationSpend, error) {
	return f.classification, nil
}
func (f *fakeAnalyticsRepo) GetUnclassifiedSpending(_ context.Context, _ string, _, _ time.Time) ([]domain.UnclassifiedSpending, error) {
	return f.unclassified, nil
}
func (f *fakeAnalyticsRepo) GetTopUnclassifiedCategories(_ context.Context, _ string, _, _ time.Time, _ int) ([]domain.CategorySpend, error) {
	return f.topUnclassified, nil
}
func (f *fakeAnalyticsRepo) GetCashFlow(_ context.Context, _ string, _, _ time.Time) ([]domain.CurrencyTotal, error) {
	return nil, nil
}
func (f *fakeAnalyticsRepo) GetMonthlyCashFlow(_ context.Context, _ string, _, _ time.Time) ([]domain.MonthlyCashFlow, error) {
	return nil, nil
}
func (f *fakeAnalyticsRepo) GetCategoryMonthlySpend(_ context.Context, _, _, _ string, _, _ time.Time) ([]domain.MonthlyAmount, error) {
	return nil, nil
}
func (f *fakeAnalyticsRepo) GetUnbudgetedSpend(_ context.Context, _, _, _ string, _, _ time.Time) (int64, error) {
	return 0, nil
}
func (f *fakeAnalyticsRepo) GetCategoryMonthlySpendAll(_ context.Context, _ string, _, _ time.Time) ([]domain.CategoryMonthlySpend, error) {
	return nil, nil
}
func (f *fakeAnalyticsRepo) GetExpenseAmounts(_ context.Context, _ string, _, _ time.Time) ([]domain.ExpenseAmount, error) {
	return nil, nil
}
func (f *fakeAnalyticsRepo) GetBillReconciliation(_ context.Context, _ string, _, _ time.Time) ([]domain.BillReconciliationRow, error) {
	return nil, nil
}
func (f *fakeAnalyticsRepo) GetEssentialMonthlySpend(_ context.Context, _ string, _, _ time.Time) ([]domain.MonthlyEssentialSpend, error) {
	return nil, nil
}

// newTestApp builds a bootstrap.App whose services run over the fakes.
func newTestApp(tx *fakeTransactionRepo, wallet *fakeWalletRepo, category *fakeCategoryRepo, analytics *fakeAnalyticsRepo) *bootstrap.App {
	app := &bootstrap.App{
		Cfg: &config.Config{UserEmail: "user@example.com"},
	}
	if tx != nil {
		app.Tx = application.NewTransactionService(tx, wallet, nil)
	}
	if category != nil {
		app.Category = application.NewCategoryService(category)
	}
	if analytics != nil {
		app.Analytics = application.NewAnalyticsService(analytics, nil, nil)
	}
	return app
}

// callTool connects an in-memory client to the server and invokes one tool.
func callTool(t *testing.T, server *mcpsdk.Server, name string, arguments any) (*mcpsdk.CallToolResult, error) {
	t.Helper()
	ctx := context.Background()
	serverTransport, clientTransport := mcpsdk.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "test"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	return clientSession.CallTool(ctx, &mcpsdk.CallToolParams{Name: name, Arguments: arguments})
}

// decodeStructured decodes a successful tool result's StructuredContent.
func decodeStructured(t *testing.T, result *mcpsdk.CallToolResult, out any) {
	t.Helper()
	if result.IsError {
		t.Fatalf("tool returned error: %s", structuredText(result))
	}
	var raw []byte
	switch v := result.StructuredContent.(type) {
	case json.RawMessage:
		raw = v
	case nil:
		t.Fatal("StructuredContent is nil")
	default:
		// The in-memory client transport decodes into map[string]any.
		var err error
		raw, err = json.Marshal(v)
		if err != nil {
			t.Fatalf("re-marshal structured content: %v", err)
		}
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("unmarshal structured content: %v", err)
	}
}

func structuredText(result *mcpsdk.CallToolResult) string {
	if len(result.Content) == 0 {
		return "(no content)"
	}
	var b strings.Builder
	for _, c := range result.Content {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

func TestCallToolReadToolReturnsData(t *testing.T) {
	analytics := &fakeAnalyticsRepo{
		classification: []domain.ClassificationSpend{
			{Currency: "PHP", Classification: domain.ClassificationNeeds, AmountCents: 6000},
			{Currency: "PHP", Classification: domain.ClassificationWants, AmountCents: 3000},
		},
		unclassified: []domain.UnclassifiedSpending{
			{Currency: "PHP", UnclassifiedCents: 0, TotalCents: 9000},
		},
	}
	app := newTestApp(nil, nil, nil, analytics)
	server := NewServer(app, Options{})

	result, err := callTool(t, server, "finance_spending_summary", map[string]any{"from": "2026-01-01T00:00:00Z", "to": "2026-02-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	var out struct {
		Currencies []struct {
			Currency string `json:"currency"`
		} `json:"currencies"`
	}
	decodeStructured(t, result, &out)
	if len(out.Currencies) != 1 || out.Currencies[0].Currency != "PHP" {
		t.Errorf("currencies = %+v, want one PHP", out.Currencies)
	}
}

func TestCallToolCreateTransactionPersists(t *testing.T) {
	tx := &fakeTransactionRepo{byKey: map[string]*domain.Transaction{}}
	wallet := &fakeWalletRepo{wallets: []*domain.Wallet{
		{ID: "w-1", UserEmail: "user@example.com", Currency: "PHP", IsDefault: true},
	}}
	app := newTestApp(tx, wallet, nil, nil)
	server := NewServer(app, Options{})

	result, err := callTool(t, server, "finance_create_transaction", map[string]any{
		"amount_cents":     2500,
		"category":         "Food",
		"description":      "groceries",
		"type":             "expense",
		"transaction_date": "2026-01-15T00:00:00Z",
		"idempotency_key":  "mcp-test-1",
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	var out struct {
		ID       string `json:"id"`
		Currency string `json:"currency"`
	}
	decodeStructured(t, result, &out)
	if out.Currency != "PHP" {
		t.Errorf("currency = %q, want PHP (from default wallet)", out.Currency)
	}
	if len(tx.txns) != 1 {
		t.Fatalf("stored transactions = %d, want 1", len(tx.txns))
	}
	if tx.txns[0].AmountCents != 2500 || tx.txns[0].Category != "Food" {
		t.Errorf("stored tx = %+v, want 2500 Food", tx.txns[0])
	}
}

func TestReadOnlyServerRejectsWriteToolAtExecution(t *testing.T) {
	tx := &fakeTransactionRepo{}
	wallet := &fakeWalletRepo{wallets: []*domain.Wallet{
		{ID: "w-1", UserEmail: "user@example.com", Currency: "PHP", IsDefault: true},
	}}
	app := newTestApp(tx, wallet, nil, nil)
	server := NewServer(app, Options{ReadOnly: true})

	_, err := callTool(t, server, "finance_create_transaction", map[string]any{
		"amount_cents":     2500,
		"category":         "Food",
		"type":             "expense",
		"transaction_date": "2026-01-15T00:00:00Z",
	})
	if err == nil {
		t.Fatal("CallTool succeeded on read-only server, want unknown-tool error")
	}
	if !strings.Contains(err.Error(), `unknown tool "finance_create_transaction"`) {
		t.Errorf("error = %v, want unknown tool", err)
	}
	if len(tx.txns) != 0 {
		t.Errorf("transaction persisted despite read-only mode: %d stored", len(tx.txns))
	}
}

func TestCallToolInvalidArgumentsErrorsNotPanics(t *testing.T) {
	tx := &fakeTransactionRepo{}
	wallet := &fakeWalletRepo{wallets: []*domain.Wallet{
		{ID: "w-1", UserEmail: "user@example.com", Currency: "PHP", IsDefault: true},
	}}
	app := newTestApp(tx, wallet, nil, nil)
	server := NewServer(app, Options{})

	// amount_cents = 0 violates the domain invariant (amount must be positive).
	result, err := callTool(t, server, "finance_create_transaction", map[string]any{
		"amount_cents":     0,
		"category":         "Food",
		"type":             "expense",
		"transaction_date": "2026-01-15T00:00:00Z",
	})
	if err != nil {
		t.Fatalf("CallTool returned protocol error, want IsError result: %v", err)
	}
	if !result.IsError {
		t.Fatalf("IsError = false, want true for invalid arguments; content: %s", structuredText(result))
	}
	if !strings.Contains(structuredText(result), "amount must be positive") {
		t.Errorf("error content = %q, want amount-must-be-positive", structuredText(result))
	}
}

func TestClassifyCategoryExplicitFalseSurvives(t *testing.T) {
	category := &fakeCategoryRepo{byName: map[string]*domain.Category{
		"Food": {Name: "Food", Classification: domain.ClassificationNeeds, Essential: true, Active: true},
	}}
	app := newTestApp(nil, nil, category, nil)
	server := NewServer(app, Options{})

	// active defaults to true when omitted; explicit false must override it.
	result, err := callTool(t, server, "finance_classify_category", map[string]any{
		"name":           "Food",
		"classification": "wants",
		"essential":      false,
		"active":         false,
	})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool error: %s", structuredText(result))
	}
	updated := category.byName["Food"]
	if updated.Essential {
		t.Error("essential = true, want explicit false to survive the *bool default")
	}
	if updated.Active {
		t.Error("active = true, want explicit false to override the true default")
	}
	if updated.Classification != domain.ClassificationWants {
		t.Errorf("classification = %q, want wants", updated.Classification)
	}
}

func TestSpendingSummarySurfacesInsufficientClassification(t *testing.T) {
	analytics := &fakeAnalyticsRepo{
		classification: []domain.ClassificationSpend{
			{Currency: "PHP", Classification: domain.ClassificationUnclassified, AmountCents: 8000},
			{Currency: "PHP", Classification: domain.ClassificationNeeds, AmountCents: 2000},
		},
		unclassified: []domain.UnclassifiedSpending{
			{Currency: "PHP", UnclassifiedCents: 8000, TotalCents: 10000},
		},
		topUnclassified: []domain.CategorySpend{
			{Category: "Food", AmountCents: 5000},
		},
	}
	app := newTestApp(nil, nil, nil, analytics)
	server := NewServer(app, Options{})

	result, err := callTool(t, server, "finance_spending_summary", map[string]any{"from": "2026-01-01T00:00:00Z", "to": "2026-02-01T00:00:00Z"})
	if err != nil {
		t.Fatalf("CallTool: %v", err)
	}
	if !result.IsError {
		t.Fatalf("IsError = false, want insufficient-classification surfaced as tool error")
	}
	if !strings.Contains(structuredText(result), "insufficient classification") {
		t.Errorf("error content = %q, want insufficient classification", structuredText(result))
	}
}
