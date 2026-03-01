-- +goose Up
ALTER TABLE jobs
    ADD COLUMN user_id TEXT NOT NULL REFERENCES users(id);

-- Replace the single-column unique constraint with a per-user one
ALTER TABLE jobs
    DROP CONSTRAINT jobs_idempotency_key_key;

ALTER TABLE jobs
    ADD CONSTRAINT jobs_user_idempotency_key_unique UNIQUE (user_id, idempotency_key);

-- +goose Down
ALTER TABLE jobs DROP CONSTRAINT jobs_user_idempotency_key_unique;
ALTER TABLE jobs ADD CONSTRAINT jobs_idempotency_key_key UNIQUE (idempotency_key);
ALTER TABLE jobs DROP COLUMN user_id;
