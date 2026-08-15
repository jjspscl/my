package infrastructure

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jjspscl/my/internal/contexts/intelligence/domain"
)

type IntelligenceRepoLibSQL struct {
	db *sql.DB
}

func NewIntelligenceRepoLibSQL(db *sql.DB) *IntelligenceRepoLibSQL {
	return &IntelligenceRepoLibSQL{db: db}
}

func (r *IntelligenceRepoLibSQL) SaveProvider(ctx context.Context, p *domain.ProviderProfile) error {
	cfg, _ := json.Marshal(map[string]bool{"allowLocal": p.AllowLocal})
	_, err := executor(ctx, r.db).ExecContext(ctx,
		`INSERT INTO intelligence_provider_profiles
		 (id, user_email, name, provider_type, base_url, model, enabled, priority, max_tokens, timeout_ms, config_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.UserEmail, p.Name, p.ProviderType, nullableString(optionalString(p.BaseURL)),
		p.Model, boolInt(p.Enabled), p.Priority, nullableInt(p.MaxTokens),
		int(p.Timeout.Milliseconds()), string(cfg), p.CreatedAt.Format(time.RFC3339), p.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save provider profile: %w", err)
	}
	return nil
}

func (r *IntelligenceRepoLibSQL) FindProvider(ctx context.Context, id, userEmail string) (*domain.ProviderProfile, error) {
	row := executor(ctx, r.db).QueryRowContext(ctx,
		`SELECT id, user_email, name, provider_type, base_url, model, enabled, priority, max_tokens, timeout_ms, config_json, created_at, updated_at
		 FROM intelligence_provider_profiles WHERE id = ? AND user_email = ?`, id, userEmail)
	return scanProvider(row)
}

func (r *IntelligenceRepoLibSQL) ListProviders(ctx context.Context, userEmail string) ([]*domain.ProviderProfile, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx,
		`SELECT id, user_email, name, provider_type, base_url, model, enabled, priority, max_tokens, timeout_ms, config_json, created_at, updated_at
		 FROM intelligence_provider_profiles WHERE user_email = ? ORDER BY priority, name`, userEmail)
	if err != nil {
		return nil, fmt.Errorf("list provider profiles: %w", err)
	}
	defer rows.Close()
	var out []*domain.ProviderProfile
	for rows.Next() {
		p, err := scanProvider(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *IntelligenceRepoLibSQL) UpdateProvider(ctx context.Context, p *domain.ProviderProfile) error {
	cfg, _ := json.Marshal(map[string]bool{"allowLocal": p.AllowLocal})
	_, err := executor(ctx, r.db).ExecContext(ctx,
		`UPDATE intelligence_provider_profiles SET name=?, provider_type=?, base_url=?, model=?, enabled=?, priority=?, max_tokens=?, timeout_ms=?, config_json=?, updated_at=?
		 WHERE id = ? AND user_email = ?`,
		p.Name, p.ProviderType, nullableString(optionalString(p.BaseURL)), p.Model,
		boolInt(p.Enabled), p.Priority, nullableInt(p.MaxTokens), int(p.Timeout.Milliseconds()),
		string(cfg), p.UpdatedAt.Format(time.RFC3339), p.ID, p.UserEmail)
	if err != nil {
		return fmt.Errorf("update provider profile: %w", err)
	}
	return nil
}

func (r *IntelligenceRepoLibSQL) DeleteProvider(ctx context.Context, id, userEmail string) error {
	if _, err := executor(ctx, r.db).ExecContext(ctx,
		`DELETE FROM intelligence_provider_profiles WHERE id = ? AND user_email = ?`, id, userEmail); err != nil {
		return fmt.Errorf("delete provider profile: %w", err)
	}
	return nil
}

func (r *IntelligenceRepoLibSQL) SaveCredential(ctx context.Context, c *domain.Credential) error {
	_, err := executor(ctx, r.db).ExecContext(ctx,
		`INSERT INTO intelligence_credentials (id, user_email, subject_type, subject_id, key_version, ciphertext, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(subject_type, subject_id) DO UPDATE SET ciphertext=excluded.ciphertext, key_version=excluded.key_version, updated_at=excluded.updated_at`,
		c.ID, c.UserEmail, c.SubjectType, c.SubjectID, c.KeyVersion, c.Ciphertext,
		c.CreatedAt.Format(time.RFC3339), c.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save credential: %w", err)
	}
	return nil
}

func (r *IntelligenceRepoLibSQL) FindCredential(ctx context.Context, subjectType, subjectID string) (*domain.Credential, error) {
	row := executor(ctx, r.db).QueryRowContext(ctx,
		`SELECT id, user_email, subject_type, subject_id, key_version, ciphertext, created_at, updated_at
		 FROM intelligence_credentials WHERE subject_type = ? AND subject_id = ?`, subjectType, subjectID)
	return scanCredential(row)
}

func (r *IntelligenceRepoLibSQL) DeleteCredential(ctx context.Context, subjectType, subjectID string) error {
	_, err := executor(ctx, r.db).ExecContext(ctx,
		`DELETE FROM intelligence_credentials WHERE subject_type = ? AND subject_id = ?`, subjectType, subjectID)
	if err != nil {
		return fmt.Errorf("delete credential: %w", err)
	}
	return nil
}

func (r *IntelligenceRepoLibSQL) SaveConnector(ctx context.Context, c *domain.MCPConnector) error {
	allowlist, _ := json.Marshal(c.Allowlist)
	_, err := executor(ctx, r.db).ExecContext(ctx,
		`INSERT INTO intelligence_mcp_connectors (id, user_email, name, endpoint, connector_kind, auth_type, enabled, allowlist_json, timeout_ms, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.UserEmail, c.Name, c.Endpoint, c.Kind, c.AuthType, boolInt(c.Enabled), string(allowlist),
		int(c.Timeout.Milliseconds()), c.CreatedAt.Format(time.RFC3339), c.UpdatedAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("save connector: %w", err)
	}
	return nil
}

func (r *IntelligenceRepoLibSQL) FindConnector(ctx context.Context, id, userEmail string) (*domain.MCPConnector, error) {
	row := executor(ctx, r.db).QueryRowContext(ctx,
		`SELECT id, user_email, name, endpoint, connector_kind, auth_type, enabled, allowlist_json, timeout_ms, created_at, updated_at
		 FROM intelligence_mcp_connectors WHERE id = ? AND user_email = ?`, id, userEmail)
	return scanConnector(row)
}

func (r *IntelligenceRepoLibSQL) ListConnectors(ctx context.Context, userEmail string) ([]*domain.MCPConnector, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx,
		`SELECT id, user_email, name, endpoint, connector_kind, auth_type, enabled, allowlist_json, timeout_ms, created_at, updated_at
		 FROM intelligence_mcp_connectors WHERE user_email = ? ORDER BY name`, userEmail)
	if err != nil {
		return nil, fmt.Errorf("list connectors: %w", err)
	}
	defer rows.Close()
	var out []*domain.MCPConnector
	for rows.Next() {
		c, err := scanConnector(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *IntelligenceRepoLibSQL) UpdateConnector(ctx context.Context, c *domain.MCPConnector) error {
	allowlist, _ := json.Marshal(c.Allowlist)
	_, err := executor(ctx, r.db).ExecContext(ctx,
		`UPDATE intelligence_mcp_connectors SET name=?, endpoint=?, connector_kind=?, auth_type=?, enabled=?, allowlist_json=?, timeout_ms=?, updated_at=?
		 WHERE id = ? AND user_email = ?`,
		c.Name, c.Endpoint, c.Kind, c.AuthType, boolInt(c.Enabled), string(allowlist), int(c.Timeout.Milliseconds()),
		c.UpdatedAt.Format(time.RFC3339), c.ID, c.UserEmail)
	if err != nil {
		return fmt.Errorf("update connector: %w", err)
	}
	return nil
}

func (r *IntelligenceRepoLibSQL) DeleteConnector(ctx context.Context, id, userEmail string) error {
	_, err := executor(ctx, r.db).ExecContext(ctx,
		`DELETE FROM intelligence_mcp_connectors WHERE id = ? AND user_email = ?`, id, userEmail)
	if err != nil {
		return fmt.Errorf("delete connector: %w", err)
	}
	return nil
}

func (r *IntelligenceRepoLibSQL) SaveRun(ctx context.Context, run *domain.AgentRun) error {
	_, err := executor(ctx, r.db).ExecContext(ctx,
		`INSERT INTO intelligence_agent_runs
		 (id, user_email, scope, scope_id, provider_id, status, model, prompt_version,
		  input_summary_json, output_summary_json, token_usage_json, duration_ms, error, created_at, completed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET
		   status=excluded.status, model=excluded.model, prompt_version=excluded.prompt_version,
		   input_summary_json=excluded.input_summary_json, output_summary_json=excluded.output_summary_json,
		   token_usage_json=excluded.token_usage_json, duration_ms=excluded.duration_ms,
		   error=excluded.error, completed_at=excluded.completed_at`,
		run.ID, run.UserEmail, run.Scope, run.ScopeID, nullableString(optionalString(run.ProviderID)),
		run.Status, nullableString(optionalString(run.Model)), nullableString(optionalString(run.PromptVersion)),
		orJSON(run.InputSummary), orJSON(run.OutputSummary), orJSON(run.TokenUsage),
		nullableInt64(run.DurationMS), nullableString(optionalString(run.Error)),
		run.CreatedAt.Format(time.RFC3339), nullableTime(run.CompletedAt))
	if err != nil {
		return fmt.Errorf("save agent run: %w", err)
	}
	return nil
}

func (r *IntelligenceRepoLibSQL) FindRun(ctx context.Context, id, userEmail string) (*domain.AgentRun, error) {
	row := executor(ctx, r.db).QueryRowContext(ctx,
		`SELECT id, user_email, scope, scope_id, provider_id, status, model, prompt_version,
		        input_summary_json, output_summary_json, token_usage_json, duration_ms, error, created_at, completed_at
		 FROM intelligence_agent_runs WHERE id = ? AND user_email = ?`, id, userEmail)
	return scanRun(row)
}

func (r *IntelligenceRepoLibSQL) ListRunsByScope(ctx context.Context, userEmail, scope, scopeID string, limit int) ([]*domain.AgentRun, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx,
		`SELECT id, user_email, scope, scope_id, provider_id, status, model, prompt_version,
		        input_summary_json, output_summary_json, token_usage_json, duration_ms, error, created_at, completed_at
		 FROM intelligence_agent_runs WHERE user_email = ? AND scope = ? AND scope_id = ?
		 ORDER BY created_at DESC LIMIT ?`, userEmail, scope, scopeID, limit)
	if err != nil {
		return nil, fmt.Errorf("list agent runs: %w", err)
	}
	defer rows.Close()
	var out []*domain.AgentRun
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, run)
	}
	return out, rows.Err()
}

