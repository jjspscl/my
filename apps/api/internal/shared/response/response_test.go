package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSON_SetsContentType(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, http.StatusOK, map[string]string{"hello": "world"})

	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
}

func TestWriteJSON_StatusAndBody(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, http.StatusCreated, map[string]bool{"ok": true})

	assert.Equal(t, http.StatusCreated, w.Code)

	var body map[string]bool
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.True(t, body["ok"])
}

func TestWriteError_SanitizedClientMessage(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)

	WriteError(w, r, http.StatusBadRequest, "invalid input", nil)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var body map[string]string
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "invalid input", body["error"])
}

func TestWriteError_DoesNotLeakInternalError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/api/v1/secret", strings.NewReader(""))

	internalErr := "database connection refused: dial tcp 127.0.0.1:3306: connect: connection refused"
	WriteError(w, r, http.StatusInternalServerError, "something went wrong", errors.New(internalErr))

	var body map[string]string
	err := json.NewDecoder(w.Body).Decode(&body)
	require.NoError(t, err)
	assert.Equal(t, "something went wrong", body["error"])
	assert.NotContains(t, body["error"], "database")
}

func TestWriteError_VariousStatusCodes(t *testing.T) {
	tests := []struct {
		status int
		msg    string
	}{
		{http.StatusBadRequest, "bad request"},
		{http.StatusUnauthorized, "unauthorized"},
		{http.StatusNotFound, "not found"},
		{http.StatusInternalServerError, "server error"},
	}

	for _, tt := range tests {
		t.Run(tt.msg, func(t *testing.T) {
			w := httptest.NewRecorder()
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			WriteError(w, r, tt.status, tt.msg, nil)

			assert.Equal(t, tt.status, w.Code)

			var body map[string]string
			err := json.NewDecoder(w.Body).Decode(&body)
			require.NoError(t, err)
			assert.Equal(t, tt.msg, body["error"])
		})
	}
}

func TestWriteJSON_ArrayPayload(t *testing.T) {
	w := httptest.NewRecorder()
	payload := []int{1, 2, 3}
	WriteJSON(w, http.StatusOK, payload)

	assert.Equal(t, http.StatusOK, w.Code)

	var decoded []int
	err := json.NewDecoder(w.Body).Decode(&decoded)
	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, decoded)
}
