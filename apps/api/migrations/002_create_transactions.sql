CREATE TABLE IF NOT EXISTS transactions (
    id TEXT PRIMARY KEY,
    user_email TEXT NOT NULL,
    amount_cents INTEGER NOT NULL,
    currency TEXT NOT NULL DEFAULT 'PHP',
    category TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    type TEXT NOT NULL CHECK(type IN ('expense', 'income')),
    transaction_date DATE NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_email ON transactions(user_email);
CREATE INDEX IF NOT EXISTS idx_transactions_date ON transactions(transaction_date);
CREATE INDEX IF NOT EXISTS idx_transactions_user_date ON transactions(user_email, transaction_date);