func (r *IntelligenceRepoLibSQL) SaveSuggestions(ctx context.Context, suggestions []*domain.Suggestion) error {
	for _, s := range suggestions {
		evidence, _ := json.Marshal(s.Evidence)
		_, err := executor(ctx, r.db).ExecContext(ctx,
			`INSERT INTO intelligence_suggestions
			 (id, run_id, scope_id, target_key, field, value, confidence, rationale, evidence_json, status, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			s.ID, s.RunID, s.ScopeID, s.TargetKey, s.Field, s.Value, s.Confidence,
			nullableString(optionalString(s.Rationale)), string(evidence), s.Status,
			s.CreatedAt.Format(time.RFC3339))
		if err != nil {
			return fmt.Errorf("save suggestion: %w", err)
		}
	}
	return nil
}

func (r *IntelligenceRepoLibSQL) ListSuggestionsByRun(ctx context.Context, runID string) ([]*domain.Suggestion, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx,
		`SELECT id, run_id, scope_id, target_key, field, value, confidence, rationale, evidence_json, status, created_at
		 FROM intelligence_suggestions WHERE run_id = ? ORDER BY created_at`, runID)
	if err != nil {
		return nil, fmt.Errorf("list suggestions: %w", err)
	}
	defer rows.Close()
	var out []*domain.Suggestion
	for rows.Next() {
		s, err := scanSuggestion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *IntelligenceRepoLibSQL) ListSuggestionsByScope(ctx context.Context, scopeID string) ([]*domain.Suggestion, error) {
	rows, err := executor(ctx, r.db).QueryContext(ctx,
		`SELECT id, run_id, scope_id, target_key, field, value, confidence, rationale, evidence_json, status, created_at
		 FROM intelligence_suggestions WHERE scope_id = ? ORDER BY created_at`, scopeID)
	if err != nil {
		return nil, fmt.Errorf("list suggestions by scope: %w", err)
	}
	defer rows.Close()
	var out []*domain.Suggestion
	for rows.Next() {
		s, err := scanSuggestion(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *IntelligenceRepoLibSQL) UpdateSuggestionStatus(ctx context.Context, id, status string) error {
	if _, err := executor(ctx, r.db).ExecContext(ctx,
		`UPDATE intelligence_suggestions SET status = ? WHERE id = ?`, status, id); err != nil {
		return fmt.Errorf("update suggestion status: %w", err)
	}
	return nil
}

// ---- scanners ----

type scanner interface {
	Scan(dest ...any) error
}

func scanProvider(row scanner) (*domain.ProviderProfile, error) {
	var p domain.ProviderProfile
	var baseURL *string
	var cfg, createdAtStr, updatedAtStr string
	var maxTokens *int
	var timeoutMS int
	var enabled int
	if err := row.Scan(&p.ID, &p.UserEmail, &p.Name, &p.ProviderType, &baseURL, &p.Model,
		&enabled, &p.Priority, &maxTokens, &timeoutMS, &cfg, &createdAtStr, &updatedAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("provider profile not found")
		}
		return nil, fmt.Errorf("scan provider profile: %w", err)
	}
	if baseURL != nil {
		p.BaseURL = *baseURL
	}
	if maxTokens != nil {
		p.MaxTokens = *maxTokens
	}
	p.Enabled = enabled == 1
	p.Timeout = time.Duration(timeoutMS) * time.Millisecond
	var cfgMap map[string]bool
	if json.Unmarshal([]byte(cfg), &cfgMap) == nil {
		p.AllowLocal = cfgMap["allowLocal"]
	}
	var err error
	if p.CreatedAt, err = parseTime(createdAtStr); err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	if p.UpdatedAt, err = parseTime(updatedAtStr); err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}
	return &p, nil
}

func scanCredential(row scanner) (*domain.Credential, error) {
	var c domain.Credential
	var createdAtStr, updatedAtStr string
	if err := row.Scan(&c.ID, &c.UserEmail, &c.SubjectType, &c.SubjectID, &c.KeyVersion,
		&c.Ciphertext, &createdAtStr, &updatedAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // absent credential == keyless connector/provider
		}
		return nil, fmt.Errorf("scan credential: %w", err)
	}
	var err error
	if c.CreatedAt, err = parseTime(createdAtStr); err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	if c.UpdatedAt, err = parseTime(updatedAtStr); err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}
	return &c, nil
}

func scanConnector(row scanner) (*domain.MCPConnector, error) {
	var c domain.MCPConnector
	var allowlist, createdAtStr, updatedAtStr string
	var enabled, timeoutMS int
	if err := row.Scan(&c.ID, &c.UserEmail, &c.Name, &c.Endpoint, &c.Kind, &c.AuthType, &enabled, &allowlist,
		&timeoutMS, &createdAtStr, &updatedAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("connector not found")
		}
		return nil, fmt.Errorf("scan connector: %w", err)
	}
	c.Enabled = enabled == 1
	c.Timeout = time.Duration(timeoutMS) * time.Millisecond
	_ = json.Unmarshal([]byte(allowlist), &c.Allowlist)
	var err error
	if c.CreatedAt, err = parseTime(createdAtStr); err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	if c.UpdatedAt, err = parseTime(updatedAtStr); err != nil {
		return nil, fmt.Errorf("parse updated_at: %w", err)
	}
	return &c, nil
}

func scanRun(row scanner) (*domain.AgentRun, error) {
	var r domain.AgentRun
	var providerID, model, promptVersion, inputSummary, outputSummary, tokenUsage, errorMsg *string
	var durationMS *int64
	var completedAtStr *string
	var createdAtStr string
	if err := row.Scan(&r.ID, &r.UserEmail, &r.Scope, &r.ScopeID, &providerID, &r.Status,
		&model, &promptVersion, &inputSummary, &outputSummary, &tokenUsage, &durationMS,
		&errorMsg, &createdAtStr, &completedAtStr); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("agent run not found")
		}
		return nil, fmt.Errorf("scan agent run: %w", err)
	}
	if providerID != nil {
		r.ProviderID = *providerID
	}
	if model != nil {
		r.Model = *model
	}
	if promptVersion != nil {
		r.PromptVersion = *promptVersion
	}
	if inputSummary != nil {
		r.InputSummary = *inputSummary
	}
	if outputSummary != nil {
		r.OutputSummary = *outputSummary
	}
	if tokenUsage != nil {
		r.TokenUsage = *tokenUsage
	}
	if durationMS != nil {
		r.DurationMS = *durationMS
	}
	if errorMsg != nil {
		r.Error = *errorMsg
	}
	var err error
	if r.CreatedAt, err = parseTime(createdAtStr); err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	if completedAtStr != nil {
		t, err := parseTime(*completedAtStr)
		if err != nil {
			return nil, fmt.Errorf("parse completed_at: %w", err)
		}
		r.CompletedAt = &t
	}
	return &r, nil
}

func scanSuggestion(row scanner) (*domain.Suggestion, error) {
	var s domain.Suggestion
	var rationale *string
	var evidenceStr, createdAtStr string
	if err := row.Scan(&s.ID, &s.RunID, &s.ScopeID, &s.TargetKey, &s.Field, &s.Value,
		&s.Confidence, &rationale, &evidenceStr, &s.Status, &createdAtStr); err != nil {
		return nil, fmt.Errorf("scan suggestion: %w", err)
	}
	if rationale != nil {
		s.Rationale = *rationale
	}
	_ = json.Unmarshal([]byte(evidenceStr), &s.Evidence)
	var err error
	if s.CreatedAt, err = parseTime(createdAtStr); err != nil {
		return nil, fmt.Errorf("parse created_at: %w", err)
	}
	return &s, nil
}

// ---- helpers ----

func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullableInt(v int) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullableInt64(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}

func nullableTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return t.Format(time.RFC3339)
}

func orJSON(v string) string {
	if v == "" {
		return "{}"
	}
	return v
}

// optionalString converts an empty string to nil (SQL NULL).
func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func nullableString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}
