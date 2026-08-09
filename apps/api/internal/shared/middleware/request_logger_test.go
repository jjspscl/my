package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
)

func TestRequestLogger_LogsRequestIDFromContext(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := RequestLogger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))

	// chimw.RequestID generates and stores the ID in context; the header is
	// not set by the client here, so the old header-read bug would log "".
	req := httptest.NewRequest(http.MethodPost, "/api/v1/test", nil)
	w := httptest.NewRecorder()

	chimw.RequestID(handler).ServeHTTP(w, req)

	var record map[string]any
	assert.NoError(t, json.Unmarshal(buf.Bytes(), &record))
	assert.Equal(t, "request", record["msg"])
	assert.Equal(t, http.MethodPost, record["method"])
	assert.Equal(t, "/api/v1/test", record["path"])
	assert.Equal(t, float64(http.StatusCreated), record["status"])
	assert.NotEmpty(t, record["request_id"], "request_id must come from context, not header")
	assert.NotEmpty(t, record["duration"])
}

func TestRequestLogger_NoRequestID_LogsEmpty(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := RequestLogger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var record map[string]any
	assert.NoError(t, json.Unmarshal(buf.Bytes(), &record))
	assert.Equal(t, "", record["request_id"])
}

func TestRequestLogger_ClientHeaderNotTrusted(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := RequestLogger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// A client-supplied header must NOT be logged as the request ID; only the
	// context value set by chimw.RequestID is authoritative.
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("X-Request-Id", "client-forged-id")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var record map[string]any
	assert.NoError(t, json.Unmarshal(buf.Bytes(), &record))
	assert.Equal(t, "", record["request_id"])
}

func TestRequestLogger_DefaultStatusIs200(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := RequestLogger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No explicit WriteHeader — statusWriter must default to 200.
		_, _ = w.Write([]byte("ok"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var record map[string]any
	assert.NoError(t, json.Unmarshal(buf.Bytes(), &record))
	assert.Equal(t, float64(http.StatusOK), record["status"])
}