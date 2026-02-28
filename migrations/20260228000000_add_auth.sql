-- +goose Up
CREATE TABLE users (
    id         TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    email      TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE magic_tokens (
    id         TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id    TEXT NOT NULL REFERENCES users(id),
    token_hash TEXT UNIQUE NOT NULL,    -- SHA-256 of the raw token
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Fast lookup at verify time; only index unused tokens
CREATE INDEX idx_magic_tokens_hash ON magic_tokens(token_hash) WHERE used_at IS NULL;

-- +goose Down
DROP TABLE magic_tokens;
DROP TABLE users;
