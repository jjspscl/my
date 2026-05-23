package middleware

import (
	"context"
	"net/http"
)

type contextKey string

const emailKey contextKey = "email"

func GetEmailFromContext(ctx context.Context) string {
	email, _ := ctx.Value(emailKey).(string)
	return email
}

func RequireAuth(sessions interface {
	Get(ctx context.Context, sessionID string) (string, error)
}) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie("my_session")
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			email, err := sessions.Get(r.Context(), cookie.Value)
			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), emailKey, email)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
