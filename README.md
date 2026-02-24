# Overview

Distributed job scheduler that can scale horizontally, written in Go.

Functionality:

- schedule a job(firing HTTP request) for execution at specific time with <1-2s latency
- execute pending jobs exactly once
- automatic retries with backoff
- get metadata about jobs and executions

Stack:

- Go
- PostgreSQL

Add Later:

- CI/CD pipeline with linting, tests, migrations
- OpenTelemetry, Prometheus, Grafana for monitoring and observability
- Dockerize
- Deploy to GCP Cloud Run and scale to zero
