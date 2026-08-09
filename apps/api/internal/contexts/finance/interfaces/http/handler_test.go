package http

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jjspscl/my/internal/contexts/finance/application"
	"github.com/jjspscl/my/internal/platform/timeutil"
	"github.com/jjspscl/my/internal/shared/middleware"
)

// fakeSession satisfies the interface RequireAuth needs: it resolves any
// session cookie to the test user.
type fakeSession struct{}

func (fakeSession) Get(_ context.Context, _ string) (string, error) {
	return analyticsTestUser, nil
}

// newAnalyticsRouter mounts both analytics handlers behind the real RequireAuth
// middleware, mirroring cmd/api/router.go. Requests must carry a my_session
// cookie; the fake session resolves it to analyticsTestUser.
func newAnalyticsRouter(analytics *application.AnalyticsService, derived *application.DerivedAnalyticsService) http.Handler {
	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireAuth(fakeSession{}))
		r.Route("/analytics", func(r chi.Router) {
			NewAnalyticsHandler(analytics, timeutil.New(time.UTC)).Routes(r)
			NewDerivedAnalyticsHandler(derived, timeutil.New(time.UTC)).Routes(r)
		})
	})
	return r
}

// newTestServices builds both analytics services over the given fakes. The
// derived service needs a BillService for affordability; it is constructed
// over the same fake bill repo.
func newTestServices(analytics *fakeAnalyticsRepo, budget *fakeBudgetRepo, goal *fakeGoalRepo, wallet *fakeWalletRepo, bill *fakeBillRepo) (*application.AnalyticsService, *application.DerivedAnalyticsService) {
	analyticsSvc := application.NewAnalyticsService(analytics, budget, goal)
	billSvc := application.NewBillService(bill)
	derived := application.NewDerivedAnalyticsService(analytics, wallet, bill, analyticsSvc, billSvc)
	return analyticsSvc, derived
}

// doAnalyticsRequest performs a GET against the router with a session cookie.
func doAnalyticsRequest(t *testing.T, h http.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.AddCookie(&http.Cookie{Name: "my_session", Value: "test-session"})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// decodeData decodes the response body into a struct whose Data field matches
// the endpoint's response shape. The caller supplies the Data type.
func decodeData(t *testing.T, rec *httptest.ResponseRecorder, data any) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("unmarshal envelope: %v; body: %s", err, rec.Body.String())
	}
	if err := json.Unmarshal(envelope.Data, data); err != nil {
		t.Fatalf("unmarshal data: %v; body: %s", err, rec.Body.String())
	}
}