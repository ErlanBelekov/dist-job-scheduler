package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
	"github.com/ErlanBelekov/dist-job-scheduler/internal/repository"
)

type JobUsecase struct {
	repo     repository.JobRepository
	attempts repository.AttemptRepository
}

func NewJobUsecase(repo repository.JobRepository, attempts repository.AttemptRepository) *JobUsecase {
	return &JobUsecase{repo: repo, attempts: attempts}
}

type CreateJobInput struct {
	UserID         string
	IdempotencyKey string
	URL            string
	Method         string
	Headers        map[string]string
	Body           *string
	TimeoutSeconds int
	ScheduledAt    time.Time
	MaxRetries     int
	Backoff        domain.Backoff
}

func (u *JobUsecase) CreateJob(ctx context.Context, input CreateJobInput) (*domain.Job, error) {
	if input.Headers == nil {
		input.Headers = make(map[string]string)
	}

	if input.TimeoutSeconds == 0 {
		input.TimeoutSeconds = 30
	}
	if input.MaxRetries == 0 {
		input.MaxRetries = 3
	}
	if input.Backoff == "" {
		input.Backoff = domain.BackoffExponential
	}

	job := &domain.Job{
		UserID:         input.UserID,
		IdempotencyKey: input.IdempotencyKey,
		URL:            input.URL,
		Method:         input.Method,
		Headers:        input.Headers,
		Body:           input.Body,
		TimeoutSeconds: input.TimeoutSeconds,
		Status:         domain.StatusPending,
		ScheduledAt:    input.ScheduledAt,
		MaxRetries:     input.MaxRetries,
		Backoff:        input.Backoff,
	}

	created, err := u.repo.Create(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("create job: %w", err)
	}

	return created, nil
}

func (u *JobUsecase) GetByID(ctx context.Context, jobID, userID string) (*domain.Job, error) {
	job, err := u.repo.GetByID(ctx, jobID, userID)
	if err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	return job, nil
}

func (u *JobUsecase) ListAttempts(ctx context.Context, jobID, userID string) ([]*domain.JobAttempt, error) {
	// Verify the job exists and belongs to this user before returning its attempts.
	if _, err := u.repo.GetByID(ctx, jobID, userID); err != nil {
		return nil, fmt.Errorf("get job: %w", err)
	}
	attempts, err := u.attempts.ListByJobID(ctx, jobID)
	if err != nil {
		return nil, fmt.Errorf("list attempts: %w", err)
	}
	return attempts, nil
}
