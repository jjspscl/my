package domain

import "context"

// IntelligenceRepository persists provider profiles, encrypted credentials,
// MCP connectors, agent runs, and suggestions. Every method honors the
// transaction coordinator via executor-style context plumbing where relevant.
type IntelligenceRepository interface {
	// Providers
	SaveProvider(ctx context.Context, p *ProviderProfile) error
	FindProvider(ctx context.Context, id, userEmail string) (*ProviderProfile, error)
	ListProviders(ctx context.Context, userEmail string) ([]*ProviderProfile, error)
	UpdateProvider(ctx context.Context, p *ProviderProfile) error
	DeleteProvider(ctx context.Context, id, userEmail string) error

	// Credentials
	SaveCredential(ctx context.Context, c *Credential) error
	FindCredential(ctx context.Context, subjectType, subjectID string) (*Credential, error)
	DeleteCredential(ctx context.Context, subjectType, subjectID string) error

	// Connectors
	SaveConnector(ctx context.Context, c *MCPConnector) error
	FindConnector(ctx context.Context, id, userEmail string) (*MCPConnector, error)
	ListConnectors(ctx context.Context, userEmail string) ([]*MCPConnector, error)
	UpdateConnector(ctx context.Context, c *MCPConnector) error
	DeleteConnector(ctx context.Context, id, userEmail string) error

	// Runs + suggestions
	SaveRun(ctx context.Context, r *AgentRun) error
	FindRun(ctx context.Context, id, userEmail string) (*AgentRun, error)
	ListRunsByScope(ctx context.Context, userEmail, scope, scopeID string, limit int) ([]*AgentRun, error)
	SaveSuggestions(ctx context.Context, suggestions []*Suggestion) error
	ListSuggestionsByRun(ctx context.Context, runID string) ([]*Suggestion, error)
	ListSuggestionsByScope(ctx context.Context, scopeID string) ([]*Suggestion, error)
	UpdateSuggestionStatus(ctx context.Context, id, status string) error
}
