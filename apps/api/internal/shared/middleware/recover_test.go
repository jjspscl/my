package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jjspscl/my/internal/shared/response"
	"github.com/stretchr/testify/assert"
)

type recoverRecordHandler struct {
	records []slog.Record
}

func (h *recoverRecordHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *recoverRecordHandler) Handle(_ context.Context, r slog.Record) error {
	h.records = append(h.records, r)
	return nil
}

func (h *recoverRecordHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *recoverRecordHandler) WithGroup(string) slog.Handler      { return h }

type discardHandler struct{}

func (discardHandler) Enabled(context.Context, slog.Level) bool { return true }
func (discardHandler) Handle(context.Context, slog.Record) error { return nil }
func (discardHandler) WithAttrs([]slog.Attr) slog.Handler       { return discardHandler{} }
func (discardHandler) WithGroup(string) slog.Handler            { return discardHandler{} }

func TestRecover_Panic_Returns500AndLogsStack(t *testing.T) {
	handler := &recoverRecordHandler{}
	log := slog.New(handler)
	// The 500 response goes through response.WriteError, which logs on the
	// response package logger; silence it so only Recover's own record is
	// captured and stderr stays clean.
	response.SetLogger(slog.New(discardHandler{}))
	t.Cleanup(func() { response.SetLogger(nil) })

	recovered := Recover(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(errors.New("boom"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/finance", nil)
	w := httptest.NewRecorder()

	recovered.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.JSONEq(t, `{"error":"internal server error"}`, w.Body.String())

	assert.Len(t, handler.records, 1)
	rec := handler.records[0]
	assert.Equal(t, slog.LevelError, rec.Level)
	assert.Equal(t, "panic recovered", rec.Message)

	var stack, path, method string
	rec.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "stack":
			stack, _ = a.Value.Any().(string)
		case "path":
			path, _ = a.Value.Any().(string)
		case "method":
			method, _ = a.Value.Any().(string)
		}
		return true
	})
	assert.Contains(t, stack, "runtime/debug", "stack trace must be captured")
	assert.Equal(t, "/api/v1/finance", path)
	assert.Equal(t, http.MethodGet, method)
}

func TestRecover_NoPanic_PassesThrough(t *testing.T) {
	handler := &recoverRecordHandler{}
	log := slog.New(handler)

	recovered := Recover(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	recovered.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, handler.records, "no log record when no panic")
}