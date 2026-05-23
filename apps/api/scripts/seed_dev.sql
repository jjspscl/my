-- Seed dev database with deterministic dummy data
-- 5 habits, ~450 habit_completions, ~75 transactions
-- Date range: 2026-01-01 to 2026-05-23
-- Run: sqlite3 apps/api/my_dev.db < apps/api/scripts/seed_dev.sql

BEGIN;

-- Clean existing data
DELETE FROM habit_completions;
DELETE FROM habits;
DELETE FROM transactions;

--------------------------------------------------------------
-- HABITS (5 total)
--------------------------------------------------------------

INSERT INTO habits (id, user_email, name, color, frequency, target_per_week)
VALUES
    ('h-001', 'jjspscl@gmail.com', 'Exercise', 'green', 'daily', 7),
    ('h-002', 'jjspscl@gmail.com', 'Read 30 min', 'blue', 'daily', 7),
    ('h-003', 'jjspscl@gmail.com', 'Meditate', 'purple', 'daily', 7),
    ('h-004', 'jjspscl@gmail.com', 'Learn Go', 'indigo', 'weekly', 3),
    ('h-005', 'jjspscl@gmail.com', 'Drink 8 glasses', 'teal', 'daily', 7);

--------------------------------------------------------------
-- HABIT COMPLETIONS (deterministic patterns)
-- ~450 total across 5 habits
--------------------------------------------------------------

WITH RECURSIVE dates(d, day_idx) AS (
    SELECT '2026-01-01', 0
    UNION ALL
    SELECT date(d, '+1 day'), day_idx + 1
    FROM dates
    WHERE d < '2026-05-23'
),
day_info AS (
    SELECT d, day_idx, CAST(strftime('%w', d) AS INTEGER) AS dow
    FROM dates
),
-- Exercise (green, ~81%): skip Sundays + every 3rd Wednesday
e AS (
    SELECT printf('hc-h-001-%s', replace(d, '-', '')) AS id, 'h-001' AS habit_id, d
    FROM day_info
    WHERE dow != 0 AND NOT (dow = 3 AND (day_idx / 7) % 3 = 0)
),
-- Read (blue, ~57%): weekdays only, skip every 5th day
r AS (
    SELECT printf('hc-h-002-%s', replace(d, '-', '')) AS id, 'h-002' AS habit_id, d
    FROM day_info
    WHERE dow BETWEEN 1 AND 5 AND day_idx % 5 != 0
),
-- Meditate (purple, ~43%): Mon/Wed/Fri only
m AS (
    SELECT printf('hc-h-003-%s', replace(d, '-', '')) AS id, 'h-003' AS habit_id, d
    FROM day_info
    WHERE dow IN (1, 3, 5)
),
-- Learn Go (indigo, ~3x/week): Tue/Thu/Sat
l AS (
    SELECT printf('hc-h-004-%s', replace(d, '-', '')) AS id, 'h-004' AS habit_id, d
    FROM day_info
    WHERE dow IN (2, 4, 6)
),
-- Drink Water (teal, ~90%): every day except every 10th
w AS (
    SELECT printf('hc-h-005-%s', replace(d, '-', '')) AS id, 'h-005' AS habit_id, d
    FROM day_info
    WHERE day_idx % 10 != 0
)
INSERT INTO habit_completions (id, habit_id, completed_date)
SELECT id, habit_id, d FROM e
UNION ALL
SELECT id, habit_id, d FROM r
UNION ALL
SELECT id, habit_id, d FROM m
UNION ALL
SELECT id, habit_id, d FROM l
UNION ALL
SELECT id, habit_id, d FROM w;

--------------------------------------------------------------
-- FINANCE TRANSACTIONS
-- Recurring monthly (23 entries)
--------------------------------------------------------------

INSERT INTO transactions (id, user_email, amount_cents, currency, category, description, type, transaction_date) VALUES
-- Rent (1st, 15000 PHP)
('tx-rent-01', 'jjspscl@gmail.com', 1500000, 'PHP', 'housing', 'Monthly Rent', 'expense', '2026-01-01'),
('tx-rent-02', 'jjspscl@gmail.com', 1500000, 'PHP', 'housing', 'Monthly Rent', 'expense', '2026-02-01'),
('tx-rent-03', 'jjspscl@gmail.com', 1500000, 'PHP', 'housing', 'Monthly Rent', 'expense', '2026-03-01'),
('tx-rent-04', 'jjspscl@gmail.com', 1500000, 'PHP', 'housing', 'Monthly Rent', 'expense', '2026-04-01'),
('tx-rent-05', 'jjspscl@gmail.com', 1500000, 'PHP', 'housing', 'Monthly Rent', 'expense', '2026-05-01'),

-- Internet (5th, 1899 PHP)
('tx-inet-01', 'jjspscl@gmail.com', 189900, 'PHP', 'utilities', 'Internet Bill', 'expense', '2026-01-05'),
('tx-inet-02', 'jjspscl@gmail.com', 189900, 'PHP', 'utilities', 'Internet Bill', 'expense', '2026-02-05'),
('tx-inet-03', 'jjspscl@gmail.com', 189900, 'PHP', 'utilities', 'Internet Bill', 'expense', '2026-03-05'),
('tx-inet-04', 'jjspscl@gmail.com', 189900, 'PHP', 'utilities', 'Internet Bill', 'expense', '2026-04-05'),
('tx-inet-05', 'jjspscl@gmail.com', 189900, 'PHP', 'utilities', 'Internet Bill', 'expense', '2026-05-05'),

