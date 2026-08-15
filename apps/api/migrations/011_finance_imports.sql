-- 011_finance_imports
-- Import batch audit, deduplication, and dependency-safe rollback.
--
-- Only references, fingerprints, and summaries are stored. The source PDF and
-- its password are never persisted.

CREATE TABLE finance_imports (
  id TEXT PRIMARY KEY,
  user_email TEXT NOT NULL,
  provider TEXT NOT NULL,                 -- 'gcash_pdf'
  file_fingerprint TEXT NOT NULL,         -- sha256 hex of the source file
  statement_from TEXT NOT NULL,           -- YYYY-MM-DD
  statement_to TEXT NOT NULL,             -- YYYY-MM-DD
  wallet_id TEXT REFERENCES wallets(id) ON DELETE SET NULL, -- NULL after rollback deletes a created wallet
  created_wallet_id TEXT,                 -- set when the import created the wallet
  opening_balance_cents INTEGER NOT NULL,
  ending_balance_cents INTEGER NOT NULL,
  reconciliation TEXT NOT NULL DEFAULT 'unknown', -- ok | mismatch | unknown
  status TEXT NOT NULL DEFAULT 'completed',       -- completed | rolled_back
  summary_json TEXT NOT NULL DEFAULT '{}',
  created_at TEXT NOT NULL,
  rolled_back_at TEXT
);

-- One import per file: re-importing the same fingerprint replays the existing
-- batch instead of duplicating rows.
CREATE UNIQUE INDEX idx_finance_imports_fingerprint
  ON finance_imports(user_email, file_fingerprint);

CREATE TABLE finance_import_entries (
  id TEXT PRIMARY KEY,
  import_id TEXT NOT NULL REFERENCES finance_imports(id) ON DELETE CASCADE,
  source_reference TEXT NOT NULL,         -- provider reference number
  occurred_at TEXT NOT NULL,              -- RFC3339
  amount_cents INTEGER NOT NULL,
  kind TEXT NOT NULL,                     -- expense | income | transfer_out | transfer_in | skipped
  category TEXT NOT NULL,
  description TEXT NOT NULL,
  counterparty TEXT,
  counter_wallet_id TEXT,
  outcome TEXT NOT NULL DEFAULT 'imported', -- imported | duplicate | excluded | error
  entity_type TEXT,                         -- transaction | transfer
  entity_id TEXT,
  UNIQUE (import_id, source_reference)
);

CREATE INDEX idx_finance_import_entries_import ON finance_import_entries(import_id);
