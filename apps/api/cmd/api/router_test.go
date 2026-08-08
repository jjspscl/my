package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	accesshttp "github.com/jjspscl/my/internal/contexts/access/interfaces/http"
	financehttp "github.com/jjspscl/my/internal/contexts/finance/interfaces/http"
	habithttp "github.com/jjspscl/my/internal/contexts/habits/interfaces/http"
	"github.com/jjspscl/my/internal/platform/session"
)

type routerTestSessions struct{}

func (routerTestSessions) Create(context.Context, string) (string, error) {
	return "", errors.New("not implemented")
}

func (routerTestSessions) Get(_ context.Context, sessionID string) (string, error) {
	if sessionID == "valid-session" {
		return "user@test.com", nil
	}
	return "", errors.New("invalid session")
}
func (routerTestSessions) Delete(context.Context, string) error { return nil }

func testRouter(t *testing.T) http.Handler {
	t.Helper()
	authHandler := accesshttp.NewAuthHandler(nil, false, 0)
	return newRouter(routerDeps{
		log:             slog.Default(),
		sessions:        routerTestSessions{},
		authHandler:     authHandler,
		financeHandler:  financehttp.NewFinanceHandler(nil),
		budgetHandler:   financehttp.NewBudgetHandler(nil),
		billHandler:     financehttp.NewBillHandler(nil),
		goalHandler:     financehttp.NewGoalHandler(nil),
		walletHandler:   financehttp.NewWalletHandler(nil),
		transferHandler: financehttp.NewTransferHandler(nil),
		habitHandler:    habithttp.NewHabitHandler(nil),
	})
}

var _ session.Store = routerTestSessions{}

func TestNewRouterDoesNotPanic(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered != nil {
			t.Fatalf("newRouter panicked: %v", recovered)
		}
	}()

	_ = testRouter(t)
}

func TestRouterHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	res := httptest.NewRecorder()
	testRouter(t).ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}
}

func TestRouterLogoutRequiresSessionBeforeCSRF(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	res := httptest.NewRecorder()
	testRouter(t).ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}

func TestRouterLogoutRequiresCSRFAfterSession(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "my_session", Value: "valid-session"})
	res := httptest.NewRecorder()
	testRouter(t).ServeHTTP(res, req)

	if res.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusForbidden)
	}
}

func TestRouterMagicLinkIsPublic(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/magic-link", nil)
	res := httptest.NewRecorder()
	testRouter(t).ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestRouterProtectedFinanceRequiresSession(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance/transactions", nil)
	res := httptest.NewRecorder()
	testRouter(t).ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}

func TestRouterMeRemainsPublic(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	res := httptest.NewRecorder()
	testRouter(t).ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}
