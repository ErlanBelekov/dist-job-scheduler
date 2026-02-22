-- +goose Up
CREATE TABLE jobs (
  id               TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
  idempotency_key  TEXT        UNIQUE NOT NULL,

  url              TEXT        NOT NULL,
  method           TEXT        NOT NULL DEFAULT 'POST',
  headers          JSONB       NOT NULL DEFAULT '{}',
  body             TEXT,
  timeout_seconds  INT         NOT NULL DEFAULT 30,

  status           TEXT        NOT NULL DEFAULT 'pending',
  scheduled_at     TIMESTAMPTZ NOT NULL,

  retry_count      INT         NOT NULL DEFAULT 0,
  max_retries      INT         NOT NULL DEFAULT 3,
  backoff          TEXT        NOT NULL DEFAULT 'exponential',

  claimed_at       TIMESTAMPTZ,
  claimed_by       TEXT,
  heartbeat_at     TIMESTAMPTZ,
  completed_at     TIMESTAMPTZ,
  last_error       TEXT,

  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- worker polls this index constantly
CREATE INDEX idx_jobs_due ON jobs (scheduled_at)
  WHERE status = 'pending';

-- reaper scans this to find crashed workers
CREATE INDEX idx_jobs_stale ON jobs (heartbeat_at)
  WHERE status = 'running';

CREATE TABLE job_attempts (
  id           TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
  job_id       TEXT        NOT NULL REFERENCES jobs(id),
  attempt_num  INT         NOT NULL,
  worker_id    TEXT        NOT NULL,
  started_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ,
  status_code  INT,
  error        TEXT,
  duration_ms  BIGINT
);

CREATE INDEX idx_attempts_job_id ON job_attempts (job_id);

-- +goose Down
DROP TABLE job_attempts;
DROP TABLE jobs;