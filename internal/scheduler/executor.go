package scheduler

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
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
		client: &http.Client{
			// Per-job timeouts are set via context; this is a safety net.
			Timeout: 5 * time.Minute,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
			CheckRedirect: func(_ *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("stopped after 10 redirects")
				}
				return nil
			},
		},
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
