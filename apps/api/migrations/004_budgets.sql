CREATE TABLE budgets (
  id TEXT PRIMARY KEY,
  user_email TEXT NOT NULL,
  month TEXT NOT NULL, -- YYYY-MM format
  created_at TEXT NOT NULL DEFAULT (datetime('now')),
  updated_at TEXT NOT NULL DEFAULT (datetime('now')),
  UNIQUE(user_email, month)
);

CREATE TABLE budget_categories (
  id TEXT PRIMARY KEY,
  budget_id TEXT NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
  category TEXT NOT NULL,
  allocated_cents INTEGER NOT NULL DEFAULT 0,
  rollover_enabled INTEGER NOT NULL DEFAULT 0,
  UNIQUE(budget_id, category)
);

CREATE INDEX idx_budgets_user_month ON budgets(user_email, month);
CREATE INDEX idx_budget_categories_budget ON budget_categories(budget_id);