package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/jjspscl/my/internal/shared/response"
)

// Recover catches panics, logs stack trace, returns 500.
func Recover(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("panic recovered",
						slog.Any("error", err),
						slog.String("stack", string(debug.Stack())),
						slog.String("path", r.URL.Path),
						slog.String("method", r.Method),
					)
					response.WriteError(w, r, http.StatusInternalServerError, "internal server error", nil)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
