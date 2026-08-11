package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jjspscl/my/internal/contexts/access/domain"
	"github.com/jjspscl/my/internal/platform/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- mocks ----

type mockTokenRepo struct {
	tokens map[string]*domain.MagicToken
	saveFn func(ctx context.Context, token *domain.MagicToken) error
}

func newMockTokenRepo() *mockTokenRepo {
	return &mockTokenRepo{
		tokens: make(map[string]*domain.MagicToken),
	}
}

func (m *mockTokenRepo) Save(ctx context.Context, token *domain.MagicToken) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, token)
	}
	m.tokens[token.Token] = token
	return nil
}

func (m *mockTokenRepo) FindByToken(ctx context.Context, tokenStr string) (*domain.MagicToken, error) {
	t, ok := m.tokens[tokenStr]
	if !ok {
		return nil, errors.New("not found")
	}
	return t, nil
}

func (m *mockTokenRepo) MarkUsed(ctx context.Context, token string, usedAt time.Time) error {
	t, ok := m.tokens[token]
	if !ok {
		return errors.New("not found")
	}
	t.UsedAt = &usedAt
	return nil
}

type mockMailer struct {
	sentTo string
	sendFn func(to, subject, body string) error
}

func (m *mockMailer) Send(to, subject, body string) error {
	if m.sendFn != nil {
		return m.sendFn(to, subject, body)
	}
	m.sentTo = to
	return nil
}

type mockSessionStore struct {
	sessions map[string]string
	createFn func(ctx context.Context, email string) (string, error)
	getFn    func(ctx context.Context, sessionID string) (string, error)
}

func newMockSessionStore() *mockSessionStore {
	return &mockSessionStore{
		sessions: make(map[string]string),
	}
}

func (m *mockSessionStore) Create(ctx context.Context, email string) (string, error) {
	if m.createFn != nil {
		return m.createFn(ctx, email)
	}
	id := "sess-" + email
	m.sessions[id] = email
	return id, nil
}

func (m *mockSessionStore) Get(ctx context.Context, sessionID string) (string, error) {
	if m.getFn != nil {
		return m.getFn(ctx, sessionID)
	}
	email, ok := m.sessions[sessionID]
	if !ok {
		return "", errors.New("session not found")
	}
	return email, nil
}

func (m *mockSessionStore) Delete(ctx context.Context, sessionID string) error {
	delete(m.sessions, sessionID)
	return nil
}

func newTestConfig() *config.Config {
	return &config.Config{
		UserEmail: "user@test.com",
		WebURL:    "http://localhost:5173",
	}
}

// ---- tests ----

func TestRequestMagicLink_ValidEmail_StoresTokenAndSendsEmail(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	mailer := &mockMailer{}
	sessions := newMockSessionStore()
	cfg := newTestConfig()

	svc := NewAuthService(tokenRepo, sessions, mailer, cfg)

	ctx := context.Background()
	err := svc.RequestMagicLink(ctx, "user@test.com")
	assert.NoError(t, err)

	// Token should be saved
	assert.Len(t, tokenRepo.tokens, 1)

	// Email should be sent
	assert.Equal(t, "user@test.com", mailer.sentTo)
}

func TestRequestMagicLink_WrongEmail_SilentlyReturnsNil(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	mailer := &mockMailer{}
	sessions := newMockSessionStore()
	cfg := newTestConfig()

	svc := NewAuthService(tokenRepo, sessions, mailer, cfg)

	ctx := context.Background()
	err := svc.RequestMagicLink(ctx, "wrong@evil.com")
	assert.NoError(t, err)

	// No token saved, no email sent
	assert.Len(t, tokenRepo.tokens, 0)
	assert.Empty(t, mailer.sentTo)
}

func TestRequestMagicLink_MailFailure_ReturnsError(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	mailer := &mockMailer{
		sendFn: func(to, subject, body string) error {
			return errors.New("SMTP connection refused")
		},
	}
	sessions := newMockSessionStore()
	cfg := newTestConfig()

	svc := NewAuthService(tokenRepo, sessions, mailer, cfg)

	ctx := context.Background()
	err := svc.RequestMagicLink(ctx, "user@test.com")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "send email")
}

