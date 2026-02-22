package usecase

import "github.com/ErlanBelekov/dist-job-scheduler/internal/domain"

type CreateJobInput struct {
	// all the fields needed to create new Job
}

// dumb service that doesn't care which transport is used, or which DB
func CreateJob(input CreateJobInput) (*domain.Job, error) {
	// make request to DB to create a new job for execution, validate user etc
	// multiple DB requests should run as one transaction
	return &domain.Job{}, nil
}
