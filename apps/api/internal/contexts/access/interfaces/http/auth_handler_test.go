package http

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/access/application"
	"github.com/jjspscl/my/internal/contexts/access/domain"
	"github.com/jjspscl/my/internal/platform/config"
)

type authTestTokenRepo struct {
	token *domain.MagicToken
}

func (r *authTestTokenRepo) Save(_ context.Context, token *domain.MagicToken) error {
	r.token = token
	return nil
}

func (r *authTestTokenRepo) FindByToken(_ context.Context, token string) (*domain.MagicToken, error) {
	if r.token == nil || r.token.Token != token {
		return nil, context.Canceled
	}
	return r.token, nil
}

func (r *authTestTokenRepo) MarkUsed(_ context.Context, _ string, _ time.Time) error { return nil }

type authTestSessionStore struct{}

func (authTestSessionStore) Create(context.Context, string) (string, error) { return "session-id", nil }
func (authTestSessionStore) Get(context.Context, string) (string, error)    { return "user@test.com", nil }
func (authTestSessionStore) Delete(context.Context, string) error           { return nil }

type authTestMailer struct{}

func (authTestMailer) Send(string, string, string) error { return nil }

func jsonBody(body string) *bytes.Reader { return bytes.NewReader([]byte(body)) }

func newAuthTestHandler(t *testing.T, secure bool, ttl time.Duration) (*AuthHandler, *authTestTokenRepo) {
	t.Helper()
	tokenRepo := &authTestTokenRepo{}
	cfg := &config.Config{UserEmail: "user@test.com", WebURL: "http://localhost:5173"}
	svc := application.NewAuthService(tokenRepo, authTestSessionStore{}, authTestMailer{}, cfg)
	return NewAuthHandler(svc, secure, ttl), tokenRepo
}

func TestVerifyTokenCookieAttributes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		secure bool
	}{
		{name: "secure", secure: true},
		{name: "insecure", secure: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handler, tokenRepo := newAuthTestHandler(t, tc.secure, 2*time.Hour)
			token, err := domain.NewMagicToken("user@test.com", time.Hour)
			if err != nil {
				t.Fatal(err)
			}
			tokenRepo.token = token

			req := httptest.NewRequest(http.MethodPost, "/verify", jsonBody(`{"token":"`+token.Token+`"}`))
			res := httptest.NewRecorder()

			handler.VerifyToken(res, req)

			if res.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
			}
			cookies := res.Result().Cookies()
			if len(cookies) != 2 {
				t.Fatalf("cookies = %d, want 2", len(cookies))
			}
			for _, cookie := range cookies {
				if cookie.Secure != tc.secure {
					t.Errorf("%s Secure = %v, want %v", cookie.Name, cookie.Secure, tc.secure)
				}
				if cookie.MaxAge != int((2 * time.Hour).Seconds()) {
					t.Errorf("%s MaxAge = %d, want %d", cookie.Name, cookie.MaxAge, int((2 * time.Hour).Seconds()))
				}
			}
		})
	}
}
