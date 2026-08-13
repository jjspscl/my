-- 010_finance_categories
-- Phase 1: category metadata. Unscoped (single-user), additive, no foreign key
-- on transactions.category — that column is free text written by the REST API,
-- MCP tools, and the offline replay queue, and a FK would break free-text entry
-- and replay of a category whose row does not exist yet.
--
-- Seeded with the nine frontend presets (they become seed data, not the source
-- of truth) plus one row per distinct category already present in
-- transactions. Everything starts unclassified and non-essential; the settings
-- surface classifies them.

CREATE TABLE finance_categories (
  name TEXT PRIMARY KEY,
  classification TEXT NOT NULL DEFAULT 'unclassified'
    CHECK(classification IN ('needs', 'wants', 'savings', 'income', 'debt', 'other', 'unclassified')),
  essential INTEGER NOT NULL DEFAULT 0,
  active INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

INSERT INTO finance_categories (name) VALUES
  ('Food'), ('Transport'), ('Groceries'), ('Bills'), ('Entertainment'),
  ('Health'), ('Shopping'), ('Education'), ('Other')
ON CONFLICT(name) DO NOTHING;

INSERT INTO finance_categories (name)
SELECT DISTINCT category FROM transactions
WHERE category IS NOT NULL AND category != ''
ON CONFLICT(name) DO NOTHING;