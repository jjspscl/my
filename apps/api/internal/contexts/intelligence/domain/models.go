package domain

import (
	"fmt"
	"strings"
	"time"
)

// Provider types.
const (
	ProviderOpenAI           = "openai"
	ProviderOpenAICompatible = "openai_compatible"
	ProviderOllama           = "ollama"
	ProviderCodexCLI         = "codex_cli"
)

// Run scopes.
const (
	ScopeFinanceImportAnalysis = "finance_import_analysis"
)

// Run statuses.
const (
	RunRunning   = "running"
	RunSucceeded = "succeeded"
	RunFailed    = "failed"
	RunCancelled = "cancelled"
)

// Suggestion fields.
const (
	FieldCategory     = "category"
	FieldMerchant     = "merchant"
	FieldTransfer     = "transfer"
	FieldRelationship = "relationship"
)

// Suggestion statuses.
const (
	SuggestionPending  = "pending"
	SuggestionAccepted = "accepted"
	SuggestionRejected = "rejected"
)

// Limits.
const (
	MaxProviderNameLen = 100
	MaxModelLen        = 200
	MaxBaseURLLen      = 500
	MaxEndpointLen     = 500
	MaxConnectorName   = 100
	MaxRunErrorLen     = 1000
)

// ProviderProfile is a configured model provider. It never carries the
// credential — secrets live in the credential store, keyed by profile ID.
type ProviderProfile struct {
	ID           string
	UserEmail    string
	Name         string
	ProviderType string
	BaseURL      string
	Model        string
	Enabled      bool
	Priority     int
	MaxTokens    int
	Timeout      time.Duration
	AllowLocal   bool // permit loopback endpoints (local providers)
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewProviderProfile validates a provider profile.
func NewProviderProfile(id, userEmail, name, providerType, baseURL, model string, maxTokens int, timeout time.Duration, allowLocal bool) (*ProviderProfile, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if userEmail == "" {
		return nil, fmt.Errorf("user email is required")
	}
	if strings.TrimSpace(name) == "" || len(name) > MaxProviderNameLen {
		return nil, fmt.Errorf("name is required (max %d)", MaxProviderNameLen)
	}
	switch providerType {
	case ProviderOpenAI, ProviderOpenAICompatible, ProviderOllama, ProviderCodexCLI:
	default:
		return nil, fmt.Errorf("invalid provider type: %s", providerType)
	}
	if providerType == ProviderCodexCLI {
		baseURL = "" // CLI credentials live in Codex home, not here
	}
	if baseURL != "" {
		if len(baseURL) > MaxBaseURLLen {
			return nil, fmt.Errorf("base url too long (max %d)", MaxBaseURLLen)
		}
	}
	if strings.TrimSpace(model) == "" || len(model) > MaxModelLen {
		return nil, fmt.Errorf("model is required (max %d)", MaxModelLen)
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &ProviderProfile{
		ID:           id,
		UserEmail:    userEmail,
		Name:         strings.TrimSpace(name),
		ProviderType: providerType,
		BaseURL:      baseURL,
		Model:        strings.TrimSpace(model),
		Enabled:      true,
		MaxTokens:    maxTokens,
		Timeout:      timeout,
		AllowLocal:   allowLocal,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}, nil
}

// Credential is a write-only encrypted secret. The plaintext exists only in
// memory during use.
type Credential struct {
	ID          string
	UserEmail   string
	SubjectType string // provider | connector
	SubjectID   string
	KeyVersion  int
	Ciphertext  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// MCPConnector is an outbound search connector (Brave, Exa, …). Only the
// allowlisted tool names may be invoked, and only search-like tools.
type MCPConnector struct {
	ID        string
	UserEmail string
	Name      string
	Endpoint  string
	Enabled   bool
	Allowlist []string
	Timeout   time.Duration
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NewMCPConnector validates a connector.
func NewMCPConnector(id, userEmail, name, endpoint string, allowlist []string, timeout time.Duration) (*MCPConnector, error) {
	if id == "" {
		return nil, fmt.Errorf("id is required")
	}
	if userEmail == "" {
		return nil, fmt.Errorf("user email is required")
	}
	if strings.TrimSpace(name) == "" || len(name) > MaxConnectorName {
		return nil, fmt.Errorf("name is required (max %d)", MaxConnectorName)
	}
	if endpoint == "" || len(endpoint) > MaxEndpointLen {
		return nil, fmt.Errorf("endpoint is required (max %d)", MaxEndpointLen)
	}
	if len(allowlist) == 0 {
		return nil, fmt.Errorf("at least one allowlisted tool is required")
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	return &MCPConnector{
		ID:        id,
		UserEmail: userEmail,
		Name:      strings.TrimSpace(name),
		Endpoint:  endpoint,
		Enabled:   true,
		Allowlist: allowlist,
		Timeout:   timeout,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}, nil
}

// EvidenceSource is where a suggestion's evidence came from.
type EvidenceSource string

const (
	EvidenceRule     EvidenceSource = "rule"    // exact prior/user-approved mapping
	EvidenceHistory  EvidenceSource = "history" // agreement with known transactions
	EvidenceModel    EvidenceSource = "model"   // LLM alone
	EvidenceWeb      EvidenceSource = "web"     // search result corroboration
	EvidenceUserRule EvidenceSource = "user_rule"
)

// Evidence is one piece of evidence behind a suggestion.
type Evidence struct {
	Source EvidenceSource `json:"source"`
	Detail string         `json:"detail"`
}

// AgentRun is one analysis execution.
type AgentRun struct {
	ID            string
	UserEmail     string
	Scope         string
	ScopeID       string
	ProviderID    string
	Status        string
	Model         string
	PromptVersion string
	InputSummary  string
	OutputSummary string
	TokenUsage    string
	DurationMS    int64
	Error         string
	CreatedAt     time.Time
	CompletedAt   *time.Time
}

// NewAgentRun starts a run.
func NewAgentRun(id, userEmail, scope, scopeID, providerID string) (*AgentRun, error) {
	if id == "" || userEmail == "" || scope == "" || scopeID == "" {
		return nil, fmt.Errorf("id, user email, scope, and scope id are required")
	}
	return &AgentRun{
		ID:         id,
		UserEmail:  userEmail,
		Scope:      scope,
		ScopeID:    scopeID,
		ProviderID: providerID,
		Status:     RunRunning,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

// Suggestion is one calibrated inference for one target/field.
type Suggestion struct {
	ID         string
	RunID      string
	ScopeID    string
	TargetKey  string
	Field      string
	Value      string
	Confidence float64
	Rationale  string
	Evidence   []Evidence
	Status     string
	CreatedAt  time.Time
}

// NewSuggestion validates a suggestion. Confidence must already be calibrated
// by the confidence engine before this is called.
func NewSuggestion(id, runID, scopeID, targetKey, field, value, rationale string, confidence float64, evidence []Evidence) (*Suggestion, error) {
	if id == "" || runID == "" {
		return nil, fmt.Errorf("id and run id are required")
	}
	if targetKey == "" {
		return nil, fmt.Errorf("target key is required")
	}
	switch field {
	case FieldCategory, FieldMerchant, FieldTransfer, FieldRelationship:
	default:
		return nil, fmt.Errorf("invalid suggestion field: %s", field)
	}
	if strings.TrimSpace(value) == "" {
		return nil, fmt.Errorf("value is required")
	}
	if confidence < 0 || confidence > 1 {
		return nil, fmt.Errorf("confidence must be in [0,1]")
	}
	if len(value) > 500 {
		return nil, fmt.Errorf("value too long")
	}
	return &Suggestion{
		ID:         id,
		RunID:      runID,
		ScopeID:    scopeID,
		TargetKey:  targetKey,
		Field:      field,
		Value:      strings.TrimSpace(value),
		Confidence: confidence,
		Rationale:  rationale,
		Evidence:   evidence,
		Status:     SuggestionPending,
		CreatedAt:  time.Now().UTC(),
	}, nil
}
