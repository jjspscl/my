package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jjspscl/my/internal/contexts/finance/application"
	"github.com/jjspscl/my/internal/contexts/finance/domain"
	"github.com/jjspscl/my/internal/platform/timeutil"
	"github.com/jjspscl/my/internal/shared/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// financeTestUser is the email the fake session resolves to.
const financeTestUser = "user@example.com"

// fakeTxRepoHTTP is a minimal TransactionRepository for handler tests.
type fakeTxRepoHTTP struct {
	txns []*domain.Transaction
	err  error
}

func (f *fakeTxRepoHTTP) Save(_ context.Context, tx *domain.Transaction) error { return nil }
func (f *fakeTxRepoHTTP) FindByID(_ context.Context, id string) (*domain.Transaction, error) {
	for _, tx := range f.txns {
		if tx.ID == id {
			return tx, nil
		}
	}
	return nil, errors.New("transaction not found")
}
func (f *fakeTxRepoHTTP) FindByIdempotencyKey(_ context.Context, _, _ string) (*domain.Transaction, error) {
	return nil, nil
}
func (f *fakeTxRepoHTTP) ListByUserAndDateRange(_ context.Context, _ string, _, _ time.Time, _, _ int) ([]*domain.Transaction, error) {
	return nil, nil
}
func (f *fakeTxRepoHTTP) Update(_ context.Context, tx *domain.Transaction, expectedRevision int) error {
	for i, stored := range f.txns {
		if stored.ID == tx.ID {
			if stored.Revision != expectedRevision {
				return domain.ErrStaleRevision
			}
			next := *tx
			next.Revision = stored.Revision + 1
			f.txns[i] = &next
			return nil
		}
	}
	return errors.New("transaction not found")
}
func (f *fakeTxRepoHTTP) Delete(_ context.Context, id, userEmail string) error { return nil }
func (f *fakeTxRepoHTTP) DeleteAtRevision(_ context.Context, id, userEmail string, expectedRevision int) error {
	for _, tx := range f.txns {
		if tx.ID == id && tx.UserEmail == userEmail && tx.Revision == expectedRevision {
			return nil
		}
	}
	return domain.ErrStaleRevision
}
func (f *fakeTxRepoHTTP) GetTodayTotals(_ context.Context, _ string, _ time.Time) ([]domain.CurrencyTotal, error) {
	return nil, nil
}

// fakeWalletRepoHTTP resolves the default wallet only.
type fakeWalletRepoHTTP struct{}

func (fakeWalletRepoHTTP) Save(_ context.Context, _ *domain.Wallet) error { return nil }
func (fakeWalletRepoHTTP) FindByID(_ context.Context, _ string) (*domain.Wallet, error) {
	return &domain.Wallet{ID: "w-1", UserEmail: financeTestUser, Name: "Cash", Currency: "PHP", IsDefault: true}, nil
}
func (fakeWalletRepoHTTP) ListByUser(_ context.Context, _ string) ([]*domain.Wallet, error) {
	return nil, nil
}
func (fakeWalletRepoHTTP) Update(_ context.Context, _ *domain.Wallet) error { return nil }
func (fakeWalletRepoHTTP) Archive(_ context.Context, _, _ string) error     { return nil }
func (fakeWalletRepoHTTP) FindDefault(_ context.Context, _ string) (*domain.Wallet, error) {
	return &domain.Wallet{ID: "w-1", UserEmail: financeTestUser, Name: "Cash", Currency: "PHP", IsDefault: true}, nil
}
func (fakeWalletRepoHTTP) GetBalancesByUser(_ context.Context, _ string) ([]*domain.WalletBalance, error) {
	return nil, nil
}

func newFinanceRouter(txRepo domain.TransactionRepository) http.Handler {
	svc := application.NewTransactionService(txRepo, fakeWalletRepoHTTP{}, timeutil.New(time.UTC))
	h := NewFinanceHandler(svc, "PHP")
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(fakeSession{}))
		h.Routes(r)
	})
	return r
}

func financeSeedTx() *domain.Transaction {
	return &domain.Transaction{
		ID: "tx-1", UserEmail: financeTestUser, AmountCents: 1000,
		Currency: "PHP", Category: "food", Description: "lunch",
		Type: domain.TransactionExpense, WalletID: "w-1",
		TransactionDate: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		CreatedAt:       time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC),
		Revision:        1,
	}
}

