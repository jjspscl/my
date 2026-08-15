package application

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
	"github.com/jjspscl/my/internal/contexts/intelligence/infrastructure"
	"github.com/jjspscl/my/internal/contexts/intelligence/infrastructure/providers"
)

// RuntimeConfig carries values that are environment-held rather than stored in
// the database (feature gate, master key, codex path).
type RuntimeConfig struct {
	LLMEnabled bool
	CodexPath  string
}

// SettingsService manages provider profiles, encrypted credentials, and MCP
// connectors. Secrets are write-only: plaintext never leaves the service and
// is never returned over HTTP. A nil box means the master key is not
// configured: credential writes fail closed, everything else still works.
type SettingsService struct {
	repo    domain.IntelligenceRepository
	box     *infrastructure.SecretBox
	runtime RuntimeConfig
}

func NewSettingsService(repo domain.IntelligenceRepository, box *infrastructure.SecretBox, runtime RuntimeConfig) *SettingsService {
	return &SettingsService{repo: repo, box: box, runtime: runtime}
}

// MasterKeyConfigured reports whether encrypted credentials are usable.
func (s *SettingsService) MasterKeyConfigured() bool {
	return s.box != nil
}

// ---- providers ----

type CreateProviderInput struct {
	Name         string
	ProviderType string
	BaseURL      string
	Model        string
	MaxTokens    int
	TimeoutMS    int
	AllowLocal   bool
	APIKey       string // optional; empty keeps no credential
}

func (s *SettingsService) CreateProvider(ctx context.Context, userEmail string, in CreateProviderInput) (*domain.ProviderProfile, error) {
	timeout := ms(in.TimeoutMS, 30000)
	p, err := domain.NewProviderProfile(uuid.New().String(), userEmail, in.Name, in.ProviderType, in.BaseURL, in.Model, in.MaxTokens, timeout, in.AllowLocal)
	if err != nil {
		return nil, err
	}
	if p.BaseURL != "" {
		if err := infrastructure.ValidateEndpoint(p.BaseURL, p.AllowLocal); err != nil {
			return nil, fmt.Errorf("base url: %w", err)
		}
	}
	if err := s.repo.SaveProvider(ctx, p); err != nil {
		return nil, err
	}
	if in.APIKey != "" {
		if err := s.SaveCredential(ctx, userEmail, "provider", p.ID, in.APIKey); err != nil {
			return nil, err
		}
	}
	return p, nil
}

func (s *SettingsService) ListProviders(ctx context.Context, userEmail string) ([]*domain.ProviderProfile, error) {
	return s.repo.ListProviders(ctx, userEmail)
}

type UpdateProviderInput struct {
	ID           string
	Name         string
	ProviderType string
	BaseURL      string
	Model        string
	MaxTokens    int
	TimeoutMS    int
	AllowLocal   bool
	Enabled      bool
}

