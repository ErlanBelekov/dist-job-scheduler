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
- The executor's `http.Client` has a 5-minute safety-net timeout as a last resort, but real per-job timeouts are enforced via context. TLS minimum version is 1.2, redirect limit is 10.

**Database**
- Use `pgxpool.Pool` — never `sql.DB`
- Always `defer rows.Close()` after `pool.Query`
- Share a `scanJob(rowScanner)` helper across single-row and multi-row queries to avoid Scan drift
- Pool is configured with `MaxConnLifetime=1h`, `MaxConnIdleTime=30m`, `HealthCheckPeriod=30s`, `ConnectTimeout=5s` — do not remove these; they prevent stale connections under K8S pod restarts and DB failovers

## Stack

| Concern | Choice |
|---|---|
| Language | Go 1.25 |
| Web framework | Gin |
| Database | PostgreSQL 16 via `pgx/v5` |
| Migrations | goose (`-- +goose Up` annotations) |
| Config | `caarlos0/env` — struct tags, no `.env` files in Go code |
| Auth | Magic links → JWT HS256 (`golang-jwt/jwt/v5`); email via Resend (`resend-go/v2`) |
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

Required env vars (already in `.envrc` for local):

| Var | Local default | Notes |
|---|---|---|
| `DATABASE_URL` | `postgres://scheduler:scheduler@localhost:5432/scheduler?sslmode=disable` | |
| `JWT_SECRET` | set in `.envrc` | min 32 chars |
| `MAGIC_LINK_BASE_URL` | `http://localhost:8080` | base for verify links in emails |
| `RESEND_API_KEY` | not required locally | required in staging/production |
| `RESEND_FROM` | not required locally | required in staging/production |

### Seeding dev data

```bash
go run ./cmd/seed
```

Creates `seed@test.local` and 20 jobs scheduled 1 minute from now — a mix of jobs that will succeed, fail with retries, and timeout. Idempotent on re-runs. The script prints job IDs and curl commands ready to copy.

### Testing the auth flow locally

In `ENV=local`, emails are never sent — the magic link is logged to stdout instead.

```bash
# 1. Request a magic link — always returns 200
curl -s -X POST http://localhost:8080/auth/magic-link \
  -H "Content-Type: application/json" \
  -d '{"email":"you@example.com"}'

# 2. Copy the raw token from the server log line:
#    msg="magic link email (local dev)" body="...<a href=\"http://localhost:8080/auth/verify?token=TOKEN\">..."

# 3. Exchange the token for a JWT
curl -s "http://localhost:8080/auth/verify?token=TOKEN"
# → {"token":"eyJ..."}

# 4. Call protected endpoints
curl -s http://localhost:8080/jobs/SOME_ID \
  -H "Authorization: Bearer eyJ..."
```

**Common gotchas:**
- The magic-link token is single-use — replaying the same verify URL returns 401
- The token expires after 15 minutes
- The JWT lasts 24 hours; re-run steps 1–3 to get a fresh one
- Pass the JWT (from `/auth/verify`), not the raw magic-link token, as the Bearer value

### Resetting dev data

Schema changes that add `NOT NULL` columns require a full reset rather than a forward migration when dev data exists:

```bash
goose -dir ./migrations postgres "$DATABASE_URL" reset
goose -dir ./migrations postgres "$DATABASE_URL" up
```

## CI

Two parallel jobs on every push/PR to `main`:
- **lint** — `golangci-lint run ./...`
- **build-test** — builds both binaries, runs goose migrations against a real `postgres:17` container, then `go test -race -count=1 ./...`

The Claude code-review workflow (`claude.yml`) requires `id-token: write`, `pull-requests: write`, and `contents: read` permissions — these must be set at the job level, not just the workflow level.

## Design notes

### Auth: magic links, no passwords
No password storage, no password reset flow. A user POSTs their email, gets a single-use tokenised link (15 min TTL), exchanges it for a JWT (24 h). Google OAuth is deferred until there is a frontend redirect flow.

The raw token is never stored — only its SHA-256 hash. `ClaimMagicToken` is a single atomic `UPDATE … WHERE used_at IS NULL AND expires_at > NOW() RETURNING …` — no separate SELECT, no TOCTOU window.

### Idempotency keys are scoped per user
`UNIQUE(user_id, idempotency_key)` — different users can reuse the same key independently. The scheduler operates on jobs without user context (it only cares about `status` and `scheduled_at`), so `user_id` is not threaded through the scheduler layer.

### Authorization at the query level
`GetByID` filters `WHERE id = $1 AND user_id = $2`. A job belonging to another user returns `ErrJobNotFound` (404), not a 403 — consistent with not revealing whether a resource exists.

### Transport error strings are constants
`internal/transport/http/handler/errors.go` holds all HTTP-facing error message strings. Domain error values (`ErrJobNotFound`, etc.) are used for `errors.Is` branching only — never `.Error()` directly in responses. This keeps casing consistent and makes copy changes a one-liner.

### Two-phase attempt writes (open before execute, close after)
`CreateAttempt` is called before `executor.Run`, `CompleteAttempt` after. If a worker crashes mid-execution the attempt row stays in the DB with `completed_at = NULL` — immediately visible in `GET /jobs/:id/attempts` as an incomplete run. This is intentional: it gives operators a signal that a worker died holding this job, even before the reaper reschedules it.

`CreateAttempt` failure is non-fatal — the job still executes, just without a history record for that run. Never let observability writes block execution.

### Never wrap HTTP calls in a DB transaction
Job execution (the outbound HTTP call) cannot be transactional. Holding a Postgres connection open for up to `timeout_seconds` (default 30s, max 3600s) while waiting for an external endpoint would starve the connection pool under any real concurrency. Each DB write in `runJob` is independent and failures are handled locally or by the reaper.

### UNIQUE constraints as defensive guards
`UNIQUE(job_id, attempt_num)` exists even though `FOR UPDATE SKIP LOCKED` already prevents two workers from claiming the same job. If a bug ever breaks the claiming logic, the DB rejects the duplicate attempt rather than silently storing phantom data. Cheap constraint, strong guarantee.

### Security headers are applied globally
`middleware.Security()` is registered on the root router via `r.Use(...)`, so every response — including 404s and 401s — gets the security headers. Do not register it per-route group or the unauthenticated error responses will be missing them.

### Unit test boundary
Unit tests cover: auth usecase (token hashing, JWT signing), JWT middleware (missing/expired/wrong-key/valid token), HTTP handlers (request parsing, status codes). Ownership enforcement and composite uniqueness are SQL guarantees — they belong in integration tests against a real DB, not unit tests with fakes.
