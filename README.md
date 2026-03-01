# dist-job-scheduler

A distributed HTTP job scheduler. Clients POST a job with a target URL, method, headers, body, and a UTC fire time. Workers claim and execute them with sub-2s latency; failed jobs retry with configurable exponential or linear backoff.

## What's built

| Area | Status |
|---|---|
| Job scheduling — create, claim, execute, retry | ✅ Done |
| Exactly-once execution (FOR UPDATE SKIP LOCKED + reaper) | ✅ Done |
| Crash recovery (heartbeat + reaper process) | ✅ Done |
| Magic-link authentication + JWT | ✅ Done |
| Per-user job isolation (ownership enforced at query level) | ✅ Done |
| CI pipeline (lint, tests, migrations against real Postgres) | ✅ Done |

## System map

```
[ Client ]
    │  REST API
    ▼
[ server ]  ──────────────────────────────────┐
    │                                         │
    ▼                                         ▼
[ PostgreSQL ] ◄──────────────── [ scheduler ]
                                  Worker + Reaper + Executor
```

Future state (see Roadmap below):

```
[ Browser ]
    │
    ▼
[ Frontend ]
    │  GraphQL
    ▼
[ GraphQL Gateway ]
    │  REST
    ▼
[ server ]  ──── [ PostgreSQL ] ──── [ scheduler ]
```

## Stack

| Concern | Choice |
|---|---|
| Language | Go 1.25 |
| Web framework | Gin |
| Database | PostgreSQL via `pgx/v5` |
| Migrations | goose |
| Auth | Magic links → JWT HS256; email via Resend (logged locally) |
| Config | `caarlos0/env` — struct tags, no `.env` files in Go code |
| Linter | golangci-lint v2 |

## Local dev

```bash
# Prerequisites: Docker, direnv, goose
eval "$(direnv hook zsh)"   # if not already in ~/.zshrc

docker compose up -d postgres
direnv allow
goose -dir ./migrations postgres "$DATABASE_URL" up

go run ./cmd/server        # terminal 1
go run ./cmd/scheduler     # terminal 2
```

See `CLAUDE.md` for the full local setup guide, auth flow walkthrough, and coding conventions.

---

## Roadmap

### Phase 1 — Core backend ✅
- Job CRUD, worker, reaper, retry with backoff
- Exactly-once execution via Postgres row-level locking
- Magic-link auth + JWT; jobs scoped to authenticated users
- CI: lint + test + migrations on every PR

### Phase 2 — Deployment 🔄 In progress
- Docker images (already present: `Dockerfile.server`, `Dockerfile.scheduler`, `Dockerfile.migrate`)
- Deploy to K8S on rented VM
- Staging and production environments
- Terraform for infra provisioning (Enkidu)

### Phase 3 — Observability
- OpenTelemetry instrumentation (traces + metrics)
- Prometheus + Grafana dashboards
- Key metrics to track:
  - Job pickup latency (created → running)
  - Reaper rescue rate (target: <1% of jobs)
  - Worker instance lifetime and shutdown count
  - API error rate and p99 latency

### Phase 4 — Frontend & GraphQL gateway
- **Frontend repo** — separate repository, TBD stack
- **GraphQL gateway** — sits between the frontend and core services; aggregates and shapes data for UI consumption
- Additional API endpoints needed before gateway:
  - Execution history per job
  - Job listing with filters (status, date range)

### Phase 5 — Documentation site
- Public-facing docs for the scheduler API
- Planned for after the frontend and gateway are stable