-- Spotify (10th, 299 PHP)
('tx-spot-01', 'jjspscl@gmail.com', 29900, 'PHP', 'entertainment', 'Spotify Premium', 'expense', '2026-01-10'),
('tx-spot-02', 'jjspscl@gmail.com', 29900, 'PHP', 'entertainment', 'Spotify Premium', 'expense', '2026-02-10'),
('tx-spot-03', 'jjspscl@gmail.com', 29900, 'PHP', 'entertainment', 'Spotify Premium', 'expense', '2026-03-10'),
('tx-spot-04', 'jjspscl@gmail.com', 29900, 'PHP', 'entertainment', 'Spotify Premium', 'expense', '2026-04-10'),
('tx-spot-05', 'jjspscl@gmail.com', 29900, 'PHP', 'entertainment', 'Spotify Premium', 'expense', '2026-05-10'),

-- Salary (15th, 85000 PHP)
('tx-sal-01', 'jjspscl@gmail.com', 8500000, 'PHP', 'salary', 'Monthly Salary', 'income', '2026-01-15'),
('tx-sal-02', 'jjspscl@gmail.com', 8500000, 'PHP', 'salary', 'Monthly Salary', 'income', '2026-02-15'),
('tx-sal-03', 'jjspscl@gmail.com', 8500000, 'PHP', 'salary', 'Monthly Salary', 'income', '2026-03-15'),
('tx-sal-04', 'jjspscl@gmail.com', 8500000, 'PHP', 'salary', 'Monthly Salary', 'income', '2026-04-15'),
('tx-sal-05', 'jjspscl@gmail.com', 8500000, 'PHP', 'salary', 'Monthly Salary', 'income', '2026-05-15'),

-- Freelance (25th, 25000 PHP, only Feb/Mar/May)
('tx-free-01', 'jjspscl@gmail.com', 2500000, 'PHP', 'freelance', 'Freelance Project', 'income', '2026-02-25'),
('tx-free-02', 'jjspscl@gmail.com', 2500000, 'PHP', 'freelance', 'Freelance Project', 'income', '2026-03-25'),
('tx-free-03', 'jjspscl@gmail.com', 2500000, 'PHP', 'freelance', 'Freelance Project', 'income', '2026-05-25');

--------------------------------------------------------------
-- Random daily transactions (deterministic day-based selection)
--------------------------------------------------------------

WITH RECURSIVE dates(d, day_idx) AS (
    SELECT '2026-01-01', 0
    UNION ALL
    SELECT date(d, '+1 day'), day_idx + 1
    FROM dates
    WHERE d < '2026-05-23'
),
-- Transport (~20 entries): every day where day_idx % 7 == 2
-- Amounts: 150-500 PHP (15000-50000 cents)
transport AS (
    SELECT
        printf('tx-trn-%04d', day_idx) AS id,
        'jjspscl@gmail.com' AS email,
        15000 + (day_idx * 251 + 7) % 35001 AS cents,
        'PHP' AS cur,
        'transport' AS cat,
        CASE day_idx % 6
            WHEN 0 THEN 'Grab to office'
            WHEN 1 THEN 'Grab ride home'
            WHEN 2 THEN 'Jeepney fare'
            WHEN 3 THEN 'Taxi to meeting'
            WHEN 4 THEN 'Angkas ride'
            ELSE 'MRT fare'
        END AS descr,
        d
    FROM dates
    WHERE day_idx % 7 = 2
),
-- Food (~24 entries): every day where day_idx % 6 = 1
-- Amounts: 80-400 PHP (8000-40000 cents)
food AS (
    SELECT
        printf('tx-food-%04d', day_idx) AS id,
        'jjspscl@gmail.com' AS email,
        8000 + (day_idx * 173 + 11) % 32001 AS cents,
        'PHP' AS cur,
        'food' AS cat,
        CASE day_idx % 6
            WHEN 0 THEN 'Lunch near office'
            WHEN 1 THEN 'Coffee & pastry'
            WHEN 2 THEN 'Dinner out'
            WHEN 3 THEN 'Street food'
            WHEN 4 THEN 'Boba tea'
            ELSE 'Snacks'
        END AS descr,
        d
    FROM dates
    WHERE day_idx % 6 = 1
),
-- Groceries (~10 entries): day_of_month = 7 or 21
-- Amounts: 2000-4000 PHP (200000-400000 cents)
grocery AS (
    SELECT
        printf('tx-gro-%s', replace(d, '-', '')) AS id,
        'jjspscl@gmail.com' AS email,
        200000 + (day_idx * 313 + 5) % 200001 AS cents,
        'PHP' AS cur,
        'groceries' AS cat,
        CASE CAST(strftime('%d', d) AS INTEGER)
            WHEN 7 THEN 'Weekly grocery run'
            ELSE 'Supermarket restock'
        END AS descr,
        d
    FROM dates
    WHERE CAST(strftime('%d', d) AS INTEGER) IN (7, 21)
)
INSERT INTO transactions (id, user_email, amount_cents, currency, category, description, type, transaction_date)
SELECT id, email, cents, cur, cat, descr, 'expense', d FROM transport
UNION ALL
SELECT id, email, cents, cur, cat, descr, 'expense', d FROM food
UNION ALL
SELECT id, email, cents, cur, cat, descr, 'expense', d FROM grocery;

COMMIT;
