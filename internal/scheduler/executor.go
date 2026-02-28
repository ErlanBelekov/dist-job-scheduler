package scheduler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ErlanBelekov/dist-job-scheduler/internal/domain"
)

type Executor struct {
	client *http.Client
}

func NewExecutor() *Executor {
	return &Executor{
		client: &http.Client{}, // no global timeout, each job sets its own
	}
}

type ExecutionResult struct {
	StatusCode int
	Err        error
	Duration   time.Duration
}

func (e *Executor) Run(ctx context.Context, job *domain.Job) ExecutionResult {
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, time.Duration(job.TimeoutSeconds)*time.Second)
	defer cancel()

	var bodyReader io.Reader
	if job.Body != nil {
		bodyReader = strings.NewReader(*job.Body)
	}

	req, err := http.NewRequestWithContext(ctx, job.Method, job.URL, bodyReader)
	if err != nil {
		return ExecutionResult{Err: fmt.Errorf("build request: %w", err), Duration: time.Since(start)}
	}

	for k, v := range job.Headers {
		req.Header.Set(k, v)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return ExecutionResult{Err: fmt.Errorf("do request: %w", err), Duration: time.Since(start)}
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, resp.Body) // drain so the connection can be reused by the pool

	return ExecutionResult{StatusCode: resp.StatusCode, Duration: time.Since(start)}
}
