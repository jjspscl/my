-- 009_finance_foundation
-- Phase 0 foundation: currency correctness, idempotency keys, and query
-- indexes. All changes are additive; existing rows are backfilled in place.

-- 1. Currency on budgets (reporting base currency for the month's plan).
ALTER TABLE budgets ADD COLUMN currency TEXT NOT NULL DEFAULT 'PHP';

-- 2. Currency on recurring bills (bills are expectations; default to base).
ALTER TABLE recurring_bills ADD COLUMN currency TEXT NOT NULL DEFAULT 'PHP';

-- 3. Currency on savings goals. The wallet is the currency authority, so
--    existing goals inherit their target wallet's currency when known.
ALTER TABLE savings_goals ADD COLUMN currency TEXT NOT NULL DEFAULT 'PHP';
UPDATE savings_goals
SET currency = COALESCE((SELECT w.currency FROM wallets w WHERE w.id = savings_goals.target_wallet_id), 'PHP')
WHERE target_wallet_id IS NOT NULL;

-- 4. Dual-leg transfer amounts. FromAmountCents leaves the source wallet in
--    the source wallet's currency; ToAmountCents arrives in the destination
--    wallet's currency. Existing rows (same-currency by construction) get both
--    legs equal to amount_cents.
ALTER TABLE wallet_transfers ADD COLUMN from_amount_cents INTEGER;
ALTER TABLE wallet_transfers ADD COLUMN to_amount_cents INTEGER;
UPDATE wallet_transfers SET from_amount_cents = amount_cents, to_amount_cents = amount_cents;

-- 5. Idempotency keys with unique indexes. The app is single-user, so keys are
--    unique on their own; partial indexes keep NULLs (untouched rows) out.
ALTER TABLE transactions ADD COLUMN idempotency_key TEXT;
CREATE UNIQUE INDEX idx_transactions_idempotency ON transactions(idempotency_key) WHERE idempotency_key IS NOT NULL;

ALTER TABLE wallet_transfers ADD COLUMN idempotency_key TEXT;
CREATE UNIQUE INDEX idx_wallet_transfers_idempotency ON wallet_transfers(idempotency_key) WHERE idempotency_key IS NOT NULL;

ALTER TABLE goal_contributions ADD COLUMN idempotency_key TEXT;
CREATE UNIQUE INDEX idx_goal_contributions_idempotency ON goal_contributions(idempotency_key) WHERE idempotency_key IS NOT NULL;

-- 6. Index transactions.wallet_id: balance aggregates and wallet-filtered
--    queries previously scanned the whole table.
CREATE INDEX idx_transactions_wallet ON transactions(wallet_id);
