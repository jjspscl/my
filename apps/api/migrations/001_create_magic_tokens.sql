CREATE TABLE IF NOT EXISTS magic_tokens (
    token TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    used_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_magic_tokens_email ON magic_tokens(email);