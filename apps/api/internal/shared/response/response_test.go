package response

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
)

// recordHandler captures slog records for assertions.
type recordHandler struct {
	records []slog.Record
}

func (h *recordHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recordHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}

func (h *recordHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recordHandler) WithGroup(string) slog.Handler      { return h }

func recordAttrs(r slog.Record) map[string]any {
	out := map[string]any{}
	r.Attrs(func(a slog.Attr) bool {
		out[a.Key] = a.Value.Any()
		return true
	})
	return out
}

func TestWriteError_WithCause_LogsErrorLevelAndCause(t *testing.T) {
	handler := &recordHandler{}
	SetLogger(slog.New(handler))
	t.Cleanup(func() { SetLogger(nil) })

	cause := errors.New("db connection refused")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/finance/transactions", nil)
	req = req.WithContext(context.WithValue(req.Context(), chimw.RequestIDKey, "req-123"))
	w := httptest.NewRecorder()

	WriteError(w, req, http.StatusInternalServerError, "internal error", cause)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"error":"internal error"}`, w.Body.String())

	assert.Len(t, handler.records, 1)
	rec := handler.records[0]
	assert.Equal(t, slog.LevelError, rec.Level)
	assert.Equal(t, "request error", rec.Message)
	attrs := recordAttrs(rec)
	assert.Equal(t, int64(500), attrs["status"])
	assert.Equal(t, "POST", attrs["method"])
	assert.Equal(t, "/api/v1/finance/transactions", attrs["path"])
	assert.Equal(t, "req-123", attrs["request_id"])
	assert.Equal(t, "internal error", attrs["client_msg"])
	assert.Equal(t, "db connection refused", attrs["cause"].(error).Error())
}

func TestWriteError_NoCause_LogsAtWarnLevel(t *testing.T) {
	handler := &recordHandler{}
	SetLogger(slog.New(handler))
	t.Cleanup(func() { SetLogger(nil) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()

	WriteError(w, req, http.StatusUnauthorized, "not authenticated", nil)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
	assert.Len(t, handler.records, 1)
	rec := handler.records[0]
	assert.Equal(t, slog.LevelWarn, rec.Level)
	attrs := recordAttrs(rec)
	assert.Equal(t, int64(401), attrs["status"])
	_, hasCause := attrs["cause"]
	assert.False(t, hasCause, "no cause attr when err is nil")
}

func TestWriteError_NoRequestID_LogsEmpty(t *testing.T) {
	handler := &recordHandler{}
	SetLogger(slog.New(handler))
	t.Cleanup(func() { SetLogger(nil) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	WriteError(w, req, http.StatusBadRequest, "bad request", nil)

	attrs := recordAttrs(handler.records[0])
	assert.Equal(t, "", attrs["request_id"])
}

func TestWriteJSON_SetsContentTypeAndBody(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, http.StatusCreated, map[string]string{"ok": "true"})

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"ok":"true"}`, w.Body.String())
}

func TestSetLogger_NilResetsToDefault(t *testing.T) {
	SetLogger(nil)
	assert.NotNil(t, logger)
	// Default logger must not panic on use.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	w := httptest.NewRecorder()
	WriteError(w, req, http.StatusBadRequest, "bad request", nil)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
