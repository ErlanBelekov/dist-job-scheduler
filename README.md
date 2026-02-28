# Overview

Distributed job scheduler for serverless HTTP and gRPC messaging, written in Go.

Functionality:

- schedule a job for execution at a specific time with <1-2s latency
- execute pending jobs exactly once
- automatic retries with backoff
- get metadata about jobs and executions
- multiple transports: REST API and gRPC

Stack:

- Go
- PostgreSQL
- gin
- pgx
- goose for migrations

Add Later:

- CI/CD pipeline with linting, tests, migrations
- OpenTelemetry, Prometheus, Grafana for monitoring and observability
- Dockerize
- Deploy to GCP Cloud Run and scale to zero

# Metrics to implement later, can be useful probably at bigger scale:

- amount of times the schedulers shut down
- average life of a single scheduler worker instance
- latency between creation of job and scheduler picking it up(when its status changes to "running")
- average client server latency, error rate, etc
