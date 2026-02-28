# dist-job-scheduler

A distributed HTTP job scheduler. Clients POST jobs with a target URL, method, headers, body, and a UTC fire time. Workers claim and execute them; failed jobs retry with configurable backoff.

## Binaries

| Binary | Path | Role |
|---|---|---|
| `server` | `cmd/server` | HTTP API — stateless, scales to zero |
| `scheduler` | `cmd/scheduler` | Worker + Reaper — always-on, never scaled horizontally beyond one replica per region |

## Architecture

```
cmd/server          cmd/scheduler
     │                    │
     ▼                    ▼
 usecase             scheduler
     │               (Worker, Reaper, Executor)
     ▼                    │
 repository              │
  (interface)            │
     │                   │
     └──────────┬────────┘
                ▼
      infrastructure/postgres
                │
                ▼
           PostgreSQL
```

**Layer rules — strictly enforced:**
- `domain` has zero imports from this project
- `repository` imports only `domain` — no infrastructure
- `usecase` imports `domain` + `repository` interface — no transport, no postgres
- `transport` imports `usecase` — never `infrastructure` directly
- `infrastructure/postgres` implements `repository.JobRepository`
- `cmd/*` (main) is the composition root — only place that wires layers together

## Key design decisions

### `FOR UPDATE SKIP LOCKED` for job claiming
No Redis, no Kafka. Workers race with a single atomic SQL statement:
```sql
UPDATE jobs SET status = 'running' ...
WHERE id IN (SELECT id FROM jobs WHERE status = 'pending' ... FOR UPDATE SKIP LOCKED)
RETURNING ...
```
Each worker gets a disjoint set of jobs. No duplicates, no coordination overhead.

### Semaphore concurrency (buffered channel, not `sync.WaitGroup`)
`Worker` uses `chan struct{}` as a semaphore. `processBatch` checks `cap(sem) - len(sem)` before claiming — it only claims what it can immediately start. Slow jobs hold their slot; the poll loop is never blocked waiting for them to finish.

### Heartbeat + Reaper for crash recovery
- Workers write `heartbeat_at = NOW()` every 10s while a job runs
- Reaper scans every 30s for jobs stuck in `running` with `heartbeat_at < now - 30s`
- Jobs under `max_retries` → reset to `pending`; exhausted jobs → `failed`
- Both queries use `FOR UPDATE SKIP LOCKED` so multiple reaper replicas are safe

### Structured logging with `slog`
- `TextHandler` in `ENV=local`, `JSONHandler` in staging/production (Datadog/Cloud Logging parseable)
- Logger created once in `main`, injected into every constructor
- Components pre-tag their logger: `logger.With("component", "job_handler")`
- Levels: `Info` for normal flow, `Warn` for retryable failures, `Error` for unexpected errors

### Graceful shutdown
- `signal.NotifyContext` for SIGINT/SIGTERM
- `stop()` called explicitly before any `log.Fatalf` and after `<-ctx.Done()` — never via `defer` before a fatal path (gocritic `exitAfterDefer`)
- `http.Server.Shutdown(ctx)` with 10s timeout for in-flight HTTP requests

## Coding conventions

**Errors**
- Wrap with `fmt.Errorf("operation: %w", err)` at every layer boundary
- Never wrap a `nil` error — always `if err != nil { return nil, fmt.Errorf(...) }`
- Map infrastructure errors to domain errors at the repo layer (`pgx.ErrNoRows` → `domain.ErrJobNotFound`)
- Map domain errors to HTTP status codes in the handler layer (`errors.Is`)

**Constructors**
- Always return a pointer: `func NewFoo(...) *Foo`
- Inject `*slog.Logger` — never call `slog.Default()` inside a package

**Context**
- Every function that touches I/O takes `context.Context` as the first argument
- In Gin handlers, use `ctx.Request.Context()` — not `*gin.Context` — when calling usecases

**HTTP responses**
- Always drain and close response bodies: `defer func() { _ = resp.Body.Close() }()` + `_, _ = io.Copy(io.Discard, resp.Body)`
- Per-job timeouts via `context.WithTimeout`, not a global `http.Client` timeout

**Database**
- Use `pgxpool.Pool` — never `sql.DB`
- Always `defer rows.Close()` after `pool.Query`
- Share a `scanJob(rowScanner)` helper across single-row and multi-row queries to avoid Scan drift

## Stack

| Concern | Choice |
|---|---|
| Language | Go 1.25 |
| Web framework | Gin |
| Database | PostgreSQL 16 via `pgx/v5` |
| Migrations | goose (`-- +goose Up` annotations) |
| Config | `caarlos0/env` — struct tags, no `.env` files in Go code |
| Linter | golangci-lint v2 (`errcheck`, `govet`+shadow, `staticcheck`, `unused`, `ineffassign`, `bodyclose`, `noctx`, `exhaustive`, `gocritic`) |

## Local dev

```bash
# Start Postgres
docker compose up -d postgres

# Apply migrations
goose -dir ./migrations postgres "$DATABASE_URL" up

# Run server
go run ./cmd/server

# Run scheduler (separate terminal)
go run ./cmd/scheduler
```

Env vars are loaded by `direnv` from `.envrc` — never by the Go binary. Add `eval "$(direnv hook zsh)"` to `~/.zshrc` if not already present.

## CI

Two parallel jobs on every push/PR to `main`:
- **lint** — `golangci-lint run ./...`
- **build-test** — builds both binaries, runs goose migrations against a real `postgres:17` container, then `go test -race -count=1 ./...`
