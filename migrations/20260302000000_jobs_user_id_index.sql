-- +goose Up
-- Covers GET /jobs list queries: filter by user, order by scheduled_at for cursor pagination.
-- The scheduler's partial indexes (idx_jobs_due, idx_jobs_stale) cover worker/reaper queries
-- and are unaffected — they filter on status with no user_id predicate.
CREATE INDEX idx_jobs_user_scheduled ON jobs(user_id, scheduled_at DESC, id DESC);

-- +goose Down
DROP INDEX idx_jobs_user_scheduled;