func TestVerifyToken_ValidToken_ReturnsEmailAndCreatesSession(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	mailer := &mockMailer{}
	sessions := newMockSessionStore()
	cfg := newTestConfig()

	svc := NewAuthService(tokenRepo, sessions, mailer, cfg)

	// First request a magic link to create a token
	ctx := context.Background()
	err := svc.RequestMagicLink(ctx, "user@test.com")
	assert.NoError(t, err)

	// Get the token string from the repo
	var tokenStr string
	for k := range tokenRepo.tokens {
		tokenStr = k
		break
	}

	// Verify the token
	sessionID, err := svc.VerifyToken(ctx, tokenStr)
	assert.NoError(t, err)
	assert.NotEmpty(t, sessionID)

	// Token should be marked used
	token, _ := tokenRepo.FindByToken(ctx, tokenStr)
	assert.True(t, token.IsUsed())

	// Session should be created
	email, err := sessions.Get(ctx, sessionID)
	assert.NoError(t, err)
	assert.Equal(t, "user@test.com", email)
}

func TestVerifyToken_ExpiredToken_ReturnsError(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	mailer := &mockMailer{}
	sessions := newMockSessionStore()
	cfg := newTestConfig()

	svc := NewAuthService(tokenRepo, sessions, mailer, cfg)

	ctx := context.Background()

	// Manually create an expired token
	expiredToken, err := domain.NewMagicToken("user@test.com", -1*time.Minute)
	assert.NoError(t, err)
	tokenRepo.tokens[expiredToken.Token] = expiredToken

	_, err = svc.VerifyToken(ctx, expiredToken.Token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestVerifyToken_NonExistentToken_ReturnsError(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	mailer := &mockMailer{}
	sessions := newMockSessionStore()
	cfg := newTestConfig()

	svc := NewAuthService(tokenRepo, sessions, mailer, cfg)

	ctx := context.Background()
	_, err := svc.VerifyToken(ctx, "nonexistent-token")
	assert.Error(t, err)
}

func TestLogout_DeletesSession(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	mailer := &mockMailer{}
	sessions := newMockSessionStore()
	cfg := newTestConfig()

	svc := NewAuthService(tokenRepo, sessions, mailer, cfg)

	ctx := context.Background()

	// Create a session
	sessionID, err := sessions.Create(ctx, "user@test.com")
	assert.NoError(t, err)

	// Logout should not error
	err = svc.Logout(ctx, sessionID)
	assert.NoError(t, err)

	// Session should be gone
	_, err = sessions.Get(ctx, sessionID)
	assert.Error(t, err)
}

func TestGetCurrentUser_ValidSession_ReturnsEmail(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	mailer := &mockMailer{}
	sessions := newMockSessionStore()
	cfg := newTestConfig()

	svc := NewAuthService(tokenRepo, sessions, mailer, cfg)

	ctx := context.Background()

	sessionID, err := sessions.Create(ctx, "user@test.com")
	assert.NoError(t, err)

	email, err := svc.GetCurrentUser(ctx, sessionID)
	assert.NoError(t, err)
	assert.Equal(t, "user@test.com", email)
}

func TestGetCurrentUser_InvalidSession_ReturnsError(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	mailer := &mockMailer{}
	sessions := newMockSessionStore()
	cfg := newTestConfig()

	svc := NewAuthService(tokenRepo, sessions, mailer, cfg)

	ctx := context.Background()
	_, err := svc.GetCurrentUser(ctx, "invalid-session")
	assert.Error(t, err)
}

func TestVerifyToken_UsedToken_ReturnsError(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	mailer := &mockMailer{}
	sessions := newMockSessionStore()
	cfg := newTestConfig()

	svc := NewAuthService(tokenRepo, sessions, mailer, cfg)

	ctx := context.Background()

	// Create token and mark as used
	token, err := domain.NewMagicToken("user@test.com", 15*time.Minute)
	assert.NoError(t, err)
	token.MarkUsed()
	tokenRepo.tokens[token.Token] = token

	_, err = svc.VerifyToken(ctx, token.Token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "used")
}

func TestCreateMagicLink_MintsTokenWithoutEmail(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	mailer := &mockMailer{}
	sessions := newMockSessionStore()
	cfg := newTestConfig()
	svc := NewAuthService(tokenRepo, sessions, mailer, cfg)

	link, err := svc.CreateMagicLink(context.Background(), "user@test.com")
	require.NoError(t, err)
	assert.Contains(t, link, "http://localhost:5173/auth/verify?token=")
	assert.Len(t, tokenRepo.tokens, 1, "token persisted for verification")
	assert.Empty(t, mailer.sentTo, "no email sent")
}

func TestCreateMagicLink_MismatchedEmailErrors(t *testing.T) {
	tokenRepo := newMockTokenRepo()
	mailer := &mockMailer{}
	cfg := newTestConfig()
	svc := NewAuthService(tokenRepo, &mockSessionStore{}, mailer, cfg)

	_, err := svc.CreateMagicLink(context.Background(), "other@test.com")
	assert.ErrorContains(t, err, "does not match")
	assert.Len(t, tokenRepo.tokens, 0)
}
