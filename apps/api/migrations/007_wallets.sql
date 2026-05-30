CREATE TABLE wallets (
    id TEXT PRIMARY KEY,
    user_email TEXT NOT NULL,
    name TEXT NOT NULL,
    kind TEXT NOT NULL CHECK(kind IN ('cash', 'bank', 'ewallet')),
    currency TEXT NOT NULL DEFAULT 'PHP',
    opening_balance_cents INTEGER NOT NULL DEFAULT 0,
    is_default INTEGER NOT NULL DEFAULT 0,
    archived_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE wallet_transfers (
    id TEXT PRIMARY KEY,
    user_email TEXT NOT NULL,
    from_wallet_id TEXT NOT NULL REFERENCES wallets(id),
    to_wallet_id TEXT NOT NULL REFERENCES wallets(id),
    amount_cents INTEGER NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    transfer_date TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

ALTER TABLE transactions ADD COLUMN wallet_id TEXT REFERENCES wallets(id);
ALTER TABLE savings_goals ADD COLUMN target_wallet_id TEXT REFERENCES wallets(id);
ALTER TABLE goal_contributions ADD COLUMN source_wallet_id TEXT REFERENCES wallets(id);
ALTER TABLE goal_contributions ADD COLUMN transfer_id TEXT REFERENCES wallet_transfers(id);

CREATE INDEX idx_wallets_user ON wallets(user_email);
CREATE INDEX idx_wallet_transfers_user ON wallet_transfers(user_email);
CREATE INDEX idx_wallet_transfers_from ON wallet_transfers(from_wallet_id);
CREATE INDEX idx_wallet_transfers_to ON wallet_transfers(to_wallet_id);