package repository

import "context"

// This code should abstract away access to DB for Job domain
// Cancelation via context, transaction inside

type JobRepository struct {
}

func (jr *JobRepository) Create(ctx context.Context) {

}