func (s *SettingsService) UpdateProvider(ctx context.Context, userEmail string, in UpdateProviderInput) (*domain.ProviderProfile, error) {
	existing, err := s.repo.FindProvider(ctx, in.ID, userEmail)
	if err != nil {
		return nil, err
	}
	timeout := ms(in.TimeoutMS, int(existing.Timeout.Milliseconds()))
	p, err := domain.NewProviderProfile(existing.ID, userEmail, in.Name, in.ProviderType, in.BaseURL, in.Model, in.MaxTokens, timeout, in.AllowLocal)
	if err != nil {
		return nil, err
	}
	if p.BaseURL != "" {
		if err := infrastructure.ValidateEndpoint(p.BaseURL, p.AllowLocal); err != nil {
			return nil, fmt.Errorf("base url: %w", err)
		}
	}
	p.Enabled = in.Enabled
	p.CreatedAt = existing.CreatedAt
	if err := s.repo.UpdateProvider(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (s *SettingsService) DeleteProvider(ctx context.Context, userEmail, id string) error {
	if err := s.repo.DeleteProvider(ctx, id, userEmail); err != nil {
		return err
	}
	return s.repo.DeleteCredential(ctx, "provider", id)
}

// SaveCredential encrypts and upserts a secret for a subject. Plaintext is
// zeroed after use where practical.
func (s *SettingsService) SaveCredential(ctx context.Context, userEmail, subjectType, subjectID, plaintext string) error {
	if s.box == nil {
		return fmt.Errorf("master key is not configured (set MY_LLM_MASTER_KEY)")
	}
	ciphertext, err := s.box.Encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("encrypt credential: %w", err)
	}
	return s.repo.SaveCredential(ctx, &domain.Credential{
		ID:          uuid.New().String(),
		UserEmail:   userEmail,
		SubjectType: subjectType,
		SubjectID:   subjectID,
		KeyVersion:  infrastructure.CurrentKeyVersion,
		Ciphertext:  ciphertext,
		CreatedAt:   now(),
		UpdatedAt:   now(),
	})
}

// TestProvider performs a live capability check against a configured provider.
func (s *SettingsService) TestProvider(ctx context.Context, userEmail, id string) error {
	p, err := s.repo.FindProvider(ctx, id, userEmail)
	if err != nil {
		return err
	}
	cred, err := s.decryptCredential(ctx, "provider", id)
	if err != nil {
		return fmt.Errorf("credential: %w", err)
	}
	provider, err := providers.New(p, cred, providers.Config{CodexPath: s.runtime.CodexPath})
	if err != nil {
		return err
	}
	_, err = provider.Complete(ctx, providers.ChatRequest{
		System: "Reply with exactly: ok",
		User:   "Ping",
	})
	return err
}

// ---- connectors ----

type CreateConnectorInput struct {
	Name      string
	Endpoint  string
	Allowlist []string
	TimeoutMS int
	Token     string // optional
}

func (s *SettingsService) CreateConnector(ctx context.Context, userEmail string, in CreateConnectorInput) (*domain.MCPConnector, error) {
	timeout := ms(in.TimeoutMS, 15000)
	c, err := domain.NewMCPConnector(uuid.New().String(), userEmail, in.Name, in.Endpoint, in.Allowlist, timeout)
	if err != nil {
		return nil, err
	}
	if err := infrastructure.ValidateEndpoint(c.Endpoint, false); err != nil {
		return nil, fmt.Errorf("endpoint: %w", err)
	}
	if err := s.repo.SaveConnector(ctx, c); err != nil {
		return nil, err
	}
	if in.Token != "" {
		if err := s.SaveCredential(ctx, userEmail, "connector", c.ID, in.Token); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (s *SettingsService) ListConnectors(ctx context.Context, userEmail string) ([]*domain.MCPConnector, error) {
	return s.repo.ListConnectors(ctx, userEmail)
}

type UpdateConnectorInput struct {
	ID        string
	Name      string
	Endpoint  string
	Allowlist []string
	TimeoutMS int
	Enabled   bool
}

func (s *SettingsService) UpdateConnector(ctx context.Context, userEmail string, in UpdateConnectorInput) (*domain.MCPConnector, error) {
	existing, err := s.repo.FindConnector(ctx, in.ID, userEmail)
	if err != nil {
		return nil, err
	}
	timeout := ms(in.TimeoutMS, int(existing.Timeout.Milliseconds()))
	c, err := domain.NewMCPConnector(existing.ID, userEmail, in.Name, in.Endpoint, in.Allowlist, timeout)
	if err != nil {
		return nil, err
	}
	if err := infrastructure.ValidateEndpoint(c.Endpoint, false); err != nil {
		return nil, fmt.Errorf("endpoint: %w", err)
	}
	c.Enabled = in.Enabled
	c.CreatedAt = existing.CreatedAt
	if err := s.repo.UpdateConnector(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *SettingsService) DeleteConnector(ctx context.Context, userEmail, id string) error {
	if err := s.repo.DeleteConnector(ctx, id, userEmail); err != nil {
		return err
	}
	return s.repo.DeleteCredential(ctx, "connector", id)
}

// TestConnector dials the connector and invokes the first allowlisted tool
// (read-only probe).
func (s *SettingsService) TestConnector(ctx context.Context, userEmail, id string) error {
	c, err := s.repo.FindConnector(ctx, id, userEmail)
	if err != nil {
		return err
	}
	if len(c.Allowlist) == 0 {
		return fmt.Errorf("connector has no allowlisted tools")
	}
	cred, err := s.decryptCredential(ctx, "connector", id)
	if err != nil {
		return fmt.Errorf("credential: %w", err)
	}
	gateway := infrastructure.NewMCPGateway()
	_, err = gateway.Call(ctx, c, cred, infrastructure.ToolCall{
		Name:      c.Allowlist[0],
		Arguments: map[string]any{"query": "test"},
	})
	return err
}

// HasCredential reports whether a subject has a stored (encrypted) credential.
func (s *SettingsService) HasCredential(ctx context.Context, subjectType, subjectID string) bool {
	cred, err := s.repo.FindCredential(ctx, subjectType, subjectID)
	return err == nil && cred != nil
}

func (s *SettingsService) decryptCredential(ctx context.Context, subjectType, subjectID string) (string, error) {
	if s.box == nil {
		return "", fmt.Errorf("master key is not configured (set MY_LLM_MASTER_KEY)")
	}
	cred, err := s.repo.FindCredential(ctx, subjectType, subjectID)
	if err != nil {
		return "", err
	}
	return s.box.Decrypt(cred.Ciphertext)
}

// ResolveProvider returns the highest-priority enabled provider that can run
// right now (has a credential, or is the CLI adapter with a configured path).
func (s *SettingsService) ResolveProvider(ctx context.Context, userEmail string) (*domain.ProviderProfile, string, error) {
	profiles, err := s.repo.ListProviders(ctx, userEmail)
	if err != nil {
		return nil, "", err
	}
	for i := range profiles {
		p := profiles[i]
		if !p.Enabled {
			continue
		}
		if p.ProviderType == domain.ProviderCodexCLI {
			if s.runtime.CodexPath != "" {
				return p, "", nil
			}
			continue
		}
		cred, err := s.decryptCredential(ctx, "provider", p.ID)
		if err == nil && cred != "" {
			return p, cred, nil
		}
	}
	return nil, "", fmt.Errorf("no enabled provider with a credential is configured")
}

func (s *SettingsService) BuildProvider(ctx context.Context, p *domain.ProviderProfile, credential string) (providers.Provider, error) {
	return providers.New(p, credential, providers.Config{CodexPath: s.runtime.CodexPath})
}

func ms(v, fallback int) time.Duration {
	if v <= 0 {
		return time.Duration(fallback) * time.Millisecond
	}
	return time.Duration(v) * time.Millisecond
}

func now() time.Time { return time.Now().UTC() }
