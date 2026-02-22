# Overview

Dist. job scheduler that can scale horizontally written in Go.

Functionality:

- schedule a job(firing HTTP request) for execution at specific time in future with <1-2s latency
- automatic retries with backoff
- get metadata about jobs and executions

Stack:

- Go
- PostgreSQL

Can be ran anywhere, Docker image for exporting.