func doFinance(t *testing.T, h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	req.AddCookie(&http.Cookie{Name: "my_session", Value: "test-session"})
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestFinanceUpdate_PatchesTransaction(t *testing.T) {
	repo := &fakeTxRepoHTTP{txns: []*domain.Transaction{financeSeedTx()}}
	h := newFinanceRouter(repo)

	rec := doFinance(t, h, http.MethodPatch, "/transactions/tx-1",
		`{"category":"groceries","description":"weekly shop"}`, map[string]string{"If-Match": `"1"`})
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data struct {
			ID          string `json:"id"`
			Category    string `json:"category"`
			Description string `json:"description"`
			Revision    int    `json:"revision"`
			AmountCents int64  `json:"amountCents"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "groceries", resp.Data.Category)
	assert.Equal(t, "weekly shop", resp.Data.Description)
	assert.Equal(t, 2, resp.Data.Revision)
	assert.Equal(t, int64(1000), resp.Data.AmountCents)
}

func TestFinanceUpdate_RequiresIfMatch(t *testing.T) {
	repo := &fakeTxRepoHTTP{txns: []*domain.Transaction{financeSeedTx()}}
	h := newFinanceRouter(repo)

	rec := doFinance(t, h, http.MethodPatch, "/transactions/tx-1", `{"category":"x"}`, nil)
	assert.Equal(t, http.StatusPreconditionRequired, rec.Code)
}

func TestFinanceUpdate_StaleRevision_Returns412(t *testing.T) {
	repo := &fakeTxRepoHTTP{txns: []*domain.Transaction{financeSeedTx()}}
	h := newFinanceRouter(repo)

	rec := doFinance(t, h, http.MethodPatch, "/transactions/tx-1", `{"category":"x"}`, map[string]string{"If-Match": `"9"`})
	assert.Equal(t, http.StatusPreconditionFailed, rec.Code)
}

func TestFinanceUpdate_UnknownField_Rejected(t *testing.T) {
	repo := &fakeTxRepoHTTP{txns: []*domain.Transaction{financeSeedTx()}}
	h := newFinanceRouter(repo)

	rec := doFinance(t, h, http.MethodPatch, "/transactions/tx-1", `{"bogus":1}`, map[string]string{"If-Match": `"1"`})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestFinanceUpdate_EmptyPatch_Rejected(t *testing.T) {
	repo := &fakeTxRepoHTTP{txns: []*domain.Transaction{financeSeedTx()}}
	h := newFinanceRouter(repo)

	rec := doFinance(t, h, http.MethodPatch, "/transactions/tx-1", `{}`, map[string]string{"If-Match": `"1"`})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestFinanceUpdate_NotFound_Returns404(t *testing.T) {
	repo := &fakeTxRepoHTTP{txns: []*domain.Transaction{financeSeedTx()}}
	h := newFinanceRouter(repo)

	rec := doFinance(t, h, http.MethodPatch, "/transactions/ghost", `{"category":"x"}`, map[string]string{"If-Match": `"1"`})
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestFinanceUpdate_MalformedIfMatch_Rejected(t *testing.T) {
	repo := &fakeTxRepoHTTP{txns: []*domain.Transaction{financeSeedTx()}}
	h := newFinanceRouter(repo)

	rec := doFinance(t, h, http.MethodPatch, "/transactions/tx-1", `{"category":"x"}`, map[string]string{"If-Match": `abc`})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestFinanceDelete_WithRevision_Succeeds(t *testing.T) {
	repo := &fakeTxRepoHTTP{txns: []*domain.Transaction{financeSeedTx()}}
	h := newFinanceRouter(repo)

	rec := doFinance(t, h, http.MethodDelete, "/transactions/tx-1", "", map[string]string{"If-Match": `"1"`})
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestFinanceDelete_StaleRevision_Returns412(t *testing.T) {
	repo := &fakeTxRepoHTTP{txns: []*domain.Transaction{financeSeedTx()}}
	h := newFinanceRouter(repo)

	rec := doFinance(t, h, http.MethodDelete, "/transactions/tx-1", "", map[string]string{"If-Match": `"5"`})
	assert.Equal(t, http.StatusPreconditionFailed, rec.Code)
}

func TestFinanceDelete_WithoutRevision_StillWorks(t *testing.T) {
	repo := &fakeTxRepoHTTP{txns: []*domain.Transaction{financeSeedTx()}}
	h := newFinanceRouter(repo)

	// Backward compatibility: MCP-style delete without If-Match.
	rec := doFinance(t, h, http.MethodDelete, "/transactions/tx-1", "", nil)
	assert.Equal(t, http.StatusOK, rec.Code)
}
