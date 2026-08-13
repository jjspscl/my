package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	accesshttp "github.com/jjspscl/my/internal/contexts/access/interfaces/http"
	financehttp "github.com/jjspscl/my/internal/contexts/finance/interfaces/http"
	habithttp "github.com/jjspscl/my/internal/contexts/habits/interfaces/http"
	"github.com/jjspscl/my/internal/platform/backup"
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
		backupHandler:   backup.NewHandler(nil),
		magicLinkRate:   6,
		financeHandler:  financehttp.NewFinanceHandler(nil, ""),
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

	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("status field = %q, want %q", body["status"], "ok")
	}
	if body["version"] == "" {
		t.Fatal("version field is empty")
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

func TestRouterDoesNotServeMCP(t *testing.T) {
	// MCP lives on a dedicated listener. Requests to the dashboard port must 404
	// rather than fall through to the SPA handler and return index.html.
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		req := httptest.NewRequest(method, "/mcp", nil)
		res := httptest.NewRecorder()
		testRouter(t).ServeHTTP(res, req)

		if res.Code != http.StatusNotFound {
			t.Fatalf("%s /mcp status = %d, want %d", method, res.Code, http.StatusNotFound)
		}
	}
}
