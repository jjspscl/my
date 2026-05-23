package application

import (
	"context"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/contexts/access/domain"
	"github.com/jjspscl/my/internal/platform/config"
	"github.com/jjspscl/my/internal/platform/mail"
	"github.com/jjspscl/my/internal/platform/session"
)

type AuthService struct {
	tokenRepo domain.TokenRepository
	sessions  session.Store
	mailer    mail.Sender
	config    *config.Config
}

func NewAuthService(
	tokenRepo domain.TokenRepository,
	sessions session.Store,
	mailer mail.Sender,
	config *config.Config,
) *AuthService {
	return &AuthService{
		tokenRepo: tokenRepo,
		sessions:  sessions,
		mailer:    mailer,
		config:    config,
	}
}

func (s *AuthService) RequestMagicLink(ctx context.Context, email string) error {
	if email != s.config.UserEmail {
		// Don't reveal whether the email is registered
		return nil
	}

	token, err := domain.NewMagicToken(email, 15*time.Minute)
	if err != nil {
		return fmt.Errorf("create token: %w", err)
	}

	if err := s.tokenRepo.Save(ctx, token); err != nil {
		return fmt.Errorf("save token: %w", err)
	}

	link := fmt.Sprintf("%s/auth/verify?token=%s", s.config.WebURL, token.Token)
	body := fmt.Sprintf("Sign in to my\n\nClick this link to sign in:\n%s\n\nThis link expires in 15 minutes.\n\nIf you didn't request this, ignore this email.", link)

	if err := s.mailer.Send(email, "Sign in to my", body); err != nil {
		return fmt.Errorf("send email: %w", err)
	}

	return nil
}

func (s *AuthService) VerifyToken(ctx context.Context, tokenStr string) (string, error) {
	token, err := s.tokenRepo.FindByToken(ctx, tokenStr)
	if err != nil {
		return "", fmt.Errorf("invalid token")
	}

	if token.IsExpired() {
		return "", fmt.Errorf("token expired")
	}

	if token.IsUsed() {
		return "", fmt.Errorf("token already used")
	}

	token.MarkUsed()
	if err := s.tokenRepo.MarkUsed(ctx, token.Token, *token.UsedAt); err != nil {
		return "", fmt.Errorf("mark token used: %w", err)
	}

	sessionID, err := s.sessions.Create(ctx, token.Email)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	return sessionID, nil
}

func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	return s.sessions.Delete(ctx, sessionID)
}

func (s *AuthService) GetCurrentUser(ctx context.Context, sessionID string) (string, error) {
	return s.sessions.Get(ctx, sessionID)
}
