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

Thinking about the plan for implementation:

- basic Job CRUD and scheduler with locks
- authentication(magic links)
- store execution history of each job
- exactly-once execution(prevent race conditions and multiple executions of a single job, I think we already have that with the DB locking now and reaper process)
- endpoints to query data for front-end: jobs, executions of each job

Add Later:

- CI/CD pipeline with linting, tests, migrations
- OpenTelemetry, Prometheus, Grafana for monitoring and observability
- Dockerize
- Deploy to GCP K8S, add staging + prod envs, modify CI/CD, add Terraform(Enkidu)

# Metrics to implement later, can be useful probably at bigger scale:

- amount of times the schedulers shut down
- average life of a single scheduler worker instance
- latency between creation of job and scheduler picking it up(when its status changes to "running")
- average client server latency, error rate, etc
- amount of jobs picked up by reaper(failed scheduler executions) and amount of jobs processed within scheduler. In perfect world, < 1% of jobs should be picked by reaper
