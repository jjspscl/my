-- 012_intelligence
-- Confidence-gated LLM analysis: provider profiles, encrypted credentials,
-- outbound MCP search connectors, agent runs, and suggestions.
--
-- Secrets are never stored in plaintext: credentials hold only an
-- AES-256-GCM ciphertext keyed by the environment-held MY_LLM_MASTER_KEY.
-- No PDFs, passwords, raw prompts, or chain-of-thought are persisted.

CREATE TABLE intelligence_provider_profiles (
  id TEXT PRIMARY KEY,
  user_email TEXT NOT NULL,
  name TEXT NOT NULL,
  provider_type TEXT NOT NULL, -- openai | openai_compatible | ollama | codex_cli
  base_url TEXT,
  model TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  priority INTEGER NOT NULL DEFAULT 0,
  max_tokens INTEGER,
  timeout_ms INTEGER NOT NULL DEFAULT 30000,
  config_json TEXT NOT NULL DEFAULT '{}', -- e.g. {"allowLocal":false}
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX idx_intelligence_providers_user ON intelligence_provider_profiles(user_email);

CREATE TABLE intelligence_credentials (
  id TEXT PRIMARY KEY,
  user_email TEXT NOT NULL,
  subject_type TEXT NOT NULL, -- provider | connector
  subject_id TEXT NOT NULL,
  key_version INTEGER NOT NULL,
  ciphertext TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE (subject_type, subject_id)
);

CREATE TABLE intelligence_mcp_connectors (
  id TEXT PRIMARY KEY,
  user_email TEXT NOT NULL,
  name TEXT NOT NULL,
  endpoint TEXT NOT NULL,
  enabled INTEGER NOT NULL DEFAULT 1,
  allowlist_json TEXT NOT NULL DEFAULT '[]', -- search tool names
  timeout_ms INTEGER NOT NULL DEFAULT 15000,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX idx_intelligence_connectors_user ON intelligence_mcp_connectors(user_email);

CREATE TABLE intelligence_agent_runs (
  id TEXT PRIMARY KEY,
  user_email TEXT NOT NULL,
  scope TEXT NOT NULL,              -- finance_import_analysis
  scope_id TEXT NOT NULL,           -- import file fingerprint
  provider_id TEXT,
  status TEXT NOT NULL DEFAULT 'running', -- running | succeeded | failed | cancelled
  model TEXT,
  prompt_version TEXT,
  input_summary_json TEXT NOT NULL DEFAULT '{}',
  output_summary_json TEXT NOT NULL DEFAULT '{}',
  token_usage_json TEXT NOT NULL DEFAULT '{}',
  duration_ms INTEGER,
  error TEXT,
  created_at TEXT NOT NULL,
  completed_at TEXT
);

CREATE INDEX idx_intelligence_runs_scope ON intelligence_agent_runs(scope, scope_id);

CREATE TABLE intelligence_suggestions (
  id TEXT PRIMARY KEY,
  run_id TEXT NOT NULL REFERENCES intelligence_agent_runs(id) ON DELETE CASCADE,
  scope_id TEXT NOT NULL,
  target_key TEXT NOT NULL, -- row source reference (or other stable key)
  field TEXT NOT NULL,      -- category | merchant | transfer | relationship
  value TEXT NOT NULL,
  confidence REAL NOT NULL,
  rationale TEXT,
  evidence_json TEXT NOT NULL DEFAULT '[]',
  status TEXT NOT NULL DEFAULT 'pending', -- pending | accepted | rejected
  created_at TEXT NOT NULL
);

CREATE INDEX idx_intelligence_suggestions_run ON intelligence_suggestions(run_id);
CREATE INDEX idx_intelligence_suggestions_scope ON intelligence_suggestions(scope_id);
