-- +goose Up

CREATE TABLE schedules (
    id              TEXT        PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id         TEXT        NOT NULL REFERENCES users(id),
    name            TEXT        NOT NULL,
    cron_expr       TEXT        NOT NULL,
    url             TEXT        NOT NULL,
    method          TEXT        NOT NULL DEFAULT 'POST',
    headers         JSONB       NOT NULL DEFAULT '{}',
    body            TEXT,
    timeout_seconds INT         NOT NULL DEFAULT 30,
    max_retries     INT         NOT NULL DEFAULT 3,
    backoff         TEXT        NOT NULL DEFAULT 'exponential',
    paused          BOOLEAN     NOT NULL DEFAULT FALSE,
    next_run_at     TIMESTAMPTZ NOT NULL,
    last_run_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT schedules_user_name_unique UNIQUE (user_id, name)
);

-- Dispatcher polls this constantly — partial index on active schedules only
CREATE INDEX idx_schedules_due ON schedules (next_run_at)
    WHERE NOT paused;

-- API list queries: filter by user
CREATE INDEX idx_schedules_user ON schedules (user_id, created_at DESC, id DESC);

-- Link jobs back to their parent schedule
ALTER TABLE jobs
    ADD COLUMN schedule_id TEXT REFERENCES schedules(id) ON DELETE SET NULL;

CREATE INDEX idx_jobs_schedule_id ON jobs (schedule_id)
    WHERE schedule_id IS NOT NULL;

-- +goose Down
DROP INDEX idx_jobs_schedule_id;
ALTER TABLE jobs DROP COLUMN schedule_id;
DROP INDEX idx_schedules_user;
DROP INDEX idx_schedules_due;
DROP TABLE schedules;
