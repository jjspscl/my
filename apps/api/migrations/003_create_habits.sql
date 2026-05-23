CREATE TABLE IF NOT EXISTS habits (
    id TEXT PRIMARY KEY,
    user_email TEXT NOT NULL,
    name TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT 'blue',
    frequency TEXT NOT NULL DEFAULT 'daily' CHECK(frequency IN ('daily', 'weekly')),
    target_per_week INTEGER NOT NULL DEFAULT 1,
    archived INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS habit_completions (
    id TEXT PRIMARY KEY,
    habit_id TEXT NOT NULL REFERENCES habits(id),
    completed_date DATE NOT NULL,
    created_at DATETIME NOT NULL DEFAULT (datetime('now')),
    UNIQUE(habit_id, completed_date)
);

CREATE INDEX IF NOT EXISTS idx_habits_user ON habits(user_email);
CREATE INDEX IF NOT EXISTS idx_completions_habit_date ON habit_completions(habit_id, completed_date);
CREATE INDEX IF NOT EXISTS idx_completions_date ON habit_completions(completed_date);