-- +goose Up

-- Magic link auth is replaced by Clerk; drop the tokens table
DROP TABLE magic_tokens;

-- Clerk user IDs (e.g. user_abc123) are now the primary key.
-- Remove the auto-UUID default so Clerk IDs can be inserted directly.
ALTER TABLE users ALTER COLUMN id DROP DEFAULT;

-- Email is not available from the Clerk JWT; make it optional.
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_email_key;

-- +goose Down
-- Not safely reversible. Run goose reset + up from scratch for local dev.
