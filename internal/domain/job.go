package domain

import (
	"time"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Backoff string

const (
	BackoffExponential Backoff = "exponentional"
	BackoffLinear      Backoff = "linear"
)

type Job struct {
	ID             string
	IdempotencyKey string
	URL            string
	Method         string
	Headers        map[string]string
	Body           *string // nil means no body
	TimeoutSeconds int

	Status      Status
	ScheduledAt time.Time

	RetryCount int
	MaxRetries int
	Backoff    Backoff

	ClaimedAt   *time.Time
	ClaimedBy   *string // worker ID
	HeartbeatAt *time.Time
	CompletedAt *time.Time
	LastError   *string

	CreatedAt time.Time // when the Job was created
}

type JobExecution struct {
	ID          string
	JobID       string
	AttemptNum  int
	WorkerID    string
	StartedAt   time.Time
	CompletedAt *time.Time
	StatusCode  *int
	Error       *string
	DurationMS  *int64
}
