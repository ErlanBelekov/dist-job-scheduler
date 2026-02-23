package postgres

import (
	"context"
	"fmt"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type JobRepository struct {
	pool *pgxpool.Pool
}

func NewJobRepository(pool *pgxpool.Pool) *JobRepository {
	return &JobRepository{pool: pool}
}

func (r *JobRepository) Create(ctx context.Context, job *domain.Job) (*domain.Job, error) {
	query := `
                INSERT INTO jobs (
                        idempotency_key, url, method, headers, body,
                        timeout_seconds, status, scheduled_at, max_retries, backoff
                ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
                RETURNING id, idempotency_key, url, method, headers, body,
                          timeout_seconds, status, scheduled_at, retry_count,
                          max_retries, backoff, claimed_at, claimed_by,
                          heartbeat_at, completed_at, last_error, created_at, updated_at`

	row := r.pool.QueryRow(ctx, query,
		job.IdempotencyKey,
		job.URL,
		job.Method,
		job.Headers,
		job.Body,
		job.TimeoutSeconds,
		job.Status,
		job.ScheduledAt,
		job.MaxRetries,
		job.Backoff,
	)

	return scanJob(row)
}

func (r *JobRepository) GetByID(ctx context.Context, id string) (*domain.Job, error) {
	query := `
                SELECT id, idempotency_key, url, method, headers, body,
                       timeout_seconds, status, scheduled_at, retry_count,
                       max_retries, backoff, claimed_at, claimed_by,
                       heartbeat_at, completed_at, last_error, created_at, updated_at
                FROM jobs WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	return scanJob(row)
}

// scanJob is a private helper — avoids repeating Scan calls across multiple queries
func scanJob(row pgx.Row) (*domain.Job, error) {
	var j domain.Job
	err := row.Scan(
		&j.ID, &j.IdempotencyKey, &j.URL, &j.Method, &j.Headers, &j.Body,
		&j.TimeoutSeconds, &j.Status, &j.ScheduledAt, &j.RetryCount,
		&j.MaxRetries, &j.Backoff, &j.ClaimedAt, &j.ClaimedBy,
		&j.HeartbeatAt, &j.CompletedAt, &j.LastError, &j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan job: %w", err)
	}
	return &j, nil
}
