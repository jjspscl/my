-- 008_wallet_backfill.sql
-- Backfill default Cash wallets for users with existing data but no wallets.
-- Migrates null wallet_id/target_wallet_id to be non-null.
--
-- NOTE: SQLite does not support ALTER TABLE ADD NOT NULL on existing columns.
-- The NOT NULL invariant is enforced at the application layer.
-- Run this migration once; new code paths will always set wallet_id/target_wallet_id.
-- NOTE: No BEGIN/COMMIT — migration runner wraps each file in its own transaction.

-- Step 1: Create a default Cash wallet for each user who has transactions or goals
-- but does NOT have any wallet yet. These users need at least one wallet so that
-- we can backfill their null references.
INSERT INTO wallets (id, user_email, name, kind, currency, opening_balance_cents, is_default, created_at, updated_at)
SELECT
    'backfill_cash_' || u.user_email,
    u.user_email,
    'Cash',
    'cash',
    'PHP',
    0,
    1,
    datetime('now'),
    datetime('now')
FROM (
    SELECT DISTINCT user_email FROM transactions WHERE wallet_id IS NULL
    UNION
    SELECT DISTINCT user_email FROM savings_goals WHERE target_wallet_id IS NULL
) u
WHERE NOT EXISTS (SELECT 1 FROM wallets w WHERE w.user_email = u.user_email);

-- Step 2: Backfill null wallet_ids on existing transactions.
-- Picks the user's default wallet (first active, preferring is_default=1).
UPDATE transactions
SET wallet_id = (
    SELECT id FROM wallets
    WHERE user_email = transactions.user_email AND archived_at IS NULL
    ORDER BY is_default DESC, created_at ASC
    LIMIT 1
)
WHERE wallet_id IS NULL;

-- Step 3: Backfill null target_wallet_ids on existing savings goals.
UPDATE savings_goals
SET target_wallet_id = (
    SELECT id FROM wallets
    WHERE user_email = savings_goals.user_email AND archived_at IS NULL
    ORDER BY is_default DESC, created_at ASC
    LIMIT 1
)
WHERE target_wallet_id IS NULL;
