package response

import (
	"encoding/json"
	"log/slog"
	"net/http"

	chimw "github.com/go-chi/chi/v5/middleware"
)

// logger is the package-wide sink for WriteError. It defaults to
// slog.Default() so the package is safe to use before any wiring; swap it
// once at startup via SetLogger.
var logger = slog.Default()

// SetLogger replaces the package-level logger used by WriteError. A nil
// logger resets to slog.Default(). Call once at startup, before serving
// traffic.
func SetLogger(l *slog.Logger) {
	if l == nil {
		logger = slog.Default()
		return
	}
	logger = l
}

// WriteJSON serializes v as JSON with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// WriteError sends clientMsg as JSON error to the client and logs the
// underlying err (if non-nil) server-side with request context.
// Use for all 4xx/5xx responses — gives server visibility without leaking
// internals to the client.
func WriteError(w http.ResponseWriter, r *http.Request, status int, clientMsg string, err error) {
	attrs := []any{
		slog.Int("status", status),
		slog.String("method", r.Method),
		slog.String("path", r.URL.Path),
		slog.String("request_id", chimw.GetReqID(r.Context())),
		slog.String("client_msg", clientMsg),
	}
	if err != nil {
		attrs = append(attrs, slog.Any("cause", err))
		logger.Error("request error", attrs...)
	} else {
		// No underlying cause: a client-caused 4xx, not a server fault.
		logger.Warn("request error", attrs...)
	}
	WriteJSON(w, status, map[string]string{"error": clientMsg})
}
